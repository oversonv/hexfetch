package modules

import (
	"errors"
	"runtime"
	"strings"
)

type osModule struct{}

func init() { Register(osModule{}) }

func (osModule) Name() string { return "os" }

func (osModule) Run(ctx Context) ([]Entry, error) {
	release, err := ReadOSRelease()
	if err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return nil, err
	}
	name := release["PRETTY_NAME"]
	if name == "" {
		name = strings.TrimSpace(release["NAME"] + " " + release["VERSION"])
	}
	if name == "" {
		name = "Linux"
	}
	return Single(ctx, "OS", name+" "+arch()), nil
}

func arch() string {
	names := map[string]string{"amd64": "x86_64", "arm64": "aarch64", "386": "i686", "arm": "armv7l"}
	if name, ok := names[runtime.GOARCH]; ok {
		return name
	}
	return runtime.GOARCH
}
