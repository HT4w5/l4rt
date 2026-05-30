package config

import (
	"fmt"
	"net/netip"

	"github.com/HT4w5/l4rt/pkg/utils/log"
)

// UDPIngressConfig implements [github.com/HT4w5/l4rt/pkg/handler/ingress/udp.UDPIngressConfig]
type UDPIngressConfig struct {
	Tag_       string     `json:"tag"`
	Listen_    string     `json:"listen"`
	Next_      string     `json:"next"`
	LogConfig_ *LogConfig `json:"log"`

	tag    string
	listen netip.AddrPort
	next   string
}

func (cfg *UDPIngressConfig) Validate() error {
	if cfg.LogConfig_ == nil {
		cfg.LogConfig_ = new(LogConfig)
	}
	if err := cfg.LogConfig_.Validate(); err != nil {
		return fmt.Errorf("UDPIngressConfig: %w", err)
	}

	if cfg.Tag_ == "" {
		return fmt.Errorf("UDPIngressConfig: empty tag")
	} else {
		cfg.tag = cfg.Tag_
	}

	if ap, err := netip.ParseAddrPort(cfg.Listen_); err != nil {
		return fmt.Errorf("UDPIngressConfig: invalid listen: %w", err)
	} else {
		cfg.listen = ap
	}

	if cfg.Next_ == "" {
		return fmt.Errorf("UDPIngressConfig: empty next")
	} else {
		cfg.next = cfg.Next_
	}

	return nil
}

func (cfg *UDPIngressConfig) LogConfig() log.Config {
	return cfg.LogConfig_
}

func (cfg *UDPIngressConfig) Tag() string {
	return cfg.tag
}

func (cfg *UDPIngressConfig) Listen() netip.AddrPort {
	return cfg.listen
}

func (cfg *UDPIngressConfig) Next() string {
	return cfg.next
}
