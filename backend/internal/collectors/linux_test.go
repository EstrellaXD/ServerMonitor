package collectors

import (
	"math"
	"testing"
)

func TestParseCombinedOutput_extractsNamedSections(t *testing.T) {
	in := "===STAT===\ncpu 1 2 3 4 5 6 7\n===MEMINFO===\nMemTotal: 100 kB\n"
	got := parseCombinedOutput(in)
	if got["STAT"] != "cpu 1 2 3 4 5 6 7" {
		t.Fatalf("STAT: %q", got["STAT"])
	}
	if got["MEMINFO"] != "MemTotal: 100 kB" {
		t.Fatalf("MEMINFO: %q", got["MEMINFO"])
	}
}

func TestParseCPUStats_parsesTotalAndCores(t *testing.T) {
	in := "cpu  100 200 300 400 50 10 5\ncpu0 50 50 50 50 50 10 5\ncpu1 50 150 250 350 0 0 0\n"
	stats := parseCPUStats(in)
	if len(stats) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(stats))
	}
	if stats["cpu"].user != 100 || stats["cpu"].softirq != 5 {
		t.Fatalf("cpu wrong: %+v", stats["cpu"])
	}
}

func TestCalculateCPUPercent_returnsZeroWhenNoHistory(t *testing.T) {
	current := map[string]cpuStat{"cpu": {user: 1, system: 1, idle: 1}}
	total, cores := calculateCPUPercent(current, nil)
	if total != 0 || cores != nil {
		t.Fatalf("expected zeros, got %v %v", total, cores)
	}
}

func TestCalculateCPUPercent_busyDeltaProducesNonZero(t *testing.T) {
	prev := map[string]cpuStat{
		"cpu":  {user: 0, system: 0, idle: 100},
		"cpu0": {user: 0, system: 0, idle: 100},
	}
	cur := map[string]cpuStat{
		"cpu":  {user: 30, system: 20, idle: 150},
		"cpu0": {user: 30, system: 20, idle: 150},
	}
	total, cores := calculateCPUPercent(cur, prev)
	want := 50.0
	if math.Abs(total-want) > 0.01 {
		t.Fatalf("total: got %v want %v", total, want)
	}
	if len(cores) != 1 || math.Abs(cores[0]-want) > 0.01 {
		t.Fatalf("cores: %v", cores)
	}
}

func TestParseMeminfo_convertskBToBytes(t *testing.T) {
	in := "MemTotal:       100 kB\nMemAvailable:    40 kB\n"
	m := parseMeminfo(in)
	if m.Total != 100*1024 || m.Available != 40*1024 || m.Used != 60*1024 {
		t.Fatalf("got %+v", m)
	}
	if math.Abs(m.Percent-60.0) > 0.001 {
		t.Fatalf("percent: %v", m.Percent)
	}
}

func TestParseDF_keepsBlockDevicesAndConvertskBToBytes(t *testing.T) {
	in := "Filesystem 1B-blocks Used Available Use% Mounted on\n" +
		"/dev/sda1 1024 512 512 50% /\n" +
		"tmpfs 100 50 50 50% /tmp\n"
	disks := parseDF(in)
	// tmpfs is intentionally filtered out (parity with Rust collector).
	if len(disks) != 1 {
		t.Fatalf("got %d disks", len(disks))
	}
	if disks[0].Mount != "/" || disks[0].Total != 1024*1024 {
		t.Fatalf("first: %+v", disks[0])
	}
}

