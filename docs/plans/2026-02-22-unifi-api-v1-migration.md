# UniFi API v1 Migration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace cookie/session-based UniFi auth with API key auth and migrate all endpoints to the official `/proxy/network/integration/v1/` API.

**Architecture:** Drop the `_login()` session flow entirely; every request sends `X-API-Key` header. Three endpoints replace the old two: `/devices`, `/clients`, `/health` (for WAN stats). Config loses `username`/`password`, gains `api_key`.

**Tech Stack:** Python 3.12, httpx (async), FastAPI/Pydantic, pytest + pytest-asyncio + respx (HTTP mocking)

---

### Task 1: Add test dependencies

**Files:**
- Modify: `backend/pyproject.toml`

**Step 1: Add pytest, pytest-asyncio, and respx to pyproject.toml**

```toml
[project.optional-dependencies]
dev = [
    "pytest==8.3.4",
    "pytest-asyncio==0.24.0",
    "respx==0.21.1",
]
```

**Step 2: Install them**

```bash
cd backend
uv pip install pytest pytest-asyncio "respx==0.21.1"
```

Expected: packages install without error.

**Step 3: Verify pytest runs**

```bash
cd backend
uv run pytest --collect-only
```

Expected: "no tests ran" (zero tests collected, no errors).

**Step 4: Commit**

```bash
git add backend/pyproject.toml
git commit -m "chore: add pytest, pytest-asyncio, respx for collector tests"
```

---

### Task 2: Update UnifiSystemConfig

**Files:**
- Modify: `backend/app/config.py:29-35`
- Create: `backend/tests/__init__.py`
- Create: `backend/tests/test_unifi_config.py`

**Step 1: Write the failing test**

Create `backend/tests/__init__.py` (empty).

Create `backend/tests/test_unifi_config.py`:

```python
from app.config import UnifiSystemConfig


def test_config_accepts_api_key():
    config = UnifiSystemConfig(host="192.168.1.1", api_key="test-key")
    assert config.api_key == "test-key"


def test_config_rejects_username_password():
    import pytest
    with pytest.raises(Exception):
        UnifiSystemConfig(host="192.168.1.1", username="admin", password="secret")


def test_config_defaults():
    config = UnifiSystemConfig(host="192.168.1.1", api_key="key")
    assert config.port == 443
    assert config.site == "default"
    assert config.verify_ssl is False
```

**Step 2: Run test to verify it fails**

```bash
cd backend
uv run pytest tests/test_unifi_config.py -v
```

Expected: FAIL — `UnifiSystemConfig` still has `username`/`password`, no `api_key`.

**Step 3: Update UnifiSystemConfig in config.py**

Replace the current `UnifiSystemConfig` class (lines 29–35):

```python
class UnifiSystemConfig(BaseModel):
    host: str
    port: int = 443
    api_key: str
    site: str = "default"
    verify_ssl: bool = False
```

**Step 4: Run test to verify it passes**

```bash
cd backend
uv run pytest tests/test_unifi_config.py -v
```

Expected: all 3 tests PASS.

**Step 5: Commit**

```bash
git add backend/app/config.py backend/tests/
git commit -m "feat(unifi): replace username/password config with api_key"
```

---

### Task 3: Rewrite UnifiCollector — auth and base URL

**Files:**
- Modify: `backend/app/collectors/unifi.py`
- Create: `backend/tests/test_unifi_collector.py`

The collector needs a `_headers` property and a `_base_url` property. Test these in isolation before touching `collect()`.

**Step 1: Write the failing tests**

Create `backend/tests/test_unifi_collector.py`:

```python
import pytest
from app.collectors.unifi import UnifiCollector
from app.config import UnifiSystemConfig


@pytest.fixture
def collector():
    config = UnifiSystemConfig(
        host="192.168.1.1",
        port=443,
        api_key="test-api-key-123",
        site="default",
        verify_ssl=False,
    )
    return UnifiCollector(system_id="test", name="Test UniFi", config=config)


def test_headers_contain_api_key(collector):
    assert collector._headers == {"X-API-Key": "test-api-key-123"}


def test_base_url(collector):
    assert collector._base_url == "https://192.168.1.1:443/proxy/network/integration/v1"


def test_no_login_method(collector):
    assert not hasattr(collector, "_login")


def test_no_cookies(collector):
    assert not hasattr(collector, "_cookies")
```

**Step 2: Run test to verify it fails**

```bash
cd backend
uv run pytest tests/test_unifi_collector.py -v
```

Expected: FAIL — `_headers` and `_base_url` don't exist yet, `_login` and `_cookies` still exist.

**Step 3: Rewrite the collector**

Replace the full contents of `backend/app/collectors/unifi.py`:

