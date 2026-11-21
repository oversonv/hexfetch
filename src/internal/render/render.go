package render

import (
	"bufio"
	"io"
	"strings"

	"hexfetch/src/internal/logo"
	"hexfetch/src/modules"
)

type Options struct {
	Logo            logo.Logo
	Position        string
	Padding         int
	Separator       string
	SeparatorLength int
	KeyType         string
	KeyColor        string
	TitleColor      string
	OutputColor     string
	SeparatorColor  string
	Color           bool
	Width           int
}

func Render(w io.Writer, entries []modules.Entry, o Options) error {
	o, avail := layout(o)
	lines, widths := buildLines(entries, o, avail)
	out := bufio.NewWriter(w)
	if len(o.Logo.Lines) == 0 {
		for _, line := range lines {
			out.WriteString(clip(line, o.Width))
			out.WriteByte('\n')
		}
		return out.Flush()
	}
	textWidth := 0
	for _, width := range widths {
		textWidth = max(textWidth, width)
	}
	for i := range max(len(lines), len(o.Logo.Lines)) {
		logoLine, logoWidth := at(o.Logo.Lines, o.Logo.Widths, i)
		textLine, textLineWidth := at(lines, widths, i)
		var row string
		if o.Position == "right" {
			row = join(textLine, textLineWidth, logoLine, textWidth, o.Padding)
		} else {
			row = join(logoLine, logoWidth, textLine, o.Logo.Width, o.Padding)
		}
		out.WriteString(clip(row, o.Width))
		out.WriteByte('\n')
	}
	return out.Flush()
}

// layout считает, сколько колонок остаётся под текст. Если справа от
// логотипа меньше minTextWidth колонок, логотип убирается целиком.
// avail == 0 означает неизвестную ширину.
func layout(o Options) (Options, int) {
	if o.Width <= 0 {
		return o, 0
	}
	if len(o.Logo.Lines) == 0 || o.Position == "right" {
		return o, o.Width
	}
	avail := o.Width - o.Logo.Width - o.Padding
	if avail < minTextWidth {
		o.Logo = logo.Logo{}
		return o, o.Width
	}
	return o, avail
}

func LogoFits(l logo.Logo, padding, width int) bool {
	return width <= 0 || l.Width+padding+minTextWidth <= width
}

func buildLines(entries []modules.Entry, o Options, avail int) ([]string, []int) {
	var lines []string
	var widths []int
	add := func(text string, width int) {
		lines = append(lines, text)
		widths = append(widths, width)
	}
	titleWidth := 0
	contentWidth := 0
	for _, entry := range entries {
		switch entry.Kind {
		case modules.KindTitle:
			if titleWidth == 0 {
				titleWidth = logo.VisibleWidth(entry.Value)
			}
		case modules.KindRaw:
			contentWidth = max(contentWidth, logo.VisibleWidth(entry.Value))
		case modules.KindPalette:
			if o.Color {
				_, paletteWidths := paletteRows(avail)
				for _, width := range paletteWidths {
					contentWidth = max(contentWidth, width)
				}
			}
		}
	}
	for _, entry := range entries {
		if entry.Kind == modules.KindNormal {
			width := logo.VisibleWidth(keyLabel(entry, o.KeyType)) + logo.VisibleWidth(o.Separator) + logo.VisibleWidth(entry.Value)
			contentWidth = max(contentWidth, width)
		}
	}
	if titleWidth == 0 {
		titleWidth = contentWidth
	}
	if o.SeparatorLength > 0 {
		titleWidth = o.SeparatorLength
	}
	for _, entry := range entries {
		switch entry.Kind {
		case modules.KindTitle:
			add(paint(entry.Value, o.TitleColor, o.Color), logo.VisibleWidth(entry.Value))
		case modules.KindSeparator:
			char := entry.Value
			if char == "" {
				char = "-"
			}
			text := repeatToWidth(char, titleWidth)
			add(paint(text, o.SeparatorColor, o.Color), logo.VisibleWidth(text))
		case modules.KindRaw:
			add(entry.Value, logo.VisibleWidth(entry.Value))
		case modules.KindBreak:
			add("", 0)
		case modules.KindPalette:
			if !o.Color {
				continue
			}
			rows, rowWidths := paletteRows(avail)
			for i := range rows {
				add(rows[i], rowWidths[i])
			}
		default:
			color := o.KeyColor
			if entry.KeyColor != nil {
				color = *entry.KeyColor
			}
			label := keyLabel(entry, o.KeyType)
			text := paint(label, color, o.Color) +
				paint(o.Separator, o.SeparatorColor, o.Color) +
				paint(entry.Value, o.OutputColor, o.Color)
			add(text, logo.VisibleWidth(label)+logo.VisibleWidth(o.Separator)+logo.VisibleWidth(entry.Value))
		}
	}
	return lines, widths
}

func keyLabel(entry modules.Entry, keyType string) string {
	if entry.KeyIcon == "" {
		return entry.Key
	}
	switch keyType {
	case "icon":
		return entry.KeyIcon
	case "both":
		return entry.KeyIcon + " " + entry.Key
	}
	return entry.Key
}

func repeatToWidth(value string, width int) string {
	unitWidth := logo.VisibleWidth(value)
	if value == "" || unitWidth <= 0 || width <= 0 {
		return ""
	}
	repeats := width / unitWidth
	remainder := width % unitWidth
	return strings.Repeat(value, repeats) + strings.Repeat(" ", remainder)
}

// paint оборачивает text в SGR-параметры sgr ("1;38;2;150;150;150").
func paint(text, sgr string, enabled bool) string {
	if !enabled || sgr == "" || text == "" {
		return text
	}
	return "\x1b[" + sgr + "m" + text + "\x1b[0m"
}

func at(lines []string, widths []int, i int) (string, int) {
	if i < len(lines) {
		return lines[i], widths[i]
	}
	return "", 0
}

func join(left string, leftWidth int, right string, column, padding int) string {
	if right == "" {
		return left
	}
	return left + strings.Repeat(" ", max(column-leftWidth+padding, 0)) + right
}
