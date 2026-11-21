package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"hexfetch/internal/diag"
	"hexfetch/internal/gradient"
	"hexfetch/internal/logo"
	moduleRegistry "hexfetch/modules"
)

//go:embed default.yaml
var DefaultYAML string

const (
	// UserDirName, UserSubDirName and UserFileName locate the personal config: $HOME/hex/set/hexfetch.yaml.
	UserDirName    = "hex"
	UserSubDirName = "set"
	UserFileName   = "hexfetch.yaml"
)

type ColorConfig struct {
	Keys      Color
	Title     Color
	Output    Color
	Separator Color
}

type DisplayConfig struct {
	Separator       string
	SeparatorChar   string
	SeparatorLength int
	KeyType         string
	Color           ColorConfig
}

type ModuleSpec struct {
	Type     string
	Key      string
	KeyIcon  string
	Text     string
	Command  string
	KeyColor *Color
	Format   string
	Options  map[string]any
}

type ModulesConfig struct {
	Items    []string
	Settings map[string]ModuleSpec
}

type Config struct {
	LogoColors [12]gradient.RGB
	Display    DisplayConfig
	Modules    ModulesConfig
}

func Default() Config {
	base := Config{
		LogoColors: logo.DefaultColors(),
		Modules: ModulesConfig{
			Settings: make(map[string]ModuleSpec),
		},
	}
	cfg, _ := loadYAML(base, []byte(DefaultYAML), "built-in defaults")
	return cfg
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, UserDirName, UserSubDirName, UserFileName)
}

func Load(path string) (Config, []diag.Diagnostic) {
	cfg := Default()
	configPath := path
	if configPath == "" {
		configPath = DefaultPath()
		if configPath == "" {
			return cfg, []diag.Diagnostic{{
				Level:   diag.Warning,
				Source:  "config",
				Message: "Unable to determine the default config path.",
			}}
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			missingLevel := diag.Info
			if path != "" {
				missingLevel = diag.Warning
			}
			diagnostics := []diag.Diagnostic{{
				Level:   missingLevel,
				Source:  configPath,
				Message: fmt.Sprintf("Config file %q not found; using built-in defaults.", configPath),
			}}
			if path == "" {
				if legacyPath := findLegacyConfigPath(); legacyPath != "" {
					diagnostics = append(diagnostics, diag.Diagnostic{
						Level:   diag.Warning,
						Source:  legacyPath,
						Message: fmt.Sprintf("Found legacy config %q; it is not loaded. Convert its settings to %q. The legacy file was left unchanged.", legacyPath, configPath),
					})
				}
			}
			return cfg, diagnostics
		}
		return cfg, []diag.Diagnostic{{
			Level:   diag.Warning,
			Source:  configPath,
			Message: fmt.Sprintf("Unable to read config file %q: %s", configPath, err.Error()),
		}}
	}
	return loadYAML(cfg, data, configPath)
}

