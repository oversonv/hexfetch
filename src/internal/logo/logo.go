package logo

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

const DefaultName = "linux"

type Logo struct {
	Lines  []string
	Widths []int
	Width  int
}

func Load(name string, color, trueColor bool) (Logo, error) {
	text, err := readArt(name)
	if err != nil {
		return Logo{}, err
	}
	lines := strings.Split(text, "\n")
	return paintClassic(lines, readPalette(name, trueColor), color), nil
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
