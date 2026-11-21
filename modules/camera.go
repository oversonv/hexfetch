//go:build linux

package modules

import (
	"path/filepath"
	"strings"
)

type camera struct{}

func init() { Register(camera{}) }

func (camera) Name() string { return "camera" }

func (camera) Run(ctx Context) ([]Entry, error) {
	paths, _ := filepath.Glob(sysPath("class/video4linux/*/name"))
	seen := map[string]bool{}
	var names []string
	for _, path := range paths {
		name, err := readTrimmed(path)
		if err != nil || name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, Unsupported("cameras unavailable")
	}
	return Single(ctx, "Camera", strings.Join(names, ", ")), nil
}