func loadYAML(cfg Config, data []byte, configPath string) (Config, []diag.Diagnostic) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		if errors.Is(err, io.EOF) {
			return cfg, []diag.Diagnostic{{
				Level:   diag.Info,
				Source:  configPath,
				Message: fmt.Sprintf("Config file %q is empty; using built-in defaults.", configPath),
			}}
		}
		return cfg, []diag.Diagnostic{{
			Level:   diag.Warning,
			Source:  configPath,
			Message: fmt.Sprintf("YAML parsing error in %q: %s", configPath, err.Error()),
		}}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		detail := "multiple YAML documents are not supported"
		if err != nil {
			detail = err.Error()
		}
		return cfg, []diag.Diagnostic{{
			Level:   diag.Warning,
			Source:  configPath,
			Message: fmt.Sprintf("YAML parsing error in %q: %s", configPath, detail),
		}}
	}
	if document == nil {
		return cfg, []diag.Diagnostic{{
			Level:   diag.Warning,
			Source:  configPath,
			Message: "Invalid YAML document.",
			Detail:  "expected a mapping",
		}}
	}

	diagnostics := []diag.Diagnostic{{
		Level:   diag.Info,
		Source:  configPath,
		Message: fmt.Sprintf("Loaded config file %q.", configPath),
	}}
	if _, present := document["hexfetch"]; !present {
		if legacyRoot, found := document["nixfetch"]; found {
			document["hexfetch"] = legacyRoot
			delete(document, "nixfetch")
			diagnostics = append(diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  configPath,
				Message: `Config root "nixfetch" is deprecated; rename it to "hexfetch".`,
			})
		}
	}
	checkUnknown(&diagnostics, configPath, "", document, "hexfetch")
	hexfetchValue, exists := document["hexfetch"]
	if !exists {
		diagnostics = append(diagnostics, diag.Diagnostic{
			Level:   diag.Warning,
			Source:  configPath,
			Message: `Config section "hexfetch" not found; using built-in hexfetch defaults.`,
		})
		return cfg, diagnostics
	}
	hexfetch, ok := asStringMap(hexfetchValue)
	if !ok {
		addDiagnostic(&diagnostics, configPath, "hexfetch", "expected a YAML mapping")
		return cfg, diagnostics
	}
	checkUnknown(&diagnostics, configPath, "hexfetch", hexfetch, "logo_colors", "display", "modules")
	decodeLogoColors(hexfetch, &cfg, configPath, &diagnostics)
	decodeDisplay(hexfetch, &cfg, configPath, &diagnostics)
	decodeModules(hexfetch, &cfg, configPath, &diagnostics)
	return cfg, diagnostics
}

func decodeLogoColors(section map[string]any, cfg *Config, source string, diagnostics *[]diag.Diagnostic) {
	raw, exists := section["logo_colors"]
	if !exists {
		return
	}
	items, ok := raw.([]any)
	if !ok {
		addDiagnostic(diagnostics, source, "hexfetch.logo_colors", "expected a sequence of exactly 12 38;2;R;G;B strings")
		return
	}
	if len(items) != len(cfg.LogoColors) {
		addDiagnostic(diagnostics, source, "hexfetch.logo_colors", fmt.Sprintf("expected exactly 12 colors, got %d", len(items)))
		return
	}
	for index, item := range items {
		value, ok := item.(string)
		if !ok {
			addDiagnostic(diagnostics, source, fmt.Sprintf("hexfetch.logo_colors[%d]", index), "expected a 38;2;R;G;B string")
			continue
		}
		color, err := gradient.ParseStop(value)
		if err != nil {
			addDiagnostic(diagnostics, source, fmt.Sprintf("hexfetch.logo_colors[%d]", index), err.Error())
			continue
		}
		cfg.LogoColors[index] = color
	}
}

func OverrideLogoColors(base [12]gradient.RGB, values []string) ([12]gradient.RGB, []diag.Diagnostic) {
	if len(values) != len(base) {
		return base, []diag.Diagnostic{{
			Level:   diag.Warning,
			Source:  "cli",
			Message: fmt.Sprintf("Invalid --logo-colors value: expected exactly 12 colors, got %d; keeping the current colors.", len(values)),
		}}
	}
	result := base
	var diagnostics []diag.Diagnostic
	for index, value := range values {
		color, err := gradient.ParseStop(value)
		if err != nil {
			diagnostics = append(diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  "cli",
				Message: fmt.Sprintf("Invalid --logo-colors value at index %d; keeping the configured color.", index+1),
				Detail:  err.Error(),
			})
			continue
		}
		result[index] = color
	}
	return result, diagnostics
}

