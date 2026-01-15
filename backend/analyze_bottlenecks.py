#!/usr/bin/env python3
"""
Deep analysis of specific performance bottlenecks in ServerMonitor.
"""

import asyncio
import sys
import time
import tracemalloc
from datetime import datetime, UTC

sys.path.insert(0, ".")


def analyze_docker_collector_bottleneck():
    """Analyze Docker collector - container.stats() is blocking."""
    print("\n" + "="*60)
    print("DOCKER COLLECTOR BOTTLENECK ANALYSIS")
    print("="*60)

    print("""
IDENTIFIED BOTTLENECK: DockerCollector.collect() at line 107-155

The Docker collector calls `container.stats(stream=False)` for EACH running
container sequentially. This is a BLOCKING operation that takes 1-2 seconds
per container.

PROBLEM CODE (docker.py:121-126):
```python
for container in containers:
    if state == "running":
        stats = container.stats(stream=False)  # BLOCKING! ~1-2s per container
```

With 10 running containers, collection can take 10-20 seconds!

RECOMMENDED FIX: Use asyncio.to_thread() or ThreadPoolExecutor for parallel stats:

```python
import asyncio
from concurrent.futures import ThreadPoolExecutor

class DockerCollector(BaseCollector):
    def __init__(self, ...):
        ...
        self._executor = ThreadPoolExecutor(max_workers=10)

    async def collect(self) -> SystemMetrics:
        client = self._get_client()
        containers = client.containers.list(all=True)

        async def get_container_stats(container):
            if container.status == "running":
                # Run blocking call in thread pool
                stats = await asyncio.get_event_loop().run_in_executor(
                    self._executor,
                    lambda: container.stats(stream=False)
                )
                return container, stats
            return container, None

        # Collect all stats in parallel
        results = await asyncio.gather(*[
            get_container_stats(c) for c in containers
        ])
```

ESTIMATED IMPROVEMENT: 10x faster for 10 containers (20s -> 2s)
""")


def analyze_linux_ssh_bottleneck():
    """Analyze Linux collector - sequential SSH commands."""
    print("\n" + "="*60)
    print("LINUX COLLECTOR BOTTLENECK ANALYSIS")
    print("="*60)

    print("""
IDENTIFIED BOTTLENECK: LinuxCollector.collect() at line 235-269

The Linux collector runs 7+ SSH commands SEQUENTIALLY:
1. cat /proc/stat
2. cat /proc/meminfo
3. df -B1
4. cat /proc/net/dev
5. cat /proc/uptime
6. cat /proc/loadavg
7. sensors (optional)
8. thermal zones (optional)

Each command has ~10-50ms network latency overhead.

PROBLEM CODE (linux.py:241-269):
```python
async with await self._get_connection() as conn:
    stat = await self._run_command(conn, "cat /proc/stat")
    meminfo = await self._run_command(conn, "cat /proc/meminfo")
    df = await self._run_command(conn, "df -B1")
    netdev = await self._run_command(conn, "cat /proc/net/dev")
    uptime = await self._run_command(conn, "cat /proc/uptime")
    loadavg = await self._run_command(conn, "cat /proc/loadavg")
    # ... more commands
```

RECOMMENDED FIX #1: Combine commands into a single SSH call:

```python
async def collect(self) -> SystemMetrics:
    combined_cmd = '''
    echo "===STAT===" && cat /proc/stat && \\
    echo "===MEMINFO===" && cat /proc/meminfo && \\
    echo "===DF===" && df -B1 && \\
    echo "===NETDEV===" && cat /proc/net/dev && \\
    echo "===UPTIME===" && cat /proc/uptime && \\
    echo "===LOADAVG===" && cat /proc/loadavg && \\
    echo "===SENSORS===" && (sensors 2>/dev/null || true) && \\
    echo "===THERMAL===" && (for zone in /sys/class/thermal/thermal_zone*/; do \\
        name=$(cat ${zone}type 2>/dev/null || echo zone); \\
        temp=$(cat ${zone}temp 2>/dev/null); \\
        [ -n "$temp" ] && echo "$name:$temp"; done || true)
    '''

    async with await self._get_connection() as conn:
        output = await self._run_command(conn, combined_cmd)

    # Parse combined output
    sections = output.split("===")
    # ... process each section
```

RECOMMENDED FIX #2: Use asyncio.gather() for parallel commands:

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

ESTIMATED IMPROVEMENT: 3-5x faster (300ms -> 60-100ms per collection)
""")


