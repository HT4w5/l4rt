package log

import (
	"github.com/rs/zerolog"
)

type Config interface {
	Level() zerolog.Level
	Output() string
	AddCaller() bool
	AddTimestamp() bool
}

type Getter interface {
	GetLogger(cfg Config, module string) (zerolog.Logger, error)
}
