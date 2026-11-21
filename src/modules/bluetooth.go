//go:build linux

package modules

import (
	"path/filepath"
	"strings"
)

type bluetooth struct{}

func init() { Register(bluetooth{}) }

func (bluetooth) Name() string { return "bluetooth" }

func (bluetooth) Run(ctx Context) ([]Entry, error) {
	paths, _ := filepath.Glob(sysPath("class", "bluetooth", "hci*"))
	var controllers []string
	for _, path := range paths {
		name, _ := readTrimmed(filepath.Join(path, "name"))
		name = cleanValue(name)
		if name == "" {
			if data, err := readFile(filepath.Join(path, "uevent")); err == nil {
				for _, line := range strings.Split(string(data), "\n") {
					if value, ok := strings.CutPrefix(line, "NAME="); ok {
						name = cleanValue(value)
					}
				}
			}
		}
		address, _ := readTrimmed(filepath.Join(path, "address"))
		value := name
		if value == "" {
			value = address
		}
		if value == "" {
			value = filepath.Base(path)
		}
		controllers = append(controllers, value)
	}
	if len(controllers) == 0 {
		return nil, Unsupported("bluetooth controllers unavailable")
	}
	return Single(ctx, "Bluetooth", strings.Join(controllers, ", ")), nil
}
