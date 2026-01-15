from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, Field


class SystemStatus(str, Enum):
    HEALTHY = "healthy"
    WARNING = "warning"
    CRITICAL = "critical"
    OFFLINE = "offline"


class SystemType(str, Enum):
    LINUX = "linux"
    DOCKER = "docker"
    QBITTORRENT = "qbittorrent"
    UNIFI = "unifi"
    UNAS = "unas"


class CpuMetrics(BaseModel):
    percent: float = 0.0
    cores: list[float] = Field(default_factory=list)
    load_avg: list[float] = Field(default_factory=list)


class MemoryMetrics(BaseModel):
    total: int = 0
    used: int = 0
    available: int = 0
    percent: float = 0.0


class DiskMetrics(BaseModel):
    mount: str
    total: int = 0
    used: int = 0
    free: int = 0
    percent: float = 0.0


class NetworkMetrics(BaseModel):
    bytes_sent: int = 0
    bytes_recv: int = 0
    upload_rate: float = 0.0
    download_rate: float = 0.0


class TemperatureMetrics(BaseModel):
    label: str
    current: float
    high: float | None = None
    critical: float | None = None


class LinuxMetrics(BaseModel):
    cpu: CpuMetrics = Field(default_factory=CpuMetrics)
    memory: MemoryMetrics = Field(default_factory=MemoryMetrics)
    disks: list[DiskMetrics] = Field(default_factory=list)
    network: NetworkMetrics = Field(default_factory=NetworkMetrics)
    temperatures: list[TemperatureMetrics] = Field(default_factory=list)
    uptime: int = 0


class ContainerMetrics(BaseModel):
    id: str
    name: str
    image: str
    status: str
    state: str
    cpu_percent: float = 0.0
    memory_usage: int = 0
    memory_limit: int = 0
    memory_percent: float = 0.0


class DockerMetrics(BaseModel):
    containers: list[ContainerMetrics] = Field(default_factory=list)
    running_count: int = 0
    stopped_count: int = 0
    total_count: int = 0


class TorrentInfo(BaseModel):
    name: str
    size: int
    progress: float
    download_speed: int
    upload_speed: int
    state: str
    eta: int | None = None


class QBittorrentMetrics(BaseModel):
    download_speed: int = 0
    upload_speed: int = 0
    active_downloads: int = 0
    active_uploads: int = 0
    total_torrents: int = 0
    torrents: list[TorrentInfo] = Field(default_factory=list)


class UnifiDevice(BaseModel):
    name: str
    mac: str
    model: str
    status: str
    uptime: int = 0


class UnifiMetrics(BaseModel):
    devices: list[UnifiDevice] = Field(default_factory=list)
    clients_count: int = 0
    guests_count: int = 0
    wan_download: float = 0.0
    wan_upload: float = 0.0


class StoragePool(BaseModel):
    name: str
    status: str
    total: int = 0
    used: int = 0
    available: int = 0
    percent: float = 0.0


class StorageDisk(BaseModel):
    name: str
    model: str
    serial: str
    status: str
    temperature: float | None = None


class UNASMetrics(BaseModel):
    pools: list[StoragePool] = Field(default_factory=list)
    disks: list[StorageDisk] = Field(default_factory=list)
    total_capacity: int = 0
    used_capacity: int = 0


class SystemMetrics(BaseModel):
    id: str
    name: str
    type: SystemType
    status: SystemStatus = SystemStatus.OFFLINE
    last_updated: datetime = Field(default_factory=datetime.utcnow)
    error: str | None = None
    metrics: (
        LinuxMetrics
        | DockerMetrics
        | QBittorrentMetrics
        | UnifiMetrics
        | UNASMetrics
        | dict[str, Any]
    ) = Field(default_factory=dict)


class MetricsUpdate(BaseModel):
    type: str = "metrics_update"
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    systems: dict[str, SystemMetrics] = Field(default_factory=dict)
