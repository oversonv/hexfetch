package modules

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type packages struct{}

func init() { Register(packages{}) }

func (packages) Name() string { return "packages" }

func (packages) Run(ctx Context) ([]Entry, error) {
	home, _ := os.UserHomeDir()
	counts := []struct {
		name  string
		count int
	}{
		{"pacman", countDirs("/var/lib/pacman/local")},
		{"dpkg", countPrefix("/var/lib/dpkg/status", "Status: install ok installed")},
		// Termux's pkg frontend stores its installed-package database in dpkg format.
		{"dpkg", countTermux()},
		{"rpm", countRPM()},
		{"apk", countPrefix("/lib/apk/db/installed", "P:")},
		{"xbps", countXbps()},
		{"nix", countNix(home)},
		{"flatpak", countDirs("/var/lib/flatpak/app") + countDirs(filepath.Join(home, ".local/share/flatpak/app"))},
		{"snap", countSnaps()},
		{"portage", countPortage("/var/db/pkg")},
		{"pkgtool", countFiles("/var/log/packages")},
	}
	var parts []string
	for _, item := range counts {
		if item.count > 0 {
			parts = append(parts, fmt.Sprintf("%d (%s)", item.count, item.name))
		}
	}
	if len(parts) == 0 {
		return nil, Unsupported("no package manager found")
	}
	return Single(ctx, "Packages", strings.Join(parts, ", ")), nil
}

func countDirs(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

func countPrefix(path, prefix string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	needle := []byte(prefix)
	count := 0
	for scanner.Scan() {
		if bytes.HasPrefix(scanner.Bytes(), needle) {
			count++
		}
	}
	return count
}

// countPortage counts Gentoo packages stored as <root>/<category>/<package>/.
func countPortage(root string) int {
	categories, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	total := 0
	for _, category := range categories {
		if category.IsDir() {
			total += countDirs(filepath.Join(root, category.Name()))
		}
	}
	return total
}

// countFiles counts regular files in a directory (Slackware package logs).
func countFiles(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

func termuxPrefix() string {
	if prefix := os.Getenv("PREFIX"); strings.Contains(prefix, "com.termux") {
		return prefix
	}
	if os.Getenv("TERMUX_VERSION") != "" {
		return "/data/data/com.termux/files/usr"
	}
	return ""
}

func countTermux() int {
	prefix := termuxPrefix()
	if prefix == "" {
		return 0
	}
	return countPrefix(filepath.Join(prefix, "var/lib/dpkg/status"), "Status: install ok installed")
}

func countRPM() int {
	if _, err := os.Stat("/var/lib/rpm"); err != nil {
		return 0
	}
	path, err := exec.LookPath("rpm")
	if err != nil {
		return 0
	}
	timeout, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(timeout, path, "-qa").Output()
	if err != nil {
		return 0
	}
	return bytes.Count(out, []byte("\n"))
}

func countXbps() int {
	files, _ := filepath.Glob("/var/db/xbps/pkgdb-*.plist")
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		return bytes.Count(data, []byte("<key>pkgver</key>"))
	}
	return 0
}

func countNix(home string) int {
	total := 0
	paths := []string{
		filepath.Join(home, ".nix-profile/manifest.json"),
		"/nix/var/nix/profiles/default/manifest.json",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var manifest struct {
			Elements json.RawMessage `json:"elements"`
		}
		if json.Unmarshal(data, &manifest) != nil {
			continue
		}
		var list []json.RawMessage
		if json.Unmarshal(manifest.Elements, &list) == nil {
			total += len(list)
			continue
		}
		var set map[string]json.RawMessage
		if json.Unmarshal(manifest.Elements, &set) == nil {
			total += len(set)
		}
	}
	return total
}

func countSnaps() int {
	files, _ := filepath.Glob("/var/lib/snapd/snaps/*.snap")
	return len(files)
}
