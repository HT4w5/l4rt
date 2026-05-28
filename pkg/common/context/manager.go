package context

import (
	"context"
	"sync"
	"sync/atomic"
)

type ContextRenter interface {
	Rent(parent context.Context) *Context
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
	kv := cm.kvPool.Get().(map[uint64]uint64)
	stack := cm.stackPool.Get().([]string)
	return &Context{
		Ctx:          parent,
		KV:           kv,
		HandlerStack: stack,
		Release: func() {
			if len(kv) <= cm.kvSize {
				clear(kv)
				cm.kvPool.Put(kv)
			}
			if cap(stack) == cm.stackSize {
				cm.stackPool.Put(stack)
			}
		},
		ID: cm.idCounter.Add(1),
	}
}
