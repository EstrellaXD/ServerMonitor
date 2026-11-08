package collectors

import (
	"math"
	"testing"
)

func TestParseSize_handlesTGMKSuffixes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"1024", 1024},
		{"1K", 1024},
		{"1.5M", int64(1.5 * 1024 * 1024)},
		{"2G", 2 * 1024 * 1024 * 1024},
		{"3T", 3 * 1024 * 1024 * 1024 * 1024},
	}
	for _, c := range cases {
		if got := parseSize(c.in); got != c.want {
			t.Errorf("parseSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseZpoolList_parsesHealthAndCapacity(t *testing.T) {
	in := `NAME    SIZE   ALLOC   FREE    HEALTH
tank    10T    4T      6T      ONLINE
spare   500G   250G    250G    DEGRADED
`
	pools := parseZpoolList(in)
	if len(pools) != 2 {
		t.Fatalf("got %d pools", len(pools))
	}
	if pools[0].Name != "tank" || pools[0].Status != "online" {
		t.Fatalf("first pool: %+v", pools[0])
	}
	if pools[1].Status != "degraded" {
		t.Fatalf("second pool status: %s", pools[1].Status)
	}
	wantPct := float64(pools[0].Used) / float64(pools[0].Total) * 100
	if math.Abs(pools[0].Percent-wantPct) > 0.01 {
		t.Fatalf("percent: %v want %v", pools[0].Percent, wantPct)
	}
}

func TestParseDiskInfo_filtersNonDataDisks(t *testing.T) {
	lsblk := "NAME SIZE MODEL\nsda 100G MyDisk\nsr0 1G CDROM\nnvme0n1 500G FastSSD\n"
	disks := parseDiskInfo(lsblk, map[string]string{})
	if len(disks) != 2 {
		t.Fatalf("got %d disks", len(disks))
	}
	if disks[0].Name != "sda" || disks[1].Name != "nvme0n1" {
		t.Fatalf("names: %+v", disks)
	}
}

func TestParseDiskInfo_appliesSmartctlAttributes(t *testing.T) {
	lsblk := "NAME\nsda\n"
	smart := map[string]string{
		"sda": `Device Model:     WDC WD40EFRX-68N
Serial Number:    WD-WCC4E2NM4XYZ
194 Temperature_Celsius     0x0022   123   115   000    Old_age   Always       -       38
SMART overall-health self-assessment test result: PASSED
`,
	}
	disks := parseDiskInfo(lsblk, smart)
	if len(disks) != 1 {
		t.Fatalf("got %d disks", len(disks))
	}
	d := disks[0]
	if d.Model != "WDC WD40EFRX-68N" {
		t.Errorf("model: %q", d.Model)
	}
	if d.Serial != "WD-WCC4E2NM4XYZ" {
		t.Errorf("serial: %q", d.Serial)
	}
	if d.Temperature == nil || *d.Temperature != 38 {
		t.Errorf("temperature: %v", d.Temperature)
	}
	if d.Status != "healthy" {
		t.Errorf("status: %q", d.Status)
	}
}

func TestDfFallbackPool_picksLargestStorageMount(t *testing.T) {
	in := `Filesystem 1B-blocks Used Available Use% Mounted on
/dev/sda1 5000000000 1000000000 4000000000 20% /
/dev/sdb1 50000000000 25000000000 25000000000 50% /volume1
/dev/sdc1 50000000000 10000000000 40000000000 20% /volume1
`
	pools := dfFallbackPool(in)
	if len(pools) != 1 {
		t.Fatalf("got %d pools", len(pools))
	}
	if pools[0].Name != "/volume1" || pools[0].Total != 50000000000 {
		t.Fatalf("got %+v", pools[0])
	}
}
