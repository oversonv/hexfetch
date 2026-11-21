package modules

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"hexfetch/src/internal/diag"
)

type unsupportedProbe struct{}

func (unsupportedProbe) Name() string { return "unsupported_probe" }
func (unsupportedProbe) Run(Context) ([]Entry, error) {
	return nil, Unsupported("no supported hardware")
}

type panicProbe struct{}

func (panicProbe) Name() string { return "panic_probe" }
func (panicProbe) Run(Context) ([]Entry, error) {
	panic("probe failure")
}

type errorProbe struct{}

func (errorProbe) Name() string { return "error_probe" }
func (errorProbe) Run(Context) ([]Entry, error) {
	return nil, errors.New("unexpected failure")
}

func TestUnsupportedWrapsErrUnsupported(t *testing.T) {
	err := Unsupported("feature %s unavailable", "probe")
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Unsupported() error = %v, want errors.ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "feature probe unavailable") {
		t.Fatalf("Unsupported() error = %q", err)
	}
}

func TestRunAllUnsupportedIsInfo(t *testing.T) {
	Register(unsupportedProbe{})
	entries, diagnostics := RunAll([]Spec{{Name: "unsupported_probe"}})
	if len(entries) != 0 {
		t.Fatalf("entries = %v, want none", entries)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %v, want one", diagnostics)
	}
	if diagnostics[0].Level != diag.Info {
		t.Fatalf("diagnostic level = %v, want Info", diagnostics[0].Level)
	}
	if diagnostics[0].Source != "unsupported_probe" {
		t.Fatalf("diagnostic source = %q", diagnostics[0].Source)
	}
	var output bytes.Buffer
	diag.Print(&output, diagnostics, false, false, false)
	if output.Len() != 0 {
		t.Fatalf("unsupported module produced default output: %q", output.String())
	}
}

func TestRunAllPanicIncludesStack(t *testing.T) {
	Register(panicProbe{})
	entries, diagnostics := RunAll([]Spec{{Name: "panic_probe"}})
	if len(entries) != 0 {
		t.Fatalf("entries = %v, want none", entries)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %v, want one", diagnostics)
	}
	diagnostic := diagnostics[0]
	if diagnostic.Level != diag.Error {
		t.Fatalf("diagnostic level = %v, want Error", diagnostic.Level)
	}
	if diagnostic.Message != "The panic_probe module crashed." {
		t.Fatalf("diagnostic message = %q", diagnostic.Message)
	}
	for _, part := range []string{"panic: probe failure", "goroutine", "module_test.go"} {
		if !strings.Contains(diagnostic.Detail, part) {
			t.Errorf("panic detail %q does not contain %q", diagnostic.Detail, part)
		}
	}
}

func TestRunAllUnexpectedErrorIsError(t *testing.T) {
	Register(errorProbe{})
	_, diagnostics := RunAll([]Spec{{Name: "error_probe"}})
	if len(diagnostics) != 1 || diagnostics[0].Level != diag.Error {
		t.Fatalf("diagnostics = %v, want one Error", diagnostics)
	}
	if diagnostics[0].Detail != "unexpected failure" {
		t.Errorf("detail = %q", diagnostics[0].Detail)
	}
}

func TestRunAllUnknownModuleIsWarning(t *testing.T) {
	_, diagnostics := RunAll([]Spec{{Name: "missing_probe"}})
	if len(diagnostics) != 1 || diagnostics[0].Level != diag.Warning {
		t.Fatalf("diagnostics = %v, want one Warning", diagnostics)
	}
}
