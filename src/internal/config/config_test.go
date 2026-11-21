package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"hexfetch/src/internal/diag"
)

func writeYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func hasWarning(diagnostics []diag.Diagnostic, text string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == diag.Warning && strings.Contains(diagnostic.Message, text) {
			return true
		}
	}
	return false
}

func TestDefaultEnablesTitleBeforeSeparator(t *testing.T) {
	cfg := Default()
	if len(cfg.Modules.Items) < 2 || cfg.Modules.Items[0] != "title" || cfg.Modules.Items[1] != "separator" {
		t.Fatalf("default module order = %q, want title then separator", cfg.Modules.Items)
	}
	if !strings.Contains(DefaultYAML, "      - title\n      - separator") {
		t.Fatal("DefaultYAML does not enable title before separator")
	}
	if !strings.Contains(DefaultYAML, "    separator_char: \"-\"") {
		t.Fatal("DefaultYAML does not expose the separator character default")
	}
	if strings.Contains(DefaultYAML, "    command:") || len(cfg.Modules.Settings) != 0 {
		t.Fatal("default config contains a dormant command module")
	}
}

func TestDefaultPathUsesHexDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	want := filepath.Join(home, "hex", "set", "hexfetch.yaml")
	if got := DefaultPath(); got != want {
		t.Fatalf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestLoadYAMLDisplayModulesAndColors(t *testing.T) {
	document := `hexfetch:
  display:
    separator: " | "
    separator_char: "="
    separator_length: 18
    color:
      keys: blue
  modules:
    items:
      - title
      - command
      - localip
    title:
      format: "Host details"
    command:
      key: "Python"
      text: "python --version"
    localip:
      options:
        family: "4"
        exclude:
          - docker
          - veth
`
	cfg, diagnostics := Load(writeYAML(t, document))
	if cfg.Display.Separator != " | " || cfg.Display.SeparatorChar != "=" || cfg.Display.SeparatorLength != 18 || cfg.Display.Color.Keys != "34" {
		t.Fatalf("display config = %+v", cfg.Display)
	}
	specs, specDiagnostics := cfg.Specs()
	if len(specDiagnostics) != 0 || len(specs) != 3 {
		t.Fatalf("Specs() = %+v, %v", specs, specDiagnostics)
	}
	if specs[0].Format != "Host details" || specs[1].Key != "Python" || specs[1].Text != "python --version" {
		t.Fatalf("module settings = %+v", specs)
	}
	options := specs[2].Options
	if options["family"] != "4" {
		t.Fatalf("localip options = %#v", options)
	}
	exclude, ok := options["exclude"].([]any)
	if !ok || len(exclude) != 2 || exclude[0] != "docker" || exclude[1] != "veth" {
		t.Fatalf("localip exclude options = %#v", options["exclude"])
	}
	if hasWarning(diagnostics, "Invalid config field") || hasWarning(diagnostics, "Unknown config field") {
		t.Fatalf("unexpected config warning: %+v", diagnostics)
	}
}

func TestTermuxFastfetchProfileLoadsWithoutWarnings(t *testing.T) {
	profilePath := writeYAML(t, `hexfetch:
  display:
    separator: " » "
    separator_char: "─"
    separator_length: 18
    key:
      type: both
    color:
      keys: "38;2;255;255;255"
      title: "38;2;255;255;255"
      output: "38;2;255;255;255"
      separator: "38;2;190;190;190"
  modules:
    items:
      - title
      - separator
      - os
      - packages
      - shell
      - command
      - terminal
    title:
      text: "You again?"
    os:
      key: "OS"
      key_icon: "\uf313"
    packages:
      key: "Packages"
      key_icon: "\uf487"
    shell:
      key: "Shell"
      key_icon: "\uf489"
    command:
      key: "Python"
      key_icon: "\ue73c"
      text: "python -V 2>&1 | cut -d' ' -f2"
    terminal:
      key: "Terminal"
      key_icon: "\uf489"
`)
	cfg, diagnostics := Load(profilePath)
	for _, diagnostic := range diagnostics {
		if diagnostic.Level >= diag.Warning {
			t.Fatalf(
				"Termux profile produced a warning or error: %+v",
				diagnostic,
			)
		}
	}
	if cfg.Display.Separator != " » " || cfg.Display.SeparatorChar != "─" || cfg.Display.SeparatorLength != 18 {
		t.Fatalf("Termux display config = %+v", cfg.Display)
	}
	if cfg.Display.KeyType != "both" {
		t.Fatalf("key type = %q, want both", cfg.Display.KeyType)
	}
	wantColors := ColorConfig{
		Keys:      "38;2;255;255;255",
		Title:     "38;2;255;255;255",
		Output:    "38;2;255;255;255",
		Separator: "38;2;190;190;190",
	}
	if cfg.Display.Color != wantColors {
		t.Fatalf("display colors = %+v, want %+v", cfg.Display.Color, wantColors)
	}
	wantItems := []string{"title", "separator", "os", "packages", "shell", "command", "terminal"}
	if !reflect.DeepEqual(cfg.Modules.Items, wantItems) {
		t.Fatalf("module order = %q, want %q", cfg.Modules.Items, wantItems)
	}
	wantModuleFields := map[string]struct {
		key  string
		icon string
	}{
		"os":       {key: "OS", icon: "\uf313"},
		"packages": {key: "Packages", icon: "\uf487"},
		"shell":    {key: "Shell", icon: "\uf489"},
		"command":  {key: "Python", icon: "\ue73c"},
		"terminal": {key: "Terminal", icon: "\uf489"},
	}
	for name, want := range wantModuleFields {
		if got := cfg.Modules.Settings[name].Key; got != want.key {
			t.Errorf("%s key = %q, want %q", name, got, want.key)
		}
		if got := cfg.Modules.Settings[name].KeyIcon; got != want.icon {
			t.Errorf("%s key_icon = %q, want %q", name, got, want.icon)
		}
	}
	if got := cfg.Modules.Settings["title"].Text; got != "You again?" {
		t.Fatalf("title text = %q, want %q", got, "You again?")
	}
	if got := cfg.Modules.Settings["command"].Text; got != "python -V 2>&1 | cut -d' ' -f2" {
		t.Fatalf("Python command = %q", got)
	}
	specs, specDiagnostics := cfg.Specs()
	if len(specDiagnostics) != 0 || len(specs) != len(wantItems) {
		t.Fatalf("Specs() = %d entries, diagnostics=%+v", len(specs), specDiagnostics)
	}
}

func TestModuleItemsTrimWhitespaceAndKeepOrder(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    items:
      - " shell "
      - " title "
      - "separator"
`))
	want := []string{"shell", "title", "separator"}
	if !reflect.DeepEqual(cfg.Modules.Items, want) {
		t.Fatalf("module items = %q, want %q", cfg.Modules.Items, want)
	}
	if hasWarning(diagnostics, "Invalid config field") {
		t.Fatalf("valid whitespace-padded module names produced warnings: %+v", diagnostics)
	}
}

func TestUnknownModuleItemsAreIgnoredAndReported(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    items:
      - shell
      - sheell
      - uptime
`))
	want := []string{"shell", "uptime"}
	if !reflect.DeepEqual(cfg.Modules.Items, want) {
		t.Fatalf("module items = %q, want %q", cfg.Modules.Items, want)
	}
	if !hasWarning(diagnostics, `Unknown module "sheell"`) {
		t.Fatalf("unknown module item did not produce a warning: %+v", diagnostics)
	}
}

