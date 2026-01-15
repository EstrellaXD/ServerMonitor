"""
Mock server for testing the ServerMonitor UI.
Generates fake metrics data for all system types.

Run with: uv run python mock_server.py
"""

import asyncio
import json
import random
from datetime import datetime
from typing import Any

from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware

app = FastAPI(title="ServerMonitor Mock Server")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Store connected WebSocket clients
clients: list[WebSocket] = []

# Mock system definitions
MOCK_SYSTEMS = [
    {"id": "proxmox-main", "name": "Proxmox Main", "type": "linux"},
    {"id": "docker-host", "name": "Docker Host", "type": "docker"},
    {"id": "qbittorrent", "name": "qBittorrent", "type": "qbittorrent"},
    {"id": "unifi-controller", "name": "UniFi Network", "type": "unifi"},
    {"id": "truenas", "name": "TrueNAS Storage", "type": "unas"},
    {"id": "pi-hole", "name": "Pi-hole DNS", "type": "linux"},
]


def generate_linux_metrics() -> dict[str, Any]:
    """Generate mock Linux system metrics."""
    cpu_percent = random.uniform(5, 85)
    mem_percent = random.uniform(30, 75)

    return {
        "cpu": {
            "percent": cpu_percent,
            "cores": [random.uniform(0, 100) for _ in range(8)],
            "load_avg": [random.uniform(0.5, 4.0) for _ in range(3)],
        },
        "memory": {
            "total": 32 * 1024 * 1024 * 1024,  # 32GB
            "used": int(32 * 1024 * 1024 * 1024 * mem_percent / 100),
            "available": int(32 * 1024 * 1024 * 1024 * (100 - mem_percent) / 100),
            "percent": mem_percent,
        },
        "disks": [
            {
                "mount": "/",
                "total": 500 * 1024 * 1024 * 1024,
                "used": int(500 * 1024 * 1024 * 1024 * random.uniform(0.3, 0.7)),
                "free": int(500 * 1024 * 1024 * 1024 * random.uniform(0.3, 0.7)),
                "percent": random.uniform(30, 70),
            },
            {
                "mount": "/data",
                "total": 2 * 1024 * 1024 * 1024 * 1024,
                "used": int(2 * 1024 * 1024 * 1024 * 1024 * random.uniform(0.4, 0.8)),
                "free": int(2 * 1024 * 1024 * 1024 * 1024 * random.uniform(0.2, 0.6)),
                "percent": random.uniform(40, 80),
            },
        ],
        "network": {
            "bytes_sent": random.randint(1000000000, 9000000000),
            "bytes_recv": random.randint(5000000000, 50000000000),
            "upload_rate": random.uniform(100000, 5000000),
            "download_rate": random.uniform(500000, 20000000),
        },
        "temperatures": [
            {"label": "CPU Package", "current": random.uniform(35, 65), "high": 80, "critical": 100},
            {"label": "Core 0", "current": random.uniform(35, 60), "high": 80, "critical": 100},
            {"label": "Core 1", "current": random.uniform(35, 60), "high": 80, "critical": 100},
            {"label": "NVMe", "current": random.uniform(30, 50), "high": 70, "critical": 85},
        ],
        "uptime": random.randint(86400, 8640000),  # 1 day to 100 days
    }


def generate_docker_metrics() -> dict[str, Any]:
    """Generate mock Docker metrics."""
    containers = [
        {"name": "nginx-proxy", "image": "nginx:latest", "state": "running"},
        {"name": "postgres-db", "image": "postgres:15", "state": "running"},
        {"name": "redis-cache", "image": "redis:7-alpine", "state": "running"},
        {"name": "grafana", "image": "grafana/grafana:latest", "state": "running"},
        {"name": "prometheus", "image": "prom/prometheus:latest", "state": "running"},
        {"name": "backup-job", "image": "restic/restic:latest", "state": "exited"},
    ]

    container_metrics = []
    running = 0
    stopped = 0

    for c in containers:
        is_running = c["state"] == "running"
        if is_running:
            running += 1
        else:
            stopped += 1

        container_metrics.append({
            "id": c["name"][:12],
            "name": c["name"],
            "image": c["image"],
            "status": "healthy" if is_running else "none",
            "state": c["state"],
            "cpu_percent": random.uniform(0.5, 15) if is_running else 0,
            "memory_usage": random.randint(50000000, 500000000) if is_running else 0,
            "memory_limit": 1024 * 1024 * 1024,
            "memory_percent": random.uniform(5, 50) if is_running else 0,
        })

    return {
        "containers": container_metrics,
        "running_count": running,
        "stopped_count": stopped,
        "total_count": len(containers),
    }


