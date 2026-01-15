from abc import ABC, abstractmethod
from datetime import datetime
from typing import Any

from app.models.metrics import SystemMetrics, SystemStatus, SystemType


class BaseCollector(ABC):
    def __init__(self, system_id: str, name: str, system_type: SystemType):
        self.system_id = system_id
        self.name = name
        self.system_type = system_type
        self._last_metrics: SystemMetrics | None = None

    @abstractmethod
    async def collect(self) -> SystemMetrics:
        """Collect metrics from the system. Must be implemented by subclasses."""
        pass

    @abstractmethod
    async def check_connection(self) -> bool:
        """Check if the system is reachable. Must be implemented by subclasses."""
        pass

    def create_error_metrics(self, error: str) -> SystemMetrics:
        """Create a metrics object representing an error state."""
        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=self.system_type,
            status=SystemStatus.OFFLINE,
            last_updated=datetime.utcnow(),
            error=error,
            metrics={},
        )

    def determine_status(self, metrics: Any) -> SystemStatus:
        """
        Determine system status based on metrics.
        Override in subclasses for custom logic.
        """
        return SystemStatus.HEALTHY

    async def safe_collect(self) -> SystemMetrics:
        """Wrapper around collect() that handles exceptions."""
        try:
            metrics = await self.collect()
            self._last_metrics = metrics
            return metrics
        except Exception as e:
            # Close connections on failure to prevent FD leaks
            await self.close()
            return self.create_error_metrics(str(e))

    async def close(self) -> None:
        """Close any open connections. Override in subclasses if needed."""
        pass
