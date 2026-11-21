package modules

type separator struct{}

func init() { Register(separator{}) }

func (separator) Name() string { return "separator" }

func (separator) ConfigFields() []string {
	return nil
}

func (separator) Run(ctx Context) ([]Entry, error) {
	return []Entry{{Kind: KindSeparator, Value: ctx.SeparatorChar}}, nil
}
