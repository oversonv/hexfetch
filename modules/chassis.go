//go:build linux

package modules

import (
	"errors"
	"strconv"
)

type chassis struct{}

func init() { Register(chassis{}) }

func (chassis) Name() string { return "chassis" }

func (chassis) Run(ctx Context) ([]Entry, error) {
	raw := dmiValue("chassis_type")
	if raw == "" {
		return nil, Unsupported("chassis type unavailable")
	}
	number, err := strconv.Atoi(raw)
	if err != nil {
		return nil, errors.New("invalid chassis type " + raw)
	}
	name := chassisTypeName(number)
	if name == "" {
		name = "Type " + raw
	}
	return Single(ctx, "Chassis", name), nil
}

var chassisTypes = map[int]string{
	1:  "Other",
	2:  "Unknown",
	3:  "Desktop",
	4:  "Low Profile Desktop",
	5:  "Pizza Box",
	6:  "Mini Tower",
	7:  "Tower",
	8:  "Portable",
	9:  "Laptop",
	10: "Notebook",
	11: "Hand Held",
	12: "Docking Station",
	13: "All In One",
	14: "Sub Notebook",
	15: "Space-saving",
	16: "Lunch Box",
	17: "Main Server Chassis",
	18: "Expansion Chassis",
	19: "Sub Chassis",
	20: "Bus Expansion Chassis",
	21: "Peripheral Chassis",
	22: "RAID Chassis",
	23: "Rack Mount Chassis",
	24: "Sealed-case PC",
	25: "Multi-system",
	26: "CompactPCI",
	27: "AdvancedTCA",
	28: "Blade",
	29: "Blade Enclosure",
	30: "Tablet",
	31: "Convertible",
	32: "Detachable",
	33: "IoT Gateway",
	34: "Embedded PC",
	35: "Mini PC",
	36: "Stick PC",
}

func chassisTypeName(number int) string {
	return chassisTypes[number]
}
