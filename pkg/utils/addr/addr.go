package addr

import "net/netip"

type Family int8

const (
	FamilyUnknown Family = iota
	FamilyIP
	FamilyIPTCP
	FamilyIPUDP
	FamilyUnix
	FamilyUnixGram
	FamilyUnixPacket
	FamilyLoop
)

var Default = Addr{
	Family: FamilyUnknown,
}

type Addr struct {
	IPAddr   netip.Addr
	Addr     string
	MuxIndex uint16
	Family   Family
}
