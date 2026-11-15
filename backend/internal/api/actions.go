package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/collectors"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

// actionTimeout bounds one action. A docker recreate pulls an image, so it
// is generous.
const actionTimeout = 10 * time.Minute

const (
	defaultLogTail = 200
	maxLogTail     = 2000
)

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

// actionHandler is the single entry point for control actions:
// allowlist → validation → the system's Actor.
func actionHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A cross-site page can send a text/plain POST without a preflight.
		// application/json needs a preflight, which fails without CORS headers.
		if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
			errorJSON(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
			return
		}
		systemID := chi.URLParam(r, "system_id")
		sys, ok := state.Manager.System(systemID)
		if !ok {
			errorJSON(w, http.StatusNotFound, "system not found")
			return
		}

		var req collectors.ActionRequest
		if err := decodeJSON(r, &req); err != nil {
			errorJSON(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		latest, _ := state.Store.Get(systemID)
		action, err := collectors.ValidateAction(sys, latest, req)
		if err != nil {
			var ae *collectors.ActionError
			if errors.As(err, &ae) {
				errorJSON(w, ae.Code, ae.Msg)
				return
			}
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		// The action must not stop half-way when the client goes away.
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), actionTimeout)
		defer cancel()
		started := time.Now()
		output, err := collectors.NewActor(sys, state.Settings.MockMode).Run(ctx, action)
		if err != nil {
			slog.Error("action failed", "system", systemID, "action", action.Name, "target", action.Target, "err", err)
			code := http.StatusInternalServerError
			var ae *collectors.ActionError
			if errors.As(err, &ae) {
				code = ae.Code
			}
			errorJSON(w, code, err.Error())
			return
		}
		slog.Info("action", "system", systemID, "action", action.Name, "target", action.Target, "took", time.Since(started))

		body := map[string]string{"status": "success", "action": action.Name, "target": action.Target}
		if output != "" {
			body["output"] = output
		}
		writeJSON(w, http.StatusOK, body)
	}
}

// parseLogTail reads the tail query value: default 200, clamped to 1..2000.
func parseLogTail(raw string) (int, error) {
	if raw == "" {
		return defaultLogTail, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid tail %q", raw)
	}
	return min(max(n, 1), maxLogTail), nil
}

// dockerLogsHandler returns recent container logs. It is read-only, so no
// control is required.
func dockerLogsHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		systemID := chi.URLParam(r, "system_id")
		sys, ok := state.Manager.System(systemID)
		if !ok {
			errorJSON(w, http.StatusNotFound, "system not found")
			return
		}
		if sys.Type != "docker" {
			errorJSON(w, http.StatusBadRequest, "logs are only available for docker systems")
			return
		}
		name := r.URL.Query().Get("container")
		if !collectors.ValidContainerName(name) {
			errorJSON(w, http.StatusBadRequest, fmt.Sprintf("invalid container name %q", name))
			return
		}
		tail, err := parseLogTail(r.URL.Query().Get("tail"))
		if err != nil {
			errorJSON(w, http.StatusBadRequest, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		reader := collectors.NewActor(sys, state.Settings.MockMode).(collectors.LogReader)
		lines, err := reader.Logs(ctx, name, tail)
		if err != nil {
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, models.DockerLogs{Container: name, Lines: lines})
	}
}
