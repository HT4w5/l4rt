package addr

import (
	"errors"
	"net/netip"
)

// Address type mismatch errors.
var (
	ErrNotIPAddr    = errors.New("not an IP address")
	ErrNotUnixAddr  = errors.New("not a Unix address")
	ErrNotLoopAddr  = errors.New("not a Loop address")
	ErrNotTCPIPAddr = errors.New("not a TCP/IP address")
	ErrNotUDPIPAddr = errors.New("not a UDP/IP address")
)

func (a *Addr) AssertIPAddr() (ap netip.AddrPort, err error) {
	if !a.IPAddr.IsValid() {
		err = ErrNotIPAddr
		return
	}
	ap = netip.AddrPortFrom(a.IPAddr, a.Port)
	return
}

func (a *Addr) AssertUnixAddr() (addr string, err error) {
	if a.Proto != ProtoUnix {
		err = ErrNotUnixAddr
		return
	}
	addr = a.Addr
	return
}

func (a *Addr) AssertLoopAddr() (port uint16, err error) {
	if a.Proto != ProtoLoop {
		err = ErrNotLoopAddr
		return
	}
	port = a.Port
	return
}

func (a *Addr) AssertTCPIPAddr() (ap netip.AddrPort, err error) {
	if !a.IPAddr.IsValid() || a.Proto != ProtoTCP {
		err = ErrNotTCPIPAddr
		return
	}
	ap = netip.AddrPortFrom(a.IPAddr, a.Port)
	return
}

func (a *Addr) AssertUDPIPAddr() (ap netip.AddrPort, err error) {
	if !a.IPAddr.IsValid() || a.Proto != ProtoUDP {
		err = ErrNotLoopAddr
		return
	}
	ap = netip.AddrPortFrom(a.IPAddr, a.Port)
	return
}
