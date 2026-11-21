package main

import (
	_ "embed"
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// metaTOML is hexfetch.toml, compiled into the binary.
//
//go:embed hexfetch.toml
var metaTOML []byte

type packageMeta struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
	Repository  string `toml:"repository"`
	License     string `toml:"license"`
}

type metaFile struct {
	Package packageMeta `toml:"package"`
}

func parseMeta(data []byte) (packageMeta, error) {
	var file metaFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return packageMeta{}, fmt.Errorf("hexfetch.toml: %w", err)
	}
	if file.Package.Version == "" {
		return packageMeta{}, fmt.Errorf("hexfetch.toml: [package].version is empty")
	}
	return file.Package, nil
}

// appVersion parses the embedded metadata only when the version is requested,
// so a damaged file can never affect a normal run.
func appVersion() string {
	meta, err := parseMeta(metaTOML)
	if err != nil {
		return "unknown"
	}
	return meta.Version
}
