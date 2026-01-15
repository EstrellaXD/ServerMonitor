import pytest
from pydantic import ValidationError

from app.config import UnifiSystemConfig


def test_config_accepts_api_key():
    config = UnifiSystemConfig(host="192.168.1.1", api_key="test-key")
    assert config.api_key == "test-key"


def test_config_rejects_username_password():
    with pytest.raises(ValidationError):
        UnifiSystemConfig(host="192.168.1.1", api_key="key", username="admin", password="secret")


def test_config_defaults():
    config = UnifiSystemConfig(host="192.168.1.1", api_key="key")
    assert config.port == 443
    assert config.site == "default"
    assert config.verify_ssl is False
