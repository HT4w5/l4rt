//go:build wireinject
// +build wireinject

package core

import (
	mctx "github.com/HT4w5/l4rt/pkg/modules/context"
	"github.com/HT4w5/l4rt/pkg/modules/handlers"
	"github.com/HT4w5/l4rt/pkg/modules/log"
	"github.com/google/wire"
)

func InitializeCore(cfg Config) (App, func(), error) {
	wire.Build(
		NewCore,
		wire.Bind(new(App), new(*Core)),

		// Config providers
		ProvideLogFactoryConfig,
		ProvideContextManagerConfig,
		ProvideHandlerFactoryConfig,

		// Module providers
		mctx.ProviderSet,
		handlers.ProviderSet,
		log.ProviderSet,
	)
	return nil, nil, nil
}
