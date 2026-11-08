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

const linuxCombinedCmd = `echo "===STAT===" && cat /proc/stat && ` +
	`echo "===MEMINFO===" && cat /proc/meminfo && ` +
	`echo "===DF===" && df -B1 && ` +
	`echo "===NETDEV===" && cat /proc/net/dev && ` +
	`echo "===UPTIME===" && cat /proc/uptime && ` +
	`echo "===LOADAVG===" && cat /proc/loadavg && ` +
	`echo "===SENSORS===" && (sensors 2>/dev/null || true) && ` +
	`echo "===THERMAL===" && (for zone in /sys/class/thermal/thermal_zone*/; do ` +
	`name=$(cat ${zone}type 2>/dev/null || echo zone${zone##*zone}); ` +
	`temp=$(cat ${zone}temp 2>/dev/null); ` +
	`[ -n "$temp" ] && echo "$name:$temp"; done 2>/dev/null || true) && ` +
	// find, not a glob: under zsh (nomatch) an unmatched glob aborts the whole command.
	`echo "===FANS===" && (for f in $(find -L /sys/class/hwmon/ -mindepth 2 -maxdepth 2 -name 'fan*_input' 2>/dev/null); do [ -r "$f" ] || continue; ` +
	`label=$(cat ${f%_input}_label 2>/dev/null || echo "$(cat ${f%/*}/name 2>/dev/null)/$(basename ${f%_input})"); ` +
	`echo "$label:$(cat $f)"; done 2>/dev/null || true) && ` +
	`echo "===PAGESIZE===" && (getconf PAGESIZE 2>/dev/null || echo 4096) && ` +
	`echo "===PROCS===" && (cat /proc/[0-9]*/stat 2>/dev/null || true)`

// topProcessCount is how many processes (by CPU) each poll reports.
const topProcessCount = 10

type cpuStat struct {
	user, nice, system, idle, iowait, irq, softirq int64
}

func (c cpuStat) total() int64 {
	return c.user + c.nice + c.system + c.idle + c.iowait + c.irq + c.softirq
}
func (c cpuStat) idleTotal() int64 { return c.idle + c.iowait }

type netStat struct {
	bytesRecv int64
	bytesSent int64
}

type LinuxCollector struct {
	id, name    string
	ssh         *SSHConnection
	prevCPU     map[string]cpuStat
	prevNet     *netStat
	prevTime    float64
	hasPrevTime bool
	prevProcs   map[int]int64 // pid → utime+stime ticks
}

type procSample struct {
	pid      int
	name     string
	ticks    int64 // utime + stime, in USER_HZ
	rssPages int64
}

func NewLinux(id, name string, cfg *config.LinuxSystemConfig) *LinuxCollector {
	return &LinuxCollector{
		id:   id,
		name: name,
		ssh:  NewSSHConnection(cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.KeyPath),
	}
}

func parseCombinedOutput(output string) map[string]string {
	sections := make(map[string]string)
	var current string
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if strings.HasPrefix(line, "===") && strings.HasSuffix(line, "===") {
			if current != "" {
				sections[current] = strings.Join(lines, "\n")
			}
			current = strings.Trim(line, "=")
			lines = lines[:0]
			continue
		}
		if current != "" {
			lines = append(lines, line)
		}
	}
	if current != "" {
		sections[current] = strings.Join(lines, "\n")
	}
	return sections
}

func parseCPUStats(stat string) map[string]cpuStat {
	stats := make(map[string]cpuStat)
	for _, line := range strings.Split(stat, "\n") {
		if !strings.HasPrefix(line, "cpu") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 8 {
			continue
		}
		nums := make([]int64, 0, 7)
		for _, p := range parts[1:8] {
			n, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				break
			}
			nums = append(nums, n)
		}
		if len(nums) != 7 {
			continue
		}
		stats[parts[0]] = cpuStat{
			user: nums[0], nice: nums[1], system: nums[2], idle: nums[3],
			iowait: nums[4], irq: nums[5], softirq: nums[6],
		}
	}
	return stats
}

