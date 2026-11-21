//go:build linux

package modules

import (
	"path/filepath"
	"strings"
)

type poweradapter struct{}

func init() { Register(poweradapter{}) }

func (poweradapter) Name() string { return "poweradapter" }

func (poweradapter) Run(ctx Context) ([]Entry, error) {
	paths, _ := filepath.Glob(sysPath("class/power_supply/*"))
	found := false
	online := false
	for _, path := range paths {
		kind, _ := readTrimmed(filepath.Join(path, "type"))
		name := filepath.Base(path)
		if !strings.EqualFold(kind, "Mains") &&
			!strings.HasPrefix(name, "AC") && !strings.HasPrefix(name, "ADP") {
			continue
		}
		value, err := readTrimmed(filepath.Join(path, "online"))
		if err != nil {
			continue
		}
		found = true
		if value == "1" {
			online = true
		}
	}
	if !found {
		return nil, Unsupported("power adapter unavailable")
	}
	status := "Disconnected"
	if online {
		status = "Connected"
	}
	return Single(ctx, "Power Adapter", status), nil
}
