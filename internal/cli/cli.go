package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/pflag"
)

type Options struct {
	Logo          *string
	LogoPosition  *string
	Structure     *string
	Config        *string
	Padding       *int
	Separator     *string
	ColorKeys     *int
	KeyType       *string
	LogoColors    []string
	NoColor       bool
	GenConfig     bool
	MigrateConfig bool
	ListModules   bool
	ListLogos     bool
	Verbose       bool
	Debug         bool
	ShowErrors    bool
	Quiet         bool
	Help          bool
	Version       bool
	Usage         string
}

func Parse(args []string) (Options, error) {
	set := pflag.NewFlagSet("hexfetch", pflag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() {}

	var o Options
	logoName := set.StringP("logo", "l", "", "Logo name, auto or none (see --list-logos)")
	position := set.StringP("logo-position", "p", "left", "Logo side: left or right")
	structure := set.StringP("structure", "s", "", "Colon-separated list of modules to print")
	configPath := set.StringP("config", "c", "", "Path to a YAML config file or --migrate-config destination")
	padding := set.Int("padding", 2, "Spaces between the logo and the modules")
	separator := set.String("separator", " » ", "Separator between key and value")
	colorKeys := set.Int("color-keys", 4, "ANSI 256-color index for module keys (0-255)")
	keyType := set.String("key-type", "string", "Key display: string, icon or both")
	logoColors := set.String("logo-colors", "", "Comma-separated 38;2;R;G;B colors for the hexos logo: 12 (start and end of 6 bands), 2 (start and end) or 1 (flat)")
	set.BoolVar(&o.NoColor, "no-color", false, "Disable all colors")
	set.BoolVar(&o.GenConfig, "gen-config", false, "Write the default config file")
	set.BoolVar(&o.MigrateConfig, "migrate-config", false, "Convert the legacy TOML config to YAML without changing the original")
	set.BoolVar(&o.ListModules, "list-modules", false, "List available modules")
	set.BoolVar(&o.ListLogos, "list-logos", false, "List available logos")
	set.BoolVar(&o.Verbose, "verbose", false, "Show module categories and descriptions with --list-modules")
	set.BoolVar(&o.Debug, "debug", false, "Show diagnostic details and skipped modules")
	set.BoolVar(&o.ShowErrors, "show-errors", false, "Show skipped modules and their reasons")
	set.BoolVar(&o.Quiet, "quiet", false, "Hide diagnostics")
	set.BoolVarP(&o.Help, "help", "h", false, "Show this help")
	set.BoolVarP(&o.Version, "version", "v", false, "Show the version")

	if err := set.Parse(args); err != nil {
		return Options{}, err
	}
	if set.NArg() > 0 {
		return Options{}, fmt.Errorf("unexpected argument %q", set.Arg(0))
	}
	if o.GenConfig && o.MigrateConfig {
		return Options{}, fmt.Errorf("--gen-config and --migrate-config cannot be used together")
	}
	if set.Changed("config") && strings.TrimSpace(*configPath) == "" {
		return Options{}, fmt.Errorf("config path must not be empty")
	}
	if set.Changed("logo-position") && *position != "left" && *position != "right" {
		return Options{}, fmt.Errorf("invalid logo position %q: use left or right", *position)
	}
	if set.Changed("padding") && *padding < 0 {
		return Options{}, fmt.Errorf("invalid padding %d: must not be negative", *padding)
	}
	if set.Changed("color-keys") && (*colorKeys < 0 || *colorKeys > 255) {
		return Options{}, fmt.Errorf("invalid key color %d: expected 0-255", *colorKeys)
	}
	if set.Changed("key-type") && *keyType != "string" && *keyType != "icon" && *keyType != "both" {
		return Options{}, fmt.Errorf("invalid key type %q: use string, icon or both", *keyType)
	}

	if set.Changed("logo") {
		o.Logo = logoName
	}
	if set.Changed("logo-position") {
		o.LogoPosition = position
	}
	if set.Changed("structure") {
		o.Structure = structure
	}
	if set.Changed("config") {
		o.Config = configPath
	}
	if set.Changed("padding") {
		o.Padding = padding
	}
	if set.Changed("separator") {
		o.Separator = separator
	}
	if set.Changed("color-keys") {
		o.ColorKeys = colorKeys
	}
	if set.Changed("key-type") {
		o.KeyType = keyType
	}
	if set.Changed("logo-colors") {
		parts := strings.Split(*logoColors, ",")
		o.LogoColors = make([]string, len(parts))
		for index, part := range parts {
			o.LogoColors[index] = strings.TrimSpace(part)
		}
	}
	o.Usage = usage(set)
	return o, nil
}

func usage(set *pflag.FlagSet) string {
	return "Usage: hexfetch [options]\n\nOptions:\n" + set.FlagUsages() + `
Examples:
  hexfetch
  hexfetch --logo-colors "38;2;255;94;98,38;2;255;153;102"
  hexfetch --logo-position right --padding 5
  hexfetch -s "title:os:kernel:uptime"
  hexfetch --structure "title:separator:os:shell:packages:break:colors"
  hexfetch -c ./custom.yaml
  hexfetch --key-type both
  hexfetch --migrate-config
  hexfetch --list-modules
  hexfetch --list-logos
  hexfetch --list-modules --verbose
  hexfetch --debug -s "title:memory:cpu"
  hexfetch --quiet
`
}
