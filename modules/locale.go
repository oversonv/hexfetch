package modules

import (
	"os"
)

type locale struct{}

func init() { Register(locale{}) }

func (locale) Name() string { return "locale" }

func (locale) Run(ctx Context) ([]Entry, error) {
	for _, variable := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(variable); value != "" {
			return Single(ctx, "Locale", value), nil
		}
	}
	return nil, Unsupported("locale not set")
}
