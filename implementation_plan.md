# ServerMonitor Implementation Plan

## Phase 2: Project Scaffolding

### Backend Setup
```bash
mkdir -p backend/app/{collectors,services,api,models}
cd backend
python -m venv venv
pip install fastapi uvicorn websockets asyncssh docker httpx pydantic pyyaml
```

### Frontend Setup
```bash
npm create vue@latest frontend -- --template vue-ts
cd frontend
npm install -D tailwindcss postcss autoprefixer
npm install pinia vue-router
npx tailwindcss init -p
```

### Directory Structure
```
ServerMonitor/
├── backend/
│   ├── app/
│   │   ├── __init__.py
│   │   ├── main.py
│   │   ├── config.py
│   │   ├── models/
│   │   ├── collectors/
│   │   ├── services/
│   │   └── api/
│   ├── config.yaml
│   └── requirements.txt
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── composables/
│   │   ├── stores/
│   │   ├── types/
│   │   └── views/
│   └── package.json
├── docker-compose.yml
├── requirements.md
├── task_plan.md
└── notes.md
```

---

## Phase 3: Backend Core

### 3.1 Configuration System (`config.py`)
- Load `config.yaml` with system definitions
- Pydantic models for config validation
- Support for credentials and connection details

### 3.2 Metrics Models (`models/metrics.py`)
```python
class SystemStatus(str, Enum):
    HEALTHY = "healthy"
    WARNING = "warning"
    CRITICAL = "critical"
    OFFLINE = "offline"

class SystemMetrics(BaseModel):
    id: str
    name: str
    type: str  # linux, docker, qbittorrent, unifi, unas
    status: SystemStatus
    last_updated: datetime
    metrics: dict[str, Any]
```

### 3.3 FastAPI Entry (`main.py`)
- CORS middleware for local dev
- Include REST and WebSocket routers
- Startup event to initialize collectors

---

## Phase 4: Collectors Implementation

### 4.1 Base Collector (`collectors/base.py`)
```python
class BaseCollector(ABC):
    @abstractmethod
    async def collect(self) -> SystemMetrics:
        pass

    @abstractmethod
    async def check_health(self) -> SystemStatus:
        pass
```

### 4.2 Linux Collector (`collectors/linux.py`)
- SSH connection using `asyncssh`
- Parse `/proc/stat` for CPU
- Parse `/proc/meminfo` for RAM
- Run `df -h` for disk usage
- Run `sensors` for temperatures
- Calculate uptime from `/proc/uptime`

### 4.3 Docker Collector (`collectors/docker.py`)
- Connect to Docker socket/API
- List containers with status
- Get resource usage per container
- Aggregate host-level stats

### 4.4 qBittorrent Collector (`collectors/qbittorrent.py`)
- Authenticate to Web API
- Fetch `/api/v2/sync/maindata`
- Extract: active torrents, DL/UL speeds, queue

### 4.5 UniFi Collector (`collectors/unifi.py`)
- Connect to UniFi Controller API
- Fetch device list and status
- Get client count and bandwidth

### 4.6 UNAS Collector (`collectors/unas.py`)
- SSH or API based on NAS type
- Fetch pool/volume status
- Get disk health and capacity

---

## Phase 5: WebSocket Streaming

### 5.1 Collector Manager (`services/collector_manager.py`)
- Initialize all configured collectors
- Run collection loop (every 5-10 sec)
- Broadcast updates to WebSocket clients

### 5.2 WebSocket Handler (`api/websocket.py`)
- Accept connections at `/ws`
- Send initial state on connect
- Push updates on each collection cycle
- Handle client disconnects gracefully

### 5.3 REST Endpoints (`api/routes.py`)
- `GET /api/systems` - List all systems
- `GET /api/systems/{id}` - Get single system details
- `GET /api/config` - Get current config (sanitized)

---

## Phase 6: Frontend Core

### 6.1 Tailwind Configuration
- Dark mode via `class` strategy
- Custom colors matching design spec
- Typography and spacing setup

