package log

import (
	"github.com/google/wire"
	"github.com/rs/zerolog"
)

type GlobalLogConfig Config

func NewGlobalLogger(getter Getter, cfg GlobalLogConfig) (zerolog.Logger, error) {
	return getter.GetLogger(cfg)
}

var LogSet = wire.NewSet(
	NewFactory,
	wire.Bind(new(Getter), new(*Factory)),
	NewGlobalLogger,
)
