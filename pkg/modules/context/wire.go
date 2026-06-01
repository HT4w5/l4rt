package context

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	NewManager,
	wire.Bind(new(Renter), new(*Manager)),
)
