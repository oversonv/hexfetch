package gradient

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

type RGB struct {
	R uint8
	G uint8
	B uint8
}

func ParseStop(value string) (RGB, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ";")
	if len(parts) != 5 || parts[0] != "38" || parts[1] != "2" {
		return RGB{}, fmt.Errorf("invalid color %q: expected 38;2;R;G;B", value)
	}
	var channels [3]uint8
	for i, part := range parts[2:] {
		if part == "" {
			return RGB{}, fmt.Errorf("invalid color %q: channels must be integers from 0 to 255", value)
		}
		for _, character := range part {
			if character < '0' || character > '9' {
				return RGB{}, fmt.Errorf("invalid color %q: channels must be integers from 0 to 255", value)
			}
		}
		n, err := strconv.ParseUint(part, 10, 8)
		if err != nil {
			return RGB{}, fmt.Errorf("invalid color %q: channels must be integers from 0 to 255", value)
		}
		channels[i] = uint8(n)
	}
	return RGB{R: channels[0], G: channels[1], B: channels[2]}, nil
}

func (c RGB) ANSI256() int {
	if c.R == c.G && c.G == c.B {
		switch {
		case c.R < 8:
			return 16
		case c.R > 248:
			return 231
		}
		return min(232+int(math.Round(float64(c.R-8)/247*24)), 255)
	}
	return 16 + 36*cubeLevel(c.R) + 6*cubeLevel(c.G) + cubeLevel(c.B)
}

func cubeLevel(value uint8) int {
	return int(math.Round(float64(value) / 255 * 5))
}

func (c RGB) SGR(trueColor bool) string {
	if trueColor {
		return fmt.Sprintf("38;2;%d;%d;%d", c.R, c.G, c.B)
	}
	return fmt.Sprintf("38;5;%d", c.ANSI256())
}

func TrueColor() bool {
	if value := strings.ToLower(os.Getenv("COLORTERM")); value == "truecolor" || value == "24bit" {
		return true
	}
	switch os.Getenv("TERM") {
	case "linux", "dumb", "vt100", "vt220", "ansi", "screen":
		return false
	}
	return true
}
