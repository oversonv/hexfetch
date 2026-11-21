package modules

type custom struct{}

func init() { Register(custom{}) }

func (custom) Name() string { return "custom" }

func (custom) ConfigFields() []string {
	return []string{"key", "key_icon", "text", "key_color"}
}

func (custom) Run(ctx Context) ([]Entry, error) {
	if ctx.Text == "" {
		return nil, Unsupported("custom text is empty")
	}
	if ctx.Key == "" {
		return []Entry{{Kind: KindRaw, Value: ctx.Text}}, nil
	}
	return []Entry{{Key: ctx.Key, Value: ctx.Text}}, nil
}
