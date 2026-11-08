package collectors

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EstrellaXD/ServerMonitor/backend/internal/config"
	"github.com/EstrellaXD/ServerMonitor/backend/internal/models"
)

type UNASCollector struct {
	id, name string
	ssh      *SSHConnection
}

func NewUNAS(id, name string, cfg *config.UNASSystemConfig) *UNASCollector {
	return &UNASCollector{
		id:   id,
		name: name,
		ssh:  NewSSHConnection(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.KeyPath),
	}
}

func parseSize(s string) int64 {
	upper := strings.ToUpper(s)
	multipliers := []struct {
		suf  string
		mult float64
	}{
		{"T", 1024 * 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"M", 1024 * 1024},
		{"K", 1024},
	}
	for _, m := range multipliers {
		if strings.HasSuffix(upper, m.suf) {
			num := strings.TrimSuffix(upper, m.suf)
			if v, err := strconv.ParseFloat(num, 64); err == nil {
				return int64(v * m.mult)
			}
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return int64(v)
	}
	return 0
}

func parseZpoolList(out string) []models.StoragePool {
	var pools []models.StoragePool
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return pools
	}
	healthCandidates := map[string]struct{}{
		"ONLINE": {}, "DEGRADED": {}, "FAULTED": {}, "OFFLINE": {},
	}
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) < 5 {
			continue
		}
		total := parseSize(parts[1])
		used := parseSize(parts[2])
		available := parseSize(parts[3])
		health := parts[4]
		for i := len(parts) - 1; i >= 0; i-- {
			if _, ok := healthCandidates[strings.ToUpper(parts[i])]; ok {
				health = parts[i]
				break
			}
		}
		var pct float64
		if total > 0 {
			pct = float64(used) / float64(total) * 100.0
		}
		pools = append(pools, models.StoragePool{
			Name:      parts[0],
			Status:    strings.ToLower(health),
			Total:     total,
			Used:      used,
			Available: available,
			Percent:   pct,
		})
	}
	return pools
}

func parseDiskInfo(lsblk string, smartByName map[string]string) []models.StorageDisk {
	var disks []models.StorageDisk
	for _, line := range strings.Split(lsblk, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		name := parts[0]
		if !strings.HasPrefix(name, "sd") && !strings.HasPrefix(name, "nvme") {
			continue
		}
		disk := models.StorageDisk{
			Name:   name,
			Model:  "Unknown",
			Serial: "Unknown",
			Status: "healthy",
		}
		if smart, ok := smartByName[name]; ok {
			parseSmartctlInto(&disk, smart)
		}
		disks = append(disks, disk)
	}
	return disks
}

func parseSmartctlInto(disk *models.StorageDisk, smart string) {
	for _, line := range strings.Split(smart, "\n") {
		switch {
		case strings.Contains(line, "Model") || strings.Contains(line, "Device Model"):
			if i := strings.LastIndex(line, ":"); i >= 0 {
				disk.Model = strings.TrimSpace(line[i+1:])
			}
		case strings.Contains(line, "Serial"):
			if i := strings.LastIndex(line, ":"); i >= 0 {
				disk.Serial = strings.TrimSpace(line[i+1:])
			}
		case strings.Contains(line, "Temperature") && strings.Contains(line, "Celsius"):
			fields := strings.Fields(line)
			// SMART attribute table: ID … VALUE WORST THRESH … RAW_VALUE
			dashIdx := -1
			for i, f := range fields {
				if f == "-" {
					dashIdx = i
					break
				}
			}
			if dashIdx >= 0 && dashIdx+1 < len(fields) {
				numStr := stripNonNumeric(fields[dashIdx+1])
				if v, err := strconv.ParseFloat(numStr, 64); err == nil {
					disk.Temperature = &v
				}
			} else if len(fields) > 9 {
				if v, err := strconv.ParseFloat(fields[9], 64); err == nil {
					disk.Temperature = &v
				}
			}
		case strings.Contains(line, "SMART overall-health"):
			if strings.Contains(line, "PASSED") {
				disk.Status = "healthy"
			} else {
				disk.Status = "unhealthy"
			}
		}
	}
}

