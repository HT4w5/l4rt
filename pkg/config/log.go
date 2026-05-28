package config

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// LogConfig implements [github.com/HT4w5/l4rt/pkg/utils/log.Config]
type LogConfig struct {
	Level_  string `json:"level"`
	Output_ string `json:"output"`

	level  zerolog.Level
	output string
}

func (cfg *LogConfig) Validate() error {
	switch strings.ToLower(cfg.Level_) {
	case "debug":
		cfg.level = zerolog.DebugLevel
	case "":
		fallthrough // default to "info"
	case "info":
		cfg.level = zerolog.InfoLevel
	case "warn":
		cfg.level = zerolog.WarnLevel
	case "error":
		cfg.level = zerolog.ErrorLevel
	case "fatal":
		cfg.level = zerolog.FatalLevel
	case "panic":
		cfg.level = zerolog.PanicLevel
	case "trace":
		cfg.level = zerolog.TraceLevel
	case "disabled":
		cfg.level = zerolog.Disabled
	default:
		return fmt.Errorf("LogConfig: unknown level %s", cfg.Level_)
	}

	switch strings.ToLower(cfg.Output_) {
	case "":
		fallthrough
	case "stderr":
		cfg.output = "stderr"
	case "stdout":
		cfg.output = "stdout"
	default:
		cfg.output = cfg.Output_
	}

	return nil
}

func (cfg *LogConfig) Level() zerolog.Level {
	return cfg.level
}

func (cfg *LogConfig) Output() string {
	return cfg.output
}