func decodeDisplay(section map[string]any, cfg *Config, source string, diagnostics *[]diag.Diagnostic) {
	value, exists := section["display"]
	if !exists {
		return
	}
	fields, ok := asStringMap(value)
	if !ok {
		addDiagnostic(diagnostics, source, "hexfetch.display", "expected a YAML mapping")
		return
	}
	checkUnknown(diagnostics, source, "hexfetch.display", fields, "separator", "separator_char", "separator_length", "key", "color")
	decodeString(fields, "separator", "hexfetch.display.separator", &cfg.Display.Separator, source, diagnostics)
	decodeString(fields, "separator_char", "hexfetch.display.separator_char", &cfg.Display.SeparatorChar, source, diagnostics)
	if rawLength, present := fields["separator_length"]; present {
		length, valid := configInteger(rawLength)
		if !valid || length < 0 || length > 4096 {
			addDiagnostic(diagnostics, source, "hexfetch.display.separator_length", "expected an integer from 0 to 4096 (0 means automatic width)")
		} else {
			cfg.Display.SeparatorLength = length
		}
	}
	if rawKey, present := fields["key"]; present {
		keyFields, valid := asStringMap(rawKey)
		if !valid {
			addDiagnostic(diagnostics, source, "hexfetch.display.key", "expected a YAML mapping")
		} else {
			checkUnknown(diagnostics, source, "hexfetch.display.key", keyFields, "type")
			if rawType, typed := keyFields["type"]; typed {
				name, isString := rawType.(string)
				name = strings.ToLower(strings.TrimSpace(name))
				if !isString || !ValidKeyType(name) {
					addDiagnostic(diagnostics, source, "hexfetch.display.key.type", "expected one of: string, icon, both")
				} else {
					cfg.Display.KeyType = name
				}
			}
		}
	}
	value, exists = fields["color"]
	if !exists {
		return
	}
	colors, ok := asStringMap(value)
	if !ok {
		addDiagnostic(diagnostics, source, "hexfetch.display.color", "expected a YAML mapping")
		return
	}
	checkUnknown(diagnostics, source, "hexfetch.display.color", colors, "keys", "title", "output", "separator")
	colorFields := []struct {
		name   string
		target *Color
	}{
		{name: "keys", target: &cfg.Display.Color.Keys},
		{name: "title", target: &cfg.Display.Color.Title},
		{name: "output", target: &cfg.Display.Color.Output},
		{name: "separator", target: &cfg.Display.Color.Separator},
	}
	for _, field := range colorFields {
		if raw, exists := colors[field.name]; exists {
			color, err := ParseColor(raw)
			if err != nil {
				addDiagnostic(diagnostics, source, "hexfetch.display.color."+field.name, err.Error())
				continue
			}
			*field.target = color
		}
	}
}

