//go:build linux

package modules

import (
	"net"
	"strings"
)

type localip struct{}

func init() { Register(localip{}) }

func (localip) Name() string { return "localip" }

func (localip) OptionSchema() map[string]OptionDefinition {
	return map[string]OptionDefinition{
		"family":    {Kind: OptionString, Allowed: []string{"both", "4", "ipv4", "6", "ipv6"}},
		"interface": {Kind: OptionString},
		"exclude":   {Kind: OptionStringList, AllowScalar: true},
	}
}

func (localip) Run(ctx Context) ([]Entry, error) {
	family := ctx.OptionString("family", "both")
	wanted := ctx.OptionString("interface", "")
	exclude := append(defaultInterfaceExcludes(), ctx.OptionStrings("exclude")...)

	defaults := map[string]bool{}
	if data, err := readFile(procPath("net", "route")); err == nil {
		for _, name := range parseIPv4DefaultRoutes(data) {
			defaults[name] = true
		}
	}
	if data, err := readFile(procPath("net", "ipv6_route")); err == nil {
		for _, name := range parseIPv6DefaultRoutes(data) {
			defaults[name] = true
		}
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		// netlink может быть запрещён (proot/контейнеры) — это повод
		// пропустить модуль, а не падать.
		return nil, Unsupported("local IP addresses unavailable")
	}
	var entries []Entry
	seen := map[string]bool{}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if wanted != "" && iface.Name != wanted {
			continue
		}
		if wanted == "" && excludedInterface(iface.Name, exclude) {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ip := addressIP(address)
			if ip == nil || !addressWanted(ip, family) {
				continue
			}
			value := ip.String()
			if seen[value] {
				continue
			}
			seen[value] = true
			key := "Local IP (" + iface.Name + ")"
			if defaults[iface.Name] {
				key = "Local IP"
			}
			entries = append(entries, Entry{Key: key, Value: value})
		}
	}
	if len(entries) == 0 {
		return nil, Unsupported("local IP addresses unavailable")
	}
	return entries, nil
}

func defaultInterfaceExcludes() []string {
	return []string{"lo", "docker", "veth", "br-", "virbr", "vnet"}
}

func excludedInterface(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix != "" && strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func addressIP(address net.Addr) net.IP {
	switch value := address.(type) {
	case *net.IPNet:
		return value.IP
	case *net.IPAddr:
		return value.IP
	default:
		host, _, err := net.ParseCIDR(address.String())
		if err != nil {
			return nil
		}
		return host
	}
}

func addressWanted(ip net.IP, family string) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	switch family {
	case "4", "ipv4":
		return ip.To4() != nil
	case "6", "ipv6":
		return ip.To4() == nil
	default:
		return true
	}
}

func parseIPv4DefaultRoutes(data []byte) []string {
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] == "Iface" {
			continue
		}
		if fields[1] == "00000000" {
			names = append(names, fields[0])
		}
	}
	return names
}

func parseIPv6DefaultRoutes(data []byte) []string {
	var names []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		if strings.Trim(fields[0], "0") == "" && strings.Trim(fields[1], "0") == "" {
			names = append(names, fields[len(fields)-1])
		}
	}
	return names
}
