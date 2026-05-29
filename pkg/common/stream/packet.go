package stream

import (
	"time"

	"github.com/HT4w5/l4rt/pkg/common/addr"
)

type PacketReader interface {
	ReadPacket() (p []byte, src addr.Addr, err error)
}

type PacketWriter interface {
	WritePacket(p []byte, dst addr.Addr) error
}

type PacketStream interface {
	PacketReader
	PacketWriter
	Close() error
	SetDeadline(t time.Time) error
}