func decodeModules(section map[string]any, cfg *Config, source string, diagnostics *[]diag.Diagnostic) {
	value, exists := section["modules"]
	if !exists {
		return
	}
	fields, ok := asStringMap(value)
	if !ok {
		addDiagnostic(diagnostics, source, "hexfetch.modules", "expected a YAML mapping")
		return
	}
	if cfg.Modules.Settings == nil {
		cfg.Modules.Settings = make(map[string]ModuleSpec)
	}
	if rawItems, exists := fields["items"]; exists {
		items, ok := rawItems.([]any)
		if !ok {
			addDiagnostic(diagnostics, source, "hexfetch.modules.items", "expected a sequence of module names")
		} else {
			parsed := make([]string, 0, len(items))
			for index, item := range items {
				name, valid := item.(string)
				if !valid || strings.TrimSpace(name) == "" {
					addDiagnostic(diagnostics, source, fmt.Sprintf("hexfetch.modules.items[%d]", index), "expected a non-empty module name")
					continue
				}
				name = strings.TrimSpace(name)
				if !moduleRegistry.Exists(name) {
					*diagnostics = append(*diagnostics, diag.Diagnostic{
						Level:   diag.Warning,
						Source:  source,
						Message: fmt.Sprintf("Unknown module %q in hexfetch.modules.items[%d]; the item was ignored.", name, index),
					})
					continue
				}
				parsed = append(parsed, name)
			}
			cfg.Modules.Items = parsed
		}
	}
	for _, name := range sortedKeys(fields) {
		if name == "items" {
			continue
		}
		if !moduleRegistry.Exists(name) {
			*diagnostics = append(*diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  source,
				Message: fmt.Sprintf("Unknown module configuration section %q; settings were ignored.", "hexfetch.modules."+name),
			})
			continue
		}
		moduleFields, valid := asStringMap(fields[name])
		if !valid {
			addDiagnostic(diagnostics, source, "hexfetch.modules."+name, "expected a YAML mapping")
			continue
		}
		spec, exists := cfg.Modules.Settings[name]
		if !exists {
			spec = ModuleSpec{Type: name}
		}
		spec.Type = name
		allowedFields := append(moduleRegistry.ConfigFields(name), "options")
		checkUnknown(diagnostics, source, "hexfetch.modules."+name, moduleFields, allowedFields...)
		if hasName(allowedFields, "key") {
			decodeString(moduleFields, "key", "hexfetch.modules."+name+".key", &spec.Key, source, diagnostics)
		}
		if hasName(allowedFields, "text") {
			decodeString(moduleFields, "text", "hexfetch.modules."+name+".text", &spec.Text, source, diagnostics)
		}
		if hasName(allowedFields, "command") {
			decodeString(moduleFields, "command", "hexfetch.modules."+name+".command", &spec.Command, source, diagnostics)
		}
		if hasName(allowedFields, "format") {
			decodeString(moduleFields, "format", "hexfetch.modules."+name+".format", &spec.Format, source, diagnostics)
		}
		if hasName(allowedFields, "key_icon") {
			decodeString(moduleFields, "key_icon", "hexfetch.modules."+name+".key_icon", &spec.KeyIcon, source, diagnostics)
		}
		// The conflict is judged on the values written in this file only.
		// spec was seeded from the built-in defaults (title.format is
		// "Welcome!"), so comparing spec fields would warn about a value the
		// user never wrote.
		fileText, _ := moduleFields["text"].(string)
		fileFormat, _ := moduleFields["format"].(string)
		fileCommand, _ := moduleFields["command"].(string)
		if name == "title" && fileText != "" && fileFormat != "" {
			*diagnostics = append(*diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  source,
				Message: "Both hexfetch.modules.title.text and format are set; text takes precedence.",
			})
		}
		if name == "command" && fileText != "" && fileCommand != "" {
			*diagnostics = append(*diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  source,
				Message: "Both hexfetch.modules.command.text and command are set; command takes precedence.",
			})
		}
		if raw, exists := moduleFields["key_color"]; exists && hasName(allowedFields, "key_color") {
			color, err := ParseColor(raw)
			if err != nil {
				addDiagnostic(diagnostics, source, "hexfetch.modules."+name+".key_color", err.Error())
			} else {
				spec.KeyColor = &color
			}
		}
		if raw, exists := moduleFields["options"]; exists {
			options, valid := asStringMap(raw)
			if !valid {
				addDiagnostic(diagnostics, source, "hexfetch.modules."+name+".options", "expected a YAML mapping")
			} else {
				validated := validateModuleOptions(name, options, source, diagnostics)
				spec.Options = mergeOptions(spec.Options, validated)
			}
		}
		cfg.Modules.Settings[name] = spec
	}
}

func validateModuleOptions(name string, options map[string]any, source string, diagnostics *[]diag.Diagnostic) map[string]any {
	schema := moduleRegistry.OptionSchema(name)
	validated := make(map[string]any, len(options))
	for _, key := range sortedKeys(options) {
		definition, exists := schema[key]
		field := "hexfetch.modules." + name + ".options." + key
		if !exists {
			*diagnostics = append(*diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  source,
				Message: fmt.Sprintf("Unknown module option %q.", field),
			})
			continue
		}
		value, err := normalizeModuleOption(definition, options[key], source, field, diagnostics)
		if err != nil {
			addDiagnostic(diagnostics, source, field, err.Error())
			continue
		}
		validated[key] = value
	}
	return validated
}

