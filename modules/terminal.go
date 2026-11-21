package modules

import (
	"os"
	"strconv"
	"strings"
)

type terminal struct{}

func init() { Register(terminal{}) }

func (terminal) Name() string { return "terminal" }

func (terminal) Run(ctx Context) ([]Entry, error) {
	if name := terminalFromProcesses(); name != "" {
		return Single(ctx, "Terminal", name), nil
	}
	if name := os.Getenv("TERM_PROGRAM"); name != "" {
		return Single(ctx, "Terminal", name), nil
	}
	hints := []struct{ variable, name string }{
		{"KITTY_WINDOW_ID", "kitty"},
		{"ALACRITTY_SOCKET", "alacritty"},
		{"ALACRITTY_LOG", "alacritty"},
		{"WEZTERM_EXECUTABLE", "wezterm"},
		{"WT_SESSION", "Windows Terminal"},
		{"KONSOLE_VERSION", "konsole"},
		{"GNOME_TERMINAL_SERVICE", "gnome-terminal"},
		{"TERMINAL_EMULATOR", ""},
		{"VTE_VERSION", "vte"},
	}
	for _, hint := range hints {
		value := os.Getenv(hint.variable)
		if value == "" {
			continue
		}
		if hint.name != "" {
			value = hint.name
		}
		return Single(ctx, "Terminal", value), nil
	}
	if term := os.Getenv("TERM"); term != "" {
		return Single(ctx, "Terminal", term), nil
	}
	return nil, Unsupported("terminal unknown")
}

func terminalFromProcesses() string {
	for pid, depth := os.Getppid(), 0; pid > 1 && depth < 32; pid, depth = parentPID(pid), depth+1 {
		comm, err := readTrimmed(procPath(strconv.Itoa(pid), "comm"))
		if err != nil || comm == "" {
			continue
		}
		if isWrapperProcess(strings.ToLower(comm)) {
			continue
		}
		return prettyTerminalName(comm)
	}
	return ""
}

func isWrapperProcess(name string) bool {
	if shellNames[strings.TrimPrefix(name, "-")] {
		return true
	}
	switch name {
	case "sudo", "su", "doas", "login", "script", "env", "nohup", "timeout",
		"sshd", "ssh", "dumb-init", "go", "make", "fastfetch", "hexfetch":
		return true
	}
	for _, prefix := range []string{"tmux", "screen", "proot", ".l2s", "ld-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func prettyTerminalName(comm string) string {
	switch {
	case strings.HasPrefix(comm, "gnome-terminal"):
		return "gnome-terminal"
	case comm == "wezterm-gui":
		return "wezterm"
	}
	return comm
}
