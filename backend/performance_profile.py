#!/usr/bin/env python3
"""
Comprehensive performance profiling for ServerMonitor backend.
Profiles CPU usage, memory, and identifies bottlenecks.
"""

import asyncio
import cProfile
import io
import pstats
import sys
import time
import tracemalloc
from functools import wraps
from pstats import SortKey

# Add app to path
sys.path.insert(0, ".")


def profile_cpu(func):
    """Decorator to profile CPU usage of a function."""
    @wraps(func)
    def wrapper(*args, **kwargs):
        profiler = cProfile.Profile()
        profiler.enable()
        try:
            result = func(*args, **kwargs)
        finally:
            profiler.disable()
            stream = io.StringIO()
            stats = pstats.Stats(profiler, stream=stream)
            stats.sort_stats(SortKey.CUMULATIVE)
            stats.print_stats(20)
            print(stream.getvalue())
        return result
    return wrapper


def profile_memory(func):
    """Decorator to profile memory usage."""
    @wraps(func)
    def wrapper(*args, **kwargs):
        tracemalloc.start()
        snapshot1 = tracemalloc.take_snapshot()

        result = func(*args, **kwargs)

        snapshot2 = tracemalloc.take_snapshot()
        top_stats = snapshot2.compare_to(snapshot1, 'lineno')

        print("\n=== Memory Allocation (Top 15) ===")
        for stat in top_stats[:15]:
            print(stat)

        current, peak = tracemalloc.get_traced_memory()
        print(f"\nCurrent memory: {current / 1024 / 1024:.2f} MB")
        print(f"Peak memory: {peak / 1024 / 1024:.2f} MB")
        tracemalloc.stop()

        return result
    return wrapper


def benchmark(func, iterations=100):
    """Benchmark a function."""
    times = []
    for _ in range(iterations):
        start = time.perf_counter()
        func()
        elapsed = time.perf_counter() - start
        times.append(elapsed)

    avg_time = sum(times) / len(times)
    min_time = min(times)
    max_time = max(times)

    return {
        "avg": avg_time,
        "min": min_time,
        "max": max_time,
        "total": sum(times),
        "iterations": iterations
    }


# ============================================
# Profile Parsing Functions (CPU-bound code)
# ============================================