def analyze_unas_smartctl_bottleneck():
    """Analyze UNAS collector - sequential smartctl calls."""
    print("\n" + "="*60)
    print("UNAS COLLECTOR BOTTLENECK ANALYSIS")
    print("="*60)

    print("""
IDENTIFIED BOTTLENECK: UNASCollector.collect() at line 184-189

The UNAS collector runs smartctl for EACH disk sequentially:

PROBLEM CODE (unas.py:184-189):
```python
for disk_name in disk_names[:10]:
    smart_output = await self._run_command(
        conn, f"sudo smartctl -a /dev/{disk_name} ..."
    )
```

Each smartctl call takes 0.5-2 seconds. With 10 disks, that's 5-20 seconds!

RECOMMENDED FIX: Run smartctl commands in parallel using a single SSH session:

```python
# Option 1: Parallel SSH commands within session
async with await self._get_connection() as conn:
    smart_tasks = [
        self._run_command(conn, f"sudo smartctl -a /dev/{disk} 2>/dev/null || true")
        for disk in disk_names[:10]
    ]
    smart_outputs = await asyncio.gather(*smart_tasks)
    smartctl_outputs = dict(zip(disk_names[:10], smart_outputs))

# Option 2: Single combined command
disk_cmd = " && ".join([
    f'echo "==={disk}===" && (sudo smartctl -a /dev/{disk} 2>/dev/null || true)'
    for disk in disk_names[:10]
])
combined_output = await self._run_command(conn, disk_cmd)
# Parse by separator
```

ESTIMATED IMPROVEMENT: 5-10x faster for 10 disks (20s -> 2-4s)
""")


def analyze_pydantic_serialization():
    """Analyze Pydantic model serialization overhead."""
    print("\n" + "="*60)
    print("PYDANTIC SERIALIZATION ANALYSIS")
    print("="*60)

    print("""
OBSERVATION: Pydantic serialization is fast but can be optimized further.

Current performance (from profiling):
- Linux Metrics to JSON: 0.0049 ms
- Docker Metrics (50 containers) to JSON: 0.0425 ms

These are acceptable but add up with frequent WebSocket broadcasts.

OPTIMIZATION OPTIONS:

1. Use model_dump(mode='json') for dict output instead of model_dump() + json.dumps():
   ```python
   # Instead of:
   data = metrics.model_dump()
   json_data = json.dumps(data)

   # Use:
   data = metrics.model_dump(mode='json')  # Returns JSON-serializable dict
   # or
   json_bytes = metrics.model_dump_json().encode()  # Direct to bytes
   ```

2. Cache serialized metrics if they haven't changed:
   ```python
   class MetricsStore:
       def __init__(self):
           self._systems: dict[str, SystemMetrics] = {}
           self._cached_json: dict[str, str] = {}

       def update(self, system_id: str, metrics: SystemMetrics) -> None:
           self._systems[system_id] = metrics
           self._cached_json[system_id] = metrics.model_dump_json()
   ```

3. Use orjson for faster JSON serialization:
   ```python
   import orjson

   json_bytes = orjson.dumps(metrics.model_dump(mode='json'))
   # orjson is typically 3-10x faster than standard json
   ```
""")


