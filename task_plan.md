# Task Plan: ServerMonitor Implementation

## Goal
Build a real-time homelab server monitoring dashboard with Vue 3 frontend and FastAPI backend that monitors Linux servers, Docker, qBittorrent, UniFi, and UNAS storage.

## Phases
- [x] Phase 1: Requirements gathering and planning
- [x] Phase 2: Project scaffolding and structure setup
- [x] Phase 3: Backend core - FastAPI setup and collector framework
- [x] Phase 4: Implement individual collectors (Linux, Docker, qBittorrent, UniFi, UNAS)
- [x] Phase 5: WebSocket real-time streaming
- [x] Phase 6: Frontend core - Vue project with TailwindCSS
- [x] Phase 7: Dashboard UI - Overview grid with status cards
- [x] Phase 8: Drill-down views for each system type
- [x] Phase 9: Dark/Light theme toggle
- [x] Phase 10: Docker Compose deployment setup
- [ ] Phase 11: Testing and configuration

## Key Questions
1. ✅ What systems to monitor? → Linux SSH, Docker, qBittorrent, UniFi, UNAS
2. ✅ What metrics? → CPU, RAM, disk, network, temps, app-specific
3. ✅ UI style? → Dark/Light toggle, card-based overview + drill-down
4. ✅ Data storage? → In-memory only (no persistence for v1)
5. ✅ Poll interval? → 5-10 seconds

## Decisions Made
- **Tech Stack:** Vue 3 + TypeScript + TailwindCSS | FastAPI + Python
- **Real-time:** WebSocket for push updates
- **No Auth:** Local network only, no login required for v1
- **Modular Collectors:** Each system type has its own collector class

## Errors Encountered
- (none)

## Status
**Phase 10 Complete** - Full stack implementation done!

### Completed
- Backend: FastAPI with all 5 collectors (Linux, Docker, qBittorrent, UniFi, UNAS)
- Frontend: Vue 3 + TypeScript + TailwindCSS with dark/light mode
- WebSocket real-time updates
- Dashboard with status overview and drill-down views
- Docker Compose deployment ready

### Files Structure
```
ServerMonitor/
├── backend/
│   ├── app/
│   │   ├── main.py
│   │   ├── config.py
│   │   ├── models/metrics.py
│   │   ├── collectors/{base,linux,docker,qbittorrent,unifi,unas}.py
│   │   ├── services/{collector_manager,metrics_store}.py
│   │   └── api/{routes,websocket}.py
│   ├── config.yaml
│   ├── requirements.txt
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/{dashboard,detail,common,charts}/
│   │   ├── composables/{useTheme,useWebSocket}.ts
│   │   ├── stores/metrics.ts
│   │   ├── types/metrics.ts
│   │   └── views/{Dashboard,SystemDetail}.vue
│   ├── package.json
│   ├── Dockerfile
│   └── nginx.conf
└── docker-compose.yml
```

**Next:** Configure systems in config.yaml and test!
