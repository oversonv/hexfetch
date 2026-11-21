package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"hexfetch/internal/cli"
	"hexfetch/internal/config"
	"hexfetch/internal/diag"
	"hexfetch/internal/gradient"
	"hexfetch/internal/logo"
	"hexfetch/internal/render"
	"hexfetch/modules"
)

type formatWarningModule struct{}

func (formatWarningModule) Name() string { return "format-warning-test" }
func (formatWarningModule) Run(modules.Context) ([]modules.Entry, error) {
	return []modules.Entry{{Key: "Key", Value: "Original"}}, nil
}

type failingDiagnosticModule struct{}

func (failingDiagnosticModule) Name() string { return "failing-diagnostic-test" }
func (failingDiagnosticModule) Run(modules.Context) ([]modules.Entry, error) {
	return nil, errors.New("failure reason")
}

func captureOutput(t *testing.T, target **os.File, run func()) string {
	t.Helper()
	original := *target
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	*target = writer
	defer func() {
		*target = original
	}()
	run()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	*target = original
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func captureStderr(t *testing.T, run func()) string {
	return captureOutput(t, &os.Stderr, run)
}

func captureStdout(t *testing.T, run func()) string {
	return captureOutput(t, &os.Stdout, run)
}

func TestRunPrintsModuleErrorsByDefault(t *testing.T) {
	modules.Register(failingDiagnosticModule{})
	output := captureStderr(t, func() {
		if code := run([]string{"--no-color", "--structure", "failing-diagnostic-test"}); code != 0 {
			t.Errorf("run() exit code = %d, want 0", code)
		}
	})
	if !strings.Contains(output, "Error: The failing-diagnostic-test module crashed.\n") {
		t.Fatalf("stderr = %q, want default Error diagnostic", output)
	}
	if strings.Contains(output, "failure reason") {
		t.Fatalf("stderr exposed error detail without --debug: %q", output)
	}
}

func TestRunQuietHidesDiagnosticsEvenWithDebug(t *testing.T) {
	modules.Register(failingDiagnosticModule{})
	output := captureStderr(t, func() {
		if code := run([]string{"--quiet", "--debug", "--no-color", "--structure", "failing-diagnostic-test:unknown-module"}); code != 0 {
			t.Errorf("run() exit code = %d, want 0", code)
		}
	})
	if output != "" {
		t.Fatalf("stderr = %q, want no diagnostics", output)
	}
}

func TestRunNoColorRendersSingleEmbeddedLogo(t *testing.T) {
	output := captureStdout(t, func() {
		if code := run([]string{"--no-color", "--quiet", "--structure", "title"}); code != 0 {
			t.Errorf("run() exit code = %d, want 0", code)
		}
	})
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("--no-color output contains ANSI sequences: %q", output)
	}
	if !strings.Contains(output, "▄") {
		t.Fatalf("output does not contain the embedded logo: %q", output)
	}
}

func TestRunGenConfigUsesHexHexfetchYAMLPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if code := run([]string{"--gen-config"}); code != 0 {
		t.Fatalf("run(--gen-config) = %d, want 0", code)
	}
	path := home + "/hex/set/hexfetch.yaml"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("generated config is missing: %v", err)
	}
	if !strings.Contains(string(data), "\nhexfetch:\n") {
		t.Fatalf("generated config does not contain YAML namespace: %s", data)
	}
	if strings.Contains(string(data), "\n  logo:\n") {
		t.Fatalf("generated config contains a logo section: %s", data)
	}
}

func TestRunGenConfigDoesNotOverwriteExistingFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := home + "/hex/set/hexfetch.yaml"
	if err := os.MkdirAll(home+"/hex/set", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--gen-config"}); code == 0 {
		t.Fatal("run(--gen-config) succeeded despite existing file")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "existing" {
		t.Fatalf("existing config changed: %q, %v", data, err)
	}
}