func TestInvalidModuleItemsKeepDefaultList(t *testing.T) {
	defaults := Default()
	cfg, diagnostics := Load(writeYAML(t, "hexfetch:\n  modules:\n    items: shell\n"))
	if !reflect.DeepEqual(cfg.Modules.Items, defaults.Modules.Items) {
		t.Fatalf("invalid items replaced defaults: %q", cfg.Modules.Items)
	}
	if !hasWarning(diagnostics, "hexfetch.modules.items") {
		t.Fatalf("missing invalid items warning: %+v", diagnostics)
	}
}

func TestResolvedConfigPreservesSeparatorLength(t *testing.T) {
	cfg := Default()
	cfg.Display.SeparatorLength = 18
	data, err := marshalResolvedConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	loaded, diagnostics := loadYAML(Default(), data, "resolved.yaml")
	if loaded.Display.SeparatorLength != 18 {
		t.Fatalf("round-trip separator length = %d, want 18", loaded.Display.SeparatorLength)
	}
	if hasWarning(diagnostics, "separator_length") {
		t.Fatalf("round-trip produced separator length warning: %+v", diagnostics)
	}
}

func TestInvalidSeparatorLengthKeepsAutomaticWidth(t *testing.T) {
	for _, value := range []string{"-1", "4097", "wide"} {
		cfg, diagnostics := Load(writeYAML(t, "hexfetch:\n  display:\n    separator_length: "+value+"\n"))
		if cfg.Display.SeparatorLength != 0 {
			t.Fatalf("separator_length %q = %d, want automatic width", value, cfg.Display.SeparatorLength)
		}
		if !hasWarning(diagnostics, "hexfetch.display.separator_length") {
			t.Fatalf("separator_length %q did not produce a validation warning: %+v", value, diagnostics)
		}
	}
}

