package config

import (
	"fmt"
	"net/netip"

	"github.com/HT4w5/l4rt/pkg/modules/log"
)

// TCPIngressConfig implements [github.com/HT4w5/l4rt/pkg/handler/ingress/tcp.TCPIngressConfig]
type TCPIngressConfig struct {
	Tag_       string     `json:"tag"`
	Listen_    string     `json:"listen"`
	Next_      string     `json:"next"`
	LogConfig_ *LogConfig `json:"log"`

	tag    string
	listen netip.AddrPort
	next   string
}

func (cfg *TCPIngressConfig) Validate() error {
	if cfg.LogConfig_ == nil {
		cfg.LogConfig_ = new(LogConfig)
	}
	if err := cfg.LogConfig_.Validate(); err != nil {
		return fmt.Errorf("TCPIngressConfig: %w", err)
	}

	if cfg.Tag_ == "" {
		return fmt.Errorf("TCPIngressConfig: empty tag")
	} else {
		cfg.tag = cfg.Tag_
	}

	if ap, err := netip.ParseAddrPort(cfg.Listen_); err != nil {
		return fmt.Errorf("TCPIngressConfig: invalid listen: %w", err)
	} else {
		cfg.listen = ap
	}

	if cfg.Next_ == "" {
		return fmt.Errorf("TCPIngressConfig: empty tag")
	} else {
		cfg.next = cfg.Next_
	}

	return nil
}

func (cfg *TCPIngressConfig) LogConfig() log.Config {
	return cfg.LogConfig_
}

func (cfg *TCPIngressConfig) Tag() string {
	return cfg.tag
}

func (cfg *TCPIngressConfig) Listen() netip.AddrPort {
	return cfg.listen
}

func (cfg *TCPIngressConfig) Next() string {
	return cfg.next
}

func (cfg *TCPIngressConfig) IsTCPIngressConfig() {}
