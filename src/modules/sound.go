//go:build linux

package modules

import (
	"regexp"
	"strings"
)

type sound struct{}

func init() { Register(sound{}) }

func (sound) Name() string { return "sound" }

func (sound) Run(ctx Context) ([]Entry, error) {
	data, err := readFile(procPath("asound", "cards"))
	if err != nil {
		return nil, err
	}
	cards := parseAsoundCards(data)
	if len(cards) == 0 {
		return nil, Unsupported("sound card information unavailable")
	}
	return Single(ctx, "Sound", strings.Join(cards, ", ")), nil
}

var asoundLine = regexp.MustCompile(`^\s*\d+\s+\[([^\]]*)\]:\s*(.*)$`)

func parseAsoundCards(data []byte) []string {
	var cards []string
	for _, line := range strings.Split(string(data), "\n") {
		match := asoundLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		rest := strings.TrimSpace(match[2])
		rest = strings.TrimSuffix(rest, ":")
		if _, after, ok := strings.Cut(rest, " - "); ok {
			rest = after
		}
		rest = strings.TrimSpace(rest)
		if rest == "" {
			continue
		}
		cards = append(cards, rest)
	}
	return cards
}
