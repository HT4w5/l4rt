package core

import (
	mctx "github.com/HT4w5/l4rt/pkg/modules/context"
	"github.com/HT4w5/l4rt/pkg/modules/handler"
	"github.com/HT4w5/l4rt/pkg/modules/log"
)

type Config interface {
	LogFactoryConfig() log.FactoryConfig
	HandlerFactoryConfig() handler.FactoryConfig
	ContextManagerConfig() mctx.ManagerConfig
	HandlerConfigs() []any
}

func ProvideLogFactoryConfig(cfg Config) log.FactoryConfig {
	return cfg.LogFactoryConfig()
}

func ProvideHandlerFactoryConfig(cfg Config) handler.FactoryConfig {
	return cfg.HandlerFactoryConfig()
}

func ProvideContextManagerConfig(cfg Config) mctx.ManagerConfig {
	return cfg.ContextManagerConfig()
}
