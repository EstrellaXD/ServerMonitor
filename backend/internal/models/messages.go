package models

import "time"

type MetricsUpdate struct {
	Type      string                    `json:"type"`
	Timestamp time.Time                 `json:"timestamp"`
	Systems   map[string]*SystemMetrics `json:"systems"`
}

func NewMetricsUpdate(systems map[string]*SystemMetrics) MetricsUpdate {
	return MetricsUpdate{
		Type:      "metrics_update",
		Timestamp: time.Now().UTC(),
		Systems:   systems,
	}
}
