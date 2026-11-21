//go:build linux

package modules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type memory struct{}

func init() { Register(memory{}) }

func (memory) Name() string { return "memory" }

func (memory) Run(ctx Context) ([]Entry, error) {
	data, err := cachedMeminfo()
	if err != nil {
		return nil, err
	}
	info := parseMeminfo(data)
	total := info["MemTotal"] * 1024
	if total == 0 {
		return nil, errors.New("memory information unavailable")
	}
	available := info["MemAvailable"] * 1024
	if available == 0 {
		available = (info["MemFree"] + info["Buffers"] + info["Cached"]) * 1024
	}
	if available > total {
		available = total
	}
	used := total - available
	value := fmt.Sprintf("%s / %s (%s)", formatBytes(used), formatBytes(total), formatPercent(used, total))
	entries := Single(ctx, "Memory", value)
	entries[0].Fields = map[string]string{
		"used": formatBytes(used), "total": formatBytes(total),
		"available": formatBytes(available), "percent": formatPercent(used, total),
	}
	return entries, nil
}

// parseMeminfo returns values in KiB, matching /proc/meminfo.
func parseMeminfo(data []byte) map[string]uint64 {
	values := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			continue
		}
		number, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		values[strings.TrimSpace(key)] = number
	}
	return values
}