def generate_qbittorrent_metrics() -> dict[str, Any]:
    """Generate mock qBittorrent metrics."""
    torrents = [
        {"name": "Ubuntu 24.04 LTS", "size": 4.5 * 1024 * 1024 * 1024, "state": "uploading"},
        {"name": "Arch Linux 2024.01", "size": 800 * 1024 * 1024, "state": "uploading"},
        {"name": "Debian 12 DVD", "size": 3.7 * 1024 * 1024 * 1024, "state": "downloading"},
        {"name": "Linux Mint 21.3", "size": 2.8 * 1024 * 1024 * 1024, "state": "stalledUP"},
    ]

    torrent_list = []
    active_dl = 0
    active_ul = 0

    for t in torrents:
        progress = 100 if "UP" in t["state"] or t["state"] == "uploading" else random.uniform(20, 95)
        dl_speed = random.randint(1000000, 10000000) if t["state"] == "downloading" else 0
        ul_speed = random.randint(100000, 2000000) if "UP" in t["state"] or t["state"] == "uploading" else 0

        if t["state"] == "downloading":
            active_dl += 1
        if "UP" in t["state"] or t["state"] == "uploading":
            active_ul += 1

        torrent_list.append({
            "name": t["name"],
            "size": int(t["size"]),
            "progress": progress,
            "download_speed": dl_speed,
            "upload_speed": ul_speed,
            "state": t["state"],
            "eta": random.randint(300, 7200) if t["state"] == "downloading" else None,
        })

    return {
        "download_speed": sum(t["download_speed"] for t in torrent_list),
        "upload_speed": sum(t["upload_speed"] for t in torrent_list),
        "active_downloads": active_dl,
        "active_uploads": active_ul,
        "total_torrents": len(torrents),
        "torrents": torrent_list,
    }


def generate_unifi_metrics() -> dict[str, Any]:
    """Generate mock UniFi metrics."""
    devices = [
        {"name": "UDM Pro", "model": "UDM-Pro", "type": "ugw"},
        {"name": "Switch 24 PoE", "model": "USW-24-POE", "type": "usw"},
        {"name": "AP Living Room", "model": "U6-Pro", "type": "uap"},
        {"name": "AP Office", "model": "U6-Lite", "type": "uap"},
        {"name": "AP Garage", "model": "U6-Mesh", "type": "uap"},
    ]

    device_list = []
    for d in devices:
        device_list.append({
            "name": d["name"],
            "mac": ":".join([f"{random.randint(0, 255):02x}" for _ in range(6)]),
            "model": d["model"],
            "status": "online",
            "uptime": random.randint(86400, 2592000),
        })

    return {
        "devices": device_list,
        "clients_count": random.randint(15, 35),
        "guests_count": random.randint(0, 5),
        "wan_download": random.uniform(50, 500),
        "wan_upload": random.uniform(10, 100),
    }


