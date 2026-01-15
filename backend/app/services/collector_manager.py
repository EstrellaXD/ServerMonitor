import asyncio
from typing import Callable

from app.collectors.base import BaseCollector
from app.collectors.docker import DockerCollector
from app.collectors.linux import LinuxCollector
from app.collectors.mock import MOCK_COLLECTORS
from app.collectors.qbittorrent import QBittorrentCollector
from app.collectors.unas import UNASCollector
from app.collectors.unifi import UnifiCollector
from app.config import AppConfig, SystemConfig, settings
from app.services.metrics_store import metrics_store


class CollectorManager:
    """Manages all system collectors and orchestrates metric collection."""

    def __init__(self):
        self._collectors: dict[str, BaseCollector] = {}
        self._running = False
        self._task: asyncio.Task | None = None
        self._poll_interval = 5
        self._on_update: Callable | None = None

    def create_collector(self, system: SystemConfig) -> BaseCollector | None:
        """Create a collector based on system type."""
        # Use mock collectors if mock mode is enabled
        if settings.mock_mode:
            mock_class = MOCK_COLLECTORS.get(system.type)
            if mock_class:
                return mock_class(
                    system_id=system.id,
                    name=system.name,
                    config=system.config,
                )
            return None

        collector_classes = {
            "linux": LinuxCollector,
            "docker": DockerCollector,
            "qbittorrent": QBittorrentCollector,
            "unifi": UnifiCollector,
            "unas": UNASCollector,
        }

        collector_class = collector_classes.get(system.type)
        if not collector_class:
            return None

        return collector_class(
            system_id=system.id,
            name=system.name,
            config=system.config,
        )

    def load_config(self, config: AppConfig) -> None:
        """Load collectors from configuration."""
        self._poll_interval = config.poll_interval
        self._collectors.clear()

        for system in config.systems:
            if not system.enabled:
                continue

            collector = self.create_collector(system)
            if collector:
                self._collectors[system.id] = collector

    def set_on_update(self, callback: Callable) -> None:
        """Set callback to be called after each collection cycle."""
        self._on_update = callback

    async def collect_all(self) -> None:
        """Collect metrics from all systems concurrently."""
        if not self._collectors:
            return

        tasks = [
            collector.safe_collect() for collector in self._collectors.values()
        ]

        results = await asyncio.gather(*tasks, return_exceptions=True)

        for collector, result in zip(self._collectors.values(), results):
            if isinstance(result, Exception):
                print(f"[{collector.system_id}] Collection error: {result}")
                metrics = collector.create_error_metrics(str(result))
            else:
                print(f"[{collector.system_id}] Collection OK: status={result.status}")
                metrics = result

            metrics_store.update(collector.system_id, metrics)

        if self._on_update:
            await self._on_update()

    async def _collection_loop(self) -> None:
        """Main collection loop."""
        print(f"Collection loop started, poll_interval={self._poll_interval}")
        while self._running:
            print(f"Running collection for {len(self._collectors)} collectors...")
            await self.collect_all()
            await asyncio.sleep(self._poll_interval)

    async def start(self) -> None:
        """Start the collection loop."""
        if self._running:
            return

        self._running = True
        self._task = asyncio.create_task(self._collection_loop())

    async def stop(self) -> None:
        """Stop the collection loop and close all connections."""
        self._running = False
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
            self._task = None

        # Close all collector connections
        close_tasks = [
            collector.close() for collector in self._collectors.values()
        ]
        if close_tasks:
            await asyncio.gather(*close_tasks, return_exceptions=True)

    @property
    def collectors(self) -> dict[str, BaseCollector]:
        """Get all collectors."""
        return self._collectors


collector_manager = CollectorManager()
