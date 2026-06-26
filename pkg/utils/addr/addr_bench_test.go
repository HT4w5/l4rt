// LLM usage: generated with deepseek-v4-pro and modified manually
package addr

import (
	"net/netip"
	"testing"
)

func BenchmarkAddrURI_Unknown(b *testing.B) {
	a := Unknown
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_IP(b *testing.B) {
	a := Addr{Family: FamilyIP, IPAddr: netip.MustParseAddr("192.168.1.1")}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_TCP(b *testing.B) {
	a := Addr{Family: FamilyTCP, IPAddr: netip.MustParseAddr("192.168.1.1"), MuxIndex: 8080}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_UDP(b *testing.B) {
	a := Addr{Family: FamilyUDP, IPAddr: netip.MustParseAddr("10.0.0.1"), MuxIndex: 5353}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_Unix(b *testing.B) {
	a := Addr{Family: FamilyUnix, Addr: "/var/run/app.sock"}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_UnixGram(b *testing.B) {
	a := Addr{Family: FamilyUnixGram, Addr: "/var/run/gram.sock"}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_UnixPacket(b *testing.B) {
	a := Addr{Family: FamilyUnixPacket, Addr: "/var/run/packet.sock"}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}

func BenchmarkAddrURI_Loop(b *testing.B) {
	a := Addr{Family: FamilyLoop, MuxIndex: 42}
	b.ResetTimer()
	for b.Loop() {
		_ = a.URI()
	}
}
