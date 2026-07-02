package stream

import (
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/rs/zerolog"
)

type Stream interface {
	zerolog.LogObjectMarshaler
	ID() uint64
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	CloseWrite() error
	Metadata() *StreamMetadata
}

type StreamMetadata struct {
	SrcAddr addr.Addr
	DstAddr addr.Addr
}
