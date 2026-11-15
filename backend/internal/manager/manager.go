package manager

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/broadcast"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/collectors"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/store"
)

type Manager struct {
	mu           sync.Mutex
	collectors   map[string]collectors.Collector
	systems      map[string]config.SystemConfig // effective config of each collected system
	pollInterval time.Duration
	allowedHosts []string
	store        *store.MetricsStore
	bus          *broadcast.Broadcaster
	mockMode     bool

	cancel context.CancelFunc
	done   chan struct{}
}

func New(store *store.MetricsStore, bus *broadcast.Broadcaster, mockMode bool) *Manager {
	return &Manager{
		collectors:   make(map[string]collectors.Collector),
		systems:      make(map[string]config.SystemConfig),
		pollInterval: 5 * time.Second,
		store:        store,
		bus:          bus,
		mockMode:     mockMode,
	}
}

// LoadConfig replaces the manager's collector set from the given config.
// Caller is expected to have stopped the manager first.
func (m *Manager) LoadConfig(cfg config.AppConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.PollInterval > 0 {
		m.pollInterval = time.Duration(cfg.PollInterval) * time.Second
	}
	m.allowedHosts = cfg.AllowedHosts
	m.collectors = make(map[string]collectors.Collector)
	m.systems = make(map[string]config.SystemConfig)

	for _, sys := range cfg.Systems {
		if !sys.Enabled {
			continue
		}
		if m.mockMode {
			sys = collectors.MockSystemConfig(sys)
		}
		col := collectors.New(sys, m.mockMode)
		if col == nil {
			slog.Warn("no collector available for system", "id", sys.ID, "type", sys.Type)
			continue
		}
		m.collectors[sys.ID] = col
		sys.Controls = append([]string{}, sys.Controls...) // never nil on the wire
		m.systems[sys.ID] = sys
	}
}

// System returns the effective config of a collected system: the one the
// action allowlist is checked against.
func (m *Manager) System(id string) (config.SystemConfig, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sys, ok := m.systems[id]
	return sys, ok
}

// AllowedHosts returns the allowed_hosts list of the loaded config.
func (m *Manager) AllowedHosts() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.allowedHosts
}

func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.collectors)
}

func (m *Manager) IDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]string, 0, len(m.collectors))
	for id := range m.collectors {
		ids = append(ids, id)
	}
	return ids
}

// Start kicks off the poll loop in a background goroutine. Subsequent calls
// without a Stop in between are no-ops.
func (m *Manager) Start(parent context.Context) {
	m.mu.Lock()
	if m.cancel != nil {
		m.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.done = make(chan struct{})
	m.mu.Unlock()

	go m.loop(ctx)
}

func (m *Manager) loop(ctx context.Context) {
	defer close(m.done)
	slog.Info("collection loop started", "poll_interval", m.pollInterval, "collectors", m.Count())

	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	m.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.tick(ctx)
		}
	}
}

func (m *Manager) tick(ctx context.Context) {
	m.mu.Lock()
	// Snapshot the collector list so we can call collectors without holding
	// the manager lock for the entire round-trip.
	snapshot := make([]struct {
		id        string
		collector collectors.Collector
		controls  []string
	}, 0, len(m.collectors))
	for id, c := range m.collectors {
		snapshot = append(snapshot, struct {
			id        string
			collector collectors.Collector
			controls  []string
		}{id, c, m.systems[id].Controls})
	}
	m.mu.Unlock()

	for _, entry := range snapshot {
		if ctx.Err() != nil {
			return
		}
		metrics := entry.collector.Collect(ctx)
		if metrics != nil {
			metrics.Controls = entry.controls
			m.store.Update(entry.id, metrics)
		}
	}

	update := m.store.Snapshot()
	payload, err := json.Marshal(update)
	if err != nil {
		slog.Error("marshal metrics update", "err", err)
		return
	}
	m.bus.Publish(payload)
}

// Stop cancels the loop, waits for it to exit, and closes all collectors.
func (m *Manager) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.cancel = nil
	m.done = nil
	m.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	if done != nil {
		<-done
	}

	m.mu.Lock()
	for _, c := range m.collectors {
		c.Close()
	}
	m.mu.Unlock()
}
