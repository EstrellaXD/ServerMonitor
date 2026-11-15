package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"slices"
	"strconv"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

// ActionRequest is the body of POST /api/systems/{id}/actions.
type ActionRequest struct {
	Action string          `json:"action"`
	Target string          `json:"target"`
	Params json.RawMessage `json:"params,omitempty"`
}

// ActionParams holds every optional parameter an action may take.
type ActionParams struct {
	DeleteFiles bool     `json:"delete_files"`
	Signal      string   `json:"signal"`
	Percent     *float64 `json:"percent"`
}

// Action is a request that passed the allowlist and validation. Only an
// Action reaches an Actor.
type Action struct {
	Name   string
	Target string
	Params ActionParams
	PID    int // linux kill only
	// ProcName is the process name for PID in the latest metrics (linux
	// kill only). The actor checks it again before the kill.
	ProcName string
}

// ActionError is a request error: Code is the HTTP status (400, 403 or 409).
type ActionError struct {
	Code int
	Msg  string
}

func (e *ActionError) Error() string { return e.Msg }

func invalid(format string, a ...any) error {
	return &ActionError{Code: http.StatusBadRequest, Msg: fmt.Sprintf(format, a...)}
}

func conflict(format string, a ...any) error {
	return &ActionError{Code: http.StatusConflict, Msg: fmt.Sprintf(format, a...)}
}

func forbidden(action, control string) error {
	return &ActionError{
		Code: http.StatusForbidden,
		Msg:  fmt.Sprintf("action '%s' requires control '%s' (not enabled for this system)", action, control),
	}
}

// Actor runs validated actions against one system.
type Actor interface {
	Run(ctx context.Context, a Action) (output string, err error)
}

// LogReader reads recent container logs (docker systems only).
type LogReader interface {
	Logs(ctx context.Context, container string, tail int) ([]models.LogLine, error)
}

type targetKind int

const (
	targetNone targetKind = iota
	targetContainer
	targetHash
	targetMAC
	targetPID
	targetUnit
	targetFan
)

type actionSpec struct {
	control string
	target  targetKind
}

var actionSpecs = map[string]map[string]actionSpec{
	"docker": {
		"start":    {"container", targetContainer},
		"stop":     {"container", targetContainer},
		"restart":  {"container", targetContainer},
		"pause":    {"container", targetContainer},
		"unpause":  {"container", targetContainer},
		"kill":     {"container", targetContainer},
		"recreate": {"recreate", targetContainer},
	},
	"qbittorrent": {
		"resume":        {"torrent", targetHash},
		"pause":         {"torrent", targetHash},
		"recheck":       {"torrent", targetHash},
		"top_prio":      {"torrent", targetHash},
		"bottom_prio":   {"torrent", targetHash},
		"increase_prio": {"torrent", targetHash},
		"decrease_prio": {"torrent", targetHash},
		"delete":        {"torrent", targetHash},
	},
	"linux": {
		"reboot":          {"power", targetNone},
		"poweroff":        {"power", targetNone},
		"kill":            {"process", targetPID},
		"service_start":   {"service", targetUnit},
		"service_stop":    {"service", targetUnit},
		"service_restart": {"service", targetUnit},
		"fan_set":         {"fan", targetFan},
		"fan_auto":        {"fan", targetFan},
	},
	"unas": {
		"reboot":   {"power", targetNone},
		"poweroff": {"power", targetNone},
	},
	"unifi": {
		"restart_device": {"device", targetMAC},
		"kick":           {"client", targetMAC},
		"block":          {"client", targetMAC},
		"unblock":        {"client", targetMAC},
	},
}

var (
	containerNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)
	torrentHashPattern   = regexp.MustCompile(`^[0-9a-f]{40}$|^[0-9a-f]{64}$`)
	macPattern           = regexp.MustCompile(`^([0-9a-f]{2}:){5}[0-9a-f]{2}$`)
)

// ValidContainerName reports whether name has the shape of a docker
// container name.
func ValidContainerName(name string) bool { return containerNamePattern.MatchString(name) }

