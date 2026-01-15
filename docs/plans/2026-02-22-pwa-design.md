# PWA Support Design

**Date:** 2026-02-22
**Status:** Approved

## Goal

Add Progressive Web App support to ServerMonitor so users can install it to their home screen/desktop and have the app shell load from cache when the backend is unreachable.

## Approach

Use `vite-plugin-pwa` (Option A) — the standard Vite + Workbox integration. No manual service worker file. All configuration lives in `vite.config.ts`.

## Architecture

`vite-plugin-pwa` generates a Workbox-powered service worker at build time and injects the web app manifest into `index.html`. Service worker registration is automatic.

Changes required:
- Install `vite-plugin-pwa` as a dev dependency
- Configure plugin in `vite.config.ts` with manifest and caching strategies
- Generate PWA icons (192×192 and 512×512 PNG) from existing `favicon.svg`
- Place icons in `frontend/public/`

## Caching Strategy

| Resource | Strategy | Rationale |
|----------|----------|-----------|
| App shell (JS, CSS, fonts, icons) | Cache-first | Instant load; updated in background on next visit |
| API routes (`/api/*`) | Network-first with cache fallback | Tries live backend; serves cached response offline |
| WebSocket (`/ws`) | Not cached | Live connection only; existing composable handles disconnect gracefully |

## Offline Behavior

When the backend is unreachable:
- The app shell loads from cache
- The Pinia store starts empty (no stale metrics persistence — Option A)
- The dashboard shows systems as offline via existing health-check logic
- A banner or status indicator from the existing UI communicates the disconnected state

## Web App Manifest

```json
{
  "name": "ServerMonitor",
  "short_name": "ServerMonitor",
  "display": "standalone",
  "theme_color": "#0f172a",
  "background_color": "#0f172a",
  "icons": [
    { "src": "/icons/pwa-192x192.png", "sizes": "192x192", "type": "image/png" },
    { "src": "/icons/pwa-512x512.png", "sizes": "512x512", "type": "image/png" }
  ]
}
```

## Out of Scope

- Stale metrics persistence (localStorage caching of last WebSocket snapshot)
- Push notifications
