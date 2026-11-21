package modules

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"

	"hexfetch/internal/diag"
)

const (
	KindNormal = iota
	KindTitle
	KindSeparator
	KindRaw
	KindBreak
	KindPalette
)

type Entry struct {
	Kind     int
	Key      string
	KeyIcon  string
	Value    string
	KeyColor *string
	// Fields: именованные значения для шаблонов format (необязательно)
	Fields map[string]string
}

type Context struct {
	Key           string
	KeyIcon       string
	Text          string
	Command       string
	KeyColor      *string
	SeparatorChar string
	Color         bool
	Debug         bool
	Options       map[string]any
	// Format: шаблон из конфига, применяется к обычным записям модуля
	Format string
}

type Module interface {
	Name() string
	Run(ctx Context) ([]Entry, error)
}

type Spec struct {
	Name    string
	Context Context
}

var registry = map[string]Module{}

func Register(m Module) {
	registry[m.Name()] = m
}

func Exists(name string) bool {
	_, ok := registry[name]
	return ok
}

func Names() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c Context) KeyOr(fallback string) string {
	if c.Key != "" {
		return c.Key
	}
	return fallback
}

func (c Context) OptionString(key, fallback string) string {
	value, ok := c.Options[key].(string)
	if !ok || value == "" {
		return fallback
	}
	return value
}

func (c Context) OptionInt(key string, fallback int) int {
	switch value := c.Options[key].(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		converted := int(value)
		if int64(converted) == value {
			return converted
		}
	case uint:
		if uint64(value) <= uint64(^uint(0)>>1) {
			return int(value)
		}
	case uint8:
		return int(value)
	case uint16:
		return int(value)
	case uint32:
		if uint64(value) <= uint64(^uint(0)>>1) {
			return int(value)
		}
	case uint64:
		if value <= uint64(^uint(0)>>1) {
			return int(value)
		}
	case string:
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func (c Context) OptionBool(key string, fallback bool) bool {
	value, ok := c.Options[key].(bool)
	if !ok {
		return fallback
	}
	return value
}

func (c Context) OptionStrings(key string) []string {
	switch raw := c.Options[key].(type) {
	case []string:
		values := make([]string, 0, len(raw))
		for _, value := range raw {
			if value != "" {
				values = append(values, value)
			}
		}
		return values
	case []any:
		var values []string
		for _, item := range raw {
			if value, ok := item.(string); ok && value != "" {
				values = append(values, value)
			}
		}
		return values
	case string:
		if raw != "" {
			return []string{raw}
		}
	}
	return nil
}

func Single(ctx Context, fallbackKey, value string) []Entry {
	return []Entry{{Key: ctx.KeyOr(fallbackKey), Value: value}}
}

func ReadTrim(path string) (string, error) {
	return readTrimmed(path)
}

func ReadOSRelease() (map[string]string, error) {
	paths := []string{
		etcPath("os-release"),
		filepath.Join(filepath.Dir(etcRoot), "usr", "lib", "os-release"),
	}
	var failures []error
	allUnavailable := true
	for _, path := range paths {
		data, err := readFile(path)
		if err != nil {
			failures = append(failures, fmt.Errorf("%q: %w", path, errorWithoutUnsupported(err)))
			if !isUnavailableError(err) {
				allUnavailable = false
			}
			continue
		}
		values := map[string]string{}
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			key, value, ok := strings.Cut(scanner.Text(), "=")
			if !ok {
				continue
			}
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("parse %q: %w", path, err)
		}
		return values, nil
	}
	return nil, combinePathErrors("read os-release", failures, allUnavailable)
}

func Unsupported(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errors.ErrUnsupported, fmt.Sprintf(format, args...))
}

func safeRun(module Module, ctx Context) (entries []Entry, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			entries = nil
			err = fmt.Errorf("panic: %v\n%s", recovered, debug.Stack())
		}
	}()
	result, err := module.Run(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func RunAll(specs []Spec) ([]Entry, []diag.Diagnostic) {
	results := make([][]Entry, len(specs))
	errs := make([]error, len(specs))
	diagnostics := make([]diag.Diagnostic, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		module, ok := registry[spec.Name]
		if !ok {
			diagnostics[i] = diag.Diagnostic{
				Level:   diag.Warning,
				Source:  spec.Name,
				Message: fmt.Sprintf("Unknown module %q.", spec.Name),
			}
			continue
		}
		wg.Add(1)
		go func(index int, module Module, context Context) {
			defer wg.Done()
			results[index], errs[index] = safeRun(module, context)
		}(i, module, spec.Context)
	}
	wg.Wait()
	for i, err := range errs {
		if err == nil {
			continue
		}
		name := specs[i].Name
		if errors.Is(err, errors.ErrUnsupported) {
			diagnostics[i] = diag.Diagnostic{
				Level:   diag.Info,
				Source:  name,
				Message: fmt.Sprintf("The %s module is not applicable.", name),
				Detail:  err.Error(),
			}
			continue
		}
		diagnostics[i] = diag.Diagnostic{
			Level:   diag.Error,
			Source:  name,
			Message: fmt.Sprintf("The %s module crashed.", name),
			Detail:  err.Error(),
		}
	}
	var entries []Entry
	var formatDiagnostics []diag.Diagnostic
	reportedFormats := make(map[string]bool)
	for i, group := range results {
		for _, entry := range group {
			if specs[i].Context.Key != "" && entry.Kind == KindNormal {
				entry.Key = specs[i].Context.Key
			}
			if specs[i].Context.KeyIcon != "" && entry.Kind == KindNormal {
				entry.KeyIcon = specs[i].Context.KeyIcon
			}
			if specs[i].Context.KeyColor != nil {
				entry.KeyColor = specs[i].Context.KeyColor
			}
			if specs[i].Context.Format != "" && entry.Kind == KindNormal {
				formatted, problems := formatEntry(specs[i].Context.Format, entry)
				if len(problems) == 0 {
					entry.Value = formatted
				} else {
					identity := specs[i].Name + "\x00" + specs[i].Context.Format + "\x00" + strings.Join(problems, "\x00")
					if !reportedFormats[identity] {
						reportedFormats[identity] = true
						formatDiagnostics = append(formatDiagnostics, diag.Diagnostic{
							Level:   diag.Warning,
							Source:  specs[i].Name,
							Message: fmt.Sprintf("Invalid format for module %q; the original value was kept.", specs[i].Name),
							Detail:  strings.Join(problems, "; ") + " (available: " + strings.Join(formatAvailableFields(entry), ", ") + ")",
						})
					}
				}
			}
			entries = append(entries, entry)
		}
	}
	collected := make([]diag.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Message != "" {
			collected = append(collected, diagnostic)
		}
	}
	collected = append(collected, formatDiagnostics...)
	return entries, collected
}
