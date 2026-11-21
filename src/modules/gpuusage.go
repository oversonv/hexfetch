//go:build linux

package modules

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type gpuusage struct{}

func init() { Register(gpuusage{}) }

func (gpuusage) Name() string { return "gpuusage" }

func (gpuusage) Run(ctx Context) ([]Entry, error) {
	var entries []Entry
	paths, _ := filepath.Glob(sysPath("class", "drm", "card[0-9]", "device", "gpu_busy_percent"))
	for _, path := range paths {
		value, _ := readTrimmed(path)
		if !validPercent(value) {
			continue
		}
		card := filepath.Base(filepath.Dir(filepath.Dir(path)))
		entries = append(entries, Entry{Key: "GPU Usage (" + card + ")", Value: value + "%"})
	}
	if len(entries) == 0 {
		values, err := gpuUsageNvidia()
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			entries = append(entries, Entry{Key: "GPU Usage (NVIDIA)", Value: value})
		}
	}
	if len(entries) == 0 {
		return nil, Unsupported("GPU usage unavailable")
	}
	return entries, nil
}

func gpuUsageNvidia() ([]string, error) {
	if !haveCommand("nvidia-smi") {
		return nil, nil
	}
	out, err := runCommandPath(3*time.Second, "nvidia-smi", "--query-gpu=utilization.gpu", "--format=csv,noheader")
	if err != nil {
		return nil, err
	}
	return parseNvidiaUtilization([]byte(out)), nil
}

func parseNvidiaUtilization(data []byte) []string {
	var percents []string
	for _, line := range strings.Split(string(data), "\n") {
		value := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "%"))
		if !validPercent(value) {
			continue
		}
		percents = append(percents, value+"%")
	}
	return percents
}

func validPercent(value string) bool {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	return err == nil && number >= 0 && number <= 100
}
