package modules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Root prefixes are variables so tests can point modules at fixture trees.
var (
	procRoot = "/proc"
	sysRoot  = "/sys"
	etcRoot  = "/etc"
)

const commandTimeout = 3 * time.Second

func procPath(elem ...string) string {
	return filepath.Join(append([]string{procRoot}, elem...)...)
}

func sysPath(elem ...string) string {
	return filepath.Join(append([]string{sysRoot}, elem...)...)
}

func etcPath(elem ...string) string {
	return filepath.Join(append([]string{etcRoot}, elem...)...)
}

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, wrapUnavailableError(fmt.Sprintf("read %q", path), err)
	}
	return data, nil
}

func readTrimmed(path string) (string, error) {
	data, err := readFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readFirst(paths ...string) (string, error) {
	var failures []error
	allUnavailable := len(paths) > 0
	for _, path := range paths {
		value, err := readTrimmed(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("%q: %w", path, errorWithoutUnsupported(err)))
			if !isUnavailableError(err) {
				allUnavailable = false
			}
			continue
		}
		if value != "" {
			return value, nil
		}
	}
	return "", combinePathErrors("no non-empty readable file", failures, allUnavailable)
}

func readFirstExisting(paths ...string) ([]byte, string, error) {
	var failures []error
	allUnavailable := len(paths) > 0
	for _, path := range paths {
		data, err := readFile(path)
		if err == nil {
			return data, path, nil
		}
		failures = append(failures, fmt.Errorf("%q: %w", path, errorWithoutUnsupported(err)))
		if !isUnavailableError(err) {
			allUnavailable = false
		}
	}
	return nil, "", combinePathErrors("no readable file", failures, allUnavailable)
}

func isUnavailableError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) || errors.Is(err, exec.ErrNotFound)
}

type operationError struct {
	operation   string
	cause       error
	unsupported bool
}

func (e *operationError) Error() string {
	if e.unsupported {
		return fmt.Sprintf("%v: %s: %v", errors.ErrUnsupported, e.operation, e.cause)
	}
	return fmt.Sprintf("%s: %v", e.operation, e.cause)
}

func (e *operationError) Unwrap() error {
	return e.cause
}

func (e *operationError) Is(target error) bool {
	return e.unsupported && target == errors.ErrUnsupported
}

func wrapUnavailableError(operation string, err error) error {
	return &operationError{
		operation:   operation,
		cause:       err,
		unsupported: isUnavailableError(err),
	}
}

func errorWithoutUnsupported(err error) error {
	var operation *operationError
	if errors.As(err, &operation) {
		return fmt.Errorf("%s: %w", operation.operation, operation.cause)
	}
	return err
}

func combinePathErrors(message string, failures []error, allUnavailable bool) error {
	joined := errors.Join(failures...)
	if allUnavailable {
		if joined == nil {
			return fmt.Errorf("%w: %s", errors.ErrUnsupported, message)
		}
		return fmt.Errorf("%w: %s: %w", errors.ErrUnsupported, message, joined)
	}
	if joined != nil {
		return fmt.Errorf("%s: %w", message, joined)
	}
	return errors.New(message)
}

func globFirst(pattern string) (string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", errors.New("no match for " + pattern)
	}
	return matches[0], nil
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// cleanValue normalizes DMI/sysfs strings and drops placeholder values.
func cleanValue(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"'`)
	switch strings.ToLower(value) {
	case "", "none", "unknown", "not specified", "not available",
		"to be filled by o.e.m.", "to be filled by o.e.m",
		"system manufacturer", "system product name", "system version",
		"default string", "oem", "chassis manufacture", "base board product name",
		"base board version", "base board manufacturer", "not applicable", "n/a":
		return ""
	}
	return value
}

func dmiValue(field string) string {
	value, err := readTrimmed(sysPath("class", "dmi", "id", field))
	if err != nil {
		return ""
	}
	return cleanValue(value)
}

func parseInt(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

func parseUint(value string) (uint64, error) {
	return strconv.ParseUint(strings.TrimSpace(value), 10, 64)
}

// formatBytes renders a byte count with binary units, fastfetch style.
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	return fmt.Sprintf("%.2f %s", value, units[unit])
}

func formatFreq(hz uint64) string {
	switch {
	case hz >= 1_000_000_000:
		return fmt.Sprintf("%.2f GHz", float64(hz)/1_000_000_000)
	case hz >= 1_000_000:
		return fmt.Sprintf("%.0f MHz", float64(hz)/1_000_000)
	case hz >= 1_000:
		return fmt.Sprintf("%.0f kHz", float64(hz)/1_000)
	default:
		return fmt.Sprintf("%d Hz", hz)
	}
}

func formatTemp(milliCelsius int64) string {
	celsius := float64(milliCelsius) / 1000
	if celsius == float64(int64(celsius)) {
		return fmt.Sprintf("%.0f °C", celsius)
	}
	return fmt.Sprintf("%.1f °C", celsius)
}

func formatPercent(used, total uint64) string {
	if total == 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", float64(used)/float64(total)*100)
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := seconds / 3600
	minutes := seconds % 3600 / 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d", hours, minutes)
	}
	return fmt.Sprintf("%d min", minutes)
}

// parseSize handles sysfs cache sizes such as "32K", "512K" and "8M".
func parseSize(value string) (uint64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	multiplier := uint64(1)
	switch value[len(value)-1] {
	case 'K', 'k':
		multiplier = 1024
		value = value[:len(value)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		value = value[:len(value)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		value = value[:len(value)-1]
	}
	number, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, false
	}
	return number * multiplier, true
}

func runCommand(timeout time.Duration, name string, args ...string) (string, error) {
	if timeout <= 0 {
		timeout = commandTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return "", wrapUnavailableError(fmt.Sprintf("run command %q", name), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runCommandPath(timeout time.Duration, name string, args ...string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", wrapUnavailableError(fmt.Sprintf("find command %q", name), err)
	}
	return runCommand(timeout, path, args...)
}

func haveCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Cached common files. Tests use the pure parsers directly instead.
var (
	meminfoOnce sync.Once
	meminfoData []byte
	meminfoErr  error

	cpuinfoOnce sync.Once
	cpuinfoData []byte
	cpuinfoErr  error

	osReleaseOnce sync.Once
	osReleaseVal  map[string]string
	osReleaseErr  error
)

func cachedMeminfo() ([]byte, error) {
	meminfoOnce.Do(func() {
		meminfoData, meminfoErr = readFile(procPath("meminfo"))
	})
	return meminfoData, meminfoErr
}

func cachedCPUInfo() ([]byte, error) {
	cpuinfoOnce.Do(func() {
		cpuinfoData, cpuinfoErr = readFile(procPath("cpuinfo"))
	})
	return cpuinfoData, cpuinfoErr
}

func cachedOSRelease() (map[string]string, error) {
	osReleaseOnce.Do(func() {
		osReleaseVal, osReleaseErr = ReadOSRelease()
	})
	return osReleaseVal, osReleaseErr
}

// envFirst returns the first non-empty environment variable among names.
// The HEXFETCH_* name is listed first; NIXFETCH_* is kept as a deprecated fallback.
func envFirst(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}
