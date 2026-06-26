package node

import (
	"context"
	"fmt"

	uctx "github.com/HT4w5/l4rt/pkg/utils/context"
)

type Config interface {
	Tag() string
}

type Node interface {
	fmt.Stringer
	Tag() string
}

type ActiveNode interface {
	Node
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type StreamHandler interface {
	Node
	HandleStream(ctx uctx.StreamCtx) error
}

type PacketHandler interface {
	Node
	HandlePacket(ctx uctx.PacketCtx) error
}

type ResolveHandler interface {
	Node
	HandleResolve(ctx uctx.ResolveCtx) error
}

type Dispatcher interface {
	Node
	InjectHandlers(func(tag string) (Node, error)) error
	Handlers() []Node
}
