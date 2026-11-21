//go:build !linux

package modules

func fallbackUptime() (int, bool) { return 0, false }
