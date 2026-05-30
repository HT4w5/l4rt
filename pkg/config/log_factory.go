package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type LogFactoryConfig struct {
	PullInterval_ string `json:"pull_interval"`
	BufferSize_   int    `json:"buffer_size"`
	TimeFormat_   string `json:"time_format"`

	pullInterval time.Duration
	bufferSize   int
	timeFormat   string
}

func (cfg *LogFactoryConfig) Validate() error {
	// pull_interval
	if cfg.PullInterval_ == "" {
		cfg.pullInterval = 10 * time.Millisecond
	} else if d, err := time.ParseDuration(cfg.PullInterval_); err != nil {
		return fmt.Errorf("LogFactoryConfig: invalid pull_interval: %w", err)
	} else {
		cfg.pullInterval = d
	}

	// buffer_size
	if cfg.BufferSize_ < 0 {
		return fmt.Errorf("LogFactoryConfig: buffer_size must be non-negative")
	}
	if cfg.BufferSize_ == 0 {
		cfg.bufferSize = 1000
	} else {
		cfg.bufferSize = cfg.BufferSize_
	}

	// time_format
	switch strings.ToLower(cfg.TimeFormat_) {
	case "":
		fallthrough
	case "unix":
		cfg.timeFormat = zerolog.TimeFormatUnix
	case "unixms":
		cfg.timeFormat = zerolog.TimeFormatUnixMs
	case "unixmicro":
		cfg.timeFormat = zerolog.TimeFormatUnixMicro
	case "unixnano":
		cfg.timeFormat = zerolog.TimeFormatUnixNano
	default:
		return fmt.Errorf("LogFactoryConfig: invalid time_format: %q", cfg.timeFormat)
	}

	return nil
}

func (cfg *LogFactoryConfig) PullInterval() time.Duration {
	return cfg.pullInterval
}

func (cfg *LogFactoryConfig) BufferSize() int {
	return cfg.bufferSize
}

func (cfg *LogFactoryConfig) TimeFormat() string {
	return cfg.timeFormat
}
