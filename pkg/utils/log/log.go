package log

import (
	"github.com/rs/zerolog"
)

type Config interface {
	Level() zerolog.Level
	Location() string
}
