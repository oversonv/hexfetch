package diag

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

type Level int

const (
	Info Level = iota
	Warning
	Error
)

type Diagnostic struct {
	Level   Level
	Source  string
	Message string
	Detail  string
}

type Collector struct {
	mu          sync.Mutex
	diagnostics []Diagnostic
}

func (c *Collector) Add(diagnostic Diagnostic) {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.diagnostics = append(c.diagnostics, diagnostic)
	c.mu.Unlock()
}

func (c *Collector) Errorf(source, format string, args ...any) {
	c.Add(Diagnostic{
		Level:   Error,
		Source:  source,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *Collector) Warnf(source, format string, args ...any) {
	c.Add(Diagnostic{
		Level:   Warning,
		Source:  source,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *Collector) Infof(source, format string, args ...any) {
	c.Add(Diagnostic{
		Level:   Info,
		Source:  source,
		Message: fmt.Sprintf(format, args...),
	})
}

func (c *Collector) All() []Diagnostic {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Diagnostic(nil), c.diagnostics...)
}

func Print(w io.Writer, diagnostics []Diagnostic, color, debug, quiet bool, showErrors ...bool) {
	if quiet || w == nil {
		return
	}
	showInfo := debug || (len(showErrors) > 0 && showErrors[0])
	for _, diagnostic := range diagnostics {
		if diagnostic.Level == Info && !showInfo {
			continue
		}
		label, code := levelStyle(diagnostic.Level)
		if color {
			fmt.Fprintf(w, "\x1b[%sm%s:\x1b[0m %s\n", code, label, diagnostic.Message)
		} else {
			fmt.Fprintf(w, "%s: %s\n", label, diagnostic.Message)
		}
		if debug && diagnostic.Detail != "" {
			for _, line := range strings.Split(strings.TrimSuffix(diagnostic.Detail, "\n"), "\n") {
				fmt.Fprintf(w, "  %s\n", line)
			}
		}
	}
}

func levelStyle(level Level) (string, string) {
	switch level {
	case Error:
		return "Error", "31"
	case Warning:
		return "Warning", "33"
	default:
		return "Info", "2"
	}
}
