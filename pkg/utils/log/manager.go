package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

type LoggerGetter interface {
	GetLogger(cfg Config, module string) (zerolog.Logger, error)
}

type LoggerManager struct {
	locationMap  map[string]io.Writer
	pullInterval time.Duration
	bufferSize   int
}

func NewLoggerManager(bufferSize int, pullInterval time.Duration) *LoggerManager {
	return &LoggerManager{
		locationMap:  make(map[string]io.Writer),
		bufferSize:   bufferSize,
		pullInterval: pullInterval,
	}
}

func (lm *LoggerManager) GetLogger(cfg Config, module string) (zerolog.Logger, error) {
	location := cfg.Location()
	if location == "" {
		location = "stderr"
	}

	var w io.Writer
	if cfg.Level() == zerolog.Disabled {
		w = io.Discard
	} else if dw, ok := lm.locationMap[location]; ok {
		w = dw
	} else {
		switch location {
		case "":
			fallthrough
		case "stderr":
			w = diode.NewWriter(os.Stderr, lm.bufferSize, lm.pullInterval, nil)
		case "stdout":
			w = diode.NewWriter(os.Stdout, lm.bufferSize, lm.pullInterval, nil)
		default:
			f, err := os.OpenFile(
				location,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0600,
			)
			if err != nil {
				return zerolog.Logger{}, fmt.Errorf("log.LoggerManager.GetLogger: failed to open log file: %w", err)
			}

			w = diode.NewWriter(f, lm.bufferSize, lm.pullInterval, nil)
		}
		lm.locationMap[location] = w
	}

	return zerolog.New(w).Level(cfg.Level()).With().Timestamp().Str("module", module).Logger(), nil
}