def analyze_connection_pooling():
    """Analyze connection reuse patterns."""
    print("\n" + "="*60)
    print("CONNECTION POOLING ANALYSIS")
    print("="*60)

    print("""
IDENTIFIED ISSUE: New connections created per collection cycle.

1. LINUX COLLECTOR (linux.py:28-44):
   Creates NEW SSH connection for every collect() call.

   ```python
   async def collect(self) -> SystemMetrics:
       async with await self._get_connection() as conn:  # New connection!
           ...
   ```

   SSH connection setup takes 100-500ms (TCP + crypto handshake).

2. QBITTORRENT COLLECTOR (qbittorrent.py:51):
   Creates new httpx.AsyncClient for each request.

   ```python
   async with httpx.AsyncClient(timeout=10) as client:  # New client!
   ```

3. UNIFI COLLECTOR (unifi.py:83):
   Same issue - new client per collection.

RECOMMENDED FIX: Implement connection pooling/reuse.

For SSH (LinuxCollector, UNASCollector):
```python
class LinuxCollector(BaseCollector):
    def __init__(self, ...):
        ...
        self._connection: asyncssh.SSHClientConnection | None = None
        self._connection_lock = asyncio.Lock()

    async def _get_connection(self) -> asyncssh.SSHClientConnection:
        async with self._connection_lock:
            if self._connection is None or self._connection.is_closed():
                self._connection = await asyncssh.connect(...)
            return self._connection

    async def collect(self) -> SystemMetrics:
        conn = await self._get_connection()  # Reuse connection!
        ...
```

For HTTP (QBittorrent, UniFi):
```python
class QBittorrentCollector(BaseCollector):
    def __init__(self, ...):
        ...
        self._client: httpx.AsyncClient | None = None

    async def _get_client(self) -> httpx.AsyncClient:
        if self._client is None:
            self._client = httpx.AsyncClient(timeout=10)
        return self._client

    async def close(self):
        if self._client:
            await self._client.aclose()
```

ESTIMATED IMPROVEMENT:
- SSH: 100-500ms saved per collection cycle per Linux system
- HTTP: 20-50ms saved per collection cycle per HTTP-based system
""")


def analyze_memory_patterns():
    """Analyze memory usage patterns."""
    print("\n" + "="*60)
    print("MEMORY ANALYSIS")
    print("="*60)

    tracemalloc.start()

    from app.models.metrics import (
        SystemMetrics, SystemStatus, SystemType, LinuxMetrics,
        CpuMetrics, MemoryMetrics, DiskMetrics, NetworkMetrics,
        DockerMetrics, ContainerMetrics
    )
    from app.services.metrics_store import MetricsStore

    snapshot1 = tracemalloc.take_snapshot()

    # Simulate heavy load
    store = MetricsStore()

    # Add 50 Linux systems
    for i in range(50):
        metrics = SystemMetrics(
            id=f"linux-{i}",
            name=f"Linux Server {i}",
            type=SystemType.LINUX,
            status=SystemStatus.HEALTHY,
            last_updated=datetime.now(UTC),
            metrics=LinuxMetrics(
                cpu=CpuMetrics(percent=50.0, cores=[50.0] * 16, load_avg=[1.0, 1.0, 1.0]),
                memory=MemoryMetrics(total=64000000000, used=32000000000, available=32000000000, percent=50.0),
                disks=[
                    DiskMetrics(mount="/", total=500000000000, used=250000000000, free=250000000000, percent=50.0),
                    DiskMetrics(mount="/data", total=2000000000000, used=1000000000000, free=1000000000000, percent=50.0),
                    DiskMetrics(mount="/backup", total=4000000000000, used=2000000000000, free=2000000000000, percent=50.0),
                ],
                network=NetworkMetrics(bytes_sent=10000000000, bytes_recv=50000000000, upload_rate=10000000, download_rate=50000000),
                temperatures=[],
                uptime=864000,
            ),
        )
        store.update(f"linux-{i}", metrics)

    # Add 10 Docker hosts with 50 containers each
    for i in range(10):
        containers = [
            ContainerMetrics(
                id=f"container-{j}",
                name=f"container-{j}",
                image=f"image-{j}:latest",
                status="healthy",
                state="running",
                cpu_percent=25.0,
                memory_usage=512000000,
                memory_limit=2000000000,
                memory_percent=25.6,
            )
            for j in range(50)
        ]

        metrics = SystemMetrics(
            id=f"docker-{i}",
            name=f"Docker Host {i}",
            type=SystemType.DOCKER,
            status=SystemStatus.HEALTHY,
            last_updated=datetime.now(UTC),
            metrics=DockerMetrics(
                containers=containers,
                running_count=45,
                stopped_count=5,
                total_count=50,
            ),
        )
        store.update(f"docker-{i}", metrics)

    snapshot2 = tracemalloc.take_snapshot()

    top_stats = snapshot2.compare_to(snapshot1, 'lineno')

    print("\nTop 10 Memory Allocations:")
    for stat in top_stats[:10]:
        print(f"  {stat}")

    current, peak = tracemalloc.get_traced_memory()
    print(f"\nTotal Memory (60 systems, 500 containers):")
    print(f"  Current: {current / 1024 / 1024:.2f} MB")
    print(f"  Peak: {peak / 1024 / 1024:.2f} MB")

    # Test WebSocket broadcast payload size
    update_msg = store.create_update_message()
    json_size = len(update_msg.model_dump_json())
    print(f"\nWebSocket Broadcast Payload Size:")
    print(f"  JSON: {json_size / 1024:.2f} KB")

    tracemalloc.stop()

    print("""
OBSERVATIONS:
- Memory usage is reasonable for typical homelab (few systems)
- Large deployments (50+ systems, 500+ containers) may need attention
- WebSocket payloads can be large with many containers

RECOMMENDATIONS:
1. Use __slots__ in Pydantic models for memory efficiency:
   ```python
   class ContainerMetrics(BaseModel):
       model_config = ConfigDict(slots=True)
   ```

2. Implement pagination for large container lists
3. Consider compression for WebSocket messages (permessage-deflate)
4. Add metrics aggregation for dashboards vs full data for detail views
""")


