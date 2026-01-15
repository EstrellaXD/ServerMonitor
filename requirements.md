# ServerMonitor - Requirements Document

## Project Overview

A homelab server monitoring dashboard with modern UI/UX that provides real-time visibility into various home infrastructure components.

## Tech Stack

- **Frontend:** Vue 3 (Composition API) + TypeScript
- **Backend:** FastAPI (Python)
- **Styling:** TailwindCSS
- **Real-time:** WebSocket for live updates
- **Data Storage:** In-memory (no persistence initially)

## Target Systems to Monitor

### 1. Linux Servers (via SSH)
- Connect to Linux machines using SSH
- Collect system metrics remotely

### 2. Docker Containers
- Monitor container status (running, stopped, health)
- Container resource usage (CPU, memory, network)

### 3. qBittorrent
- Active torrent sessions
- Download/upload speeds
- Queue status

### 4. UniFi Router
- Network status
- Connected devices
- Bandwidth usage

### 5. UNAS Storage
- Storage pool status
- Disk health
- Available capacity

## Metrics to Collect

### System Basics
- CPU usage (per core and total)
- RAM usage (used, available, cached)
- Disk usage (per mount point)
- Network I/O (upload/download rates)
- System uptime

### Hardware Sensors
- CPU/GPU temperatures
- Fan speeds
- Power consumption (if available)

### Application-Specific
- qBittorrent: active downloads, seeds, speeds, queue length

## UI/UX Requirements

### Dashboard Layout
- **Overview mode:** Grid/list of all monitored systems showing status at a glance
- **Drill-down mode:** Click on any system to see detailed metrics

### Theme
- Dark mode and light mode with user toggle
- Default to dark mode (suitable for always-on displays)

### Design Style
- Modern, clean interface
- Card-based layout for system overview
- Clear status indicators (healthy/warning/critical)
- Responsive design for various screen sizes

## Technical Requirements

### Data Collection
- Poll interval: 5-10 seconds for real-time feel
- WebSocket push for instant UI updates
- Graceful handling of unreachable systems

### Deployment
- Local network access only
- No authentication required (initial version)
- Easy to run via Docker Compose or directly

### API Design
- RESTful endpoints for configuration
- WebSocket endpoint for real-time metrics stream

## Future Considerations (Out of Scope for v1)

- Alert notifications (email, Discord, etc.)
- Historical data persistence
- User authentication
- Mobile app

## Success Criteria

1. Dashboard loads and displays all configured systems
2. Metrics update in real-time (5-10 second refresh)
3. Drill-down provides detailed view per system
4. UI is responsive and visually appealing
5. System handles offline/unreachable targets gracefully
