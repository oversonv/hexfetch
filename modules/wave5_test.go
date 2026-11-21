package modules

import "testing"

func TestParseProcStat(t *testing.T) {
	idle, total, ok := parseProcStat(readFixture(t, "proc/stat"))
	if !ok {
		t.Fatal("expected ok")
	}
	wantIdle := uint64(538039 + 18797)
	wantTotal := uint64(6211 + 763 + 1954 + 538039 + 18797 + 1108)
	if idle != wantIdle {
		t.Errorf("idle = %d, want %d", idle, wantIdle)
	}
	if total != wantTotal {
		t.Errorf("total = %d, want %d", total, wantTotal)
	}
	if _, _, ok := parseProcStat([]byte("garbage\n")); ok {
		t.Error("expected not ok for garbage")
	}
	if _, _, ok := parseProcStat([]byte("cpu 1 2\n")); ok {
		t.Error("expected not ok for short cpu line")
	}
	if idle, total, ok := parseProcStat([]byte("cpu 10 2 3 40\n")); !ok || idle != 40 || total != 55 {
		t.Errorf("parseProcStat without iowait = %d, %d, %v", idle, total, ok)
	}
	if idle, total, ok := parseProcStat([]byte("cpu 100 10 20 200 30 1 2 3 50 5\n")); !ok || idle != 230 || total != 366 {
		t.Errorf("parseProcStat with guest counters = %d, %d, %v", idle, total, ok)
	}
}

func TestCPUUsagePercent(t *testing.T) {
	before := sample{idle: 100, total: 200}
	after := sample{idle: 190, total: 300}
	if got := cpuUsagePercent(before, after); got != "10%" {
		t.Errorf("cpuUsagePercent = %q, want 10%%", got)
	}
	if got := cpuUsagePercent(before, before); got != "" {
		t.Errorf("cpuUsagePercent no-delta = %q", got)
	}
	if got := cpuUsagePercent(after, before); got != "" {
		t.Errorf("cpuUsagePercent backwards = %q", got)
	}
	afterBogus := sample{idle: 500, total: 500}
	if got := cpuUsagePercent(before, afterBogus); got != "" {
		t.Errorf("cpuUsagePercent idle>total = %q", got)
	}
}

func TestChassisTypeNames(t *testing.T) {
	names := chassisTypeNames("9 10 1")
	if len(names) != 3 || names[0] != "Laptop" || names[1] != "Notebook" || names[2] != "Other" {
		t.Errorf("chassisTypeNames = %v", names)
	}
	if got := chassisTypeNames("9 10 9"); len(got) != 2 || got[0] != "Laptop" || got[1] != "Notebook" {
		t.Errorf("chassisTypeNames dedupe = %v", got)
	}
	if got := chassisTypeNames(""); len(got) != 0 {
		t.Errorf("chassisTypeNames empty = %v", got)
	}
	if got := chassisTypeNames("abc 3"); len(got) != 1 || got[0] != "Desktop" {
		t.Errorf("chassisTypeNames invalid = %v", got)
	}
}

func TestParseNvidiaUtilization(t *testing.T) {
	percents := parseNvidiaUtilization([]byte("12 %\n45%\n101\nbad\n"))
	if len(percents) != 2 || percents[0] != "12%" || percents[1] != "45%" {
		t.Errorf("percents = %v", percents)
	}
	if got := parseNvidiaUtilization(nil); len(got) != 0 {
		t.Errorf("empty = %v", got)
	}
}

func TestValidPercent(t *testing.T) {
	tests := map[string]bool{
		"0": true, "100": true, "45": true, "-1": false, "101": false, "": false, "x": false,
	}
	for input, want := range tests {
		if got := validPercent(input); got != want {
			t.Errorf("validPercent(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	tests := map[int]string{
		0:          "0 days",
		5:          "5 days",
		30:         "1 month",
		45:         "1 month, 15 days",
		400:        "1 year, 1 month",
		800:        "2 years, 2 months",
		365*2 + 45: "2 years, 1 month",
	}
	for days, want := range tests {
		if got := formatAge(days); got != want {
			t.Errorf("formatAge(%d) = %q, want %q", days, got, want)
		}
	}
}
