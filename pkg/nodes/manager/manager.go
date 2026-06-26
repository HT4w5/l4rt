package manager

import (
	"fmt"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/nodes/factory"
	"github.com/HT4w5/l4rt/pkg/nodes/node"
	"github.com/HT4w5/l4rt/pkg/utils/queue"
	"github.com/rs/zerolog"
)

type Config interface {
	Nodes() []node.Config
}

type Manager struct {
	nodes struct {
		all          map[string]node.Node
		startupOrder []node.ActiveNode
	}

	deps struct {
		logger       zerolog.Logger
		loggerGetter log.Getter
	}
}

// create all nodes
// inject handlers to nodes
// build startup order
func NewManager(cfg Config, logger zerolog.Logger, loggerGetter log.Getter) (*Manager, error) {
	mgr := &Manager{}

	mgr.nodes.all = make(map[string]node.Node)

	mgr.deps.logger = logger.With().Str(log.Module, "NodeManager").Logger()
	mgr.deps.loggerGetter = loggerGetter

	for _, c := range cfg.Nodes() {
		h, err := factory.NewNode(c)
		if err != nil {
			return nil, fmt.Errorf("NewManager: failed to create handler: %w", err)
		}
		if _, ok := mgr.nodes.all[h.Tag()]; ok {
			return nil, fmt.Errorf("NewManager: handlers with duplicate tag: %q", h.Tag())
		}
		mgr.nodes.all[h.Tag()] = h
	}

	// Inject handlers to dispatchers
	for _, h := range mgr.nodes.all {
		if dp, ok := h.(node.Dispatcher); ok {
			if err := dp.InjectHandlers(func(tag string) (node.Node, error) {
				if h, ok := mgr.nodes.all[tag]; !ok {
					return nil, fmt.Errorf("no handler with tag %q exists", tag)
				} else {
					return h, nil
				}
			}); err != nil {
				return nil, fmt.Errorf("NewManager: failed to inject handlers for dispatcher %q: %w", dp.Tag(), err)
			}
		}
	}

	// Build dispatch dependency array
	inDegrees := make(map[string]int)
	depMap := make(map[string][]string) // dependency -> dependents

	for tag, h := range mgr.nodes.all {
		inDegrees[tag] = 0
		if dp, ok := h.(node.Dispatcher); ok {
			handlers := dp.Handlers()
			inDegrees[tag] += len(handlers)

			for _, hh := range handlers {
				deps := depMap[hh.Tag()]
				deps = append(deps, tag)
				depMap[hh.Tag()] = deps
			}
		}
	}

	fringe := queue.NewQueue[string]()

	for tag, inDegree := range inDegrees {
		if inDegree == 0 {
			fringe.Push(tag)
		}
	}

	var startupOrder []string
	for fringe.Len() != 0 {
		tag, _ := fringe.Pop()
		startupOrder = append(startupOrder, tag)

		for _, depTag := range depMap[tag] {
			inDegrees[depTag]--
			if inDegrees[depTag] == 0 {
				fringe.Push(depTag)
			}
		}
	}

	if len(mgr.nodes.all) != len(startupOrder) {
		// Have loop
		inStartupOrder := make(map[string]struct{})
		for _, tag := range startupOrder {
			inStartupOrder[tag] = struct{}{}
		}

		var loopNodes []string
		for tag := range mgr.nodes.all {
			if _, ok := inStartupOrder[tag]; !ok {
				loopNodes = append(loopNodes, tag)
			}
		}

		return nil, fmt.Errorf("NewManager: cycle or unreachable nodes detected: %v", loopNodes)
	}

	for _, tag := range startupOrder {
		h := mgr.nodes.all[tag]
		if ah, ok := h.(node.ActiveNode); ok {
			mgr.nodes.startupOrder = append(mgr.nodes.startupOrder, ah)
		}
	}

	return mgr, nil
}
