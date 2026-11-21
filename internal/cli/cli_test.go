package cli

import (
	"strings"
	"testing"
)

func TestParseQuietAndDebug(t *testing.T) {
	options, err := Parse([]string{"--quiet", "--debug"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.Quiet || !options.Debug {
		t.Fatalf("options = %+v, want Quiet and Debug enabled", options)
	}
	if !strings.Contains(options.Usage, "--quiet") {
		t.Fatalf("usage does not include --quiet: %q", options.Usage)
	}
}

func TestParseLogoColors(t *testing.T) {
	values := "38;2;1;2;3, 38;2;4;5;6,38;2;7;8;9"
	options, err := Parse([]string{"--logo-colors", values, "--logo-position", "right", "--padding", "5"})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.LogoColors) != 3 || options.LogoColors[0] != "38;2;1;2;3" || options.LogoColors[1] != "38;2;4;5;6" {
		t.Fatalf("LogoColors = %q", options.LogoColors)
	}
	if options.LogoPosition == nil || *options.LogoPosition != "right" {
		t.Fatalf("LogoPosition = %v, want right", options.LogoPosition)
	}
	if options.Padding == nil || *options.Padding != 5 {
		t.Fatalf("Padding = %v, want 5", options.Padding)
	}
}

func TestRemovedLogoSelectionFlagsAreRejected(t *testing.T) {
	for _, args := range [][]string{
		{"--logo", "arch"},
		{"-l", "arch"},
		{"--logo-dir", "./logos"},
		{"--logo-gradient", "38;2;1;2;3,38;2;4;5;6"},
		{"--logo-direction", "diagonal"},
		{"--list-logos"},
	} {
		if _, err := Parse(args); err == nil {
			t.Errorf("Parse(%q) succeeded, want removed flag error", args)
		}
	}
}

func TestParsePositionAndPaddingValidation(t *testing.T) {
	for _, args := range [][]string{
		{"--logo-position", "center"},
		{"--padding", "-1"},
	} {
		if _, err := Parse(args); err == nil {
			t.Errorf("Parse(%q) succeeded, want validation error", args)
		}
	}
}

func TestParseColorKeysRange(t *testing.T) {
	for _, value := range []string{"-1", "256", "999"} {
		if _, err := Parse([]string{"--color-keys", value}); err == nil {
			t.Errorf("Parse(--color-keys %s) succeeded, want range error", value)
		}
	}
	for _, value := range []string{"0", "4", "255"} {
		if _, err := Parse([]string{"--color-keys", value}); err != nil {
			t.Errorf("Parse(--color-keys %s) failed: %v", value, err)
		}
	}
}

func TestHelpUsesYAMLAndCurrentFlags(t *testing.T) {
	options, err := Parse([]string{"--help"})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"YAML config file", "--logo-colors", "--logo-position", "--padding", "--list-modules", "--migrate-config"} {
		if !strings.Contains(options.Usage, value) {
			t.Errorf("help output does not contain %q", value)
		}
	}
	for _, value := range []string{"--logo-dir", "--logo-gradient", "--logo-direction", "--list-logos"} {
		if strings.Contains(options.Usage, value) {
			t.Errorf("help output still contains removed flag %q", value)
		}
	}
}

func TestMigrateConfigFlag(t *testing.T) {
	options, err := Parse([]string{"--migrate-config", "--config", "./migrated.yaml"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.MigrateConfig || options.Config == nil || *options.Config != "./migrated.yaml" {
		t.Fatalf("migration options = %+v", options)
	}
	if _, err := Parse([]string{"--migrate-config", "--gen-config"}); err == nil {
		t.Fatal("conflicting config commands were accepted")
	}
}

func TestConfigPathMustNotBeEmpty(t *testing.T) {
	if _, err := Parse([]string{"--config", ""}); err == nil {
		t.Fatal("empty config path was accepted")
	}
}

func TestParseKeyType(t *testing.T) {
	options, err := Parse([]string{"--key-type", "both"})
	if err != nil {
		t.Fatal(err)
	}
	if options.KeyType == nil || *options.KeyType != "both" {
		t.Fatalf("KeyType = %v, want both", options.KeyType)
	}
	if _, err := Parse([]string{"--key-type", "wide"}); err == nil {
		t.Fatal("invalid key type was accepted")
	}
}
