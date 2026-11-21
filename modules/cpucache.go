//go:build linux

package modules

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
)

type cpucache struct{}

func init() { Register(cpucache{}) }

func (cpucache) Name() string { return "cpucache" }

func (cpucache) Run(ctx Context) ([]Entry, error) {
	levels, err := collectCPUCache()
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, level := range sortedLevels(levels) {
		entries = append(entries, Entry{
			Key:   "L" + strconv.Itoa(level) + " Cache",
			Value: formatBytes(levels[level]),
		})
	}
	return entries, nil
}

type cacheEntry struct {
	level int
	size  uint64
}

func collectCPUCache() (map[int]uint64, error) {
	paths, err := filepath.Glob(sysPath("devices/system/cpu/cpu0/cache/index*"))
	if err != nil {
		return nil, fmt.Errorf("list CPU cache entries: %w", err)
	}
	if len(paths) == 0 {
		return nil, Unsupported("CPU cache information unavailable")
	}
	var caches []cacheEntry
	var failures []error
	allUnavailable := true
	for _, path := range paths {
		levelText, err := readTrimmed(filepath.Join(path, "level"))
		if err != nil {
			if !errors.Is(err, errors.ErrUnsupported) {
				allUnavailable = false
				failures = append(failures, err)
			}
			continue
		}
		level, err := strconv.Atoi(levelText)
		if err != nil || level <= 0 {
			allUnavailable = false
			failures = append(failures, fmt.Errorf("invalid CPU cache level %q in %q", levelText, path))
			continue
		}
		sizeText, err := readTrimmed(filepath.Join(path, "size"))
		if err != nil {
			if !errors.Is(err, errors.ErrUnsupported) {
				allUnavailable = false
				failures = append(failures, err)
			}
			continue
		}
		size, ok := parseSize(sizeText)
		if !ok || size == 0 {
			allUnavailable = false
			failures = append(failures, fmt.Errorf("invalid CPU cache size %q in %q", sizeText, path))
			continue
		}
		caches = append(caches, cacheEntry{level: level, size: size})
	}
	if len(caches) == 0 {
		if allUnavailable {
			return nil, Unsupported("CPU cache information unavailable")
		}
		return nil, fmt.Errorf("CPU cache information unavailable: %w", errors.Join(failures...))
	}
	return aggregateCache(caches), nil
}

func aggregateCache(caches []cacheEntry) map[int]uint64 {
	levels := map[int]uint64{}
	for _, cache := range caches {
		levels[cache.level] += cache.size
	}
	return levels
}

func sortedLevels(levels map[int]uint64) []int {
	keys := make([]int, 0, len(levels))
	for level := range levels {
		keys = append(keys, level)
	}
	sort.Ints(keys)
	return keys
}
