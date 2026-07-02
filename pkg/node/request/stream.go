package request

import (
	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/HT4w5/l4rt/pkg/utils/iox"
	"github.com/rs/zerolog"
)

type Stream struct {
	Conn     iox.Conn
	Metadata StreamMetadata
	ID       uint64
}

type StreamMetadata struct {
	SrcAddr addr.Addr
	DstAddr addr.Addr
}

func (req *Stream) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64(log.KeyRequestID, req.ID).
		Str(log.KeyRequestType, "stream").
		Stringer(log.KeyAddrSrc, &req.Metadata.SrcAddr).
		Stringer(log.KeyAddrDst, &req.Metadata.DstAddr)
}

func GetCapability[T any](req *Stream) (cap T, ok bool) {
	conn := req.Conn
	for {
		if val, ok := conn.(T); ok {
			return val, true
		}

		if unwrapper, ok := conn.(iox.Unwrapper); ok {
			conn = unwrapper.Unwrap()
		} else {
			break
		}
	}
	return
}
