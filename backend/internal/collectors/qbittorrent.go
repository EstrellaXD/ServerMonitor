package collectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

const qbitNoETA = int64(8_640_000)

type QBittorrentCollector struct {
	id, name string
	cfg      *config.QBittorrentSystemConfig
}

func NewQBittorrent(id, name string, cfg *config.QBittorrentSystemConfig) *QBittorrentCollector {
	return &QBittorrentCollector{id: id, name: name, cfg: cfg}
}

// NewQBittorrentHTTPClient returns an HTTP client configured to mirror the Rust
// reqwest setup: cookie jar, 10s timeout, accept invalid certs.
func NewQBittorrentHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 10 * time.Second,
		Jar:     jar,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

// AuthenticateQBittorrent posts the login form and returns nil on success.
func AuthenticateQBittorrent(ctx context.Context, client *http.Client, cfg *config.QBittorrentSystemConfig) error {
	form := url.Values{}
	form.Set("username", cfg.Username)
	form.Set("password", cfg.Password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/api/v2/auth/login", cfg.URL),
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) != "Ok." {
		return errors.New("qBittorrent authentication failed")
	}
	return nil
}

func (c *QBittorrentCollector) getJSON(ctx context.Context, client *http.Client, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.URL+path, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

type qbitTransfer struct {
	DLInfoSpeed int64 `json:"dl_info_speed"`
	UpInfoSpeed int64 `json:"up_info_speed"`
}

type qbitTorrent struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
	DLSpeed  int64   `json:"dlspeed"`
	UpSpeed  int64   `json:"upspeed"`
	State    string  `json:"state"`
	ETA      int64   `json:"eta"`
}

func (c *QBittorrentCollector) Collect(ctx context.Context) *models.SystemMetrics {
	client := NewQBittorrentHTTPClient()
	if err := AuthenticateQBittorrent(ctx, client, c.cfg); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeQBittorrent, err.Error())
	}

	var transfer qbitTransfer
	if err := c.getJSON(ctx, client, "/api/v2/transfer/info", &transfer); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeQBittorrent, err.Error())
	}

	var torrents []qbitTorrent
	if err := c.getJSON(ctx, client, "/api/v2/torrents/info", &torrents); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeQBittorrent, err.Error())
	}

	var activeDL, activeUL int32
	infos := make([]models.TorrentInfo, 0, len(torrents))
	for _, t := range torrents {
		switch t.State {
		case "downloading", "forcedDL", "metaDL", "queuedDL":
			activeDL++
		case "uploading", "forcedUP", "stalledUP", "queuedUP":
			activeUL++
		}
		var eta *int64
		if t.ETA < qbitNoETA {
			v := t.ETA
			eta = &v
		}
		infos = append(infos, models.TorrentInfo{
			Hash:          t.Hash,
			Name:          t.Name,
			Size:          t.Size,
			Progress:      t.Progress * 100,
			DownloadSpeed: t.DLSpeed,
			UploadSpeed:   t.UpSpeed,
			State:         t.State,
			ETA:           eta,
		})
	}
	if len(infos) > 50 {
		infos = infos[:50]
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeQBittorrent,
		Status:      models.StatusHealthy,
		LastUpdated: time.Now().UTC(),
		Metrics: models.QBittorrentMetrics{
			DownloadSpeed:   transfer.DLInfoSpeed,
			UploadSpeed:     transfer.UpInfoSpeed,
			ActiveDownloads: activeDL,
			ActiveUploads:   activeUL,
			TotalTorrents:   int32(len(torrents)),
			Torrents:        infos,
		},
	}
}

func (c *QBittorrentCollector) Close() {}