func stripNonNumeric(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		} else {
			break
		}
	}
	return b.String()
}

func unasStatus(m models.UNASMetrics) models.SystemStatus {
	for _, p := range m.Pools {
		if p.Status == "faulted" {
			return models.StatusCritical
		}
		if p.Status == "degraded" {
			return models.StatusWarning
		}
		if p.Percent > 90 {
			return models.StatusWarning
		}
	}
	for _, d := range m.Disks {
		if d.Status == "unhealthy" {
			return models.StatusWarning
		}
		if d.Temperature != nil && *d.Temperature > 50 {
			return models.StatusWarning
		}
	}
	return models.StatusHealthy
}

func diskNamesFromLsblk(lsblk string) []string {
	var names []string
	lines := strings.Split(lsblk, "\n")
	if len(lines) < 2 {
		return names
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if strings.HasPrefix(name, "sd") || strings.HasPrefix(name, "nvme") {
			names = append(names, name)
			if len(names) >= 10 {
				break
			}
		}
	}
	return names
}

func dfFallbackPool(df string) []models.StoragePool {
	type candidate struct {
		size int64
		pool models.StoragePool
	}
	seen := make(map[int64]struct{})
	var candidates []candidate
	lines := strings.Split(df, "\n")
	if len(lines) < 2 {
		return nil
	}
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		mount := parts[5]
		isStorage := mount == "/" || mount == "/volume1" || mount == "/mnt" || mount == "/data" ||
			(strings.HasPrefix(mount, "/volume/") && !strings.Contains(mount, "/.srv") && !strings.Contains(mount, "/."))
		if !isStorage {
			continue
		}
		total, _ := strconv.ParseInt(parts[1], 10, 64)
		if total > 1_000_000_000 {
			if _, ok := seen[total]; ok {
				continue
			}
		}
		seen[total] = struct{}{}
		used, _ := strconv.ParseInt(parts[2], 10, 64)
		avail, _ := strconv.ParseInt(parts[3], 10, 64)
		pct, _ := strconv.ParseFloat(strings.TrimSuffix(parts[4], "%"), 64)
		candidates = append(candidates, candidate{
			size: total,
			pool: models.StoragePool{
				Name:      mount,
				Status:    "online",
				Total:     total,
				Used:      used,
				Available: avail,
				Percent:   pct,
			},
		})
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].size > candidates[j].size })
	return []models.StoragePool{candidates[0].pool}
}

func (c *UNASCollector) Collect(ctx context.Context) *models.SystemMetrics {
	zpool, err := c.ssh.Run(ctx, "zpool list 2>/dev/null || true")
	if err != nil {
		c.ssh.Close()
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnas, err.Error())
	}
	df, err := c.ssh.Run(ctx, "df -B1 2>/dev/null")
	if err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnas, err.Error())
	}
	lsblk, err := c.ssh.Run(ctx, "lsblk -d -o NAME,SIZE,MODEL 2>/dev/null || true")
	if err != nil {
		return models.NewErrorMetrics(c.id, c.name, models.TypeUnas, err.Error())
	}

	smartByName := make(map[string]string)
	for _, name := range diskNamesFromLsblk(lsblk) {
		cmd := "sudo smartctl -a /dev/" + name + " 2>/dev/null || smartctl -a /dev/" + name + " 2>/dev/null || true"
		if out, err := c.ssh.Run(ctx, cmd); err == nil && out != "" {
			smartByName[name] = out
		}
	}

	pools := parseZpoolList(zpool)
	if len(pools) == 0 {
		pools = dfFallbackPool(df)
	}
	disks := parseDiskInfo(lsblk, smartByName)

	var total, used int64
	for _, p := range pools {
		total += p.Total
		used += p.Used
	}

	m := models.UNASMetrics{
		Pools:         pools,
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

func (c *UNASCollector) Close() { c.ssh.Close() }
