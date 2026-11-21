//go:build linux

package modules

import (
	"path/filepath"
	"strings"
)

type display struct{}

func init() { Register(display{}) }

func (display) Name() string { return "display" }

func (display) Run(ctx Context) ([]Entry, error) {
	connectors, _ := filepath.Glob(sysPath("class/drm/card*-*"))
	var entries []Entry
	for _, connector := range connectors {
		status, err := readTrimmed(filepath.Join(connector, "status"))
		if err != nil || status != "connected" {
			continue
		}
		modes, err := readFile(filepath.Join(connector, "modes"))
		if err != nil {
			continue
		}
		mode, _, _ := strings.Cut(strings.TrimSpace(string(modes)), "\n")
		mode = strings.TrimSpace(mode)
		if mode == "" {
			continue
		}
		name := strings.TrimPrefix(filepath.Base(connector), "card0-")
		value := mode
		if name != "" && name != mode {
			value += " (" + name + ")"
		}
		entries = append(entries, Entry{Key: "Display", Value: value})
	}
	if len(entries) == 0 {
		return nil, Unsupported("display information unavailable")
	}
	return entries, nil
}
