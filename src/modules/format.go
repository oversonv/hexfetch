package modules

import (
	"sort"
	"strings"
)

// applyFormat подставляет в шаблон значения записи.
// {} и {value} - итоговое значение модуля, {key} - название, {имя} - поле
// из Entry.Fields. {{ и }} дают литеральные скобки.
// Второй результат - список проблем; при любой проблеме строка не годится.
func applyFormat(format string, e Entry) (string, []string) {
	var out strings.Builder
	var problems []string
	for i := 0; i < len(format); i++ {
		c := format[i]
		switch {
		case c == '{' && i+1 < len(format) && format[i+1] == '{':
			out.WriteByte('{')
			i++
		case c == '}' && i+1 < len(format) && format[i+1] == '}':
			out.WriteByte('}')
			i++
		case c == '{':
			end := strings.IndexByte(format[i:], '}')
			if end < 0 {
				return "", append(problems, "unterminated { in format")
			}
			name := strings.TrimSpace(format[i+1 : i+end])
			i += end
			switch name {
			case "", "value":
				out.WriteString(e.Value)
			case "key":
				out.WriteString(e.Key)
			default:
				if v, ok := e.Fields[name]; ok {
					out.WriteString(v)
				} else {
					problems = append(problems, "unknown placeholder {"+name+"}")
				}
			}
		default:
			out.WriteByte(c)
		}
	}
	return out.String(), problems
}

func formatEntry(format string, e Entry) (string, []string) {
	result, problems := applyFormat(format, e)
	if len(problems) > 0 {
		return e.Value, problems
	}
	return result, nil
}

func formatAvailableFields(e Entry) []string {
	available := []string{"value", "key"}
	var fields []string
	for name := range e.Fields {
		fields = append(fields, name)
	}
	sort.Strings(fields)
	return append(available, fields...)
}
