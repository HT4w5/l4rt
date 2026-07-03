package request

import (
	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/rs/zerolog"
)

type Packet struct {
	Metadata PacketMetadata
	P        []byte
	ID       uint64
}

type PacketMetadata struct {
	SrcAddr addr.Addr
	DstAddr addr.Addr
}

func (req *Packet) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64(log.KeyRequestID, req.ID).
		Str(log.KeyRequestType, "packet").
		Stringer(log.KeyAddrSrc, &req.Metadata.SrcAddr).
		Stringer(log.KeyAddrDst, &req.Metadata.DstAddr)
}
