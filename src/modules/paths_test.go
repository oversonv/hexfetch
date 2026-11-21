package modules

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// rootLiteralAllowed lists files that legitimately contain /proc or /sys
// literals; every other module must go through procPath and sysPath so tests
// can redirect them to fixture trees.
var rootLiteralAllowed = map[string]string{
	"helpers.go": "defines the roots",
	"disks.go":   "filters pseudo-filesystem mount points",
	"loadavg.go": "error text only",
}

func TestNoHardcodedProcOrSysPaths(t *testing.T) {
	pattern := regexp.MustCompile(`"/(proc|sys)(/[^"]*)?"`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if _, ok := rootLiteralAllowed[file]; ok {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for number, line := range strings.Split(string(data), "\n") {
			if pattern.MatchString(line) {
				t.Errorf("%s:%d: use procPath/sysPath instead of a literal: %s", file, number+1, strings.TrimSpace(line))
			}
		}
	}
}

func TestUserConfigFileHonorsXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	got, ok := userConfigFile("kdeglobals")
	if !ok || got != filepath.Join(root, "kdeglobals") {
		t.Fatalf("userConfigFile() = %q, %v, want %q", got, ok, filepath.Join(root, "kdeglobals"))
	}
}

func TestUserConfigFileFallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)
	got, ok := userConfigFile("a", "b")
	if !ok || got != filepath.Join(home, ".config", "a", "b") {
		t.Fatalf("userConfigFile() = %q, %v, want %q", got, ok, filepath.Join(home, ".config", "a", "b"))
	}
}

func TestCountPortageCountsPackagesInCategories(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"app-misc/foo-1.0", "app-misc/bar-2.1", "sys-libs/baz-3"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "stray-file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := countPortage(root); got != 3 {
		t.Fatalf("countPortage() = %d, want 3", got)
	}
	if got := countPortage(filepath.Join(root, "missing")); got != 0 {
		t.Fatalf("countPortage(missing) = %d, want 0", got)
	}
}

func TestCountFilesCountsOnlyRegularEntries(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"bash-5.2-x86_64-1", "glibc-2.39-x86_64-1"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := countFiles(root); got != 2 {
		t.Fatalf("countFiles() = %d, want 2", got)
	}
}

func TestWMAndUptimeReadFromProcRoot(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	defer func() { procRoot = old }()
	// wm.Run prefers these hints over scanning the process table.
	for _, variable := range []string{"HYPRLAND_INSTANCE_SIGNATURE", "SWAYSOCK", "I3SOCK", "NIRI_SOCKET"} {
		t.Setenv(variable, "")
	}

	if err := os.MkdirAll(procPath("42"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(procPath("42", "comm"), []byte("sway\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(procPath("uptime"), []byte("3600.50 100.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	seconds, err := readUptime()
	if err != nil || seconds != 3600 {
		t.Fatalf("readUptime() = %d, %v, want 3600", seconds, err)
	}
	entries, err := wm{}.Run(Context{})
	if err != nil || len(entries) != 1 || entries[0].Value != "sway" {
		t.Fatalf("wm.Run() = %+v, %v, want sway from the proc fixture", entries, err)
	}
}
