package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

type DockerCollector struct {
	id, name string
	cfg      *config.DockerSystemConfig
	cli      *client.Client
}

func NewDocker(id, name string, cfg *config.DockerSystemConfig) *DockerCollector {
	return &DockerCollector{id: id, name: name, cfg: cfg}
}

func (c *DockerCollector) ensureClient() error {
	if c.cli != nil {
		return nil
	}
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(10 * time.Second),
	}
	switch {
	case c.cfg.Host != "":
		opts = append(opts, client.WithHost(c.cfg.Host))
	case c.cfg.Socket != "":
		opts = append(opts, client.WithHost(c.cfg.Socket))
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("docker connect: %w", err)
	}
	c.cli = cli
	return nil
}

// statsView is a structural subset of the Docker container stats JSON. Defining
// it locally insulates us from churn in github.com/docker/docker between
// types.StatsJSON / container.StatsResponse and similar renames.
type statsView struct {
	CPUStats struct {
		CPUUsage    struct{ TotalUsage uint64 `json:"total_usage"` } `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint64 `json:"online_cpus"`
	} `json:"cpu_stats"`
	PreCPUStats struct {
		CPUUsage    struct{ TotalUsage uint64 `json:"total_usage"` } `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
	} `json:"precpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
}

func calculateDockerCPU(s *statsView) float64 {
	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	if sysDelta <= 0 || cpuDelta <= 0 {
		return 0
	}
	cpuCount := float64(s.CPUStats.OnlineCPUs)
	if cpuCount == 0 {
		cpuCount = 1
	}
	return cpuDelta / sysDelta * cpuCount * 100.0
}

func calculateDockerMemory(s *statsView) (int64, int64, float64) {
	usage := int64(s.MemoryStats.Usage)
	limit := int64(s.MemoryStats.Limit)
	var pct float64
	if limit > 0 {
		pct = float64(usage) / float64(limit) * 100.0
	}
	return usage, limit, pct
}

func dockerStatus(m models.DockerMetrics) models.SystemStatus {
	if m.TotalCount == 0 {
		return models.StatusHealthy
	}
	var unhealthy, highResource int
	for _, ct := range m.Containers {
		if ct.State != "running" && ct.State != "paused" {
			continue
		}
		if ct.Status == "unhealthy" {
			unhealthy++
		}
		if ct.CPUPercent > 90 || ct.MemoryPercent > 90 {
			highResource++
		}
	}
	if unhealthy > 0 || m.StoppedCount > m.RunningCount {
		return models.StatusWarning
	}
	if highResource > 0 {
		return models.StatusWarning
	}
	return models.StatusHealthy
}

func healthFromStatus(status string) string {
	switch {
	case strings.Contains(status, "healthy") && !strings.Contains(status, "unhealthy"):
		return "healthy"
	case strings.Contains(status, "unhealthy"):
		return "unhealthy"
	}
	return "none"
}

func (c *DockerCollector) containerStats(ctx context.Context, id string) (*statsView, error) {
	resp, err := c.cli.ContainerStatsOneShot(ctx, id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var s statsView
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

func roundTwo(v float64) float64 {
	return math.Round(v*100) / 100
}

func (c *DockerCollector) Collect(ctx context.Context) *models.SystemMetrics {
	if err := c.ensureClient(); err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeDocker, err.Error())
	}
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeDocker, err.Error())
	}

	var containerMetrics []models.ContainerMetrics
	var running, stopped int32
	for _, ct := range containers {
		state := ct.State
		if state == "" {
			state = "unknown"
		}
		name := "unknown"
		if len(ct.Names) > 0 {
			name = strings.TrimPrefix(ct.Names[0], "/")
		}
		id := ct.ID
		if len(id) > 12 {
			id = id[:12]
		}
		image := ct.Image
		if image == "" {
			image = "unknown"
		}

		var cpuPct float64
		var memUsage, memLimit int64
		var memPct float64
		if state == "running" {
			running++
			if s, err := c.containerStats(ctx, ct.ID); err == nil && s != nil {
				cpuPct = calculateDockerCPU(s)
				memUsage, memLimit, memPct = calculateDockerMemory(s)
			}
		} else {
			stopped++
		}

		containerMetrics = append(containerMetrics, models.ContainerMetrics{
			ID:            id,
			Name:          name,
			Image:         image,
			Status:        healthFromStatus(ct.Status),
			State:         state,
			CPUPercent:    roundTwo(cpuPct),
			MemoryUsage:   memUsage,
			MemoryLimit:   memLimit,
			MemoryPercent: roundTwo(memPct),
		})
	}

	docker := models.DockerMetrics{
		Containers:   containerMetrics,
		RunningCount: running,
		StoppedCount: stopped,
		TotalCount:   int32(len(containers)),
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeDocker,
		Status:      dockerStatus(docker),
		LastUpdated: time.Now().UTC(),
		Metrics:     docker,
	}
}

func (c *DockerCollector) Close() {
	if c.cli != nil {
		_ = c.cli.Close()
		c.cli = nil
	}
}
