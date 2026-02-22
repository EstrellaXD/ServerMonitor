# ServerMonitor

A real-time homelab monitoring dashboard that tracks Linux servers, Docker containers, qBittorrent, UniFi networking gear, and NAS storage -- all from a single page.

## Features

- **Real-time metrics** via WebSocket (5-second updates, HTTP polling fallback)
- **5 collector types** -- Linux (SSH), Docker, qBittorrent, UniFi, UNAS/NAS
- **Status at a glance** -- healthy / warning / critical / offline per system
- **Drill-down views** -- click any system card for detailed metrics
- **Dark / Light mode** with persistence
- **PWA support** -- installable, offline-capable
- **Mock mode** for development without real infrastructure

## Tech Stack

| Layer    | Technology                              |
|----------|-----------------------------------------|
| Frontend | Vue 3, TypeScript, TailwindCSS, Pinia   |
| Backend  | FastAPI, asyncio, WebSocket             |
| Deploy   | Docker Compose, Nginx reverse proxy     |

## Quick Start

### 1. Configure your systems

```bash
cp backend/config.example.yaml backend/config.yaml
```

Edit `backend/config.yaml` with your hosts, credentials, and which collectors to enable. See [Configuration](#configuration) below.

### 2. Run with Docker Compose

```bash
docker compose up --build -d
```

Open `http://<host-ip>:4829` in your browser.

### 3. Or run locally for development

**Backend:**

```bash
cd backend
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --host 0.0.0.0 --port 8742
```

**Frontend (separate terminal):**

```bash
cd frontend
npm install
npm run dev
```

The dev server proxies `/api` and `/ws` to `localhost:8742` automatically.

## Configuration

All monitored systems are defined in `backend/config.yaml`:

```yaml
poll_interval: 5  # seconds between collection cycles

systems:
  # Linux server via SSH
  - id: linux-server-1
    name: "My Server"
    type: linux
    enabled: true
    config:
      host: 192.168.1.100
      port: 22
      username: monitor
      password: changeme        # or use key_path for SSH keys

  # Docker host
  - id: docker-host
    name: "Docker Host"
    type: docker
    enabled: true
    config:
      host: tcp://192.168.1.100:2375   # omit for local socket

  # qBittorrent
  - id: qbittorrent
    name: "qBittorrent"
    type: qbittorrent
    enabled: true
    config:
      url: http://192.168.1.100:8080
      username: admin
      password: changeme

  # UniFi Controller
  - id: unifi
    name: "UniFi Controller"
    type: unifi
    enabled: true
    config:
      host: 192.168.1.1
      port: 443
      api_key: "YOUR_API_KEY"
      site: default
      verify_ssl: false

  # NAS Storage (SSH-based)
  - id: unas
    name: "NAS Storage"
    type: unas
    enabled: true
    config:
      host: 192.168.1.200
      port: 22
      username: root
      password: changeme
```

Set `enabled: false` on any system you don't use.

### Environment Variables

| Variable                    | Default      | Description                 |
|-----------------------------|--------------|-----------------------------|
| `SERVERMONITOR_CONFIG_PATH` | `config.yaml`| Path to config file         |
| `SERVERMONITOR_HOST`        | `0.0.0.0`   | Backend bind address        |
| `SERVERMONITOR_PORT`        | `8000`       | Backend port (Docker uses 8742) |
| `SERVERMONITOR_DEBUG`       | `false`      | Enable debug logging        |
| `SERVERMONITOR_MOCK_MODE`   | `false`      | Use mock collectors (no real infra needed) |

## What Each Collector Monitors

### Linux (SSH)

CPU (total + per-core), memory, disk usage per mount, network throughput, temperatures, uptime. Thresholds: CPU >75% warning / >90% critical, memory >85% / >95%, disk >85% / >95%.

### Docker

Per-container state, health, CPU %, memory usage/limit. Summary of running/stopped/total counts. Status degrades when containers are unhealthy or stopped.

### qBittorrent

Download/upload speeds, per-torrent progress, active transfer counts.

### UniFi

Device inventory with online/offline status, client counts. Uses the UniFi Unified API v1. Status based on offline device ratio.

### UNAS / NAS

ZFS/storage pool health, disk SMART status, temperatures, capacity. Falls back to `df` if `zpool` is unavailable.

## API

| Endpoint             | Method | Description                    |
|----------------------|--------|--------------------------------|
| `/api/systems`       | GET    | All systems with latest metrics|
| `/api/systems/{id}`  | GET    | Single system metrics          |
| `/api/reload`        | POST   | Hot-reload config.yaml         |
| `/health`            | GET    | Backend health check           |
| `/ws`                | WS     | Real-time metrics stream       |

## Architecture

```
Browser ──WebSocket──► Nginx (:4829) ──proxy──► FastAPI (:8742)
                         │                          │
                         │ serves Vue SPA            ├── LinuxCollector (SSH)
                         │                          ├── DockerCollector (API)
                         │                          ├── QbitCollector (HTTP)
                         │                          ├── UniFiCollector (HTTP)
                         │                          └── UNASCollector (SSH)
```

- Collectors run concurrently via `asyncio.gather()` every `poll_interval` seconds
- Metrics are stored in-memory and broadcast to all WebSocket clients on each cycle
- Frontend falls back to HTTP polling (`GET /api/systems`) if WebSocket disconnects

## Ports

| Service  | Port |
|----------|------|
| Frontend | 4829 |
| Backend  | 8742 |

## License

Private project.
