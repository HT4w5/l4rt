package handler

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewFactory,
	wire.Bind(new(Builder), new(*Factory)),
)
