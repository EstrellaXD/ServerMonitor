package collectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

type UnifiCollector struct {
	id, name string
	cfg      *config.UnifiSystemConfig
	siteUUID string

	// blocked caches the blocked clients from rest/user, fetched at
	// blockedAt (zero before the first successful fetch).
	blocked   []models.UnifiClient
	blockedAt time.Time
}

// unifiBlockedRefresh is the minimum time between two fetches of the legacy
// rest/user list. That list holds every client the controller has ever seen,
// so it is too large to fetch on every poll.
const unifiBlockedRefresh = 60 * time.Second

func NewUnifi(id, name string, cfg *config.UnifiSystemConfig) *UnifiCollector {
	return &UnifiCollector{id: id, name: name, cfg: cfg}
}

func (c *UnifiCollector) baseURL() string {
	return fmt.Sprintf("https://%s:%d/proxy/network/integration/v1", c.cfg.Host, c.cfg.Port)
}

func (c *UnifiCollector) siteURL(path string) string {
	return fmt.Sprintf("%s/sites/%s/%s", c.baseURL(), c.siteUUID, path)
}

func (c *UnifiCollector) newClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !c.cfg.VerifySSL}, //nolint:gosec
		},
	}
}

func (c *UnifiCollector) getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

type unifiSitesResp struct {
	Data []struct {
		ID                string `json:"id"`
		InternalReference string `json:"internalReference"`
	} `json:"data"`
}

type unifiDevicesResp struct {
	Data []struct {
		Name   string `json:"name"`
		MAC    string `json:"macAddress"`
		Model  string `json:"model"`
		State  string `json:"state"`
		Uptime int64  `json:"uptime"`
	} `json:"data"`
}

// unifiTempWarning is the sensor reading (°C) at which a device turns the
// system status to warning.
const unifiTempWarning = 70.0

// The Integration API has no sensor data, so temperatures come from the
// legacy Network API (stat/device), which accepts the same API key.
type unifiLegacyTemp struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

type unifiLegacyDevice struct {
	MAC          string            `json:"mac"`
	Temperatures []unifiLegacyTemp `json:"temperatures"`
}

type unifiLegacyDevicesResp struct {
	Data []unifiLegacyDevice `json:"data"`
}

// unifiTemperatures maps lowercase MAC → sensor readings.
func unifiTemperatures(resp unifiLegacyDevicesResp) map[string][]models.TemperatureMetrics {
	high := unifiTempWarning
	out := make(map[string][]models.TemperatureMetrics, len(resp.Data))
	for _, d := range resp.Data {
		temps := make([]models.TemperatureMetrics, 0, len(d.Temperatures))
		for _, t := range d.Temperatures {
			temps = append(temps, models.TemperatureMetrics{Label: t.Name, Current: t.Value, High: &high})
		}
		out[strings.ToLower(d.MAC)] = temps
	}
	return out
}

type unifiClientEntry struct {
	IsGuest bool   `json:"isGuest"`
	Name    string `json:"name"`
	IP      string `json:"ipAddress"`
	MAC     string `json:"macAddress"`
}

type unifiClientsResp struct {
	TotalCount int                `json:"totalCount"`
	Data       []unifiClientEntry `json:"data"`
}

// unifiPageLimit is the page size for the paginated clients list. The API
// default is 25; 200 is its maximum.
const unifiPageLimit = 200

