package config

import (
	"fmt"
	"strconv"
	"strings"
)

type Color string

var colorNames = map[string]int{
	"black":   0,
	"red":     1,
	"green":   2,
	"yellow":  3,
	"blue":    4,
	"magenta": 5,
	"cyan":    6,
	"white":   7,
}

func Palette(value int) Color {
	return Color("38;5;" + strconv.Itoa(value))
}

func ParseColor(value any) (Color, error) {
	switch raw := value.(type) {
	case int:
		return paletteColor(raw)
	case int8:
		return paletteColor(int(raw))
	case int16:
		return paletteColor(int(raw))
	case int32:
		return paletteColor(int(raw))
	case int64:
		if int64(int(raw)) != raw {
			return "", fmt.Errorf("invalid color: integer is out of range")
		}
		return paletteColor(int(raw))
	case uint:
		if uint64(raw) > uint64(^uint(0)>>1) {
			return "", fmt.Errorf("invalid color: integer is out of range")
		}
		return paletteColor(int(raw))
	case uint8:
		return paletteColor(int(raw))
	case uint16:
		return paletteColor(int(raw))
	case uint32:
		if uint64(raw) > uint64(^uint(0)>>1) {
			return "", fmt.Errorf("invalid color: integer is out of range")
		}
		return paletteColor(int(raw))
	case uint64:
		if raw > uint64(int(^uint(0)>>1)) {
			return "", fmt.Errorf("invalid color: integer is out of range")
		}
		return paletteColor(int(raw))
	case string:
		return parseColorString(raw)
	default:
		return "", fmt.Errorf("invalid color: expected an integer or string, got %T", value)
	}
}

func paletteColor(value int) (Color, error) {
	if value < 0 || value > 255 {
		return "", fmt.Errorf("invalid color %d: expected 0-255", value)
	}
	return Palette(value), nil
}

func parseColorString(value string) (Color, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	base := 30
	name := value
	for _, prefix := range []string{"light_", "bright_"} {
		if rest, ok := strings.CutPrefix(value, prefix); ok {
			name, base = rest, 90
		}
	}
	if number, ok := colorNames[name]; ok {
		return Color(strconv.Itoa(base + number)), nil
	}
	if number, err := strconv.Atoi(value); err == nil {
		return paletteColor(number)
	}
	if value == "" {
		return "", nil
	}
	parameters := strings.Split(value, ";")
	parsed := make([]int, len(parameters))
	for index, parameter := range parameters {
		if parameter == "" {
			return "", fmt.Errorf("invalid SGR color %q: empty parameter", value)
		}
		number, err := strconv.Atoi(parameter)
		if err != nil || number < 0 || number > 255 {
			return "", fmt.Errorf("invalid SGR color %q: parameters must be integers from 0 to 255", value)
		}
		parsed[index] = number
	}
	if err := validateSGR(parsed); err != nil {
		return "", fmt.Errorf("invalid SGR color %q: %w", value, err)
	}
	return Color(value), nil
}

func validateSGR(parameters []int) error {
	for index := 0; index < len(parameters); index++ {
		code := parameters[index]
		switch code {
		case 38, 48, 58:
			if index+1 >= len(parameters) {
				return fmt.Errorf("color code %d requires a color mode", code)
			}
			mode := parameters[index+1]
			switch mode {
			case 5:
				if index+2 >= len(parameters) {
					return fmt.Errorf("indexed color requires one channel")
				}
				index += 2
			case 2:
				if index+4 >= len(parameters) {
					return fmt.Errorf("truecolor requires three channels")
				}
				index += 4
			default:
				return fmt.Errorf("unsupported color mode %d", mode)
			}
		case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 39, 49, 50, 51, 52, 53, 54, 55, 59, 60, 61, 62, 63, 64, 65, 73, 74, 75:
		case 30, 31, 32, 33, 34, 35, 36, 37, 40, 41, 42, 43, 44, 45, 46, 47, 90, 91, 92, 93, 94, 95, 96, 97, 100, 101, 102, 103, 104, 105, 106, 107:
		default:
			return fmt.Errorf("unsupported SGR parameter %d", code)
		}
	}
	return nil
}
