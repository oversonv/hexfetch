package render

import (
	"bytes"
	"strings"
	"testing"

	"hexfetch/src/internal/logo"
	"hexfetch/src/modules"
)

func fakeLogo(width int) logo.Logo {
	l := logo.Logo{Width: width}
	for range 6 {
		l.Lines = append(l.Lines, strings.Repeat("#", width))
		l.Widths = append(l.Widths, width)
	}
	return l
}

func render(t *testing.T, width int, lg logo.Logo) []string {
	t.Helper()
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", Value: "Test"},
		{Kind: modules.KindBreak},
		{Kind: modules.KindPalette},
	}
	var buf bytes.Buffer
	err := Render(&buf, entries, Options{Logo: lg, Position: "left", Padding: 3, Separator: ": ", Width: width, Color: true})
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
}

func TestPaletteSurvivesAnyWidth(t *testing.T) {
	for width := 1; width <= 120; width++ {
		out := strings.Join(render(t, width, fakeLogo(30)), "\n")
		if got := strings.Count(out, "48;5;"); got != 16 {
			t.Fatalf("width %d: %d of 16 palette cells rendered", width, got)
		}
		for _, line := range strings.Split(out, "\n") {
			if w := logo.VisibleWidth(line); w > width {
				t.Fatalf("width %d: line is %d columns wide", width, w)
			}
		}
	}
}

func TestPaletteFullWidthKeepsLayout(t *testing.T) {
	rows, widths := paletteRows(0)
	if len(rows) != 2 || widths[0] != 24 {
		t.Fatalf("unknown width: rows=%d width=%d", len(rows), widths[0])
	}
	rows, widths = paletteRows(24)
	if len(rows) != 2 || widths[0] != 24 {
		t.Fatalf("24 cols: rows=%d width=%d", len(rows), widths[0])
	}
}

func TestPaletteCellsAreNeverBlank(t *testing.T) {
	for _, avail := range []int{0, 24, 16, 8, 4, 2, 1} {
		rows, widths := paletteRows(avail)
		for i, row := range rows {
			plain := logo.StripANSI(row)
			if strings.Trim(plain, "\u2588") != "" {
				t.Fatalf("avail %d: row %d has non-glyph cells: %q", avail, i, plain)
			}
			if got := len([]rune(plain)); got != widths[i] {
				t.Fatalf("avail %d: row %d has %d cells, want %d", avail, i, got, widths[i])
			}
		}
	}
}

func TestNarrowTerminalHidesLogo(t *testing.T) {
	logoImage, err := logo.Load(logo.DefaultName, false, true)
	if err != nil {
		t.Fatal(err)
	}
	entries := []modules.Entry{{Kind: modules.KindNormal, Key: "OS", Value: "Test"}}
	var output bytes.Buffer
	if err := Render(&output, entries, Options{
		Logo:      logoImage,
		Position:  "left",
		Padding:   1,
		Separator: ": ",
		Width:     1,
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "▄") {
		t.Fatalf("narrow terminal output still contains the logo: %q", output.String())
	}
}

func TestTitleSeparatorUsesVisibleTitleWidthAndConfiguredColor(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindTitle, Key: "long-user-name@host", Value: "Welcome!"},
		{Kind: modules.KindSeparator, Value: "-"},
	}
	lines, widths := buildLines(entries, Options{
		TitleColor:     "38;2;255;255;255",
		SeparatorColor: "38;2;1;2;3",
		Color:          true,
	}, 0)
	if len(lines) != 2 {
		t.Fatalf("buildLines() returned %d lines: %q", len(lines), lines)
	}
	if widths[1] != 8 {
		t.Fatalf("separator width = %d, want visible title width 8", widths[1])
	}
	if !strings.Contains(lines[1], "\x1b[38;2;1;2;3m--------\x1b[0m") {
		t.Fatalf("separator does not use configured color or width: %q", lines[1])
	}
}

func TestSeparatorWithoutTitleUsesModuleOutputWidth(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindSeparator, Value: "-"},
		{Kind: modules.KindNormal, Key: "Shell", Value: "zsh"},
	}
	lines, widths := buildLines(entries, Options{Separator: " » "}, 0)
	if len(lines) != 2 || lines[0] != "-----------" || widths[0] != 11 {
		t.Fatalf("separator without title = %q widths=%v, want 11-column line", lines, widths)
	}
}

func TestSeparatorSupportsWideCharactersWithoutExceedingWidth(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindTitle, Value: "abcde"},
		{Kind: modules.KindSeparator, Value: "界"},
	}
	lines, widths := buildLines(entries, Options{}, 0)
	if widths[1] != 5 || logo.VisibleWidth(lines[1]) != 5 {
		t.Fatalf("wide separator width = %d, visible %d, want 5", widths[1], logo.VisibleWidth(lines[1]))
	}
}

func TestPaletteIsOmittedWhenColorIsDisabled(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindPalette},
	}
	var output bytes.Buffer
	if err := Render(&output, entries, Options{Color: false}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "\x1b[") || output.Len() != 0 {
		t.Fatalf("disabled-color palette output = %q, want empty output", output.String())
	}
}

func TestSeparatorUsesPaletteWidth(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindSeparator, Value: "-"},
		{Kind: modules.KindPalette},
	}
	lines, widths := buildLines(entries, Options{Color: true}, 0)
	if len(lines) != 3 || widths[0] != 24 {
		t.Fatalf("palette separator width = %d with %d lines, want 24 with 3 lines", widths[0], len(lines))
	}
}

