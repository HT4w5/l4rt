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

// Worker represents nodes with long-running routines.
type Worker interface {
	Node
	// Start initializes the node.
	Start(ctx context.Context) error
	// Run is the node's main routine.
	// Run is expected to block until Stop is called and downstream jobs finish.
	Run(ctx context.Context) error
	// Stop signals node shutdown.
	Stop(ctx context.Context) error
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
