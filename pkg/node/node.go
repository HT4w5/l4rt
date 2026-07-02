package node

import (
	"context"
	"fmt"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/node/request"
)

type Config interface {
	Tag() string
	Log() log.Config
}

type Node interface {
	fmt.Stringer
	Tag() string
}

type Starter interface {
	Start(ctx context.Context) error
}

type Stopper interface {
	Stop(ctx context.Context) error
}

type ActiveNode interface {
	Node
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type StreamHandler interface {
	Node
	HandleStream(ctx context.Context, req *request.Stream) error
}

type PacketHandler interface {
	Node
	HandlePacket(ctx context.Context, req *request.Packet) error
}

type Dispatcher interface {
	Node
	InjectHandlers(func(tag string) (Node, error)) error
	Handlers() []Node
}