func TestInvalidDisplayColorKeepsOnlyThatFieldDefault(t *testing.T) {
	defaults := Default()
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  display:
    separator: " :: "
    color:
      keys: "38;2;256;0;0"
      output: "38;2;1;2;3"
`))
	if cfg.Display.Separator != " :: " || cfg.Display.Color.Keys != defaults.Display.Color.Keys || cfg.Display.Color.Output != "38;2;1;2;3" {
		t.Fatalf("partial display color validation failed: %+v", cfg.Display)
	}
	if !hasWarning(diagnostics, "hexfetch.display.color.keys") {
		t.Fatalf("missing invalid display color warning: %+v", diagnostics)
	}
}

func TestInvalidModuleFieldsPreserveDefaultsAndValidNeighbors(t *testing.T) {
	document := `hexfetch:
  modules:
    items:
      - title
      - command
      - localip
    title:
      format: 42
    command:
      key: false
      text: "python --version"
      options: invalid
    localip:
      options:
        family: "4"
        interface: 42
        exclude:
          - docker
          - 42
          - veth
        fammily: "6"
        extra: enabled
`
	cfg := Default()
	cfg.Modules.Settings["command"] = ModuleSpec{Type: "command", Key: "Python"}
	cfg.Modules.Settings["localip"] = ModuleSpec{
		Type: "localip",
		Options: map[string]any{
			"family":    "6",
			"interface": "wlan0",
			"exclude":   []any{"old"},
		},
	}
	path := writeYAML(t, document)
	loaded, diagnostics := loadYAML(cfg, mustRead(t, path), path)
	if loaded.Modules.Settings["title"].Format != "" {
		t.Fatalf("invalid title format replaced default: %+v", loaded.Modules.Settings["title"])
	}
	command := loaded.Modules.Settings["command"]
	if command.Key != "Python" || command.Text != "python --version" {
		t.Fatalf("command settings did not merge field by field: %+v", command)
	}
	if command.Options != nil {
		t.Fatalf("invalid command options replaced defaults: %#v", command.Options)
	}
	options := loaded.Modules.Settings["localip"].Options
	if options["family"] != "4" || options["interface"] != "wlan0" {
		t.Fatalf("valid and invalid localip options were not resolved independently: %#v", options)
	}
	exclude, ok := options["exclude"].([]any)
	if !ok || !reflect.DeepEqual(exclude, []any{"docker", "veth"}) {
		t.Fatalf("valid exclude entries were not retained: %#v", options["exclude"])
	}
	for _, field := range []string{
		"hexfetch.modules.title.format",
		"hexfetch.modules.command.key",
		"hexfetch.modules.command.options",
		"hexfetch.modules.localip.options.interface",
		"hexfetch.modules.localip.options.fammily",
		"hexfetch.modules.localip.options.extra",
		"hexfetch.modules.localip.options.exclude[1]",
	} {
		if !hasWarning(diagnostics, field) {
			t.Errorf("missing warning for %s: %+v", field, diagnostics)
		}
	}
}

func TestModuleOptionsValidateTypesRangesAndURLs(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    items:
      - cpuusage
      - localip
      - publicip
      - weather
    cpuusage:
      options:
        interval_ms: 500
    localip:
      options:
        family: "ipv6"
        interface: "wlan0"
        exclude: "docker"
    publicip:
      options:
        url: "https://example.com/ip"
    weather:
      options:
        url: "file:///etc/passwd"
`))
	if cfg.Modules.Settings["cpuusage"].Options["interval_ms"] != 500 {
		t.Fatalf("integer cpuusage interval was not retained: %#v", cfg.Modules.Settings["cpuusage"].Options)
	}
	if cfg.Modules.Settings["localip"].Options["family"] != "ipv6" || cfg.Modules.Settings["localip"].Options["exclude"] != "docker" {
		t.Fatalf("valid localip options were not retained: %#v", cfg.Modules.Settings["localip"].Options)
	}
	if cfg.Modules.Settings["publicip"].Options["url"] != "https://example.com/ip" {
		t.Fatalf("valid publicip URL was not retained: %#v", cfg.Modules.Settings["publicip"].Options)
	}
	if _, exists := cfg.Modules.Settings["weather"].Options["url"]; exists {
		t.Fatal("invalid weather URL was retained")
	}
	if !hasWarning(diagnostics, "hexfetch.modules.weather.options.url") {
		t.Fatalf("invalid weather URL did not produce a warning: %+v", diagnostics)
	}
}

