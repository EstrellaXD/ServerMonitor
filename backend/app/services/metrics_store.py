from datetime import datetime

from app.models.metrics import MetricsUpdate, SystemMetrics


class MetricsStore:
    """In-memory store for system metrics."""

    def __init__(self):
        self._systems: dict[str, SystemMetrics] = {}
        self._last_update: datetime | None = None

    def update(self, system_id: str, metrics: SystemMetrics) -> None:
        """Update metrics for a specific system."""
        self._systems[system_id] = metrics
        self._last_update = datetime.utcnow()

    def get(self, system_id: str) -> SystemMetrics | None:
        """Get metrics for a specific system."""
        return self._systems.get(system_id)

    def get_all(self) -> dict[str, SystemMetrics]:
        """Get all system metrics."""
        return self._systems.copy()

    def remove(self, system_id: str) -> None:
        """Remove a system from the store."""
        self._systems.pop(system_id, None)

    def clear(self) -> None:
        """Clear all metrics."""
        self._systems.clear()
        self._last_update = None

    def create_update_message(self) -> MetricsUpdate:
        """Create a metrics update message for WebSocket broadcast."""
        return MetricsUpdate(
            type="metrics_update",
            timestamp=datetime.utcnow(),
            systems=self._systems.copy(),
        )


metrics_store = MetricsStore()
