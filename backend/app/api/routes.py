from pathlib import Path

from fastapi import APIRouter

from app.config import load_config, settings
from app.models.metrics import SystemMetrics
from app.services.collector_manager import collector_manager
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


@router.post("/reload")
async def reload_config() -> dict:
    """Reload configuration from config.yaml without restarting the server."""
    try:
        # Load new config
        config_path = Path(settings.config_path)
        if not config_path.is_absolute():
            config_path = Path(__file__).parent.parent.parent / config_path

        config = load_config(config_path)

        # Stop current collectors
        await collector_manager.stop()

        # Clear metrics store
        metrics_store.clear()

        # Load new collectors
        collector_manager.load_config(config)

        # Restart collection
        await collector_manager.start()

        return {
            "status": "success",
            "message": f"Config reloaded successfully. Monitoring {len(collector_manager.collectors)} systems",
            "systems": list(collector_manager.collectors.keys())
        }
    except Exception as e:
        return {
            "status": "error",
            "message": f"Failed to reload config: {str(e)}"
        }