// fetchClients reads every page of the Integration API clients list.
func (c *UnifiCollector) fetchClients(ctx context.Context, client *http.Client) ([]unifiClientEntry, error) {
	var all []unifiClientEntry
	for {
		var page unifiClientsResp
		url := fmt.Sprintf("%s?offset=%d&limit=%d", c.siteURL("clients"), len(all), unifiPageLimit)
		if err := c.getJSON(ctx, client, url, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if len(page.Data) == 0 || len(all) >= page.TotalCount {
			return all, nil
		}
	}
}

// unifiLegacyUsersResp is the legacy rest/user list: every known client,
// including offline ones, with its block flag.
type unifiLegacyUsersResp struct {
	Data []struct {
		MAC      string `json:"mac"`
		Name     string `json:"name"`
		Hostname string `json:"hostname"`
		LastIP   string `json:"last_ip"`
		IsGuest  bool   `json:"is_guest"`
		Blocked  bool   `json:"blocked"`
	} `json:"data"`
}

func unifiBlockedClients(resp unifiLegacyUsersResp) []models.UnifiClient {
	out := []models.UnifiClient{}
	for _, u := range resp.Data {
		if !u.Blocked {
			continue
		}
		mac := strings.ToLower(u.MAC)
		name := u.Name
		if name == "" {
			name = u.Hostname
		}
		if name == "" {
			name = mac
		}
		out = append(out, models.UnifiClient{MAC: mac, Name: name, IP: u.LastIP, IsGuest: u.IsGuest})
	}
	return out
}

// legacyURL builds a legacy Network API URL (api/s/{site}/...), which
// accepts the same API key as the Integration API.
// blockedClients returns the cached blocked clients and refetches them when
// the cache is older than unifiBlockedRefresh. They are supplementary like
// temperatures: a failed fetch keeps the last result and retries next poll.
func (c *UnifiCollector) blockedClients(ctx context.Context, client *http.Client) []models.UnifiClient {
	if time.Since(c.blockedAt) >= unifiBlockedRefresh {
		var users unifiLegacyUsersResp
		if err := c.getJSON(ctx, client, c.legacyURL("rest/user"), &users); err != nil {
			slog.Warn("unifi: blocked client fetch failed", "id", c.id, "err", err)
		} else {
			c.blocked, c.blockedAt = unifiBlockedClients(users), time.Now()
		}
	}
	if c.blocked == nil {
		return []models.UnifiClient{}
	}
	return c.blocked
}

func (c *UnifiCollector) legacyURL(path string) string {
	return fmt.Sprintf("https://%s:%d/proxy/network/api/s/%s/%s", c.cfg.Host, c.cfg.Port, c.cfg.Site, path)
}

func (c *UnifiCollector) resolveSiteUUID(ctx context.Context, client *http.Client) error {
	if c.siteUUID != "" {
		return nil
	}
	var sites unifiSitesResp
	if err := c.getJSON(ctx, client, c.baseURL()+"/sites", &sites); err != nil {
		return err
	}
	for _, s := range sites.Data {
		if s.InternalReference == c.cfg.Site {
			c.siteUUID = s.ID
			return nil
		}
	}
	return fmt.Errorf("UniFi site %q not found", c.cfg.Site)
}

func unifiStatus(m models.UnifiMetrics) models.SystemStatus {
	if len(m.Devices) == 0 {
		return models.StatusHealthy
	}
	offline := 0
	for _, d := range m.Devices {
		if d.Status != "online" {
			offline++
		}
	}
	if offline > len(m.Devices)/2 {
		return models.StatusCritical
	}
	if offline > 0 {
		return models.StatusWarning
	}
	for _, d := range m.Devices {
		for _, t := range d.Temperatures {
			if t.High != nil && t.Current >= *t.High {
				return models.StatusWarning
			}
		}
	}
	return models.StatusHealthy
}

func (c *UnifiCollector) Collect(ctx context.Context) *models.SystemMetrics {
	client := c.newClient()
	if err := c.resolveSiteUUID(ctx, client); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnifi, err.Error())
	}

	var devices unifiDevicesResp
	if err := c.getJSON(ctx, client, c.siteURL("devices"), &devices); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnifi, err.Error())
	}

	clients, err := c.fetchClients(ctx, client)
	if err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnifi, err.Error())
	}

	// Temperatures are supplementary: if the legacy API fails, still report
	// devices and clients, just without sensor readings.
	var legacy unifiLegacyDevicesResp
	if err := c.getJSON(ctx, client, c.legacyURL("stat/device"), &legacy); err != nil {
		slog.Warn("unifi: temperature fetch failed", "id", c.id, "err", err)
	}
	temps := unifiTemperatures(legacy)

	blocked := c.blockedClients(ctx, client)

	out := make([]models.UnifiDevice, 0, len(devices.Data))
	for _, d := range devices.Data {
		status := "offline"
		if strings.EqualFold(d.State, "ONLINE") {
			status = "online"
		}
		name := d.Name
		if name == "" {
			name = d.MAC
		}
		if name == "" {
			name = "Unknown"
		}
		model := d.Model
		if model == "" {
			model = "Unknown"
		}
		out = append(out, models.UnifiDevice{
			Name:         name,
			MAC:          d.MAC,
			Model:        model,
			Status:       status,
			Uptime:       d.Uptime,
			Temperatures: temps[strings.ToLower(d.MAC)],
		})
	}

	var guests int32
	active := make([]models.UnifiClient, 0, len(clients))
	for _, cl := range clients {
		if cl.IsGuest {
			guests++
		}
		mac := strings.ToLower(cl.MAC)
		name := cl.Name
		if name == "" {
			name = mac
		}
		active = append(active, models.UnifiClient{MAC: mac, Name: name, IP: cl.IP, IsGuest: cl.IsGuest})
	}
	total := int32(len(clients))

	m := models.UnifiMetrics{
		Devices:        out,
		ClientsCount:   total - guests,
		GuestsCount:    guests,
		Clients:        active,
		BlockedClients: blocked,
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeUnifi,
		Status:      unifiStatus(m),
		LastUpdated: time.Now().UTC(),
		Metrics:     m,
	}
}

func (c *UnifiCollector) Close() {}