func TestConflictingTitleAndCommandAliasesAreReported(t *testing.T) {
	_, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    title:
      text: "Text title"
      format: "Format title"
    command:
      text: "python --version"
      command: "python -V"
`))
	if !hasWarning(diagnostics, "text takes precedence") || !hasWarning(diagnostics, "command takes precedence") {
		t.Fatalf("conflicting aliases were not reported: %+v", diagnostics)
	}
}

func TestTitleTextAloneDoesNotConflictWithDefaultFormat(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    title:
      text: "You again?"
    command:
      text: "python -V"
`))
	if hasWarning(diagnostics, "takes precedence") {
		t.Fatalf("a value inherited from the defaults was reported as a conflict: %+v", diagnostics)
	}
	if cfg.Modules.Settings["title"].Text != "You again?" {
		t.Fatalf("title text was not retained: %#v", cfg.Modules.Settings["title"])
	}
}

func TestModuleSettingsRejectFieldsNotUsedByModule(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    items:
      - shell
      - separator
      - title
    shell:
      text: "ignored text"
      command: "ignored command"
    separator:
      key: "ignored key"
    title:
      key: "ignored title key"
`))
	if cfg.Modules.Settings["shell"].Text != "" || cfg.Modules.Settings["shell"].Command != "" {
		t.Fatalf("unsupported shell fields were retained: %+v", cfg.Modules.Settings["shell"])
	}
	if cfg.Modules.Settings["separator"].Key != "" || cfg.Modules.Settings["title"].Key != "" {
		t.Fatalf("unsupported special module fields were retained: %+v", cfg.Modules.Settings)
	}
	for _, field := range []string{
		"hexfetch.modules.shell.text",
		"hexfetch.modules.shell.command",
		"hexfetch.modules.separator.key",
		"hexfetch.modules.title.key",
	} {
		if !hasWarning(diagnostics, field) {
			t.Errorf("missing warning for unsupported module field %s: %+v", field, diagnostics)
		}
	}
}

func TestCompletelyInvalidOptionListPreservesPreviousValue(t *testing.T) {
	cfg := Default()
	cfg.Modules.Settings["localip"] = ModuleSpec{
		Type:    "localip",
		Options: map[string]any{"exclude": []any{"existing"}},
	}
	loaded, diagnostics := loadYAML(cfg, []byte(`hexfetch:
  modules:
    localip:
      options:
        exclude:
          - 42
          - false
`), "config.yaml")
	values := loaded.Modules.Settings["localip"].Options["exclude"]
	if !reflect.DeepEqual(values, []any{"existing"}) {
		t.Fatalf("previous list value was replaced by invalid items: %#v", values)
	}
	if !hasWarning(diagnostics, "no valid string items were provided") {
		t.Fatalf("missing invalid list diagnostic: %+v", diagnostics)
	}
}

func TestModuleOptionRangeAndUnknownModuleSettingsAreReported(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, `hexfetch:
  modules:
    items:
      - cpuusage
    cpuusage:
      options:
        interval_ms: 5000
        typo: true
    sheell:
      key: "Shell"
`))
	if _, exists := cfg.Modules.Settings["sheell"]; exists {
		t.Fatal("unknown module settings were retained")
	}
	if _, exists := cfg.Modules.Settings["cpuusage"].Options["interval_ms"]; exists {
		t.Fatal("out-of-range cpuusage interval was retained")
	}
	for _, field := range []string{"hexfetch.modules.sheell", "interval_ms", "hexfetch.modules.cpuusage.options.typo"} {
		if !hasWarning(diagnostics, field) {
			t.Errorf("missing warning for %s: %+v", field, diagnostics)
		}
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSpecsForKeepsConfiguredModuleSettingsAndOrder(t *testing.T) {
	cfg := Default()
	cfg.Modules.Settings["shell"] = ModuleSpec{Type: "shell", Key: "Shell name", Format: "{value}"}
	cfg.Modules.Settings["command"] = ModuleSpec{Type: "command", Key: "Python", Text: "python -V"}
	specs, diagnostics := cfg.SpecsFor([]string{" command ", "title", "shell"})
	if len(diagnostics) != 0 {
		t.Fatalf("SpecsFor() diagnostics = %+v", diagnostics)
	}
	if len(specs) != 3 || specs[0].Type != "command" || specs[0].Key != "Python" || specs[0].Text != "python -V" || specs[1].Format != "" || specs[2].Key != "Shell name" {
		t.Fatalf("SpecsFor() = %+v", specs)
	}
}

func TestSpecsForWarnsWhenModuleSettingsAreSharedByRepeatedItems(t *testing.T) {
	cfg := Default()
	_, diagnostics := cfg.SpecsFor([]string{"shell", "command", "shell"})
	if !hasWarning(diagnostics, `Module "shell" appears more than once`) {
		t.Fatalf("missing repeated module warning: %+v", diagnostics)
	}
}

func TestSpecsForWarnsWhenSeparatorHasNoTitle(t *testing.T) {
	cfg := Default()
	_, diagnostics := cfg.SpecsFor([]string{"separator", "shell"})
	if !hasWarning(diagnostics, "without a title module") {
		t.Fatalf("missing title selection warning: %+v", diagnostics)
	}
}

func TestLoadDefaultYAMLMatchesDefaultConfig(t *testing.T) {
	path := writeYAML(t, DefaultYAML)
	loaded, diagnostics := Load(path)
	if !reflect.DeepEqual(loaded, Default()) {
		t.Fatalf("Load(DefaultYAML) differs from Default():\nloaded: %+v\ndefault: %+v", loaded, Default())
	}
	if hasWarning(diagnostics, "Invalid") || hasWarning(diagnostics, "Unknown") {
		t.Fatalf("default YAML generated warnings: %+v", diagnostics)
	}
}

func TestGenerateCreatesDirectoryAndDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "hexfetch", "config.yaml")
	if err := Generate(path); err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("generated config is missing: %v", err)
	}
	if string(data) != DefaultYAML {
		t.Fatalf("generated config differs from DefaultYAML:\n%s", data)
	}
	if err := os.WriteFile(path, []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Generate(path); err == nil {
		t.Fatal("Generate() overwrote an existing file")
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("existing file changed: %q, %v", data, err)
	}
}

func TestLoadUsesHomeHexfetchConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "hex", "set", "hexfetch.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("hexfetch:\n  display:\n    separator: ' | '\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, diagnostics := Load("")
	if cfg.Display.Separator != " | " {
		t.Fatalf("Display.Separator = %q, want configured value", cfg.Display.Separator)
	}
	if hasWarning(diagnostics, "Unable to") {
		t.Fatalf("unexpected load warning: %+v", diagnostics)
	}
}

func TestOldTOMLPathIsNotMigrated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldPath := filepath.Join(home, "nixos", "nixos.toml")
	legacyDir := filepath.Join(home, ".config", "hexfetch")
	if err := os.MkdirAll(filepath.Dir(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	if err := os.WriteFile(oldPath, []byte("[hexfetch.display]\nseparator = 'old'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.jsonc", "config.json"} {
		if err := os.WriteFile(filepath.Join(legacyDir, name), []byte(`{"display":{"separator":"legacy"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, diagnostics := Load("")
	if cfg.Display.Separator != Default().Display.Separator {
		t.Fatalf("legacy TOML unexpectedly changed defaults: %q", cfg.Display.Separator)
	}
	if !hasWarning(diagnostics, "Found legacy config") {
		t.Fatalf("legacy configuration was ignored without a warning: %+v", diagnostics)
	}
	if _, err := os.Stat(DefaultPath()); !os.IsNotExist(err) {
		t.Fatalf("loading legacy TOML unexpectedly created YAML config: %v", err)
	}
}

func TestMigrateLegacyTOMLToYAMLPreservesSourceAndSupportedSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	source := filepath.Join(home, "nixos", "nixos.toml")
	destination := filepath.Join(home, "hex", "set", "hexfetch.yaml")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := `[hexfetch.display]
separator = " :: "
separator_char = "="

[hexfetch.display.color]
keys = "38;2;1;2;3"

[hexfetch.modules]
items = ["title", "separator", "localip"]

[hexfetch.modules.title]
format = "Migrated title"

[hexfetch.modules.localip.options]
family = "4"
exclude = ["docker", "veth"]

[hexfetch.logo]
source = "old-logo"
`
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := MigrateLegacyTOML(source, destination)
	if err != nil {
		t.Fatalf("MigrateLegacyTOML() error = %v", err)
	}
	if !hasWarning(diagnostics, "Legacy logo settings were omitted") {
		t.Fatalf("legacy logo removal was not reported: %+v", diagnostics)
	}
	original, err := os.ReadFile(source)
	if err != nil || string(original) != legacy {
		t.Fatalf("legacy source changed: %v", err)
	}
	loaded, loadDiagnostics := Load(destination)
	if loaded.Display.Separator != " :: " || loaded.Display.SeparatorChar != "=" || loaded.Display.Color.Keys != "38;2;1;2;3" {
		t.Fatalf("display settings were not migrated: %+v", loaded.Display)
	}
	if loaded.Modules.Settings["title"].Format != "Migrated title" {
		t.Fatalf("title settings were not migrated: %+v", loaded.Modules.Settings["title"])
	}
	options := loaded.Modules.Settings["localip"].Options
	if options["family"] != "4" || !reflect.DeepEqual(options["exclude"], []any{"docker", "veth"}) {
		t.Fatalf("module options were not migrated: %#v", options)
	}
	if hasWarning(loadDiagnostics, "Invalid config") || hasWarning(loadDiagnostics, "Unknown") {
		t.Fatalf("migrated YAML produced validation warnings: %+v", loadDiagnostics)
	}
	if _, err := os.Stat(destination); err != nil {
		t.Fatalf("destination config was not created: %v", err)
	}
}

func TestMigrateLegacyTOMLRestoresConfiguredTitleBeforeSeparator(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy.toml")
	destination := filepath.Join(directory, "config.yaml")
	legacy := `[hexfetch.modules]
items = ["separator", "shell", "uptime"]

[hexfetch.modules.title]
format = "Welcome!"
`
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := MigrateLegacyTOML(source, destination)
	if err != nil {
		t.Fatalf("MigrateLegacyTOML() error = %v", err)
	}
	loaded, loadDiagnostics := Load(destination)
	want := []string{"title", "separator", "shell", "uptime"}
	if !reflect.DeepEqual(loaded.Modules.Items, want) {
		t.Fatalf("migrated module order = %q, want %q", loaded.Modules.Items, want)
	}
	if !hasWarning(diagnostics, "migration inserted title before the separator") {
		t.Fatalf("title restoration was not reported: %+v", diagnostics)
	}
	if hasWarning(loadDiagnostics, "Invalid config") || hasWarning(loadDiagnostics, "Unknown") {
		t.Fatalf("migrated YAML is invalid: %+v", loadDiagnostics)
	}
}

func TestMigrateLegacyTOMLReportsUnknownRootTables(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy.toml")
	destination := filepath.Join(directory, "config.yaml")
	legacy := "[hexfetch.display]\nseparator = \" :: \"\n\n[other]\nvalue = true\n"
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := MigrateLegacyTOML(source, destination)
	if err != nil {
		t.Fatalf("MigrateLegacyTOML() error = %v", err)
	}
	if !hasWarning(diagnostics, "Unknown legacy TOML root field \"other\"") {
		t.Fatalf("unknown TOML root table was not reported: %+v", diagnostics)
	}
	loaded, loadDiagnostics := Load(destination)
	if loaded.Display.Separator != " :: " {
		t.Fatalf("supported TOML value was not migrated: %+v", loaded.Display)
	}
	if hasWarning(loadDiagnostics, "Unknown") || hasWarning(loadDiagnostics, "Invalid config") {
		t.Fatalf("normalized YAML contains unsupported fields: %+v", loadDiagnostics)
	}
}

func TestMigrateLegacyTOMLIgnoresInvalidAndUnknownSettings(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy.toml")
	destination := filepath.Join(directory, "config.yaml")
	legacy := `[hexfetch.modules]
items = ["shell", "sheell"]

[hexfetch.modules.shell]
key = "Custom shell"
extra = "unsupported"

[hexfetch.modules.shell.options]
unknown_option = true

[hexfetch.modules.sheell]
key = "Typo"
`
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics, err := MigrateLegacyTOML(source, destination)
	if err != nil {
		t.Fatalf("MigrateLegacyTOML() error = %v", err)
	}
	for _, expected := range []string{"Unknown module \"sheell\"", "Unknown config field", "Unknown module option"} {
		if !hasWarning(diagnostics, expected) {
			t.Errorf("missing migration warning %q: %+v", expected, diagnostics)
		}
	}
	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, omitted := range []string{"sheell", "unsupported", "unknown_option", "Typo"} {
		if strings.Contains(text, omitted) {
			t.Errorf("unsupported legacy setting %q was written to migrated YAML: %s", omitted, text)
		}
	}
	loaded, loadDiagnostics := Load(destination)
	if loaded.Modules.Settings["shell"].Key != "Custom shell" {
		t.Fatalf("supported module setting was not migrated: %+v", loaded.Modules.Settings["shell"])
	}
	if hasWarning(loadDiagnostics, "Unknown") || hasWarning(loadDiagnostics, "Invalid config") {
		t.Fatalf("normalized migrated YAML still contains invalid settings: %+v", loadDiagnostics)
	}
}

func TestMigrateLegacyTOMLDoesNotOverwriteDestination(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy.toml")
	destination := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(source, []byte("[hexfetch.display]\nseparator = \"old\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyTOML(source, destination); err == nil {
		t.Fatal("migration overwrote an existing destination")
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "keep" {
		t.Fatalf("destination changed: %q, %v", data, err)
	}
}

func TestYAMLRejectsMultipleDocumentsAndDuplicateKeys(t *testing.T) {
	inputs := []string{
		"hexfetch: {}\n---\nhexfetch: {}\n",
		"hexfetch:\n  display:\n    separator: one\n    separator: two\n",
	}
	for _, input := range inputs {
		cfg, diagnostics := Load(writeYAML(t, input))
		if !reflect.DeepEqual(cfg, Default()) {
			t.Errorf("invalid YAML changed defaults for %q", input)
		}
		if !hasWarning(diagnostics, "YAML parsing error") {
			t.Errorf("missing YAML parsing warning for %q: %+v", input, diagnostics)
		}
	}
}

func TestYAMLSyntaxErrorKeepsDefaults(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, "hexfetch:\n  display: [\n"))
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("invalid YAML changed defaults: %+v", cfg)
	}
	if !hasWarning(diagnostics, "YAML parsing error") {
		t.Fatalf("missing YAML parsing warning: %+v", diagnostics)
	}
}

