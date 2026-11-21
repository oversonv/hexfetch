package render

import (
	"fmt"
	"strings"
)

const (
	paletteColors = 16
	// minTextWidth: минимум колонок справа от логотипа; если меньше,
	// логотип скрывается, чтобы текст и палитра не пропадали.
	minTextWidth = 8
)

// paletteRows строит палитру 16 цветов под avail колонок.
// avail <= 0: ширина неизвестна, полный вид 2x8 по 3 колонки.
// Ячейка сужается 3 -> 2 -> 1, затем число ячеек в ряду делится пополам
// (8 -> 4 -> 2 -> 1), чтобы обычные и яркие цвета не смешивались в рядах.
func paletteRows(avail int) ([]string, []int) {
	cell, perRow := 3, 8
	if avail > 0 {
		switch {
		case avail >= 8*3:
		case avail >= 8*2:
			cell = 2
		default:
			cell = 1
			for perRow > 1 && perRow > avail {
				perRow /= 2
			}
		}
	}
	block := strings.Repeat("\u2588", cell)
	var lines []string
	var widths []int
	for start := 0; start < paletteColors; start += perRow {
		var line strings.Builder
		for i := start; i < start+perRow; i++ {
			fmt.Fprintf(&line, "\x1b[38;5;%d;48;5;%dm%s\x1b[0m", i, i, block)
		}
		lines = append(lines, line.String())
		widths = append(widths, perRow*cell)
	}
	return lines, widths
}
