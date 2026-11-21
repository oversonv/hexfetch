//go:build linux

package modules

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOptionalModulesWithoutSourcesAreUnsupported(t *testing.T) {
	oldProcRoot := procRoot
	oldSysRoot := sysRoot
	oldEtcRoot := etcRoot
	procRoot = t.TempDir()
	sysRoot = t.TempDir()
	etcRoot = t.TempDir()
	t.Cleanup(func() {
		procRoot = oldProcRoot
		sysRoot = oldSysRoot
		etcRoot = oldEtcRoot
	})

	if err := os.WriteFile(procPath("mounts"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcPath("resolv.conf"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	for _, variable := range []string{"XDG_CURRENT_DESKTOP", "DESKTOP_SESSION", "GDMSESSION", "LC_ALL", "LC_MESSAGES", "LANG"} {
		t.Setenv(variable, "")
	}

	oldPath := os.Getenv("PATH")
	pathDir := t.TempDir()
	if err := os.Setenv("PATH", pathDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", oldPath)
	})

	cases := []struct {
		name   string
		module Module
	}{
		{name: "chassis", module: chassis{}},
		{name: "chassistype", module: chassistype{}},
		{name: "cpucache", module: cpucache{}},
		{name: "de", module: de{}},
		{name: "disks", module: disks{}},
		{name: "display", module: display{}},
		{name: "dns", module: dns{}},
		{name: "fans", module: fans{}},
		{name: "gpu", module: gpu{}},
		{name: "gpuusage", module: gpuusage{}},
		{name: "host", module: host{}},
		{name: "locale", module: locale{}},
		{name: "media", module: mediaModule{name: "media", key: "Media", format: "{{title}}"}},
		{name: "monitor", module: monitor{}},
		{name: "physicaldisk", module: physicaldisk{}},
		{name: "poweradapter", module: poweradapter{}},
		{name: "processes", module: processes{}},
		{name: "sound", module: sound{}},
		{name: "wifi", module: wifi{}},
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

func TestChassisModulesKeepMalformedDataAsFailure(t *testing.T) {
	oldSysRoot := sysRoot
	sysRoot = t.TempDir()
	t.Cleanup(func() { sysRoot = oldSysRoot })

	path := sysPath("class", "dmi", "id", "chassis_type")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, module := range []Module{chassis{}, chassistype{}} {
		_, err := module.Run(Context{})
		if err == nil {
			t.Fatalf("%s returned nil error for malformed chassis data", module.Name())
		}
		if errors.Is(err, errors.ErrUnsupported) {
			t.Fatalf("%s classified malformed chassis data as unsupported: %v", module.Name(), err)
		}
	}
}

func TestCPUCacheMalformedDataIsFailure(t *testing.T) {
	oldSysRoot := sysRoot
	sysRoot = t.TempDir()
	t.Cleanup(func() { sysRoot = oldSysRoot })

	cache := sysPath("devices", "system", "cpu", "cpu0", "cache", "index0")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{"level": "1\n", "size": "invalid\n"} {
		if err := os.WriteFile(filepath.Join(cache, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}

	_, err := (cpucache{}).Run(Context{})
	if err == nil {
		t.Fatal("Run() returned nil error for malformed cache data")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("malformed cache data was classified as unsupported: %v", err)
	}
}

func TestMediaCommandFailureRemainsFailure(t *testing.T) {
	pathDir := t.TempDir()
	commandPath := filepath.Join(pathDir, "playerctl")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", pathDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	_, err := (mediaModule{name: "media", key: "Media", format: "{{title}}"}).Run(Context{})
	if err == nil {
		t.Fatal("Run() returned nil error for failing playerctl")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("playerctl execution failure was classified as unsupported: %v", err)
	}
}

func TestGPUUsageCommandFailureRemainsFailure(t *testing.T) {
	oldSysRoot := sysRoot
	sysRoot = t.TempDir()
	t.Cleanup(func() { sysRoot = oldSysRoot })

	pathDir := t.TempDir()
	commandPath := filepath.Join(pathDir, "nvidia-smi")
	if err := os.WriteFile(commandPath, []byte("#!/bin/sh\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", pathDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", oldPath) })

	_, err := (gpuusage{}).Run(Context{})
	if err == nil {
		t.Fatal("Run() returned nil error for failing nvidia-smi")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("nvidia-smi execution failure was classified as unsupported: %v", err)
	}
}
