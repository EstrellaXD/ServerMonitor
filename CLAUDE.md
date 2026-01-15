# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Developer Preferences

- Use `uv` instead of direct `python` commands (e.g., `uv run python` instead of `python`)

## Project Overview

ServerMonitor is a homelab server monitoring dashboard with real-time metrics visualization.

**Tech Stack:**
- Frontend: Vue 3 (Composition API) + TypeScript + TailwindCSS
- Backend: FastAPI (Python) with WebSocket support
- Data: In-memory storage (no persistence)

**Monitored Systems:**
- Linux servers via SSH
- Docker containers
- qBittorrent (Web API)
- UniFi router
- UNAS storage

## Build and Development Commands

### Backend
```bash
cd backend
python -m venv venv
source venv/bin/activate  # or `venv\Scripts\activate` on Windows
pip install -r requirements.txt
uvicorn app.main:app --reload --host 0.0.0.0 --port 8742
```

### Frontend
```bash
cd frontend
npm install
npm run dev          # Development server
npm run build        # Production build
npm run preview      # Preview production build
```

### Full Stack (Docker)
```bash
docker-compose up --build
```

## Architecture

```
backend/app/
├── main.py              # FastAPI entry, CORS, routers
├── config.py            # YAML config loader
├── collectors/          # System-specific metric collectors
│   ├── base.py          # Abstract BaseCollector
│   ├── linux.py         # SSH-based Linux metrics
│   ├── docker.py        # Docker API collector
│   ├── qbittorrent.py   # qBit Web API
│   ├── unifi.py         # UniFi Controller API
│   └── unas.py          # NAS storage collector
├── services/
│   ├── collector_manager.py  # Orchestrates all collectors
│   └── metrics_store.py      # In-memory metrics cache
└── api/
    ├── routes.py        # REST endpoints
    └── websocket.py     # Real-time streaming

frontend/src/
├── components/
│   ├── dashboard/       # Overview grid and system cards
│   └── detail/          # Per-system drill-down views
├── composables/         # useWebSocket, useTheme, useMetrics
├── stores/metrics.ts    # Pinia store for system state
└── types/metrics.ts     # TypeScript interfaces
```

## Key Patterns

- **Collectors:** Each system type extends `BaseCollector` with `collect()` and `check_health()` methods
- **Real-time:** Backend pushes metrics via WebSocket every 5-10 seconds
- **Status:** Systems are `healthy`, `warning`, `critical`, or `offline`
- **Theming:** Dark/Light mode via TailwindCSS `class` strategy

## Configuration

Systems are defined in `config.yaml`:
```yaml
poll_interval: 5
systems:
  - id: linux-server-1
    type: linux
    host: 192.168.1.100
    ssh_user: monitor
```

## Planning Files

- `requirements.md` - Full requirements specification
- `implementation_plan.md` - Phase-by-phase build guide
- `task_plan.md` - Current progress tracker
- `notes.md` - Architecture notes and decisions
