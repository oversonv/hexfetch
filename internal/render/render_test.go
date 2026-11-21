package render

import (
	"bytes"
	"strings"
	"testing"

	"hexfetch/internal/logo"
	"hexfetch/modules"
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
	logoImage := logo.Render(logo.DefaultColors(), false, true)
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

func TestValuesAlignAfterVariableWidthIconsAndKeys(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", KeyIcon: "界", Value: "Linux"},
		{Kind: modules.KindNormal, Key: "Packages", KeyIcon: "\uf1b2", Value: "119"},
		{Kind: modules.KindNormal, Key: "Shell", KeyIcon: "\uf199", Value: "zsh"},
	}
	lines, widths := buildLines(entries, Options{Separator: " » ", KeyType: "both"}, 0)
	if len(lines) != len(entries) {
		t.Fatalf("buildLines() returned %d lines, want %d", len(lines), len(entries))
	}
	var expectedColumn int
	for index, line := range lines {
		label, _, found := strings.Cut(line, " » ")
		if !found {
			t.Fatalf("line %d has no separator: %q", index, line)
		}
		column := logo.VisibleWidth(label)
		if index == 0 {
			expectedColumn = column
		} else if column != expectedColumn {
			t.Fatalf("line %d separator starts at visible column %d, want %d: %q", index, column, expectedColumn, line)
		}
		if widths[index] != expectedColumn+logo.VisibleWidth(" » ")+logo.VisibleWidth(entries[index].Value) {
			t.Fatalf("line %d width = %d, want calculated visible width", index, widths[index])
		}
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

func TestKeyTypeSelectsIconAndKeyLabel(t *testing.T) {
	entries := []modules.Entry{
		{Kind: modules.KindNormal, Key: "OS", KeyIcon: "\uf313", Value: "NixOS"},
		{Kind: modules.KindNormal, Key: "Shell", Value: "zsh"},
	}
	tests := []struct {
		keyType string
		want    []string
	}{
		{keyType: "", want: []string{"OS    » NixOS", "Shell » zsh"}},
		{keyType: "string", want: []string{"OS    » NixOS", "Shell » zsh"}},
		{keyType: "both", want: []string{"\uf313 OS  » NixOS", "Shell » zsh"}},
		{keyType: "icon", want: []string{"\uf313     » NixOS", "Shell » zsh"}},
	}
	for _, test := range tests {
		lines, widths := buildLines(entries, Options{Separator: " » ", KeyType: test.keyType}, 0)
		if len(lines) != 2 || lines[0] != test.want[0] || lines[1] != test.want[1] {
			t.Fatalf("key type %q: lines = %q, want %q", test.keyType, lines, test.want)
		}
		if widths[1] != 11 {
			t.Fatalf("key type %q: widths = %v", test.keyType, widths)
		}
	}
}
