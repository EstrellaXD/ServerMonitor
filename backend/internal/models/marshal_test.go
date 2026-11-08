package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSystemMetrics_omitsErrorWhenNil(t *testing.T) {
	m := SystemMetrics{
		ID:          "x",
		Name:        "n",
		Type:        TypeLinux,
		Status:      StatusHealthy,
		LastUpdated: time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
		Metrics:     struct{}{},
	}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"error"`) {
		t.Fatalf("expected error omitted: %s", b)
	}
	if !strings.Contains(string(b), `"type":"linux"`) {
		t.Fatalf("type should be lowercase: %s", b)
	}
}

func TestSystemMetrics_includesErrorWhenSet(t *testing.T) {
	msg := "boom"
	m := SystemMetrics{Type: TypeDocker, Error: &msg}
	b, _ := json.Marshal(m)
	if !strings.Contains(string(b), `"error":"boom"`) {
		t.Fatalf("expected error field: %s", b)
	}
}

func TestTorrentInfo_omitsETAWhenNil(t *testing.T) {
	b, _ := json.Marshal(TorrentInfo{Hash: "h"})
	if strings.Contains(string(b), `"eta"`) {
		t.Fatalf("eta should be omitted: %s", b)
	}
}

func TestSystemStatus_lowercaseMatchesFrontendUnion(t *testing.T) {
	cases := []SystemStatus{StatusHealthy, StatusWarning, StatusCritical, StatusOffline}
	for _, s := range cases {
		b, _ := json.Marshal(s)
		v := strings.Trim(string(b), `"`)
		if v != string(s) {
			t.Fatalf("marshal of %q produced %q", s, v)
		}
	}
}

func TestMetricsUpdate_carriesTypeField(t *testing.T) {
	u := NewMetricsUpdate(nil)
	b, _ := json.Marshal(u)
	if !strings.Contains(string(b), `"type":"metrics_update"`) {
		t.Fatalf("missing metrics_update type: %s", b)
	}
}