func normalizeModuleOption(definition moduleRegistry.OptionDefinition, raw any, source, field string, diagnostics *[]diag.Diagnostic) (any, error) {
	switch definition.Kind {
	case moduleRegistry.OptionString:
		value, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected a string, got %T", raw)
		}
		value = strings.TrimSpace(value)
		if len(definition.Allowed) > 0 {
			allowed := false
			for _, candidate := range definition.Allowed {
				if value == candidate {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("expected one of: %s", strings.Join(definition.Allowed, ", "))
			}
		}
		if definition.URL && value != "" {
			parsed, err := url.Parse(value)
			scheme := ""
			if parsed != nil {
				scheme = strings.ToLower(parsed.Scheme)
			}
			if err != nil || parsed == nil || (scheme != "http" && scheme != "https") || parsed.Host == "" {
				return nil, fmt.Errorf("expected an absolute HTTP(S) URL")
			}
		}
		return value, nil
	case moduleRegistry.OptionInteger:
		value, ok := configInteger(raw)
		if !ok {
			return nil, fmt.Errorf("expected an integer, got %T", raw)
		}
		if definition.HasRange && (value < definition.Minimum || value > definition.Maximum) {
			return nil, fmt.Errorf("expected an integer between %d and %d", definition.Minimum, definition.Maximum)
		}
		return value, nil
	case moduleRegistry.OptionStringList:
		if value, ok := raw.(string); ok && definition.AllowScalar {
			value = strings.TrimSpace(value)
			if value == "" {
				return nil, fmt.Errorf("expected a non-empty string")
			}
			return value, nil
		}
		items, ok := raw.([]any)
		if !ok {
			return nil, fmt.Errorf("expected a sequence of strings, got %T", raw)
		}
		values := make([]any, 0, len(items))
		for index, item := range items {
			value, valid := item.(string)
			if !valid || strings.TrimSpace(value) == "" {
				addDiagnostic(diagnostics, source, fmt.Sprintf("%s[%d]", field, index), "expected a non-empty string; this item was ignored")
				continue
			}
			values = append(values, strings.TrimSpace(value))
		}
		if len(items) > 0 && len(values) == 0 {
			return nil, fmt.Errorf("no valid string items were provided")
		}
		return values, nil
	default:
		return nil, fmt.Errorf("unsupported option schema")
	}
}

func configInteger(raw any) (int, bool) {
	switch value := raw.(type) {
	case int:
		return value, true
	case int64:
		converted := int(value)
		return converted, int64(converted) == value
	case uint64:
		if value > uint64(^uint(0)>>1) {
			return 0, false
		}
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func decodeString(fields map[string]any, key, field string, target *string, source string, diagnostics *[]diag.Diagnostic) {
	raw, exists := fields[key]
	if !exists {
		return
	}
	value, ok := raw.(string)
	if !ok {
		addDiagnostic(diagnostics, source, field, fmt.Sprintf("expected a string, got %T", raw))
		return
	}
	*target = value
}

func ValidKeyType(value string) bool {
	switch value {
	case "string", "icon", "both":
		return true
	}
	return false
}

func hasName(values []string, name string) bool {
	for _, value := range values {
		if value == name {
			return true
		}
	}
	return false
}

func checkUnknown(diagnostics *[]diag.Diagnostic, source, prefix string, fields map[string]any, allowed ...string) {
	known := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		known[name] = struct{}{}
	}
	for _, name := range sortedKeys(fields) {
		if _, exists := known[name]; !exists {
			field := name
			if prefix != "" {
				field = prefix + "." + name
			}
			*diagnostics = append(*diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  source,
				Message: fmt.Sprintf("Unknown config field %q.", field),
			})
		}
	}
}

