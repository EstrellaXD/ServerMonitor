from fastapi import APIRouter

from app.models.metrics import SystemMetrics
from app.services.metrics_store import metrics_store

router = APIRouter(prefix="/api", tags=["systems"])


@router.get("/systems")
async def get_all_systems() -> dict[str, SystemMetrics]:
    """Get metrics for all monitored systems."""
    return metrics_store.get_all()


@router.get("/systems/{system_id}")
async def get_system(system_id: str) -> SystemMetrics | dict:
    """Get metrics for a specific system."""
    metrics = metrics_store.get(system_id)
    if metrics is None:
        return {"error": "System not found"}
    return metrics


@router.get("/health")
async def health_check() -> dict:
    """Health check endpoint."""
    return {"status": "ok"}
