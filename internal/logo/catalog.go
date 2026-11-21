package logo

import (
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/mattn/go-runewidth"

	"hexfetch/internal/gradient"
	logodata "hexfetch/logo"
)

type catalog struct {
	names []string
	paths map[string]string
}

var (
	catalogOnce sync.Once
	catalogData catalog
)

func loadCatalog() catalog {
	catalogOnce.Do(func() {
		catalogData.paths = make(map[string]string)
		_ = fs.WalkDir(logodata.FS, "ascii", func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			name, ok := strings.CutSuffix(entry.Name(), ".txt")
			if !ok {
				return nil
			}
			if _, taken := catalogData.paths[name]; !taken {
				catalogData.paths[name] = path
				catalogData.names = append(catalogData.names, name)
			}
			return nil
		})
		sort.Strings(catalogData.names)
	})
	return catalogData
}

// Names returns the sorted names of all embedded logos.
func Names() []string {
	return append([]string(nil), loadCatalog().names...)
}

func readArt(name string) (string, error) {
	path, ok := loadCatalog().paths[name]
	if !ok {
		return "", fs.ErrNotExist
	}
	data, err := fs.ReadFile(logodata.FS, path)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n"), nil
}

func normalize(s string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// Resolve maps a user supplied name (a logo name, an alias, "Arch Linux") to
// the name of an embedded logo.
func Resolve(names []string, aliases map[string]string, query string) (string, bool) {
	exact := make(map[string]string, len(names))
	for _, name := range names {
		key := normalize(name)
		if _, taken := exact[key]; !taken {
			exact[key] = name
		}
	}
	n := normalize(query)
	keys := []string{n, n + "linux", strings.TrimSuffix(n, "linux")}
	for _, key := range keys {
		if name, ok := exact[key]; ok && key != "" {
			return name, true
		}
	}
	for _, key := range keys {
		target, ok := aliases[key]
		if !ok || key == "" {
			continue
		}
		for _, name := range names {
			if name == target {
				return name, true
			}
		}
	}
	return "", false
}

// Detect picks a logo from the os-release fields ID, ID_LIKE and NAME.
func Detect(names []string, aliases map[string]string, release map[string]string) string {
	candidates := []string{release["ID"]}
	candidates = append(candidates, strings.Fields(release["ID_LIKE"])...)
	candidates = append(candidates, release["NAME"], "linux")
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if name, ok := Resolve(names, aliases, candidate); ok {
			return name
		}
	}
	return ""
}

// Aliases reads logo/aliases.txt (alias=logo). The key is normalize(alias);
// for duplicates the first line wins.
func Aliases() map[string]string {
	aliases := map[string]string{}
	data, err := fs.ReadFile(logodata.FS, "aliases.txt")
	if err != nil {
		return aliases
	}
	for _, line := range strings.Split(string(data), "\n") {
		alias, target, ok := strings.Cut(strings.TrimRight(line, "\r"), "=")
		key := normalize(alias)
		if !ok || key == "" {
			continue
		}
		if _, taken := aliases[key]; !taken {
			aliases[key] = strings.TrimSpace(target)
		}
	}
	return aliases
}

func defaultPalette() []string {
	var palette []string
	for _, n := range []int{7, 4, 2, 3, 1, 5, 6, 8, 9} {
		palette = append(palette, "38;5;"+strconv.Itoa(n))
	}
	return palette
}

// tokenToSGR converts a colors.txt token to an SGR parameter string.
func tokenToSGR(token string, trueColor bool) string {
	token = strings.TrimSpace(token)
	switch {
	case token == "default":
		return "39"
	case strings.HasPrefix(token, "sgr:"):
		return strings.TrimPrefix(token, "sgr:")
	case strings.Contains(token, ";"):
		parts := strings.Split(token, ";")
		if len(parts) != 3 {
			return ""
		}
		var rgb [3]uint8
		for i, part := range parts {
			value, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || value < 0 || value > 255 {
				return ""
			}
			rgb[i] = uint8(value)
		}
		return gradient.RGB{R: rgb[0], G: rgb[1], B: rgb[2]}.SGR(trueColor)
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 0 || n > 255 {
		return ""
	}
	return "38;5;" + strconv.Itoa(n)
}

// readPalette returns the flat SGR colors of a classic logo, one per slot.
func readPalette(name string, trueColor bool) []string {
	fallback := defaultPalette()
	data, err := fs.ReadFile(logodata.FS, "colors.txt")
	if err != nil {
		return fallback
	}
	target := normalize(name)
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimRight(line, "\r"), "=")
		if !ok || normalize(key) != target {
			continue
		}
		var colors []string
		found := false
		for _, field := range strings.Split(value, ",") {
			sgr := tokenToSGR(field, trueColor)
			found = found || sgr != ""
			colors = append(colors, sgr)
		}
		if found {
			return colors
		}
	}
	return fallback
}

func colorEscape(palette []string, index int) string {
	if index < 0 || index >= len(palette) || palette[index] == "" {
		return "\x1b[1m"
	}
	return "\x1b[1;" + palette[index] + "m"
}

func paintClassic(lines []string, palette []string, color bool) Logo {
	var loaded Logo
	current := 0
	for _, line := range lines {
		rendered, width, next := expand(line, palette, color, current)
		current = next
		loaded.Lines = append(loaded.Lines, rendered)
		loaded.Widths = append(loaded.Widths, width)
		loaded.Width = max(loaded.Width, width)
	}
	return loaded
}

func expand(line string, palette []string, color bool, current int) (string, int, int) {
	var out strings.Builder
	var plain strings.Builder
	if color {
		out.WriteString(colorEscape(palette, current))
	}
	runes := []rune(line)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case r == '$' && i+1 < len(runes) && runes[i+1] == '$':
			out.WriteRune('$')
			plain.WriteRune('$')
			i++
		case r == '$' && i+1 < len(runes) && runes[i+1] >= '1' && runes[i+1] <= '9':
			current = int(runes[i+1] - '1')
			i++
			if color {
				out.WriteString(colorEscape(palette, current))
			}
		default:
			out.WriteRune(r)
			plain.WriteRune(r)
		}
	}
	result := out.String()
	if color {
		result += "\x1b[0m"
	}
	return result, runewidth.StringWidth(plain.String()), current
}
