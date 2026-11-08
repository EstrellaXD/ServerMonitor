package store

import (
	"sync"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

type MetricsStore struct {
	mu      sync.RWMutex
	systems map[string]*models.SystemMetrics
}

func New() *MetricsStore {
	return &MetricsStore{systems: make(map[string]*models.SystemMetrics)}
}

func (s *MetricsStore) Update(id string, metrics *models.SystemMetrics) {
	s.mu.Lock()
	s.systems[id] = metrics
	s.mu.Unlock()
}

func (s *MetricsStore) Get(id string) (*models.SystemMetrics, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.systems[id]
	return m, ok
}

func (s *MetricsStore) GetAll() map[string]*models.SystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*models.SystemMetrics, len(s.systems))
	for k, v := range s.systems {
		out[k] = v
	}
	return out
}

func (s *MetricsStore) Clear() {
	s.mu.Lock()
	s.systems = make(map[string]*models.SystemMetrics)
	s.mu.Unlock()
}

func (s *MetricsStore) Snapshot() models.MetricsUpdate {
	return models.NewMetricsUpdate(s.GetAll())
}
