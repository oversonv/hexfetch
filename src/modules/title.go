package modules

import (
	"os"
	"os/user"
)

type title struct{}

func init() { Register(title{}) }

func (title) Name() string { return "title" }

func (title) ConfigFields() []string {
	return []string{"text", "format"}
}

func (title) Run(ctx Context) ([]Entry, error) {
	real := userName() + "@" + hostName()
	value := ctx.Text
	if value == "" {
		value = real
	}
	return []Entry{{Kind: KindTitle, Key: real, Value: value}}, nil
}

func userName() string {
	if current, err := user.Current(); err == nil && current.Username != "" {
		return current.Username
	}
	for _, variable := range []string{"USER", "LOGNAME"} {
		if value := os.Getenv(variable); value != "" {
			return value
		}
	}
	return "unknown"
}

func hostName() string {
	if name, err := os.Hostname(); err == nil && name != "" {
		return name
	}
	if name, err := ReadTrim(procPath("sys", "kernel", "hostname")); err == nil && name != "" {
		return name
	}
	return "localhost"
}
