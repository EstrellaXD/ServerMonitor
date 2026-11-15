package config

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type SystemConfig struct {
	ID      string
	Name    string
	Type    string
	Enabled bool
	Config  any
	// Controls lists the control groups enabled for this system. Never nil.
	Controls []string
}

type AppConfig struct {
	PollInterval int
	Systems      []SystemConfig
	// AllowedHosts are extra host names (lower-case) the HTTP server
	// accepts in the Host header, besides IP literals, localhost and *.local.
	AllowedHosts []string
}

type rawSystem struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Type     string    `yaml:"type"`
	Enabled  *bool     `yaml:"enabled"`
	Config   yaml.Node `yaml:"config"`
	Controls []string  `yaml:"controls"`
}

// controlsByType is the set of control values each system type accepts.
var controlsByType = map[string][]string{
	"docker":      {"container", "recreate"},
	"qbittorrent": {"torrent", "delete_files"},
	"linux":       {"power", "process", "service", "fan"},
	"unas":        {"power"},
	"unifi":       {"device", "client"},
}

// AllControls returns every control value valid for sysType.
func AllControls(sysType string) []string {
	return slices.Clone(controlsByType[sysType])
}

func validateControls(sysType string, controls []string) error {
	allowed := controlsByType[sysType]
	for _, c := range controls {
		if !slices.Contains(allowed, c) {
			return fmt.Errorf("unknown control %q for type %s (allowed: %s)", c, sysType, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// unitPattern is the shape of a systemd unit name. Unit names are
// interpolated into SSH commands, so anything else is rejected at load.
var unitPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9@._:-]*$`)

func validateServices(cfg any) error {
	lc, ok := cfg.(*LinuxSystemConfig)
	if !ok {
		return nil
	}
	for _, u := range lc.Services {
		if !unitPattern.MatchString(u) {
			return fmt.Errorf("invalid service unit name %q", u)
		}
	}
	return nil
}

type rawApp struct {
	PollInterval *int        `yaml:"poll_interval"`
	AllowedHosts []string    `yaml:"allowed_hosts"`
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

// Load reads the config file. A missing file yields defaults; an unparsable
// file or an invalid controls or services entry is an error.
func Load(path string) (AppConfig, error) {
	defaults := AppConfig{PollInterval: 5}

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Warn("config file not found, using defaults", "path", path, "err", err)
		return defaults, nil
	}

	var raw rawApp
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return AppConfig{}, fmt.Errorf("parse %s: %w", path, err)
	}

	app := AppConfig{PollInterval: 5}
	for _, h := range raw.AllowedHosts {
		app.AllowedHosts = append(app.AllowedHosts, strings.ToLower(h))
	}
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
		if err := validateControls(rs.Type, rs.Controls); err != nil {
			return AppConfig{}, fmt.Errorf("system %q: %w", rs.ID, err)
		}
		if err := validateServices(cfg); err != nil {
			return AppConfig{}, fmt.Errorf("system %q: %w", rs.ID, err)
		}
		app.Systems = append(app.Systems, SystemConfig{
			ID:       rs.ID,
			Name:     rs.Name,
			Type:     rs.Type,
			Enabled:  enabled,
			Config:   cfg,
			Controls: append([]string{}, rs.Controls...),
		})
	}
	return app, nil
}

// Validate is a noop today; future room for schema checks.
func (a AppConfig) Validate() error {
	if a.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be > 0")
	}
	return nil
}
