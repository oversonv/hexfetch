//go:build linux

package modules

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type osage struct{}

func init() { Register(osage{}) }

func (osage) Name() string { return "osage" }

var osAgeCandidates = []string{
	"/etc/machine-id",
	"/var/lib/dbus/machine-id",
	"/etc/hostname",
	"/etc/passwd",
	"/lost+found",
	"/boot",
	"/var/log",
}

func (osage) Run(ctx Context) ([]Entry, error) {
	oldest, found := oldestFileTime(osAgeCandidates)
	if !found {
		return nil, Unsupported("install date unavailable")
	}
	days := int(time.Since(oldest).Hours() / 24)
	if days < 0 {
		return nil, errors.New("install date unavailable")
	}
	return Single(ctx, "OS Age", formatAge(days)), nil
}

// oldestFileTime picks the oldest modification time among candidate files.
func oldestFileTime(paths []string) (time.Time, bool) {
	var oldest time.Time
	found := false
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !found || info.ModTime().Before(oldest) {
			oldest = info.ModTime()
			found = true
		}
	}
	return oldest, found
}

// formatAge renders a day count the way fastfetch shows OS age.
func formatAge(days int) string {
	if days < 0 {
		days = 0
	}
	years := days / 365
	months := (days % 365) / 30
	daysLeft := (days % 365) % 30

	var parts []string
	if years > 0 {
		parts = append(parts, pluralUnit(years, "year"))
	}
	if months > 0 {
		parts = append(parts, pluralUnit(months, "month"))
	}
	if daysLeft > 0 || len(parts) == 0 {
		parts = append(parts, pluralUnit(daysLeft, "day"))
	}
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, ", ")
}

func pluralUnit(value int, unit string) string {
	if value == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %ss", value, unit)
}
