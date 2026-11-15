package collectors

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

const testHash = "0123456789abcdef0123456789abcdef01234567"

func linuxSystem(controls ...string) config.SystemConfig {
	return config.SystemConfig{
		ID: "pi", Type: "linux", Controls: controls,
		Config: &config.LinuxSystemConfig{Username: "monitor", Services: []string{"servermonitor"}},
	}
}

func linuxLatest() *models.SystemMetrics {
	return &models.SystemMetrics{Metrics: models.LinuxMetrics{
		Processes: []models.ProcessMetrics{{PID: 2978, Name: "qbittorrent-nox"}, {PID: 1, Name: "systemd"}},
		Fans: []models.FanMetrics{
			{Label: "pwmfan/fan1", Mode: "auto"},
			{Label: "nct6775/fan2"},
		},
	}}
}

func req(action, target, params string) ActionRequest {
	r := ActionRequest{Action: action, Target: target}
	if params != "" {
		r.Params = json.RawMessage(params)
	}
	return r
}

func errCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return http.StatusOK
	}
	var ae *ActionError
	if !errors.As(err, &ae) {
		t.Fatalf("not an ActionError: %v", err)
	}
	return ae.Code
}

func TestValidateAction_enforcesControlAllowlist(t *testing.T) {
	cases := []struct {
		name string
		sys  config.SystemConfig
		req  ActionRequest
		want int
	}{
		{"linux power not enabled", linuxSystem("process"), req("reboot", "", ""), 403},
		{"linux power enabled", linuxSystem("power"), req("reboot", "", ""), 200},
		{"docker recreate needs recreate", config.SystemConfig{Type: "docker", Controls: []string{"container"}}, req("recreate", "nginx", ""), 403},
		{"docker start with container", config.SystemConfig{Type: "docker", Controls: []string{"container"}}, req("start", "nginx", ""), 200},
		{"qbit delete files needs delete_files", config.SystemConfig{Type: "qbittorrent", Controls: []string{"torrent"}}, req("delete", testHash, `{"delete_files":true}`), 403},
		{"qbit delete keeps files", config.SystemConfig{Type: "qbittorrent", Controls: []string{"torrent"}}, req("delete", testHash, `{"delete_files":false}`), 200},
		{"unifi kick without client", config.SystemConfig{Type: "unifi", Controls: []string{"device"}}, req("kick", "aa:bb:cc:dd:ee:ff", ""), 403},
		{"no controls at all", config.SystemConfig{Type: "unas", Controls: []string{}}, req("poweroff", "", ""), 403},
		{"action of another type", config.SystemConfig{Type: "unas", Controls: []string{"power"}}, req("kill", "2978", ""), 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateAction(tc.sys, linuxLatest(), tc.req)
			if got := errCode(t, err); got != tc.want {
				t.Fatalf("got %d (%v), want %d", got, err, tc.want)
			}
		})
	}
}

func TestValidateAction_forbiddenMessageNamesControl(t *testing.T) {
	_, err := ValidateAction(linuxSystem(), nil, req("reboot", "", ""))
	want := "action 'reboot' requires control 'power' (not enabled for this system)"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v", err)
	}
}

