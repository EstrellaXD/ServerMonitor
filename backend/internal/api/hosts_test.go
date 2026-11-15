package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/broadcast"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/manager"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/store"
)

func newHostTestRouter(t *testing.T, allowed []string) http.Handler {
	t.Helper()
	st := store.New()
	bus := broadcast.New()
	mgr := manager.New(st, bus, false)
	mgr.LoadConfig(config.AppConfig{PollInterval: 5, AllowedHosts: allowed})
	return Routes(NewAppState(st, mgr, config.Settings{}, bus), "")
}

func TestHostAllowlist_allowsLocalNamesAndConfiguredHostsAndRejectsOthers(t *testing.T) {
	h := newHostTestRouter(t, []string{"monitor.example.org"})
	cases := []struct {
		host string
		want int
	}{
		{"192.168.31.170:8742", 200},
		{"192.168.31.170", 200},
		{"[::1]:8742", 200},
		{"[fe80::1]", 200},
		{"localhost:4829", 200},
		{"LOCALHOST", 200},
		{"raspberrypi.local:8742", 200},
		{"Monitor.Example.org", 200},
		{"evil.example", 403},
		{"evil.example:8742", 403},
		{"monitor.example.org.evil.example", 403},
		{"local", 403},
		{"", 403},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			req.Host = tc.host
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status %d, want %d", rec.Code, tc.want)
			}
			if tc.want == 403 && decodeBody(t, rec)["error"] != "host not allowed" {
				t.Fatalf("body %s", rec.Body.String())
			}
		})
	}
}

func TestOriginAllowed_rules(t *testing.T) {
	allowed := []string{"monitor.example.org"}
	cases := map[string]bool{
		"":                              true, // native client
		"http://localhost:4829":         true, // vite dev server
		"http://192.168.31.170:8742":    true,
		"http://[::1]:8742":             true,
		"http://raspberrypi.local":      true,
		"https://monitor.example.org":   true,
		"http://evil.example":           false,
		"http://evil.example:8742":      false,
		"null":                          false,
		"http://localhost.evil.example": false,
	}
	for origin, want := range cases {
		if got := originAllowed(origin, allowed); got != want {
			t.Fatalf("originAllowed(%q) = %v, want %v", origin, got, want)
		}
	}
}

func TestWebsocket_checksOrigin(t *testing.T) {
	srv := httptest.NewServer(newHostTestRouter(t, nil))
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	for _, origin := range []string{"", "http://localhost:4829"} {
		hdr := http.Header{}
		if origin != "" {
			hdr.Set("Origin", origin)
		}
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, hdr)
		if err != nil {
			t.Fatalf("origin %q: %v", origin, err)
		}
		conn.Close()
	}

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": {"http://evil.example"}})
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site origin: err=%v resp=%v", err, resp)
	}
}
