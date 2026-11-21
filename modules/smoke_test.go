package modules

import (
	"errors"
	"strings"
	"testing"
)

// TestSmokeAllModulesAgainstTestdata прогоняет каждый зарегистрированный
// модуль на синтетических данных (testdata/proc, testdata/sys) и проверяет
// два инварианта: модуль не паникует и любая ошибка является Unsupported
// (мягкий пропуск), а не неожиданным сбоем.
func TestSmokeAllModulesAgainstTestdata(t *testing.T) {
	oldProc, oldSys := procRoot, sysRoot
	procRoot, sysRoot = "testdata/proc", "testdata/sys"
	defer func() { procRoot, sysRoot = oldProc, oldSys }()

	for _, name := range Names() {
		name := name
		if strings.HasSuffix(name, "_probe") {
			// Служебные модули-зонды из других тестов — не цель смоука.
			continue
		}
		t.Run(name, func(t *testing.T) {
			module, ok := registry[name]
			if !ok {
				t.Fatalf("module %q not registered", name)
			}
			ctx := Context{Debug: false}
			switch name {
			case "publicip", "weather":
				// Не ходить в сеть: заведомо недоступный локальный адрес.
				ctx.Options = map[string]any{"url": "http://127.0.0.1:9/"}
			case "command":
				ctx.Command = "true"
			}
			entries, err := safeRun(module, ctx)
			if err != nil {
				if strings.HasPrefix(err.Error(), "panic:") {
					t.Fatalf("panic: %v", err)
				}
				if !errors.Is(err, errors.ErrUnsupported) {
					t.Fatalf("unexpected error (want Unsupported): %v", err)
				}
				return
			}
			for _, entry := range entries {
				if entry.Value == "" && entry.Kind == 0 {
					t.Fatalf("empty entry for module %q: %+v", name, entry)
				}
			}
		})
	}
}
