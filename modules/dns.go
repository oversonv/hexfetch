//go:build linux

package modules

import (
	"strings"
)

type dns struct{}

func init() { Register(dns{}) }

func (dns) Name() string { return "dns" }

func (dns) Run(ctx Context) ([]Entry, error) {
	data, err := readFile(etcPath("resolv.conf"))
	if err != nil {
		return nil, err
	}
	servers := parseResolvConf(data)
	if len(servers) == 0 {
		return nil, Unsupported("DNS servers unavailable")
	}
	return Single(ctx, "DNS", strings.Join(servers, ", ")), nil
}

func parseResolvConf(data []byte) []string {
	seen := map[string]bool{}
	var servers []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "nameserver" {
			continue
		}
		if seen[fields[1]] {
			continue
		}
		seen[fields[1]] = true
		servers = append(servers, fields[1])
	}
	return servers
}