func TestParseColorSupportsExistingValues(t *testing.T) {
	cases := []struct {
		value any
		want  Color
	}{
		{value: int64(4), want: Palette(4)},
		{value: "4", want: Palette(4)},
		{value: "0", want: Palette(0)},
		{value: "blue", want: "34"},
		{value: "light_cyan", want: "96"},
		{value: "38;2;135;215;255", want: "38;2;135;215;255"},
	}
	for _, test := range cases {
		if got, err := ParseColor(test.value); err != nil || got != test.want {
			t.Errorf("ParseColor(%v) = %q, %v; want %q", test.value, got, err, test.want)
		}
	}
	for _, value := range []any{int64(-1), int64(256), 4.0, true, "not-a-color", "999", "38;2;256;0;0", "38;5;256", "38;2;1;2", "38;;2;1;2;3"} {
		if _, err := ParseColor(value); err == nil {
			t.Errorf("ParseColor(%v) succeeded, want error", value)
		}
	}
}

func TestUnknownTopLevelFieldsAreReported(t *testing.T) {
	_, diagnostics := Load(writeYAML(t, "other: true\nhexfetch: {}\n"))
	if !hasWarning(diagnostics, "Unknown config field \"other\"") {
		t.Fatalf("unknown top-level field was not reported: %+v", diagnostics)
	}
}

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	cfg, diagnostics := Load("")
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("missing config changed defaults: %+v", cfg)
	}
	if len(diagnostics) != 1 || diagnostics[0].Level != diag.Info || !strings.Contains(diagnostics[0].Message, "using built-in defaults") {
		t.Fatalf("missing config diagnostics = %+v", diagnostics)
	}
}

