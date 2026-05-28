package stream

import (
	"time"

	"github.com/HT4w5/l4rt/pkg/common/addr"
)

type PacketStream interface {
	ReadFrom(p []byte) (n int, addr addr.Addr, err error)
	WriteTo(p []byte, addr addr.Addr) (n int, err error)
	Close() error
	SetDeadline(t time.Time) error
}
