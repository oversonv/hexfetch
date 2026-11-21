//go:build linux

package modules

import (
	"path/filepath"
	"strings"
)

type gpu struct{}

func init() { Register(gpu{}) }

func (gpu) Name() string { return "gpu" }

func (gpu) Run(ctx Context) ([]Entry, error) {
	devices, _ := filepath.Glob(sysPath("bus/pci/devices/*"))
	var entries []Entry
	seen := map[string]bool{}
	for _, device := range devices {
		class, err := readTrimmed(filepath.Join(device, "class"))
		if err != nil || !strings.HasPrefix(strings.ToLower(class), "0x03") {
			continue
		}
		vendor, _ := readTrimmed(filepath.Join(device, "vendor"))
		product, _ := readTrimmed(filepath.Join(device, "device"))
		if vendor == "" {
			continue
		}
		key := vendor + ":" + product
		if seen[key] {
			continue
		}
		seen[key] = true
		driver := ""
		if target, err := filepath.EvalSymlinks(filepath.Join(device, "driver")); err == nil {
			driver = filepath.Base(target)
		}
		value := gpuName(vendor, product)
		if driver != "" {
			value += " (" + driver + ")"
		}
		entries = append(entries, Entry{Key: "GPU", Value: value})
	}
	if len(entries) == 0 {
		return nil, Unsupported("GPU information unavailable")
	}
	return entries, nil
}

var pciVendors = map[string]string{
	"0x1002": "AMD",
	"0x1010": "ImgTec",
	"0x10de": "NVIDIA",
	"0x1234": "QEMU",
	"0x13b5": "ARM",
	"0x1af4": "VirtIO",
	"0x1b36": "QXL",
	"0x5143": "Qualcomm",
	"0x8086": "Intel",
	"0x15ad": "VMware",
}

func gpuName(vendor, product string) string {
	vendor = strings.ToLower(strings.TrimSpace(vendor))
	product = strings.ToLower(strings.TrimSpace(product))
	name := pciVendors[vendor]
	if name == "" {
		name = vendor
	}
	if product != "" {
		name += " " + product
	}
	return name
}
