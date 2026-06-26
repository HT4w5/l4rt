package log

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/pkgerrors"
)

func init() {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}

type Getter interface {
	GetLogger(cfg Config) (zerolog.Logger, error)
}

type Config interface {
	Level() zerolog.Level
	Output() string
}

type FactoryConfig interface {
	PullInterval() time.Duration
	BufferSize() int
}

type Factory struct {
	cfg struct {
		pullInterval time.Duration
		bufferSize   int
	}

	state struct {
		sync.Mutex
		outputMap map[string]io.Writer
		closers   []io.Closer
	}
}

func NewFactory(cfg FactoryConfig) (*Factory, func()) {
	f := &Factory{}
	f.cfg.pullInterval = cfg.PullInterval()
	f.cfg.bufferSize = cfg.BufferSize()
	f.state.outputMap = make(map[string]io.Writer)
	return f, func() {
		f.state.Lock()
		defer f.state.Unlock()
		for _, closer := range f.state.closers {
			closer.Close()
		}
	}
}

func (f *Factory) GetLogger(cfg Config) (zerolog.Logger, error) {
	f.state.Lock()
	defer f.state.Unlock()

	output := cfg.Output()

	var w io.Writer
	if cfg.Level() == zerolog.Disabled {
		return zerolog.Nop(), nil
	} else if dw, ok := f.state.outputMap[output]; ok {
		w = dw
	} else {
		switch output {
		case "":
			fallthrough
		case "stderr":
			w = os.Stderr
		case "stdout":
			w = os.Stdout
		default:
			file, err := os.OpenFile(
				output,
				os.O_APPEND|os.O_CREATE|os.O_WRONLY,
				0600,
			)
			if err != nil {
				return zerolog.Logger{}, fmt.Errorf("GetLogger: failed to open log file: %w", err)
			}

			dw := diode.NewWriter(file, f.cfg.bufferSize, f.cfg.pullInterval, nil)

			f.state.closers = append(f.state.closers, dw)
			f.state.closers = append(f.state.closers, file)

			w = dw
		}
		f.state.outputMap[output] = w
	}

	loggerCtx := zerolog.New(w).Level(cfg.Level()).With()

	if cfg.Level() == zerolog.DebugLevel {
		loggerCtx = loggerCtx.Caller()
	}

	return loggerCtx.Logger(), nil
}
