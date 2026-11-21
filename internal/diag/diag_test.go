package diag

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintLevelsWithoutColor(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Message: "failure"},
		{Level: Warning, Message: "caution"},
		{Level: Info, Message: "skipped"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, false, true, false)
	want := "Error: failure\nWarning: caution\nInfo: skipped\n"
	if output.String() != want {
		t.Fatalf("Print() = %q, want %q", output.String(), want)
	}
}

func TestPrintLevelsWithColor(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Message: "failure"},
		{Level: Warning, Message: "caution"},
		{Level: Info, Message: "skipped"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, true, true, false)
	want := "\x1b[31mError:\x1b[0m failure\n" +
		"\x1b[33mWarning:\x1b[0m caution\n" +
		"\x1b[2mInfo:\x1b[0m skipped\n"
	if output.String() != want {
		t.Fatalf("Print() = %q, want %q", output.String(), want)
	}
}

func TestPrintDebugDetails(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Source: "uptime", Message: "The uptime module crashed.", Detail: "open /proc/uptime: permission denied"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, false, false, false)
	if strings.Contains(output.String(), "permission denied") {
		t.Fatalf("details printed without debug: %q", output.String())
	}
	output.Reset()
	Print(&output, diagnostics, false, true, false)
	if !strings.Contains(output.String(), "  open /proc/uptime: permission denied\n") {
		t.Fatalf("debug detail missing: %q", output.String())
	}
}

func TestPrintHidesInfoWithoutDebug(t *testing.T) {
	diagnostics := []Diagnostic{{Level: Info, Message: "The battery module is not applicable."}}
	var output bytes.Buffer
	Print(&output, diagnostics, false, false, false)
	if output.Len() != 0 {
		t.Fatalf("Info printed without debug: %q", output.String())
	}
}

func TestPrintQuiet(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Message: "failure", Detail: "detail"},
		{Level: Warning, Message: "warning"},
		{Level: Info, Message: "info"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, true, true, true)
	if output.Len() != 0 {
		t.Fatalf("quiet output = %q", output.String())
	}
}

func TestCollector(t *testing.T) {
	var collector Collector
	collector.Errorf("uptime", "The %s module crashed.", "uptime")
	collector.Warnf("config", "Parsing error: %s", "colors")
	collector.Infof("battery", "The battery module is not applicable.")

	got := collector.All()
	if len(got) != 3 {
		t.Fatalf("All() returned %d diagnostics, want 3", len(got))
	}
	if got[0].Level != Error || got[0].Source != "uptime" || got[0].Message != "The uptime module crashed." {
		t.Errorf("unexpected first diagnostic: %+v", got[0])
	}
	got[0].Message = "mutated"
	if collector.All()[0].Message != "The uptime module crashed." {
		t.Error("All() exposed the collector's backing slice")
	}
}

func TestPrintShowsWarningsAndErrorsWithoutDebug(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Message: "module failed"},
		{Level: Warning, Message: "config invalid"},
		{Level: Info, Message: "module skipped"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, false, false, false)
	want := "Error: module failed\nWarning: config invalid\n"
	if output.String() != want {
		t.Fatalf("Print() = %q, want %q", output.String(), want)
	}
}

func TestPrintShowErrorsIncludesSkippedModules(t *testing.T) {
	diagnostics := []Diagnostic{
		{Level: Error, Message: "module failed"},
		{Level: Info, Message: "module skipped"},
	}
	var output bytes.Buffer
	Print(&output, diagnostics, false, false, false, true)
	want := "Error: module failed\nInfo: module skipped\n"
	if output.String() != want {
		t.Fatalf("Print() = %q, want %q", output.String(), want)
	}
}
