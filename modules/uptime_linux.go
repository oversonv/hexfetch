//go:build linux

package modules

import (
	"syscall"
	"unsafe"
)

func fallbackUptime() (int, bool) {
	var ts syscall.Timespec
	_, _, errno := syscall.Syscall(syscall.SYS_CLOCK_GETTIME, 7, uintptr(unsafe.Pointer(&ts)), 0)
	if errno == 0 && ts.Sec > 0 {
		return int(ts.Sec), true
	}
	var info syscall.Sysinfo_t
	if syscall.Sysinfo(&info) == nil && info.Uptime > 0 {
		return int(info.Uptime), true
	}
	return 0, false
}
