# ServerMonitor Performance Analysis Report

Generated: 2026-01-25

## Executive Summary

Performance profiling of the ServerMonitor backend reveals **three critical bottlenecks** that can cause collection cycles to take 20-30 seconds. With the recommended optimizations, this can be reduced to 2-5 seconds (4-15x improvement).

### Key Findings

| Component | Current Performance | Bottleneck | Potential Improvement |
|-----------|---------------------|------------|----------------------|
| Docker Collector | ~1-2s per container | Sequential `container.stats()` | 10x (parallelize) |
| Linux Collector | ~300ms per system | Sequential SSH commands | 3-5x (combine commands) |
| SSH Connections | ~100-500ms per cycle | New connection each time | Eliminate overhead |
| UNAS Collector | ~0.5-2s per disk | Sequential smartctl | 5-10x (parallelize) |

## Profiling Results

### Parsing Functions (CPU-bound)

All parsing functions are highly efficient:

| Function | Avg Time | Notes |
|----------|----------|-------|
| `_parse_cpu_stats()` | 0.0044 ms | Excellent |
| `_parse_meminfo()` | 0.0027 ms | Excellent |
| `_parse_df()` | 0.0028 ms | Excellent |
| `_parse_net_stats()` | 0.0011 ms | Excellent |
| `_parse_sensors()` | 0.0038 ms | Excellent |
| `_parse_zpool_list()` | 0.0055 ms | Excellent |

### Metrics Store Operations

| Operation | Avg Time | Notes |
|-----------|----------|-------|
| Single Update | 0.0005 ms | Excellent |
| Get Operation | 0.0001 ms | Excellent |
| Get All (100 systems) | 0.0003 ms | Excellent |
| Create Update Message | 0.0052 ms | Excellent |

### Model Serialization (Pydantic)

| Operation | Avg Time | Notes |
|-----------|----------|-------|
| Linux Metrics → dict | 0.0034 ms | Good |
| Linux Metrics → JSON | 0.0049 ms | Good |
| Docker (50 containers) → dict | 0.0216 ms | Acceptable |
| Docker (50 containers) → JSON | 0.0425 ms | Acceptable |

### Memory Usage

| Scenario | Memory | Notes |
|----------|--------|-------|
| Single SystemMetrics | 9.28 KB | Reasonable |
| 100 systems store | 17.87 KB | Efficient |
| 60 systems + 500 containers | 8.71 MB | Acceptable |
| WebSocket payload (60 sys) | 139.61 KB | Consider compression |

## Priority 1: Critical Bottlenecks

### 1. Docker Collector - Sequential Stats Collection

**Location:** `backend/app/collectors/docker.py:107-155`

**Problem:** The `container.stats(stream=False)` call is blocking and takes 1-2 seconds per container. With 10 running containers, collection takes 10-20 seconds.

**Current Code:**
```python
for container in containers:
    if state == "running":
        stats = container.stats(stream=False)  # BLOCKING!
```

**Recommended Fix:**
```python
import asyncio
from concurrent.futures import ThreadPoolExecutor

class DockerCollector(BaseCollector):
    def __init__(self, ...):
        ...
        self._executor = ThreadPoolExecutor(max_workers=10)

    async def _get_container_stats(self, container):
        if container.status != "running":
            return container, None
        loop = asyncio.get_event_loop()
        stats = await loop.run_in_executor(
            self._executor,
            lambda: container.stats(stream=False)
        )
        return container, stats

    async def collect(self) -> SystemMetrics:
        client = self._get_client()
        containers = client.containers.list(all=True)

        # Collect stats in parallel
        results = await asyncio.gather(*[
            self._get_container_stats(c) for c in containers
        ])
        # Process results...
```

**Impact:** 10x improvement for 10+ containers

---

### 2. Linux Collector - Sequential SSH Commands

**Location:** `backend/app/collectors/linux.py:235-269`

**Problem:** 7+ SSH commands run sequentially, each with 10-50ms latency overhead.

**Current Code:**
```python
async with await self._get_connection() as conn:
    stat = await self._run_command(conn, "cat /proc/stat")
    meminfo = await self._run_command(conn, "cat /proc/meminfo")
    df = await self._run_command(conn, "df -B1")
    # ... 4+ more commands
```