func addDiagnostic(diagnostics *[]diag.Diagnostic, source, field, detail string) {
	*diagnostics = append(*diagnostics, diag.Diagnostic{
		Level:   diag.Warning,
		Source:  source,
		Message: fmt.Sprintf("Invalid config field %q: %s.", field, detail),
	})
}

func asStringMap(value any) (map[string]any, bool) {
	result, ok := value.(map[string]any)
	return result, ok
}

func mergeOptions(base, override map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(override))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range override {
		if nested, ok := asStringMap(value); ok {
			if previous, ok := asStringMap(merged[key]); ok {
				merged[key] = mergeOptions(previous, nested)
				continue
			}
		}
		merged[key] = value
	}
	return merged
}

func LegacyTOMLPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, "nixos", "nixos.toml")
}

func findLegacyConfigPath() string {
	if path := LegacyTOMLPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	configDirectory, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	legacyDirectory := filepath.Join(configDirectory, "hexfetch")
	for _, name := range []string{"config.jsonc", "config.json"} {
		path := filepath.Join(legacyDirectory, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func MigrateLegacyTOML(sourcePath, destinationPath string) ([]diag.Diagnostic, error) {
	if strings.TrimSpace(sourcePath) == "" {
		sourcePath = LegacyTOMLPath()
	}
	if strings.TrimSpace(destinationPath) == "" {
		destinationPath = DefaultPath()
	}
	if sourcePath == "" || destinationPath == "" {
		return nil, errors.New("cannot determine legacy or destination config path")
	}
	sourceAbsolute, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("resolve legacy config path: %w", err)
	}
	destinationAbsolute, err := filepath.Abs(destinationPath)
	if err != nil {
		return nil, fmt.Errorf("resolve destination config path: %w", err)
	}
	if sourceAbsolute == destinationAbsolute {
		return nil, errors.New("legacy config and destination config paths must be different")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read legacy TOML config %q: %w", sourcePath, err)
	}
	var document map[string]any
	if err := toml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse legacy TOML config %q: %w", sourcePath, err)
	}
	hexfetchValue, exists := document["hexfetch"]
	if !exists {
		return nil, errors.New("legacy TOML config does not contain the hexfetch table")
	}
	hexfetch, ok := asStringMap(hexfetchValue)
	if !ok {
		return nil, errors.New("legacy TOML hexfetch value must be a table")
	}
	var diagnostics []diag.Diagnostic
	for _, name := range sortedKeys(document) {
		if name != "hexfetch" {
			diagnostics = append(diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  sourcePath,
				Message: fmt.Sprintf("Unknown legacy TOML root field %q was omitted during migration.", name),
			})
		}
	}
	if _, exists := hexfetch["logo"]; exists {
		delete(hexfetch, "logo")
		diagnostics = append(diagnostics, diag.Diagnostic{
			Level:   diag.Warning,
			Source:  sourcePath,
			Message: "Legacy logo settings were omitted because the current logo is fixed and has no equivalent configuration fields.",
		})
	}
	restoreLegacyTitleSelection(hexfetch, sourcePath, &diagnostics)
	converted, err := yaml.Marshal(map[string]any{"hexfetch": hexfetch})
	if err != nil {
		return diagnostics, fmt.Errorf("encode migrated YAML config: %w", err)
	}
	resolved, validationDiagnostics := loadYAML(Default(), converted, destinationPath)
	for _, diagnostic := range validationDiagnostics {
		if diagnostic.Level != diag.Info {
			diagnostics = append(diagnostics, diagnostic)
		}
	}
	normalized, err := marshalResolvedConfig(resolved)
	if err != nil {
		return diagnostics, fmt.Errorf("encode validated YAML config: %w", err)
	}
	if err := writeNewFile(destinationPath, normalized, 0o644); err != nil {
		return diagnostics, fmt.Errorf("write migrated YAML config: %w", err)
	}
	diagnostics = append(diagnostics, diag.Diagnostic{
		Level:   diag.Info,
		Source:  destinationPath,
		Message: fmt.Sprintf("Migrated TOML config %q to %q; the original file was preserved.", sourcePath, destinationPath),
	})
	return diagnostics, nil
}