func TestDefaultConfigurationRendersTitleAndSeparator(t *testing.T) {
	cfg := config.Default()
	var diagnostics diag.Collector
	specs := buildSpecs(cfg, nil, false, false, &diagnostics)
	if len(specs) < 2 || specs[0].Name != "title" || specs[1].Name != "separator" {
		t.Fatalf("default specs begin with %+v, want title and separator", specs)
	}
	entries, moduleDiagnostics := modules.RunAll(specs[:2])
	if len(moduleDiagnostics) != 0 {
		t.Fatalf("default title and separator modules failed: %+v", moduleDiagnostics)
	}
	var output bytes.Buffer
	if err := render.Render(&output, entries, render.Options{SeparatorColor: string(cfg.Display.Color.Separator)}); err != nil {
		t.Fatal(err)
	}
	if output.String() != "Welcome!\n--------\n" {
		t.Fatalf("default title and separator output = %q", output.String())
	}
}

func TestBuildSpecsStructureRetainsYAMLSettings(t *testing.T) {
	cfg := config.Default()
	cfg.Modules.Settings["title"] = config.ModuleSpec{Type: "title", Format: "Configured title"}
	cfg.Modules.Settings["command"] = config.ModuleSpec{Type: "command", Key: "Python", Text: "python --version"}
	structure := "command:title"
	var diagnostics diag.Collector
	specs := buildSpecs(cfg, &structure, false, false, &diagnostics)
	if len(specs) != 2 {
		t.Fatalf("buildSpecs() returned %d specs: %+v", len(specs), specs)
	}
	if specs[0].Name != "command" || specs[0].Context.Key != "Python" || specs[0].Context.Command != "python --version" {
		t.Fatalf("command settings were lost by --structure: %+v", specs[0])
	}
	if specs[1].Name != "title" || specs[1].Context.Text != "Configured title" {
		t.Fatalf("title settings were lost by --structure: %+v", specs[1])
	}
	if len(diagnostics.All()) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics.All())
	}
}

func TestApplyOverridesLogoColors(t *testing.T) {
	cfg := config.Default()
	original := cfg.LogoColors
	options := cli.Options{LogoColors: []string{
		"38;2;1;2;3", "38;2;4;5;6", "38;2;7;8;9", "38;2;10;11;12",
		"38;2;13;14;15", "38;2;16;17;18", "38;2;19;20;21", "38;2;22;23;24",
		"38;2;25;26;27", "38;2;28;29;30", "38;2;31;32;33", "38;2;34;35;36",
	}}
	var diagnostics diag.Collector
	applyOverrides(&cfg, options, &diagnostics)
	if cfg.LogoColors[0] == original[0] || cfg.LogoColors[11] == original[11] {
		t.Fatalf("CLI colors were not applied: %+v", cfg.LogoColors)
	}
	if len(diagnostics.All()) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", diagnostics.All())
	}
}

func TestRunMigrateConfigPreservesLegacyTOML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := home + "/nixos/nixos.toml"
	if err := os.MkdirAll(home+"/nixos", 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := "[hexfetch.display]\nseparator = \" :: \"\n\n[hexfetch.modules]\nitems = [\"title\", \"separator\"]\n\n[hexfetch.modules.title]\nformat = \"Migrated\"\n"
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--migrate-config", "--quiet"}); code != 0 {
		t.Fatalf("run(--migrate-config) = %d", code)
	}
	destination := home + "/hex/set/hexfetch.yaml"
	loaded, diagnostics := config.Load(destination)
	if loaded.Display.Separator != " :: " || loaded.Modules.Settings["title"].Format != "Migrated" {
		t.Fatalf("migrated configuration = %+v, %+v", loaded.Display, loaded.Modules.Settings)
	}
	if len(diagnostics) != 1 || diagnostics[0].Level != diag.Info {
		t.Fatalf("migrated YAML diagnostics = %+v", diagnostics)
	}
	data, err := os.ReadFile(source)
	if err != nil || string(data) != legacy {
		t.Fatalf("legacy TOML was modified: %v", err)
	}
}

