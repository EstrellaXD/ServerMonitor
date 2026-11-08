package config

import (
	"os"
	"strconv"
)

type Settings struct {
	ConfigPath string
	Host       string
	Port       int
	Debug      bool
	MockMode   bool
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "true" || v == "1"
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func LoadSettings() Settings {
	return Settings{
		ConfigPath: envStr("SERVERMONITOR_CONFIG_PATH", "config.yaml"),
		Host:       envStr("SERVERMONITOR_HOST", "0.0.0.0"),
		Port:       envInt("SERVERMONITOR_PORT", 8742),
		Debug:      envBool("SERVERMONITOR_DEBUG"),
		MockMode:   envBool("SERVERMONITOR_MOCK_MODE"),
	}
}
