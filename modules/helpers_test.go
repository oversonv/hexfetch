package modules

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KiB"},
		{1536, "1.50 KiB"},
		{1024 * 1024, "1.00 MiB"},
		{8 * 1024 * 1024 * 1024, "8.00 GiB"},
	}
	for _, test := range tests {
		if got := formatBytes(test.input); got != test.want {
			t.Errorf("formatBytes(%d) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestFormatFreq(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{3_400_000_000, "3.40 GHz"},
		{800_000_000, "800 MHz"},
		{1_500_000, "2 MHz"},
		{500, "500 Hz"},
	}
	for _, test := range tests {
		if got := formatFreq(test.input); got != test.want {
			t.Errorf("formatFreq(%d) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestFormatTemp(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{45000, "45 °C"},
		{45500, "45.5 °C"},
		{-5000, "-5 °C"},
	}
	for _, test := range tests {
		if got := formatTemp(test.input); got != test.want {
			t.Errorf("formatTemp(%d) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
		ok    bool
	}{
		{"32K", 32 * 1024, true},
		{"512K", 512 * 1024, true},
		{"8M", 8 * 1024 * 1024, true},
		{"1G", 1024 * 1024 * 1024, true},
		{"2048", 2048, true},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, test := range tests {
		got, ok := parseSize(test.input)
		if got != test.want || ok != test.ok {
			t.Errorf("parseSize(%q) = (%d, %v), want (%d, %v)", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestCleanValue(t *testing.T) {
	tests := map[string]string{
		`"Dell Inc."`:            "Dell Inc.",
		"To Be Filled By O.E.M.": "",
		"System Product Name":    "",
		"Unknown":                "",
		"  XPS 13  ":             "XPS 13",
	}
	for input, want := range tests {
		if got := cleanValue(input); got != want {
			t.Errorf("cleanValue(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestIsUsableShellPath(t *testing.T) {
	usablePath := filepath.Join(t.TempDir(), "zsh")
	if err := os.WriteFile(usablePath, []byte("shell"), 0o700); err != nil {
		t.Fatal(err)
	}
	tests := map[string]bool{
		usablePath:           true,
		"/bin/.l2s.zsh0000":  false,
		"zsh":                false,
		"/nonexistent/shell": false,
		"/bin/l2s.zsh":       false,
	}
	for input, want := range tests {
		if got := isUsableShellPath(input); got != want {
			t.Errorf("isUsableShellPath(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestPrettyShellName(t *testing.T) {
	tests := map[string]string{
		"pwsh": "PowerShell",
		"nu":   "nushell",
		"zsh":  "zsh",
	}
	for input, want := range tests {
		if got := prettyShellName(input); got != want {
			t.Errorf("prettyShellName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestContextOptions(t *testing.T) {
	ctx := Context{Options: map[string]any{
		"family": "4",
		"flag":   true,
		"list":   []any{"docker", "veth"},
	}}
	if got := ctx.OptionString("family", "6"); got != "4" {
		t.Errorf("OptionString = %q", got)
	}
	if got := ctx.OptionString("missing", "6"); got != "6" {
		t.Errorf("OptionString fallback = %q", got)
	}
	if !ctx.OptionBool("flag", false) {
		t.Error("OptionBool = false")
	}
	if got := ctx.OptionStrings("list"); len(got) != 2 || got[0] != "docker" {
		t.Errorf("OptionStrings = %v", got)
	}
}

func TestReadFileClassifiesMissingSysfsPath(t *testing.T) {
	old := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = old }()

	_, err := readFile(sysPath("class", "battery", "capacity"))
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("readFile() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readFile() error = %v, want os.ErrNotExist to remain reachable", err)
	}
}

func TestReadTrimmedAndReadTrimUseProcFixture(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	defer func() { procRoot = old }()

	if err := os.WriteFile(procPath("uptime"), []byte("123.45 678.90\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := readTrimmed(procPath("uptime"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "123.45 678.90" {
		t.Fatalf("readTrimmed() = %q", got)
	}
	got, err = ReadTrim(procPath("uptime"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "123.45 678.90" {
		t.Fatalf("ReadTrim() = %q", got)
	}
}

func TestReadFirstReturnsUnsupportedWhenAllPathsAreUnavailable(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	defer func() { procRoot = old }()

	_, err := readFirst(procPath("missing-a"), procPath("missing-b"))
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("readFirst() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readFirst() error = %v, want os.ErrNotExist to remain reachable", err)
	}
}

func TestReadFirstKeepsUnexpectedErrorsAsFailures(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	defer func() { procRoot = old }()

	file := procPath("not-a-directory")
	if err := os.WriteFile(file, []byte("value"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readFirst(filepath.Join(file, "child"), procPath("missing"))
	if err == nil {
		t.Fatal("readFirst() returned nil error")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("readFirst() error = %v, unexpected errors must remain failures", err)
	}
}

func TestReadFirstExistingClassifiesUnavailablePaths(t *testing.T) {
	old := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = old }()

	_, _, err := readFirstExisting(sysPath("missing-a"), sysPath("missing-b"))
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("readFirstExisting() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("readFirstExisting() error = %v, want os.ErrNotExist to remain reachable", err)
	}
}

func TestReadFirstExistingReturnsFirstReadableFixture(t *testing.T) {
	old := sysRoot
	sysRoot = t.TempDir()
	defer func() { sysRoot = old }()

	path := sysPath("class", "net", "wlan0", "operstate")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("up\n"), 0600); err != nil {
		t.Fatal(err)
	}
	data, gotPath, err := readFirstExisting(sysPath("missing"), path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "up\n" || gotPath != path {
		t.Fatalf("readFirstExisting() = (%q, %q), want (%q, %q)", data, gotPath, "up\n", path)
	}
}

func TestReadOSReleaseUsesEtcFixture(t *testing.T) {
	old := etcRoot
	etcRoot = filepath.Join(t.TempDir(), "etc")
	defer func() { etcRoot = old }()

	if err := os.MkdirAll(etcRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(etcPath("os-release"), []byte("NAME=Fixture\nPRETTY_NAME=\"Fixture Linux\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := ReadOSRelease()
	if err != nil {
		t.Fatal(err)
	}
	if release["NAME"] != "Fixture" || release["PRETTY_NAME"] != "Fixture Linux" {
		t.Fatalf("ReadOSRelease() = %v", release)
	}
}

func TestReadOSReleaseUsesUsrFallbackFixture(t *testing.T) {
	old := etcRoot
	root := t.TempDir()
	etcRoot = filepath.Join(root, "etc")
	defer func() { etcRoot = old }()

	fallback := filepath.Join(root, "usr", "lib", "os-release")
	if err := os.MkdirAll(filepath.Dir(fallback), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fallback, []byte("NAME=Fallback\n"), 0600); err != nil {
		t.Fatal(err)
	}
	release, err := ReadOSRelease()
	if err != nil {
		t.Fatal(err)
	}
	if release["NAME"] != "Fallback" {
		t.Fatalf("ReadOSRelease() = %v", release)
	}
}

func TestReadOSReleaseMissingIsUnsupported(t *testing.T) {
	old := etcRoot
	etcRoot = filepath.Join(t.TempDir(), "etc")
	defer func() { etcRoot = old }()

	_, err := ReadOSRelease()
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("ReadOSRelease() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadOSRelease() error = %v, want os.ErrNotExist to remain reachable", err)
	}
}

func TestRunCommandClassifiesMissingExecutable(t *testing.T) {
	_, err := runCommand(time.Second, "hexfetch-command-that-does-not-exist")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("runCommand() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("runCommand() error = %v, want exec.ErrNotFound to remain reachable", err)
	}
}

func TestRunCommandPathKeepsCommandFailures(t *testing.T) {
	_, err := runCommandPath(time.Second, "sh", "-c", "exit 7")
	if err == nil {
		t.Fatal("runCommandPath() returned nil error")
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("runCommandPath() error = %v, command exit failures must remain errors", err)
	}
}

func TestWrapUnavailableErrorPreservesPermissionCause(t *testing.T) {
	err := wrapUnavailableError("read fixture", os.ErrPermission)
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("wrapUnavailableError() error = %v, want errors.ErrUnsupported", err)
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("wrapUnavailableError() error = %v, want os.ErrPermission to remain reachable", err)
	}
}

func TestReadFirstTreatsEmptyFilesAsUnsupported(t *testing.T) {
	old := procRoot
	procRoot = t.TempDir()
	defer func() { procRoot = old }()

	path := procPath("empty")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := readFirst(path)
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("readFirst() error = %v, want errors.ErrUnsupported", err)
	}
}

func TestContextOptionStringsSupportsSliceTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  []string
	}{
		{name: "interface slice", value: []any{"docker", "veth"}, want: []string{"docker", "veth"}},
		{name: "string slice", value: []string{"docker", "veth"}, want: []string{"docker", "veth"}},
		{name: "scalar", value: "docker", want: []string{"docker"}},
		{name: "empty values", value: []string{"docker", ""}, want: []string{"docker"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := Context{Options: map[string]any{"exclude": test.value}}
			got := ctx.OptionStrings("exclude")
			if len(got) != len(test.want) {
				t.Fatalf("OptionStrings() = %v, want %v", got, test.want)
			}
			for index := range got {
				if got[index] != test.want[index] {
					t.Fatalf("OptionStrings() = %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestContextOptionInt(t *testing.T) {
	ctx := Context{Options: map[string]any{"integer": 500, "string": "250", "invalid": "nope"}}
	if got := ctx.OptionInt("integer", 150); got != 500 {
		t.Fatalf("integer option = %d, want 500", got)
	}
	if got := ctx.OptionInt("string", 150); got != 250 {
		t.Fatalf("string integer option = %d, want 250", got)
	}
	if got := ctx.OptionInt("invalid", 150); got != 150 {
		t.Fatalf("invalid integer option = %d, want fallback 150", got)
	}
}

type configKeyOverrideModule struct{}

func (configKeyOverrideModule) Name() string { return "config-key-override-test" }
func (configKeyOverrideModule) Run(Context) ([]Entry, error) {
	return []Entry{{Key: "First", Value: "1"}, {Key: "Second", Value: "2"}}, nil
}

func TestRunAllAppliesKeyOverrideToAllNormalEntries(t *testing.T) {
	Register(configKeyOverrideModule{})
	entries, diagnostics := RunAll([]Spec{{Name: "config-key-override-test", Context: Context{Key: "Custom"}}})
	if len(diagnostics) != 0 || len(entries) != 2 {
		t.Fatalf("RunAll() = %v, %v", entries, diagnostics)
	}
	for index, entry := range entries {
		if entry.Key != "Custom" {
			t.Errorf("entry %d key = %q, want Custom", index, entry.Key)
		}
	}
}
