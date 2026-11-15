package collectors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
)

type qbitActor struct {
	cfg *config.QBittorrentSystemConfig
}

// qbitEndpoints maps an action to its qBittorrent Web API endpoint.
var qbitEndpoints = map[string]string{
	"resume":        "/api/v2/torrents/start",
	"pause":         "/api/v2/torrents/stop",
	"recheck":       "/api/v2/torrents/recheck",
	"top_prio":      "/api/v2/torrents/topPrio",
	"bottom_prio":   "/api/v2/torrents/bottomPrio",
	"increase_prio": "/api/v2/torrents/increasePrio",
	"decrease_prio": "/api/v2/torrents/decreasePrio",
	"delete":        "/api/v2/torrents/delete",
}

// qbitLegacyEndpoints are the names before qBittorrent 5.0 (WebAPI 2.11)
// renamed resume/pause to start/stop. They are used when the new name
// returns 404.
var qbitLegacyEndpoints = map[string]string{
	"resume": "/api/v2/torrents/resume",
	"pause":  "/api/v2/torrents/pause",
}

func (q *qbitActor) Run(ctx context.Context, a Action) (string, error) {
	endpoint, ok := qbitEndpoints[a.Name]
	if !ok {
		return "", fmt.Errorf("unsupported qbittorrent action %q", a.Name)
	}
	hc := NewQBittorrentHTTPClient()
	if err := AuthenticateQBittorrent(ctx, hc, q.cfg); err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("hashes", a.Target)
	if a.Name == "delete" {
		form.Set("deleteFiles", strconv.FormatBool(a.Params.DeleteFiles))
	}
	resp, err := qbitPost(ctx, hc, q.cfg.URL+endpoint, form)
	if legacy, ok := qbitLegacyEndpoints[a.Name]; ok && err == nil && resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		resp, err = qbitPost(ctx, hc, q.cfg.URL+legacy, form)
	}
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 409 on the priority endpoints means torrent queueing is disabled.
		text, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("qBittorrent API returned %s: %s", resp.Status, strings.TrimSpace(string(text)))
	}
	return "", nil
}

func qbitPost(ctx context.Context, hc *http.Client, u string, form url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return hc.Do(req)
}
