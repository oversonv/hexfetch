package modules

import (
	"strings"
	"time"
)

type mediaModule struct {
	name, key, format string
}

func (m mediaModule) Name() string { return m.name }

func (m mediaModule) Run(ctx Context) ([]Entry, error) {
	out, err := runCommandPath(3*time.Second, "playerctl", "metadata", "--format", m.format)
	if err != nil {
		return nil, err
	}
	value := cleanMediaValue(out)
	if value == "" {
		return nil, Unsupported("nothing is playing")
	}
	return Single(ctx, m.key, value), nil
}

func init() {
	Register(mediaModule{"player", "Player", "{{playerName}}"})
	Register(mediaModule{"media", "Media", "{{playerName}}: {{title}} - {{artist}}"})
}

// cleanMediaValue drops empty slots such as ":  -" from a playerctl format.
func cleanMediaValue(value string) string {
	value = strings.TrimSpace(value)
	pieces := strings.Split(value, " - ")
	var kept []string
	for _, piece := range pieces {
		piece = strings.TrimSpace(strings.Trim(strings.TrimSpace(piece), "-"))
		if piece == "" || piece == ":" {
			continue
		}
		if _, rest, ok := strings.Cut(piece, ": "); ok {
			if strings.TrimSpace(strings.Trim(rest, "-")) == "" {
				continue
			}
		} else if strings.HasSuffix(piece, ":") {
			continue
		}
		kept = append(kept, piece)
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, " - ")
}
