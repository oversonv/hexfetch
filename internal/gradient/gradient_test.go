package gradient

import "testing"

func TestParseStop(t *testing.T) {
	cases := map[string]RGB{
		"38;2;0;0;0":         {R: 0, G: 0, B: 0},
		"38;2;135;215;255":   {R: 135, G: 215, B: 255},
		" 38;2;255;255;255 ": {R: 255, G: 255, B: 255},
	}
	for input, want := range cases {
		if got, err := ParseStop(input); err != nil || got != want {
			t.Errorf("ParseStop(%q) = %v, %v; want %v", input, got, err, want)
		}
	}
	bad := []string{
		"", "red", "#ff0000", "38;5;100", "38;2;1;2", "38;2;1;2;3;4",
		"38;2;256;0;0", "38;2;-1;0;0", "38;2;a;0;0", "38;2;+1;0;0",
		"1;38;2;1;2;3", "48;2;1;2;3",
	}
	for _, input := range bad {
		if _, err := ParseStop(input); err == nil {
			t.Errorf("ParseStop(%q) succeeded, want error", input)
		}
	}
}

func TestANSI256(t *testing.T) {
	cases := map[RGB]int{
		{R: 255, G: 0, B: 0}:     196,
		{R: 0, G: 255, B: 0}:     46,
		{R: 0, G: 0, B: 255}:     21,
		{R: 0, G: 0, B: 0}:       16,
		{R: 255, G: 255, B: 255}: 231,
		{R: 128, G: 128, B: 128}: 244,
	}
	for color, want := range cases {
		if got := color.ANSI256(); got != want {
			t.Errorf("%v.ANSI256() = %d, want %d", color, got, want)
		}
	}
}

func TestSGR(t *testing.T) {
	color := RGB{R: 135, G: 215, B: 255}
	if got := color.SGR(true); got != "38;2;135;215;255" {
		t.Errorf("SGR(true) = %q", got)
	}
	if got := color.SGR(false); got != "38;5;153" {
		t.Errorf("SGR(false) = %q, want 38;5;153", got)
	}
}

func TestTrueColorDetection(t *testing.T) {
	cases := []struct {
		colorterm string
		term      string
		want      bool
	}{
		{"truecolor", "xterm-256color", true},
		{"24bit", "linux", true},
		{"", "xterm-256color", true},
		{"", "linux", false},
		{"", "dumb", false},
		{"", "", true},
	}
	for _, test := range cases {
		t.Setenv("COLORTERM", test.colorterm)
		t.Setenv("TERM", test.term)
		if got := TrueColor(); got != test.want {
			t.Errorf("TrueColor() with COLORTERM=%q TERM=%q = %v, want %v", test.colorterm, test.term, got, test.want)
		}
	}
}