func TestParseNetStats_skipsLoopbackAndVeth(t *testing.T) {
	in := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes packets ...
    lo: 100 0 0 0 0 0 0 0 200 0
docker0: 50 0 0 0 0 0 0 0 100 0
veth123: 1 0 0 0 0 0 0 0 2 0
  eth0: 1000 5 0 0 0 0 0 0 2000 10`
	s := parseNetStats(in)
	if s.bytesRecv != 1000 || s.bytesSent != 2000 {
		t.Fatalf("got %+v", s)
	}
}

func TestParseSensors_handlesPlusPrefixAndCelsius(t *testing.T) {
	in := "Core 0:        +45.0°C  (high = +80.0°C, crit = +100.0°C)\nfan1:           1200 RPM\n"
	temps := parseSensors(in)
	if len(temps) != 1 || temps[0].Label != "Core 0" || temps[0].Current != 45.0 {
		t.Fatalf("got %+v", temps)
	}
}

func TestParseThermalZones_dividesMillicelsius(t *testing.T) {
	in := "x86_pkg_temp:48000\nacpitz:39500\n"
	temps := parseThermalZones(in)
	if len(temps) != 2 || temps[0].Current != 48.0 || temps[1].Current != 39.5 {
		t.Fatalf("got %+v", temps)
	}
}

func TestParseUptime_takesFirstFloatAsSeconds(t *testing.T) {
	if got := parseUptime("12345.67 8910.11"); got != 12345 {
		t.Fatalf("got %d", got)
	}
	if got := parseUptime(""); got != 0 {
		t.Fatalf("empty should be 0, got %d", got)
	}
}

func TestParseFans_keepsLabelWithColonAndSkipsInvalid(t *testing.T) {
	got := parseFans("pwmfan/fan1:4011\nnct6775: CPU Fan:1200\ngarbage\nbad:abc\n")
	if len(got) != 2 {
		t.Fatalf("expected 2 fans, got %+v", got)
	}
	if got[0].Label != "pwmfan/fan1" || got[0].RPM != 4011 {
		t.Fatalf("fan0: %+v", got[0])
	}
	if got[1].Label != "nct6775: CPU Fan" || got[1].RPM != 1200 {
		t.Fatalf("fan1: %+v", got[1])
	}
}

func TestParseProcStats_handlesCommWithSpacesAndParens(t *testing.T) {
	// Fields after comm: state ppid pgrp session tty tpgid flags minflt cminflt
	// majflt cmajflt utime stime cutime cstime priority nice threads itreal
	// starttime vsize rss ...
	in := "42 (Web Content (x)) S 1 42 42 0 -1 0 0 0 0 0 30 12 0 0 20 0 1 0 100 1000 250 18446744073709551615\n" +
		"7 (short) R 1 7 7 0 -1 0 0 0 0 0 5 5 0 0 20 0 1 0 100 1000\n"
	got := parseProcStats(in)
	p, ok := got[42]
	if !ok || p.name != "Web Content (x)" || p.ticks != 42 || p.rssPages != 250 {
		t.Fatalf("pid 42: %+v", p)
	}
	if _, ok := got[7]; ok {
		t.Fatalf("truncated line must be skipped")
	}
}

func TestTopProcesses_computesPerCoreCPUFromTickDelta(t *testing.T) {
	cur := map[int]procSample{
		1: {pid: 1, name: "busy", ticks: 150, rssPages: 10},
		2: {pid: 2, name: "idle", ticks: 10, rssPages: 100},
		3: {pid: 3, name: "new", ticks: 999, rssPages: 1},
	}
	prev := map[int]int64{1: 50, 2: 10}
	// 4 cores, 400 aggregate jiffies → 100 ticks per core; busy used 100 → 100%.
	got := topProcesses(cur, prev, 400, 4, 4096, 4096*1000, 2)
	if len(got) != 2 {
		t.Fatalf("expected top 2, got %d", len(got))
	}
	if got[0].Name != "busy" || math.Abs(got[0].CPUPercent-100) > 1e-9 {
		t.Fatalf("first: %+v", got[0])
	}
	// Ties at 0% CPU fall back to memory; "new" has no history so its CPU is 0.
	if got[1].Name != "idle" || got[1].MemoryUsage != 409600 || math.Abs(got[1].MemoryPercent-10) > 1e-9 {
		t.Fatalf("second: %+v", got[1])
	}
}
