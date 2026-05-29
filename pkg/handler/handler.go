package handler

import (
	"context"

	scontext "github.com/HT4w5/l4rt/pkg/common/context"
	"github.com/HT4w5/l4rt/pkg/utils/log"

	"github.com/HT4w5/l4rt/pkg/common/stream"
)

// Handler is the basic pipeline unit.
type Handler interface {
	Tag() string
	Stats() map[string]any
}

// StreamHandler handles one byte stream.
type StreamHandler interface {
	Handler
	HandleStream(ctx *scontext.Context, s stream.ByteStream) error
}

// PacketHandler handles one packet stream.
type PacketHandler interface {
	Handler
	HandlePacket(ctx *scontext.Context, s stream.PacketStream) error
}

// IngressHandler handles incoming external streams.
type IngressHandler interface {
	Handler
	// Start starts ingress handling, blocks during handle, and stops when context expires.
	Start(ctx context.Context) error
	// Shutdown stops handling, waits for existing routines to finish or context to expire.
	Shutdown(ctx context.Context) error
}

type WireFunc func(tag string) (Handler, bool)

// Wireable can be wired into pipeline by calling Wire.
type Wireable interface {
	Handler
	// Wire registers handlers returned by WireFunc into Wireable's deps.
	Wire(getHandler WireFunc) error
}

type HandlerDeps struct {
	LoggerGetter  log.LoggerGetter
	ContextRenter scontext.ContextRenter
}

type HandlerConfig interface {
	Tag() string
	LogConfig() log.Config
}
