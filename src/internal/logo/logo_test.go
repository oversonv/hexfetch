package logo

import (
	"regexp"
	"strings"
	"testing"
)

var slotMarker = regexp.MustCompile(`\$[1-9]`)

func stripMarkers(art string) string {
	return strings.ReplaceAll(slotMarker.ReplaceAllString(art, ""), "$$", "$")
}

func TestDefaultLogoLoadsAsClassicArt(t *testing.T) {
	art, err := readArt(DefaultName)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Load(DefaultName, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if rendered := strings.Join(plain.Lines, "\n"); rendered != stripMarkers(art) {
		t.Fatalf("plain logo = %q, want embedded art %q", rendered, stripMarkers(art))
	}
	if strings.Contains(strings.Join(plain.Lines, "\n"), "\x1b[") {
		t.Fatal("plain logo contains ANSI sequences")
	}
}

func TestColoredLogoUsesPalette(t *testing.T) {
	loaded, err := Load(DefaultName, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Lines) == 0 {
		t.Fatal("colored logo is empty")
	}
	for index, line := range loaded.Lines {
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line %d does not end with reset", index)
		}
		if !strings.Contains(line, "\x1b[1;") {
			t.Errorf("line %d has no palette color", index)
		}
	}
}

func TestLogoUsesANSI256Fallback(t *testing.T) {
	truecolor, err := Load("axos", true, true)
	if err != nil {
		t.Fatal(err)
	}
	fallback, err := Load("axos", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(truecolor.Lines, "\n"), "\x1b[1;38;2;222;6;255m") {
		t.Fatal("truecolor logo has no RGB sequence")
	}
	joined := strings.Join(fallback.Lines, "\n")
	if !strings.Contains(joined, "\x1b[1;38;5;") {
		t.Fatal("fallback logo has no ANSI 256 color")
	}
	if strings.Contains(joined, "38;2;") {
		t.Fatal("fallback logo contains truecolor sequence")
	}
}

func TestVisibleWidthIgnoresANSI(t *testing.T) {
	loaded, err := Load(DefaultName, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range loaded.Lines {
		if got, want := VisibleWidth(line), loaded.Widths[index]; got != want {
			t.Errorf("line %d visible width = %d, want %d", index, got, want)
		}
	}
}
