//go:build linux

package modules

import (
	"fmt"
)

type swap struct{}

func init() { Register(swap{}) }

func (swap) Name() string { return "swap" }

func (swap) Run(ctx Context) ([]Entry, error) {
	data, err := cachedMeminfo()
	if err != nil {
		return nil, err
	}
	info := parseMeminfo(data)
	total := info["SwapTotal"] * 1024
	if total == 0 {
		return nil, Unsupported("swap is not configured")
	}
	free := info["SwapFree"] * 1024
	if free > total {
		free = total
	}
	used := total - free
	value := fmt.Sprintf("%s / %s (%s)", formatBytes(used), formatBytes(total), formatPercent(used, total))
	entries := Single(ctx, "Swap", value)
	entries[0].Fields = map[string]string{
		"used": formatBytes(used), "total": formatBytes(total),
		"free": formatBytes(free), "percent": formatPercent(used, total),
	}
	return entries, nil
}