func TestNoColorEnvironmentIsConsistent(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !noColorRequested(false) {
		t.Fatal("non-empty NO_COLOR did not disable color")
	}
	t.Setenv("NO_COLOR", "")
	if noColorRequested(false) {
		t.Fatal("empty NO_COLOR should not disable color")
	}
	if !noColorRequested(true) {
		t.Fatal("--no-color did not disable color")
	}
}

func TestRunLoadsYAMLAndRendersConfiguredTitleSeparatorWithStructure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := home + "/hex/set/hexfetch.yaml"
	if err := os.MkdirAll(home+"/hex/set", 0o755); err != nil {
		t.Fatal(err)
	}
	document := "hexfetch:\n  modules:\n    items:\n      - title\n      - separator\n    title:\n      format: \"Config title\"\n"
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	output := captureStdout(t, func() {
		if code := run([]string{"--no-color", "--quiet", "--structure", "title:separator"}); code != 0 {
			t.Errorf("run() exit code = %d", code)
		}
	})
	if !strings.Contains(output, "Config title") || !strings.Contains(output, strings.Repeat("-", len("Config title"))) {
		t.Fatalf("configured title and separator were not rendered: %q", output)
	}
	if strings.Contains(output, "\x1b[") {
		t.Fatalf("--no-color output contains ANSI sequences: %q", output)
	}
}

func TestRunFormatWarningsUseDiagnosticsAndRespectQuiet(t *testing.T) {
	modules.Register(formatWarningModule{})
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := home + "/hex/set/hexfetch.yaml"
	if err := os.MkdirAll(home+"/hex/set", 0o755); err != nil {
		t.Fatal(err)
	}
	document := "hexfetch:\n  modules:\n    items:\n      - format-warning-test\n    format-warning-test:\n      format: \"{unknown}\"\n"
	if err := os.WriteFile(path, []byte(document), 0o644); err != nil {
		t.Fatal(err)
	}
	stderr := captureStderr(t, func() {
		captureStdout(t, func() {
			if code := run([]string{"--no-color"}); code != 0 {
				t.Errorf("run() exit code = %d", code)
			}
		})
	})
	if !strings.Contains(stderr, "Invalid format for module \"format-warning-test\"") {
		t.Fatalf("format warning did not use the diagnostics collector: %q", stderr)
	}
	quietStderr := captureStderr(t, func() {
		captureStdout(t, func() {
			if code := run([]string{"--no-color", "--quiet"}); code != 0 {
				t.Errorf("quiet run() exit code = %d", code)
			}
		})
	})
	if quietStderr != "" {
		t.Fatalf("--quiet did not suppress format diagnostics: %q", quietStderr)
	}
}

func logoText(l logo.Logo) string { return strings.Join(l.Lines, "\n") }

func TestLoadLogoDefaultsToHexos(t *testing.T) {
	var diagnostics diag.Collector
	got := loadLogo(config.Default(), false, &diagnostics)
	want := logo.Render(logo.DefaultColors(), false, gradient.TrueColor())
	if logoText(got) != logoText(want) || len(diagnostics.All()) != 0 {
		t.Fatalf("default logo is not hexos; diagnostics: %+v", diagnostics.All())
	}
}

func TestLoadLogoResolvesNamesAndAliases(t *testing.T) {
	hexos := logoText(loadLogo(config.Default(), false, &diag.Collector{}))
	cfg := config.Default()
	cfg.Logo = "arch"
	if arch := logoText(loadLogo(cfg, false, &diag.Collector{})); arch == hexos || arch == "" {
		t.Fatal("--logo arch did not select a different logo")
	}
	cfg.Logo = "HEX"
	if got := logoText(loadLogo(cfg, false, &diag.Collector{})); got != hexos {
		t.Fatal("alias hex did not resolve to hexos")
	}
}

