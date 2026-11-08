# Go Rewrite — Design Spec

Date: 2026-05-24
Status: Approved (in-flight)
Author: Estrella Pan + Claude

## 1. Goals & non-goals

**Goal:** Replace the Rust backend with a Go implementation that is byte-compatible on the wire (frontend unchanged), feature-complete (5 collectors + 3 mocks + WS broadcast + `/api/reload` + Docker/qBittorrent action endpoints), and runs at parity with Rust for the poll-every-5s workload on a Raspberry Pi-class aarch64 device.

**Non-goals:**
- No new features.
- No frontend changes.
- No new collectors.
- No persistence layer.
- No schema changes.

## 2. Stack

| Concern        | Choice                                             |
|----------------|----------------------------------------------------|
| Go version     | 1.23+                                              |
| Router         | `github.com/go-chi/chi/v5`                         |
| WebSocket      | `github.com/gorilla/websocket`                     |
| SSH            | `golang.org/x/crypto/ssh`                          |
| Docker SDK     | `github.com/docker/docker/client` (official)       |
| YAML           | `gopkg.in/yaml.v3`                                 |
| JSON           | `encoding/json` (stdlib)                           |
| Time           | `time` (stdlib)                                    |
| Logging        | `log/slog` (stdlib)                                |
| Randomness     | `math/rand/v2` (stdlib)                            |
| HTTP client    | `net/http` + `net/http/cookiejar` (stdlib)         |

## 3. Repo layout

```
backend/
├── go.mod                          # module github.com/EstrellaXD/ServerMonitor/backend
├── go.sum
├── Dockerfile                      # golang:1.23-alpine → alpine:3 runtime
├── config.yaml, config.example.yaml (unchanged on disk)
├── cmd/server-monitor/main.go      # ← src/main.rs
└── internal/
    ├── api/         routes.go, actions.go, websocket.go
    ├── broadcast/   broadcast.go    # ← tokio::sync::broadcast replacement
    ├── collectors/  collector.go, ssh.go, linux.go, docker.go,
    │                qbittorrent.go, unifi.go, unas.go, mock.go
    ├── config/      config.go, settings.go, systems.go
    ├── manager/     manager.go
    ├── models/      metrics.go, status.go, system_metrics.go, messages.go
    └── store/       metrics_store.go
```

`internal/` enforces module-private boundaries. The binary stays named `server-monitor` (`go build -o server-monitor ./cmd/server-monitor`).

## 4. Concurrency model

| Rust pattern                           | Go replacement                                          |
|----------------------------------------|---------------------------------------------------------|
| `tokio::spawn` poll loop               | `go manager.Run(ctx)`                                   |
| `tokio::sync::RwLock<HashMap>`         | `sync.RWMutex` + `map[string]*SystemMetrics`            |
| `tokio::sync::broadcast::channel(64)`  | custom `Broadcaster`: registry of `chan []byte` subs    |
| `Arc<AppState>`                        | `*AppState` (pointer; GC handles lifetime)              |
| `tokio::sync::Mutex<CollectorManager>` | `sync.Mutex` (held only during reload)                  |
| `tokio::select!` in WS handler         | per-connection goroutine pair + `context.WithCancel`    |
| Graceful shutdown via `ctrl_c()`       | `signal.NotifyContext` + `http.Server.Shutdown(ctx)`    |

The **Broadcaster** type exposes `Subscribe() (<-chan []byte, func())` returning a 64-buffered channel and an unsubscribe closure, plus `Publish([]byte)` that iterates subscribers with non-blocking sends (drops on slow consumers, logs once via `slog`). This matches Rust's "ignore lagged receivers" semantic.

## 5. Wire format (byte-identical contract)

The frontend `frontend/src/types/metrics.ts` is the contract source. Preserved details:

- `SystemStatus` and `SystemType` serialize lowercase (`"healthy"`, `"linux"`, …) — implemented as Go string consts whose underlying values are already lowercase, so default JSON marshaling produces the right bytes.
- `SystemMetrics.type` field uses `json:"type"`.
- `MetricsPayload` (Rust `#[serde(untagged)]`) → Go field `Metrics any` holding the concrete `*LinuxMetrics` / `*DockerMetrics` / etc. The producer is one-way, so untagged-deserialize complexity is irrelevant.
- `error` → `*string` with `json:"error,omitempty"`.
- `last_updated` → `time.Time`; default `MarshalJSON` produces RFC3339Nano which the frontend `new Date(...)` parses identically to `chrono::Utc` output.
- Number fields keep i64/f64 semantics (`int64`/`float64`).
- Optional fields (`temperature`, `eta`, `high`, `critical`) → `*float64` / `*int64` with `omitempty`.

**Verification gate:** golden JSON file checked into `backend/testdata/golden_metrics_update.json` captured from mock mode; `TestWireFormat_Snapshot` asserts a byte-level match. CI fails on schema drift.

## 6. Collector mapping

