package collectors

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

// --- Mock Linux ---

type MockLinuxCollector struct {
	id, name string
	baseCPU  float64
	baseMem  float64
	uptime   int64
	services []string
}

func NewMockLinux(id, name string, services []string) *MockLinuxCollector {
	return &MockLinuxCollector{
		id:       id,
		name:     name,
		services: services,
		baseCPU:  10 + rand.Float64()*30,
		baseMem:  30 + rand.Float64()*30,
		uptime:   86400 + rand.Int64N(864000-86400),
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
			{Label: "pwmfan/fan1", RPM: 3000 + rand.Int64N(2000), PWMPercent: ptr(40 + float64(rand.IntN(20))), Mode: "auto"},
			{Label: "nct6775/fan2", RPM: 900 + rand.Int64N(200)},
		},
		Processes: []models.ProcessMetrics{
			{PID: 2978, Name: "qbittorrent-nox", CPUPercent: rand.Float64() * 40, MemoryUsage: 600_000_000, MemoryPercent: 14.5},
			{PID: 1351, Name: "traefik", CPUPercent: rand.Float64() * 5, MemoryUsage: 45_000_000, MemoryPercent: 1.1},
			{PID: 1, Name: "systemd", CPUPercent: rand.Float64(), MemoryUsage: 12_000_000, MemoryPercent: 0.3},
		},
		Services: mockServices(c.services),
		Uptime:   c.uptime,
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
			Hash:          fmt.Sprintf("%040x", uint64(len(t.name))*0xdeadbeef+uint64(idx)),
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

func ptr[T any](v T) *T { return &v }

// mockServices reports every configured unit as running.
func mockServices(units []string) []models.ServiceStatus {
	out := make([]models.ServiceStatus, 0, len(units))
	for _, u := range units {
		out = append(out, models.ServiceStatus{Name: u, ActiveState: "active", SubState: "running"})
	}
	return out
}

// MockSystemConfig returns the config a system runs with in mock mode: every
// control of its type is enabled, and a Linux system without services gets
// one, so that the whole UI can be exercised.
func MockSystemConfig(sys config.SystemConfig) config.SystemConfig {
	sys.Controls = config.AllControls(sys.Type)
	if lc, ok := sys.Config.(*config.LinuxSystemConfig); ok && len(lc.Services) == 0 {
		c := *lc
		c.Services = []string{"servermonitor"}
		sys.Config = &c
	}
	return sys
}

// --- Mock UniFi ---

type MockUnifiCollector struct {
	id, name string
}

func NewMockUnifi(id, name string) *MockUnifiCollector {
	return &MockUnifiCollector{id: id, name: name}
}

var mockUnifiClients = []models.UnifiClient{
	{MAC: "3c:22:fb:10:aa:01", Name: "macbook-pro", IP: "192.168.1.21"},
	{MAC: "a4:83:e7:42:bb:02", Name: "iphone", IP: "192.168.1.22"},
	{MAC: "dc:a6:32:01:cc:03", Name: "raspberrypi", IP: "192.168.1.170"},
	{MAC: "48:b0:2d:5e:dd:04", Name: "jetson-orin", IP: "192.168.1.14"},
	{MAC: "f0:18:98:7a:ee:05", Name: "ipad", IP: "192.168.1.23"},
	{MAC: "b8:27:eb:33:ff:06", Name: "printer", IP: "192.168.1.40"},
	{MAC: "70:ee:50:12:a0:07", Name: "thermostat", IP: "192.168.1.41"},
	{MAC: "8a:11:22:33:b0:08", Name: "guest-phone", IP: "192.168.10.5", IsGuest: true},
}

func (c *MockUnifiCollector) Collect(_ context.Context) *models.SystemMetrics {
	high := unifiTempWarning
	devices := []models.UnifiDevice{
		{Name: "UDM Pro", MAC: "74:ac:b9:00:00:01", Model: "UDMPRO", Status: "online", Uptime: 1_209_600,
			Temperatures: []models.TemperatureMetrics{
				{Label: "CPU", Current: 55 + rand.Float64()*8, High: &high},
				{Label: "Local", Current: 45 + rand.Float64()*5, High: &high},
			}},
		{Name: "USW Lite 8", MAC: "74:ac:b9:00:00:02", Model: "USL8LP", Status: "online", Uptime: 1_100_000},
		{Name: "U6 Lite", MAC: "74:ac:b9:00:00:03", Model: "U6-Lite", Status: "online", Uptime: 980_000},
	}
	m := models.UnifiMetrics{
		Devices:        devices,
		ClientsCount:   7,
		GuestsCount:    1,
		WANDownload:    20 + rand.Float64()*480,
		WANUpload:      5 + rand.Float64()*95,
		Clients:        mockUnifiClients,
		BlockedClients: []models.UnifiClient{{MAC: "00:11:22:aa:bb:cc", Name: "unknown-tv", IP: ""}},
	}
	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeUnifi,
		Status:      unifiStatus(m),
		LastUpdated: time.Now().UTC(),
		Metrics:     m,
	}
}

func (c *MockUnifiCollector) Close() {}

// --- Mock UNAS ---

type MockUNASCollector struct {
	id, name string
}

func NewMockUNAS(id, name string) *MockUNASCollector {
	return &MockUNASCollector{id: id, name: name}
}

func (c *MockUNASCollector) Collect(_ context.Context) *models.SystemMetrics {
	total := gib(16 * 1024)
	used := int64(float64(total) * 0.58)
	disks := make([]models.StorageDisk, 0, 4)
	for i := range 4 {
		disks = append(disks, models.StorageDisk{
			Name:        fmt.Sprintf("sd%c", 'a'+i),
			Model:       "WDC WD80EFZZ",
			Serial:      fmt.Sprintf("WD-MOCK%04d", i+1),
			Status:      "healthy",
			Temperature: ptr(36 + rand.Float64()*6),
		})
	}
	m := models.UNASMetrics{
		Pools: []models.StoragePool{{
			Name: "/volume1", Status: "online", Total: total, Used: used, Available: total - used,
			Percent: float64(used) / float64(total) * 100,
		}},
		Disks:         disks,
		TotalCapacity: total,
		UsedCapacity:  used,
	}
	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeUnas,
		Status:      unasStatus(m),
		LastUpdated: time.Now().UTC(),
		Metrics:     m,
	}
}

func (c *MockUNASCollector) Close() {}

// --- Mock actor ---

// mockActor simulates every action. The allowlist and validation run before
// it, exactly as for a real actor.
type mockActor struct{}

func (mockActor) Run(ctx context.Context, a Action) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(300 * time.Millisecond):
	}
	if a.Target == "" {
		return fmt.Sprintf("mock: %s ok", a.Name), nil
	}
	return fmt.Sprintf("mock: %s %s ok", a.Name, a.Target), nil
}

func (mockActor) Logs(_ context.Context, name string, tail int) ([]models.LogLine, error) {
	const total = 400
	n := min(tail, total)
	lines := make([]models.LogLine, 0, n)
	start := time.Now().Add(-time.Duration(n) * time.Second).UTC()
	for i := range n {
		ts := start.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		line := models.LogLine{Stream: "stdout", Text: fmt.Sprintf("%s %s: handled request #%d in %dms", ts, name, total-n+i+1, 2+rand.IntN(40))}
		if i%17 == 16 {
			line = models.LogLine{Stream: "stderr", Text: fmt.Sprintf("%s %s: warning: upstream timeout, retrying", ts, name)}
		}
		lines = append(lines, line)
	}
	return lines, nil
}
