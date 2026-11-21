package modules

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}

func TestParseCPUInfoX86(t *testing.T) {
	info := parseCPUInfo(readFixture(t, "proc/cpuinfo_x86"))
	if info.model != "Intel Core i7-9750H CPU @ 2.60GHz" {
		t.Errorf("model = %q", info.model)
	}
	if info.logical != 4 {
		t.Errorf("logical = %d, want 4", info.logical)
	}
	if info.physical != 4 {
		t.Errorf("physical = %d, want 4", info.physical)
	}
	if info.freqHz != 3_600_000_000 {
		t.Errorf("freqHz = %d, want 3600000000", info.freqHz)
	}
}

func TestParseCPUInfoARM(t *testing.T) {
	// fastfetch uses the "Hardware" line on ARM; the header "Processor"
	// line is only a last resort.
	info := parseCPUInfo(readFixture(t, "proc/cpuinfo_arm"))
	if info.model != "BCM2835" {
		t.Errorf("model = %q", info.model)
	}
	if info.logical != 4 {
		t.Errorf("logical = %d, want 4", info.logical)
	}
	if info.physical != 0 {
		t.Errorf("physical = %d, want 0", info.physical)
	}
}

func TestParseCPUInfoARMDoesNotInferModelFromPartIDs(t *testing.T) {
	data := []byte(strings.Join([]string{
		"Processor	: AArch64 Processor rev 1 (aarch64)",
		"processor	: 0",
		"CPU implementer	: 0x51",
		"CPU part	: 0x801",
		"",
	}, "\n"))
	info := parseCPUInfo(data)
	if info.model != "AArch64 Processor rev 1 (aarch64)" {
		t.Errorf("model = %q, want the kernel-provided Processor value", info.model)
	}
	if info.logical != 1 {
		t.Errorf("logical = %d, want 1", info.logical)
	}
}

func TestParseCPUInfoEmpty(t *testing.T) {
	info := parseCPUInfo(nil)
	if info.model != "" || info.logical != 0 {
		t.Errorf("unexpected info: %+v", info)
	}
}

func TestParseMeminfo(t *testing.T) {
	info := parseMeminfo(readFixture(t, "proc/meminfo"))
	tests := map[string]uint64{
		"MemTotal":     16384000,
		"MemAvailable": 8192000,
		"SwapTotal":    4194304,
		"SwapFree":     2097152,
		"Cached":       4096000,
	}
	for key, want := range tests {
		if got := info[key]; got != want {
			t.Errorf("%s = %d, want %d", key, got, want)
		}
	}
}

func TestParseMeminfoInvalid(t *testing.T) {
	info := parseMeminfo([]byte("garbage\nMemTotal: nope kB\nMemFree: 10 kB\n"))
	if info["MemFree"] != 10 {
		t.Errorf("MemFree = %d, want 10", info["MemFree"])
	}
	if _, ok := info["MemTotal"]; ok {
		t.Error("MemTotal should be absent")
	}
}

func TestParseLoadAvg(t *testing.T) {
	one, five, fifteen, err := parseLoadAvg(readFixture(t, "proc/loadavg"))
	if err != nil {
		t.Fatal(err)
	}
	if one != "0.52" || five != "0.58" || fifteen != "0.59" {
		t.Errorf("got %s %s %s", one, five, fifteen)
	}
	if _, _, _, err := parseLoadAvg([]byte("1.0")); err == nil {
		t.Error("expected error for short loadavg")
	}
}

func TestFormatSysinfoLoads(t *testing.T) {
	info := syscall.Sysinfo_t{Loads: [3]uint64{1 << 16, 123456 << 16, (50 << 16) + (1 << 15)}}
	one, five, fifteen := formatSysinfoLoads(&info)
	if one != "1.00" || five != "123456.00" || fifteen != "50.50" {
		t.Errorf("formatSysinfoLoads() = %q %q %q", one, five, fifteen)
	}
}

func TestAggregateCache(t *testing.T) {
	levels := aggregateCache([]cacheEntry{
		{level: 1, size: 32 * 1024},
		{level: 1, size: 32 * 1024},
		{level: 2, size: 256 * 1024},
		{level: 3, size: 8 * 1024 * 1024},
	})
	if levels[1] != 64*1024 {
		t.Errorf("L1 = %d", levels[1])
	}
	if levels[3] != 8*1024*1024 {
		t.Errorf("L3 = %d", levels[3])
	}
}

func TestCollectCPUCache(t *testing.T) {
	root := t.TempDir()
	cacheBase := filepath.Join(root, "devices/system/cpu/cpu0/cache")
	entries := []struct {
		dir, level, size string
	}{
		{"index0", "1", "32K"},
		{"index1", "1", "32K"},
		{"index2", "2", "256K"},
		{"index3", "3", "8M"},
	}
	for _, entry := range entries {
		dir := filepath.Join(cacheBase, entry.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, "level"), []byte(entry.level), 0o644)
		os.WriteFile(filepath.Join(dir, "size"), []byte(entry.size), 0o644)
	}
	old := sysRoot
	sysRoot = root
	defer func() { sysRoot = old }()

	levels, err := collectCPUCache()
	if err != nil {
		t.Fatal(err)
	}
	if levels[1] != 64*1024 || levels[2] != 256*1024 || levels[3] != 8*1024*1024 {
		t.Errorf("levels = %v", levels)
	}
}

func TestDMIValues(t *testing.T) {
	old := sysRoot
	sysRoot = "testdata/sys"
	defer func() { sysRoot = old }()

	tests := map[string]string{
		"product_name":    "XPS 13 9310",
		"sys_vendor":      "Dell Inc.",
		"bios_date":       "01/01/2024",
		"bios_version":    "1.4.0",
		"chassis_type":    "10",
		"product_version": "1.2.3",
	}
	for field, want := range tests {
		if got := dmiValue(field); got != want {
			t.Errorf("dmiValue(%q) = %q, want %q", field, got, want)
		}
	}
	if got := dmiValue("missing_field"); got != "" {
		t.Errorf("missing field = %q", got)
	}
}

func TestDMIModulesDegrade(t *testing.T) {
	old := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = old }()
	for _, module := range []Module{host{}, board{}, bios{}, product{}, chassis{}} {
		if entries, err := module.Run(Context{}); err == nil || len(entries) != 0 {
			t.Errorf("%s: expected skip, got %v, %v", module.Name(), entries, err)
		}
	}
}

func TestChassisTypeName(t *testing.T) {
	if got := chassisTypeName(10); got != "Notebook" {
		t.Errorf("chassis 10 = %q", got)
	}
	if got := chassisTypeName(99); got != "" {
		t.Errorf("chassis 99 = %q", got)
	}
}

func TestProcessesCount(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"1", "42", "999", "self", "net"} {
		os.Mkdir(filepath.Join(root, name), 0o755)
	}
	old := procRoot
	procRoot = root
	defer func() { procRoot = old }()

	entries, err := (processes{}).Run(Context{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Value != "3" {
		t.Errorf("entries = %v", entries)
	}
}

func TestMemoryAndSwapSkipWithoutData(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	meminfoOnce = sync.Once{}
	meminfoData = nil
	meminfoErr = nil
	defer func() {
		procRoot = old
		meminfoOnce = sync.Once{}
		meminfoData = nil
		meminfoErr = nil
	}()

	if _, err := (memory{}).Run(Context{}); err == nil {
		t.Error("memory should skip")
	}
	if _, err := (swap{}).Run(Context{}); err == nil {
		t.Error("swap should skip")
	}
}
