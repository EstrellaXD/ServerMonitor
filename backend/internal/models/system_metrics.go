package models

import "time"

type SystemMetrics struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        SystemType   `json:"type"`
	Status      SystemStatus `json:"status"`
	LastUpdated time.Time    `json:"last_updated"`
	Error       *string      `json:"error,omitempty"`
	Metrics     any          `json:"metrics"`
}

func NewErrorMetrics(id, name string, sysType SystemType, errMsg string) *SystemMetrics {
	return &SystemMetrics{
		ID:          id,
		Name:        name,
		Type:        sysType,
		Status:      StatusOffline,
		LastUpdated: time.Now().UTC(),
		Error:       &errMsg,
		Metrics:     struct{}{},
	}
}