def generate_unas_metrics() -> dict[str, Any]:
    """Generate mock UNAS storage metrics."""
    pools = [
        {"name": "tank", "total": 20 * 1024 * 1024 * 1024 * 1024},  # 20TB
        {"name": "fast", "total": 2 * 1024 * 1024 * 1024 * 1024},   # 2TB NVMe
    ]

    pool_list = []
    total_cap = 0
    used_cap = 0

    for p in pools:
        used_pct = random.uniform(40, 75)
        used = int(p["total"] * used_pct / 100)
        pool_list.append({
            "name": p["name"],
            "status": "online",
            "total": p["total"],
            "used": used,
            "available": p["total"] - used,
            "percent": used_pct,
        })
        total_cap += p["total"]
        used_cap += used

    disks = [
        {"name": "sda", "model": "WDC WD80EFAX", "status": "healthy"},
        {"name": "sdb", "model": "WDC WD80EFAX", "status": "healthy"},
        {"name": "sdc", "model": "WDC WD80EFAX", "status": "healthy"},
        {"name": "sdd", "model": "WDC WD80EFAX", "status": "healthy"},
        {"name": "nvme0n1", "model": "Samsung 980 Pro", "status": "healthy"},
    ]

    disk_list = []
    for d in disks:
        disk_list.append({
            "name": d["name"],
            "model": d["model"],
            "serial": f"WD-{random.randint(100000, 999999)}",
            "status": d["status"],
            "temperature": random.uniform(28, 42),
        })

    return {
        "pools": pool_list,
        "disks": disk_list,
        "total_capacity": total_cap,
        "used_capacity": used_cap,
    }


def generate_metrics_for_system(system: dict) -> dict[str, Any]:
    """Generate metrics based on system type."""
    generators = {
        "linux": generate_linux_metrics,
        "docker": generate_docker_metrics,
        "qbittorrent": generate_qbittorrent_metrics,
        "unifi": generate_unifi_metrics,
        "unas": generate_unas_metrics,
    }

    generator = generators.get(system["type"], generate_linux_metrics)
    metrics = generator()

    # Randomly make one system have warning status
    status = "healthy"
    if system["type"] == "linux" and metrics["cpu"]["percent"] > 70:
        status = "warning"
    elif system["type"] == "docker" and metrics["stopped_count"] > 2:
        status = "warning"

    return {
        "id": system["id"],
        "name": system["name"],
        "type": system["type"],
        "status": status,
        "last_updated": datetime.utcnow().isoformat(),
        "error": None,
        "metrics": metrics,
    }


def generate_all_metrics() -> dict[str, Any]:
    """Generate metrics for all mock systems."""
    systems = {}
    for system in MOCK_SYSTEMS:
        systems[system["id"]] = generate_metrics_for_system(system)
    return systems


@app.get("/")
async def root():
    return {"name": "ServerMonitor Mock Server", "version": "1.0.0"}


@app.get("/api/systems")
async def get_systems():
    return generate_all_metrics()


@app.get("/api/systems/{system_id}")
async def get_system(system_id: str):
    for system in MOCK_SYSTEMS:
        if system["id"] == system_id:
            return generate_metrics_for_system(system)
    return {"error": "System not found"}


@app.get("/api/health")
async def health():
    return {"status": "ok"}


@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    await websocket.accept()
    clients.append(websocket)

    try:
        # Send initial data
        initial_data = {
            "type": "metrics_update",
            "timestamp": datetime.utcnow().isoformat(),
            "systems": generate_all_metrics(),
        }
        await websocket.send_json(initial_data)

        # Keep connection alive and send updates
        while True:
            try:
                # Wait for ping or timeout
                data = await asyncio.wait_for(websocket.receive_text(), timeout=5.0)
                if data == "ping":
                    await websocket.send_text("pong")
            except asyncio.TimeoutError:
                # Send metrics update every 5 seconds
                update = {
                    "type": "metrics_update",
                    "timestamp": datetime.utcnow().isoformat(),
                    "systems": generate_all_metrics(),
                }
                await websocket.send_json(update)

    except WebSocketDisconnect:
        pass
    finally:
        if websocket in clients:
            clients.remove(websocket)


if __name__ == "__main__":
    import uvicorn

    print("\n" + "=" * 50)
    print("  ServerMonitor Mock Server")
    print("=" * 50)
    print("\n  Backend:  http://localhost:8000")
    print("  API Docs: http://localhost:8000/docs")
    print("\n  Start frontend in another terminal:")
    print("  cd frontend && npm run dev")
    print("\n  Then open: http://localhost:3000")
    print("=" * 50 + "\n")

    uvicorn.run(app, host="0.0.0.0", port=8000)
