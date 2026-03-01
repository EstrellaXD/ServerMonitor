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
| Backend  | Rust, Axum, Tokio, WebSocket            |
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

**Backend (requires Rust toolchain):**

```bash
cd backend
cargo run --release
```

Or with mock data (no real infrastructure needed):

```bash
cd backend
SERVERMONITOR_MOCK_MODE=true cargo run --release
```

**Frontend (separate terminal):**

```bash
cd frontend
bun install
bun run dev
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
| `SERVERMONITOR_PORT`        | `8742`       | Backend port                |
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
| `/api/health`        | GET    | Backend health check           |
| `/ws`                | WS     | Real-time metrics stream       |

## Architecture

```
Browser --WebSocket---> Nginx (:4829) --proxy---> Axum (:8742)
                         |                          |
                         | serves Vue SPA            |-- LinuxCollector (SSH via russh)
                         |                          |-- DockerCollector (bollard)
                         |                          |-- QbitCollector (reqwest)
                         |                          |-- UniFiCollector (reqwest)
                         |                          +-- UNASCollector (SSH via russh)
```

- Collectors run concurrently via Tokio tasks every `poll_interval` seconds
- Metrics are stored in-memory (`RwLock<HashMap>`) and broadcast to all WebSocket clients via `tokio::sync::broadcast`
- Frontend falls back to HTTP polling (`GET /api/systems`) if WebSocket disconnects

## Docker Image

Pre-built images are available from GitHub Container Registry:

```bash
docker pull ghcr.io/<owner>/server-monitor:latest
```

## Ports

| Service  | Port |
|----------|------|
| Frontend | 4829 |
| Backend  | 8742 |

## License

Private project.
