package addr

import (
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

type Proto uint8

const (
	ProtoUnknown Proto = iota
	ProtoTCP
	ProtoUDP
	ProtoUnix
	ProtoLoop
)

// Addr represents the origin or destination of a stream.
//
// If Proto is TCP or UDP and IPAddr is invalid, Addr is used as FQDN.
// If Proto is Unix, Addr is used as socket path.
//
// Addr implements [net.Addr].
type Addr struct {
	Addr   string
	IPAddr netip.Addr
	Port   uint16
	Proto  Proto
}

// Network returns protocol type string of address.
//
// Possible values: "tcp", "udp", "unix", "loop", "unknown".
//
// Network implements [net.Addr.Network].
func (a *Addr) Network() string {
	switch a.Proto {
	case ProtoTCP:
		return "tcp"
	case ProtoUDP:
		return "udp"
	case ProtoUnix:
		return "unix"
	case ProtoLoop:
		return "loop"
	default:
		return "unknown"
	}
}

func (a *Addr) IsIPAddr() bool {
	return a.IPAddr.IsValid()
}

// String returns string form of address.
//
// For TCP and UDP, "<ipaddr/fqdn>:<port>".
// For Unix, "<socket_path>".
// For Loop, "<port>".
// For Unknown, "".
//
// String implements [net.Addr.String].
func (a *Addr) String() string {
	switch a.Proto {
	case ProtoTCP:
		fallthrough
	case ProtoUDP:
		if a.IPAddr.IsValid() {
			return a.IPAddr.String() + ":" + strconv.FormatUint(uint64(a.Port), 10)
		} else {
			return a.Addr + ":" + strconv.FormatUint(uint64(a.Port), 10)
		}
	case ProtoUnix:
		return a.Addr
	case ProtoLoop:
		return strconv.FormatUint(uint64(a.Port), 10)
	case ProtoUnknown:
		fallthrough
	default:
		return ""
	}
}

// URI returns URI string of address.
func (a *Addr) URI() string {
	switch a.Proto {
	case ProtoTCP:
		if a.IPAddr.IsValid() {
			if a.IPAddr.Is6() {
				return "tcp://[" + a.IPAddr.String() + "]:" + strconv.FormatUint(uint64(a.Port), 10)
			}
			return "tcp://" + a.IPAddr.String() + ":" + strconv.FormatUint(uint64(a.Port), 10)
		} else {
			return "tcp://" + a.Addr + ":" + strconv.FormatUint(uint64(a.Port), 10)
		}
	case ProtoUDP:
		if a.IPAddr.IsValid() {
			if a.IPAddr.Is6() {
				return "udp://[" + a.IPAddr.String() + "]:" + strconv.FormatUint(uint64(a.Port), 10)
			}
			return "udp://" + a.IPAddr.String() + ":" + strconv.FormatUint(uint64(a.Port), 10)
		} else {
			return "udp://" + a.Addr + ":" + strconv.FormatUint(uint64(a.Port), 10)
		}
	case ProtoUnix:
		return "unix://" + a.Addr
	case ProtoLoop:
		return "loop://" + strconv.FormatUint(uint64(a.Port), 10)
	case ProtoUnknown:
		fallthrough
	default:
		return "unknown://"
	}
}

// Equals determines whether two Addrs are equal.
func (a *Addr) Equals(other *Addr) bool {
	if a == other {
		return true
	}

	if a.Proto != other.Proto {
		return false
	}

	switch a.Proto {
	case ProtoTCP:
		fallthrough
	case ProtoUDP:
		if a.Port != other.Port {
			return false
		}

		if a.IPAddr.IsValid() {
			if other.IPAddr.IsValid() {
				if a.IPAddr == other.IPAddr {
					return true
				}
				return false
			}
			return false
		}

		if other.IPAddr.IsValid() {
			return false
		}

		if a.Addr == other.Addr {
			return true
		}
		return false
	case ProtoUnix:
		return a.Addr == other.Addr
	case ProtoLoop:
		return a.Port == other.Port
	case ProtoUnknown:
		fallthrough
	default:
		return false
	}
}

// FromURI parses a URI string into an Addr.
func FromURI(rawURI string) (Addr, error) {
	scheme, rest, ok := strings.Cut(rawURI, "://")
	if !ok {
		return Addr{}, fmt.Errorf("addr.FromURI: missing scheme in %q", rawURI)
	}

	switch scheme {
	case "tcp":
		return parseAddrPort(rest, ProtoTCP)
	case "udp":
		return parseAddrPort(rest, ProtoUDP)
	case "unix":
		return Addr{
			Addr:  rest,
			Proto: ProtoUnix,
		}, nil
	case "loop":
		port, err := strconv.ParseUint(rest, 10, 16)
		if err != nil {
			return Addr{}, fmt.Errorf("addr.FromURI: parse loop port: %w", err)
		}
		return Addr{
			Port:  uint16(port),
			Proto: ProtoLoop,
		}, nil
	case "unknown":
		return Addr{Proto: ProtoUnknown}, nil
	default:
		return Addr{}, fmt.Errorf("addr.FromURI: unknown scheme %q", scheme)
	}
}

func parseAddrPort(hostPort string, proto Proto) (Addr, error) {
	ap, err := netip.ParseAddrPort(hostPort)
	if err == nil {
		return Addr{
			Addr:   ap.Addr().String(),
			IPAddr: ap.Addr(),
			Port:   ap.Port(),
			Proto:  proto,
		}, nil
	}

	// Not a valid IP:port, treat as FQDN:port.
	colon := strings.LastIndex(hostPort, ":")
	if colon < 0 {
		return Addr{}, fmt.Errorf("addr.FromURI: missing port in %q", hostPort)
	}
	port, err := strconv.ParseUint(hostPort[colon+1:], 10, 16)
	if err != nil {
		return Addr{}, fmt.Errorf("addr.FromURI: parse port: %w", err)
	}
	return Addr{
		Addr:  hostPort[:colon],
		Port:  uint16(port),
		Proto: proto,
	}, nil
}

func FromAddrPort(addrPort netip.AddrPort, proto Proto) (Addr, error) {
	addr := addrPort.Addr().Unmap()
	if !addr.IsValid() {
		return Addr{}, errors.New("addr.FromAddrPort: invalid IP address")
	}

	return Addr{
		IPAddr: addr,
		Port:   addrPort.Port(),
		Proto:  proto,
	}, nil
}