func TestValidateAction_validatesTargetsAndParams(t *testing.T) {
	all := linuxSystem("power", "process", "service", "fan")
	docker := config.SystemConfig{Type: "docker", Controls: []string{"container"}}
	qbit := config.SystemConfig{Type: "qbittorrent", Controls: []string{"torrent"}}
	unifi := config.SystemConfig{Type: "unifi", Controls: []string{"client"}}
	cases := []struct {
		name string
		sys  config.SystemConfig
		req  ActionRequest
		want int
	}{
		{"pid in process list", all, req("kill", "2978", ""), 200},
		{"pid not in process list", all, req("kill", "4242", ""), 400},
		{"pid 1", all, req("kill", "1", ""), 400},
		{"pid not a number", all, req("kill", "29; reboot", ""), 400},
		{"kill signal KILL", all, req("kill", "2978", `{"signal":"KILL"}`), 200},
		{"kill signal HUP", all, req("kill", "2978", `{"signal":"HUP"}`), 400},
		{"configured unit", all, req("service_restart", "servermonitor", ""), 200},
		{"unconfigured unit", all, req("service_stop", "sshd", ""), 400},
		{"power with target", all, req("reboot", "now", ""), 400},
		{"fan_set controllable", all, req("fan_set", "pwmfan/fan1", `{"percent":60}`), 200},
		{"fan_set other device", all, req("fan_set", "nct6775/fan2", `{"percent":60}`), 400},
		{"fan_set missing percent", all, req("fan_set", "pwmfan/fan1", ""), 400},
		{"fan_set percent above 100", all, req("fan_set", "pwmfan/fan1", `{"percent":101}`), 400},
		{"fan_auto", all, req("fan_auto", "pwmfan/fan1", ""), 200},
		{"params wrong type", all, req("fan_set", "pwmfan/fan1", `{"percent":"high"}`), 400},
		{"container name", docker, req("restart", "my_app.1-x", ""), 200},
		{"container name with shell chars", docker, req("restart", "app;rm", ""), 400},
		{"container name leading dash", docker, req("restart", "-app", ""), 400},
		{"hash 40", qbit, req("pause", testHash, ""), 200},
		{"hash 64", qbit, req("pause", strings.Repeat("ab", 32), ""), 200},
		{"hash upper case", qbit, req("pause", strings.ToUpper(testHash), ""), 400},
		{"hash 32", qbit, req("pause", testHash[:32], ""), 400},
		{"mac lower", unifi, req("block", "aa:bb:cc:dd:ee:ff", ""), 200},
		{"mac upper", unifi, req("block", "AA:BB:CC:DD:EE:FF", ""), 400},
		{"mac dashes", unifi, req("block", "aa-bb-cc-dd-ee-ff", ""), 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateAction(tc.sys, linuxLatest(), tc.req)
			if got := errCode(t, err); got != tc.want {
				t.Fatalf("got %d (%v), want %d", got, err, tc.want)
			}
		})
	}
}

func TestValidateAction_killWithoutLatestMetricsIsRejected(t *testing.T) {
	_, err := ValidateAction(linuxSystem("process"), nil, req("kill", "2978", ""))
	if errCode(t, err) != 400 {
		t.Fatalf("got %v", err)
	}
}

func TestValidateAction_killDefaultsToTERM(t *testing.T) {
	a, err := ValidateAction(linuxSystem("process"), linuxLatest(), req("kill", "2978", ""))
	if err != nil || a.Params.Signal != "TERM" || a.PID != 2978 || a.ProcName != "qbittorrent-nox" {
		t.Fatalf("got %+v, %v", a, err)
	}
}

func TestCheckProcComm_conflictWhenPIDChangedOrGone(t *testing.T) {
	a := Action{Name: "kill", PID: 2978, ProcName: "qbittorrent-nox"}
	if err := checkProcComm(a, "qbittorrent-nox\n"); err != nil {
		t.Fatalf("same process: %v", err)
	}
	for _, comm := range []string{"bash\n", ""} {
		err := checkProcComm(a, comm)
		if errCode(t, err) != http.StatusConflict || err.Error() != "process 2978 changed since the last poll; refresh and retry" {
			t.Fatalf("comm %q: %v", comm, err)
		}
	}
	if got := procCommCmd(2978); got != `sh -c 'cat /proc/2978/comm 2>/dev/null; true'` {
		t.Fatalf("procCommCmd: %s", got)
	}
}

func mustCommand(t *testing.T, a Action, user string) string {
	t.Helper()
	cmd, err := sshCommand(a, user)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cmd, `sh -c '`) || !strings.HasSuffix(cmd, `'`) {
		t.Fatalf("not wrapped in sh -c: %s", cmd)
	}
	// The script is single-quoted; a quote inside it would end the quoting.
	if inner := cmd[len(`sh -c '`) : len(cmd)-1]; strings.Contains(inner, `'`) {
		t.Fatalf("script contains a single quote: %s", cmd)
	}
	return cmd
}

