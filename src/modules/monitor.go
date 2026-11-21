//go:build linux

package modules

import (
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

type monitor struct{}

func init() { Register(monitor{}) }

func (monitor) Name() string { return "monitor" }

func (monitor) Run(ctx Context) ([]Entry, error) {
	connectors, _ := filepath.Glob(sysPath("class/drm/card*-*"))
	var entries []Entry
	seen := map[string]bool{}
	for _, connector := range connectors {
		status, err := readTrimmed(filepath.Join(connector, "status"))
		if err != nil || status != "connected" {
			continue
		}
		data, err := readFile(filepath.Join(connector, "edid"))
		if err != nil || len(data) == 0 {
			continue
		}
		info, ok := parseEDID(data)
		if !ok {
			continue
		}
		value := info.describe()
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		entries = append(entries, Entry{Key: "Monitor", Value: value})
	}
	if len(entries) == 0 {
		return nil, Unsupported("monitor information unavailable")
	}
	return entries, nil
}

type edidInfo struct {
	manufacturer string
	name         string
	year         int
	widthCM      int
	heightCM     int
}

func (info edidInfo) describe() string {
	parts := []string{}
	if info.manufacturer != "" && info.name != "" {
		parts = append(parts, info.manufacturer+" "+info.name)
	} else if info.name != "" {
		parts = append(parts, info.name)
	} else if info.manufacturer != "" {
		parts = append(parts, info.manufacturer)
	}
	var details []string
	if info.widthCM > 0 && info.heightCM > 0 {
		diagonal := math.Sqrt(float64(info.widthCM*info.widthCM+info.heightCM*info.heightCM)) / 2.54
		details = append(details, strconv.FormatFloat(diagonal, 'f', 1, 64)+" in")
	}
	if info.year > 0 {
		details = append(details, strconv.Itoa(info.year))
	}
	value := strings.Join(parts, " ")
	if len(details) > 0 {
		if value != "" {
			value += " "
		}
		value += "(" + strings.Join(details, ", ") + ")"
	}
	return strings.TrimSpace(value)
}

func parseEDID(data []byte) (edidInfo, bool) {
	var info edidInfo
	if len(data) < 128 {
		return info, false
	}
	if data[0] != 0x00 || data[1] != 0xff || data[7] != 0x00 {
		return info, false
	}
	info.manufacturer = edidManufacturer(data[8], data[9])
	info.year = int(data[16]) + 1990
	if data[21] > 0 && data[22] > 0 {
		info.widthCM = int(data[21])
		info.heightCM = int(data[22])
	}
	for offset := 54; offset+18 <= 126; offset += 18 {
		block := data[offset : offset+18]
		if block[0] != 0 || block[1] != 0 {
			continue
		}
		switch block[3] {
		case 0xfc:
			info.name = edidString(block[5:18])
		case 0xff:
			if info.name == "" {
				info.name = edidString(block[5:18])
			}
		}
	}
	return info, true
}

func edidManufacturer(high, low byte) string {
	value := uint16(high)<<8 | uint16(low)
	letters := []byte{
		byte((value>>10)&0x1f) + 'A' - 1,
		byte((value>>5)&0x1f) + 'A' - 1,
		byte(value&0x1f) + 'A' - 1,
	}
	for _, letter := range letters {
		if letter < 'A' || letter > 'Z' {
			return ""
		}
	}
	return string(letters)
}

func edidString(raw []byte) string {
	for index, value := range raw {
		if value == 0x0a || value == 0x00 {
			raw = raw[:index]
			break
		}
	}
	return strings.TrimSpace(string(raw))
}
