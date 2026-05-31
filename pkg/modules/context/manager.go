package context

import (
	"context"
	"sync"
	"sync/atomic"

	scontext "github.com/HT4w5/l4rt/pkg/common/context"
)

type ManagerConfig interface {
	KVSize() int
	StackSize() int
}

// Manager implements [Renter]
type Manager struct {
	cfg struct {
		kvSize    int
		stackSize int
	}

	state struct {
		kvPool    sync.Pool
		stackPool sync.Pool
		idCounter atomic.Uint64
	}
}

func NewManager(cfg ManagerConfig) *Manager {
	mgr := &Manager{}

	mgr.cfg.kvSize = cfg.KVSize()
	mgr.cfg.stackSize = cfg.StackSize()

	mgr.state.kvPool = sync.Pool{
		New: func() any { return make(map[uint64]uint64, cfg.KVSize()) },
	}

	mgr.state.stackPool = sync.Pool{
		New: func() any { return make([]string, 0, cfg.StackSize()) },
	}

	return mgr
}

func (mgr *Manager) Rent(parent context.Context) *scontext.Context {
	return &scontext.Context{
		Ctx:          parent,
		KV:           mgr.state.kvPool.Get().(map[uint64]uint64),
		HandlerStack: mgr.state.stackPool.Get().([]string),
		ID:           mgr.state.idCounter.Add(1),
	}
}

func (mgr *Manager) Release(c *scontext.Context) {
	if len(c.KV) <= mgr.cfg.kvSize {
		clear(c.KV)
		mgr.state.kvPool.Put(c.KV)
	}
	if cap(c.HandlerStack) == mgr.cfg.stackSize {
		mgr.state.stackPool.Put(c.HandlerStack)
	}
}
