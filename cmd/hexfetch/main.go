package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"hexfetch/internal/cli"
	"hexfetch/internal/config"
	"hexfetch/internal/diag"
	"hexfetch/internal/gradient"
	"hexfetch/internal/logo"
	"hexfetch/internal/render"
	"hexfetch/modules"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	opts, err := cli.Parse(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hexfetch: %v\nTry 'hexfetch --help' for more information.\n", err)
		return 2
	}
	switch {
	case opts.Help:
		fmt.Print(opts.Usage)
		return 0
	case opts.Version:
		fmt.Println("hexfetch " + appVersion())
		return 0
	case opts.ListModules:
		for _, name := range modules.Names() {
			if !opts.Verbose {
				fmt.Println(name)
				continue
			}
			meta := modules.Describe(name)
			flags := []string{}
			if meta.LinuxOnly {
				flags = append(flags, "linux")
			}
			if meta.External {
				flags = append(flags, "external")
			}
			if meta.Network {
				flags = append(flags, "network")
			}
			fmt.Printf("%-14s %-9s %s", name, meta.Category, meta.Description)
			if len(flags) > 0 {
				fmt.Printf(" [%s]", strings.Join(flags, ","))
			}
			fmt.Println()
		}
		return 0
	case opts.ListLogos:
		for _, name := range logo.Names() {
			fmt.Println(name)
		}
		return 0
	case opts.GenConfig:
		return generateConfig(opts)
	case opts.MigrateConfig:
		return migrateConfig(opts)
	}

	configPath := ""
	if opts.Config != nil {
		configPath = *opts.Config
	}
	diagnostics := &diag.Collector{}
	cfg, configDiagnostics := config.Load(configPath)
	for _, diagnostic := range configDiagnostics {
		diagnostics.Add(diagnostic)
	}
	applyOverrides(&cfg, opts, diagnostics)

	debug := opts.Debug || envEnabled("HEXFETCH_DEBUG") || envEnabled("NIXFETCH_DEBUG")
	color := colorEnabled(opts.NoColor)
	logoImage := loadLogo(cfg, color, diagnostics)
	position := "left"
	padding := 2
	if opts.LogoPosition != nil {
		position = *opts.LogoPosition
	}
	if opts.Padding != nil {
		padding = *opts.Padding
	}
	specs := buildSpecs(cfg, opts.Structure, color, debug, diagnostics)
	entries, moduleDiagnostics := modules.RunAll(specs)
	for _, diagnostic := range moduleDiagnostics {
		diagnostics.Add(diagnostic)
	}
	renderOptions := render.Options{
		Logo:            logoImage,
		Position:        position,
		Padding:         padding,
		Separator:       cfg.Display.Separator,
		SeparatorLength: cfg.Display.SeparatorLength,
		KeyType:         cfg.Display.KeyType,
		KeyColor:        string(cfg.Display.Color.Keys),
		TitleColor:      string(cfg.Display.Color.Title),
		OutputColor:     string(cfg.Display.Color.Output),
		SeparatorColor:  string(cfg.Display.Color.Separator),
		Color:           color,
		Width:           render.TermWidth(),
	}
	if len(renderOptions.Logo.Lines) > 0 && renderOptions.Position != "right" &&
		!render.LogoFits(logoImage, renderOptions.Padding, renderOptions.Width) {
		diagnostics.Warnf("logo", "Logo is %d columns wide and does not fit the %d-column terminal; it is hidden.", logoImage.Width, renderOptions.Width)
	}
	diag.Print(os.Stderr, diagnostics.All(), diagnosticColorEnabled(opts.NoColor), debug, opts.Quiet, opts.ShowErrors)
	if err := render.Render(os.Stdout, entries, renderOptions); err != nil {
		fmt.Fprintf(os.Stderr, "hexfetch: %v\n", err)
		return 1
	}
	return 0
}

func generateConfig(opts cli.Options) int {
	path := config.DefaultPath()
	if opts.Config != nil {
		path = *opts.Config
	}
	if err := config.Generate(path); err != nil {
		fmt.Fprintf(os.Stderr, "hexfetch: %v\n", err)
		return 1
	}
	fmt.Println("Config written to " + path)
	return 0
}

func migrateConfig(opts cli.Options) int {
	destination := config.DefaultPath()
	if opts.Config != nil {
		destination = *opts.Config
	}
	diagnostics, err := config.MigrateLegacyTOML(config.LegacyTOMLPath(), destination)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hexfetch: %v\n", err)
		return 1
	}
	diag.Print(os.Stderr, diagnostics, false, opts.Debug, opts.Quiet, opts.ShowErrors)
	fmt.Println("Legacy TOML config migrated to " + destination)
	return 0
}

