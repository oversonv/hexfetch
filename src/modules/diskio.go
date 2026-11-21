//go:build linux

package modules

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type diskio struct{}

func init() { Register(diskio{}) }

func (diskio) Name() string { return "diskio" }

func (diskio) Run(ctx Context) ([]Entry, error) {
	data, err := readFile(procPath("diskstats"))
	if err != nil {
		return nil, err
	}
	read, written, ok := parseDiskstats(data)
	if !ok {
		return nil, errors.New("disk statistics unavailable")
	}
	value := fmt.Sprintf("Read %s, Write %s", formatBytes(read), formatBytes(written))
	return Single(ctx, "Disk I/O", value), nil
}

// wholeDiskName filters out partitions and virtual block devices.
func wholeDiskName(name string) bool {
	if !physicalBlockName(name) {
		return false
	}
	for _, prefix := range []string{"sd", "vd", "hd", "xvd"} {
		if strings.HasPrefix(name, prefix) {
			rest := strings.TrimPrefix(name, prefix)
			for _, char := range rest {
				if char < 'a' || char > 'z' {
					return false
				}
			}
			return rest != ""
		}
	}
	if strings.HasPrefix(name, "nvme") || strings.HasPrefix(name, "mmcblk") {
		rest := strings.TrimPrefix(strings.TrimPrefix(name, "nvme"), "mmcblk")
		for _, char := range rest {
			if (char < '0' || char > '9') && char != 'n' {
				return false
			}
		}
		return rest != ""
	}
	return false
}

func parseDiskstats(data []byte) (uint64, uint64, bool) {
	var read, written uint64
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		if !wholeDiskName(fields[2]) {
			continue
		}
		sectorsRead, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			continue
		}
		sectorsWritten, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		read += sectorsRead * 512
		written += sectorsWritten * 512
		found = true
	}
	return read, written, found
}
