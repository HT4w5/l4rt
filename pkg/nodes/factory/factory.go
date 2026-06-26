package factory

import (
	"fmt"
	"sync"

	"github.com/HT4w5/l4rt/pkg/nodes/node"
)

var (
	mu       sync.RWMutex
	fallback func(node.Config) (node.Node, error)
)

// For mock testing
func SetFallback(fn func(node.Config) (node.Node, error)) {
	mu.Lock()
	defer mu.Unlock()
	fallback = fn
}

func NewNode(cfg node.Config) (node.Node, error) {
	switch v := cfg.(type) {
	default:
		mu.RLock()
		fn := fallback
		mu.RUnlock()
		if fn != nil {
			return fn(cfg)
		}
		return nil, fmt.Errorf("NewNode: node %q: unknown config type: %T", v.Tag(), v)
	}
}
