//go:build linux

package modules

import (
	"errors"
	"strconv"
	"strings"
)

type chassistype struct{}

func init() { Register(chassistype{}) }

func (chassistype) Name() string { return "chassistype" }

func (chassistype) Run(ctx Context) ([]Entry, error) {
	raw := dmiValue("chassis_type")
	if raw == "" {
		return nil, Unsupported("chassis type unavailable")
	}
	names := chassisTypeNames(raw)
	if len(names) == 0 {
		return nil, errors.New("invalid chassis type " + raw)
	}
	value := strings.Join(names, " / ")
	if vendor := dmiValue("chassis_vendor"); vendor != "" {
		value += " (" + vendor + ")"
	}
	return Single(ctx, "Chassis Type", value), nil
}

// chassisTypeNames maps one or more space-separated SMBIOS chassis numbers.
func chassisTypeNames(raw string) []string {
	var names []string
	seen := map[string]bool{}
	for _, field := range strings.Fields(raw) {
		number, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		name := chassisTypeName(number)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
