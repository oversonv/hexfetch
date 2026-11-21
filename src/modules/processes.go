//go:build linux

package modules

import (
	"os"
	"strconv"
)

type processes struct{}

func init() { Register(processes{}) }

func (processes) Name() string { return "processes" }

func (processes) Run(ctx Context) ([]Entry, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, wrapUnavailableError("read process table", err)
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err == nil {
			count++
		}
	}
	if count == 0 {
		return nil, Unsupported("no processes found")
	}
	return Single(ctx, "Processes", strconv.Itoa(count)), nil
}
