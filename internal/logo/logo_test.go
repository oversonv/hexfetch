package logo

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"hexfetch/internal/gradient"
)

const referenceArt = `       ▄▄▖     ▄▄▄  ▄▄▄
       ▜██▖    ▀██▙▟██▘
    ▄▄▄▄███▄▄▄▄▖▀████▘
   ▟████████████▖▜██▙  ▗█▙
       ▄▄▄▖       ▜██▌▗███
      ▟██▛         ▜▛▗███▘
▟███████▛           ▗███████▙
▝▀▀▀███▛▗▖         ▗███▀▀▀▀▀▘
   ▟██▛▗██▖       ▗███▘
   ██▛ ▝███▖▗▄▄▄▄▄▄▄▄▄▄▄▄▖
   ▝▀   ▟███ ▜██████████▛
       ▟█████      ▜██▖
      ▟██▘▝███      ▜██▖`

var referenceBands = []string{
	"       000     111  111",
	"       0000    11111111",
	"    000000000000111111",
	"   000000000000001111  222",
	"       5555       11112222",
	"      5555         1122222",
	"555555555           222222222",
	"5555555544         2222222222",
	"   55554444       22222",
	"   555 4444433333333333333",
	"   55   4444 333333333333",
	"       444444      3333",
	"      44444444      3333",
}

type referenceRGB struct {
	r int
	g int
	b int
}

type referenceGradient struct {
	dx float64
	dy float64
	t0 float64
	t1 float64
	c0 referenceRGB
	c1 referenceRGB
}

var referenceGradients = [...]referenceGradient{
	{-0.816, -0.578, -25.36, -8.16, referenceRGB{133, 104, 242}, referenceRGB{206, 177, 240}},
	{-0.781, -0.625, -35.76, -16.4, referenceRGB{107, 106, 255}, referenceRGB{79, 185, 254}},
	{-0.84, 0.542, -26.75, -8.59, referenceRGB{206, 177, 240}, referenceRGB{133, 104, 242}},
	{-0.941, 0.339, -25.31, -7.43, referenceRGB{79, 185, 254}, referenceRGB{107, 106, 255}},
	{-0.512, -0.859, -43.39, -25.04, referenceRGB{206, 177, 240}, referenceRGB{133, 104, 242}},
	{-0.603, 0.798, 0.53, 20.91, referenceRGB{107, 106, 255}, referenceRGB{79, 185, 254}},
}

func referenceColorAt(x, y int) referenceRGB {
	band := int(referenceBands[y][x] - '0')
	g := referenceGradients[band]
	px := float64(x) * (42.0 / 28.0)
	py := float64(y) * (19.0 / 12.0)
	t := (px*g.dx + py*2*g.dy - g.t0) / (g.t1 - g.t0)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return referenceRGB{
		r: int(math.RoundToEven(float64(g.c0.r) + float64(g.c1.r-g.c0.r)*t)),
		g: int(math.RoundToEven(float64(g.c0.g) + float64(g.c1.g-g.c0.g)*t)),
		b: int(math.RoundToEven(float64(g.c0.b) + float64(g.c1.b-g.c0.b)*t)),
	}
}

func referenceRender() string {
	lines := strings.Split(referenceArt, "\n")
	var output strings.Builder
	for y, line := range lines {
		for x, character := range []rune(line) {
			if character == ' ' {
				output.WriteRune(character)
				continue
			}
			color := referenceColorAt(x, y)
			fmt.Fprintf(&output, "\x1b[38;2;%d;%d;%dm%c", color.r, color.g, color.b, character)
		}
		output.WriteString("\x1b[0m")
		if y < len(lines)-1 {
			output.WriteByte('\n')
		}
	}
	return output.String()
}

func TestDefaultRenderMatchesReference(t *testing.T) {
	got := Render(DefaultColors(), true, true)
	if rendered := strings.Join(got.Lines, "\n"); rendered != referenceRender() {
		t.Fatalf("rendered logo differs from reference")
	}
	if got.Width == 0 || len(got.Lines) != 13 || len(got.Widths) != 13 {
		t.Fatalf("logo dimensions = width %d, lines %d, widths %d", got.Width, len(got.Lines), len(got.Widths))
	}
}

func TestPlainRenderHasNoANSI(t *testing.T) {
	got := Render(DefaultColors(), false, true)
	if rendered := strings.Join(got.Lines, "\n"); rendered != referenceArt {
		t.Fatalf("plain logo = %q, want reference art", rendered)
	}
	if strings.Contains(strings.Join(got.Lines, "\n"), "\x1b[") {
		t.Fatal("plain logo contains ANSI sequences")
	}
}

func TestRenderUsesANSI256Fallback(t *testing.T) {
	got := Render(DefaultColors(), true, false)
	for index, line := range got.Lines {
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

func TestNixosAverageMaskMatchesReferenceBands(t *testing.T) {
	text, err := readArt("nixos_average")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(text, "\n")
	if len(lines) != len(referenceBands) {
		t.Fatalf("nixos_average has %d lines, reference bands have %d", len(lines), len(referenceBands))
	}
	current := 0
	for y, line := range lines {
		cells, _, next := parseCells(line, current)
		current = next
		for _, c := range cells {
			if c.r == ' ' {
				continue
			}
			if c.x >= len(referenceBands[y]) || int(referenceBands[y][c.x]-'0') != c.slot {
				t.Fatalf("nixos_average cell (%d,%d) has slot %d, reference band %q", c.x, y, c.slot, referenceBands[y])
			}
		}
	}
}

func TestBandColorPairsFollowBandOrder(t *testing.T) {
	var colors [12]gradient.RGB
	for index := 0; index < 6; index++ {
		colors[index*2] = gradient.RGB{R: uint8(index + 1), G: uint8(index + 11), B: uint8(index + 21)}
		colors[index*2+1] = gradient.RGB{R: uint8(index + 101), G: uint8(index + 111), B: uint8(index + 121)}
	}
	for band := 0; band < 6; band++ {
		if got := interpolate(band, 0, colors); got != colors[band*2] {
			t.Errorf("band %d start = %v, want color %d: %v", band, got, band*2+1, colors[band*2])
		}
		if got := interpolate(band, 1, colors); got != colors[band*2+1] {
			t.Errorf("band %d end = %v, want color %d: %v", band, got, band*2+2, colors[band*2+1])
		}
	}
}

func TestBandInterpolationClamps(t *testing.T) {
	colors := DefaultColors()
	if got := interpolate(0, -10, colors); got != colors[0] {
		t.Errorf("negative position = %v, want %v", got, colors[0])
	}
	if got := interpolate(0, 10, colors); got != colors[1] {
		t.Errorf("position above one = %v, want %v", got, colors[1])
	}
}

func TestVisibleWidthIgnoresANSI(t *testing.T) {
	logo := Render(DefaultColors(), true, true)
	for index, line := range logo.Lines {
		if got, want := VisibleWidth(line), logo.Widths[index]; got != want {
			t.Errorf("line %d visible width = %d, want %d", index, got, want)
		}
	}
}
