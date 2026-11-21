package modules

import "testing"

func TestModuleConfigurationSchemas(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
		option string
	}{
		{name: "title", fields: []string{"text", "format"}},
		{name: "command", fields: []string{"key", "key_icon", "text", "command", "key_color", "format"}},
		{name: "custom", fields: []string{"key", "key_icon", "text", "key_color"}},
		{name: "separator", fields: nil},
		{name: "break", fields: nil},
		{name: "colors", fields: nil},
		{name: "shell", fields: []string{"key", "key_icon", "key_color", "format"}},
	}
	for _, test := range tests {
		if got := ConfigFields(test.name); len(got) != len(test.fields) {
			t.Errorf("ConfigFields(%q) = %v, want %v", test.name, got, test.fields)
		} else {
			for index := range got {
				if got[index] != test.fields[index] {
					t.Errorf("ConfigFields(%q) = %v, want %v", test.name, got, test.fields)
					break
				}
			}
		}
	}
	for _, test := range []struct {
		name string
		key  string
	}{
		{name: "cpuusage", key: "interval_ms"},
		{name: "localip", key: "family"},
		{name: "publicip", key: "url"},
		{name: "weather", key: "url"},
	} {
		if _, exists := OptionSchema(test.name)[test.key]; !exists {
			t.Errorf("OptionSchema(%q) does not define %q", test.name, test.key)
		}
	}
	if ConfigFields("missing-module") != nil || OptionSchema("missing-module") != nil {
		t.Fatal("unknown module returned a configuration schema")
	}
}
