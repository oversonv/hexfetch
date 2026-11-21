package logo

import "strings"

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
