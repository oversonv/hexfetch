package modules

import "testing"

func TestPrettyTerminalName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "com.termux", want: "Termux"},
		{input: "Termux", want: "Termux"},
		{input: "termux", want: "Termux"},
		{input: "gnome-terminal-server", want: "gnome-terminal"},
		{input: "wezterm-gui", want: "wezterm"},
		{input: "kitty", want: "kitty"},
	}
	for _, test := range tests {
		if got := prettyTerminalName(test.input); got != test.want {
			t.Errorf("prettyTerminalName(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}
