package logo

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

var leakedMarker = regexp.MustCompile(`\$[1-9]`)

func TestCatalogContainsDefaultAndManyLogos(t *testing.T) {
	names := Names()
	if len(names) < 500 {
		t.Fatalf("catalog has %d logos, want at least 500", len(names))
	}
	if !slices.Contains(names, DefaultName) {
		t.Fatalf("catalog does not contain %q", DefaultName)
	}
	if slices.Contains(names, "nixos_average") {
		t.Fatal("removed nixos_average logo remains in the catalog")
	}
	if !slices.IsSorted(names) {
		t.Fatal("Names() is not sorted")
	}
}

func TestEveryLogoLoadsAndWidthsAreConsistent(t *testing.T) {
	for _, name := range Names() {
		for _, mode := range []struct{ color, trueColor bool }{{true, true}, {true, false}, {false, true}} {
			loaded, err := Load(name, mode.color, mode.trueColor)
			if err != nil {
				t.Fatalf("Load(%q): %v", name, err)
			}
			if len(loaded.Lines) == 0 || len(loaded.Lines) != len(loaded.Widths) || loaded.Width == 0 {
				t.Fatalf("%s: lines=%d widths=%d width=%d", name, len(loaded.Lines), len(loaded.Widths), loaded.Width)
			}
			for index, line := range loaded.Lines {
				if got := VisibleWidth(line); got != loaded.Widths[index] {
					t.Fatalf("%s line %d: visible width %d, recorded %d", name, index, got, loaded.Widths[index])
				}
				if !mode.color && strings.Contains(line, "\x1b") {
					t.Fatalf("%s line %d: plain output contains escape", name, index)
				}
				if leakedMarker.MatchString(StripANSI(line)) {
					if art, _ := readArt(name); !strings.Contains(art, "$$") {
						t.Fatalf("%s line %d: color marker leaked into output: %q", name, index, line)
					}
				}
				if mode.color && !mode.trueColor && strings.Contains(line, "38;2;") {
					t.Fatalf("%s line %d: truecolor sequence in 256-color mode", name, index)
				}
			}
		}
	}
}

func TestLoadUnknownLogo(t *testing.T) {
	if _, err := Load("no-such-logo", true, true); err == nil {
		t.Fatal("Load of an unknown logo succeeded")
	}
}

func TestResolveAndAliases(t *testing.T) {
	names, aliases := Names(), Aliases()
	for query, want := range map[string]string{
		"amzn": "amazon_linux",
	} {
		got, ok := Resolve(names, aliases, query)
		if !ok || got != want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", query, got, ok, want)
		}
	}
	if _, ok := Resolve(names, aliases, "definitely-not-a-distro"); ok {
		t.Error("Resolve accepted an unknown name")
	}
}

func TestDetectUsesIDThenIDLike(t *testing.T) {
	names, aliases := Names(), Aliases()
	if got := Detect(names, aliases, map[string]string{"ID": "arch"}); got != "arch" {
		t.Errorf("Detect(arch) = %q", got)
	}
	got := Detect(names, aliases, map[string]string{"ID": "unknown-derivative", "ID_LIKE": "arch"})
	if got != "arch" {
		t.Errorf("Detect(ID_LIKE=arch) = %q", got)
	}
}

func TestRemovedNixosAverageCannotBeLoaded(t *testing.T) {
	if _, err := Load("nixos_average", true, true); err == nil {
		t.Fatal("removed nixos_average logo can still be loaded")
	}
	if _, ok := Resolve(Names(), Aliases(), "hex"); ok {
		t.Fatal("removed hex alias still resolves")
	}
}

func TestClassicUsesPaletteFromColorsFile(t *testing.T) {
	loaded, err := Load("arch", true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(loaded.Lines[0], "\x1b[1;") {
		t.Fatalf("classic line does not start with a bold color: %q", loaded.Lines[0])
	}
}

func TestTokenToSGR(t *testing.T) {
	cases := []struct {
		token     string
		trueColor bool
		want      string
	}{
		{"default", true, "39"},
		{"sgr:34", true, "34"},
		{"36", true, "38;5;36"},
		{"79;157;207", true, "38;2;79;157;207"},
		{"300", true, ""},
		{"1;2", true, ""},
		{"1;2;999", true, ""},
		{"x", true, ""},
	}
	for _, c := range cases {
		if got := tokenToSGR(c.token, c.trueColor); got != c.want {
			t.Errorf("tokenToSGR(%q) = %q, want %q", c.token, got, c.want)
		}
	}
	if got := tokenToSGR("79;157;207", false); !strings.HasPrefix(got, "38;5;") {
		t.Errorf("RGB token in 256-color mode = %q", got)
	}
}

func TestExpandHandlesEscapedDollarAndCarriesSlot(t *testing.T) {
	rendered, width, next := expand("$2a$$b", nil, false, 0)
	if rendered != "a$b" || width != 3 || next != 1 {
		t.Fatalf("rendered=%q width=%d next=%d", rendered, width, next)
	}
	rendered, width, next = expand("x", nil, false, 4)
	if rendered != "x" || width != 1 || next != 4 {
		t.Fatalf("slot was not carried over: rendered=%q width=%d next=%d", rendered, width, next)
	}
}