func TestLabelsAreNotAlignedAcrossModules(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", Value: "Linux aarch64"},
		{Kind: modules.KindNormal, Key: "Packages", Value: "119 (dpkg)"},
		{Kind: modules.KindNormal, Key: "Shell", Value: "zsh 5.9.2"},
		{Kind: modules.KindNormal, Key: "Python", Value: "3.14.10"},
		{Kind: modules.KindNormal, Key: "Terminal", Value: "Termux"},
	}
	want := []string{
		"OS » Linux aarch64",
		"Packages » 119 (dpkg)",
		"Shell » zsh 5.9.2",
		"Python » 3.14.10",
		"Terminal » Termux",
	}
	lines, widths := buildLines(entries, Options{Separator: " » "}, 0)
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(lines), len(want))
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("line %d = %q, want %q", i, lines[i], want[i])
		}
		if widths[i] != logo.VisibleWidth(want[i]) {
			t.Fatalf("line %d width = %d, want %d", i, widths[i], logo.VisibleWidth(want[i]))
		}
	}
}

func TestSeparatorIsPreservedExactly(t *testing.T) {
	entry := []modules.Entry{{Kind: modules.KindNormal, Key: "OS", Value: "Linux"}}
	for _, sep := range []string{"", ":", ": ", " » ", "  →  ", "界"} {
		lines, widths := buildLines(entry, Options{Separator: sep}, 0)
		if want := "OS" + sep + "Linux"; lines[0] != want || widths[0] != logo.VisibleWidth(want) {
			t.Fatalf("separator %q: line %q width %d, want %q", sep, lines[0], widths[0], want)
		}
	}
}

func TestIconsMissingCustomAndUnicode(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", KeyIcon: "\uf313", Value: "NixOS"},
		{Kind: modules.KindNormal, Key: "Shell", Value: "zsh"},
		{Kind: modules.KindNormal, Key: "ОС", KeyIcon: "界", Value: "x"},
	}
	tests := []struct {
		keyType string
		want    []string
	}{
		{"", []string{"OS » NixOS", "Shell » zsh", "ОС » x"}},
		{"string", []string{"OS » NixOS", "Shell » zsh", "ОС » x"}},
		{"both", []string{"\uf313 OS » NixOS", "Shell » zsh", "界 ОС » x"}},
		{"icon", []string{"\uf313 » NixOS", "Shell » zsh", "界 » x"}},
	}
	for _, test := range tests {
		lines, _ := buildLines(entries, Options{Separator: " » ", KeyType: test.keyType}, 0)
		for i := range test.want {
			if lines[i] != test.want[i] {
				t.Fatalf("key type %q line %d = %q, want %q", test.keyType, i, lines[i], test.want[i])
			}
		}
	}
}

func TestWidthIgnoresANSIAndCountsDisplayColumns(t *testing.T) {
	entries := []modules.Entry{{Kind: modules.KindNormal, Key: "界界", Value: "é"}}
	lines, widths := buildLines(entries, Options{
		Separator: ": ", KeyColor: "1;31", SeparatorColor: "2", OutputColor: "32", Color: true,
	}, 0)
	want := "\x1b[1;31m界界\x1b[0m\x1b[2m: \x1b[0m\x1b[32mé\x1b[0m"
	if lines[0] != want {
		t.Fatalf("line = %q, want %q", lines[0], want)
	}
	// 界界 = 2 code points, 6 bytes, 4 columns; ": " = 2; é = 1.
	if widths[0] != 7 || logo.VisibleWidth(lines[0]) != 7 {
		t.Fatalf("width = %d, visible = %d, want 7", widths[0], logo.VisibleWidth(lines[0]))
	}
}

func TestLongLabelDoesNotAffectOtherLines(t *testing.T) {
	long := strings.Repeat("L", 60)
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: long, Value: "v"},
		{Kind: modules.KindNormal, Key: "OS", Value: "Linux"},
	}
	lines, _ := buildLines(entries, Options{Separator: " » "}, 0)
	if lines[0] != long+" » v" || lines[1] != "OS » Linux" {
		t.Fatalf("lines = %q", lines)
	}
}

func TestMultilineAndLogoSpacingUnchanged(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", Value: "Linux"},
		{Kind: modules.KindNormal, Key: "Packages", Value: "119"},
		{Kind: modules.KindBreak},
		{Kind: modules.KindNormal, Key: "Shell", Value: "zsh"},
	}
	var buf bytes.Buffer
	err := Render(&buf, entries, Options{Logo: fakeLogo(4), Position: "left", Padding: 2, Separator: " » "})
	if err != nil {
		t.Fatal(err)
	}
	want := "####  OS » Linux\n" +
		"####  Packages » 119\n" +
		"####\n" +
		"####  Shell » zsh\n" +
		"####\n" +
		"####\n"
	if buf.String() != want {
		t.Fatalf("output = %q, want %q", buf.String(), want)
	}
}

func TestSeparatorLengthOverridesTitleWidth(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindTitle, Value: "Hey"},
		{Kind: modules.KindSeparator, Value: "─"},
	}
	lines, widths := buildLines(entries, Options{SeparatorLength: 18}, 0)
	if len(lines) != 2 || widths[1] != 18 || logo.VisibleWidth(lines[1]) != 18 {
		t.Fatalf("fixed separator = %q with widths %v, want 18 columns", lines, widths)
	}
	lines, widths = buildLines(entries, Options{}, 0)
	if widths[1] != 3 || logo.VisibleWidth(lines[1]) != 3 {
		t.Fatalf("automatic separator = %q with widths %v, want title width 3", lines, widths)
	}
}
