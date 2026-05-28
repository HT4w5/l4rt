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
	outputMap    map[string]io.Writer
	pullInterval time.Duration
	bufferSize   int
}

func NewLoggerManager(bufferSize int, pullInterval time.Duration) *LoggerManager {
	return &LoggerManager{
		outputMap:    make(map[string]io.Writer),
		bufferSize:   bufferSize,
		pullInterval: pullInterval,
	}
}

func (lm *LoggerManager) GetLogger(cfg Config, module string) (zerolog.Logger, error) {
	output := cfg.Output()

	var w io.Writer
	if cfg.Level() == zerolog.Disabled {
		w = io.Discard
	} else if dw, ok := lm.outputMap[output]; ok {
		w = dw
	} else {
		switch output {
		case "":
			fallthrough
		case "stderr":
			w = diode.NewWriter(os.Stderr, lm.bufferSize, lm.pullInterval, nil)
		case "stdout":
			w = diode.NewWriter(os.Stdout, lm.bufferSize, lm.pullInterval, nil)
		default:
			f, err := os.OpenFile(
				output,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0600,
			)
			if err != nil {
				return zerolog.Logger{}, fmt.Errorf("log.LoggerManager.GetLogger: failed to open log file: %w", err)
			}

			w = diode.NewWriter(f, lm.bufferSize, lm.pullInterval, nil)
		}
		lm.outputMap[output] = w
	}

	return zerolog.New(w).Level(cfg.Level()).With().Timestamp().Str("module", module).Logger(), nil
}
