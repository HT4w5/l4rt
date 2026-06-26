// LLM usage: generated with deepseek-v4-pro and modified manually
package addr

import (
	"net/netip"
	"testing"
)

// =============================================================================
// URI correctness tests
// =============================================================================

func TestAddr_URI_Unknown(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "zero value sentinel",
			addr: Unknown,
			want: "unknown://",
		},
		{
			name: "explicit FamilyUnknown",
			addr: Addr{Family: FamilyUnknown},
			want: "unknown://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// ===== TCP =====

func TestAddr_URI_TCP(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "IPv4",
			addr: Addr{Family: FamilyTCP, IPAddr: netip.MustParseAddr("1.2.3.4"), MuxIndex: 8080},
			want: "tcp://1.2.3.4:8080",
		},
		{
			name: "IPv6",
			addr: Addr{Family: FamilyTCP, IPAddr: netip.MustParseAddr("::1"), MuxIndex: 443},
			want: "tcp://[::1]:443",
		},
		{
			name: "IPv4-in-IPv6",
			addr: Addr{Family: FamilyTCP, IPAddr: netip.MustParseAddr("::ffff:1.2.3.4"), MuxIndex: 53},
			want: "tcp://[::ffff:1.2.3.4]:53",
		},
		{
			name: "IPv6 with zone",
			addr: Addr{Family: FamilyTCP, IPAddr: netip.MustParseAddr("fe80::1%eth0"), MuxIndex: 9090},
			want: "tcp://[fe80::1%eth0]:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// ===== UDP =====

func TestAddr_URI_UDP(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "IPv4",
			addr: Addr{Family: FamilyUDP, IPAddr: netip.MustParseAddr("10.0.0.1"), MuxIndex: 5353},
			want: "udp://10.0.0.1:5353",
		},
		{
			name: "IPv6",
			addr: Addr{Family: FamilyUDP, IPAddr: netip.MustParseAddr("::1"), MuxIndex: 123},
			want: "udp://[::1]:123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// ===== IP =====

func TestAddr_URI_IP(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "IPv4",
			addr: Addr{Family: FamilyIP, IPAddr: netip.MustParseAddr("1.2.3.4")},
			want: "ip://1.2.3.4",
		},
		{
			name: "IPv6",
			addr: Addr{Family: FamilyIP, IPAddr: netip.MustParseAddr("::1")},
			want: "ip://[::1]",
		},
		{
			name: "IPv4-in-IPv6",
			addr: Addr{Family: FamilyIP, IPAddr: netip.MustParseAddr("::ffff:1.2.3.4")},
			want: "ip://[::ffff:1.2.3.4]",
		},
		{
			name: "IPv6 with zone",
			addr: Addr{Family: FamilyIP, IPAddr: netip.MustParseAddr("fe80::1%eth0")},
			want: "ip://[fe80::1%eth0]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// ===== Unix =====

func TestAddr_URI_Unix(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "path",
			addr: Addr{Family: FamilyUnix, Addr: "/var/run/app.sock"},
			want: "unix:///var/run/app.sock",
		},
		{
			name: "empty path",
			addr: Addr{Family: FamilyUnix, Addr: ""},
			want: "unix://",
		},
		{
			name: "abstract socket",
			addr: Addr{Family: FamilyUnix, Addr: "@abstract"},
			want: "unix://@abstract",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

// ===== UnixGram =====

func TestAddr_URI_UnixGram(t *testing.T) {
	addr := Addr{Family: FamilyUnixGram, Addr: "/var/run/gram.sock"}
	want := "unixgram:///var/run/gram.sock"
	got := addr.URI()
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// ===== UnixPacket =====

func TestAddr_URI_UnixPacket(t *testing.T) {
	addr := Addr{Family: FamilyUnixPacket, Addr: "/var/run/packet.sock"}
	want := "unixpacket:///var/run/packet.sock"
	got := addr.URI()
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// ===== Loop =====

func TestAddr_URI_Loop(t *testing.T) {
	tests := []struct {
		name string
		addr Addr
		want string
	}{
		{
			name: "index 0",
			addr: Addr{Family: FamilyLoop, MuxIndex: 0},
			want: "loop://0",
		},
		{
			name: "max uint16",
			addr: Addr{Family: FamilyLoop, MuxIndex: 65535},
			want: "loop://65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.addr.URI()
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
