//go:build linux

package modules

import (
	"strings"
)

type bios struct{}

func init() { Register(bios{}) }

func (bios) Name() string { return "bios" }

func (bios) Run(ctx Context) ([]Entry, error) {
	vendor := dmiValue("bios_vendor")
	version := dmiValue("bios_version")
	date := dmiValue("bios_date")
	value := strings.TrimSpace(vendor + " " + version)
	if value == "" {
		return nil, Unsupported("BIOS information unavailable")
	}
	if date != "" {
		value += " (" + date + ")"
	}
	return Single(ctx, "BIOS", value), nil
}
