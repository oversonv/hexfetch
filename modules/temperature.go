//go:build linux

package modules

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type temperature struct{}

func init() { Register(temperature{}) }

func (temperature) Name() string { return "temperature" }

func (temperature) Run(ctx Context) ([]Entry, error) {
	sensors, err := collectTemperatures()
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(sensors))
	for _, sensor := range sensors {
		entries = append(entries, Entry{Key: sensor.label, Value: formatTemp(sensor.milliC)})
	}
	return entries, nil
}

type temperatureSensor struct {
	label  string
	milliC int64
}

func collectTemperatures() ([]temperatureSensor, error) {
	hwmons, _ := filepath.Glob(sysPath("class/hwmon/hwmon*"))
	var sensors []temperatureSensor
	for _, hwmon := range hwmons {
		name, _ := readTrimmed(filepath.Join(hwmon, "name"))
		if name == "" {
			name = filepath.Base(hwmon)
		}
		inputs, _ := filepath.Glob(filepath.Join(hwmon, "temp*_input"))
		for _, input := range inputs {
			raw, err := readTrimmed(input)
			if err != nil {
				continue
			}
			milliC, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
			if err != nil || !validTemperature(milliC) {
				continue
			}
			label := sensorLabel(input, name)
			sensors = append(sensors, temperatureSensor{label: label, milliC: milliC})
		}
	}
	if len(sensors) == 0 {
		return nil, Unsupported("temperature sensors unavailable")
	}
	sort.SliceStable(sensors, func(i, j int) bool {
		return sensors[i].label < sensors[j].label
	})
	if len(sensors) > 12 {
		sensors = sensors[:12]
	}
	return sensors, nil
}

func validTemperature(milliC int64) bool {
	return milliC > 0 && milliC <= 200_000
}

func sensorLabel(input, fallback string) string {
	base := strings.TrimSuffix(input, "_input")
	label, err := readTrimmed(base + "_label")
	if err == nil && cleanValue(label) != "" {
		return cleanValue(label)
	}
	return fallback
}
