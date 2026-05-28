package context

import (
	"context"

	"github.com/HT4w5/l4rt/pkg/common/addr"
)

type Context struct {
	Ctx          context.Context
	KV           map[uint64]uint64
	Release      func()
	HandlerStack []string
	Src          addr.Addr
	Dst          addr.Addr
	ID           uint64
	IsPacket     bool
}
