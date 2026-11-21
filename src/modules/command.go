package modules

import (
	"context"
	"os/exec"
	"strings"
)

type command struct{}

func init() { Register(command{}) }

func (command) Name() string { return "command" }

func (command) ConfigFields() []string {
	return []string{"key", "key_icon", "text", "command", "key_color", "format"}
}

func (command) Run(ctx Context) ([]Entry, error) {
	if ctx.Command == "" {
		return nil, Unsupported("command is not configured")
	}
	timeout, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(timeout, "sh", "-c", ctx.Command).Output()
	if err != nil {
		return nil, err
	}
	value, _, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, Unsupported("command produced no output")
	}
	return Single(ctx, "Command", value), nil
}
