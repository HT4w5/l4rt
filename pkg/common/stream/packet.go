package stream

import (
	"time"

	"github.com/HT4w5/l4rt/pkg/common/addr"
)

type PacketReader interface {
	// A second call invalidates the previous returned payload slice.
	ReadPacket() (b []byte, src addr.Addr, err error)
}

type PacketWriter interface {
	WritePacket(b []byte, dst addr.Addr) error
}

type PacketStream interface {
	PacketReader
	PacketWriter
	Close() error
	SetDeadline(t time.Time) error
}