// ValidateAction enforces the per-system control allowlist and validates the
// target and params. latest is the last stored metrics for the system (may
// be nil); it is the source of valid PIDs and fan labels. The returned error
// is an *ActionError.
func ValidateAction(sys config.SystemConfig, latest *models.SystemMetrics, req ActionRequest) (Action, error) {
	spec, ok := actionSpecs[sys.Type][req.Action]
	if !ok {
		return Action{}, invalid("unknown action %q for %s system", req.Action, sys.Type)
	}
	if !slices.Contains(sys.Controls, spec.control) {
		return Action{}, forbidden(req.Action, spec.control)
	}

	a := Action{Name: req.Action, Target: req.Target}
	if len(req.Params) > 0 && string(req.Params) != "null" {
		if err := json.Unmarshal(req.Params, &a.Params); err != nil {
			return Action{}, invalid("invalid params: %v", err)
		}
	}

	switch {
	case sys.Type == "qbittorrent" && a.Name == "delete" && a.Params.DeleteFiles:
		if !slices.Contains(sys.Controls, "delete_files") {
			return Action{}, forbidden(a.Name, "delete_files")
		}
	case sys.Type == "linux" && a.Name == "kill":
		if a.Params.Signal == "" {
			a.Params.Signal = "TERM"
		}
		if a.Params.Signal != "TERM" && a.Params.Signal != "KILL" {
			return Action{}, invalid("signal must be TERM or KILL, got %q", a.Params.Signal)
		}
	case sys.Type == "linux" && a.Name == "fan_set":
		p := a.Params.Percent
		if p == nil {
			return Action{}, invalid("params.percent is required")
		}
		if math.IsNaN(*p) || *p < 0 || *p > 100 {
			return Action{}, invalid("percent must be between 0 and 100, got %v", *p)
		}
	}

	if err := validateTarget(spec.target, &a, sys, latest); err != nil {
		return Action{}, err
	}
	return a, nil
}

func validateTarget(kind targetKind, a *Action, sys config.SystemConfig, latest *models.SystemMetrics) error {
	t := a.Target
	switch kind {
	case targetNone:
		if t != "" {
			return invalid("action %q takes no target", a.Name)
		}
	case targetContainer:
		if !ValidContainerName(t) {
			return invalid("invalid container name %q", t)
		}
	case targetHash:
		if !torrentHashPattern.MatchString(t) {
			return invalid("invalid torrent hash %q", t)
		}
	case targetMAC:
		if !macPattern.MatchString(t) {
			return invalid("invalid MAC address %q (expected lower-case aa:bb:cc:dd:ee:ff)", t)
		}
	case targetPID:
		pid, err := strconv.Atoi(t)
		if err != nil || pid <= 1 {
			return invalid("invalid pid %q", t)
		}
		lm, _ := latestLinux(latest)
		i := slices.IndexFunc(lm.Processes, func(p models.ProcessMetrics) bool { return p.PID == pid })
		if i < 0 {
			return invalid("pid %d is not in the latest process list", pid)
		}
		a.PID = pid
		a.ProcName = lm.Processes[i].Name
	case targetUnit:
		lc, _ := sys.Config.(*config.LinuxSystemConfig)
		if lc == nil || !slices.Contains(lc.Services, t) {
			return invalid("service %q is not configured for this system", t)
		}
	case targetFan:
		lm, _ := latestLinux(latest)
		// Mode is set only for fans of the controllable hwmon device.
		if !slices.ContainsFunc(lm.Fans, func(f models.FanMetrics) bool { return f.Label == t && f.Mode != "" }) {
			return invalid("fan %q is not a controllable fan of this system", t)
		}
	}
	return nil
}

func latestLinux(m *models.SystemMetrics) (models.LinuxMetrics, bool) {
	if m == nil {
		return models.LinuxMetrics{}, false
	}
	lm, ok := m.Metrics.(models.LinuxMetrics)
	return lm, ok
}

// NewActor returns the Actor for a system. In mock mode every system gets a
// mock actor that only simulates the action.
func NewActor(sys config.SystemConfig, mockMode bool) Actor {
	if mockMode {
		return mockActor{}
	}
	switch cfg := sys.Config.(type) {
	case *config.LinuxSystemConfig:
		return &sshActor{conn: NewSSHConnection(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.KeyPath), username: cfg.Username}
	case *config.UNASSystemConfig:
		return &sshActor{conn: NewSSHConnection(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.KeyPath), username: cfg.Username}
	case *config.DockerSystemConfig:
		return &dockerActor{cfg: cfg}
	case *config.QBittorrentSystemConfig:
		return &qbitActor{cfg: cfg}
	case *config.UnifiSystemConfig:
		return &unifiActor{col: NewUnifi(sys.ID, sys.Name, cfg)}
	}
	return nil
}
