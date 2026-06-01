package core

import (
	"context"
	"sync"

	"github.com/HT4w5/l4rt/pkg/common"
	mctx "github.com/HT4w5/l4rt/pkg/modules/context"
	"github.com/HT4w5/l4rt/pkg/modules/handler"
	"github.com/HT4w5/l4rt/pkg/modules/log"
)

type App interface {
	common.RunnerStopper
}

type CoreState int8

const (
	StateNew CoreState = iota
	StateStarting
	StateRunning
	StateStopping
	StateStopped
)

func (s CoreState) String() string {
	return [...]string{"NEW", "STARTING", "RUNNING", "STOPPING", "STOPPED"}[s]
}

type Core struct {
	state struct {
		sync.RWMutex
		coreState CoreState
	}

	deps struct {
		contextRenter  mctx.Renter
		loggerGetter   log.Getter
		handlerBuilder handler.Builder
	}
}

func NewCore(
	cfg Config,
	contextRenter mctx.Renter,
	loggerGetter log.Getter,
	handlerBuilder handler.Builder,
) (*Core, error) {
	c := &Core{}

	c.deps.contextRenter = contextRenter
	c.deps.loggerGetter = loggerGetter
	c.deps.handlerBuilder = handlerBuilder

	return c, nil
}

func (c *Core) Run(ctx context.Context) error {
	return nil
}

func (c *Core) Stop(ctx context.Context) error {
	return nil
}
