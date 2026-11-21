package render

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"

	"hexfetch/internal/logo"
)

// clip обрезает строку с ANSI-кодами до width колонок и сбрасывает стиль,
// чтобы фон не протекал на перенос строки.
func clip(s string, width int) string {
	if width <= 0 || logo.VisibleWidth(s) <= width {
		return s
	}
	var out strings.Builder
	used := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < '@' || s[j] > '~') {
				j++
			}
			if j < len(s) {
				j++
			}
			out.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		w := runewidth.RuneWidth(r)
		if used+w > width {
			break
		}
		out.WriteString(s[i : i+size])
		used += w
		i += size
	}
	out.WriteString("\x1b[0m")
	return out.String()
}
