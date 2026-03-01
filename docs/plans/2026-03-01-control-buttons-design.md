# Control Buttons Design

## Scope

Add interactive controls to two detail views:
- **DockerDetail.vue** - Start / Stop / Restart containers
- **QbitDetail.vue** - Resume / Pause / Delete torrents

## Interaction Pattern

- **Kebab menu** (three-dot) per row opens a glassmorphic dropdown
- **Modal confirmation** for destructive actions (Stop, Restart, Delete)
- **No confirmation** for safe actions (Start, Resume, Pause)

## Components

### KebabMenu.vue (components/common/)
- Reusable dropdown trigger with glassmorphic dropdown
- Props: `items: Array<{ label, icon, action, variant }>`
- Click-outside and Escape to close

### ConfirmModal.vue (components/common/)
- Centered overlay with glassmorphic card
- Icon, title, message, Cancel/Confirm buttons
- Delete torrent variant: "Also delete files" checkbox
- Props: `title, message, confirmLabel, variant, show`

### useActions.ts (composables/)
- API calls to backend action endpoints
- Per-item loading states via Map
- Error handling with auto-dismiss inline indicators

## State-Aware Actions

### Docker
| State | Actions |
|---|---|
| running/paused | Restart, Stop |
| exited/dead/created | Start |

### qBittorrent
| State | Actions |
|---|---|
| downloading/uploading/stalled | Pause |
| paused | Resume |
| any | Delete |

## Backend Endpoints

```
POST /api/systems/{system_id}/actions/docker
  { container_name, action: "start"|"stop"|"restart" }

POST /api/systems/{system_id}/actions/qbittorrent
  { hash, action: "resume"|"pause"|"delete", delete_files?: bool }
```

## Visual Style

- Glassmorphic: backdrop-blur-md, semi-transparent bg, border, rounded-xl
- Semantic hover colors: green (start/resume), amber (restart), red (stop/delete)
- Loading: spinner replaces kebab icon during action
- Error: red inline text, auto-dismiss 3s
