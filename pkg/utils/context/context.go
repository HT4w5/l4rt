package context

import (
	"io"
	"time"
)

type CtxCommon interface {
	ID() uint64
	Done() bool
	Deadline() (time.Time, bool)
	SetDeadline(t time.Time) error
}

type StreamCtx interface {
	CtxCommon
	io.ReadWriteCloser
}

type PacketCtx interface {
	CtxCommon
	Packet() []byte
}