func FuzzParseColor(f *testing.F) {
	for _, value := range []string{"38;2;1;2;3", "38;5;255", "blue", "38;2;256;0;0", ""} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = ParseColor(value)
	})
}

func FuzzLoadYAML(f *testing.F) {
	for _, value := range []string{"hexfetch: {}", "", "hexfetch: [", "hexfetch:\n  modules:\n    items: [shell]"} {
		f.Add([]byte(value))
	}
	defaults := Default()
	f.Fuzz(func(t *testing.T, data []byte) {
		loadYAML(defaults, data, "fuzz.yaml")
	})
}

func TestLoadKeyIconAndKeyType(t *testing.T) {
	path := writeYAML(t, "hexfetch:\n  display:\n    key:\n      type: both\n  modules:\n    items:\n      - os\n    os:\n      key: \"OS\"\n      key_icon: \"\\uf313\"\n")
	loaded, diagnostics := Load(path)
	for _, diagnostic := range diagnostics {
		if diagnostic.Level != diag.Info {
			t.Fatalf("unexpected diagnostic: %+v", diagnostic)
		}
	}
	if loaded.Display.KeyType != "both" {
		t.Fatalf("KeyType = %q, want both", loaded.Display.KeyType)
	}
	if got := loaded.Modules.Settings["os"].KeyIcon; got != "\uf313" {
		t.Fatalf("KeyIcon = %q, want U+F313", got)
	}
}

