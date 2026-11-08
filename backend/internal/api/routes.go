package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/broadcast"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/manager"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/store"
)

type AppState struct {
	Store    *store.MetricsStore
	Manager  *manager.Manager
	Settings config.Settings
	Bus      *broadcast.Broadcaster

	systemConfigsMu sync.RWMutex
	systemConfigs   map[string]any
}

func NewAppState(s *store.MetricsStore, m *manager.Manager, settings config.Settings, bus *broadcast.Broadcaster, initial map[string]any) *AppState {
	return &AppState{
		Store:         s,
		Manager:       m,
		Settings:      settings,
		Bus:           bus,
		systemConfigs: initial,
	}
}

func (s *AppState) SystemConfig(id string) (any, bool) {
	s.systemConfigsMu.RLock()
	defer s.systemConfigsMu.RUnlock()
	v, ok := s.systemConfigs[id]
	return v, ok
}

func (s *AppState) ReplaceSystemConfigs(next map[string]any) {
	s.systemConfigsMu.Lock()
	s.systemConfigs = next
	s.systemConfigsMu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("write response", "err", err)
	}
}

func errorJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func Routes(state *AppState, staticDir string) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)

	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Get("/api/systems", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, state.Store.GetAll())
	})

	r.Get("/api/systems/{system_id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "system_id")
		if m, ok := state.Store.Get(id); ok {
			writeJSON(w, http.StatusOK, m)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"error": "System not found"})
	})

	r.Post("/api/reload", reloadHandler(state))
	r.Post("/api/systems/{system_id}/actions/docker", dockerActionHandler(state))
	r.Post("/api/systems/{system_id}/actions/qbittorrent", qbitActionHandler(state))

	r.Get("/ws", websocketHandler(state))

	if staticDir != "" {
		r.Handle("/*", spaFileServer(staticDir))
	}

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// spaFileServer serves staticDir and falls back to index.html for unknown
// paths so client-side routes work.
func spaFileServer(staticDir string) http.Handler {
	root := http.Dir(staticDir)
	files := http.FileServer(root)
	indexPath := fmt.Sprintf("%s/index.html", staticDir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check existence up front. Letting FileServer 404 first would leave
		// its text/plain Content-Type on the shared header map.
		// http.Dir.Open cleans the path, so ".." cannot escape staticDir.
		f, err := root.Open(r.URL.Path)
		if err != nil {
			http.ServeFile(w, r, indexPath)
			return
		}
		f.Close()
		files.ServeHTTP(w, r)
	})
}

func reloadHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		cfg := config.Load(state.Settings.ConfigPath)

		state.Manager.Stop()
		state.Store.Clear()
		state.Manager.LoadConfig(cfg)
		state.ReplaceSystemConfigs(cfg.EnabledSystemConfigs())
		state.Manager.Start(context.Background())

		ids := state.Manager.IDs()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": fmt.Sprintf("Config reloaded successfully. Monitoring %d systems", len(ids)),
			"systems": ids,
		})
	}
}
