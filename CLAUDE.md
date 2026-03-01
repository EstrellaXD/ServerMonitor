# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ServerMonitor is a homelab server monitoring dashboard with real-time metrics visualization.

**Tech Stack:**
- Frontend: Vue 3 (Composition API) + TypeScript + TailwindCSS
- Backend: Rust (Axum + Tokio) with WebSocket support
- Data: In-memory storage (no persistence)

**Monitored Systems:**
- Linux servers via SSH (russh)
- Docker containers (bollard)
- qBittorrent (Web API via reqwest)
- UniFi router (API via reqwest)
- UNAS storage (SSH via russh)

## Build and Development Commands

### Backend
```bash
cd backend
cargo build --release           # Build
cargo run --release              # Run
cargo test                       # Tests
cargo clippy -- -D warnings      # Lint

# Mock mode (no real infrastructure needed)
SERVERMONITOR_MOCK_MODE=true cargo run --release
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
backend/src/
├── main.rs              # Tokio runtime, Axum router, graceful shutdown
├── lib.rs               # Module re-exports
├── error.rs             # CollectorError, AppError with IntoResponse
├── config/
│   ├── settings.rs      # Environment variable settings
│   ├── app_config.rs    # YAML config loading
│   └── system_configs.rs # Per-type config structs
├── models/
│   ├── status.rs        # SystemStatus, SystemType enums
│   ├── metrics.rs       # All metric structs (Linux, Docker, etc.)
│   ├── system_metrics.rs # SystemMetrics + MetricsPayload union
│   └── messages.rs      # MetricsUpdate WebSocket message
├── collectors/
│   ├── mod.rs           # Collector trait + factory function
│   ├── ssh_connection.rs # Shared SSH abstraction (Linux + UNAS)
│   ├── linux.rs         # SSH-based Linux metrics
│   ├── docker.rs        # Docker API via bollard
│   ├── qbittorrent.rs   # qBit Web API via reqwest
│   ├── unifi.rs         # UniFi API via reqwest
│   ├── unas.rs          # NAS storage via SSH
│   └── mock.rs          # Mock collectors for testing
├── services/
│   ├── metrics_store.rs # RwLock<HashMap> in-memory cache
│   └── collector_manager.rs # Collection loop orchestration
└── api/
    ├── routes.rs        # REST handlers + AppState
    └── websocket.rs     # WebSocket broadcast via tokio::sync::broadcast

frontend/src/
├── components/
│   ├── dashboard/       # Overview grid and system cards
│   └── detail/          # Per-system drill-down views
├── composables/         # useWebSocket, useTheme, useMetrics
├── stores/metrics.ts    # Pinia store for system state
└── types/metrics.ts     # TypeScript interfaces
```

## Key Patterns

- **Collectors:** Each system type implements the `Collector` trait with `collect()` and `close()` methods
- **SSH Sharing:** `SshConnection` abstracts SSH for both Linux and UNAS collectors
- **State:** `Arc<AppState>` passed through Axum `State` extractor (replaces Python module-level singletons)
- **Real-time:** `tokio::sync::broadcast` for WebSocket fan-out (serialize once, not per-connection)
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

- **CI:** `.github/workflows/ci.yml` -- cargo check, clippy, test, build on push/PR
- **Docker Publish:** `.github/workflows/docker-publish.yml` -- multi-arch image to ghcr.io on version tags

## Planning Files

- `requirements.md` - Full requirements specification
- `implementation_plan.md` - Phase-by-phase build guide
- `task_plan.md` - Current progress tracker
- `notes.md` - Architecture notes and decisions
