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

func TestRemovedLogoSelectionFlagsAreRejected(t *testing.T) {
	for _, args := range [][]string{
		{"--logo-dir", "./logos"},
		{"--logo-direction", "diagonal"},
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
	for _, value := range []string{"YAML config file", "--logo-position", "--padding", "--list-modules", "--migrate-config"} {
		if !strings.Contains(options.Usage, value) {
			t.Errorf("help output does not contain %q", value)
		}
	}
	for _, value := range []string{"--logo-dir", "--logo-direction"} {
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

func TestParseLogoFlag(t *testing.T) {
	for _, args := range [][]string{
		{"--logo", "arch"},
		{"--logo=arch"},
		{"-l", "arch"},
		{"-l=arch"},
		{"-larch"},
	} {
		options, err := Parse(args)
		if err != nil {
			t.Errorf("Parse(%q) failed: %v", args, err)
			continue
		}
		if options.Logo == nil || *options.Logo != "arch" {
			t.Errorf("Parse(%q): Logo = %v, want arch", args, options.Logo)
		}
	}
	if options, err := Parse(nil); err != nil || options.Logo != nil || options.ListLogos {
		t.Fatalf("defaults changed: %+v, %v", options, err)
	}
}

func TestParseListLogos(t *testing.T) {
	options, err := Parse([]string{"--list-logos"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.ListLogos {
		t.Fatal("--list-logos did not enable ListLogos")
	}
	if !strings.Contains(options.Usage, "--list-logos") {
		t.Fatal("help output does not contain --list-logos")
	}
}

func TestParseLicenses(t *testing.T) {
	options, err := Parse([]string{"--licenses"})
	if err != nil {
		t.Fatal(err)
	}
	if !options.Licenses {
		t.Fatal("--licenses did not enable Licenses")
	}
}