func restoreLegacyTitleSelection(hexfetch map[string]any, source string, diagnostics *[]diag.Diagnostic) {
	modules, ok := asStringMap(hexfetch["modules"])
	if !ok {
		return
	}
	title, ok := asStringMap(modules["title"])
	if !ok || (!nonEmptyString(title["text"]) && !nonEmptyString(title["format"])) {
		return
	}
	rawItems, exists := modules["items"]
	if !exists {
		return
	}
	items, ok := legacyStringItems(rawItems)
	if !ok {
		return
	}
	for _, item := range items {
		if item == "title" {
			return
		}
	}
	insertAt := 0
	for index, item := range items {
		if item == "separator" {
			insertAt = index
			break
		}
	}
	items = append(items, "")
	copy(items[insertAt+1:], items[insertAt:])
	items[insertAt] = "title"
	modules["items"] = items
	*diagnostics = append(*diagnostics, diag.Diagnostic{
		Level:   diag.Warning,
		Source:  source,
		Message: "The legacy TOML config defines title settings but omits title from modules.items; migration inserted title before the separator to restore the configured title.",
	})
}

func legacyStringItems(raw any) ([]string, bool) {
	switch items := raw.(type) {
	case []string:
		return append([]string(nil), items...), true
	case []any:
		result := make([]string, 0, len(items))
		for _, item := range items {
			name, ok := item.(string)
			if !ok {
				return nil, false
			}
			result = append(result, name)
		}
		return result, true
	default:
		return nil, false
	}
}

func nonEmptyString(value any) bool {
	text, ok := value.(string)
	return ok && strings.TrimSpace(text) != ""
}

func marshalResolvedConfig(cfg Config) ([]byte, error) {
	logoColors := make([]string, len(cfg.LogoColors))
	for index, color := range cfg.LogoColors {
		logoColors[index] = fmt.Sprintf("38;2;%d;%d;%d", color.R, color.G, color.B)
	}
	display := map[string]any{
		"separator":      cfg.Display.Separator,
		"separator_char": cfg.Display.SeparatorChar,
		"color": map[string]any{
			"keys":      string(cfg.Display.Color.Keys),
			"title":     string(cfg.Display.Color.Title),
			"output":    string(cfg.Display.Color.Output),
			"separator": string(cfg.Display.Color.Separator),
		},
	}
	if cfg.Display.SeparatorLength > 0 {
		display["separator_length"] = cfg.Display.SeparatorLength
	}
	if cfg.Display.KeyType != "" {
		display["key"] = map[string]any{"type": cfg.Display.KeyType}
	}
	moduleSettings := map[string]any{"items": cfg.Modules.Items}
	for _, name := range sortedModuleNames(cfg.Modules.Settings) {
		spec := cfg.Modules.Settings[name]
		fields := make(map[string]any)
		switch name {
		case "title":
			if spec.Text != "" {
				fields["text"] = spec.Text
			} else {
				fields["format"] = spec.Format
			}
		case "command":
			if spec.Command != "" {
				fields["command"] = spec.Command
			} else if spec.Text != "" {
				fields["text"] = spec.Text
			}
			if spec.Key != "" {
				fields["key"] = spec.Key
			}
			if spec.KeyIcon != "" {
				fields["key_icon"] = spec.KeyIcon
			}
			if spec.KeyColor != nil {
				fields["key_color"] = string(*spec.KeyColor)
			}
			if spec.Format != "" {
				fields["format"] = spec.Format
			}
		default:
			for _, field := range moduleRegistry.ConfigFields(name) {
				switch field {
				case "key":
					if spec.Key != "" {
						fields[field] = spec.Key
					}
				case "key_icon":
					if spec.KeyIcon != "" {
						fields[field] = spec.KeyIcon
					}
				case "text":
					if spec.Text != "" {
						fields[field] = spec.Text
					}
				case "command":
					if spec.Command != "" {
						fields[field] = spec.Command
					}
				case "format":
					if spec.Format != "" {
						fields[field] = spec.Format
					}
				case "key_color":
					if spec.KeyColor != nil {
						fields[field] = string(*spec.KeyColor)
					}
				}
			}
		}
		if len(spec.Options) > 0 {
			fields["options"] = spec.Options
		}
		moduleSettings[name] = fields
	}
	hexfetch := map[string]any{
		"logo_colors": logoColors,
		"display":     display,
		"modules":     moduleSettings,
	}
	return yaml.Marshal(map[string]any{"hexfetch": hexfetch})
}

