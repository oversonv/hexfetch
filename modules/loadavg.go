//go:build linux

package modules

import (
	"errors"
	"strconv"
	"strings"
	"syscall"
)

// siLoadShift is the fixed-point shift the kernel applies to sysinfo loads.
const siLoadShift = 16

type loadavg struct{}

func init() { Register(loadavg{}) }

func (loadavg) Name() string { return "loadavg" }

func (loadavg) Run(ctx Context) ([]Entry, error) {
	data, err := readFile(procPath("loadavg"))
	if err == nil {
		one, five, fifteen, parseErr := parseLoadAvg(data)
		if parseErr == nil {
			return Single(ctx, "Load Average", one+", "+five+", "+fifteen), nil
		}
		err = parseErr
	}
	if one, five, fifteen, ok := sysinfoLoadAvg(); ok {
		return Single(ctx, "Load Average", one+", "+five+", "+fifteen), nil
	}
	return nil, wrapUnavailableError(`read "/proc/loadavg"`, err)
}

func parseLoadAvg(data []byte) (string, string, string, error) {
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return "", "", "", errors.New("invalid /proc/loadavg")
	}
	return fields[0], fields[1], fields[2], nil
}

// sysinfoLoadAvg reads the load average through the sysinfo() syscall,
// which stays available where /proc/loadavg is restricted (containers, proot).
func sysinfoLoadAvg() (string, string, string, bool) {
	var info syscall.Sysinfo_t
	if syscall.Sysinfo(&info) != nil {
		return "", "", "", false
	}
	one, five, fifteen := formatSysinfoLoads(&info)
	return one, five, fifteen, true
}

func formatSysinfoLoads(info *syscall.Sysinfo_t) (string, string, string) {
	format := func(load uint64) string {
		return strconv.FormatFloat(float64(load)/(1<<siLoadShift), 'f', 2, 64)
	}
	return format(uint64(info.Loads[0])), format(uint64(info.Loads[1])), format(uint64(info.Loads[2]))
}
