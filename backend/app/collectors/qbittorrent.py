from datetime import datetime

import httpx

from app.collectors.base import BaseCollector
from app.config import QBittorrentSystemConfig
from app.models.metrics import (
    QBittorrentMetrics,
    SystemMetrics,
    SystemStatus,
    SystemType,
    TorrentInfo,
)


class QBittorrentCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: QBittorrentSystemConfig):
        super().__init__(system_id, name, SystemType.QBITTORRENT)
        self.config = config
        self._session_cookie: str | None = None

    async def _login(self, client: httpx.AsyncClient) -> bool:
        """Authenticate with qBittorrent Web API."""
        response = await client.post(
            f"{self.config.url}/api/v2/auth/login",
            data={
                "username": self.config.username,
                "password": self.config.password,
            },
        )

        if response.status_code == 200 and response.text == "Ok.":
            self._session_cookie = response.cookies.get("SID")
            return True
        return False

    async def check_connection(self) -> bool:
        """Check if qBittorrent Web API is accessible."""
        try:
            async with httpx.AsyncClient(timeout=10, verify=False) as client:
                return await self._login(client)
        except Exception:
            return False

    def determine_status(self, metrics: QBittorrentMetrics) -> SystemStatus:
        """Determine status based on qBittorrent metrics."""
        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        """Collect qBittorrent metrics."""
        async with httpx.AsyncClient(timeout=10, verify=False) as client:
            if not await self._login(client):
                return self.create_error_metrics("Failed to authenticate")

            cookies = {"SID": self._session_cookie}

            transfer_resp = await client.get(
                f"{self.config.url}/api/v2/transfer/info",
                cookies=cookies,
            )
            transfer_data = transfer_resp.json()

            torrents_resp = await client.get(
                f"{self.config.url}/api/v2/torrents/info",
                cookies=cookies,
            )
            torrents_data = torrents_resp.json()

        download_speed = transfer_data.get("dl_info_speed", 0)
        upload_speed = transfer_data.get("up_info_speed", 0)

        active_downloads = 0
        active_uploads = 0
        torrents = []

        for torrent in torrents_data:
            state = torrent.get("state", "")

            if state in ("downloading", "forcedDL", "metaDL", "queuedDL"):
                active_downloads += 1
            elif state in ("uploading", "forcedUP", "stalledUP", "queuedUP"):
                active_uploads += 1

            torrents.append(
                TorrentInfo(
                    name=torrent.get("name", "Unknown"),
                    size=torrent.get("size", 0),
                    progress=torrent.get("progress", 0) * 100,
                    download_speed=torrent.get("dlspeed", 0),
                    upload_speed=torrent.get("upspeed", 0),
                    state=state,
                    eta=torrent.get("eta") if torrent.get("eta", 8640000) < 8640000 else None,
                )
            )

        qbit_metrics = QBittorrentMetrics(
            download_speed=download_speed,
            upload_speed=upload_speed,
            active_downloads=active_downloads,
            active_uploads=active_uploads,
            total_torrents=len(torrents),
            torrents=torrents[:50],
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.QBITTORRENT,
            status=self.determine_status(qbit_metrics),
            last_updated=datetime.utcnow(),
            metrics=qbit_metrics,
        )
