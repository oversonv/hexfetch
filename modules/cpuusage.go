//go:build linux

package modules

import (
	"strconv"
	"strings"
	"time"
)

type cpuusage struct{}

func init() { Register(cpuusage{}) }

func (cpuusage) Name() string { return "cpuusage" }

func (cpuusage) OptionSchema() map[string]OptionDefinition {
	return map[string]OptionDefinition{
		"interval_ms": {Kind: OptionInteger, Minimum: 1, Maximum: 2000, HasRange: true},
	}
}

func (cpuusage) Run(ctx Context) ([]Entry, error) {
	interval := 150 * time.Millisecond
	if ms := ctx.OptionInt("interval_ms", 150); ms > 0 && ms <= 2000 {
		interval = time.Duration(ms) * time.Millisecond
	}
	first, err := readProcStatSample()
	if err != nil {
		return nil, err
	}
	time.Sleep(interval)
	second, err := readProcStatSample()
	if err != nil {
		return nil, err
	}
	percent := cpuUsagePercent(first, second)
	if percent == "" {
		return nil, Unsupported("cpu usage unavailable")
	}
	return Single(ctx, "CPU Usage", percent), nil
}

func readProcStatSample() (sample, error) {
	data, err := readFile(procPath("stat"))
	if err != nil {
		return sample{}, err
	}
	idle, total, ok := parseProcStat(data)
	if !ok {
		return sample{}, Unsupported("cpu stat unavailable")
	}
	return sample{idle: idle, total: total}, nil
}

type sample struct {
	idle  uint64
	total uint64
}

// parseProcStat reads the aggregate cpu line: idle includes iowait.
func parseProcStat(data []byte) (idle, total uint64, ok bool) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] != "cpu" {
			continue
		}
		counters := fields[1:]
		if len(counters) > 8 {
			counters = counters[:8]
		}
		for _, field := range counters {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return 0, 0, false
			}
			total += value
		}
		firstIdle, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		if len(fields) > 5 {
			if iowait, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
				firstIdle += iowait
			}
		}
		return firstIdle, total, true
	}
	return 0, 0, false
}

func cpuUsagePercent(before, after sample) string {
	idleDelta := int64(after.idle) - int64(before.idle)
	totalDelta := int64(after.total) - int64(before.total)
	if totalDelta <= 0 || idleDelta < 0 || idleDelta > totalDelta {
		return ""
	}
	percent := 100 - float64(idleDelta)/float64(totalDelta)*100
	return strconv.FormatFloat(percent, 'f', 0, 64) + "%"
}
