package stream

import (
	"net"
	"net/netip"
	"time"

	"github.com/HT4w5/l4rt/pkg/common/addr"
	"github.com/HT4w5/l4rt/pkg/common/constants"
)

type UDPPacketStream struct {
	conn *net.UDPConn
	buf  []byte
}

func NewUDPPacketStream(conn *net.UDPConn) *UDPPacketStream {
	return &UDPPacketStream{
		conn: conn,
		buf:  make([]byte, constants.MaxUDPPayloadSize),
	}
}

func (s *UDPPacketStream) ReadPacket() (b []byte, src addr.Addr, err error) {
	var n int
	var ap netip.AddrPort
	n, ap, err = s.conn.ReadFromUDPAddrPort(s.buf)
	if err != nil {
		return
	}

	src, err = addr.FromAddrPort(ap, addr.ProtoUDP)
	if err != nil {
		return
	}

	b = s.buf[:n]
	return
}

func (s *UDPPacketStream) WritePacket(b []byte, dst addr.Addr) error {
	ap, err := dst.AssertUDPIPAddr()
	if err != nil {
		return err
	}
	_, err = s.conn.WriteToUDPAddrPort(b, ap)
	if err != nil {
		return err
	}
	return nil
}

func (s *UDPPacketStream) Close() error {
	return s.conn.Close()
}

func (s *UDPPacketStream) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}
