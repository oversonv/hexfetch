//go:build linux

package modules

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHardwareModulesWithoutDevicesAreUnsupported(t *testing.T) {
	oldSysRoot := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = oldSysRoot }()

	cases := []struct {
		name   string
		module Module
	}{
		{name: "battery", module: battery{}},
		{name: "bios", module: bios{}},
		{name: "bluetooth", module: bluetooth{}},
		{name: "board", module: board{}},
		{name: "brightness", module: brightness{}},
		{name: "camera", module: camera{}},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.module.Run(Context{})
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Fatalf("Run() error = %v, want errors.ErrUnsupported", err)
			}
		})
	}
}

func TestBrightnessInvalidDataIsFailure(t *testing.T) {
	oldSysRoot := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = oldSysRoot }()

	backlight := filepath.Join(sysRoot, "class", "backlight", "test")
	if err := os.MkdirAll(backlight, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlight, "brightness"), []byte("invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backlight, "max_brightness"), []byte("100\n"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := (brightness{}).Run(Context{})
	if err == nil {
		t.Fatal("Run() returned nil error for invalid brightness data")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("invalid brightness data was classified as unsupported: %v", err)
	}
	if !strings.Contains(err.Error(), "parse brightness") {
		t.Fatalf("error %q does not identify the parse failure", err)
	}
}