func TestLoadLogoUnknownNameFallsBackToHexosWithWarning(t *testing.T) {
	var diagnostics diag.Collector
	cfg := config.Default()
	cfg.Logo = "definitely-not-a-distro"
	got := loadLogo(cfg, false, &diagnostics)
	if logoText(got) != logoText(logo.Render(logo.DefaultColors(), false, gradient.TrueColor())) {
		t.Fatal("unknown logo did not fall back to hexos")
	}
	warned := false
	for _, d := range diagnostics.All() {
		warned = warned || d.Level == diag.Warning
	}
	if !warned {
		t.Fatalf("no warning for an unknown logo: %+v", diagnostics.All())
	}
}

func TestLoadLogoNoneAndAuto(t *testing.T) {
	cfg := config.Default()
	cfg.Logo = "none"
	if got := loadLogo(cfg, false, &diag.Collector{}); len(got.Lines) != 0 {
		t.Fatalf("none rendered a logo: %q", logoText(got))
	}
	cfg.Logo = "auto"
	if got := loadLogo(cfg, false, &diag.Collector{}); len(got.Lines) == 0 {
		t.Fatal("auto rendered no logo")
	}
}

func TestLoadLogoCustomColorsAffectOnlyHexos(t *testing.T) {
	cfg := config.Default()
	for index := range cfg.LogoColors {
		cfg.LogoColors[index] = gradient.RGB{R: 9, G: 8, B: 7}
	}
	hexos := logoText(loadLogo(cfg, true, &diag.Collector{}))
	if !strings.Contains(hexos, "9;8;7") && !strings.Contains(hexos, "38;5;") {
		t.Fatalf("hexos ignored custom colors: %q", hexos)
	}
	cfg.Logo = "arch"
	custom := logoText(loadLogo(cfg, true, &diag.Collector{}))
	plain := config.Default()
	plain.Logo = "arch"
	if custom != logoText(loadLogo(plain, true, &diag.Collector{})) {
		t.Fatal("logo_colors changed a classic logo")
	}
}

func TestApplyOverridesLogoName(t *testing.T) {
	cfg := config.Default()
	name := "arch"
	applyOverrides(&cfg, cli.Options{Logo: &name}, &diag.Collector{})
	if cfg.Logo != "arch" {
		t.Fatalf("cfg.Logo = %q, want arch", cfg.Logo)
	}
}

func TestRunListLogos(t *testing.T) {
	output := captureStdout(t, func() {
		if code := run([]string{"--list-logos"}); code != 0 {
			t.Errorf("run(--list-logos) = %d, want 0", code)
		}
	})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 500 || !strings.Contains("\n"+output, "\nhexos\n") {
		t.Fatalf("--list-logos printed %d names, hexos present: %v", len(lines), strings.Contains(output, "hexos"))
	}
}

func TestRunLogoNoneHidesLogo(t *testing.T) {
	configPath := t.TempDir() + "/hexfetch.yaml"
	if err := os.WriteFile(configPath, []byte("hexfetch:\n  logo: none\n"), 0600); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() {
		if code := run([]string{"--no-color", "--quiet", "--config", configPath, "--structure", "title"}); code != 0 {
			t.Errorf("run() exit code = %d, want 0", code)
		}
	})
	if strings.Contains(output, "▄") || strings.Contains(output, "▟") {
		t.Fatalf("logo none still printed a logo: %q", output)
	}
}

func TestRunLicenses(t *testing.T) {
	output := captureStdout(t, func() {
		if code := run([]string{"--licenses"}); code != 0 {
			t.Errorf("run(--licenses) = %d, want 0", code)
		}
	})
	if !strings.Contains(output, "MIT License") || !strings.Contains(output, "fastfetch") {
		t.Fatalf("--licenses output is incomplete: %.200q", output)
	}
}