func calculateCPUPercent(current, previous map[string]cpuStat) (total float64, cores []float64) {
	if previous == nil {
		return 0, nil
	}
	calc := func(cur, prv cpuStat) float64 {
		totalDiff := cur.total() - prv.total()
		idleDiff := cur.idleTotal() - prv.idleTotal()
		if totalDiff == 0 {
			return 0
		}
		return float64(totalDiff-idleDiff) / float64(totalDiff) * 100.0
	}
	if cur, ok := current["cpu"]; ok {
		if prv, ok := previous["cpu"]; ok {
			total = calc(cur, prv)
		}
	}
	keys := make([]string, 0, len(current))
	for k := range current {
		if strings.HasPrefix(k, "cpu") && k != "cpu" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if cur, ok := current[k]; ok {
			if prv, ok := previous[k]; ok {
				cores = append(cores, calc(cur, prv))
			}
		}
	}
	return
}

func parseMeminfo(meminfo string) models.MemoryMetrics {
	info := make(map[string]int64)
	for _, line := range strings.Split(meminfo, "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		v, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			continue
		}
		info[key] = v * 1024 // kB → bytes
	}
	total := info["MemTotal"]
	available := info["MemAvailable"]
	used := total - available
	var percent float64
	if total > 0 {
		percent = float64(used) / float64(total) * 100.0
	}
	return models.MemoryMetrics{Total: total, Used: used, Available: available, Percent: percent}
}

func parseDF(df string) []models.DiskMetrics {
	var disks []models.DiskMetrics
	lines := strings.Split(df, "\n")
	if len(lines) < 2 {
		return disks
	}
	for _, line := range lines[1:] {
		parts := strings.Fields(line)
		if len(parts) < 6 || !strings.HasPrefix(parts[0], "/") {
			continue
		}
		total, err1 := strconv.ParseInt(parts[1], 10, 64)
		used, err2 := strconv.ParseInt(parts[2], 10, 64)
		free, err3 := strconv.ParseInt(parts[3], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		pct, _ := strconv.ParseFloat(strings.TrimSuffix(parts[4], "%"), 64)
		disks = append(disks, models.DiskMetrics{
			Mount:   parts[5],
			Total:   total * 1024,
			Used:    used * 1024,
			Free:    free * 1024,
			Percent: pct,
		})
	}
	return disks
}

func parseNetStats(netdev string) netStat {
	var s netStat
	lines := strings.Split(netdev, "\n")
	if len(lines) < 3 {
		return s
	}
	for _, line := range lines[2:] {
		ifacePart, valuesPart, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		iface := strings.TrimSpace(ifacePart)
		if iface == "lo" || iface == "docker0" || strings.HasPrefix(iface, "veth") {
			continue
		}
		values := strings.Fields(valuesPart)
		if len(values) <= 8 {
			continue
		}
		recv, err1 := strconv.ParseInt(values[0], 10, 64)
		sent, err2 := strconv.ParseInt(values[8], 10, 64)
		if err1 == nil && err2 == nil {
			s.bytesRecv += recv
			s.bytesSent += sent
		}
	}
	return s
}

func parseSensors(out string) []models.TemperatureMetrics {
	var temps []models.TemperatureMetrics
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, ":") || !strings.Contains(line, "°C") {
			continue
		}
		labelPart, tempPart, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		label := strings.TrimSpace(labelPart)
		tempStr := strings.TrimSpace(strings.SplitN(tempPart, "°C", 2)[0])
		tempStr = strings.TrimPrefix(tempStr, "+")
		t, err := strconv.ParseFloat(tempStr, 64)
		if err != nil {
			continue
		}
		temps = append(temps, models.TemperatureMetrics{Label: label, Current: t})
	}
	return temps
}

