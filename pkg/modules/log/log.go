package log

import (
	"github.com/HT4w5/l4rt/pkg/modules"
	"github.com/rs/zerolog"
)

type Config interface {
	Level() zerolog.Level
	Output() string
	AddCaller() bool
	AddTimestamp() bool
}

type LoggerGetter interface {
	modules.Module
	GetLogger(cfg Config, module string) (zerolog.Logger, error)
}
