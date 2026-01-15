from pathlib import Path
from typing import Any

import yaml
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings


class LinuxSystemConfig(BaseModel):
    host: str
    port: int = 22
    username: str
    password: str | None = None
    key_path: str | None = None


class DockerSystemConfig(BaseModel):
    host: str | None = None
    socket: str = "unix:///var/run/docker.sock"
    tls: bool = False


class QBittorrentSystemConfig(BaseModel):
    url: str
    username: str
    password: str


class UnifiSystemConfig(BaseModel):
    host: str
    port: int = 443
    username: str
    password: str
    site: str = "default"
    verify_ssl: bool = False


class UNASSystemConfig(BaseModel):
    host: str
    port: int = 22
    username: str
    password: str | None = None
    key_path: str | None = None
    api_url: str | None = None


class SystemConfig(BaseModel):
    id: str
    name: str
    type: str
    enabled: bool = True
    config: (
        LinuxSystemConfig
        | DockerSystemConfig
        | QBittorrentSystemConfig
        | UnifiSystemConfig
        | UNASSystemConfig
        | dict[str, Any]
    ) = Field(default_factory=dict)


class AppConfig(BaseModel):
    poll_interval: int = 5
    systems: list[SystemConfig] = Field(default_factory=list)


class Settings(BaseSettings):
    config_path: str = "config.yaml"
    host: str = "0.0.0.0"
    port: int = 8000
    debug: bool = False
    mock_mode: bool = False

    class Config:
        env_prefix = "SERVERMONITOR_"


def load_config(config_path: str | Path) -> AppConfig:
    path = Path(config_path)
    if not path.exists():
        return AppConfig()

    with open(path) as f:
        data = yaml.safe_load(f) or {}

    systems = []
    for sys_data in data.get("systems", []):
        sys_type = sys_data.get("type", "")
        sys_config = sys_data.get("config", {})

        config_class = {
            "linux": LinuxSystemConfig,
            "docker": DockerSystemConfig,
            "qbittorrent": QBittorrentSystemConfig,
            "unifi": UnifiSystemConfig,
            "unas": UNASSystemConfig,
        }.get(sys_type)

        if config_class and sys_config:
            sys_data["config"] = config_class(**sys_config)

        systems.append(SystemConfig(**sys_data))

    return AppConfig(
        poll_interval=data.get("poll_interval", 5),
        systems=systems,
    )


settings = Settings()
