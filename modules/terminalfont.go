package modules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type terminalFont struct{}

func init() { Register(terminalFont{}) }

func (terminalFont) Name() string { return "terminalfont" }

func (terminalFont) Run(ctx Context) ([]Entry, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	probes := []struct {
		active func() bool
		file   string
		re     *regexp.Regexp
	}{
		{func() bool { return os.Getenv("KITTY_WINDOW_ID") != "" },
			"kitty/kitty.conf", regexp.MustCompile(`(?m)^\s*font_family\s+(.+?)\s*$`)},
		{func() bool { return os.Getenv("ALACRITTY_SOCKET") != "" || os.Getenv("ALACRITTY_LOG") != "" },
			"alacritty/alacritty.toml", regexp.MustCompile(`(?m)^\s*family\s*=\s*"([^"]+)"`)},
		{func() bool { return strings.HasPrefix(os.Getenv("TERM"), "foot") },
			"foot/foot.ini", regexp.MustCompile(`(?m)^\s*font\s*=\s*([^:#\n]+)`)},
	}
	for _, probe := range probes {
		if !probe.active() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, probe.file))
		if err != nil {
			continue
		}
		if m := probe.re.FindSubmatch(data); m != nil {
			return Single(ctx, "Terminal Font", strings.TrimSpace(string(m[1]))), nil
		}
	}
	return nil, Unsupported("terminal font unknown")
}
