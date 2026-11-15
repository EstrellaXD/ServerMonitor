package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_parsesControlsAndServices(t *testing.T) {
	path := writeConfig(t, `
systems:
  - id: pi
    type: linux
    controls: [power, process, service, fan]
    config:
      host: 10.0.0.1
      username: u
      services: [servermonitor, nginx.service]
  - id: dock
    type: docker
    config: {}
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Systems) != 2 {
		t.Fatalf("systems: %+v", cfg.Systems)
	}
	pi := cfg.Systems[0]
	if !slices.Equal(pi.Controls, []string{"power", "process", "service", "fan"}) {
		t.Fatalf("controls: %v", pi.Controls)
	}
	if lc := pi.Config.(*LinuxSystemConfig); !slices.Equal(lc.Services, []string{"servermonitor", "nginx.service"}) {
		t.Fatalf("services: %v", lc.Services)
	}
	// Omitted controls means none, and the slice is never nil.
	if dock := cfg.Systems[1]; dock.Controls == nil || len(dock.Controls) != 0 {
		t.Fatalf("docker controls: %#v", dock.Controls)
	}
}

func TestLoad_rejectsControlUnknownForType(t *testing.T) {
	// "power" is valid for linux but not for docker.
	path := writeConfig(t, `
systems:
  - id: dock
    type: docker
    controls: [container, power]
    config: {}
`)
	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), `unknown control "power"`) || !strings.Contains(err.Error(), `"dock"`) {
		t.Fatalf("err: %v", err)
	}
}

func TestLoad_rejectsServiceNameWithShellCharacters(t *testing.T) {
	path := writeConfig(t, `
systems:
  - id: pi
    type: linux
    config:
      host: 10.0.0.1
      services: ["x; reboot"]
`)
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "invalid service unit name") {
		t.Fatalf("err: %v", err)
	}
}

func TestLoad_unparsableFileIsAnErrorAndMissingFileYieldsDefaults(t *testing.T) {
	if _, err := Load(writeConfig(t, "systems: [\n  - id: x\n    type: : bad")); err == nil {
		t.Fatal("expected a parse error")
	}
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil || cfg.PollInterval != 5 || len(cfg.Systems) != 0 {
		t.Fatalf("missing file: cfg=%+v err=%v", cfg, err)
	}
}

func TestLoad_lowercasesAllowedHosts(t *testing.T) {
	cfg, err := Load(writeConfig(t, "allowed_hosts: [Monitor.Example.org]\nsystems: []\n"))
	if err != nil || !slices.Equal(cfg.AllowedHosts, []string{"monitor.example.org"}) {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}