**Recommended Fix - Option A (Single Combined Command):**
```python
combined_cmd = '''
echo "===STAT===" && cat /proc/stat && \
echo "===MEMINFO===" && cat /proc/meminfo && \
echo "===DF===" && df -B1 && \
echo "===NETDEV===" && cat /proc/net/dev && \
echo "===UPTIME===" && cat /proc/uptime && \
echo "===LOADAVG===" && cat /proc/loadavg
'''

async with await self._get_connection() as conn:
    output = await self._run_command(conn, combined_cmd)

# Parse sections
sections = {}
for section in output.split("==="):
    if section.strip():
        lines = section.strip().split("\n", 1)
        if len(lines) == 2:
            sections[lines[0]] = lines[1]
```

**Recommended Fix - Option B (Parallel Commands):**
```python
async with await self._get_connection() as conn:
    stat, meminfo, df, netdev, uptime, loadavg = await asyncio.gather(
        self._run_command(conn, "cat /proc/stat"),
        self._run_command(conn, "cat /proc/meminfo"),
        self._run_command(conn, "df -B1"),
        self._run_command(conn, "cat /proc/net/dev"),
        self._run_command(conn, "cat /proc/uptime"),
        self._run_command(conn, "cat /proc/loadavg"),
    )
```

**Impact:** 3-5x improvement per Linux system

---

### 3. SSH Connection Pooling

**Location:** `backend/app/collectors/linux.py:28-44`, `backend/app/collectors/unas.py:22-36`

**Problem:** New SSH connection created for every collection cycle (~100-500ms overhead).

**Recommended Fix:**
```python
class LinuxCollector(BaseCollector):
    def __init__(self, ...):
        ...
        self._connection: asyncssh.SSHClientConnection | None = None
        self._connection_lock = asyncio.Lock()

    async def _get_or_create_connection(self) -> asyncssh.SSHClientConnection:
        async with self._connection_lock:
            if self._connection is None or self._connection.is_closed():
                self._connection = await asyncssh.connect(
                    host=self.config.host,
                    port=self.config.port,
                    username=self.config.username,
                    known_hosts=None,
                    connect_timeout=10,
                    client_keys=[self.config.key_path] if self.config.key_path else None,
                    password=self.config.password if not self.config.key_path else None,
                )
            return self._connection

    async def close(self):
        if self._connection:
            self._connection.close()
            await self._connection.wait_closed()
            self._connection = None
```

**Impact:** 100-500ms saved per collection cycle per system

---

## Priority 2: Medium Impact

### 4. UNAS Collector - Sequential smartctl

**Location:** `backend/app/collectors/unas.py:184-189`

**Fix:** Parallelize smartctl calls using `asyncio.gather()` or combine into single command.

### 5. HTTP Client Pooling

**Location:** `backend/app/collectors/qbittorrent.py:51`, `backend/app/collectors/unifi.py:83`

**Fix:** Create persistent `httpx.AsyncClient` instances.

### 6. Use orjson for JSON

**Fix:** Replace Pydantic's JSON serialization with orjson for 3-10x speedup.

```bash
pip install orjson
```

```python
import orjson

# Instead of model_dump_json()
json_bytes = orjson.dumps(metrics.model_dump(mode='json'))
```

---

## Priority 3: Low Impact

- Cache unchanged metrics for WebSocket broadcasts
- Add `__slots__` to Pydantic models via `model_config`
- Enable WebSocket compression (permessage-deflate)
- String parsing micro-optimizations (use `str.partition()` instead of `str.split()`)

---

## Implementation Checklist

### High Priority
- [ ] Parallelize Docker container.stats() with ThreadPoolExecutor
- [ ] Combine Linux SSH commands into single command
- [ ] Implement SSH connection pooling for Linux/UNAS collectors

### Medium Priority
- [ ] Parallelize UNAS smartctl commands
- [ ] Add HTTP client pooling for QBittorrent/UniFi
- [ ] Add orjson dependency and use for serialization

### Low Priority
- [ ] Cache serialized metrics
- [ ] Enable WebSocket compression
- [ ] Add `__slots__` to Pydantic models

---

## Benchmarking Commands

Run the profiling scripts:

```bash
cd backend

# Full profiling
uv run python performance_profile.py

# Bottleneck analysis
uv run python analyze_bottlenecks.py
```

---

## Conclusion

The ServerMonitor backend has efficient parsing and serialization code. The main performance bottlenecks are I/O-related:

1. **Blocking Docker API calls** - easily fixed with thread pool
2. **Sequential SSH commands** - combine or parallelize
3. **Connection overhead** - implement pooling

Implementing the Priority 1 fixes will reduce collection cycle time from 20-30 seconds to 2-5 seconds, making the dashboard much more responsive.
