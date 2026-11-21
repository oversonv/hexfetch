// Package logo loads the embedded ASCII logos and paints them.
//
// Every logo is a text file in logo/ascii/<letter>/<name>.txt where "$1".."$9"
// switch the active color slot and "$$" prints a literal "$". Two painters share
// that format:
//
//   - classic logos use flat colors per slot taken from logo/colors.txt;
//   - profile logos (currently only nixos_average) map each slot to a band with its own
//     gradient geometry and two configurable endpoint colors.
package logo

import (
	"math"
	"strings"

	"github.com/mattn/go-runewidth"

	"hexfetch/internal/gradient"
)

// DefaultName is the logo used when nothing else is configured.
const DefaultName = "nixos_average"

type Logo struct {
	Lines  []string
	Widths []int
	Width  int
}

const scaleX = 42.0 / 28.0
const scaleY = 19.0 / 12.0

type bandGradient struct {
	dx float64
	dy float64
	t0 float64
	t1 float64
}

// gradients holds the geometry of the nixos_average bands. Band n is color slot n+1 in
// logo/ascii/n/nixos_average.txt and uses colors[2n] (start) and colors[2n+1] (end).
var gradients = [...]bandGradient{
	{-0.816, -0.578, -25.36, -8.16},
	{-0.781, -0.625, -35.76, -16.4},
	{-0.84, 0.542, -26.75, -8.59},
	{-0.941, 0.339, -25.31, -7.43},
	{-0.512, -0.859, -43.39, -25.04},
	{-0.603, 0.798, 0.53, 20.91},
}

// profiles lists the logos that are painted with band gradients.
var profiles = map[string]struct{}{
	DefaultName: {},
}

// HasProfile reports whether the logo is painted with the 12 configurable
// gradient endpoint colors instead of the flat classic palette.
func HasProfile(name string) bool {
	_, ok := profiles[name]
	return ok
}

func DefaultColors() [12]gradient.RGB {
	return [12]gradient.RGB{
		{R: 133, G: 104, B: 242},
		{R: 206, G: 177, B: 240},
		{R: 107, G: 106, B: 255},
		{R: 79, G: 185, B: 254},
		{R: 206, G: 177, B: 240},
		{R: 133, G: 104, B: 242},
		{R: 79, G: 185, B: 254},
		{R: 107, G: 106, B: 255},
		{R: 206, G: 177, B: 240},
		{R: 133, G: 104, B: 242},
		{R: 107, G: 106, B: 255},
		{R: 79, G: 185, B: 254},
	}
}

// Render paints the nixos_average logo. It cannot fail: the logo is embedded.
func Render(colors [12]gradient.RGB, color, trueColor bool) Logo {
	loaded, err := Load(DefaultName, colors, color, trueColor)
	if err != nil {
		return Logo{}
	}
	return loaded
}

// Load paints the named embedded logo. colors is used only by profile logos.
func Load(name string, colors [12]gradient.RGB, color, trueColor bool) (Logo, error) {
	text, err := readArt(name)
	if err != nil {
		return Logo{}, err
	}
	lines := strings.Split(text, "\n")
	if HasProfile(name) {
		return paintProfile(lines, colors, color, trueColor), nil
	}
	return paintClassic(lines, readPalette(name, trueColor), color), nil
}

func paintProfile(lines []string, colors [12]gradient.RGB, color, trueColor bool) Logo {
	var loaded Logo
	current := 0
	for y, line := range lines {
		cells, width, next := parseCells(line, current)
		current = next
		var output strings.Builder
		for _, c := range cells {
			if c.r == ' ' {
				output.WriteRune(c.r)
				continue
			}
			if color {
				value := colorAt(c.x, y, c.slot, colors)
				output.WriteString("\x1b[")
				output.WriteString(value.SGR(trueColor))
				output.WriteByte('m')
			}
			output.WriteRune(c.r)
		}
		if color {
			output.WriteString("\x1b[0m")
		}
		loaded.Lines = append(loaded.Lines, output.String())
		loaded.Widths = append(loaded.Widths, width)
		loaded.Width = max(loaded.Width, width)
	}
	return loaded
}

func colorAt(x, y, band int, colors [12]gradient.RGB) gradient.RGB {
	band = min(max(band, 0), len(gradients)-1)
	g := gradients[band]
	px := float64(x) * scaleX
	py := float64(y) * scaleY
	t := (px*g.dx + py*2*g.dy - g.t0) / (g.t1 - g.t0)
	return interpolate(band, t, colors)
}

func interpolate(index int, t float64, colors [12]gradient.RGB) gradient.RGB {
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	from := colors[index*2]
	to := colors[index*2+1]
	return gradient.RGB{
		R: uint8(math.RoundToEven(float64(from.R) + float64(int(to.R)-int(from.R))*t)),
		G: uint8(math.RoundToEven(float64(from.G) + float64(int(to.G)-int(from.G))*t)),
		B: uint8(math.RoundToEven(float64(from.B) + float64(int(to.B)-int(from.B))*t)),
	}
}

type cell struct {
	r    rune
	slot int
	x    int
}

// parseCells splits one art line into cells. current is the color slot that is
// active at the start of the line; the slot active at its end is returned.
func parseCells(line string, current int) ([]cell, int, int) {
	var cells []cell
	var plain strings.Builder
	column := 0
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '$' && i+1 < len(runes) && runes[i+1] == '$':
			cells = append(cells, cell{'$', current, column})
			plain.WriteRune('$')
			column++
			i++
		case r == '$' && i+1 < len(runes) && runes[i+1] >= '1' && runes[i+1] <= '9':
			current = int(runes[i+1] - '1')
			i++
		default:
			cells = append(cells, cell{r, current, column})
			plain.WriteRune(r)
			column += runewidth.RuneWidth(r)
		}
	}
	return cells, runewidth.StringWidth(plain.String()), current
}

func StripANSI(value string) string {
	var output strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] == 0x1b && i+1 < len(value) && value[i+1] == '[' {
			i += 2
			for i < len(value) && (value[i] < '@' || value[i] > '~') {
				i++
			}
			continue
		}
		output.WriteByte(value[i])
	}
	return output.String()
}

func VisibleWidth(value string) int {
	return runewidth.StringWidth(StripANSI(value))
}
