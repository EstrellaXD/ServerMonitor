# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ServerMonitor is a homelab server monitoring dashboard with real-time metrics visualization.

**Tech Stack:**
- Frontend: Vue 3 (Composition API) + TypeScript + TailwindCSS
- Backend: Go 1.23 (chi router + gorilla/websocket) with goroutine-based concurrency
- Data: In-memory storage (no persistence)

**Monitored Systems:**
- Linux servers via SSH (`golang.org/x/crypto/ssh`)
- Docker containers (`github.com/docker/docker/client`)
- qBittorrent (Web API via `net/http` + `cookiejar`)
- UniFi router (API via `net/http`)
- UNAS storage (SSH via `golang.org/x/crypto/ssh`)

## Build and Development Commands

### Backend
```bash
cd backend
go build -o server-monitor ./cmd/server-monitor   # Build
go run ./cmd/server-monitor                        # Run
go test ./...                                      # Tests
go vet ./...                                       # Vet

# Mock mode (no real infrastructure needed)
SERVERMONITOR_MOCK_MODE=true go run ./cmd/server-monitor
```

### Frontend
```bash
cd frontend
bun install
bun run dev          # Development server
bun run build        # Production build
bun run preview      # Preview production build
```

### Full Stack (Docker)
```bash
docker compose up --build
```

## Architecture

```
backend/
├── go.mod                          # module github.com/EstrellaXD/ServerMonitor/backend
├── cmd/server-monitor/main.go      # Entry: signal handling, lifecycle, http.Server
└── internal/
    ├── api/
    │   ├── routes.go               # chi router, AppState, CORS, SPA fallback
    │   ├── actions.go              # Docker / qBittorrent action handlers
    │   └── websocket.go            # gorilla/websocket upgrade + per-conn pump
    ├── broadcast/broadcast.go      # Subscriber registry (lag-tolerant fan-out)
    ├── collectors/
    │   ├── collector.go            # Collector interface + factory
    │   ├── ssh.go                  # Shared SSH abstraction (Linux + UNAS)
    │   ├── linux.go, docker.go, qbittorrent.go, unifi.go, unas.go, mock.go
    ├── config/                     # settings, systems, YAML loader
    ├── manager/manager.go          # Poll loop, reload-safe collector set
    ├── models/                     # Wire-format structs (status, metrics, messages)
    └── store/metrics_store.go      # sync.RWMutex + map cache

frontend/src/
├── components/
│   ├── dashboard/       # Overview grid and system cards
│   └── detail/          # Per-system drill-down views
├── composables/         # useWebSocket, useTheme, useMetrics
├── stores/metrics.ts    # Pinia store for system state
└── types/metrics.ts     # TypeScript interfaces
```

## Key Patterns

- **Collectors:** Each system type implements the `Collector` interface with `Collect(ctx)` and `Close()` methods
- **SSH Sharing:** `SSHConnection` abstracts SSH for both Linux and UNAS collectors (lazy reconnect, single mutex)
- **State:** `*api.AppState` passed by pointer (GC-managed; no `Arc` ceremony)
- **Real-time:** Custom `Broadcaster` does serialize-once fan-out to a registry of buffered channels; slow consumers are dropped
- **Status:** Systems are `healthy`, `warning`, `critical`, or `offline`
- **Theming:** Dark/Light mode via TailwindCSS `class` strategy

## Configuration

Systems are defined in `backend/config.yaml`:
```yaml
poll_interval: 5
systems:
  - id: linux-server-1
    name: "My Server"
    type: linux
    config:
      host: 192.168.1.100
      username: monitor
```

## CI/CD

- **CI:** `.github/workflows/ci.yml` -- `go vet`, `go test -race`, `go build` on push/PR
- **Release:** `.github/workflows/release.yml` -- aarch64 cross-compile + tarball on `v*` tags

## Planning Files

- `requirements.md` - Full requirements specification
- `implementation_plan.md` - Phase-by-phase build guide
- `task_plan.md` - Current progress tracker
- `notes.md` - Architecture notes and decisions