| Rust crate                       | Go equivalent                                       |
|----------------------------------|-----------------------------------------------------|
| `russh` + `russh-keys`           | `golang.org/x/crypto/ssh` (+ `ssh.ParsePrivateKey`) |
| `bollard`                        | `github.com/docker/docker/client`                   |
| `reqwest` (cookies, TLS)         | `net/http` + `net/http/cookiejar` + `tls.Config`    |
| `serde_yaml`                     | `gopkg.in/yaml.v3`                                  |
| `serde_json`                     | `encoding/json`                                     |
| `chrono`                         | `time`                                              |
| `tracing` + `tracing-subscriber` | `log/slog` (JSON handler)                           |
| `thiserror` / `anyhow`           | `errors.New` / `fmt.Errorf("%w", err)`              |
| `rand`                           | `math/rand/v2`                                      |
| `async_trait`                    | not needed; methods take `ctx context.Context`      |

Parser functions (CPU stat, meminfo, df, zpool list, smartctl) become pure Go funcs taking strings and returning typed structs — they gain unit tests since they are easy to verify without infra.

## 7. SSH shared connection

```go
type SSHConnection struct {
    host, username string
    port           int
    auth           []ssh.AuthMethod
    mu             sync.Mutex
    client         *ssh.Client  // lazy, reconnected on close/error
}

func (c *SSHConnection) Run(ctx context.Context, cmd string) (string, error)
func (c *SSHConnection) Close()
```

Lazy-reconnect pattern mirrors Rust's `ensure_connected`. `HostKeyCallback` accepts any host key — explicit parity choice with current Rust behavior, not a regression.

## 8. HTTP API

chi router; routes match Axum 1:1:

```
GET    /api/health
GET    /api/systems
GET    /api/systems/{system_id}
POST   /api/reload
POST   /api/systems/{system_id}/actions/docker
POST   /api/systems/{system_id}/actions/qbittorrent
GET    /ws
*      static file server (SPA fallback to index.html)
```

CORS is a 15-line middleware; no extra dep. Static files via `http.FileServer` with a 404→`index.html` fallback for SPA routing.

## 9. Build & deploy

**Dockerfile** (`backend/Dockerfile`): multi-stage `golang:1.23-alpine` → `alpine:3`. Static build with `CGO_ENABLED=0`, stripped with `-trimpath -ldflags="-s -w"`. Final image ~15 MB.

**Top-level Dockerfile**: builds frontend (node:20-alpine) + backend (golang:1.23-alpine) → alpine:3.

**Release workflow** (`.github/workflows/release.yml`): aarch64 cross-compile via `GOARCH=arm64 GOOS=linux go build`. No aarch64 runner needed; moves back to `ubuntu-latest`. Same tarball layout: `release/server-monitor` + `release/static/*`.

**CI workflow** (`.github/workflows/ci.yml`): three jobs — `go vet ./...`, `go test -race ./...`, `go build ./...`. Cache `~/go/pkg/mod` and `~/.cache/go-build`.

## 10. Testing

- **Unit** (new, ports of pure logic):
  - `linux_test.go` covers `parseCPUStats`, `parseMeminfo`, `parseDF`, `parseNetStats`, `parseSensors`, `parseThermalZones`, `parseUptime`.
  - `unas_test.go` covers `parseZpoolList`, `parseDiskInfo`, `parseSize`.
  - `manager_test.go` runs the loop against fake collectors and asserts broadcast fires.
- **Integration**: `TestMockMode_FullCycle` starts the server with `SERVERMONITOR_MOCK_MODE=true`, hits `/api/systems` + WS, asserts payload shape.
- **Golden JSON**: `testdata/golden_metrics_update.json` ensures byte-level wire-format stability.

## 11. Migration order

1. `go.mod` + `internal/models/*` + golden JSON test.
2. `internal/config/*` + `internal/store/*` + `internal/broadcast/*`.
3. `internal/collectors/mock.go` → `ssh.go` → `linux.go` → `docker.go` → `qbittorrent.go` → `unifi.go` → `unas.go`.
4. `internal/manager/` + `cmd/server-monitor/main.go` + `internal/api/*`.
5. Dockerfile + CI/release workflow swap.
6. Delete Rust sources (`backend/src/`, `backend/Cargo.*`, `backend/target/`) in one commit.
7. Tag `v2.0.0`, update CHANGELOG.

## 12. Risks

- **Docker SDK API surface**: official `docker/docker/client` is heavier than `bollard`. `Stats()` returns a streaming reader; we wrap it with one-shot semantics by reading one frame and closing.
- **Time-format jitter**: `time.Time.MarshalJSON` precision differs from `chrono` by trailing zeros. Golden test catches this; if needed, custom marshaller emits microsecond precision.
- **WS frame size**: gorilla/websocket has 4 KB default write buffer; bumped to 64 KB to match Rust's permissive defaults.
- **Module path**: `github.com/EstrellaXD/ServerMonitor/backend` derived from the existing GitHub remote. Adjust before tagging if mirrors change.
- **HostKey check**: Deliberately accept-any to match Rust. Tracked as known weakness; not in scope for this rewrite.
