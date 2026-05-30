package config

import (
	"fmt"

	"github.com/HT4w5/l4rt/pkg/utils/log"
)

// UDPEgressConfig implements [github.com/HT4w5/l4rt/pkg/handler/egress/udp.UDPEgressConfig]
type UDPEgressConfig struct {
	Tag_       string     `json:"tag"`
	LogConfig_ *LogConfig `json:"log"`

	tag string
}

func (cfg *UDPEgressConfig) Validate() error {
	if cfg.LogConfig_ == nil {
		cfg.LogConfig_ = new(LogConfig)
	}
	if err := cfg.LogConfig_.Validate(); err != nil {
		return fmt.Errorf("UDPEgressConfig: %w", err)
	}

	if cfg.Tag_ == "" {
		return fmt.Errorf("UDPEgressConfig: empty tag")
	} else {
		cfg.tag = cfg.Tag_
	}

	return nil
}

func (cfg *UDPEgressConfig) LogConfig() log.Config {
	return cfg.LogConfig_
}

func (cfg *UDPEgressConfig) Tag() string {
	return cfg.tag
}
