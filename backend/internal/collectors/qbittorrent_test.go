package collectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
)

func TestAuthenticateQBittorrent_loginResponses(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		ok     bool
	}{
		{"legacy ok", http.StatusOK, "Ok.", true},
		{"5.1+ no content", http.StatusNoContent, "", true},
		{"legacy fails", http.StatusOK, "Fails.", false},
		{"unauthorized", http.StatusUnauthorized, "", false},
		{"banned", http.StatusForbidden, "Your IP address has been banned", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			err := AuthenticateQBittorrent(context.Background(), srv.Client(), &config.QBittorrentSystemConfig{URL: srv.URL})
			if (err == nil) != tc.ok {
				t.Fatalf("ok=%v, err=%v", tc.ok, err)
			}
		})
	}
}

func TestQbitActorRun_resumeUsesStartAndFallsBackToLegacyOn404(t *testing.T) {
	hash := "0123456789abcdef0123456789abcdef01234567"
	cases := []struct {
		name     string
		hasStart bool
		want     []string
	}{
		{"5.0+ start", true, []string{"/api/v2/torrents/start"}},
		{"legacy resume", false, []string{"/api/v2/torrents/start", "/api/v2/torrents/resume"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v2/auth/login":
					w.WriteHeader(http.StatusNoContent)
					return
				case "/api/v2/torrents/start":
					got = append(got, r.URL.Path)
					if !tc.hasStart {
						w.WriteHeader(http.StatusNotFound)
					}
				case "/api/v2/torrents/resume":
					got = append(got, r.URL.Path)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
				if err := r.ParseForm(); err != nil || r.PostForm.Get("hashes") != hash {
					t.Errorf("hashes=%q err=%v", r.PostForm.Get("hashes"), err)
				}
			}))
			defer srv.Close()

			q := &qbitActor{cfg: &config.QBittorrentSystemConfig{URL: srv.URL}}
			if _, err := q.Run(context.Background(), Action{Name: "resume", Target: hash}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("calls=%v, want %v", got, tc.want)
			}
		})
	}
}
