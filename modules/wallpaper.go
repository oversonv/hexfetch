//go:build linux

package modules

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type wallpaper struct{}

func init() { Register(wallpaper{}) }

func (wallpaper) Name() string { return "wallpaper" }

func (wallpaper) Run(ctx Context) ([]Entry, error) {
	sources := []func() string{
		wallpaperSwww,
		wallpaperHyprpaper,
		wallpaperGNOME,
		wallpaperXFCE,
		wallpaperKDE,
		wallpaperFeh,
	}
	for _, source := range sources {
		if value := source(); value != "" {
			return Single(ctx, "Wallpaper", value), nil
		}
	}
	return nil, Unsupported("wallpaper not found")
}

func wallpaperSwww() string {
	if !haveCommand("swww") {
		return ""
	}
	out, err := runCommandPath(2*time.Second, "swww", "query")
	if err != nil {
		return ""
	}
	return firstSwwwImage(out)
}

func wallpaperHyprpaper() string {
	if !haveCommand("hyprctl") {
		return ""
	}
	out, err := runCommandPath(2*time.Second, "hyprctl", "hyprpaper", "listactive")
	if err != nil {
		return ""
	}
	return cleanWallpaperURL(strings.SplitN(out, "\n", 2)[0])
}

func wallpaperGNOME() string {
	if !haveCommand("gsettings") {
		return ""
	}
	out, err := runCommandPath(2*time.Second, "gsettings", "get", "org.gnome.desktop.background", "picture-uri")
	if err != nil {
		return ""
	}
	return cleanWallpaperURL(out)
}

func wallpaperXFCE() string {
	if !haveCommand("xfconf-query") {
		return ""
	}
	probe := "/backdrop/screen0/monitor0/workspace0/last-image"
	if value := xfconfWallpaper(probe); value != "" {
		return value
	}
	listing, err := runCommandPath(2*time.Second, "xfconf-query", "-c", "xfce4-desktop", "-l")
	if err != nil {
		return ""
	}
	queried := 0
	for _, line := range strings.Split(listing, "\n") {
		property := strings.TrimSpace(line)
		if !strings.HasSuffix(property, "last-image") || property == probe {
			continue
		}
		if value := xfconfWallpaper(property); value != "" {
			return value
		}
		queried++
		if queried >= 5 {
			break
		}
	}
	return ""
}

func xfconfWallpaper(property string) string {
	out, err := runCommandPath(2*time.Second, "xfconf-query", "-c", "xfce4-desktop", "-p", property)
	if err != nil {
		return ""
	}
	return cleanWallpaperURL(strings.SplitN(out, "\n", 2)[0])
}

func wallpaperKDE() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "plasma-org.kde.plasma-desktop-appletsrc"))
	if err != nil {
		return ""
	}
	return plasmaImageFromConfig(data)
}

func wallpaperFeh() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".fehbg"))
	if err != nil {
		return ""
	}
	return fehbgImage(data)
}

// cleanWallpaperURL strips quotes and the file:// scheme from wallpaper values.
func cleanWallpaperURL(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = strings.TrimPrefix(value, "file://")
	return strings.Trim(value, `"'`)
}

func firstSwwwImage(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if index := strings.Index(line, "displaying: "); index >= 0 {
			if value := strings.TrimSpace(line[index+len("displaying: "):]); value != "" {
				return cleanWallpaperURL(value)
			}
			continue
		}
		if _, value, ok := strings.Cut(line, ": "); ok && strings.HasPrefix(value, "/") {
			return cleanWallpaperURL(value)
		}
	}
	return ""
}

func plasmaImageFromConfig(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok && strings.TrimSpace(key) == "image" {
			if clean := cleanWallpaperURL(value); clean != "" {
				return clean
			}
		}
	}
	return ""
}

func fehbgImage(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if _, rest, ok := strings.Cut(line, "--bg-"); ok {
			start := strings.IndexByte(rest, '\'')
			end := strings.LastIndexByte(rest, '\'')
			if start >= 0 && end > start {
				return cleanWallpaperURL(rest[start+1 : end])
			}
		}
	}
	return ""
}
