"""Mock collectors that generate fake data for testing."""

import random
from datetime import datetime

from app.collectors.base import BaseCollector
from app.models.metrics import (
    ContainerMetrics,
    CpuMetrics,
    DiskMetrics,
    DockerMetrics,
    LinuxMetrics,
    MemoryMetrics,
    NetworkMetrics,
    QBittorrentMetrics,
    SystemMetrics,
    SystemStatus,
    SystemType,
    TemperatureMetrics,
    TorrentInfo,
)


class MockLinuxCollector(BaseCollector):
    """Mock Linux collector with realistic random data."""

    def __init__(self, system_id: str, name: str, config: dict):
        super().__init__(system_id, name, SystemType.LINUX)
        self.config = config
        self._base_cpu = random.uniform(10, 40)
        self._base_mem = random.uniform(30, 60)
        self._uptime = random.randint(86400, 864000)

    async def check_connection(self) -> bool:
        return True

    def determine_status(self, metrics: LinuxMetrics) -> SystemStatus:
        if metrics.cpu.percent > 90 or metrics.memory.percent > 95:
            return SystemStatus.CRITICAL
        if metrics.cpu.percent > 75 or metrics.memory.percent > 85:
            return SystemStatus.WARNING
        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        cpu_percent = max(0, min(100, self._base_cpu + random.uniform(-10, 15)))
        mem_percent = max(0, min(100, self._base_mem + random.uniform(-5, 10)))

        total_mem = 16 * 1024 * 1024 * 1024  # 16GB
        used_mem = int(total_mem * mem_percent / 100)

        core_count = 8
        core_percents = [max(0, min(100, cpu_percent + random.uniform(-20, 20))) for _ in range(core_count)]

        linux_metrics = LinuxMetrics(
            cpu=CpuMetrics(
                percent=cpu_percent,
                cores=core_percents,
                load_avg=[cpu_percent / 25, cpu_percent / 30, cpu_percent / 35],
            ),
            memory=MemoryMetrics(
                total=total_mem,
                used=used_mem,
                available=total_mem - used_mem,
                percent=mem_percent,
            ),
            disks=[
                DiskMetrics(
                    mount="/",
                    total=500 * 1024 * 1024 * 1024,
                    used=int(500 * 1024 * 1024 * 1024 * 0.45),
                    free=int(500 * 1024 * 1024 * 1024 * 0.55),
                    percent=45.0 + random.uniform(-2, 2),
                ),
                DiskMetrics(
                    mount="/home",
                    total=1000 * 1024 * 1024 * 1024,
                    used=int(1000 * 1024 * 1024 * 1024 * 0.62),
                    free=int(1000 * 1024 * 1024 * 1024 * 0.38),
                    percent=62.0 + random.uniform(-2, 2),
                ),
            ],
            network=NetworkMetrics(
                bytes_sent=random.randint(1000000000, 5000000000),
                bytes_recv=random.randint(5000000000, 20000000000),
                upload_rate=random.uniform(100000, 5000000),
                download_rate=random.uniform(500000, 10000000),
            ),
            temperatures=[
                TemperatureMetrics(label="CPU", current=45 + random.uniform(-5, 15)),
                TemperatureMetrics(label="GPU", current=40 + random.uniform(-5, 10)),
            ],
            uptime=self._uptime,
        )

        self._uptime += 5

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.LINUX,
            status=self.determine_status(linux_metrics),
            last_updated=datetime.utcnow(),
            metrics=linux_metrics,
        )


