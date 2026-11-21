//go:build linux

package modules

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type fans struct{}

func init() { Register(fans{}) }

func (fans) Name() string { return "fans" }

func (fans) Run(ctx Context) ([]Entry, error) {
	hwmons, _ := filepath.Glob(sysPath("class/hwmon/hwmon*"))
	type fan struct {
		label string
		rpm   int64
	}
	var found []fan
	for _, hwmon := range hwmons {
		name, _ := readTrimmed(filepath.Join(hwmon, "name"))
		if name == "" {
			name = filepath.Base(hwmon)
		}
		inputs, _ := filepath.Glob(filepath.Join(hwmon, "fan*_input"))
		for _, input := range inputs {
			raw, err := readTrimmed(input)
			if err != nil {
				continue
			}
			rpm, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil || rpm <= 0 {
				continue
			}
			found = append(found, fan{label: sensorLabel(input, name), rpm: rpm})
		}
	}
	if len(found) == 0 {
		return nil, Unsupported("fan sensors unavailable")
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].label < found[j].label })
	entries := make([]Entry, 0, len(found))
	for _, item := range found {
		entries = append(entries, Entry{Key: item.label, Value: fmt.Sprintf("%d RPM", item.rpm)})
	}
	return entries, nil
}
