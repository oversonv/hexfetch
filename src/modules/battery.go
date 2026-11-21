//go:build linux

package modules

import (
	"path/filepath"
	"strconv"
	"strings"
)

type battery struct{}

func init() { Register(battery{}) }

func (battery) Name() string { return "battery" }

func (battery) Run(ctx Context) ([]Entry, error) {
	paths, _ := filepath.Glob(sysPath("class/power_supply/BAT*"))
	for _, path := range paths {
		files := map[string]string{}
		for _, name := range []string{
			"capacity", "status", "energy_now", "energy_full", "power_now",
			"charge_now", "charge_full", "current_now", "voltage_now",
		} {
			if value, err := readTrimmed(filepath.Join(path, name)); err == nil {
				files[name] = value
			}
		}
		info, ok := batteryFromFiles(files)
		if !ok {
			continue
		}
		value := strconv.Itoa(info.percent) + "%"
		if info.status != "" {
			value += " (" + info.status + ")"
		}
		if info.hasRemaining {
			value += ", " + formatDuration(info.remaining)
			if info.status == "Charging" {
				value += " until full"
			} else {
				value += " remaining"
			}
		}
		entries := Single(ctx, "Battery", value)
		entries[0].Fields = map[string]string{
			"percent": strconv.Itoa(info.percent), "status": info.status,
		}
		if info.hasRemaining {
			entries[0].Fields["remaining"] = formatDuration(info.remaining)
		}
		return entries, nil
	}
	return nil, Unsupported("battery information unavailable")
}

type batteryInfo struct {
	percent      int
	status       string
	remaining    int64
	hasRemaining bool
}

func batteryFromFiles(files map[string]string) (batteryInfo, bool) {
	var info batteryInfo
	if status, ok := files["status"]; ok {
		info.status = strings.TrimSpace(status)
	}
	if capacity, err := strconv.ParseFloat(files["capacity"], 64); err == nil && capacity >= 0 {
		info.percent = int(capacity + 0.5)
	}
	if info.percent == 0 {
		if now, ok := number(files, "energy_now"); ok {
			if full, ok := number(files, "energy_full"); ok && full > 0 {
				info.percent = int(now/full*100 + 0.5)
			}
		} else if now, ok := number(files, "charge_now"); ok {
			if full, ok := number(files, "charge_full"); ok && full > 0 {
				info.percent = int(now/full*100 + 0.5)
			}
		}
	}
	if _, ok := files["capacity"]; !ok && info.percent == 0 {
		if now, ok := number(files, "energy_now"); !ok || now == 0 {
			if now, ok := number(files, "charge_now"); !ok || now == 0 {
				return info, false
			}
		}
	}
	info.percent = min(max(info.percent, 0), 100)

	energy, hasEnergy := number(files, "energy_now")
	energyFull, hasEnergyFull := number(files, "energy_full")
	charge, hasCharge := number(files, "charge_now")
	chargeFull, hasChargeFull := number(files, "charge_full")

	power, hasPower := number(files, "power_now")
	var current float64
	hasCurrent := false
	if value, ok := number(files, "current_now"); ok {
		current, hasCurrent = value, true
	}
	voltage, hasVoltage := number(files, "voltage_now")

	if !hasPower && hasCurrent && hasVoltage && voltage > 0 {
		power = current * voltage / 1_000_000
		hasPower = true
	}

	if info.status == "Full" {
		return info, true
	}
	if hasPower && power > 0 {
		switch {
		case (info.status == "Discharging" || info.status == "") && hasEnergy:
			info.remaining = int64(energy / power * 3600)
			info.hasRemaining = true
		case info.status == "Charging" && hasEnergy && hasEnergyFull && energyFull > energy:
			info.remaining = int64((energyFull - energy) / power * 3600)
			info.hasRemaining = true
		case (info.status == "Discharging" || info.status == "") && hasCharge:
			info.remaining = int64(charge / power * 3600)
			info.hasRemaining = true
		case info.status == "Charging" && hasCharge && hasChargeFull && chargeFull > charge:
			info.remaining = int64((chargeFull - charge) / power * 3600)
			info.hasRemaining = true
		}
	} else if hasCurrent && current > 0 {
		switch {
		case (info.status == "Discharging" || info.status == "") && hasCharge:
			info.remaining = int64(charge / current * 3600)
			info.hasRemaining = true
		case info.status == "Charging" && hasCharge && hasChargeFull && chargeFull > charge:
			info.remaining = int64((chargeFull - charge) / current * 3600)
			info.hasRemaining = true
		}
	}
	return info, true
}

func number(files map[string]string, key string) (float64, bool) {
	value, ok := files[key]
	if !ok {
		return 0, false
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, false
	}
	return number, true
}
