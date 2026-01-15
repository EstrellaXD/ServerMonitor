from datetime import datetime

import httpx

from app.collectors.base import BaseCollector
from app.config import UnifiSystemConfig
from app.models.metrics import (
    SystemMetrics,
    SystemStatus,
    SystemType,
    UnifiDevice,
    UnifiMetrics,
)


class UnifiCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: UnifiSystemConfig):
        super().__init__(system_id, name, SystemType.UNIFI)
        self.config = config
        self._site_uuid: str | None = None

    @property
    def _headers(self) -> dict:
        return {"X-API-Key": self.config.api_key}

    @property
    def _base_url(self) -> str:
        return f"https://{self.config.host}:{self.config.port}/proxy/network/integration/v1"

    def _site_url(self, path: str) -> str:
        return f"{self._base_url}/sites/{self._site_uuid}/{path}"

    async def _resolve_site_uuid(self, client: httpx.AsyncClient) -> None:
        if self._site_uuid is not None:
            return
        resp = await client.get(f"{self._base_url}/sites", headers=self._headers)
        resp.raise_for_status()
        for site in resp.json().get("data", []):
            if site.get("internalReference") == self.config.site:
                self._site_uuid = site["id"]
                return
        raise ValueError(f"UniFi site '{self.config.site}' not found")

    async def check_connection(self) -> bool:
        try:
            async with httpx.AsyncClient(
                timeout=10, verify=self.config.verify_ssl
            ) as client:
                resp = await client.get(
                    f"{self._base_url}/info",
                    headers=self._headers,
                )
                return resp.status_code == 200
        except Exception:
            return False

    def determine_status(self, metrics: UnifiMetrics) -> SystemStatus:
        if not metrics.devices:
            return SystemStatus.HEALTHY
        offline = sum(1 for d in metrics.devices if d.status != "online")
        if offline > len(metrics.devices) / 2:
            return SystemStatus.CRITICAL
        if offline > 0:
            return SystemStatus.WARNING
        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        try:
            async with httpx.AsyncClient(
                timeout=10, verify=self.config.verify_ssl
            ) as client:
                await self._resolve_site_uuid(client)
                devices_resp = await client.get(
                    self._site_url("devices"), headers=self._headers
                )
                devices_resp.raise_for_status()
                devices_data = devices_resp.json().get("data", [])

                clients_resp = await client.get(
                    self._site_url("clients"), headers=self._headers
                )
                clients_resp.raise_for_status()
                clients_data = clients_resp.json().get("data", [])
        except Exception as e:
            return self.create_error_metrics(str(e))

        devices = []
        for device in devices_data:
            state = device.get("state", "")
            status = "online" if state.upper() == "ONLINE" else "offline"
            devices.append(
                UnifiDevice(
                    name=device.get("name", device.get("mac", "Unknown")),
                    mac=device.get("mac", ""),
                    model=device.get("model", "Unknown"),
                    status=status,
                    uptime=device.get("uptime", 0),
                )
            )

        guests_count = sum(1 for c in clients_data if c.get("isGuest", False))
        clients_count = len(clients_data) - guests_count

        unifi_metrics = UnifiMetrics(
            devices=devices,
            clients_count=clients_count,
            guests_count=guests_count,
            wan_download=0.0,
            wan_upload=0.0,
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.UNIFI,
            status=self.determine_status(unifi_metrics),
            last_updated=datetime.utcnow(),
            metrics=unifi_metrics,
        )
