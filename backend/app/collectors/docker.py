from datetime import datetime

import docker
from docker.errors import DockerException

from app.collectors.base import BaseCollector
from app.config import DockerSystemConfig
from app.models.metrics import (
    ContainerMetrics,
    DockerMetrics,
    SystemMetrics,
    SystemStatus,
    SystemType,
)


class DockerCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: DockerSystemConfig):
        super().__init__(system_id, name, SystemType.DOCKER)
        self.config = config
        self._client: docker.DockerClient | None = None

    def _get_client(self) -> docker.DockerClient:
        """Get or create Docker client."""
        if self._client is None:
            if self.config.host:
                self._client = docker.DockerClient(
                    base_url=self.config.host,
                    tls=self.config.tls,
                )
            else:
                self._client = docker.DockerClient(
                    base_url=self.config.socket,
                )
        return self._client

    async def check_connection(self) -> bool:
        """Check if Docker daemon is accessible."""
        try:
            client = self._get_client()
            client.ping()
            return True
        except DockerException:
            return False

    def _calculate_cpu_percent(self, stats: dict) -> float:
        """Calculate CPU percentage from container stats."""
        try:
            cpu_delta = (
                stats["cpu_stats"]["cpu_usage"]["total_usage"]
                - stats["precpu_stats"]["cpu_usage"]["total_usage"]
            )
            system_delta = (
                stats["cpu_stats"]["system_cpu_usage"]
                - stats["precpu_stats"]["system_cpu_usage"]
            )

            if system_delta > 0 and cpu_delta > 0:
                cpu_count = stats["cpu_stats"]["online_cpus"]
                return (cpu_delta / system_delta) * cpu_count * 100.0
        except (KeyError, ZeroDivisionError):
            pass
        return 0.0

    def _calculate_memory_percent(self, stats: dict) -> tuple[int, int, float]:
        """Calculate memory usage from container stats."""
        try:
            usage = stats["memory_stats"]["usage"]
            limit = stats["memory_stats"]["limit"]
            percent = (usage / limit) * 100.0 if limit > 0 else 0.0
            return usage, limit, percent
        except KeyError:
            return 0, 0, 0.0

    def determine_status(self, metrics: DockerMetrics) -> SystemStatus:
        """Determine system status based on Docker metrics."""
        if metrics.total_count == 0:
            return SystemStatus.HEALTHY

        unhealthy_count = 0
        high_resource_count = 0

        for container in metrics.containers:
            if container.state not in ("running", "paused"):
                continue
            if container.status == "unhealthy":
                unhealthy_count += 1
            if container.cpu_percent > 90 or container.memory_percent > 90:
                high_resource_count += 1

        if unhealthy_count > 0 or metrics.stopped_count > metrics.running_count:
            return SystemStatus.WARNING
        if high_resource_count > 0:
            return SystemStatus.WARNING

        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        """Collect Docker container metrics."""
        client = self._get_client()

        containers = client.containers.list(all=True)
        container_metrics = []
        running_count = 0
        stopped_count = 0

        for container in containers:
            state = container.status

            if state == "running":
                running_count += 1
                try:
                    stats = container.stats(stream=False)
                    cpu_percent = self._calculate_cpu_percent(stats)
                    mem_usage, mem_limit, mem_percent = self._calculate_memory_percent(
                        stats
                    )
                except Exception:
                    cpu_percent = 0.0
                    mem_usage = mem_limit = 0
                    mem_percent = 0.0
            else:
                stopped_count += 1
                cpu_percent = 0.0
                mem_usage = mem_limit = 0
                mem_percent = 0.0

            health_status = "none"
            if container.attrs.get("State", {}).get("Health"):
                health_status = container.attrs["State"]["Health"].get(
                    "Status", "none"
                )

            container_metrics.append(
                ContainerMetrics(
                    id=container.short_id,
                    name=container.name,
                    image=container.image.tags[0] if container.image.tags else "unknown",
                    status=health_status,
                    state=state,
                    cpu_percent=round(cpu_percent, 2),
                    memory_usage=mem_usage,
                    memory_limit=mem_limit,
                    memory_percent=round(mem_percent, 2),
                )
            )

        docker_metrics = DockerMetrics(
            containers=container_metrics,
            running_count=running_count,
            stopped_count=stopped_count,
            total_count=len(containers),
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.DOCKER,
            status=self.determine_status(docker_metrics),
            last_updated=datetime.utcnow(),
            metrics=docker_metrics,
        )
