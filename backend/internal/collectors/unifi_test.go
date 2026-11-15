package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

func TestUnifiTemperatures_keysByLowercaseMACAndSetsHighThreshold(t *testing.T) {
	resp := unifiLegacyDevicesResp{Data: []unifiLegacyDevice{{
		MAC:          "AA:BB:CC:00:11:22",
		Temperatures: []unifiLegacyTemp{{Name: "CPU", Value: 61.2}},
	}}}

	got := unifiTemperatures(resp)
	temps := got["aa:bb:cc:00:11:22"]
	if len(temps) != 1 || temps[0].Label != "CPU" || temps[0].Current != 61.2 {
		t.Fatalf("temps: %+v", got)
	}
	if temps[0].High == nil || *temps[0].High != unifiTempWarning {
		t.Fatalf("high: %v", temps[0].High)
	}
}

func TestUnifiStatus_temperatureThreshold(t *testing.T) {
	high := unifiTempWarning
	cases := []struct {
		name string
		temp float64
		want models.SystemStatus
	}{
		{"below threshold", 69.9, models.StatusHealthy},
		{"at threshold", 70, models.StatusWarning},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := models.UnifiMetrics{Devices: []models.UnifiDevice{{
				Status:       "online",
				Temperatures: []models.TemperatureMetrics{{Label: "CPU", Current: tc.temp, High: &high}},
			}}}
			if got := unifiStatus(m); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestUnifiDevicesResp_decodesIntegrationMACAddressField(t *testing.T) {
	// Shape returned by /proxy/network/integration/v1/sites/{id}/devices.
	in := `{"data":[{"macAddress":"8c:ed:e1:70:f9:ed","name":"sw","model":"USW","state":"ONLINE"}]}`
	var resp unifiDevicesResp
	if err := json.Unmarshal([]byte(in), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data[0].MAC != "8c:ed:e1:70:f9:ed" {
		t.Fatalf("mac: %q", resp.Data[0].MAC)
	}
}

func TestUnifiFetchClients_readsEveryPage(t *testing.T) {
	const total = 230
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		resp := unifiClientsResp{TotalCount: total, Data: []unifiClientEntry{}}
		for i := offset; i < min(offset+limit, total); i++ {
			resp.Data = append(resp.Data, unifiClientEntry{MAC: fmt.Sprintf("aa:bb:cc:00:%02x:%02x", i/256, i%256)})
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	c := NewUnifi("u", "u", &config.UnifiSystemConfig{Host: u.Hostname(), Port: port, Site: "default"})
	c.siteUUID = "s1"

	got, err := c.fetchClients(context.Background(), c.newClient())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != total || got[total-1].MAC != "aa:bb:cc:00:00:e5" {
		t.Fatalf("got %d clients, last %+v", len(got), got[len(got)-1])
	}
}

func TestUnifiBlockedClients_fetchesAtMostOncePerRefresh(t *testing.T) {
	hits := 0
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"data":[{"mac":"AA:BB:CC:00:00:01","name":"tv","blocked":true},{"mac":"aa:bb:cc:00:00:02"}]}`))
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	c := NewUnifi("u", "u", &config.UnifiSystemConfig{Host: u.Hostname(), Port: port, Site: "default"})
	client := c.newClient()

	for range 3 {
		got := c.blockedClients(context.Background(), client)
		if len(got) != 1 || got[0].MAC != "aa:bb:cc:00:00:01" {
			t.Fatalf("blocked: %+v", got)
		}
	}
	if hits != 1 {
		t.Fatalf("rest/user fetched %d times within the refresh interval, want 1", hits)
	}

	c.blockedAt = time.Now().Add(-unifiBlockedRefresh)
	c.blockedClients(context.Background(), client)
	if hits != 2 {
		t.Fatalf("stale cache not refreshed: %d fetches", hits)
	}
}