class MockDockerCollector(BaseCollector):
    """Mock Docker collector."""

    def __init__(self, system_id: str, name: str, config: dict):
        super().__init__(system_id, name, SystemType.DOCKER)
        self.config = config
        self._containers = [
            ("nginx", "running"),
            ("postgres", "running"),
            ("redis", "running"),
            ("api-server", "running"),
            ("worker", "running"),
            ("monitoring", "stopped"),
        ]

    async def check_connection(self) -> bool:
        return True

    async def collect(self) -> SystemMetrics:
        containers = []
        running = 0
        stopped = 0

        for name, state in self._containers:
            is_running = state == "running"
            if is_running:
                running += 1
            else:
                stopped += 1

            containers.append(
                ContainerMetrics(
                    id=f"mock_{name}",
                    name=name,
                    image=f"{name}:latest",
                    state=state,
                    status="Up 3 days" if is_running else "Exited (0) 2 hours ago",
                    cpu_percent=random.uniform(0.5, 15) if is_running else 0,
                    memory_percent=random.uniform(1, 25) if is_running else 0,
                    memory_usage=random.randint(50000000, 500000000) if is_running else 0,
                    memory_limit=1024 * 1024 * 1024,
                )
            )

        docker_metrics = DockerMetrics(
            containers=containers,
            total_count=len(self._containers),
            running_count=running,
            stopped_count=stopped,
        )

        status = SystemStatus.HEALTHY
        if stopped > running:
            status = SystemStatus.WARNING

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.DOCKER,
            status=status,
            last_updated=datetime.utcnow(),
            metrics=docker_metrics,
        )


class MockQBittorrentCollector(BaseCollector):
    """Mock qBittorrent collector."""

    def __init__(self, system_id: str, name: str, config: dict):
        super().__init__(system_id, name, SystemType.QBITTORRENT)
        self.config = config
        self._torrents = [
            {"name": "Ubuntu 24.04 LTS", "size": 4.5 * 1024**3, "progress": 100, "state": "stalledUP"},
            {"name": "Debian 12.0", "size": 3.8 * 1024**3, "progress": 100, "state": "uploading"},
            {"name": "Fedora 40", "size": 2.1 * 1024**3, "progress": 78.5, "state": "downloading"},
            {"name": "Arch Linux", "size": 850 * 1024**2, "progress": 45.2, "state": "downloading"},
            {"name": "Linux Mint 21.3", "size": 2.8 * 1024**3, "progress": 92.1, "state": "downloading"},
        ]

    async def check_connection(self) -> bool:
        return True

    async def collect(self) -> SystemMetrics:
        torrents = []
        active_downloads = 0
        total_download_speed = 0
        total_upload_speed = 0

        for t in self._torrents:
            # Simulate progress
            if t["progress"] < 100 and t["state"] == "downloading":
                t["progress"] = min(100, t["progress"] + random.uniform(0.1, 0.5))
                if t["progress"] >= 100:
                    t["state"] = "stalledUP"
                    t["progress"] = 100

            is_downloading = t["state"] in ("downloading", "forcedDL")
            is_uploading = t["state"] in ("uploading", "stalledUP", "forcedUP")

            dl_speed = int(random.uniform(1, 10) * 1024 * 1024) if is_downloading else 0
            ul_speed = int(random.uniform(0.1, 2) * 1024 * 1024) if is_uploading else 0

            if is_downloading:
                active_downloads += 1
                total_download_speed += dl_speed

            total_upload_speed += ul_speed

            eta = None
            if is_downloading and dl_speed > 0:
                remaining = t["size"] * (100 - t["progress"]) / 100
                eta = int(remaining / dl_speed)

            torrents.append(
                TorrentInfo(
                    name=t["name"],
                    size=int(t["size"]),
                    progress=t["progress"],
                    state=t["state"],
                    download_speed=dl_speed,
                    upload_speed=ul_speed,
                    eta=eta,
                )
            )

        qbit_metrics = QBittorrentMetrics(
            download_speed=total_download_speed,
            upload_speed=total_upload_speed,
            active_downloads=active_downloads,
            total_torrents=len(torrents),
            torrents=torrents,
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.QBITTORRENT,
            status=SystemStatus.HEALTHY,
            last_updated=datetime.utcnow(),
            metrics=qbit_metrics,
        )


MOCK_COLLECTORS = {
    "linux": MockLinuxCollector,
    "docker": MockDockerCollector,
    "qbittorrent": MockQBittorrentCollector,
}
