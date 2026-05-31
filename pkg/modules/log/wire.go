package log

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewFactory,
	wire.Bind(new(Getter), new(Factory)),
)
