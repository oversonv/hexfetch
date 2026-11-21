package logo

import (
	"strings"
	"testing"
)

func TestDefaultLogoLoadsAsClassicArt(t *testing.T) {
	art, err := readArt(DefaultName)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Load(DefaultName, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if rendered := strings.Join(plain.Lines, "\n"); rendered != art {
		t.Fatalf("plain logo = %q, want embedded art %q", rendered, art)
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
	loaded, err := Load(DefaultName, true, false)
	if err != nil {
		t.Fatal(err)
	}
	for index, line := range loaded.Lines {
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Errorf("line %d does not end with reset", index)
		}
		if !strings.Contains(line, "\x1b[38;5;") {
			t.Errorf("line %d has no ANSI 256 color", index)
		}
		if strings.Contains(line, "\x1b[38;2;") {
			t.Errorf("line %d contains truecolor sequence", index)
		}
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
