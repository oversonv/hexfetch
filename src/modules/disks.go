//go:build linux

package modules

import (
	"fmt"
	"strings"
	"syscall"
)

type disks struct{}

func init() { Register(disks{}) }

func (disks) Name() string { return "disks" }

func (disks) Run(ctx Context) ([]Entry, error) {
	data, err := readFile(procPath("mounts"))
	if err != nil {
		return nil, err
	}
	var entries []Entry
	seen := map[string]bool{}
	// Different device nodes can map to the same storage (device mapper,
	// proot overlays). When two mounts report identical usage, keep the first.
	seenUsage := map[[2]uint64]bool{}
	for _, mount := range parseMounts(data) {
		if !usableMount(mount) || seen[mount.device] {
			continue
		}
		seen[mount.device] = true
		var stat syscall.Statfs_t
		if err := syscall.Statfs(mount.mountpoint, &stat); err != nil {
			continue
		}
		total := stat.Blocks * uint64(stat.Bsize)
		if total == 0 {
			continue
		}
		used := (stat.Blocks - stat.Bfree) * uint64(stat.Bsize)
		usage := [2]uint64{total, used}
		if seenUsage[usage] {
			continue
		}
		seenUsage[usage] = true
		value := fmt.Sprintf("%s / %s (%s)", formatBytes(used), formatBytes(total), formatPercent(used, total))
		entries = append(entries, Entry{Key: "Disk (" + mount.mountpoint + ")", Value: value})
	}
	if len(entries) == 0 {
		return nil, Unsupported("mounted disks unavailable")
	}
	return entries, nil
}

type mountEntry struct {
	device     string
	mountpoint string
	fstype     string
}

func parseMounts(data []byte) []mountEntry {
	var mounts []mountEntry
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		mounts = append(mounts, mountEntry{
			device:     unescapeMount(fields[0]),
			mountpoint: unescapeMount(fields[1]),
			fstype:     fields[2],
		})
	}
	return mounts
}

func unescapeMount(value string) string {
	replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	return replacer.Replace(value)
}

var pseudoFilesystems = map[string]bool{
	"proc": true, "sysfs": true, "tmpfs": true, "devtmpfs": true,
	"devpts": true, "cgroup": true, "cgroup2": true, "overlay": true,
	"squashfs": true, "autofs": true, "mqueue": true, "hugetlbfs": true,
	"debugfs": true, "tracefs": true, "securityfs": true, "pstore": true,
	"bpf": true, "configfs": true, "fusectl": true, "ramfs": true,
	"nsfs": true, "efivarfs": true, "binfmt_misc": true, "rpc_pipefs": true,
}

var ignoredMountpoints = []string{
	"/apex", "/system", "/vendor", "/product", "/odm", "/oem",
	"/metadata", "/mnt/vendor", "/mnt/product", "/mnt/installer",
}

func usableMount(mount mountEntry) bool {
	if pseudoFilesystems[mount.fstype] {
		return false
	}
	for _, prefix := range ignoredMountpoints {
		if mount.mountpoint == prefix || strings.HasPrefix(mount.mountpoint, prefix+"/") {
			return false
		}
	}
	if strings.HasPrefix(mount.device, "/dev/") {
		return true
	}
	switch mount.fstype {
	case "zfs", "btrfs", "xfs", "ext2", "ext3", "ext4", "f2fs",
		"vfat", "ntfs", "ntfs3", "exfat", "reiserfs", "jfs", "bcachefs":
		return !strings.HasPrefix(mount.mountpoint, "/proc") &&
			!strings.HasPrefix(mount.mountpoint, "/sys") &&
			!strings.HasPrefix(mount.mountpoint, "/dev")
	}
	return false
}
