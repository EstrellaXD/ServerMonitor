package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/broadcast"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/manager"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/store"
)

// newTestRouter loads one linux and one docker system. The manager is not
// started, so no collector runs; the store holds hand-made metrics.
func newTestRouter(t *testing.T, mock bool, linuxControls []string) http.Handler {
	t.Helper()
	st := store.New()
	bus := broadcast.New()
	mgr := manager.New(st, bus, mock)
	mgr.LoadConfig(config.AppConfig{PollInterval: 5, Systems: []config.SystemConfig{
		{ID: "pi", Type: "linux", Enabled: true, Controls: linuxControls,
			Config: &config.LinuxSystemConfig{Host: "127.0.0.1", Port: 1, Username: "u", Services: []string{"servermonitor"}}},
		{ID: "dock", Type: "docker", Enabled: true, Controls: []string{},
			Config: &config.DockerSystemConfig{Socket: "unix:///nonexistent.sock"}},
	}})
	st.Update("pi", &models.SystemMetrics{ID: "pi", Type: models.TypeLinux, Metrics: models.LinuxMetrics{
		Processes: []models.ProcessMetrics{{PID: 2978, Name: "qbittorrent-nox"}},
	}})
	return Routes(NewAppState(st, mgr, config.Settings{MockMode: mock}, bus), "")
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	return postAs(h, path, body, "application/json")
}

func postAs(h http.Handler, path, body, contentType string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "localhost" // httptest's default example.com fails the Host allowlist
	req.Header.Set("Content-Type", contentType)
	h.ServeHTTP(rec, req)
	return rec
}

func TestActionHandler_rejectsCrossSiteRequestShapes(t *testing.T) {
	// Mock mode enables every control, so only the request shape can refuse.
	h := newTestRouter(t, true, nil)
	body := `{"action":"reboot","target":""}`
	for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
		if rec := postAs(h, "/api/systems/pi/actions", body, ct); rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("Content-Type %q: status %d", ct, rec.Code)
		}
	}
	if rec := postAs(h, "/api/systems/pi/actions", body, "application/json; charset=utf-8"); rec.Code != 200 {
		t.Fatalf("json with charset: status %d: %s", rec.Code, rec.Body.String())
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/systems/pi/actions", nil)
	req.Host = "localhost"
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "POST")
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("preflight allowed origin %q", got)
	}
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var out map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	return out
}

func TestActionHandler_statusCodes(t *testing.T) {
	h := newTestRouter(t, false, []string{"process"})
	cases := []struct {
		name, path, body string
		want             int
		wantErr          string
	}{
		{"unknown system", "/api/systems/nope/actions", `{"action":"reboot","target":""}`, 404, "system not found"},
		{"control not enabled", "/api/systems/pi/actions", `{"action":"reboot","target":""}`, 403,
			"action 'reboot' requires control 'power' (not enabled for this system)"},
		{"docker without controls", "/api/systems/dock/actions", `{"action":"start","target":"nginx"}`, 403,
			"action 'start' requires control 'container' (not enabled for this system)"},
		{"unknown action", "/api/systems/pi/actions", `{"action":"format","target":""}`, 400, `unknown action "format" for linux system`},
		{"pid not in latest metrics", "/api/systems/pi/actions", `{"action":"kill","target":"4242"}`, 400, "pid 4242 is not in the latest process list"},
		{"bad body", "/api/systems/pi/actions", `{`, 400, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := post(h, tc.path, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
			if tc.wantErr != "" && decodeBody(t, rec)["error"] != tc.wantErr {
				t.Fatalf("error %q", rec.Body.String())
			}
		})
	}
}

func TestActionHandler_mockModeRunsValidatedAction(t *testing.T) {
	// Mock mode enables every control, but validation still applies.
	h := newTestRouter(t, true, nil)
	rec := post(h, "/api/systems/pi/actions", `{"action":"kill","target":"2978","params":{"signal":"KILL"}}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	if body["status"] != "success" || body["action"] != "kill" || body["target"] != "2978" || body["output"] == "" {
		t.Fatalf("body %+v", body)
	}
	if rec := post(h, "/api/systems/pi/actions", `{"action":"service_stop","target":"sshd"}`); rec.Code != 400 {
		t.Fatalf("unconfigured unit: status %d", rec.Code)
	}
}

func TestParseLogTail_defaultsAndClamps(t *testing.T) {
	cases := map[string]int{"": 200, "50": 50, "0": 1, "-3": 1, "2000": 2000, "99999": 2000}
	for in, want := range cases {
		got, err := parseLogTail(in)
		if err != nil || got != want {
			t.Fatalf("parseLogTail(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseLogTail("ten"); err == nil {
		t.Fatal("non-numeric tail must be an error")
	}
}

func TestDockerLogsHandler_validatesAndReturnsLines(t *testing.T) {
	h := newTestRouter(t, true, nil)
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "localhost"
		h.ServeHTTP(rec, req)
		return rec
	}
	if rec := get("/api/systems/pi/docker/logs?container=nginx"); rec.Code != 400 {
		t.Fatalf("non-docker system: %d", rec.Code)
	}
	if rec := get("/api/systems/dock/docker/logs?container=../x"); rec.Code != 400 {
		t.Fatalf("bad container: %d", rec.Code)
	}
	rec := get("/api/systems/dock/docker/logs?container=nginx&tail=5000")
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var logs models.DockerLogs
	if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil {
		t.Fatal(err)
	}
	if logs.Container != "nginx" || len(logs.Lines) == 0 || len(logs.Lines) > maxLogTail {
		t.Fatalf("logs: %s %d lines", logs.Container, len(logs.Lines))
	}
}
