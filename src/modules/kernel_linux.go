//go:build linux

package modules

import (
	"syscall"
)

func kernelDisplay() string {
	release, err := ReadTrim(procPath("sys", "kernel", "osrelease"))
	if err != nil || release == "" {
		release = unameRelease()
	}
	if release == "" {
		return ""
	}
	return "Linux " + release
}

func unameRelease() string {
	var info syscall.Utsname
	if syscall.Uname(&info) != nil {
		return ""
	}
	raw := make([]byte, 0, len(info.Release))
	for _, c := range info.Release {
		if c == 0 {
			break
		}
		raw = append(raw, byte(c))
	}
	return string(raw)
}
