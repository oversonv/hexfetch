//go:build linux

package modules

import (
	"strings"
)

type board struct{}

func init() { Register(board{}) }

func (board) Name() string { return "board" }

func (board) Run(ctx Context) ([]Entry, error) {
	vendor := dmiValue("board_vendor")
	name := dmiValue("board_name")
	version := dmiValue("board_version")
	parts := []string{}
	if vendor != "" {
		parts = append(parts, vendor)
	}
	if name != "" {
		parts = append(parts, name)
	}
	if len(parts) == 0 {
		return nil, Unsupported("motherboard information unavailable")
	}
	value := strings.Join(parts, " ")
	if version != "" && version != name {
		value += " (" + version + ")"
	}
	return Single(ctx, "Board", value), nil
}