### 6.2 Pinia Store (`stores/metrics.ts`)
```typescript
interface MetricsState {
  systems: Map<string, SystemMetrics>
  connected: boolean
  lastUpdate: Date | null
}
```

### 6.3 WebSocket Composable (`composables/useWebSocket.ts`)
- Connect to backend WebSocket
- Auto-reconnect on disconnect
- Update Pinia store on messages

### 6.4 Theme Composable (`composables/useTheme.ts`)
- Toggle dark/light mode
- Persist preference in localStorage
- Apply class to document root

---

## Phase 7: Dashboard UI

### 7.1 Overview Grid (`components/dashboard/OverviewGrid.vue`)
- Responsive grid layout (1-4 columns)
- Map over systems from store
- Render SystemCard for each

### 7.2 System Card (`components/dashboard/SystemCard.vue`)
- Show: name, type icon, status indicator
- Key metrics preview (CPU, RAM)
- Click to navigate to detail view

### 7.3 Status Indicator (`components/dashboard/StatusIndicator.vue`)
- Color-coded dot/badge
- Pulse animation for warnings/critical

---

## Phase 8: Drill-Down Views

### 8.1 Router Setup
```typescript
routes: [
  { path: '/', component: Dashboard },
  { path: '/system/:id', component: SystemDetail }
]
```

### 8.2 Linux Detail View
- CPU usage (per-core bars)
- Memory breakdown (used/cached/free)
- Disk usage per mount
- Temperature readings
- Uptime display

### 8.3 Docker Detail View
- Container list with status
- Per-container CPU/memory
- Image and port info

### 8.4 qBittorrent Detail View
- Active torrent list
- Global speeds
- Queue statistics

### 8.5 UniFi Detail View
- Device status list
- Client count
- Bandwidth graphs

### 8.6 Storage Detail View
- Pool/volume status
- Disk health indicators
- Capacity bars

---

## Phase 9: Theme Toggle

### 9.1 ThemeToggle Component
- Sun/Moon icon button
- Smooth transition animation
- Sync with composable

### 9.2 CSS Variables
- Define colors as CSS vars
- Switch values based on `.dark` class

---

## Phase 10: Integration & Polish

### 10.1 Error Handling
- Offline system display (greyed out card)
- Connection lost indicator
- Retry logic for collectors

### 10.2 Loading States
- Skeleton loaders on initial load
- Spinner during reconnection

### 10.3 Responsive Testing
- Mobile breakpoints
- Tablet layout
- Large screen optimization

---

## Phase 11: Deployment

### 11.1 Docker Compose
```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8000:8000"
    volumes:
      - ./config.yaml:/app/config.yaml

  frontend:
    build: ./frontend
    ports:
      - "3000:80"
    depends_on:
      - backend
```

### 11.2 Configuration File (`config.yaml`)
```yaml
poll_interval: 5

systems:
  - id: linux-server-1
    type: linux
    name: "Main Server"
    host: 192.168.1.100
    ssh_user: monitor
    ssh_key_path: /path/to/key

  - id: docker-host
    type: docker
    name: "Docker Host"
    socket: unix:///var/run/docker.sock

  - id: qbit
    type: qbittorrent
    name: "qBittorrent"
    url: http://192.168.1.100:8080
    username: admin
    password: secret
```

---

## Milestone Checklist

| Milestone | Description | Deliverable |
|-----------|-------------|-------------|
| M1 | Backend runs, single Linux collector works | `/api/systems` returns data |
| M2 | WebSocket streams metrics to browser | Console logs show updates |
| M3 | Dashboard shows system cards | Visual grid renders |
| M4 | All collectors implemented | 5 system types working |
| M5 | Drill-down views complete | Click-through works |
| M6 | Theme toggle works | Dark/Light switch |
| M7 | Docker Compose runs full stack | One-command deployment |

---

## Next Action

**Ready to begin Phase 2: Project Scaffolding**

Run the following to start:
```bash
# Create backend structure
mkdir -p backend/app/{collectors,services,api,models}
touch backend/app/__init__.py
touch backend/requirements.txt

# Create frontend with Vue
npm create vue@latest frontend
```
