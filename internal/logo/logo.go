package logo

import (
	"math"
	"strings"

	"github.com/mattn/go-runewidth"

	"hexfetch/internal/gradient"
)

const logo = `       ▄▄▖     ▄▄▄  ▄▄▄
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

const scaleX = 42.0 / 28.0
const scaleY = 19.0 / 12.0

var bands = [...]string{
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

type bandGradient struct {
	dx float64
	dy float64
	t0 float64
	t1 float64
}

var gradients = [...]bandGradient{
	{-0.816, -0.578, -25.36, -8.16},
	{-0.781, -0.625, -35.76, -16.4},
	{-0.84, 0.542, -26.75, -8.59},
	{-0.941, 0.339, -25.31, -7.43},
	{-0.512, -0.859, -43.39, -25.04},
	{-0.603, 0.798, 0.53, 20.91},
}

type Logo struct {
	Lines  []string
	Widths []int
	Width  int
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

func Render(colors [12]gradient.RGB, color, trueColor bool) Logo {
	lines := strings.Split(logo, "\n")
	var loaded Logo
	for y, line := range lines {
		var output strings.Builder
		for x, character := range []rune(line) {
			if character == ' ' {
				output.WriteRune(character)
				continue
			}
			if color {
				value := colorAt(x, y, colors)
				output.WriteString("\x1b[")
				output.WriteString(value.SGR(trueColor))
				output.WriteByte('m')
			}
			output.WriteRune(character)
		}
		if color {
			output.WriteString("\x1b[0m")
		}
		loaded.Lines = append(loaded.Lines, output.String())
		width := runewidth.StringWidth(line)
		loaded.Widths = append(loaded.Widths, width)
		loaded.Width = max(loaded.Width, width)
	}
	return loaded
}

func colorAt(x, y int, colors [12]gradient.RGB) gradient.RGB {
	index := int(bands[y][x] - '0')
	g := gradients[index]
	px := float64(x) * scaleX
	py := float64(y) * scaleY
	t := (px*g.dx + py*2*g.dy - g.t0) / (g.t1 - g.t0)
	return interpolate(index, t, colors)
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