def profile_parsing_functions():
    """Profile the parsing functions from collectors."""
    print("\n" + "="*60)
    print("PROFILING PARSING FUNCTIONS")
    print("="*60)

    from app.collectors.linux import LinuxCollector
    from app.collectors.unas import UNASCollector
    from app.config import LinuxSystemConfig, UNASSystemConfig

    # Create test instances
    linux_config = LinuxSystemConfig(host="test", username="test")
    linux_collector = LinuxCollector("test", "Test", linux_config)

    unas_config = UNASSystemConfig(host="test", username="test")
    unas_collector = UNASCollector("test", "Test", unas_config)

    # Sample data for parsing tests
    sample_stat = """cpu  1234567 12345 234567 8765432 12345 0 1234 0 0 0
cpu0 123456 1234 23456 876543 1234 0 123 0 0 0
cpu1 123456 1234 23456 876543 1234 0 123 0 0 0
cpu2 123456 1234 23456 876543 1234 0 123 0 0 0
cpu3 123456 1234 23456 876543 1234 0 123 0 0 0
"""

    sample_meminfo = """MemTotal:       16384000 kB
MemFree:         4096000 kB
MemAvailable:    8192000 kB
Buffers:          512000 kB
Cached:          2048000 kB
SwapTotal:       8192000 kB
SwapFree:        8192000 kB
"""

    sample_df = """Filesystem     1B-blocks      Used Available Use% Mounted on
/dev/sda1      107374182400 53687091200 53687091200  50% /
/dev/sdb1      1099511627776 549755813888 549755813888  50% /data
"""

    sample_netdev = """Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1234567890     1234    0    0    0     0          0         0 1234567890     1234    0    0    0     0       0          0
  eth0: 9876543210     9876    0    0    0     0          0         0 1234567890     1234    0    0    0     0       0          0
"""

    sample_sensors = """coretemp-isa-0000
Core 0:       +45.0°C  (high = +80.0°C, crit = +100.0°C)
Core 1:       +46.0°C  (high = +80.0°C, crit = +100.0°C)
Core 2:       +44.0°C  (high = +80.0°C, crit = +100.0°C)
Core 3:       +47.0°C  (high = +80.0°C, crit = +100.0°C)
"""

    sample_zpool = """NAME    SIZE  ALLOC   FREE  CKPOINT  EXPANDSZ   FRAG    CAP  DEDUP    HEALTH  ALTROOT
tank    1.0T   500G   500G        -         -     5%    50%  1.00x    ONLINE  -
backup  2.0T   1.0T   1.0T        -         -     3%    50%  1.00x    ONLINE  -
"""

    # Benchmark each parsing function
    print("\n--- CPU Stats Parsing ---")
    result = benchmark(lambda: linux_collector._parse_cpu_stats(sample_stat), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Memory Info Parsing ---")
    result = benchmark(lambda: linux_collector._parse_meminfo(sample_meminfo), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Disk (df) Parsing ---")
    result = benchmark(lambda: linux_collector._parse_df(sample_df), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Network Stats Parsing ---")
    result = benchmark(lambda: linux_collector._parse_net_stats(sample_netdev), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Sensors Parsing ---")
    result = benchmark(lambda: linux_collector._parse_sensors(sample_sensors), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- ZPool List Parsing ---")
    result = benchmark(lambda: unas_collector._parse_zpool_list(sample_zpool), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")


# ============================================
# Profile Metrics Store Operations
# ============================================

def profile_metrics_store():
    """Profile metrics store operations."""
    print("\n" + "="*60)
    print("PROFILING METRICS STORE")
    print("="*60)

    from datetime import datetime
    from app.services.metrics_store import MetricsStore
    from app.models.metrics import SystemMetrics, SystemStatus, SystemType, LinuxMetrics, CpuMetrics, MemoryMetrics, NetworkMetrics

    store = MetricsStore()

    # Create sample metrics
    def create_sample_metrics(system_id: str) -> SystemMetrics:
        return SystemMetrics(
            id=system_id,
            name=f"System {system_id}",
            type=SystemType.LINUX,
            status=SystemStatus.HEALTHY,
            last_updated=datetime.utcnow(),
            metrics=LinuxMetrics(
                cpu=CpuMetrics(percent=50.0, cores=[50.0, 50.0, 50.0, 50.0], load_avg=[1.0, 1.0, 1.0]),
                memory=MemoryMetrics(total=16000000000, used=8000000000, available=8000000000, percent=50.0),
                disks=[],
                network=NetworkMetrics(bytes_sent=1000000, bytes_recv=5000000, upload_rate=1000, download_rate=5000),
                temperatures=[],
                uptime=86400,
            ),
        )

    print("\n--- Single Update Operation ---")
    metrics = create_sample_metrics("test-1")
    result = benchmark(lambda: store.update("test-1", metrics), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Get Operation ---")
    result = benchmark(lambda: store.get("test-1"), iterations=10000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    # Populate store with many systems
    for i in range(100):
        store.update(f"system-{i}", create_sample_metrics(f"system-{i}"))

    print("\n--- Get All (100 systems) ---")
    result = benchmark(lambda: store.get_all(), iterations=1000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Create Update Message (100 systems) ---")
    result = benchmark(lambda: store.create_update_message(), iterations=1000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")


# ============================================
# Profile Model Serialization
# ============================================

def profile_model_serialization():
    """Profile Pydantic model serialization."""
    print("\n" + "="*60)
    print("PROFILING MODEL SERIALIZATION")
    print("="*60)

    from datetime import datetime
    from app.models.metrics import (
        SystemMetrics, SystemStatus, SystemType, LinuxMetrics,
        CpuMetrics, MemoryMetrics, DiskMetrics, NetworkMetrics,
        DockerMetrics, ContainerMetrics
    )

    # Complex Linux metrics
    linux_metrics = SystemMetrics(
        id="linux-1",
        name="Linux Server",
        type=SystemType.LINUX,
        status=SystemStatus.HEALTHY,
        last_updated=datetime.utcnow(),
        metrics=LinuxMetrics(
            cpu=CpuMetrics(
                percent=65.5,
                cores=[60.0, 70.0, 65.0, 66.0, 62.0, 68.0, 64.0, 67.0],
                load_avg=[2.5, 2.0, 1.8]
            ),
            memory=MemoryMetrics(
                total=32000000000,
                used=24000000000,
                available=8000000000,
                percent=75.0
            ),
            disks=[
                DiskMetrics(mount="/", total=500000000000, used=250000000000, free=250000000000, percent=50.0),
                DiskMetrics(mount="/data", total=2000000000000, used=1500000000000, free=500000000000, percent=75.0),
            ],
            network=NetworkMetrics(bytes_sent=1000000000, bytes_recv=5000000000, upload_rate=1000000, download_rate=5000000),
            temperatures=[],
            uptime=864000,
        ),
    )

    # Docker metrics with many containers
    containers = [
        ContainerMetrics(
            id=f"abc{i:04d}",
            name=f"container-{i}",
            image=f"image-{i}:latest",
            status="healthy" if i % 2 == 0 else "running",
            state="running",
            cpu_percent=float(i % 100),
            memory_usage=i * 1000000,
            memory_limit=8000000000,
            memory_percent=float(i % 100) / 10,
        )
        for i in range(50)
    ]

    docker_metrics = SystemMetrics(
        id="docker-1",
        name="Docker Host",
        type=SystemType.DOCKER,
        status=SystemStatus.HEALTHY,
        last_updated=datetime.utcnow(),
        metrics=DockerMetrics(
            containers=containers,
            running_count=40,
            stopped_count=10,
            total_count=50,
        ),
    )

    print("\n--- Linux Metrics to dict ---")
    result = benchmark(lambda: linux_metrics.model_dump(), iterations=5000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Linux Metrics to JSON ---")
    result = benchmark(lambda: linux_metrics.model_dump_json(), iterations=5000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Docker Metrics (50 containers) to dict ---")
    result = benchmark(lambda: docker_metrics.model_dump(), iterations=1000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Docker Metrics (50 containers) to JSON ---")
    result = benchmark(lambda: docker_metrics.model_dump_json(), iterations=1000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")


# ============================================
# Profile Memory Usage
# ============================================

def profile_memory_usage():
    """Profile memory usage of key components."""
    print("\n" + "="*60)
    print("PROFILING MEMORY USAGE")
    print("="*60)

    import sys
    from datetime import datetime
    from app.models.metrics import (
        SystemMetrics, SystemStatus, SystemType, LinuxMetrics,
        CpuMetrics, MemoryMetrics, DiskMetrics, NetworkMetrics
    )
    from app.services.metrics_store import MetricsStore

    def get_size(obj, seen=None):
        """Recursively calculate object size."""
        size = sys.getsizeof(obj)
        if seen is None:
            seen = set()
        obj_id = id(obj)
        if obj_id in seen:
            return 0
        seen.add(obj_id)
        if isinstance(obj, dict):
            size += sum([get_size(v, seen) for v in obj.values()])
            size += sum([get_size(k, seen) for k in obj.keys()])
        elif hasattr(obj, '__dict__'):
            size += get_size(obj.__dict__, seen)
        elif hasattr(obj, '__iter__') and not isinstance(obj, (str, bytes, bytearray)):
            size += sum([get_size(i, seen) for i in obj])
        return size

    # Single metrics object
    metrics = SystemMetrics(
        id="linux-1",
        name="Linux Server",
        type=SystemType.LINUX,
        status=SystemStatus.HEALTHY,
        last_updated=datetime.utcnow(),
        metrics=LinuxMetrics(
            cpu=CpuMetrics(percent=50.0, cores=[50.0] * 8, load_avg=[1.0, 1.0, 1.0]),
            memory=MemoryMetrics(total=16000000000, used=8000000000, available=8000000000, percent=50.0),
            disks=[DiskMetrics(mount="/", total=500000000000, used=250000000000, free=250000000000, percent=50.0)],
            network=NetworkMetrics(bytes_sent=1000000, bytes_recv=5000000, upload_rate=1000, download_rate=5000),
            temperatures=[],
            uptime=86400,
        ),
    )

    print(f"\n--- Single SystemMetrics Object ---")
    print(f"  Size: {get_size(metrics) / 1024:.2f} KB")

    # Metrics store with many systems
    store = MetricsStore()
    for i in range(100):
        store.update(f"system-{i}", metrics)

    print(f"\n--- MetricsStore with 100 systems ---")
    print(f"  Size: {get_size(store) / 1024:.2f} KB")

    # JSON serialized size
    json_data = metrics.model_dump_json()
    print(f"\n--- Single SystemMetrics as JSON ---")
    print(f"  Size: {len(json_data) / 1024:.2f} KB")


# ============================================
# Profile String Operations (Common Bottleneck)
# ============================================

def profile_string_operations():
    """Profile string parsing patterns."""
    print("\n" + "="*60)
    print("PROFILING STRING OPERATIONS")
    print("="*60)

    # Large sample data
    large_meminfo = "\n".join([f"Key{i}:        {i*1000} kB" for i in range(100)])

    def parse_meminfo_original(meminfo: str) -> dict:
        """Original parsing method."""
        info = {}
        for line in meminfo.strip().split("\n"):
            if ":" in line:
                key, value = line.split(":", 1)
                value = value.strip().split()[0]
                info[key] = int(value) * 1024
        return info

    def parse_meminfo_optimized(meminfo: str) -> dict:
        """Optimized parsing method using partition."""
        info = {}
        for line in meminfo.strip().split("\n"):
            key, sep, value = line.partition(":")
            if sep:
                info[key] = int(value.split()[0]) * 1024
        return info

    print("\n--- Original Meminfo Parsing (100 lines) ---")
    result = benchmark(lambda: parse_meminfo_original(large_meminfo), iterations=5000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")

    print("\n--- Optimized Meminfo Parsing (100 lines) ---")
    result = benchmark(lambda: parse_meminfo_optimized(large_meminfo), iterations=5000)
    print(f"  Avg: {result['avg']*1000:.4f} ms, Min: {result['min']*1000:.4f} ms, Max: {result['max']*1000:.4f} ms")


# ============================================
# cProfile Full Analysis
# ============================================

def run_cprofile_analysis():
    """Run cProfile on key operations."""
    print("\n" + "="*60)
    print("CPROFILE DETAILED ANALYSIS")
    print("="*60)

    from datetime import datetime
    from app.collectors.linux import LinuxCollector
    from app.config import LinuxSystemConfig
    from app.models.metrics import SystemMetrics, SystemStatus, SystemType, LinuxMetrics, CpuMetrics, MemoryMetrics, NetworkMetrics
    from app.services.metrics_store import MetricsStore

    linux_config = LinuxSystemConfig(host="test", username="test")
    linux_collector = LinuxCollector("test", "Test", linux_config)
    store = MetricsStore()

    sample_stat = """cpu  1234567 12345 234567 8765432 12345 0 1234 0 0 0
cpu0 123456 1234 23456 876543 1234 0 123 0 0 0
cpu1 123456 1234 23456 876543 1234 0 123 0 0 0
"""
    sample_meminfo = """MemTotal:       16384000 kB
MemFree:         4096000 kB
MemAvailable:    8192000 kB
"""

    def test_function():
        """Function to profile."""
        for _ in range(1000):
            linux_collector._parse_cpu_stats(sample_stat)
            linux_collector._parse_meminfo(sample_meminfo)

            metrics = SystemMetrics(
                id="test",
                name="Test",
                type=SystemType.LINUX,
                status=SystemStatus.HEALTHY,
                last_updated=datetime.utcnow(),
                metrics=LinuxMetrics(
                    cpu=CpuMetrics(percent=50.0, cores=[50.0], load_avg=[1.0]),
                    memory=MemoryMetrics(total=16000000000, used=8000000000, available=8000000000, percent=50.0),
                    disks=[],
                    network=NetworkMetrics(),
                    temperatures=[],
                    uptime=86400,
                ),
            )
            store.update("test", metrics)
            metrics.model_dump_json()

    profiler = cProfile.Profile()
    profiler.enable()
    test_function()
    profiler.disable()

    print("\n--- Top 25 Functions by Cumulative Time ---")
    stats = pstats.Stats(profiler)
    stats.sort_stats(SortKey.CUMULATIVE)
    stats.print_stats(25)


# ============================================
# Main
# ============================================

def main():
    """Run all profiling tests."""
    print("="*60)
    print("SERVERMONITOR PERFORMANCE PROFILING")
    print("="*60)

    profile_parsing_functions()
    profile_metrics_store()
    profile_model_serialization()
    profile_memory_usage()
    profile_string_operations()
    run_cprofile_analysis()

    print("\n" + "="*60)
    print("PROFILING COMPLETE")
    print("="*60)


if __name__ == "__main__":
    main()
