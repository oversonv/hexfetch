//go:build linux

package modules

import (
	"os"
	"strings"
	"time"
)

type wmtheme struct{}

func init() { Register(wmtheme{}) }

func (wmtheme) Name() string { return "wmtheme" }

func (wmtheme) Run(ctx Context) ([]Entry, error) {
	desktop := strings.ToLower(os.Getenv("XDG_CURRENT_DESKTOP"))
	switch {
	case strings.Contains(desktop, "kde"):
		if value := wmthemeKDE(); value != "" {
			return Single(ctx, "WM Theme", value), nil
		}
	case strings.Contains(desktop, "xfce"):
		if value := xfconfTheme(); value != "" {
			return Single(ctx, "WM Theme", value), nil
		}
	case strings.Contains(desktop, "gnome"), strings.Contains(desktop, "unity"), strings.Contains(desktop, "pantheon"):
		if value := gsettingsWMTheme(); value != "" {
			return Single(ctx, "WM Theme", value), nil
		}
	}
	for _, source := range []func() string{wmthemeKDE, xfconfTheme, gsettingsWMTheme} {
		if value := source(); value != "" {
			return Single(ctx, "WM Theme", value), nil
		}
	}
	return nil, Unsupported("window manager theme not found")
}

func wmthemeKDE() string {
	path, ok := userConfigFile("kdeglobals")
	if !ok {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return kdeWMTheme(data)
}

func xfconfTheme() string {
	if !haveCommand("xfconf-query") {
		return ""
	}
	out, err := runCommandPath(2*time.Second, "xfconf-query", "-c", "xsettings", "-p", "/Net/ThemeName")
	if err != nil {
		return ""
	}
	return strings.SplitN(out, "\n", 2)[0]
}

func gsettingsWMTheme() string {
	if !haveCommand("gsettings") {
		return ""
	}
	out, err := runCommandPath(2*time.Second, "gsettings", "get", "org.gnome.desktop.wm.preferences", "theme")
	if err != nil {
		return ""
	}
	return cleanValue(out)
}

// kdeWMTheme reads the KWin theme name, falling back to the widget style.
func kdeWMTheme(data []byte) string {
	if value := iniGroupValue(data, "KWin", "Name"); value != "" {
		return value
	}
	return iniGroupValue(data, "KDE", "widgetStyle")
}

func iniGroupValue(data []byte, group, key string) string {
	current := ""
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.Trim(line, "[]")
			continue
		}
		if current != group {
			continue
		}
		if name, value, ok := strings.Cut(line, "="); ok && strings.TrimSpace(name) == key {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
