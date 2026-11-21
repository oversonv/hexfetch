//go:build linux

package modules

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type wifi struct{}

func init() { Register(wifi{}) }

func (wifi) Name() string { return "wifi" }

func (wifi) Run(ctx Context) ([]Entry, error) {
	interfaces := wirelessInterfaces()
	if len(interfaces) == 0 {
		return nil, Unsupported("no Wi-Fi interface found")
	}
	signals := map[string]int{}
	if data, err := readFile(procPath("net", "wireless")); err == nil {
		signals = parseProcWireless(data)
	}
	var entries []Entry
	for _, name := range interfaces {
		value := name
		if ssid := wifiSSID(name); ssid != "" {
			value += " (" + ssid
			if signal, ok := signals[name]; ok {
				value += ", " + strconv.Itoa(signal) + "%"
			}
			value += ")"
		} else if signal, ok := signals[name]; ok {
			value += " (" + strconv.Itoa(signal) + "%)"
		} else {
			state, _ := readTrimmed(sysPath("class", "net", name, "operstate"))
			if state != "" {
				value += " (" + state + ")"
			}
		}
		entries = append(entries, Entry{Key: "Wi-Fi", Value: value})
	}
	return entries, nil
}

func wirelessInterfaces() []string {
	paths, _ := filepath.Glob(sysPath("class", "net", "*"))
	var names []string
	for _, path := range paths {
		name := filepath.Base(path)
		if isDir(filepath.Join(path, "wireless")) || isDir(filepath.Join(path, "phy80211")) {
			names = append(names, name)
		}
	}
	return names
}

func parseProcWireless(data []byte) map[string]int {
	signals := map[string]int{}
	for _, line := range strings.Split(string(data), "\n") {
		name, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" || name == "Inter-|" || strings.Contains(name, "|") {
			continue
		}
		fields := strings.Fields(rest)
		if len(fields) < 2 {
			continue
		}
		quality := strings.TrimSuffix(fields[1], ".")
		number, err := strconv.ParseFloat(quality, 64)
		if err != nil {
			continue
		}
		signals[name] = int(number + 0.5)
	}
	return signals
}

func wifiSSID(name string) string {
	if haveCommand("iwgetid") {
		if out, err := runCommandPath(2*time.Second, "iwgetid", "-r", name); err == nil {
			out = strings.TrimSpace(out)
			if out != "" {
				return out
			}
		}
	}
	if haveCommand("iw") {
		if out, err := runCommandPath(2*time.Second, "iw", "dev", name, "link"); err == nil {
			for _, line := range strings.Split(out, "\n") {
				key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
				if ok && key == "SSID" {
					return strings.TrimSpace(value)
				}
			}
		}
	}
	return ""
}
