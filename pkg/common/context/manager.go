package context

import (
	"context"
	"sync"
	"sync/atomic"
)

type ContextRenter interface {
	Rent(parent context.Context) *Context
	Release(ctx *Context)
}

type ContextManager struct {
	kvPool    sync.Pool
	stackPool sync.Pool
	idCounter atomic.Uint64
	kvSize    int
	stackSize int
}

func NewContextManager(kvSize int, stackSize int) *ContextManager {
	return &ContextManager{
		kvPool: sync.Pool{
			New: func() any { return make(map[uint64]uint64, kvSize) },
		},
		stackPool: sync.Pool{
			New: func() any { return make([]string, 0, stackSize) },
		},
		kvSize:    kvSize,
		stackSize: stackSize,
	}
}

func (cm *ContextManager) Rent(parent context.Context) *Context {
	return &Context{
		Ctx:          parent,
		KV:           cm.kvPool.Get().(map[uint64]uint64),
		HandlerStack: cm.stackPool.Get().([]string),
		ID:           cm.idCounter.Add(1),
	}
}

func (cm *ContextManager) Release(c *Context) {
	if len(c.KV) <= cm.kvSize {
		clear(c.KV)
		cm.kvPool.Put(c.KV)
	}
	if cap(c.HandlerStack) == cm.stackSize {
		cm.stackPool.Put(c.HandlerStack)
	}
}
