package addr

import (
	"net/netip"
	"strconv"
)

type Family int8

const (
	FamilyUnknown Family = iota
	FamilyIP
	FamilyTCP
	FamilyUDP
	FamilyUnix
	FamilyUnixGram
	FamilyUnixPacket
	FamilyLoop
)

func (f Family) String() string {
	switch f {
	default:
		fallthrough
	case FamilyUnknown:
		return "unknown"
	case FamilyIP:
		return "ip"
	case FamilyTCP:
		return "tcp"
	case FamilyUDP:
		return "udp"
	case FamilyUnix:
		return "unix"
	case FamilyUnixGram:
		return "unixgram"
	case FamilyUnixPacket:
		return "unixpacket"
	case FamilyLoop:
		return "loop"
	}
}

var Unknown = Addr{
	Family: FamilyUnknown,
}

type Addr struct {
	IPAddr   netip.Addr
	Addr     string
	MuxIndex uint16
	Family   Family
}

func (addr *Addr) URI() string {
	var b []byte
	switch addr.Family {
	default:
		fallthrough
	case FamilyUnknown:
		return "unknown://"
	case FamilyIP:
		const base = len("ip://")
		const max4 = len("255.255.255.255:65535") + base
		const max4in6 = len("[::ffff:255.255.255.255%enp5s0]") + base
		const max6 = len("[ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff%enp5s0]") + base

		if addr.IPAddr.Is4() {
			b = make([]byte, 0, max4)
			b = append(b, "ip://"...)
			b = addr.IPAddr.AppendTo(b)
		} else if addr.IPAddr.Is4In6() {
			b = make([]byte, 0, max4in6)
			b = append(b, "ip://"...)
			b = append(b, '[')
			b = addr.IPAddr.AppendTo(b)
			b = append(b, ']')
		} else {
			b = make([]byte, 0, max6)
			b = append(b, "ip://"...)
			b = append(b, '[')
			b = addr.IPAddr.AppendTo(b)
			b = append(b, ']')
		}

	case FamilyTCP:
		fallthrough
	case FamilyUDP:
		const base = len("tcp://")
		const max4 = len("255.255.255.255:65535") + base
		const max4in6 = len("[::ffff:255.255.255.255%enp5s0]:65535") + base
		const max6 = len("[ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff%enp5s0]:65535") + base

		scheme := addr.Family.String()
		if addr.IPAddr.Is4() {
			b = make([]byte, 0, max4)
			b = append(b, scheme...)
			b = append(b, "://"...)
			b = addr.IPAddr.AppendTo(b)
		} else if addr.IPAddr.Is4In6() {
			b = make([]byte, 0, max4in6)
			b = append(b, scheme...)
			b = append(b, "://"...)
			b = append(b, '[')
			b = addr.IPAddr.AppendTo(b)
			b = append(b, ']')
		} else {
			b = make([]byte, 0, max6)
			b = append(b, scheme...)
			b = append(b, "://"...)
			b = append(b, '[')
			b = addr.IPAddr.AppendTo(b)
			b = append(b, ']')
		}
		b = append(b, ':')
		b = strconv.AppendUint(b, uint64(addr.MuxIndex), 10)
	case FamilyUnix:
		const scheme = "unix://"
		const base = len(scheme)
		b = make([]byte, 0, base+len(addr.Addr))
		b = append(b, scheme...)
		b = append(b, addr.Addr...)
	case FamilyUnixGram:
		const scheme = "unixgram://"
		const base = len(scheme)
		b = make([]byte, 0, base+len(addr.Addr))
		b = append(b, scheme...)
		b = append(b, addr.Addr...)
	case FamilyUnixPacket:
		const scheme = "unixpacket://"
		const base = len(scheme)
		b = make([]byte, 0, base+len(addr.Addr))
		b = append(b, scheme...)
		b = append(b, addr.Addr...)
	case FamilyLoop:
		const scheme = "loop://"
		const base = len(scheme)
		const max = len("65535") + base
		b = make([]byte, 0, max)
		b = append(b, scheme...)
		b = strconv.AppendUint(b, uint64(addr.MuxIndex), 10)
	}

	return string(b)
}
