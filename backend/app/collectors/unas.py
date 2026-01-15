from datetime import datetime

import asyncssh

from app.collectors.base import BaseCollector
from app.config import UNASSystemConfig
from app.models.metrics import (
    StorageDisk,
    StoragePool,
    SystemMetrics,
    SystemStatus,
    SystemType,
    UNASMetrics,
)


class UNASCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: UNASSystemConfig):
        super().__init__(system_id, name, SystemType.UNAS)
        self.config = config

    async def _get_connection(self) -> asyncssh.SSHClientConnection:
        """Establish SSH connection."""
        connect_kwargs = {
            "host": self.config.host,
            "port": self.config.port,
            "username": self.config.username,
            "known_hosts": None,
        }

        if self.config.key_path:
            connect_kwargs["client_keys"] = [self.config.key_path]
        elif self.config.password:
            connect_kwargs["password"] = self.config.password

        return await asyncssh.connect(**connect_kwargs)

    async def _run_command(
        self, conn: asyncssh.SSHClientConnection, cmd: str
    ) -> str:
        """Run a command and return output."""
        result = await conn.run(cmd, check=False)
        return result.stdout or ""

    async def check_connection(self) -> bool:
        """Check if NAS is accessible via SSH."""
        try:
            async with await self._get_connection():
                return True
        except Exception:
            return False

    def _parse_zpool_list(self, output: str) -> list[StoragePool]:
        """Parse zpool list output."""
        pools = []
        lines = output.strip().split("\n")

        if len(lines) < 2:
            return pools

        for line in lines[1:]:
            parts = line.split()
            if len(parts) >= 5:
                name = parts[0]
                size_str = parts[1]
                alloc_str = parts[2]
                free_str = parts[3]
                health = parts[-1] if parts[-1] in ("ONLINE", "DEGRADED", "FAULTED", "OFFLINE") else parts[4]

                def parse_size(s: str) -> int:
                    s = s.upper()
                    multipliers = {"K": 1024, "M": 1024**2, "G": 1024**3, "T": 1024**4}
                    for suffix, mult in multipliers.items():
                        if s.endswith(suffix):
                            return int(float(s[:-1]) * mult)
                    return int(float(s))

                total = parse_size(size_str)
                used = parse_size(alloc_str)
                available = parse_size(free_str)

                pools.append(
                    StoragePool(
                        name=name,
                        status=health.lower(),
                        total=total,
                        used=used,
                        available=available,
                        percent=(used / total * 100) if total > 0 else 0,
                    )
                )

        return pools

    def _parse_disk_info(self, lsblk_output: str, smartctl_outputs: dict) -> list[StorageDisk]:
        """Parse disk information."""
        disks = []

        for line in lsblk_output.strip().split("\n"):
            parts = line.split()
            if len(parts) >= 1:
                name = parts[0]
                if name.startswith("sd") or name.startswith("nvme"):
                    disk = StorageDisk(
                        name=name,
                        model="Unknown",
                        serial="Unknown",
                        status="healthy",
                        temperature=None,
                    )

                    if name in smartctl_outputs:
                        smart_data = smartctl_outputs[name]
                        for smart_line in smart_data.split("\n"):
                            if "Model" in smart_line or "Device Model" in smart_line:
                                disk.model = smart_line.split(":")[-1].strip()
                            elif "Serial" in smart_line:
                                disk.serial = smart_line.split(":")[-1].strip()
                            elif "Temperature" in smart_line and "Celsius" in smart_line:
                                try:
                                    temp_parts = smart_line.split()
                                    for i, p in enumerate(temp_parts):
                                        if p.isdigit():
                                            disk.temperature = float(p)
                                            break
                                except (ValueError, IndexError):
                                    pass
                            elif "SMART overall-health" in smart_line:
                                if "PASSED" in smart_line:
                                    disk.status = "healthy"
                                else:
                                    disk.status = "unhealthy"

                    disks.append(disk)

        return disks

    def determine_status(self, metrics: UNASMetrics) -> SystemStatus:
        """Determine status based on storage metrics."""
        for pool in metrics.pools:
            if pool.status == "faulted":
                return SystemStatus.CRITICAL
            if pool.status == "degraded":
                return SystemStatus.WARNING
            if pool.percent > 90:
                return SystemStatus.WARNING

        for disk in metrics.disks:
            if disk.status == "unhealthy":
                return SystemStatus.WARNING
            if disk.temperature and disk.temperature > 50:
                return SystemStatus.WARNING

        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        """Collect NAS storage metrics."""
        async with await self._get_connection() as conn:
            zpool_output = await self._run_command(conn, "zpool list 2>/dev/null || true")

            df_output = await self._run_command(conn, "df -B1 2>/dev/null")

            lsblk_output = await self._run_command(
                conn, "lsblk -d -o NAME,SIZE,MODEL 2>/dev/null || true"
            )

            smartctl_outputs = {}
            disk_names = [
                line.split()[0]
                for line in lsblk_output.strip().split("\n")[1:]
                if line.split() and (line.split()[0].startswith("sd") or line.split()[0].startswith("nvme"))
            ]

            for disk_name in disk_names[:10]:
                smart_output = await self._run_command(
                    conn, f"sudo smartctl -a /dev/{disk_name} 2>/dev/null || smartctl -a /dev/{disk_name} 2>/dev/null || true"
                )
                if smart_output:
                    smartctl_outputs[disk_name] = smart_output

        pools = self._parse_zpool_list(zpool_output) if zpool_output else []

        if not pools and df_output:
            for line in df_output.strip().split("\n")[1:]:
                parts = line.split()
                if len(parts) >= 6:
                    mount = parts[5]
                    if mount in ("/", "/volume1", "/mnt", "/data"):
                        pools.append(
                            StoragePool(
                                name=mount,
                                status="online",
                                total=int(parts[1]),
                                used=int(parts[2]),
                                available=int(parts[3]),
                                percent=float(parts[4].rstrip("%")),
                            )
                        )

        disks = self._parse_disk_info(lsblk_output, smartctl_outputs)

        total_capacity = sum(p.total for p in pools)
        used_capacity = sum(p.used for p in pools)

        unas_metrics = UNASMetrics(
            pools=pools,
            disks=disks,
            total_capacity=total_capacity,
            used_capacity=used_capacity,
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.UNAS,
            status=self.determine_status(unas_metrics),
            last_updated=datetime.utcnow(),
            metrics=unas_metrics,
        )
