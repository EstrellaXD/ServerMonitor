package collectors

import (
	"context"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

// Collector produces a SystemMetrics snapshot. Implementations must be
// safe to call serially from the manager loop; concurrent calls are not
// supported and not required.
type Collector interface {
	Collect(ctx context.Context) *models.SystemMetrics
	Close()
}

// New builds a Collector for a system config; returns nil for unknown types.
// When mockMode is true, the live config is ignored and a deterministic
// mock collector is returned for supported types.
func New(sys config.SystemConfig, mockMode bool) Collector {
	if mockMode {
		switch sys.Type {
		case "linux":
			return NewMockLinux(sys.ID, sys.Name)
		case "docker":
			return NewMockDocker(sys.ID, sys.Name)
		case "qbittorrent":
			return NewMockQBittorrent(sys.ID, sys.Name)
		}
		return nil
	}

	switch cfg := sys.Config.(type) {
	case *config.LinuxSystemConfig:
		return NewLinux(sys.ID, sys.Name, cfg)
	case *config.DockerSystemConfig:
		return NewDocker(sys.ID, sys.Name, cfg)
	case *config.QBittorrentSystemConfig:
		return NewQBittorrent(sys.ID, sys.Name, cfg)
	case *config.UnifiSystemConfig:
		return NewUnifi(sys.ID, sys.Name, cfg)
	case *config.UNASSystemConfig:
		return NewUNAS(sys.ID, sys.Name, cfg)
	}
	return nil
}
