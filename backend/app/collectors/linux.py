from datetime import datetime

import asyncssh

from app.collectors.base import BaseCollector
from app.config import LinuxSystemConfig
from app.models.metrics import (
    CpuMetrics,
    DiskMetrics,
    LinuxMetrics,
    MemoryMetrics,
    NetworkMetrics,
    SystemMetrics,
    SystemStatus,
    SystemType,
    TemperatureMetrics,
)


class LinuxCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: LinuxSystemConfig):
        super().__init__(system_id, name, SystemType.LINUX)
        self.config = config
        self._prev_cpu_stats: dict | None = None
        self._prev_net_stats: dict | None = None
        self._prev_time: float | None = None

    async def _get_connection(self) -> asyncssh.SSHClientConnection:
        """Establish SSH connection."""
        connect_kwargs = {
            "host": self.config.host,
            "port": self.config.port,
            "username": self.config.username,
            "known_hosts": None,
            "connect_timeout": 10,
        }

        if self.config.key_path:
            connect_kwargs["client_keys"] = [self.config.key_path]
        elif self.config.password:
            connect_kwargs["password"] = self.config.password

        print(f"Connecting to {self.config.host}:{self.config.port}...")
        return await asyncssh.connect(**connect_kwargs)

    async def _run_command(
        self, conn: asyncssh.SSHClientConnection, cmd: str
    ) -> str:
        """Run a command and return output."""
        result = await conn.run(cmd, check=True)
        return result.stdout or ""

    def _parse_cpu_stats(self, stat_output: str) -> dict:
        """Parse /proc/stat for CPU statistics."""
        lines = stat_output.strip().split("\n")
        cpu_stats = {}

        for line in lines:
            if line.startswith("cpu"):
                parts = line.split()
                name = parts[0]
                values = [int(x) for x in parts[1:8]]
                cpu_stats[name] = {
                    "user": values[0],
                    "nice": values[1],
                    "system": values[2],
                    "idle": values[3],
                    "iowait": values[4] if len(values) > 4 else 0,
                    "irq": values[5] if len(values) > 5 else 0,
                    "softirq": values[6] if len(values) > 6 else 0,
                }

        return cpu_stats

    def _calculate_cpu_percent(
        self, current: dict, previous: dict | None
    ) -> tuple[float, list[float]]:
        """Calculate CPU percentage from stats."""
        if not previous:
            return 0.0, []

        def calc_percent(cur: dict, prev: dict) -> float:
            cur_idle = cur["idle"] + cur["iowait"]
            prev_idle = prev["idle"] + prev["iowait"]

            cur_total = sum(cur.values())
            prev_total = sum(prev.values())

            total_diff = cur_total - prev_total
            idle_diff = cur_idle - prev_idle

            if total_diff == 0:
                return 0.0
            return ((total_diff - idle_diff) / total_diff) * 100

        total_percent = calc_percent(current["cpu"], previous["cpu"])

        core_percents = []
        for key in sorted(current.keys()):
            if key.startswith("cpu") and key != "cpu":
                if key in previous:
                    core_percents.append(calc_percent(current[key], previous[key]))

        return total_percent, core_percents

    def _parse_meminfo(self, meminfo: str) -> MemoryMetrics:
        """Parse /proc/meminfo."""
        info = {}
        for line in meminfo.strip().split("\n"):
            if ":" in line:
                key, value = line.split(":", 1)
                value = value.strip().split()[0]
                info[key] = int(value) * 1024

        total = info.get("MemTotal", 0)
        available = info.get("MemAvailable", 0)
        used = total - available

        return MemoryMetrics(
            total=total,
            used=used,
            available=available,
            percent=(used / total * 100) if total > 0 else 0,
        )

    def _parse_df(self, df_output: str) -> list[DiskMetrics]:
        """Parse df output."""
        disks = []
        lines = df_output.strip().split("\n")[1:]

        for line in lines:
            parts = line.split()
            if len(parts) >= 6 and parts[0].startswith("/"):
                disks.append(
                    DiskMetrics(
                        mount=parts[5],
                        total=int(parts[1]) * 1024,
                        used=int(parts[2]) * 1024,
                        free=int(parts[3]) * 1024,
                        percent=float(parts[4].rstrip("%")),
                    )
                )

        return disks

    def _parse_net_stats(self, net_output: str) -> dict:
        """Parse /proc/net/dev."""
        stats = {"bytes_recv": 0, "bytes_sent": 0}

        for line in net_output.strip().split("\n")[2:]:
            if ":" in line:
                parts = line.split(":")
                iface = parts[0].strip()
                if iface not in ("lo", "docker0") and not iface.startswith("veth"):
                    values = parts[1].split()
                    stats["bytes_recv"] += int(values[0])
                    stats["bytes_sent"] += int(values[8])

        return stats

    def _parse_sensors(self, sensors_output: str) -> list[TemperatureMetrics]:
        """Parse sensors output."""
        temps = []

        for line in sensors_output.strip().split("\n"):
            if ":" in line and "°C" in line:
                parts = line.split(":")
                label = parts[0].strip()
                temp_str = parts[1].split("°C")[0].strip()
                if temp_str.startswith("+"):
                    temp_str = temp_str[1:]

                try:
                    temp = float(temp_str)
                    temps.append(TemperatureMetrics(label=label, current=temp))
                except ValueError:
                    pass

        return temps

    def _parse_thermal_zones(self, thermal_output: str) -> list[TemperatureMetrics]:
        """Parse thermal zone output (for Jetson/ARM devices)."""
        temps = []

        for line in thermal_output.strip().split("\n"):
            if ":" in line:
                parts = line.split(":")
                if len(parts) == 2:
                    label = parts[0].strip()
                    try:
                        # Thermal zones report in millidegrees
                        temp = float(parts[1].strip()) / 1000.0
                        temps.append(TemperatureMetrics(label=label, current=temp))
                    except ValueError:
                        pass

        return temps

    def _parse_uptime(self, uptime_output: str) -> int:
        """Parse /proc/uptime."""
        try:
            return int(float(uptime_output.strip().split()[0]))
        except (ValueError, IndexError):
            return 0

    async def check_connection(self) -> bool:
        """Check if SSH connection can be established."""
        try:
            async with await self._get_connection():
                return True
        except Exception:
            return False

    def determine_status(self, metrics: LinuxMetrics) -> SystemStatus:
        """Determine system status based on metrics."""
        if metrics.cpu.percent > 90 or metrics.memory.percent > 95:
            return SystemStatus.CRITICAL
        if metrics.cpu.percent > 75 or metrics.memory.percent > 85:
            return SystemStatus.WARNING

        for disk in metrics.disks:
            if disk.percent > 95:
                return SystemStatus.CRITICAL
            if disk.percent > 85:
                return SystemStatus.WARNING

        for temp in metrics.temperatures:
            if temp.critical and temp.current >= temp.critical:
                return SystemStatus.CRITICAL
            if temp.high and temp.current >= temp.high:
                return SystemStatus.WARNING

        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        """Collect all metrics from the Linux system."""
        import time

        current_time = time.time()

        async with await self._get_connection() as conn:
            stat = await self._run_command(conn, "cat /proc/stat")
            meminfo = await self._run_command(conn, "cat /proc/meminfo")
            df = await self._run_command(conn, "df -B1")
            netdev = await self._run_command(conn, "cat /proc/net/dev")
            uptime = await self._run_command(conn, "cat /proc/uptime")
            loadavg = await self._run_command(conn, "cat /proc/loadavg")

            # Try lm-sensors first, then fall back to thermal zones (for Jetson/ARM)
            sensors = ""
            thermal_zones = ""
            try:
                sensors = await self._run_command(conn, "sensors 2>/dev/null")
            except Exception:
                pass

            if not sensors.strip():
                try:
                    # Read thermal zones (Jetson, ARM, etc.)
                    thermal_zones = await self._run_command(
                        conn,
                        "for zone in /sys/class/thermal/thermal_zone*/; do "
                        "name=$(cat ${zone}type 2>/dev/null || echo zone${zone##*zone}); "
                        "temp=$(cat ${zone}temp 2>/dev/null); "
                        "[ -n \"$temp\" ] && echo \"$name:$temp\"; done"
                    )
                except Exception:
                    pass

        cpu_stats = self._parse_cpu_stats(stat)
        cpu_percent, core_percents = self._calculate_cpu_percent(
            cpu_stats, self._prev_cpu_stats
        )
        self._prev_cpu_stats = cpu_stats

        load_parts = loadavg.split()[:3]
        load_avg = [float(x) for x in load_parts] if load_parts else []

        memory = self._parse_meminfo(meminfo)
        disks = self._parse_df(df)

        net_stats = self._parse_net_stats(netdev)
        upload_rate = download_rate = 0.0

        if self._prev_net_stats and self._prev_time:
            time_diff = current_time - self._prev_time
            if time_diff > 0:
                download_rate = (
                    net_stats["bytes_recv"] - self._prev_net_stats["bytes_recv"]
                ) / time_diff
                upload_rate = (
                    net_stats["bytes_sent"] - self._prev_net_stats["bytes_sent"]
                ) / time_diff

        self._prev_net_stats = net_stats
        self._prev_time = current_time

        network = NetworkMetrics(
            bytes_sent=net_stats["bytes_sent"],
            bytes_recv=net_stats["bytes_recv"],
            upload_rate=upload_rate,
            download_rate=download_rate,
        )

        if sensors.strip():
            temperatures = self._parse_sensors(sensors)
        elif thermal_zones.strip():
            temperatures = self._parse_thermal_zones(thermal_zones)
        else:
            temperatures = []
        uptime_seconds = self._parse_uptime(uptime)

        linux_metrics = LinuxMetrics(
            cpu=CpuMetrics(
                percent=cpu_percent,
                cores=core_percents,
                load_avg=load_avg,
            ),
            memory=memory,
            disks=disks,
            network=network,
            temperatures=temperatures,
            uptime=uptime_seconds,
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.LINUX,
            status=self.determine_status(linux_metrics),
            last_updated=datetime.utcnow(),
            metrics=linux_metrics,
        )