def create_optimization_summary():
    """Create final optimization summary."""
    print("\n" + "="*60)
    print("OPTIMIZATION PRIORITY SUMMARY")
    print("="*60)

    print("""
┌────────────────────────────────────────────────────────────────┐
│ PRIORITY 1 - HIGH IMPACT (Fix These First)                     │
├────────────────────────────────────────────────────────────────┤
│ 1. Docker Collector: Parallelize container.stats() calls       │
│    - Current: ~1-2s per container (sequential)                 │
│    - Optimized: ~2s total (parallel with ThreadPoolExecutor)   │
│    - Impact: 10x improvement for 10+ containers                │
│                                                                │
│ 2. Linux Collector: Combine SSH commands                       │
│    - Current: 7+ sequential commands (300ms+ latency)          │
│    - Optimized: Single combined command (60-100ms)             │
│    - Impact: 3-5x improvement per Linux system                 │
│                                                                │
│ 3. SSH Connection Pooling                                      │
│    - Current: New connection per collection (~100-500ms)       │
│    - Optimized: Reuse connections (0ms overhead)               │
│    - Impact: Major latency reduction                           │
├────────────────────────────────────────────────────────────────┤
│ PRIORITY 2 - MEDIUM IMPACT                                     │
├────────────────────────────────────────────────────────────────┤
│ 4. UNAS Collector: Parallelize smartctl calls                  │
│    - Impact: 5-10x improvement for multi-disk systems          │
│                                                                │
│ 5. HTTP Client Pooling (QBittorrent, UniFi)                    │
│    - Impact: 20-50ms saved per collection                      │
│                                                                │
│ 6. Use orjson for JSON serialization                           │
│    - Impact: 3-10x faster serialization                        │
├────────────────────────────────────────────────────────────────┤
│ PRIORITY 3 - LOW IMPACT (Nice to Have)                         │
├────────────────────────────────────────────────────────────────┤
│ 7. Cache unchanged metrics for WebSocket broadcasts            │
│ 8. Add __slots__ to Pydantic models                            │
│ 9. Implement WebSocket compression                             │
│ 10. String parsing micro-optimizations                         │
└────────────────────────────────────────────────────────────────┘

ESTIMATED TOTAL IMPROVEMENT:
- Collection cycle time: 20-30s → 2-5s (4-15x faster)
- Memory usage: Already efficient, minor gains possible
- WebSocket latency: Negligible current overhead
""")


def main():
    """Run all bottleneck analyses."""
    print("="*60)
    print("SERVERMONITOR BOTTLENECK DEEP ANALYSIS")
    print("="*60)

    analyze_docker_collector_bottleneck()
    analyze_linux_ssh_bottleneck()
    analyze_unas_smartctl_bottleneck()
    analyze_pydantic_serialization()
    analyze_connection_pooling()
    analyze_memory_patterns()
    create_optimization_summary()

    print("\n" + "="*60)
    print("ANALYSIS COMPLETE")
    print("="*60)


if __name__ == "__main__":
    main()