func parseThermalZones(out string) []models.TemperatureMetrics {
	var temps []models.TemperatureMetrics
	for _, line := range strings.Split(out, "\n") {
		label, tempStr, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimSpace(tempStr), 64)
		if err != nil {
			continue
		}
		temps = append(temps, models.TemperatureMetrics{
			Label:   strings.TrimSpace(label),
			Current: v / 1000.0,
		})
	}
	return temps
}

// parseFans reads "label:rpm" lines. The label may itself contain ':'.
func parseFans(out string) []models.FanMetrics {
	fans := []models.FanMetrics{}
	for _, line := range strings.Split(out, "\n") {
		i := strings.LastIndexByte(line, ':')
		if i <= 0 {
			continue
		}
		rpm, err := strconv.ParseInt(strings.TrimSpace(line[i+1:]), 10, 64)
		if err != nil {
			continue
		}
		fans = append(fans, models.FanMetrics{Label: strings.TrimSpace(line[:i]), RPM: rpm})
	}
	return fans
}

// parseProcStats parses concatenated /proc/[pid]/stat lines. The comm field
// is wrapped in parentheses and may contain spaces or ')', so the last ')'
// marks its end.
func parseProcStats(out string) map[int]procSample {
	procs := make(map[int]procSample)
	for _, line := range strings.Split(out, "\n") {
		open := strings.IndexByte(line, '(')
		end := strings.LastIndexByte(line, ')')
		if open <= 0 || end < open {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(line[:open]))
		if err != nil {
			continue
		}
		// rest[0] is field 3 (state), so field n is rest[n-3].
		rest := strings.Fields(line[end+1:])
		if len(rest) < 22 {
			continue
		}
		utime, err1 := strconv.ParseInt(rest[11], 10, 64)
		stime, err2 := strconv.ParseInt(rest[12], 10, 64)
		rss, err3 := strconv.ParseInt(rest[21], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		procs[pid] = procSample{pid: pid, name: line[open+1 : end], ticks: utime + stime, rssPages: rss}
	}
	return procs
}

// topProcesses converts per-process tick deltas into top-style CPU percent
// (100% = one full core). totalDelta is the aggregate jiffies delta across
// all cores between the two polls, so totalDelta/cores is the elapsed time
// in ticks for one core.
func topProcesses(cur map[int]procSample, prev map[int]int64, totalDelta int64, cores int, pageSize, memTotal int64, n int) []models.ProcessMetrics {
	procs := make([]models.ProcessMetrics, 0, len(cur))
	for _, p := range cur {
		var cpu float64
		if prevTicks, ok := prev[p.pid]; ok && totalDelta > 0 && cores > 0 {
			cpu = float64(p.ticks-prevTicks) / (float64(totalDelta) / float64(cores)) * 100.0
		}
		mem := p.rssPages * pageSize
		var memPct float64
		if memTotal > 0 {
			memPct = float64(mem) / float64(memTotal) * 100.0
		}
		procs = append(procs, models.ProcessMetrics{
			PID: p.pid, Name: p.name, CPUPercent: cpu, MemoryUsage: mem, MemoryPercent: memPct,
		})
	}
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].CPUPercent != procs[j].CPUPercent {
			return procs[i].CPUPercent > procs[j].CPUPercent
		}
		return procs[i].MemoryUsage > procs[j].MemoryUsage
	})
	if len(procs) > n {
		procs = procs[:n]
	}
	return procs
}

func parseUptime(out string) int64 {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	return int64(f)
}

