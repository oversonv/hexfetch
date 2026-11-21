package modules

import (
	"bytes"
	"encoding/binary"
	"os"
	"slices"
	"strings"
)

type users struct{}

func init() { Register(users{}) }

func (users) Name() string { return "users" }

func (users) Run(ctx Context) ([]Entry, error) {
	names := loggedIn()
	if len(names) == 0 {
		if name := os.Getenv("USER"); name != "" {
			names = []string{name}
		}
	}
	if len(names) == 0 {
		return nil, Unsupported("no logged-in users found")
	}
	return Single(ctx, "Users", strings.Join(names, ", ")), nil
}

func loggedIn() []string {
	var data []byte
	for _, path := range []string{"/var/run/utmp", "/run/utmp"} {
		content, err := os.ReadFile(path)
		if err == nil {
			data = content
			break
		}
	}
	const recordSize = 384
	var names []string
	for offset := 0; offset+recordSize <= len(data); offset += recordSize {
		record := data[offset : offset+recordSize]
		if binary.NativeEndian.Uint16(record[0:2]) != 7 {
			continue
		}
		name := cString(record[44:76])
		if name != "" && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

func cString(raw []byte) string {
	if end := bytes.IndexByte(raw, 0); end >= 0 {
		raw = raw[:end]
	}
	return string(raw)
}
