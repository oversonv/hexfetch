package modules

import (
	"os"
	"path/filepath"
	"strings"
)

func gtkSettings() map[string]string {
	settings := map[string]string{}
	dir, err := os.UserConfigDir()
	if err != nil {
		return settings
	}
	for _, version := range []string{"gtk-3.0", "gtk-4.0"} {
		data, err := os.ReadFile(filepath.Join(dir, version, "settings.ini"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			if _, exists := settings[key]; !exists {
				settings[key] = strings.TrimSpace(value)
			}
		}
	}
	return settings
}

type gtkModule struct{ name, key, setting string }

func (g gtkModule) Name() string { return g.name }

func (g gtkModule) Run(ctx Context) ([]Entry, error) {
	value := gtkSettings()[g.setting]
	if value == "" {
		return nil, Unsupported("gtk setting %s not found", g.setting)
	}
	return Single(ctx, g.key, value), nil
}

func init() {
	Register(gtkModule{"theme", "Theme", "gtk-theme-name"})
	Register(gtkModule{"icons", "Icons", "gtk-icon-theme-name"})
	Register(gtkModule{"cursor", "Cursor", "gtk-cursor-theme-name"})
	Register(gtkModule{"font", "Font", "gtk-font-name"})
}
