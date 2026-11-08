package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/collectors"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
)

type dockerActionReq struct {
	ContainerName string `json:"container_name"`
	Action        string `json:"action"`
}

type qbitActionReq struct {
	Hash        string `json:"hash"`
	Action      string `json:"action"`
	DeleteFiles bool   `json:"delete_files"`
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func dockerActionHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		systemID := chi.URLParam(r, "system_id")
		raw, ok := state.SystemConfig(systemID)
		if !ok {
			errorJSON(w, http.StatusNotFound, "Docker system not found")
			return
		}
		cfg, ok := raw.(*config.DockerSystemConfig)
		if !ok {
			errorJSON(w, http.StatusNotFound, "Docker system not found")
			return
		}

		var body dockerActionReq
		if err := decodeJSON(r, &body); err != nil {
			errorJSON(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.Action != "start" && body.Action != "stop" && body.Action != "restart" {
			errorJSON(w, http.StatusBadRequest, "Invalid action: "+body.Action)
			return
		}

		opts := []client.Opt{
			client.WithAPIVersionNegotiation(),
			client.WithTimeout(10 * time.Second),
		}
		switch {
		case cfg.Host != "":
			opts = append(opts, client.WithHost(cfg.Host))
		case cfg.Socket != "":
			opts = append(opts, client.WithHost(cfg.Socket))
		}
		cli, err := client.NewClientWithOpts(opts...)
		if err != nil {
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer cli.Close()

		ctx := r.Context()
		switch body.Action {
		case "start":
			err = cli.ContainerStart(ctx, body.ContainerName, container.StartOptions{})
		case "stop":
			err = cli.ContainerStop(ctx, body.ContainerName, container.StopOptions{})
		case "restart":
			err = cli.ContainerRestart(ctx, body.ContainerName, container.StopOptions{})
		}
		if err != nil {
			slog.Error("docker action failed", "system", systemID, "action", body.Action, "container", body.ContainerName, "err", err)
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		slog.Info("docker action", "system", systemID, "action", body.Action, "container", body.ContainerName)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "success",
			"action":    body.Action,
			"container": body.ContainerName,
		})
	}
}

func qbitActionHandler(state *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		systemID := chi.URLParam(r, "system_id")
		raw, ok := state.SystemConfig(systemID)
		if !ok {
			errorJSON(w, http.StatusNotFound, "qBittorrent system not found")
			return
		}
		cfg, ok := raw.(*config.QBittorrentSystemConfig)
		if !ok {
			errorJSON(w, http.StatusNotFound, "qBittorrent system not found")
			return
		}

		var body qbitActionReq
		if err := decodeJSON(r, &body); err != nil {
			errorJSON(w, http.StatusBadRequest, err.Error())
			return
		}
		if body.Action != "resume" && body.Action != "pause" && body.Action != "delete" {
			errorJSON(w, http.StatusBadRequest, "Invalid action: "+body.Action)
			return
		}

		ctx := r.Context()
		hc := collectors.NewQBittorrentHTTPClient()
		if err := collectors.AuthenticateQBittorrent(ctx, hc, cfg); err != nil {
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}

		endpoint := ""
		form := url.Values{}
		switch body.Action {
		case "resume":
			endpoint = "/api/v2/torrents/resume"
			form.Set("hashes", body.Hash)
		case "pause":
			endpoint = "/api/v2/torrents/pause"
			form.Set("hashes", body.Hash)
		case "delete":
			endpoint = "/api/v2/torrents/delete"
			form.Set("hashes", body.Hash)
			form.Set("deleteFiles", fmt.Sprintf("%t", body.DeleteFiles))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL+endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := hc.Do(req)
		if err != nil {
			slog.Error("qbit action failed", "system", systemID, "action", body.Action, "err", err)
			errorJSON(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Info("qbit action", "system", systemID, "action", body.Action, "hash", body.Hash)
			writeJSON(w, http.StatusOK, map[string]any{
				"status": "success",
				"action": body.Action,
				"hash":   body.Hash,
			})
			return
		}
		text, _ := io.ReadAll(resp.Body)
		slog.Error("qbit action returned non-2xx", "system", systemID, "action", body.Action, "status", resp.StatusCode, "body", string(text))
		errorJSON(w, http.StatusInternalServerError, fmt.Sprintf("qBittorrent API returned %d %s", resp.StatusCode, resp.Status))
	}
}
