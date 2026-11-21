package modules

import (
	"slices"
	"strings"
	"testing"

	"hexfetch/internal/diag"
)

func TestApplyFormat(t *testing.T) {
	e := Entry{Key: "Memory", Value: "1.0 GiB / 2.0 GiB (50%)", Fields: map[string]string{"used": "1.0 GiB", "percent": "50%"}}
	cases := []struct{ format, want string }{
		{"{}", "1.0 GiB / 2.0 GiB (50%)"},
		{"{value}", "1.0 GiB / 2.0 GiB (50%)"},
		{"{key}: {used} ({percent})", "Memory: 1.0 GiB (50%)"},
		{"{ used }", "1.0 GiB"},
		{"{{used}} = {used}", "{used} = 1.0 GiB"},
		{"100}% {percent}", "100}% 50%"},
		{"plain text", "plain text"},
	}
	for _, c := range cases {
		got, problems := applyFormat(c.format, e)
		if got != c.want || len(problems) != 0 {
			t.Errorf("applyFormat(%q) = %q, %v; want %q", c.format, got, problems, c.want)
		}
	}
	for _, bad := range []string{"{nope}", "{used", "{used} {x}"} {
		if _, problems := applyFormat(bad, e); len(problems) == 0 {
			t.Errorf("applyFormat(%q): expected a problem", bad)
		}
	}
}

type formatProbe struct{}

func (formatProbe) Name() string { return "format_probe" }
func (formatProbe) Run(ctx Context) ([]Entry, error) {
	return []Entry{
		{Key: "K", Value: "v", Fields: map[string]string{"a": "1"}},
		{Kind: KindBreak},
	}, nil
}

func TestRunAllFormatReportsInvalidTemplates(t *testing.T) {
	Register(formatProbe{})
	entries, diagnostics := RunAll([]Spec{{Name: "format_probe", Context: Context{Format: "{unknown}"}}})
	if len(entries) == 0 || entries[0].Value != "v" {
		t.Fatalf("invalid format changed the original value: %+v", entries)
	}
	if len(diagnostics) != 1 || diagnostics[0].Level != diag.Warning {
		t.Fatalf("invalid format diagnostics = %+v, want one warning", diagnostics)
	}
	if diagnostics[0].Source != "format_probe" || !strings.Contains(diagnostics[0].Detail, "available: value, key, a") {
		t.Fatalf("invalid format diagnostic lacks module context: %+v", diagnostics[0])
	}
}

func TestRunAllFormat(t *testing.T) {
	Register(formatProbe{})
	run := func(format string) []Entry {
		entries, diagnostics := RunAll([]Spec{{Name: "format_probe", Context: Context{Format: format}}})
		if len(diagnostics) != 0 {
			t.Fatalf("RunAll returned unexpected diagnostics: %v", diagnostics)
		}
		return entries
	}
	if got := run("{key}={a}"); got[0].Value != "K=1" {
		t.Errorf("formatted value = %q", got[0].Value)
	}
	if got := run(""); got[0].Value != "v" {
		t.Errorf("no format: value = %q", got[0].Value)
	}
	if got := run("x"); !slices.ContainsFunc(got, func(e Entry) bool { return e.Kind == KindBreak }) {
		t.Error("non-normal entries must pass through untouched")
	}
}
