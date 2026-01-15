# UniFi API v1 Migration Design

**Date:** 2026-02-22
**Status:** Approved

## Goal

Migrate the UniFi collector from cookie-based username/password auth + legacy `/api/s/{site}/` endpoints to the official UniFi Network Integration API v1 using API key authentication.

## API Reference

- Base path: `https://{host}:{port}/proxy/network/integration/v1`
- Auth: `X-API-Key: <key>` header on every request (no login session)
- Docs: https://developer.ui.com/network/v10.1.84/gettingstarted

## Configuration Changes

`UnifiSystemConfig` in `backend/app/config.py`:

- **Remove:** `username: str`, `password: str`
- **Add:** `api_key: str`
- **Keep:** `host`, `port`, `site`, `verify_ssl`

YAML config changes from username/password to `api_key` field.

## Collector Changes (`backend/app/collectors/unifi.py`)

### Removed
- `_csrf_token` and `_cookies` instance variables
- `_login()` method
- `check_connection()` method
- Dual try/except fallback blocks for both auth paths and endpoint paths

### Added
- `_headers` property returning `{"X-API-Key": self.config.api_key}`
- `_base_url` property returning `https://{host}:{port}/proxy/network/integration/v1`

### Endpoints
| Data | Endpoint |
|------|----------|
| Devices | `GET .../sites/{site}/devices` |
| Clients | `GET .../sites/{site}/clients` |
| WAN stats | `GET .../sites/{site}/health` |

### Response Format
New API returns `{"data": [...], "count": int}` — `data` wrapper is preserved.

### Field Mapping
| Old | New | Notes |
|-----|-----|-------|
| `state == 1` | `state == "connected"` | string not int |
| `mac` | `mac` | unchanged |
| `uptime` | `uptime` | unchanged |
| `type == "ugw"/"udm"` | gateway detection via health endpoint | different source |
| `wan1.rx_rate` / `wan1.tx_rate` | from `/health` WAN subsystem | different source |
| `is_guest` | `isGuest` | camelCase |

Exact field names validated at runtime; mismatches logged clearly.

## Models

`UnifiMetrics` and `UnifiDevice` in `backend/app/models/metrics.py` are unchanged. No frontend changes required.

## Error Handling

Single try/except around the full collect block. On any failure, return `create_error_metrics(...)`. Log field mapping mismatches with `.get()` defaults so partial data still renders.