func TestSSHCommand_power(t *testing.T) {
	cmd := mustCommand(t, Action{Name: "reboot"}, "monitor")
	for _, want := range []string{`S="sudo -n"`, `$S -l systemctl reboot`, `nohup sh -c "sleep 2; $S systemctl reboot"`} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("missing %q in %s", want, cmd)
		}
	}
	if cmd := mustCommand(t, Action{Name: "poweroff"}, "root"); !strings.Contains(cmd, `S=""`) || !strings.Contains(cmd, "systemctl poweroff") {
		t.Fatalf("root poweroff: %s", cmd)
	}
}

func TestSSHCommand_killAndService(t *testing.T) {
	got := mustCommand(t, Action{Name: "kill", PID: 2978, Params: ActionParams{Signal: "KILL"}}, "monitor")
	if got != `sh -c 'S="sudo -n"; $S kill -s KILL 2978'` {
		t.Fatalf("kill: %s", got)
	}
	got = mustCommand(t, Action{Name: "service_restart", Target: "servermonitor"}, "monitor")
	if got != `sh -c 'S="sudo -n"; $S systemctl restart servermonitor'` {
		t.Fatalf("service: %s", got)
	}
}

func TestSSHCommand_fanSetAndAuto(t *testing.T) {
	pct := 50.0
	set := mustCommand(t, Action{Name: "fan_set", Target: "pwmfan/fan1", Params: ActionParams{Percent: &pct}}, "monitor")
	for _, want := range []string{
		`= pwmfan ]`,
		`STATE=/run/servermonitor-fan.state`,
		`echo nvfancontrol | $S tee -a "$STATE"`,
		`$S systemctl stop nvfancontrol`,
		`= pwm-fan ]`,
		`echo user_space | $S tee "$z/policy"`,
		`echo "enable $HW/pwm1_enable $old" | $S tee -a "$STATE"`,
		`echo 1 | $S tee "$HW/pwm1_enable"`,
		`echo 128 | $S tee "$HW/pwm1"`,
	} {
		if !strings.Contains(set, want) {
			t.Fatalf("fan_set missing %q", want)
		}
	}
	// Changes are only recorded when made, so a second call keeps the record.
	if !strings.Contains(set, `[ "$old" = user_space ] && continue`) || !strings.Contains(set, `[ -e "$STATE" ] ||`) {
		t.Fatalf("fan_set is not idempotent: %s", set)
	}

	auto := mustCommand(t, Action{Name: "fan_auto", Target: "pwmfan/fan1"}, "monitor")
	for _, want := range []string{`${policy:-step_wise}`, `enable) echo "$policy" | $S tee "$path"`, `$S systemctl start nvfancontrol`, `$S rm -f "$STATE"`} {
		if !strings.Contains(auto, want) {
			t.Fatalf("fan_auto missing %q", want)
		}
	}
}

func TestPWMValue_roundsPercentTo255Scale(t *testing.T) {
	for pct, want := range map[float64]int{0: 0, 25: 64, 50: 128, 100: 255} {
		if got := pwmValue(pct); got != want {
			t.Fatalf("pwmValue(%v) = %d, want %d", pct, got, want)
		}
	}
}

func TestRecreateConfig_dropsImageDefaultsAndKeepsContainerSettings(t *testing.T) {
	id := "abcdef0123456789"
	img := &container.Config{
		Env:    []string{"PATH=/usr/bin", "NGINX_VERSION=1.25"},
		Labels: map[string]string{"maintainer": "nginx"},
		Cmd:    []string{"nginx", "-g", "daemon off;"},
	}
	cur := &container.Config{
		Hostname: "abcdef012345",
		Image:    "nginx:latest",
		Env:      []string{"PATH=/usr/bin", "NGINX_VERSION=1.25", "TZ=UTC"},
		Labels:   map[string]string{"maintainer": "nginx", "com.docker.compose.project": "web"},
		Cmd:      []string{"nginx", "-g", "daemon off;"},
	}
	got := recreateConfig(cur, img, id)
	if got.Image != "nginx:latest" || got.Hostname != "" || got.Cmd != nil {
		t.Fatalf("got %+v", got)
	}
	if len(got.Env) != 1 || got.Env[0] != "TZ=UTC" {
		t.Fatalf("env: %v", got.Env)
	}
	if len(got.Labels) != 1 || got.Labels["com.docker.compose.project"] != "web" {
		t.Fatalf("labels: %v", got.Labels)
	}
	if len(cur.Env) != 3 || len(cur.Labels) != 2 {
		t.Fatalf("input was mutated: %+v", cur)
	}
}

