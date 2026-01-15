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
        self._csrf_token: str | None = None
        self._cookies: dict = {}

    def _get_base_url(self) -> str:
        """Get the base URL for the UniFi Controller."""
        return f"https://{self.config.host}:{self.config.port}"

    async def _login(self, client: httpx.AsyncClient) -> bool:
        """Authenticate with UniFi Controller."""
        base_url = self._get_base_url()

        response = await client.post(
            f"{base_url}/api/auth/login",
            json={
                "username": self.config.username,
                "password": self.config.password,
            },
        )

        if response.status_code == 200:
            self._cookies = dict(response.cookies)
            self._csrf_token = response.headers.get("x-csrf-token")
            return True

        response = await client.post(
            f"{base_url}/api/login",
            json={
                "username": self.config.username,
                "password": self.config.password,
            },
        )

        if response.status_code == 200:
            self._cookies = dict(response.cookies)
            return True

        return False

    async def check_connection(self) -> bool:
        """Check if UniFi Controller is accessible."""
        try:
            async with httpx.AsyncClient(
                timeout=10, verify=self.config.verify_ssl
            ) as client:
                return await self._login(client)
        except Exception:
            return False

    def determine_status(self, metrics: UnifiMetrics) -> SystemStatus:
        """Determine status based on UniFi metrics."""
        offline_devices = sum(
            1 for d in metrics.devices if d.status != "online"
        )

        if offline_devices > len(metrics.devices) / 2:
            return SystemStatus.CRITICAL
        if offline_devices > 0:
            return SystemStatus.WARNING

        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        """Collect UniFi metrics."""
        async with httpx.AsyncClient(
            timeout=10, verify=self.config.verify_ssl
        ) as client:
            if not await self._login(client):
                return self.create_error_metrics("Failed to authenticate")

            base_url = self._get_base_url()
            site = self.config.site

            headers = {}
            if self._csrf_token:
                headers["x-csrf-token"] = self._csrf_token

            try:
                devices_resp = await client.get(
                    f"{base_url}/api/s/{site}/stat/device",
                    cookies=self._cookies,
                    headers=headers,
                )
                devices_data = devices_resp.json().get("data", [])
            except Exception:
                devices_resp = await client.get(
                    f"{base_url}/proxy/network/api/s/{site}/stat/device",
                    cookies=self._cookies,
                    headers=headers,
                )
                devices_data = devices_resp.json().get("data", [])

            try:
                clients_resp = await client.get(
                    f"{base_url}/api/s/{site}/stat/sta",
                    cookies=self._cookies,
                    headers=headers,
                )
                clients_data = clients_resp.json().get("data", [])
            except Exception:
                clients_resp = await client.get(
                    f"{base_url}/proxy/network/api/s/{site}/stat/sta",
                    cookies=self._cookies,
                    headers=headers,
                )
                clients_data = clients_resp.json().get("data", [])

        devices = []
        wan_download = 0.0
        wan_upload = 0.0

        for device in devices_data:
            status = "online" if device.get("state", 0) == 1 else "offline"

            devices.append(
                UnifiDevice(
                    name=device.get("name", device.get("mac", "Unknown")),
                    mac=device.get("mac", ""),
                    model=device.get("model", "Unknown"),
                    status=status,
                    uptime=device.get("uptime", 0),
                )
            )

            if device.get("type") == "ugw" or device.get("type") == "udm":
                wan_stats = device.get("wan1", {}) or device.get("uplink", {})
                wan_download = wan_stats.get("rx_rate", 0) / 1_000_000
                wan_upload = wan_stats.get("tx_rate", 0) / 1_000_000

        guests_count = sum(1 for c in clients_data if c.get("is_guest", False))
        clients_count = len(clients_data) - guests_count

        unifi_metrics = UnifiMetrics(
            devices=devices,
            clients_count=clients_count,
            guests_count=guests_count,
            wan_download=round(wan_download, 2),
            wan_upload=round(wan_upload, 2),
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.UNIFI,
            status=self.determine_status(unifi_metrics),
            last_updated=datetime.utcnow(),
            metrics=unifi_metrics,
        )
