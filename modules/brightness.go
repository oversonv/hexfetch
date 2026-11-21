//go:build linux

package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
)

type brightness struct{}

func init() { Register(brightness{}) }

func (brightness) Name() string { return "brightness" }

func (brightness) Run(ctx Context) ([]Entry, error) {
	paths, _ := filepath.Glob(sysPath("class/backlight/*"))
	var failures []error
	for _, path := range paths {
		current, err := readTrimmed(filepath.Join(path, "brightness"))
		if err != nil {
			if !errors.Is(err, errors.ErrUnsupported) {
				failures = append(failures, fmt.Errorf("read brightness for %q: %w", path, err))
			}
			continue
		}
		maximum, err := readTrimmed(filepath.Join(path, "max_brightness"))
		if err != nil {
			if !errors.Is(err, errors.ErrUnsupported) {
				failures = append(failures, fmt.Errorf("read maximum brightness for %q: %w", path, err))
			}
			continue
		}
		value, err := strconv.ParseUint(current, 10, 64)
		if err != nil {
			failures = append(failures, fmt.Errorf("parse brightness for %q: %w", path, err))
			continue
		}
		maxValue, err := strconv.ParseUint(maximum, 10, 64)
		if err != nil {
			failures = append(failures, fmt.Errorf("parse maximum brightness for %q: %w", path, err))
			continue
		}
		if maxValue == 0 {
			failures = append(failures, fmt.Errorf("maximum brightness for %q is zero", path))
			continue
		}
		return Single(ctx, "Brightness", formatPercent(value, maxValue)), nil
	}
	if len(failures) > 0 {
		return nil, fmt.Errorf("backlight information unavailable: %w", errors.Join(failures...))
	}
	return nil, Unsupported("backlight information unavailable")
}
