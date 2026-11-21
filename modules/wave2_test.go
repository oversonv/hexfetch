package modules

import (
	"strings"
	"testing"
)

func TestBatteryDischarging(t *testing.T) {
	info, ok := batteryFromFiles(map[string]string{
		"capacity":    "85",
		"status":      "Discharging",
		"energy_now":  "42500000",
		"energy_full": "50000000",
		"power_now":   "20000000",
	})
	if !ok {
		t.Fatal("expected battery data")
	}
	if info.percent != 85 {
		t.Errorf("percent = %d", info.percent)
	}
	if info.status != "Discharging" {
		t.Errorf("status = %q", info.status)
	}
	if !info.hasRemaining || info.remaining != 7650 {
		t.Errorf("remaining = %d (%v), want 7650", info.remaining, info.hasRemaining)
	}
}

func TestBatteryChargingFromCurrent(t *testing.T) {
	info, ok := batteryFromFiles(map[string]string{
		"status":      "Charging",
		"energy_now":  "10000000",
		"energy_full": "50000000",
		"current_now": "2000000",
		"voltage_now": "12000000",
	})
	if !ok {
		t.Fatal("expected battery data")
	}
	if info.percent != 20 {
		t.Errorf("percent = %d, want 20", info.percent)
	}
	if !info.hasRemaining || info.remaining != 6000 {
		t.Errorf("remaining = %d (%v), want 6000", info.remaining, info.hasRemaining)
	}
}

func TestBatteryEmpty(t *testing.T) {
	if _, ok := batteryFromFiles(map[string]string{"status": "Unknown"}); ok {
		t.Error("expected no battery data")
	}
}

func TestParseAsoundCards(t *testing.T) {
	cards := parseAsoundCards(readFixture(t, "proc/asound_cards"))
	if len(cards) != 2 {
		t.Fatalf("cards = %v", cards)
	}
	if cards[0] != "HDA Intel PCH" || cards[1] != "HDA ATI HDMI" {
		t.Errorf("cards = %v", cards)
	}
	if got := parseAsoundCards([]byte("garbage\n")); len(got) != 0 {
		t.Errorf("garbage = %v", got)
	}
}

func TestValidTemperature(t *testing.T) {
	tests := map[int64]bool{0: false, -5000: false, 45000: true, 200000: true, 250000: false}
	for input, want := range tests {
		if got := validTemperature(input); got != want {
			t.Errorf("validTemperature(%d) = %v, want %v", input, got, want)
		}
	}
}

func TestParseEDID(t *testing.T) {
	edid := make([]byte, 128)
	edid[0], edid[1], edid[7] = 0x00, 0xff, 0x00
	edid[8], edid[9] = 0x10, 0xac // DEL
	edid[16] = 30                 // 2020
	edid[21], edid[22] = 53, 30
	copy(edid[54:], []byte{0x00, 0x00, 0x00, 0xfc, 0x00})
	copy(edid[59:], []byte("U2419H\n"))

	info, ok := parseEDID(edid)
	if !ok {
		t.Fatal("expected valid EDID")
	}
	if info.manufacturer != "DEL" {
		t.Errorf("manufacturer = %q", info.manufacturer)
	}
	if info.name != "U2419H" {
		t.Errorf("name = %q", info.name)
	}
	if info.year != 2020 {
		t.Errorf("year = %d", info.year)
	}
	value := info.describe()
	if !strings.Contains(value, "DEL U2419H") || !strings.Contains(value, "2020") {
		t.Errorf("describe = %q", value)
	}
	if _, ok := parseEDID([]byte{0x00, 0xff}); ok {
		t.Error("short EDID should be rejected")
	}
}

func TestParseMounts(t *testing.T) {
	mounts := parseMounts(readFixture(t, "proc/mounts"))
	if len(mounts) != 6 {
		t.Fatalf("mounts = %v", mounts)
	}
	if mounts[4].mountpoint != "/mnt/my disk" {
		t.Errorf("unescaped mountpoint = %q", mounts[4].mountpoint)
	}
	if usableMount(mounts[1]) || usableMount(mounts[2]) || usableMount(mounts[3]) || usableMount(mounts[5]) {
		t.Error("pseudo filesystems should be filtered")
	}
	if !usableMount(mounts[0]) || !usableMount(mounts[4]) {
		t.Error("real filesystems should be kept")
	}
}

func TestParseDiskstats(t *testing.T) {
	read, written, ok := parseDiskstats(readFixture(t, "proc/diskstats"))
	if !ok {
		t.Fatal("expected diskstats")
	}
	wantRead := uint64(5678901+400000) * 512
	wantWritten := uint64(345678+800000) * 512
	if read != wantRead || written != wantWritten {
		t.Errorf("read/write = %d/%d, want %d/%d", read, written, wantRead, wantWritten)
	}
}

func TestWholeDiskName(t *testing.T) {
	tests := map[string]bool{
		"sda":       true,
		"sda1":      false,
		"nvme0n1":   true,
		"nvme0n1p1": false,
		"mmcblk0":   true,
		"mmcblk0p1": false,
		"loop0":     false,
		"dm-0":      false,
	}
	for input, want := range tests {
		if got := wholeDiskName(input); got != want {
			t.Errorf("wholeDiskName(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestDescribePhysicalDisk(t *testing.T) {
	if got := describePhysicalDisk("sda", "Samsung SSD 860", 500*1024*1024*1024, false); !strings.Contains(got, "Samsung SSD 860") || !strings.Contains(got, "SSD") {
		t.Errorf("describe = %q", got)
	}
	if got := describePhysicalDisk("sdb", "", 2*1024*1024*1024*1024, true); !strings.HasPrefix(got, "sdb") || !strings.Contains(got, "HDD") {
		t.Errorf("describe = %q", got)
	}
}

func TestGPUName(t *testing.T) {
	if got := gpuName("0x10de", "0x1f95"); got != "NVIDIA 0x1f95" {
		t.Errorf("gpuName = %q", got)
	}
	if got := gpuName("0x1234", ""); got != "QEMU" {
		t.Errorf("gpuName = %q", got)
	}
}
