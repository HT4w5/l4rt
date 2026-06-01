package config

import (
	"fmt"
	"time"

	"github.com/HT4w5/l4rt/pkg/modules/log"
)

// TCPEgressConfig implements [github.com/HT4w5/l4rt/pkg/handler/egress/tcp.TCPEgressConfig]
type TCPEgressConfig struct {
	Tag_         string     `json:"tag"`
	LogConfig_   *LogConfig `json:"log"`
	DialTimeout_ string     `json:"dial_timeout"`

	tag         string
	dialTimeout time.Duration
}

func (cfg *TCPEgressConfig) Validate() error {
	if cfg.LogConfig_ == nil {
		cfg.LogConfig_ = new(LogConfig)
	}
	if err := cfg.LogConfig_.Validate(); err != nil {
		return fmt.Errorf("TCPEgressConfig: %w", err)
	}

	if cfg.Tag_ == "" {
		return fmt.Errorf("TCPEgressConfig: empty tag")
	} else {
		cfg.tag = cfg.Tag_
	}

	if cfg.DialTimeout_ == "" {
		cfg.dialTimeout = 30 * time.Second
	} else if d, err := time.ParseDuration(cfg.DialTimeout_); err != nil {
		return fmt.Errorf("TCPEgressConfig: invalid dial_timeout: %w", err)
	} else {
		cfg.dialTimeout = d
	}

	return nil
}

func (cfg *TCPEgressConfig) LogConfig() log.Config {
	return cfg.LogConfig_
}

func (cfg *TCPEgressConfig) Tag() string {
	return cfg.tag
}

func (cfg *TCPEgressConfig) DialTimeout() time.Duration {
	return cfg.dialTimeout
}

func (cfg *TCPEgressConfig) IsTCPEgressConfig() {}
