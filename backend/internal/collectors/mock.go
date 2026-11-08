package collectors

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

// --- Mock Linux ---

type MockLinuxCollector struct {
	id, name string
	baseCPU  float64
	baseMem  float64
	uptime   int64
}

func NewMockLinux(id, name string) *MockLinuxCollector {
	return &MockLinuxCollector{
		id:      id,
		name:    name,
		baseCPU: 10 + rand.Float64()*30,
		baseMem: 30 + rand.Float64()*30,
		uptime:  86400 + rand.Int64N(864000-86400),
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (c *MockLinuxCollector) Collect(_ context.Context) *models.SystemMetrics {
	cpuPct := clamp(c.baseCPU+(-10+rand.Float64()*25), 0, 100)
	memPct := clamp(c.baseMem+(-5+rand.Float64()*15), 0, 100)

	totalMem := int64(16) * 1024 * 1024 * 1024
	usedMem := int64(float64(totalMem) * memPct / 100.0)

	cores := make([]float64, 8)
	for i := range cores {
		cores[i] = clamp(cpuPct+(-20+rand.Float64()*40), 0, 100)
	}

	linux := models.LinuxMetrics{
		CPU: models.CPUMetrics{
			Percent: cpuPct,
			Cores:   cores,
			LoadAvg: []float64{cpuPct / 25, cpuPct / 30, cpuPct / 35},
		},
		Memory: models.MemoryMetrics{
			Total:     totalMem,
			Used:      usedMem,
			Available: totalMem - usedMem,
			Percent:   memPct,
		},
		Disks: []models.DiskMetrics{
			{
				Mount:   "/",
				Total:   500 * 1024 * 1024 * 1024,
				Used:    int64(float64(500*1024*1024*1024) * 0.45),
				Free:    int64(float64(500*1024*1024*1024) * 0.55),
				Percent: 45 + (-2 + rand.Float64()*4),
			},
			{
				Mount:   "/home",
				Total:   1000 * 1024 * 1024 * 1024,
				Used:    int64(float64(1000*1024*1024*1024) * 0.62),
				Free:    int64(float64(1000*1024*1024*1024) * 0.38),
				Percent: 62 + (-2 + rand.Float64()*4),
			},
		},
		Network: models.NetworkMetrics{
			BytesSent:    1_000_000_000 + rand.Int64N(4_000_000_000),
			BytesRecv:    5_000_000_000 + rand.Int64N(15_000_000_000),
			UploadRate:   100_000 + rand.Float64()*4_900_000,
			DownloadRate: 500_000 + rand.Float64()*9_500_000,
		},
		Temperatures: []models.TemperatureMetrics{
			{Label: "CPU", Current: 45 + (-5 + rand.Float64()*20)},
			{Label: "GPU", Current: 40 + (-5 + rand.Float64()*15)},
		},
		Fans: []models.FanMetrics{
			{Label: "pwmfan/fan1", RPM: 3000 + rand.Int64N(2000)},
		},
		Processes: []models.ProcessMetrics{
			{PID: 2978, Name: "qbittorrent-nox", CPUPercent: rand.Float64() * 40, MemoryUsage: 600_000_000, MemoryPercent: 14.5},
			{PID: 1351, Name: "traefik", CPUPercent: rand.Float64() * 5, MemoryUsage: 45_000_000, MemoryPercent: 1.1},
			{PID: 1, Name: "systemd", CPUPercent: rand.Float64(), MemoryUsage: 12_000_000, MemoryPercent: 0.3},
		},
		Uptime: c.uptime,
	}
	c.uptime += 5

	var status models.SystemStatus
	switch {
	case linux.CPU.Percent > 90 || linux.Memory.Percent > 95:
		status = models.StatusCritical
	case linux.CPU.Percent > 75 || linux.Memory.Percent > 85:
		status = models.StatusWarning
	default:
		status = models.StatusHealthy
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeLinux,
		Status:      status,
		LastUpdated: time.Now().UTC(),
		Metrics:     linux,
	}
}

func (c *MockLinuxCollector) Close() {}

// --- Mock Docker ---

type mockContainer struct {
	name  string
	state string
}

type MockDockerCollector struct {
	id, name   string
	containers []mockContainer
}

func NewMockDocker(id, name string) *MockDockerCollector {
	return &MockDockerCollector{
		id:   id,
		name: name,
		containers: []mockContainer{
			{"nginx", "running"},
			{"postgres", "running"},
			{"redis", "running"},
			{"api-server", "running"},
			{"worker", "running"},
			{"monitoring", "stopped"},
		},
	}
}

func (c *MockDockerCollector) Collect(_ context.Context) *models.SystemMetrics {
	var containerMetrics []models.ContainerMetrics
	var running, stopped int32
	for _, ct := range c.containers {
		isRunning := ct.state == "running"
		if isRunning {
			running++
		} else {
			stopped++
		}
		var cpu, memPct float64
		var memUsage int64
		status := "Exited (0) 2 hours ago"
		if isRunning {
			cpu = 0.5 + rand.Float64()*14.5
			memPct = 1 + rand.Float64()*24
			memUsage = 50_000_000 + rand.Int64N(450_000_000)
			status = "Up 3 days"
		}
		containerMetrics = append(containerMetrics, models.ContainerMetrics{
			ID:            fmt.Sprintf("mock_%s", ct.name),
			Name:          ct.name,
			Image:         fmt.Sprintf("%s:latest", ct.name),
			State:         ct.state,
			Status:        status,
			CPUPercent:    cpu,
			MemoryPercent: memPct,
			MemoryUsage:   memUsage,
			MemoryLimit:   1024 * 1024 * 1024,
		})
	}

	status := models.StatusHealthy
	if stopped > running {
		status = models.StatusWarning
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeDocker,
		Status:      status,
		LastUpdated: time.Now().UTC(),
		Metrics: models.DockerMetrics{
			Containers:   containerMetrics,
			TotalCount:   int32(len(c.containers)),
			RunningCount: running,
			StoppedCount: stopped,
		},
	}
}

func (c *MockDockerCollector) Close() {}

// --- Mock qBittorrent ---

type mockTorrent struct {
	name     string
	size     int64
	progress float64
	state    string
}

// gib converts a runtime float64 GiB count to bytes. Used because Go forbids
// implicit truncation of untyped float constants.
func gib(f float64) int64 {
	return int64(f * 1024 * 1024 * 1024)
}

type MockQBittorrentCollector struct {
	id, name string
	torrents []mockTorrent
}

func NewMockQBittorrent(id, name string) *MockQBittorrentCollector {
	return &MockQBittorrentCollector{
		id:   id,
		name: name,
		torrents: []mockTorrent{
			{"Ubuntu 24.04 LTS", gib(4.5), 100.0, "stalledUP"},
			{"Debian 12.0", gib(3.8), 100.0, "uploading"},
			{"Fedora 40", gib(2.1), 78.5, "downloading"},
			{"Arch Linux", int64(850 * 1024 * 1024), 45.2, "downloading"},
			{"Linux Mint 21.3", gib(2.8), 92.1, "downloading"},
		},
	}
}

func (c *MockQBittorrentCollector) Collect(_ context.Context) *models.SystemMetrics {
	var infos []models.TorrentInfo
	var activeDL int32
	var totalDL, totalUL int64

	for idx := range c.torrents {
		t := &c.torrents[idx]
		if t.progress < 100 && t.state == "downloading" {
			t.progress = min(t.progress+0.1+rand.Float64()*0.4, 100)
			if t.progress >= 100 {
				t.state = "stalledUP"
				t.progress = 100
			}
		}
		isDownloading := t.state == "downloading" || t.state == "forcedDL"
		isUploading := t.state == "uploading" || t.state == "stalledUP" || t.state == "forcedUP"

		var dl, ul int64
		if isDownloading {
			dl = int64((1 + rand.Float64()*9) * 1024 * 1024)
		}
		if isUploading {
			ul = int64((0.1 + rand.Float64()*1.9) * 1024 * 1024)
		}
		if isDownloading {
			activeDL++
			totalDL += dl
		}
		totalUL += ul

		var eta *int64
		if isDownloading && dl > 0 {
			remaining := int64(float64(t.size) * (100 - t.progress) / 100)
			v := remaining / dl
			eta = &v
		}
		infos = append(infos, models.TorrentInfo{
			Hash:          fmt.Sprintf("%032x", uint64(len(t.name))*0xdeadbeef+uint64(idx)),
			Name:          t.name,
			Size:          t.size,
			Progress:      t.progress,
			State:         t.state,
			DownloadSpeed: dl,
			UploadSpeed:   ul,
			ETA:           eta,
		})
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeQBittorrent,
		Status:      models.StatusHealthy,
		LastUpdated: time.Now().UTC(),
		Metrics: models.QBittorrentMetrics{
			DownloadSpeed:   totalDL,
			UploadSpeed:     totalUL,
			ActiveDownloads: activeDL,
			ActiveUploads:   0,
			TotalTorrents:   int32(len(infos)),
			Torrents:        infos,
		},
	}
}

func (c *MockQBittorrentCollector) Close() {}