func applyOverrides(cfg *config.Config, opts cli.Options, diagnostics *diag.Collector) {
	if opts.Logo != nil {
		cfg.Logo = *opts.Logo
	}
	if opts.LogoColors != nil {
		colors, colorDiagnostics := config.OverrideLogoColors(cfg.LogoColors, opts.LogoColors)
		cfg.LogoColors = colors
		for _, diagnostic := range colorDiagnostics {
			diagnostics.Add(diagnostic)
		}
	}
	if opts.Separator != nil {
		cfg.Display.Separator = *opts.Separator
	}
	if opts.KeyType != nil {
		cfg.Display.KeyType = *opts.KeyType
	}
	if opts.ColorKeys != nil {
		cfg.Display.Color.Keys = config.Palette(*opts.ColorKeys)
	}
}

// loadLogo resolves cfg.Logo ("none", "auto", a name or an alias) and paints it.
// Unknown names and failed detection fall back to the default logo, hexos.
func loadLogo(cfg config.Config, color bool, diagnostics *diag.Collector) logo.Logo {
	source := strings.TrimSpace(cfg.Logo)
	if source == "" {
		source = logo.DefaultName
	}
	if strings.EqualFold(source, "none") {
		return logo.Logo{}
	}
	names := logo.Names()
	aliases := logo.Aliases()
	name := ""
	if strings.EqualFold(source, "auto") {
		release, err := modules.ReadOSRelease()
		if err != nil && !errors.Is(err, errors.ErrUnsupported) {
			diagnostics.Add(diag.Diagnostic{
				Level:   diag.Warning,
				Source:  "logo",
				Message: "OS release information could not be read for logo selection.",
				Detail:  err.Error(),
			})
		}
		name = logo.Detect(names, aliases, release)
		if name == "" {
			diagnostics.Warnf("logo", "No suitable logo found for this system; using %s.", logo.DefaultName)
		}
	} else {
		var found bool
		name, found = logo.Resolve(names, aliases, source)
		if !found {
			diagnostics.Warnf("logo", "Logo %q not found (see --list-logos); using %s.", source, logo.DefaultName)
		}
	}
	if name == "" {
		name = logo.DefaultName
	}
	if !logo.HasProfile(name) && cfg.LogoColors != logo.DefaultColors() {
		diagnostics.Infof("logo", "logo_colors only apply to the %s logo; %q uses its own colors.", logo.DefaultName, name)
	}
	loaded, err := logo.Load(name, cfg.LogoColors, color, gradient.TrueColor())
	if err != nil {
		diagnostics.Add(diag.Diagnostic{
			Level:   diag.Warning,
			Source:  "logo",
			Message: fmt.Sprintf("Logo %q could not be loaded.", name),
			Detail:  err.Error(),
		})
		return logo.Logo{}
	}
	return loaded
}

func colorEnabled(noColor bool) bool {
	if noColorRequested(noColor) {
		return false
	}
	info, err := os.Stdout.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func diagnosticColorEnabled(noColor bool) bool {
	if noColorRequested(noColor) {
		return false
	}
	info, err := os.Stderr.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func noColorRequested(noColor bool) bool {
	return noColor || os.Getenv("NO_COLOR") != ""
}

func envEnabled(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value != "" && value != "0" && value != "false" && value != "no"
}

func buildSpecs(cfg config.Config, structure *string, color, debug bool, diagnostics *diag.Collector) []modules.Spec {
	var wanted []config.ModuleSpec
	var specDiagnostics []diag.Diagnostic
	if structure != nil {
		wanted, specDiagnostics = cfg.SpecsFor(strings.Split(*structure, ":"))
	} else {
		wanted, specDiagnostics = cfg.Specs()
	}
	for _, diagnostic := range specDiagnostics {
		diagnostics.Add(diagnostic)
	}
	var specs []modules.Spec
	for _, item := range wanted {
		if !modules.Exists(item.Type) {
			diagnostics.Warnf(item.Type, "Unknown module %q.", item.Type)
			continue
		}
		text, command := item.Text, item.Command
		format := item.Format
		if item.Type == "title" || item.Type == "custom" {
			format = ""
		}
		if text == "" {
			text = item.Format
		}
		if command == "" && item.Type == "command" {
			command = item.Text
		}
		var keyColor *string
		if item.KeyColor != nil {
			value := string(*item.KeyColor)
			keyColor = &value
		}
		options := item.Options
		specs = append(specs, modules.Spec{
			Name: item.Type,
			Context: modules.Context{
				Key:           item.Key,
				KeyIcon:       item.KeyIcon,
				Text:          text,
				Format:        format,
				Command:       command,
				KeyColor:      keyColor,
				SeparatorChar: cfg.Display.SeparatorChar,
				Color:         color,
				Debug:         debug,
				Options:       options,
			},
		})
	}
	return specs
}
