package modules

type kernel struct{}

func init() { Register(kernel{}) }

func (kernel) Name() string { return "kernel" }

func (kernel) Run(ctx Context) ([]Entry, error) {
	display := kernelDisplay()
	if display == "" {
		return nil, Unsupported("kernel release unavailable")
	}
	return Single(ctx, "Kernel", display), nil
}