func sortedModuleNames(settings map[string]ModuleSpec) []string {
	names := make([]string, 0, len(settings))
	for name := range settings {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedKeys(fields map[string]any) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (c Config) Specs() ([]ModuleSpec, []diag.Diagnostic) {
	return c.SpecsFor(c.Modules.Items)
}

func (c Config) SpecsFor(items []string) ([]ModuleSpec, []diag.Diagnostic) {
	specs := make([]ModuleSpec, 0, len(items))
	var diagnostics []diag.Diagnostic
	hasTitle := false
	hasSeparator := false
	seen := make(map[string]bool, len(items))
	for _, rawName := range items {
		name := strings.TrimSpace(rawName)
		if name != "" && seen[name] {
			diagnostics = append(diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  "config",
				Message: fmt.Sprintf("Module %q appears more than once in hexfetch.modules.items; every occurrence reuses the same module settings.", name),
			})
		} else if name != "" {
			seen[name] = true
		}
		if name == "title" {
			hasTitle = true
		}
		if name == "separator" {
			hasSeparator = true
		}
	}
	if hasSeparator && !hasTitle {
		diagnostics = append(diagnostics, diag.Diagnostic{
			Level:   diag.Warning,
			Source:  "config",
			Message: "A separator module is selected without a title module; the horizontal line will use the width of the module output. Add title to hexfetch.modules.items to display the configured title.",
		})
	}
	for index, rawName := range items {
		name := strings.TrimSpace(rawName)
		if name == "" {
			diagnostics = append(diagnostics, diag.Diagnostic{
				Level:   diag.Warning,
				Source:  "config",
				Message: fmt.Sprintf("Config module at hexfetch.modules.items[%d] has no name.", index),
			})
			continue
		}
		spec := ModuleSpec{Type: name}
		if configured, exists := c.Modules.Settings[name]; exists {
			spec = configured
			spec.Type = name
		}
		specs = append(specs, spec)
	}
	return specs, diagnostics
}

func Generate(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("cannot determine the config path")
	}
	return writeNewFile(path, []byte(DefaultYAML), 0o644)
}

func writeNewFile(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", directory, err)
	}
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("%s already exists, refusing to overwrite: %w", path, os.ErrExist)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check config path %q: %w", path, err)
	}
	temporary, err := os.CreateTemp(directory, ".config.yaml.tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary config file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return fmt.Errorf("set temporary config permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary config file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary config file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary config file: %w", err)
	}
	if err := os.Link(temporaryPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s already exists, refusing to overwrite: %w", path, os.ErrExist)
		}
		// Some filesystems (Android/Termux app data) forbid hard links.
		return createExclusive(path, data, mode)
	}
	return nil
}

// createExclusive writes a new file without overwriting an existing one.
// It is the fallback for filesystems where os.Link is not permitted.
func createExclusive(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%s already exists, refusing to overwrite: %w", path, os.ErrExist)
		}
		return fmt.Errorf("create config file %q: %w", path, err)
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("write config file %q: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(path)
		return fmt.Errorf("sync config file %q: %w", path, err)
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("close config file %q: %w", path, err)
	}
	return nil
}