```python
from datetime import datetime

import httpx

from app.collectors.base import BaseCollector
from app.config import UnifiSystemConfig
from app.models.metrics import (
    SystemMetrics,
    SystemStatus,
    SystemType,
    UnifiDevice,
    UnifiMetrics,
)


class UnifiCollector(BaseCollector):
    def __init__(self, system_id: str, name: str, config: UnifiSystemConfig):
        super().__init__(system_id, name, SystemType.UNIFI)
        self.config = config

    @property
    def _headers(self) -> dict:
        return {"X-API-Key": self.config.api_key}

    @property
    def _base_url(self) -> str:
        return f"https://{self.config.host}:{self.config.port}/proxy/network/integration/v1"

    def _site_url(self, path: str) -> str:
        return f"{self._base_url}/sites/{self.config.site}/{path}"

    def determine_status(self, metrics: UnifiMetrics) -> SystemStatus:
        offline = sum(1 for d in metrics.devices if d.status != "online")
        if offline > len(metrics.devices) / 2:
            return SystemStatus.CRITICAL
        if offline > 0:
            return SystemStatus.WARNING
        return SystemStatus.HEALTHY

    async def collect(self) -> SystemMetrics:
        try:
            async with httpx.AsyncClient(
                timeout=10, verify=self.config.verify_ssl
            ) as client:
                devices_resp = await client.get(
                    self._site_url("devices"), headers=self._headers
                )
                devices_resp.raise_for_status()
                devices_data = devices_resp.json().get("data", [])

                clients_resp = await client.get(
                    self._site_url("clients"), headers=self._headers
                )
                clients_resp.raise_for_status()
                clients_data = clients_resp.json().get("data", [])

                health_resp = await client.get(
                    self._site_url("health"), headers=self._headers
                )
                health_resp.raise_for_status()
                health_data = health_resp.json().get("data", [])
        except Exception as e:
            return self.create_error_metrics(str(e))

        devices = []
        for device in devices_data:
            state = device.get("state", "")
            status = "online" if state == "connected" else "offline"
            devices.append(
                UnifiDevice(
                    name=device.get("name", device.get("mac", "Unknown")),
                    mac=device.get("mac", ""),
                    model=device.get("model", "Unknown"),
                    status=status,
                    uptime=device.get("uptime", 0),
                )
            )

        guests_count = sum(1 for c in clients_data if c.get("isGuest", False))
        clients_count = len(clients_data) - guests_count

        wan_download, wan_upload = _parse_wan_rates(health_data)

        unifi_metrics = UnifiMetrics(
            devices=devices,
            clients_count=clients_count,
            guests_count=guests_count,
            wan_download=round(wan_download, 2),
            wan_upload=round(wan_upload, 2),
        )

        return SystemMetrics(
            id=self.system_id,
            name=self.name,
            type=SystemType.UNIFI,
            status=self.determine_status(unifi_metrics),
            last_updated=datetime.utcnow(),
            metrics=unifi_metrics,
        )


def _parse_wan_rates(health_data: list) -> tuple[float, float]:
    """Extract WAN download/upload Mbps from health endpoint data."""
    for subsystem in health_data:
        if subsystem.get("subsystem") == "wan":
            rx = subsystem.get("rxBytesPerSecond", 0) or 0
            tx = subsystem.get("txBytesPerSecond", 0) or 0
            return rx / 1_000_000, tx / 1_000_000
    return 0.0, 0.0
```

**Step 4: Run test to verify it passes**

```bash
cd backend
uv run pytest tests/test_unifi_collector.py -v
```

Expected: all 4 tests PASS.

**Step 5: Commit**

```bash
git add backend/app/collectors/unifi.py backend/tests/test_unifi_collector.py
git commit -m "feat(unifi): rewrite collector with API key auth and v1 endpoints"
```

---

### Task 4: Test collect() with mocked HTTP

**Files:**
- Modify: `backend/tests/test_unifi_collector.py`

Test the full `collect()` flow using `respx` to mock the three HTTP calls.

**Step 1: Add integration tests to the existing test file**

Append to `backend/tests/test_unifi_collector.py`:

```python
import respx
from httpx import Response
from app.models.metrics import SystemStatus, UnifiMetrics


DEVICES_RESPONSE = {
    "data": [
        {
            "mac": "aa:bb:cc:dd:ee:ff",
            "name": "Living Room AP",
            "model": "U6-Lite",
            "state": "connected",
            "uptime": 86400,
        },
        {
            "mac": "11:22:33:44:55:66",
            "name": "Office AP",
            "model": "U6-Pro",
            "state": "disconnected",
            "uptime": 0,
        },
    ]
}

CLIENTS_RESPONSE = {
    "data": [
        {"mac": "de:ad:be:ef:00:01", "isGuest": False},
        {"mac": "de:ad:be:ef:00:02", "isGuest": False},
        {"mac": "de:ad:be:ef:00:03", "isGuest": True},
    ]
}

HEALTH_RESPONSE = {
    "data": [
        {"subsystem": "wan", "rxBytesPerSecond": 5_000_000, "txBytesPerSecond": 1_000_000},
        {"subsystem": "wlan"},
    ]
}


@pytest.mark.asyncio
@respx.mock
async def test_collect_parses_devices(collector):
    base = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"
    respx.get(f"{base}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{base}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{base}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert len(metrics.devices) == 2
    assert metrics.devices[0].name == "Living Room AP"
    assert metrics.devices[0].status == "online"
    assert metrics.devices[1].status == "offline"


@pytest.mark.asyncio
@respx.mock
async def test_collect_counts_clients(collector):
    base = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"
    respx.get(f"{base}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{base}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{base}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert metrics.clients_count == 2
    assert metrics.guests_count == 1


@pytest.mark.asyncio
@respx.mock
async def test_collect_parses_wan_rates(collector):
    base = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"
    respx.get(f"{base}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{base}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{base}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert metrics.wan_download == 5.0   # 5_000_000 bytes/s → 5 Mbps
    assert metrics.wan_upload == 1.0


@pytest.mark.asyncio
@respx.mock
async def test_collect_returns_error_on_http_failure(collector):
    base = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"
    respx.get(f"{base}/devices").mock(return_value=Response(401))
    respx.get(f"{base}/clients").mock(return_value=Response(401))
    respx.get(f"{base}/health").mock(return_value=Response(401))

    result = await collector.collect()

    assert result.error is not None
    assert result.status.value == "offline"


@pytest.mark.asyncio
@respx.mock
async def test_collect_status_warning_when_one_device_offline(collector):
    base = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"
    respx.get(f"{base}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{base}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{base}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()

    # 1 of 2 devices offline → WARNING
    assert result.status == SystemStatus.WARNING
```

**Step 2: Add pytest-asyncio config to pyproject.toml**

Append to `backend/pyproject.toml`:

```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
```

**Step 3: Run tests to verify they fail**

```bash
cd backend
uv run pytest tests/test_unifi_collector.py -v -k "collect"
```

Expected: ImportError or test failures (respx not wired yet, or collect not implemented).

**Step 4: Run all tests**

```bash
cd backend
uv run pytest tests/ -v
```

Expected: all tests PASS (the collector was already written in Task 3).

**Step 5: Commit**

```bash
git add backend/tests/test_unifi_collector.py backend/pyproject.toml
git commit -m "test(unifi): add collect() integration tests with respx mocks"
```

---

### Task 5: Verify no regressions in dependent code

**Files:**
- Read: `backend/app/services/collector_manager.py`
- Read: `backend/app/api/routes.py`

**Step 1: Check collector_manager instantiates UnifiCollector**

```bash
cd backend
grep -n "UnifiCollector\|unifi" app/services/collector_manager.py
```

Confirm it passes `config` from `UnifiSystemConfig` — no username/password references. If it passes individual fields instead of the config object, update accordingly.

**Step 2: Run full test suite**

```bash
cd backend
uv run pytest tests/ -v
```

Expected: all tests PASS, zero errors.

**Step 3: Smoke-test server startup**

```bash
cd backend
uv run uvicorn app.main:app --host 0.0.0.0 --port 8742 &
sleep 2
curl -s http://localhost:8742/api/systems | python3 -m json.tool | head -20
kill %1
```

Expected: JSON response, no import errors or crashes.

**Step 4: Commit if any fixes were needed**

```bash
git add -p
git commit -m "fix(unifi): update collector_manager for new UnifiSystemConfig fields"
```

---

### Task 6: Update config.yaml (if present)

**Files:**
- Read: `backend/config.yaml`

**Step 1: Check if config.yaml has a unifi entry**

```bash
grep -A 10 "type: unifi" backend/config.yaml 2>/dev/null || echo "no unifi entry"
```

**Step 2: If present, replace username/password with api_key**

Change:
```yaml
config:
  host: 192.168.1.1
  username: admin
  password: secret
  site: default
```
To:
```yaml
config:
  host: 192.168.1.1
  api_key: "REPLACE_WITH_YOUR_API_KEY"
  site: default
```

**Step 3: Commit**

```bash
git add backend/config.yaml
git commit -m "chore: update config.yaml unifi entry to use api_key"
```

---

## Field Name Caveat

The UniFi Network API v1 field names (`state`, `isGuest`, `rxBytesPerSecond`, etc.) are based on documented patterns but **must be validated against your actual controller**. If fields don't match, add a `print(devices_data[0])` temporarily in `collect()` to inspect live response shapes, then update the `.get()` keys and the test fixtures to match.
