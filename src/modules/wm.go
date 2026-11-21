package modules

import (
	"os"
	"slices"
)

type wm struct{}

func init() { Register(wm{}) }

func (wm) Name() string { return "wm" }

func (wm) Run(ctx Context) ([]Entry, error) {
	hints := []struct{ variable, name string }{
		{"HYPRLAND_INSTANCE_SIGNATURE", "Hyprland"},
		{"SWAYSOCK", "sway"},
		{"I3SOCK", "i3"},
		{"NIRI_SOCKET", "niri"},
	}
	for _, hint := range hints {
		if os.Getenv(hint.variable) != "" {
			return Single(ctx, "WM", hint.name), nil
		}
	}
	known := []string{
		"sway", "Hyprland", "river", "wayfire", "niri", "labwc", "weston", "cage",
		"kwin_wayland", "kwin_x11", "gnome-shell", "mutter", "i3", "bspwm", "openbox",
		"awesome", "dwm", "xmonad", "herbstluftwm", "qtile", "fluxbox", "icewm",
		"xfwm4", "marco", "metacity", "muffin", "budgie-wm", "enlightenment", "jwm",
		"spectrwm", "leftwm", "ratpoison",
	}
	aliases := map[string]string{"gnome-shell": "Mutter", "kwin_wayland": "KWin", "kwin_x11": "KWin"}
	entries, err := os.ReadDir(procPath())
	if err != nil {
		return nil, wrapUnavailableError("read process table", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		comm, err := ReadTrim(procPath(name, "comm"))
		if err != nil || !slices.Contains(known, comm) {
			continue
		}
		if alias, ok := aliases[comm]; ok {
			comm = alias
		}
		return Single(ctx, "WM", comm), nil
	}
	return nil, Unsupported("window manager not found")
}
