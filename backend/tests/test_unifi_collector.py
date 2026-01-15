import pytest
import respx
from httpx import Response
from app.collectors.unifi import UnifiCollector, _parse_wan_rates
from app.config import UnifiSystemConfig
from app.models.metrics import SystemStatus, UnifiMetrics, UnifiDevice


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


# --- _parse_wan_rates ---

def test_parse_wan_rates_found():
    data = [{"subsystem": "wan", "rxBytesPerSecond": 10_000_000, "txBytesPerSecond": 5_000_000}]
    assert _parse_wan_rates(data) == (10.0, 5.0)


def test_parse_wan_rates_missing_subsystem():
    assert _parse_wan_rates([]) == (0.0, 0.0)


def test_parse_wan_rates_none_values():
    data = [{"subsystem": "wan", "rxBytesPerSecond": None, "txBytesPerSecond": None}]
    assert _parse_wan_rates(data) == (0.0, 0.0)


def test_parse_wan_rates_non_wan_subsystem():
    data = [{"subsystem": "wlan", "rxBytesPerSecond": 999_999}]
    assert _parse_wan_rates(data) == (0.0, 0.0)


# --- determine_status ---

def test_determine_status_all_online(collector):
    m = UnifiMetrics(devices=[UnifiDevice(name="sw", mac="aa:bb:cc:dd:ee:ff", model="USW", status="online")])
    assert collector.determine_status(m) == SystemStatus.HEALTHY


def test_determine_status_one_offline_warning(collector):
    devices = [
        UnifiDevice(name="d1", mac="aa:bb:cc:dd:ee:01", model="USW", status="online"),
        UnifiDevice(name="d2", mac="aa:bb:cc:dd:ee:02", model="USW", status="offline"),
    ]
    m = UnifiMetrics(devices=devices)
    assert collector.determine_status(m) == SystemStatus.WARNING


def test_determine_status_majority_offline_critical(collector):
    devices = [
        UnifiDevice(name=f"d{i}", mac=f"aa:bb:cc:dd:ee:{i:02x}", model="USW", status="offline")
        for i in range(3)
    ]
    m = UnifiMetrics(devices=devices)
    assert collector.determine_status(m) == SystemStatus.CRITICAL


def test_determine_status_empty_devices(collector):
    assert collector.determine_status(UnifiMetrics()) == SystemStatus.HEALTHY


# --- collect() with mocked HTTP ---

DEVICES_RESPONSE = {
    "data": [
        {"mac": "aa:bb:cc:dd:ee:ff", "name": "Living Room AP", "model": "U6-Lite", "state": "connected", "uptime": 86400},
        {"mac": "11:22:33:44:55:66", "name": "Office AP", "model": "U6-Pro", "state": "disconnected", "uptime": 0},
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

BASE = "https://192.168.1.1:443/proxy/network/integration/v1/sites/default"


@pytest.mark.asyncio
@respx.mock
async def test_collect_parses_devices(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert len(metrics.devices) == 2
    assert metrics.devices[0].name == "Living Room AP"
    assert metrics.devices[0].status == "online"
    assert metrics.devices[1].status == "offline"


@pytest.mark.asyncio
@respx.mock
async def test_collect_counts_clients(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert metrics.clients_count == 2
    assert metrics.guests_count == 1


@pytest.mark.asyncio
@respx.mock
async def test_collect_parses_wan_rates(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()
    metrics: UnifiMetrics = result.metrics

    assert metrics.wan_download == 5.0
    assert metrics.wan_upload == 1.0


@pytest.mark.asyncio
@respx.mock
async def test_collect_returns_error_on_http_failure(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(401))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()

    assert result.error is not None
    assert result.status.value == "offline"


@pytest.mark.asyncio
@respx.mock
async def test_collect_status_warning_one_device_offline(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()

    assert result.status == SystemStatus.WARNING


@pytest.mark.asyncio
@respx.mock
async def test_collect_returns_error_on_clients_failure(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(401))
    respx.get(f"{BASE}/health").mock(return_value=Response(200, json=HEALTH_RESPONSE))

    result = await collector.collect()

    assert result.error is not None
    assert result.status.value == "offline"


@pytest.mark.asyncio
@respx.mock
async def test_collect_returns_error_on_health_failure(collector):
    respx.get(f"{BASE}/devices").mock(return_value=Response(200, json=DEVICES_RESPONSE))
    respx.get(f"{BASE}/clients").mock(return_value=Response(200, json=CLIENTS_RESPONSE))
    respx.get(f"{BASE}/health").mock(return_value=Response(401))

    result = await collector.collect()

    assert result.error is not None
    assert result.status.value == "offline"
