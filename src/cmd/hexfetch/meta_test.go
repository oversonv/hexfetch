package main

import (
	"regexp"
	"testing"
)

func TestEmbeddedMetaHasValidFields(t *testing.T) {
	meta, err := parseMeta(metaTOML)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(meta.Version) {
		t.Fatalf("version = %q, want X.Y.Z", meta.Version)
	}
	if meta.Name != "hexfetch" || meta.Repository == "" || meta.License == "" {
		t.Fatalf("incomplete metadata: %+v", meta)
	}
}

func TestParseMetaRejectsBrokenInput(t *testing.T) {
	for name, data := range map[string]string{
		"empty":           "",
		"no version":      "[package]\nname = \"hexfetch\"\n",
		"version in root": "version = \"1.2.3\"\n",
		"invalid toml":    "[package\nversion = ",
	} {
		if _, err := parseMeta([]byte(data)); err == nil {
			t.Errorf("%s: parseMeta() succeeded, want error", name)
		}
	}
}

func TestRunVersionPrintsEmbeddedVersion(t *testing.T) {
	meta, err := parseMeta(metaTOML)
	if err != nil {
		t.Fatal(err)
	}
	want := "hexfetch " + meta.Version + "\n"
	for _, flag := range []string{"--version", "-v"} {
		output := captureStdout(t, func() {
			if code := run([]string{flag}); code != 0 {
				t.Errorf("run(%s) = %d, want 0", flag, code)
			}
		})
		if output != want {
			t.Fatalf("run(%s) output = %q, want %q", flag, output, want)
		}
	}
}
