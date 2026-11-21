package modules

type lineBreak struct{}

func init() { Register(lineBreak{}) }

func (lineBreak) Name() string { return "break" }

func (lineBreak) ConfigFields() []string {
	return nil
}

func (lineBreak) Run(ctx Context) ([]Entry, error) {
	return []Entry{{Kind: KindBreak}}, nil
}
