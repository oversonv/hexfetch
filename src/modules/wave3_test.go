package modules

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseIPv4DefaultRoutes(t *testing.T) {
	names := parseIPv4DefaultRoutes(readFixture(t, "proc/net_route"))
	if len(names) != 2 || names[0] != "eth0" || names[1] != "wlan0" {
		t.Errorf("routes = %v, want [eth0 wlan0]", names)
	}
	if got := parseIPv4DefaultRoutes([]byte("Iface\tDestination\n")); len(got) != 0 {
		t.Errorf("header = %v", got)
	}
	if got := parseIPv4DefaultRoutes([]byte("garbage line")); len(got) != 0 {
		t.Errorf("garbage = %v", got)
	}
}

func TestParseIPv6DefaultRoutes(t *testing.T) {
	names := parseIPv6DefaultRoutes(readFixture(t, "proc/net_ipv6_route"))
	if len(names) != 1 || names[0] != "eth0" {
		t.Errorf("routes = %v, want [eth0]", names)
	}
	if got := parseIPv6DefaultRoutes([]byte("short line\n")); len(got) != 0 {
		t.Errorf("short = %v", got)
	}
}

func TestExcludedInterface(t *testing.T) {
	tests := map[string]bool{
		"lo":       true,
		"docker0":  true,
		"veth1234": true,
		"br-f00d":  true,
		"virbr0":   true,
		"vnet0":    true,
		"eth0":     false,
		"wlan0":    false,
		"bridge9":  false,
	}
	for name, want := range tests {
		if got := excludedInterface(name, defaultInterfaceExcludes()); got != want {
			t.Errorf("excludedInterface(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestAddressWanted(t *testing.T) {
	tests := []struct {
		ip     string
		family string
		want   bool
	}{
		{"192.168.1.10", "both", true},
		{"192.168.1.10", "4", true},
		{"192.168.1.10", "6", false},
		{"192.168.1.10", "ipv6", false},
		{"fd00::1234", "both", true},
		{"fd00::1234", "6", true},
		{"fd00::1234", "4", false},
		{"127.0.0.1", "both", false},
		{"::1", "both", false},
		{"169.254.1.5", "both", false},
		{"fe80::1234", "both", false},
		{"224.0.0.1", "both", false},
	}
	for _, test := range tests {
		if got := addressWanted(net.ParseIP(test.ip), test.family); got != test.want {
			t.Errorf("addressWanted(%q, %q) = %v, want %v", test.ip, test.family, got, test.want)
		}
	}
}

func TestParseProcWireless(t *testing.T) {
	signals := parseProcWireless(readFixture(t, "proc/net_wireless"))
	if signals["wlan0"] != 60 || signals["wlan1"] != 5555 {
		t.Errorf("signals = %v, want wlan0=60 wlan1=5555", signals)
	}
	if got := parseProcWireless([]byte("Inter-|   Receive\n face |bytes\n")); len(got) != 0 {
		t.Errorf("headers = %v", got)
	}
	if got := parseProcWireless([]byte(" wlan0: 0000   abc.  -40.\n")); len(got) != 0 {
		t.Errorf("non-numeric = %v", got)
	}
}

func TestParseResolvConf(t *testing.T) {
	servers := parseResolvConf(readFixture(t, "etc/resolv.conf"))
	if len(servers) != 2 || servers[0] != "8.8.8.8" || servers[1] != "1.1.1.1" {
		t.Errorf("servers = %v, want [8.8.8.8 1.1.1.1]", servers)
	}
	if got := parseResolvConf([]byte("# only comments\n; more\n")); len(got) != 0 {
		t.Errorf("comments = %v", got)
	}
}

func TestFetchPublicIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("203.0.113.7\n"))
	}))
	defer server.Close()

	ip, err := fetchPublicIP(server.URL, 3e9)
	if err != nil {
		t.Fatalf("fetchPublicIP: %v", err)
	}
	if ip != "203.0.113.7" {
		t.Errorf("ip = %q, want 203.0.113.7", ip)
	}

	status := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer status.Close()
	if _, err := fetchPublicIP(status.URL, 3e9); err == nil {
		t.Error("expected error for non-2xx response")
	}

	body := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-an-ip"))
	}))
	defer body.Close()
	if _, err := fetchPublicIP(body.URL, 3e9); err == nil || !strings.Contains(err.Error(), "invalid address") {
		t.Errorf("err = %v, want invalid address", err)
	}
}

func TestPublicIPInvalidEndpoint(t *testing.T) {
	_, err := publicip{}.Run(Context{Options: map[string]any{"url": "ftp://example.com"}})
	if err == nil || !strings.Contains(err.Error(), "http(s) URL") {
		t.Errorf("err = %v, want scheme error", err)
	}
}
