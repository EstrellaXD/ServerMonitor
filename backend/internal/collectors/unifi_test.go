package collectors

import (
	"encoding/json"
	"testing"

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
