# Notes: ServerMonitor Architecture & Research

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Vue 3 Frontend                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │
│  │ Overview    │  │ Drill-down  │  │ Theme Toggle            │ │
│  │ Dashboard   │  │ Views       │  │ (Dark/Light)            │ │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘ │
└────────────────────────────┬────────────────────────────────────┘
                             │ WebSocket
┌────────────────────────────┴────────────────────────────────────┐
│                       FastAPI Backend                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   Collector Manager                       │  │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────┐ │  │
│  │  │ Linux   │ │ Docker  │ │ qBit    │ │ UniFi   │ │UNAS │ │  │
│  │  │Collector│ │Collector│ │Collector│ │Collector│ │Coll.│ │  │
│  │  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘ └──┬──┘ │  │
│  └───────┼──────────┼──────────┼──────────┼─────────┼──────┘  │
└──────────┼──────────┼──────────┼──────────┼─────────┼──────────┘
           │          │          │          │         │
           ▼          ▼          ▼          ▼         ▼
       Linux VMs   Docker     qBittorrent  UniFi    UNAS
       (SSH)       (API)      (Web API)    (API)    (API)
```

## Collector Integration Methods

### Linux Servers (SSH)
- Library: `asyncssh` or `paramiko`
- Commands: `top`, `free`, `df`, `/proc/stat`, `sensors`
- Metrics: CPU%, RAM, disk, temps via lm-sensors

### Docker
- Library: `docker` Python SDK
- Connect to: Local socket or remote TCP
- Metrics: `docker stats` equivalent via API

### qBittorrent
- API: qBittorrent Web API (built-in)
- Endpoints: `/api/v2/sync/maindata`, `/api/v2/transfer/info`
- Metrics: Active torrents, speeds, queue

### UniFi
- Library: `aiounifi` or direct API calls
- Requires: UniFi Controller credentials
- Metrics: Devices, clients, bandwidth

### UNAS Storage
- Method: SSH or NAS API (depends on NAS type)
- Metrics: Pool status, disk SMART, capacity

## Frontend Component Structure

```
src/
├── components/
│   ├── dashboard/
│   │   ├── OverviewGrid.vue      # Main grid of system cards
│   │   ├── SystemCard.vue        # Individual system status card
│   │   └── StatusIndicator.vue   # Health status dot/badge
│   ├── detail/
│   │   ├── LinuxDetail.vue       # Linux server drill-down
│   │   ├── DockerDetail.vue      # Docker containers detail
│   │   ├── QbitDetail.vue        # qBittorrent detail
│   │   ├── UnifiDetail.vue       # UniFi detail
│   │   └── StorageDetail.vue     # UNAS detail
│   ├── charts/
│   │   ├── CpuChart.vue          # CPU usage gauge/chart
│   │   ├── MemoryChart.vue       # Memory usage
│   │   └── NetworkChart.vue      # Network I/O
│   └── common/
│       ├── ThemeToggle.vue       # Dark/Light switch
│       └── MetricCard.vue        # Reusable metric display
├── composables/
│   ├── useWebSocket.ts           # WebSocket connection
│   ├── useMetrics.ts             # Metrics state management
│   └── useTheme.ts               # Theme state
├── stores/
│   └── metrics.ts                # Pinia store for metrics
└── types/
    └── metrics.ts                # TypeScript interfaces
```

## Backend Structure

```
backend/
├── app/
│   ├── main.py                   # FastAPI app entry
│   ├── config.py                 # Configuration loading
│   ├── models/
│   │   └── metrics.py            # Pydantic models
│   ├── collectors/
│   │   ├── base.py               # Abstract collector class
│   │   ├── linux.py              # Linux SSH collector
│   │   ├── docker.py             # Docker collector
│   │   ├── qbittorrent.py        # qBittorrent collector
│   │   ├── unifi.py              # UniFi collector
│   │   └── unas.py               # UNAS storage collector
│   ├── services/
│   │   ├── collector_manager.py  # Manages all collectors
│   │   └── metrics_store.py      # In-memory metrics storage
│   └── api/
│       ├── routes.py             # REST endpoints
│       └── websocket.py          # WebSocket handler
├── config.yaml                   # System configuration
└── requirements.txt
```

## WebSocket Message Format

```json
{
  "type": "metrics_update",
  "timestamp": "2024-01-15T12:00:00Z",
  "systems": {
    "linux-server-1": {
      "type": "linux",
      "status": "healthy",
      "metrics": {
        "cpu_percent": 45.2,
        "memory_percent": 62.1,
        "disk_percent": 78.5,
        "temperature": 52,
        "uptime_seconds": 1234567
      }
    },
    "docker-host": {
      "type": "docker",
      "status": "healthy",
      "containers": [...]
    }
  }
}
```

## UI Color Scheme (Dark Mode Default)

- Background: `#0f172a` (slate-900)
- Card Background: `#1e293b` (slate-800)
- Text Primary: `#f8fafc` (slate-50)
- Text Secondary: `#94a3b8` (slate-400)
- Status Healthy: `#22c55e` (green-500)
- Status Warning: `#f59e0b` (amber-500)
- Status Critical: `#ef4444` (red-500)
- Accent: `#3b82f6` (blue-500)

## Key Libraries

### Backend (Python)
- `fastapi` - Web framework
- `uvicorn` - ASGI server
- `websockets` - WebSocket support
- `asyncssh` - SSH connections
- `docker` - Docker SDK
- `httpx` - Async HTTP client (for qBit, UniFi APIs)
- `pydantic` - Data validation
- `pyyaml` - Config parsing

### Frontend (Node/Vue)
- `vue@3` - Framework
- `typescript` - Type safety
- `tailwindcss` - Styling
- `pinia` - State management
- `vue-router` - Routing
- `chart.js` + `vue-chartjs` - Charts (optional)
