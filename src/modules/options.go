package modules

type OptionKind uint8

const (
	OptionString OptionKind = iota
	OptionInteger
	OptionStringList
)

type OptionDefinition struct {
	Kind        OptionKind
	Minimum     int
	Maximum     int
	HasRange    bool
	Allowed     []string
	URL         bool
	AllowScalar bool
}

type configFieldProvider interface {
	ConfigFields() []string
}

type optionSchemaProvider interface {
	OptionSchema() map[string]OptionDefinition
}

func OptionSchema(name string) map[string]OptionDefinition {
	module, exists := registry[name]
	if !exists {
		return nil
	}
	provider, ok := module.(optionSchemaProvider)
	if !ok {
		return nil
	}
	return provider.OptionSchema()
}

func ConfigFields(name string) []string {
	module, exists := registry[name]
	if !exists {
		return nil
	}
	provider, ok := module.(configFieldProvider)
	if ok {
		return provider.ConfigFields()
	}
	return []string{"key", "key_icon", "key_color", "format"}
}