func linuxStatus(m models.LinuxMetrics) models.SystemStatus {
	if m.CPU.Percent > 90 || m.Memory.Percent > 95 {
		return models.StatusCritical
	}
	if m.CPU.Percent > 75 || m.Memory.Percent > 85 {
		return models.StatusWarning
	}
	for _, d := range m.Disks {
		if d.Percent > 95 {
			return models.StatusCritical
		}
		if d.Percent > 85 {
			return models.StatusWarning
		}
	}
	for _, t := range m.Temperatures {
		if t.Critical != nil && t.Current >= *t.Critical {
			return models.StatusCritical
		}
		if t.High != nil && t.Current >= *t.High {
			return models.StatusWarning
		}
	}
	return models.StatusHealthy
}

func (c *LinuxCollector) Collect(ctx context.Context) *models.SystemMetrics {
	currentTime := float64(time.Now().UnixNano()) / 1e9

	output, err := c.ssh.Run(ctx, linuxCombinedCmd)
	if err != nil {
		c.ssh.Close()
		return models.NewErrorMetrics(c.id, c.name, models.TypeLinux, err.Error())
	}

	sections := parseCombinedOutput(output)

	cpuStats := parseCPUStats(sections["STAT"])
	cpuPercent, cores := calculateCPUPercent(cpuStats, c.prevCPU)
	var totalDelta int64
	if prv, ok := c.prevCPU["cpu"]; ok {
		totalDelta = cpuStats["cpu"].total() - prv.total()
	}
	c.prevCPU = cpuStats

	memory := parseMeminfo(sections["MEMINFO"])
	pageSize, err := strconv.ParseInt(strings.TrimSpace(sections["PAGESIZE"]), 10, 64)
	if err != nil {
		pageSize = 4096
	}
	procSamples := parseProcStats(sections["PROCS"])
	processes := topProcesses(procSamples, c.prevProcs, totalDelta, len(cpuStats)-1, pageSize, memory.Total, topProcessCount)
	c.prevProcs = make(map[int]int64, len(procSamples))
	for pid, p := range procSamples {
		c.prevProcs[pid] = p.ticks
	}

	var loadAvg []float64
	for i, p := range strings.Fields(sections["LOADAVG"]) {
		if i >= 3 {
			break
		}
		if v, err := strconv.ParseFloat(p, 64); err == nil {
			loadAvg = append(loadAvg, v)
		}
	}

	netCurrent := parseNetStats(sections["NETDEV"])
	var uploadRate, downloadRate float64
	if c.prevNet != nil && c.hasPrevTime {
		dt := currentTime - c.prevTime
		if dt > 0 {
			uploadRate = float64(netCurrent.bytesSent-c.prevNet.bytesSent) / dt
			downloadRate = float64(netCurrent.bytesRecv-c.prevNet.bytesRecv) / dt
		}
	}
	c.prevNet = &netCurrent
	c.prevTime = currentTime
	c.hasPrevTime = true

	var temps []models.TemperatureMetrics
	if s := strings.TrimSpace(sections["SENSORS"]); s != "" {
		temps = parseSensors(s)
	} else if t := strings.TrimSpace(sections["THERMAL"]); t != "" {
		temps = parseThermalZones(t)
	}

	linuxMetrics := models.LinuxMetrics{
		CPU: models.CPUMetrics{
			Percent: cpuPercent,
			Cores:   cores,
			LoadAvg: loadAvg,
		},
		Memory: memory,
		Disks:  parseDF(sections["DF"]),
		Network: models.NetworkMetrics{
			BytesSent:    netCurrent.bytesSent,
			BytesRecv:    netCurrent.bytesRecv,
			UploadRate:   uploadRate,
			DownloadRate: downloadRate,
		},
		Temperatures: temps,
		Fans:         parseFans(sections["FANS"]),
		Processes:    processes,
		Uptime:       parseUptime(sections["UPTIME"]),
	}

	return &models.SystemMetrics{
		ID:          c.id,
		Name:        c.name,
		Type:        models.TypeLinux,
		Status:      linuxStatus(linuxMetrics),
		LastUpdated: time.Now().UTC(),
		Metrics:     linuxMetrics,
	}
}

func (c *LinuxCollector) Close() { c.ssh.Close() }
