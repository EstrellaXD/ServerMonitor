from contextlib import asynccontextmanager
from pathlib import Path

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router as api_router
from app.api.websocket import broadcast_metrics
from app.api.websocket import router as ws_router
from app.config import load_config, settings
from app.services.collector_manager import collector_manager


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    config_path = Path(settings.config_path)
    if not config_path.is_absolute():
        config_path = Path(__file__).parent.parent / config_path

    config = load_config(config_path)
    collector_manager.load_config(config)
    collector_manager.set_on_update(broadcast_metrics)

    await collector_manager.start()
    print(f"Started monitoring {len(collector_manager.collectors)} systems")

    yield

    await collector_manager.stop()
    print("Stopped monitoring")


app = FastAPI(
    title="ServerMonitor",
    description="Homelab server monitoring dashboard",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(api_router)
app.include_router(ws_router)


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "name": "ServerMonitor",
        "version": "1.0.0",
        "docs": "/docs",
    }
