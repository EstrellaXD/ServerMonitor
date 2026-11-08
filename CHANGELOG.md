# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

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