func TestLoadInvalidKeyTypeKeepsDefault(t *testing.T) {
	path := writeYAML(t, "hexfetch:\n  display:\n    key:\n      type: wide\n")
	loaded, diagnostics := Load(path)
	if loaded.Display.KeyType != "" {
		t.Fatalf("KeyType = %q, want the default", loaded.Display.KeyType)
	}
	found := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == diag.Warning && strings.Contains(diagnostic.Message, "hexfetch.display.key.type") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing warning for invalid key type: %+v", diagnostics)
	}
}

func TestEmbeddedDefaultYAMLIsNotEmpty(t *testing.T) {
	if !strings.Contains(DefaultYAML, "\nhexfetch:\n") {
		t.Fatalf("embedded default.yaml has no hexfetch section: %q", DefaultYAML)
	}
}

func TestDeprecatedNixfetchRootIsAliasedWithWarning(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, "nixfetch:\n  display:\n    separator: \" :: \"\n"))
	if cfg.Display.Separator != " :: " {
		t.Fatalf("separator = %q, want the value from the nixfetch root", cfg.Display.Separator)
	}
	if !hasWarning(diagnostics, "deprecated") {
		t.Fatalf("missing deprecation warning: %+v", diagnostics)
	}
	if hasWarning(diagnostics, "Unknown config field") || hasWarning(diagnostics, "not found") {
		t.Fatalf("alias still reports an unknown root: %+v", diagnostics)
	}
}

