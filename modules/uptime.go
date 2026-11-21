package modules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type uptime struct{}

func init() { Register(uptime{}) }

func (uptime) Name() string { return "uptime" }

func readUptime() (int, error) {
	text, readErr := ReadTrim(procPath("uptime"))
	var parseErr error
	if readErr == nil {
		field, _, _ := strings.Cut(text, " ")
		seconds, err := strconv.ParseFloat(field, 64)
		if err == nil && seconds > 0 {
			return int(seconds), nil
		}
		if err != nil {
			parseErr = fmt.Errorf("parse /proc/uptime: %w", err)
		} else {
			parseErr = errors.New("invalid /proc/uptime value")
		}
	}
	if total, ok := fallbackUptime(); ok {
		return total, nil
	}
	if readErr != nil {
		return 0, readErr
	}
	return 0, parseErr
}

func (uptime) Run(ctx Context) ([]Entry, error) {
	total, err := readUptime()
	if err != nil {
		return nil, err
	}
	days := total / 86400
	hours := total % 86400 / 3600
	mins := total % 3600 / 60
	var parts []string
	if days > 0 {
		parts = append(parts, plural(days, "day"))
	}
	if hours > 0 {
		parts = append(parts, plural(hours, "hour"))
	}
	if mins > 0 || len(parts) == 0 {
		parts = append(parts, plural(mins, "min"))
	}
	entries := Single(ctx, "Uptime", strings.Join(parts, ", "))
	entries[0].Fields = map[string]string{
		"days": strconv.Itoa(days), "hours": strconv.Itoa(hours),
		"minutes": strconv.Itoa(mins), "seconds": strconv.Itoa(total),
	}
	return entries, nil
}

func plural(n int, unit string) string {
	if n == 1 {
		return "1 " + unit
	}
	return strconv.Itoa(n) + " " + unit + "s"
}
