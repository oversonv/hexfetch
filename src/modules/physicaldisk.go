//go:build linux

package modules

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type physicaldisk struct{}

func init() { Register(physicaldisk{}) }

func (physicaldisk) Name() string { return "physicaldisk" }

func (physicaldisk) Run(ctx Context) ([]Entry, error) {
	entries, err := collectPhysicalDisks()
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, disk := range entries {
		result = append(result, Entry{Key: "Disk", Value: disk})
	}
	return result, nil
}

func collectPhysicalDisks() ([]string, error) {
	paths, _ := filepath.Glob(sysPath("block/*"))
	sort.Strings(paths)
	var disks []string
	for _, path := range paths {
		name := filepath.Base(path)
		if !physicalBlockName(name) {
			continue
		}
		model, _ := readTrimmed(filepath.Join(path, "device", "model"))
		if model == "" {
			model, _ = readTrimmed(filepath.Join(path, "device", "name"))
		}
		model = cleanValue(model)
		sizeText, err := readTrimmed(filepath.Join(path, "size"))
		if err != nil {
			continue
		}
		sectors, err := strconv.ParseUint(sizeText, 10, 64)
		if err != nil || sectors == 0 {
			continue
		}
		rotational, _ := readTrimmed(filepath.Join(path, "queue", "rotational"))
		disks = append(disks, describePhysicalDisk(name, model, sectors*512, rotational == "1"))
	}
	if len(disks) == 0 {
		return nil, Unsupported("physical disks unavailable")
	}
	return disks, nil
}

func physicalBlockName(name string) bool {
	for _, prefix := range []string{"loop", "ram", "zram", "dm-", "md", "sr", "fd"} {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

func describePhysicalDisk(name, model string, bytes uint64, rotating bool) string {
	if model == "" {
		model = name
	}
	kind := "SSD"
	if rotating {
		kind = "HDD"
	}
	return model + " (" + formatBytes(bytes) + ", " + kind + ")"
}
