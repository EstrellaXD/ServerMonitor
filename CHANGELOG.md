# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

### Breaking Changes
- Control actions use one route, `POST /api/systems/{id}/actions` (`{action, target, params}`). The routes `/actions/docker` and `/actions/qbittorrent` are removed.
- Each system needs a `controls:` list in `config.yaml`. A system without it allows no control actions, including Docker start/stop.
- Requests whose `Host` is not an IP, `localhost`, a `*.local` name or an `allowed_hosts` entry get 403. `/ws` accepts no Origin or an allowed Origin only.
- An unparsable `config.yaml` is now an error: startup exits, and `/api/reload` keeps the running config.

### Added
- Frontend rewritten in a terminal "command block" style: side list of systems, collapsible blocks, filter prompt, keyboard keys (`1`–`9`, `esc`, `/`, `t`), dark and light terminal palettes.
- Confirm steps by risk: low runs at once, mid asks `[y/N]`, high asks for the system id.
- Docker: pause, unpause, kill, pull + recreate (keeps config, networks and anonymous volumes; rolls back on failure), and a logs view (`GET /api/systems/{id}/docker/logs`).
- qBittorrent: recheck, priority (top/up/down/bottom), delete with files.
- Linux and UNAS: reboot, poweroff. Linux: kill process (TERM/KILL, checks that the PID still has the same name), systemd service start/stop/restart for units listed in `services:`, fan speed (0–100 %) and back to automatic control (Raspberry Pi 5 and Jetson `pwmfan`).
- UniFi: restart device, kick, block and unblock client; client and blocked-client lists.

### Fixed
- qBittorrent 5.1+ login (HTTP 204 with no body) was reported as an authentication failure.
- qBittorrent 5.0+ resume/pause use `/torrents/start` and `/torrents/stop`, with the old names as fallback.
- UniFi client list read only the first page (25 clients).

### Changed
- Backend rewritten from Rust (Axum + Tokio) to Go 1.23 (chi + goroutines). Wire format unchanged: existing frontend, API contract, config file, and Docker invocation are byte-identical. Build time drops from ~2 min to ~20 s; binary size ~15 MB.

### Added
- Docker container controls: start, stop, restart via kebab menu on each container card
- qBittorrent torrent controls: resume, pause, delete via kebab menu on each torrent card
- Glassmorphic confirmation modals for destructive actions (stop, restart, delete)
- Delete torrent modal includes "also delete downloaded files" checkbox
- Context-aware menus: only shows applicable actions based on container/torrent state
- Backend POST endpoints for Docker actions (`/api/systems/{id}/actions/docker`)
- Backend POST endpoints for qBittorrent actions (`/api/systems/{id}/actions/qbittorrent`)
- `KebabMenu.vue` and `ConfirmModal.vue` reusable components
- `useActions.ts` composable for action API calls with per-item loading states
- `hash` field on `TorrentInfo` for torrent identification
