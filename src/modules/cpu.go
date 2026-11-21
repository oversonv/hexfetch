//go:build linux

package modules

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type cpu struct{}

func init() { Register(cpu{}) }

func (cpu) Name() string { return "cpu" }

func (cpu) Run(ctx Context) ([]Entry, error) {
	data, err := cachedCPUInfo()
	if err != nil {
		return nil, err
	}
	info := parseCPUInfo(data)
	if info.model == "" {
		return nil, Unsupported("CPU model unavailable")
	}
	if info.freqHz == 0 {
		info.freqHz = maxCPUFreq()
	}

	entries := []Entry{{Key: ctx.KeyOr("CPU"), Value: info.model}}
	if info.physical > 0 && info.logical > 0 {
		entries = append(entries, Entry{
			Key:   "Cores",
			Value: fmt.Sprintf("%d cores / %d threads", info.physical, info.logical),
		})
	} else if info.logical > 0 {
		entries = append(entries, Entry{
			Key:   "Threads",
			Value: strconv.Itoa(info.logical),
		})
	}
	if info.freqHz > 0 {
		entries = append(entries, Entry{Key: "CPU Frequency", Value: formatFreq(info.freqHz)})
	}
	return entries, nil
}

type cpuInfo struct {
	model    string
	physical int
	logical  int
	freqHz   uint64
}

func parseCPUInfo(data []byte) cpuInfo {
	var info cpuInfo
	sockets := map[string]bool{}
	corePairs := map[string]bool{}
	socketCores := 0
	currentPhysical := "0"
	var maxMHz float64
	var modelName, hardware, processorLine string

	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "processor":
			info.logical++
			currentPhysical = "0"
		case "model name":
			if modelName == "" {
				modelName = value
			}
		case "Hardware":
			if hardware == "" {
				hardware = value
			}
		case "Processor":
			if processorLine == "" {
				processorLine = value
			}
		case "cpu model", "CPU model":
			if processorLine == "" {
				processorLine = value
			}
		case "physical id":
			currentPhysical = value
			sockets[value] = true
		case "core id":
			corePairs[currentPhysical+"/"+value] = true
		case "cpu cores":
			if number, err := strconv.Atoi(value); err == nil && number > socketCores {
				socketCores = number
			}
		case "cpu MHz":
			if number, err := strconv.ParseFloat(value, 64); err == nil && number > maxMHz {
				maxMHz = number
			}
		}
	}

	info.model = cleanCPUModel(firstCPUModel(modelName, hardware, processorLine))
	info.physical = len(corePairs)
	if info.physical == 0 && socketCores > 0 {
		info.physical = max(len(sockets), 1) * socketCores
	}
	if maxMHz > 0 {
		info.freqHz = uint64(maxMHz * 1_000_000)
	}
	return info
}

func firstCPUModel(modelName, hardware, processorLine string) string {
	for _, candidate := range []string{modelName, hardware, processorLine} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func cleanCPUModel(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "(R)", "")
	value = strings.ReplaceAll(value, "(TM)", "")
	value = strings.ReplaceAll(value, "(tm)", "")
	return strings.Join(strings.Fields(value), " ")
}

// maxCPUFreq returns the highest advertised frequency across every CPU cluster,
// mirroring fastfetch's policy scan instead of trusting core 0 only.
func maxCPUFreq() uint64 {
	candidates := []string{
		sysPath("devices/system/cpu/cpufreq"),
	}
	for _, dir := range candidates {
		entries, err := filepath.Glob(filepath.Join(dir, "policy*", "*_max_freq"))
		if err != nil {
			continue
		}
		var best uint64
		for _, file := range entries {
			value, err := readTrimmed(file)
			if err != nil || value == "" {
				continue
			}
			number, err := parseUint(value)
			if err != nil || number == 0 {
				continue
			}
			// cpufreq files use kHz; take the raw maximum across policies.
			if number > best {
				best = number
			}
		}
		if best > 0 {
			if best < 10_000_000 {
				best *= 1000
			}
			return best
		}
	}
	// Fallback: per-CPU directories when the cpufreq/policy* layout is absent.
	perCPU, err := filepath.Glob(sysPath("devices/system/cpu/cpu*", "cpufreq", "cpuinfo_max_freq"))
	if err != nil {
		return 0
	}
	var best uint64
	for _, file := range perCPU {
		value, err := readTrimmed(file)
		if err != nil || value == "" {
			continue
		}
		number, err := parseUint(value)
		if err != nil || number == 0 {
			continue
		}
		if number > best {
			best = number
		}
	}
	if best == 0 {
		return 0
	}
	if best < 10_000_000 {
		best *= 1000
	}
	return best
}
