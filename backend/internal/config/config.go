package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type SystemConfig struct {
	ID       string
	Name     string
	Type     string
	Enabled  bool
	Config   any
}

type AppConfig struct {
	PollInterval int
	Systems      []SystemConfig
}

type rawSystem struct {
	ID      string    `yaml:"id"`
	Name    string    `yaml:"name"`
	Type    string    `yaml:"type"`
	Enabled *bool     `yaml:"enabled"`
	Config  yaml.Node `yaml:"config"`
}

type rawApp struct {
	PollInterval *int        `yaml:"poll_interval"`
	Systems      []rawSystem `yaml:"systems"`
}

func defaultLinuxPort(c *LinuxSystemConfig) {
	if c.Port == 0 {
		c.Port = 22
	}
}

func defaultDockerSocket(c *DockerSystemConfig) {
	if c.Socket == "" {
		c.Socket = "unix:///var/run/docker.sock"
	}
}

func defaultUnifi(c *UnifiSystemConfig) {
	if c.Port == 0 {
		c.Port = 443
	}
	if c.Site == "" {
		c.Site = "default"
	}
}

func defaultUNAS(c *UNASSystemConfig) {
	if c.Port == 0 {
		c.Port = 22
	}
}

func decodeSystemConfig(sysType string, node yaml.Node) any {
	switch sysType {
	case "linux":
		var c LinuxSystemConfig
		if err := node.Decode(&c); err != nil {
			return nil
		}
		defaultLinuxPort(&c)
		return &c
	case "docker":
		var c DockerSystemConfig
		if err := node.Decode(&c); err != nil {
			return nil
		}
		defaultDockerSocket(&c)
		return &c
	case "qbittorrent":
		var c QBittorrentSystemConfig
		if err := node.Decode(&c); err != nil {
			return nil
		}
		return &c
	case "unifi":
		var c UnifiSystemConfig
		if err := node.Decode(&c); err != nil {
			return nil
		}
		defaultUnifi(&c)
		return &c
	case "unas":
		var c UNASSystemConfig
		if err := node.Decode(&c); err != nil {
			return nil
		}
		defaultUNAS(&c)
		return &c
	}
	return nil
}

func Load(path string) AppConfig {
	defaults := AppConfig{PollInterval: 5}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("config file not found, using defaults", "path", path, "err", err)
		return defaults
	}

	var raw rawApp
	if err := yaml.Unmarshal(data, &raw); err != nil {
		slog.Error("config parse failed, using defaults", "err", err)
		return defaults
	}

	app := AppConfig{PollInterval: 5}
	if raw.PollInterval != nil && *raw.PollInterval > 0 {
		app.PollInterval = *raw.PollInterval
	}

	for _, rs := range raw.Systems {
		enabled := true
		if rs.Enabled != nil {
			enabled = *rs.Enabled
		}
		cfg := decodeSystemConfig(rs.Type, rs.Config)
		if cfg == nil {
			slog.Warn("skipping system with unknown or invalid type", "id", rs.ID, "type", rs.Type)
			continue
		}
		app.Systems = append(app.Systems, SystemConfig{
			ID:      rs.ID,
			Name:    rs.Name,
			Type:    rs.Type,
			Enabled: enabled,
			Config:  cfg,
		})
	}
	return app
}

// EnabledSystemConfigs returns a map of system-id → config payload for enabled systems.
// Used by action handlers to look up the live connection settings.
func (a AppConfig) EnabledSystemConfigs() map[string]any {
	out := make(map[string]any, len(a.Systems))
	for _, s := range a.Systems {
		if !s.Enabled {
			continue
		}
		out[s.ID] = s.Config
	}
	return out
}

// Validate is a noop today; future room for schema checks.
func (a AppConfig) Validate() error {
	if a.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be > 0")
	}
	return nil
}
