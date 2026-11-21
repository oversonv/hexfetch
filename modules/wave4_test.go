package modules

import "testing"

func TestCleanWallpaperURL(t *testing.T) {
	tests := map[string]string{
		`'file:///home/user/pictures/wall.png'`: "/home/user/pictures/wall.png",
		`"file:///tmp/x.jpg"`:                   "/tmp/x.jpg",
		`/plain/path.png`:                       "/plain/path.png",
		`''`:                                    "",
	}
	for input, want := range tests {
		if got := cleanWallpaperURL(input); got != want {
			t.Errorf("cleanWallpaperURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFirstSwwwImage(t *testing.T) {
	tests := map[string]string{
		"eDP-1: 1920x1080, currently displaying: /home/u/w.png\n":      "/home/u/w.png",
		"HDMI-1: 540x960, currently displaying: 'file:///tmp/a.jpg'\n": "/tmp/a.jpg",
		"\n\n":         "",
		"no paths":     "",
		"DP-1: /x.png": "/x.png",
	}
	for input, want := range tests {
		if got := firstSwwwImage(input); got != want {
			t.Errorf("firstSwwwImage(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPlasmaImageFromConfig(t *testing.T) {
	data := []byte(`[Containments][1]
plugin=org.kde.desktopcontainment
[Containments][2][Wallpaper][org.kde.image][General]
Image=file:///usr/share/wallpapers/Next.png
image=file:///home/u/pics/w2.png
`)
	if got := plasmaImageFromConfig(data); got != "/home/u/pics/w2.png" {
		t.Errorf("plasmaImageFromConfig = %q", got)
	}
	if got := plasmaImageFromConfig([]byte("[Containments]\nplugin=x\n")); got != "" {
		t.Errorf("plasmaImageFromConfig no-match = %q", got)
	}
}

func TestFehbgImage(t *testing.T) {
	data := []byte(`#!/bin/sh
feh --no-fehbg --bg-scale '/home/user/pictures/wallpaper.jpg' &`)
	if got := fehbgImage(data); got != "/home/user/pictures/wallpaper.jpg" {
		t.Errorf("fehbgImage = %q", got)
	}
	if got := fehbgImage([]byte("echo hi\n")); got != "" {
		t.Errorf("fehbgImage no-match = %q", got)
	}
}

func TestKdeWMTheme(t *testing.T) {
	data := []byte(`[General]
XftHintFont=medium
[KDE]
widgetStyle=Breeze
[WM]
activeBackground=49,54,59
`)
	if got := kdeWMTheme(data); got != "Breeze" {
		t.Errorf("kdeWMTheme = %q, want Breeze", got)
	}
	kwin := []byte("[KWin]\nName=Klassy\n[KDE]\nwidgetStyle=Breeze\n")
	if got := kdeWMTheme(kwin); got != "Klassy" {
		t.Errorf("kdeWMTheme kwin = %q, want Klassy", got)
	}
	if got := kdeWMTheme([]byte("nothing")); got != "" {
		t.Errorf("kdeWMTheme empty = %q", got)
	}
}

func TestIniGroupValue(t *testing.T) {
	data := []byte("[a]\nX=1\nY=2\n[b]\nX=3\n")
	if got := iniGroupValue(data, "b", "X"); got != "3" {
		t.Errorf("iniGroupValue b/X = %q", got)
	}
	if got := iniGroupValue(data, "a", "X"); got != "1" {
		t.Errorf("iniGroupValue a/X = %q", got)
	}
	if got := iniGroupValue(data, "a", "missing"); got != "" {
		t.Errorf("iniGroupValue a/missing = %q", got)
	}
}

func TestCleanMediaValue(t *testing.T) {
	tests := map[string]string{
		"vlc:  - ":                        "",
		"":                                "",
		"vlc: Never Gonna Give You Up - ": "vlc: Never Gonna Give You Up",
		"mpv: Song - Artist":              "mpv: Song - Artist",
		"stray:  - Rick Astley":           "Rick Astley",
		"\nvlc:  - ":                      "",
	}
	for input, want := range tests {
		if got := cleanMediaValue(input); got != want {
			t.Errorf("cleanMediaValue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestWeatherReport(t *testing.T) {
	if got := weatherReport("\nLondon: ☀️ +12.0°C\nParis\n"); got != "London: ☀️ +12.0°C" {
		t.Errorf("weatherReport = %q", got)
	}
	if got := weatherReport("\n\n   \n"); got != "" {
		t.Errorf("weatherReport blank = %q", got)
	}
}

func TestLoaderEntryTitle(t *testing.T) {
	data := []byte("timeout 3\ntitle   Arch Linux\nlinux /vmlinuz\n")
	if got := loaderEntryTitle(data); got != "Arch Linux" {
		t.Errorf("loaderEntryTitle = %q", got)
	}
	if got := loaderEntryTitle([]byte("# comment\nlinux /vmlinuz\n")); got != "" {
		t.Errorf("loaderEntryTitle no-title = %q", got)
	}
}
