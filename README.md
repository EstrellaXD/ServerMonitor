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
| Backend  | Go 1.23, chi, gorilla/websocket         |
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

**Backend (requires Go 1.23+):**

```bash
cd backend
go run ./cmd/server-monitor
```

Or with mock data (no real infrastructure needed):

```bash
cd backend
SERVERMONITOR_MOCK_MODE=true go run ./cmd/server-monitor
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
allowed_hosts: []  # extra Host names to accept (see Host Allowlist)

systems:
  # Linux server via SSH
  - id: linux-server-1
    name: "My Server"
    type: linux
    enabled: true
    controls: [power, process, service, fan]
    config:
      host: 192.168.1.100
      port: 22
      username: monitor
      services: [servermonitor]  # systemd units to show and control
      password: changeme        # or use key_path for SSH keys

  # Docker host
  - id: docker-host
    name: "Docker Host"
    type: docker
    enabled: true
    controls: [container, recreate]
    config:
      host: tcp://192.168.1.100:2375   # omit for local socket

  # qBittorrent
  - id: qbittorrent
    name: "qBittorrent"
    type: qbittorrent
    enabled: true
    controls: [torrent, delete_files]
    config:
      url: http://192.168.1.100:8080
      username: admin
      password: changeme

  # UniFi Controller
  - id: unifi
    name: "UniFi Controller"
    type: unifi
    enabled: true
    controls: [device, client]
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
    controls: [power]
    config:
      host: 192.168.1.200
      port: 22
      username: root
      password: changeme
```

Set `enabled: false` on any system you don't use.

### Host Allowlist

The server answers only requests whose `Host` header (port removed) is an IP
address, `localhost`, a name that ends in `.local`, or an entry of the
top-level `allowed_hosts` list (case-insensitive). All other requests get
403 `{"error": "host not allowed"}`. This stops DNS rebinding, where a web
page under the attacker's name resolves to this server. Add a name to
`allowed_hosts` when you open the dashboard by a DNS name, for example:

```yaml
allowed_hosts: [monitor.home.arpa]
```

`/ws` also checks the `Origin` header: no Origin (native clients) is allowed,
and a browser Origin must pass the same Host rule. `POST /api/reload` applies
a changed list.

### Control Actions

The API has no authentication. The `controls` list of each system is the
allowlist: an action whose control is not in the list returns 403. Omit
`controls` to keep a system read-only. An unknown value is a config error.

| Type        | Control        | Actions                                                     |
|-------------|----------------|-------------------------------------------------------------|
| docker      | `container`    | start, stop, restart, pause, unpause, kill                  |
| docker      | `recreate`     | recreate (pull the image again, replace the container)      |
| qbittorrent | `torrent`      | resume, pause, recheck, top/bottom/increase/decrease priority, delete |
| qbittorrent | `delete_files` | delete with `delete_files: true` (also needs `torrent`)     |
| linux       | `power`        | reboot, poweroff                                            |
| linux       | `process`      | kill (TERM or KILL) a PID from the latest process list      |
| linux       | `service`      | start, stop, restart a unit from `config.services`          |
| linux       | `fan`          | set the `pwmfan` duty cycle, return it to automatic control |
| unas        | `power`        | reboot, poweroff                                            |
| unifi       | `device`       | restart a device                                            |
| unifi       | `client`       | kick, block, unblock a client                               |

Docker container logs (`GET /api/systems/{id}/docker/logs`) are read-only
and need no control.

#### sudo on Linux hosts

Linux and UNAS actions run over SSH as `sh -c '...'`. Each privileged step is
`sudo -n <command>`, so the SSH user needs passwordless sudo for exactly
these commands (a `root` user runs them without sudo). Put a rule like this
in `/etc/sudoers.d/servermonitor` (edit with `visudo -f`), and remove the
lines for controls you do not enable:

```sudoers
# power
monitor ALL=(root) NOPASSWD: /usr/bin/systemctl reboot, /usr/bin/systemctl poweroff
# process: any PID; the backend allows only PIDs from the latest process list
monitor ALL=(root) NOPASSWD: /usr/bin/kill -s TERM *, /usr/bin/kill -s KILL *
# service: one line per unit in config.services
monitor ALL=(root) NOPASSWD: /usr/bin/systemctl start servermonitor, /usr/bin/systemctl stop servermonitor, /usr/bin/systemctl restart servermonitor
# fan: replace hwmon2 and thermal_zone0 with the indices of the host
monitor ALL=(root) NOPASSWD: /usr/bin/tee /sys/class/hwmon/hwmon2/pwm1, /usr/bin/tee /sys/class/hwmon/hwmon2/pwm1_enable, \
  /usr/bin/tee /sys/class/thermal/thermal_zone0/policy, \
  /usr/bin/tee /run/servermonitor-fan.state, /usr/bin/tee -a /run/servermonitor-fan.state, \
  /usr/bin/rm -f /run/servermonitor-fan.state, \
  /usr/bin/systemctl stop nvfancontrol, /usr/bin/systemctl start nvfancontrol
```

Use exact paths, not wildcards, for `tee`: in sudoers, `*` also matches
spaces and `/`, so `tee /sys/*` would allow writes to any file. Find the
indices with `grep -l pwmfan /sys/class/hwmon/*/name` and
`ls -l /sys/class/thermal/thermal_zone*/cdev*`.

Fan control (`fan_set`) stops `nvfancontrol` if it runs, sets the thermal
zones bound to the `pwm-fan` cooling device to `user_space`, and writes
`pwm1`. It records these changes in `/run/servermonitor-fan.state`;
`fan_auto` restores them. The fan mode shows `manual` while that file exists.

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
| `/api/systems/{id}/actions` | POST | Run a control action: `{"action", "target", "params"?}` |
| `/api/systems/{id}/docker/logs?container=&tail=` | GET | Container logs (tail default 200, max 2000) |
| `/api/health`        | GET    | Backend health check           |
| `/ws`                | WS     | Real-time metrics stream       |

## Architecture

```
Browser --WebSocket---> Nginx (:4829) --proxy---> chi router (:8742)
                         |                          |
                         | serves Vue SPA            |-- LinuxCollector (golang.org/x/crypto/ssh)
                         |                          |-- DockerCollector (docker/docker/client)
                         |                          |-- QbitCollector (net/http + cookiejar)
                         |                          |-- UniFiCollector (net/http)
                         |                          +-- UNASCollector (golang.org/x/crypto/ssh)
```

- Collectors run serially in a single goroutine every `poll_interval` seconds
- Metrics are stored in-memory (`sync.RWMutex` + `map`) and fanned out to all WebSocket clients via a custom buffered broadcaster
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
