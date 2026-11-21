package modules

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type shell struct{}

func init() { Register(shell{}) }

func (shell) Name() string { return "shell" }

func (shell) Run(ctx Context) ([]Entry, error) {
	info := detectShell()
	if info.name == "" {
		return nil, Unsupported("shell unknown")
	}
	value := prettyShellName(info.name)
	if version := shellVersion(info.path, info.name); version != "" {
		value += " " + version
	}
	return Single(ctx, "Shell", value), nil
}

var shellNames = map[string]bool{
	"ash": true, "bash": true, "csh": true, "dash": true, "elvish": true,
	"fish": true, "ion": true, "ksh": true, "mksh": true, "nu": true,
	"oksh": true, "pdksh": true, "pwsh": true, "sh": true, "tcsh": true,
	"xonsh": true, "yash": true, "zsh": true,
}

var shellPrettyNames = map[string]string{
	"pwsh": "PowerShell",
	"nu":   "nushell",
	"sh":   "sh",
}

type shellInfo struct {
	name string
	path string
}

func detectShell() shellInfo {
	for pid := os.Getppid(); pid > 1; pid = parentPID(pid) {
		info := inspectProcess(pid)
		for _, name := range info.names {
			if !shellNames[name] {
				continue
			}
			return shellInfo{name: name, path: shellPath(name, info)}
		}
	}
	if value := os.Getenv("SHELL"); value != "" {
		name := strings.TrimPrefix(filepath.Base(value), "-")
		if name != "" {
			return shellInfo{name: name, path: value}
		}
	}
	return shellInfo{}
}

type processData struct {
	names []string
	argv0 string
	exe   string
}

func inspectProcess(pid int) processData {
	base := procPath(strconv.Itoa(pid))
	var info processData
	if comm, err := readTrimmed(base + "/comm"); err == nil {
		info.names = append(info.names, comm)
	}
	if cmdline, err := readFile(base + "/cmdline"); err == nil {
		first := strings.SplitN(string(cmdline), "\x00", 2)[0]
		if first != "" {
			info.argv0 = first
			name := strings.TrimPrefix(filepath.Base(first), "-")
			if shellNames[name] {
				info.names = append(info.names, name)
			}
		}
	}
	if target, err := os.Readlink(base + "/exe"); err == nil {
		info.exe = strings.TrimSuffix(target, " (deleted)")
		// In proot /proc/<pid>/exe may point at an internal loader artifact.
		if name := filepath.Base(info.exe); shellNames[name] {
			info.names = append(info.names, name)
		}
	}
	return info
}

func shellPath(name string, info processData) string {
	if isUsableShellPath(info.exe) {
		return info.exe
	}
	if isUsableShellPath(info.argv0) {
		return info.argv0
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	if info.argv0 != "" {
		if path, err := exec.LookPath(filepath.Base(info.argv0)); err == nil {
			return path
		}
	}
	return name
}

// isUsableShellPath rejects proot/link2symlink artifacts such as /bin/.l2s..apk...
func isUsableShellPath(path string) bool {
	if path == "" {
		return false
	}
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".l2s") || strings.HasPrefix(base, "l2s") {
		return false
	}
	if !filepath.IsAbs(path) {
		return false
	}
	return exists(path)
}

func prettyShellName(name string) string {
	if pretty, ok := shellPrettyNames[name]; ok {
		return pretty
	}
	return name
}

func parentPID(pid int) int {
	data, err := readFile(procPath(strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0
	}
	text := string(data)
	index := strings.LastIndexByte(text, ')')
	if index < 0 {
		return 0
	}
	fields := strings.Fields(text[index+1:])
	if len(fields) < 2 {
		return 0
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return ppid
}

func shellVersion(path, name string) string {
	if path == "" {
		return ""
	}
	if !isUsableShellPath(path) {
		if resolved, err := exec.LookPath(name); err == nil {
			path = resolved
		}
	}
	if path == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	return regexp.MustCompile(`\d+(\.\d+)+`).FindString(string(out))
}
