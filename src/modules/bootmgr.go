//go:build linux

package modules

import (
	"path/filepath"
	"slices"
	"strings"
)

type bootmgr struct{}

func init() { Register(bootmgr{}) }

func (bootmgr) Name() string { return "bootmgr" }

func (bootmgr) Run(ctx Context) ([]Entry, error) {
	value := bootManagerName()
	if value == "" {
		return nil, Unsupported("boot manager unknown")
	}
	return Single(ctx, "Boot Manager", value), nil
}

func bootManagerName() string {
	firmware := ""
	if exists(sysPath("firmware", "efi")) {
		firmware = "EFI"
	}
	managers := detectBootManagers()
	if len(managers) == 0 {
		return firmware
	}
	if firmware != "" {
		return firmware + " (" + strings.Join(managers, ", ") + ")"
	}
	return strings.Join(managers, ", ")
}

func detectBootManagers() []string {
	var managers []string
	if entries, _ := filepath.Glob("/boot/loader/entries/*.conf"); len(entries) > 0 {
		managers = append(managers, "systemd-boot")
	} else if exists("/efi/loader/entries") {
		managers = append(managers, "systemd-boot")
	}
	if exists("/boot/grub/grub.cfg") || exists("/boot/grub2/grub.cfg") {
		managers = append(managers, "GRUB")
	}
	if !slices.Contains(managers, "systemd-boot") && exists("/boot/efi/EFI/systemd") {
		managers = append(managers, "systemd-boot")
	}
	return managers
}

// loaderEntryTitle reads the title line of a systemd-boot entry.
func loaderEntryTitle(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if key, value, ok := strings.Cut(line, " "); ok && key == "title" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
