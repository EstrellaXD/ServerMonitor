# Deployment Guide

This guide covers deploying ServerMonitor to a server (Raspberry Pi, VPS, or any Linux host) using Docker Compose.

## Prerequisites

- Docker and Docker Compose installed on the target machine
- Git (to clone the repo)
- Network access from the target machine to all monitored systems (SSH, Docker API, qBit Web UI, UniFi controller)

## Step 1: Get the Code on Your Server

```bash
ssh your-server
mkdir -p ~/container && cd ~/container
git clone <repo-url> ServerMonitor
cd ServerMonitor
```

Or if you're syncing from a local machine:

```bash
rsync -avz --exclude node_modules --exclude venv --exclude __pycache__ \
  ./ your-server:~/container/ServerMonitor/
```

## Step 2: Create Your Configuration

```bash
cp backend/config.example.yaml backend/config.yaml
nano backend/config.yaml
```

Fill in the actual IPs, usernames, and passwords for your systems. Only enable the collectors you need -- set `enabled: false` on the rest.

**Minimal example** (just one Linux server):

```yaml
poll_interval: 5

systems:
  - id: my-server
    name: "My Server"
    type: linux
    enabled: true
    config:
      host: 192.168.1.100
      port: 22
      username: monitor
      password: your-password
```

### Collector-Specific Setup

**Linux servers:** The SSH user needs read access to `/proc/stat`, `/proc/meminfo`, `/proc/net/dev`, and permission to run `df`, `sensors` (optional, for temperatures).

**Docker:** Either mount the local socket (default in docker-compose.yml) or expose the Docker API on a TCP port (`tcp://host:2375`). For remote Docker hosts, make sure the API port is reachable.

**qBittorrent:** Enable the Web UI in qBittorrent settings. Note the port (default 8080) and credentials.

**UniFi:** Generate an API key from your UniFi controller's settings. The controller must be reachable over HTTPS (port 443 by default). Set `verify_ssl: false` if using a self-signed certificate.

**UNAS/NAS:** SSH access with permissions to run `zpool`, `df`, `lsblk`, and `smartctl`.

## Step 3: Deploy with Docker Compose

```bash
docker compose up --build -d
```

This builds both images and starts:
- **backend** on port 8742 (host networking)
- **frontend** on port 4829 (Nginx serving the Vue SPA + reverse proxy)

Verify everything is running:

```bash
docker compose ps
docker compose logs --tail=30 backend
docker compose logs --tail=30 frontend
```

Open `http://<server-ip>:4829` in your browser.

## Step 4: Verify

1. The dashboard should load at `http://<server-ip>:4829`
2. Check the connection indicator in the top nav -- it should show "Connected"
3. System cards should appear within 5-10 seconds as the first collection cycle completes
4. If a system shows "offline", check that the target is reachable from the server and credentials are correct

Quick API test:

```bash
curl http://localhost:8742/health
curl http://localhost:8742/api/systems
```

## Updating

Pull the latest code and rebuild:

```bash
cd ~/container/ServerMonitor
git pull
docker compose up --build -d
```

To rebuild a single service:

```bash
docker compose up --build -d backend   # just the backend
docker compose up --build -d frontend  # just the frontend
```

## Raspberry Pi Notes

ServerMonitor runs well on a Raspberry Pi 4 (2GB+ RAM recommended). A few things to keep in mind:

- **First build is slow** -- npm install and pip install take a while on ARM. Subsequent rebuilds use Docker layer caching and are much faster.
- **Memory:** With all collectors enabled, expect ~150-200MB total RAM usage for both containers.
- **Docker socket:** The compose file already mounts `/var/run/docker.sock` so the Docker collector can monitor containers on the Pi itself.
- **Temperatures:** If the Pi is also one of your monitored Linux servers, `sensors` may not be available. The collector will fall back to reading `/sys/class/thermal/thermal_zone*/temp`.

### Recommended Pi Setup

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in

# Install Docker Compose plugin
sudo apt install docker-compose-plugin

# Clone and deploy
mkdir -p ~/container && cd ~/container
git clone <repo-url> ServerMonitor
cd ServerMonitor
cp backend/config.example.yaml backend/config.yaml
nano backend/config.yaml   # configure your systems
docker compose up --build -d
```

## Running Without Docker

If you prefer to run directly on the host:

### Backend

```bash
cd backend
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Set config path (optional, defaults to config.yaml in cwd)
export SERVERMONITOR_CONFIG_PATH=./config.yaml

uvicorn app.main:app --host 0.0.0.0 --port 8742
```

### Frontend (Production Build)

```bash
cd frontend
npm install
npm run build
```

Serve the `frontend/dist/` directory with any web server. You'll need to configure a reverse proxy for `/api` and `/ws` to point to the backend.

Example Nginx config:

```nginx
server {
    listen 4829;
    root /path/to/frontend/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:8742;
    }

    location /ws {
        proxy_pass http://127.0.0.1:8742;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 86400;
    }
}
```

## Mock Mode

To run with fake data (useful for development or demos):

```bash
SERVERMONITOR_MOCK_MODE=true uvicorn app.main:app --host 0.0.0.0 --port 8742
```

Or in Docker Compose, add to the backend environment:

```yaml
environment:
  - SERVERMONITOR_CONFIG_PATH=/app/config.yaml
  - SERVERMONITOR_MOCK_MODE=true
```

## Troubleshooting

### Dashboard loads but no systems appear

- Check backend logs: `docker compose logs backend`
- Verify config.yaml has at least one system with `enabled: true`
- Test connectivity from the server to the target system (e.g., `ssh user@host` or `curl http://host:port`)

### WebSocket disconnects / falls back to polling

- The frontend automatically reconnects and falls back to HTTP polling -- this is by design
- If you're behind a reverse proxy or load balancer, make sure WebSocket upgrade headers are forwarded
- Check that port 8742 is not blocked by a firewall

### A system shows "offline"

- The collector can't reach the target. Common causes:
  - Wrong IP/port/credentials in config.yaml
  - Firewall blocking the connection
  - Target service not running (e.g., Docker API not exposed, qBit Web UI disabled)
- Check the error in backend logs for specifics

### Docker collector can't connect

- **Local socket:** Make sure `/var/run/docker.sock` is mounted (it is by default in docker-compose.yml)
- **Remote TCP:** Docker API must be explicitly exposed on the target host (`dockerd -H tcp://0.0.0.0:2375`). This has no authentication -- only use on trusted networks.

### High CPU on the server

- Increase `poll_interval` in config.yaml (e.g., from 5 to 15 seconds)
- Disable collectors you don't need
- Hot-reload config without restart: `curl -X POST http://localhost:8742/api/reload`
