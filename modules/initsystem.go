package modules

import (
	"os"
)

type initsystem struct{}

func init() { Register(initsystem{}) }

func (initsystem) Name() string { return "initsystem" }

func (initsystem) Run(ctx Context) ([]Entry, error) {
	comm, _ := ReadTrim("/proc/1/comm")
	switch comm {
	case "systemd", "dinit", "sinit":
		return Single(ctx, "Init", comm), nil
	case "runit", "runit-init":
		return Single(ctx, "Init", "runit"), nil
	case "openrc-init":
		return Single(ctx, "Init", "OpenRC"), nil
	case "s6-svscan":
		return Single(ctx, "Init", "s6"), nil
	}
	checks := []struct{ path, name string }{
		{"/run/systemd/system", "systemd"},
		{"/run/openrc", "OpenRC"},
		{"/etc/runit", "runit"},
	}
	for _, check := range checks {
		if _, err := os.Stat(check.path); err == nil {
			return Single(ctx, "Init", check.name), nil
		}
	}
	if comm != "" {
		return Single(ctx, "Init", comm), nil
	}
	return nil, Unsupported("init system unknown")
}