func TestRecreateHostConfig_bindsAnonymousVolumesOnly(t *testing.T) {
	cur := &container.HostConfig{
		Binds:  []string{"data:/data", "/srv/conf:/etc/app:ro"},
		Mounts: []mount.Mount{{Type: mount.TypeVolume, Source: "cache", Target: "/cache"}},
	}
	mounts := []types.MountPoint{
		{Type: mount.TypeVolume, Name: "data", Destination: "/data", RW: true},
		{Type: mount.TypeBind, Source: "/srv/conf", Destination: "/etc/app"},
		{Type: mount.TypeVolume, Name: "cache", Destination: "/cache", RW: true},
		{Type: mount.TypeVolume, Name: "3f9a", Destination: "/var/lib/postgresql/data", RW: true},
		{Type: mount.TypeVolume, Name: "77c1", Destination: "/seed"},
	}
	got := recreateHostConfig(cur, mounts)
	want := []string{"data:/data", "/srv/conf:/etc/app:ro", "3f9a:/var/lib/postgresql/data", "77c1:/seed:ro"}
	if strings.Join(got.Binds, ",") != strings.Join(want, ",") {
		t.Fatalf("binds: %v", got.Binds)
	}
	if len(cur.Binds) != 2 {
		t.Fatalf("input was mutated: %v", cur.Binds)
	}
}

func TestRecreateNetworks_createsOnModeNetworkAndConnectsRest(t *testing.T) {
	nets := map[string]*network.EndpointSettings{
		"web_default": {Aliases: []string{"web", "abcdef012345"}, IPAddress: "172.18.0.5", EndpointID: "e1"},
		"proxy":       {Aliases: []string{"web"}},
	}
	first, rest := recreateNetworks("web_default", nets, "abcdef012345")
	ep, ok := first.EndpointsConfig["web_default"]
	if !ok || len(first.EndpointsConfig) != 1 {
		t.Fatalf("first: %+v", first.EndpointsConfig)
	}
	if ep.IPAddress != "" || ep.EndpointID != "" || len(ep.Aliases) != 1 || ep.Aliases[0] != "web" {
		t.Fatalf("endpoint copied runtime data or old ID alias: %+v", ep)
	}
	if _, ok := rest["proxy"]; !ok || len(rest) != 1 {
		t.Fatalf("rest: %+v", rest)
	}
}

func TestLineWriter_keepsStreamOrderAcrossDemux(t *testing.T) {
	var buf bytes.Buffer
	out := stdcopy.NewStdWriter(&buf, stdcopy.Stdout)
	errw := stdcopy.NewStdWriter(&buf, stdcopy.Stderr)
	out.Write([]byte("one\ntw"))
	errw.Write([]byte("boom\n"))
	out.Write([]byte("o\r\n"))

	var lines []models.LogLine
	stdout := &lineWriter{stream: "stdout", out: &lines}
	stderr := &lineWriter{stream: "stderr", out: &lines}
	if _, err := stdcopy.StdCopy(stdout, stderr, &buf); err != nil {
		t.Fatal(err)
	}
	stdout.flush()
	stderr.flush()
	want := []models.LogLine{{Stream: "stdout", Text: "one"}, {Stream: "stderr", Text: "boom"}, {Stream: "stdout", Text: "two"}}
	if len(lines) != len(want) {
		t.Fatalf("got %+v", lines)
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d: got %+v, want %+v", i, lines[i], want[i])
		}
	}
}

func TestMockSystemConfig_enablesAllControlsAndAService(t *testing.T) {
	sys := MockSystemConfig(config.SystemConfig{Type: "linux", Config: &config.LinuxSystemConfig{}})
	if len(sys.Controls) != 4 {
		t.Fatalf("controls: %v", sys.Controls)
	}
	if lc := sys.Config.(*config.LinuxSystemConfig); len(lc.Services) != 1 {
		t.Fatalf("services: %v", lc.Services)
	}
}
