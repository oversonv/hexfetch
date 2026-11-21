package modules

import (
	"os"
	"strings"
)

type de struct{}

func init() { Register(de{}) }

func (de) Name() string { return "de" }

func (de) Run(ctx Context) ([]Entry, error) {
	for _, variable := range []string{"XDG_CURRENT_DESKTOP", "DESKTOP_SESSION", "GDMSESSION"} {
		value := os.Getenv(variable)
		if value == "" {
			continue
		}
		if index := strings.LastIndex(value, ":"); index >= 0 {
			value = value[index+1:]
		}
		return Single(ctx, "DE", value), nil
	}
	return nil, Unsupported("desktop environment unknown")
}
