package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
)

type FactoryConfig interface {
	PullInterval() time.Duration
	BufferSize() int
	TimeFormat() string
}

// NewFactory creates a new log.Factory from the given config.
func NewFactory(cfg FactoryConfig) (*Factory, error) {
	zerolog.TimeFieldFormat = cfg.TimeFormat()
	f := &Factory{}
	f.cfg.pullInterval = cfg.PullInterval()
	f.cfg.bufferSize = cfg.BufferSize()
	f.state.outputMap = make(map[string]io.Writer)
	return f, nil
}

type Factory struct {
	cfg struct {
		pullInterval time.Duration
		bufferSize   int
	}

	state struct {
		outputMap map[string]io.Writer
	}
}

func (f *Factory) GetLogger(cfg Config, module string) (zerolog.Logger, error) {
	output := cfg.Output()

	var w io.Writer
	if cfg.Level() == zerolog.Disabled {
		w = io.Discard
	} else if dw, ok := f.state.outputMap[output]; ok {
		w = dw
	} else {
		switch output {
		case "":
			fallthrough
		case "stderr":
			w = diode.NewWriter(os.Stderr, f.cfg.bufferSize, f.cfg.pullInterval, nil)
		case "stdout":
			w = diode.NewWriter(os.Stdout, f.cfg.bufferSize, f.cfg.pullInterval, nil)
		default:
			file, err := os.OpenFile(
				output,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0600,
			)
			if err != nil {
				return zerolog.Logger{}, fmt.Errorf("log.LoggerManager.GetLogger: failed to open log file: %w", err)
			}

			w = diode.NewWriter(file, f.cfg.bufferSize, f.cfg.pullInterval, nil)
		}
		f.state.outputMap[output] = w
	}

	loggerCtx := zerolog.New(w).Level(cfg.Level()).With().Str("module", module)

	if cfg.AddTimestamp() {
		loggerCtx = loggerCtx.Timestamp()
	}
	if cfg.AddCaller() {
		loggerCtx = loggerCtx.Caller()
	}
	return loggerCtx.Logger(), nil
}
