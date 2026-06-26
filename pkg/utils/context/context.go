package context

import (
	"errors"
	"net/netip"
	"time"

	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/rs/zerolog"
)

var (
	ErrCanceled         = errors.New("context canceled")
	ErrDeadlineExceeded = errors.New("context deadline exceeded")
)

// Methods of commonCtx must be concurrent-safe.
type commonCtx interface {
	zerolog.LogObjectMarshaler
	ID() uint64
	Cancel() error
	Err() error
	Deadline() (time.Time, bool)
	SetDeadline(t time.Time) error
}

type StreamCtx interface {
	commonCtx
	// Data
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	CloseWrite() error
	// Metadata
	SetSrcAddr(addr *addr.Addr)
	SrcAddr() *addr.Addr
	SetDstAddr(addr *addr.Addr)
	DstAddr() *addr.Addr
}

type PacketCtx interface {
	commonCtx
	Packet() []byte
}

type ResolveCtx interface {
	commonCtx
	FQDN() []byte
	SetIPAddr(addr netip.Addr) error
	IPAddr() netip.Addr
}
