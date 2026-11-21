//go:build linux

package modules

type host struct{}

func init() { Register(host{}) }

func (host) Name() string { return "host" }

func (host) Run(ctx Context) ([]Entry, error) {
	name := dmiValue("product_name")
	version := dmiValue("product_version")
	if name == "" {
		return nil, Unsupported("host model unavailable")
	}
	if version != "" && version != name {
		name += " (" + version + ")"
	}
	return Single(ctx, "Host", name), nil
}
