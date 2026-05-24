package addr

import (
	"net/netip"
	"testing"
)

var testAddrs = map[string]struct {
	uri       string
	addr      Addr
	selfEqual bool
}{
	"tcp ipv4": {
		uri: "tcp://192.168.1.1:8080",
		addr: Addr{
			Addr:   "192.168.1.1",
			IPAddr: netip.MustParseAddr("192.168.1.1"),
			Port:   8080,
			Proto:  ProtoTCP,
		},
		selfEqual: true,
	},
	"tcp ipv6": {
		uri: "tcp://[::1]:9090",
		addr: Addr{
			Addr:   "::1",
			IPAddr: netip.MustParseAddr("::1"),
			Port:   9090,
			Proto:  ProtoTCP,
		},
		selfEqual: true,
	},
	"tcp ipv4 no-bracket": {
		uri: "tcp://10.0.0.1:8080",
		addr: Addr{
			Addr:   "10.0.0.1",
			IPAddr: netip.MustParseAddr("10.0.0.1"),
			Port:   8080,
			Proto:  ProtoTCP,
		},
		selfEqual: true,
	},
	"tcp fqdn": {
		uri: "tcp://example.com:443",
		addr: Addr{
			Addr:  "example.com",
			Port:  443,
			Proto: ProtoTCP,
		},
		selfEqual: true,
	},
	"tcp fqdn short": {
		uri: "tcp://host:1234",
		addr: Addr{
			Addr:  "host",
			Port:  1234,
			Proto: ProtoTCP,
		},
		selfEqual: true,
	},
	"udp ipv4": {
		uri: "udp://10.0.0.1:53",
		addr: Addr{
			Addr:   "10.0.0.1",
			IPAddr: netip.MustParseAddr("10.0.0.1"),
			Port:   53,
			Proto:  ProtoUDP,
		},
		selfEqual: true,
	},
	"udp ipv6": {
		uri: "udp://[fe80::1]:5353",
		addr: Addr{
			Addr:   "fe80::1",
			IPAddr: netip.MustParseAddr("fe80::1"),
			Port:   5353,
			Proto:  ProtoUDP,
		},
		selfEqual: true,
	},
	"udp fqdn": {
		uri: "udp://dns.example.com:5353",
		addr: Addr{
			Addr:  "dns.example.com",
			Port:  5353,
			Proto: ProtoUDP,
		},
		selfEqual: true,
	},
	"unix socket": {
		uri: "unix:///var/run/app.sock",
		addr: Addr{
			Addr:  "/var/run/app.sock",
			Proto: ProtoUnix,
		},
		selfEqual: true,
	},
	"unix abstract": {
		uri: "unix://@abstract",
		addr: Addr{
			Addr:  "@abstract",
			Proto: ProtoUnix,
		},
		selfEqual: true,
	},
	"loop": {
		uri: "loop://12345",
		addr: Addr{
			Port:  12345,
			Proto: ProtoLoop,
		},
		selfEqual: true,
	},
	"unknown": {
		uri: "unknown://",
		addr: Addr{
			Proto: ProtoUnknown,
		},
		selfEqual: false,
	},
}

func TestFromURI(t *testing.T) {
	for name, tc := range testAddrs {
		t.Run(name, func(t *testing.T) {
			got, err := FromURI(tc.uri)
			if err != nil {
				t.Errorf("FromURI(%q) unexpected error: %v", tc.uri, err)
				return
			}
			if tc.selfEqual != got.Equals(&tc.addr) {
				t.Errorf("FromURI(%q)\n  got  %+v\n  want %+v", tc.uri, got, tc.addr)
			}
		})
	}

	errTests := []struct {
		name string
		uri  string
	}{
		{"missing scheme", "192.168.1.1:8080"},
		{"unknown scheme", "ftp://example.com:21"},
		{"invalid loop port", "loop://99999"},
	}
	for _, tt := range errTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromURI(tt.uri)
			if err == nil {
				t.Errorf("FromURI(%q) expected error, got nil", tt.uri)
			}
		})
	}
}

func TestAddr_URI(t *testing.T) {
	for name, tc := range testAddrs {
		t.Run(name, func(t *testing.T) {
			got := tc.addr.URI()
			if got != tc.uri {
				t.Errorf("Addr.URI()\n  got  %q\n  want %q", got, tc.uri)
			}
		})
	}
}

func TestFromURI_URI_RoundTrip(t *testing.T) {
	for name, tc := range testAddrs {
		t.Run(name, func(t *testing.T) {
			addr, err := FromURI(tc.uri)
			if err != nil {
				t.Fatalf("FromURI(%q) unexpected error: %v", tc.uri, err)
			}
			got := addr.URI()
			if got != tc.uri {
				t.Errorf("round-trip failed\n  in   %q\n  out  %q", tc.uri, got)
			}
		})
	}
}