func TestEmptyConfigFileIsNotAnError(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, "# only a comment\n"))
	if !reflect.DeepEqual(cfg, Default()) {
		t.Fatalf("empty config changed defaults: %+v", cfg)
	}
	for _, diagnostic := range diagnostics {
		if diagnostic.Level != diag.Info {
			t.Fatalf("empty config produced %+v", diagnostic)
		}
	}
}

func TestMissingExplicitConfigPathWarns(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	_, diagnostics := Load(missing)
	if !hasWarning(diagnostics, "not found") {
		t.Fatalf("explicit missing path did not warn: %+v", diagnostics)
	}
}

func TestYAMLErrorMessageContainsReason(t *testing.T) {
	_, diagnostics := Load(writeYAML(t, "hexfetch:\n  display: [\n"))
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == diag.Warning && strings.Contains(diagnostic.Message, "YAML parsing error") {
			if !strings.Contains(diagnostic.Message, "yaml:") {
				t.Fatalf("message has no parser reason: %q", diagnostic.Message)
			}
			return
		}
	}
	t.Fatalf("missing YAML parsing warning: %+v", diagnostics)
}

func TestMigrationKeepsKeyTypeAndKeyIcon(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "legacy.toml")
	destination := filepath.Join(directory, "hexfetch.yaml")
	legacy := "[hexfetch.display.key]\ntype = \"both\"\n\n[hexfetch.modules]\nitems = [\"os\"]\n\n[hexfetch.modules.os]\nkey_icon = \"X\"\n"
	if err := os.WriteFile(source, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MigrateLegacyTOML(source, destination); err != nil {
		t.Fatal(err)
	}
	loaded, _ := Load(destination)
	if loaded.Display.KeyType != "both" {
		t.Fatalf("migrated KeyType = %q, want both", loaded.Display.KeyType)
	}
	if got := loaded.Modules.Settings["os"].KeyIcon; got != "X" {
		t.Fatalf("migrated KeyIcon = %q, want X", got)
	}
}

func TestDefaultLogoUsesAutoDetection(t *testing.T) {
	if got := Default().Logo; got != "auto" {
		t.Fatalf("Default().Logo = %q, want auto", got)
	}
	if !strings.Contains(DefaultYAML, "\n  logo: auto\n") {
		t.Fatal("DefaultYAML does not select automatic logo detection")
	}
}

func TestLogoNameFromYAML(t *testing.T) {
	cfg, diagnostics := Load(writeYAML(t, "hexfetch:\n  logo: arch\n"))
	if cfg.Logo != "arch" || hasWarning(diagnostics, "hexfetch.logo") {
		t.Fatalf("logo = %q, diagnostics = %+v", cfg.Logo, diagnostics)
	}
	cfg, diagnostics = Load(writeYAML(t, "hexfetch:\n  logo: 5\n"))
	if cfg.Logo != "auto" || !hasWarning(diagnostics, "hexfetch.logo") {
		t.Fatalf("invalid logo changed the default: %q, diagnostics = %+v", cfg.Logo, diagnostics)
	}
}
