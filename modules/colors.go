package modules

type colors struct{}

func init() { Register(colors{}) }

func (colors) Name() string { return "colors" }

func (colors) ConfigFields() []string {
	return nil
}

// Run отдаёт только маркер палитры: ширину ячеек и перенос строк
// выбирает render, потому что только он знает свободное место в терминале.
func (colors) Run(ctx Context) ([]Entry, error) {
	if !ctx.Color {
		return nil, nil
	}
	return []Entry{{Kind: KindPalette}}, nil
}
