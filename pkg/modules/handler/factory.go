package handler

import (
	"fmt"

	"github.com/HT4w5/l4rt/pkg/handlers"
	mctx "github.com/HT4w5/l4rt/pkg/modules/context"
	"github.com/HT4w5/l4rt/pkg/modules/log"

	egress_tcp "github.com/HT4w5/l4rt/pkg/handlers/egress/tcp"
	egress_udp "github.com/HT4w5/l4rt/pkg/handlers/egress/udp"
	ingress_tcp "github.com/HT4w5/l4rt/pkg/handlers/ingress/tcp"
	ingress_udp "github.com/HT4w5/l4rt/pkg/handlers/ingress/udp"
)

type FactoryConfig interface {
}

type Factory struct {
	deps struct {
		loggerGetter  log.Getter
		contextRenter mctx.Renter
	}
}

func NewFactory(cfg FactoryConfig, loggerGetter log.Getter, contextRenter mctx.Renter) *Factory {
	f := &Factory{}

	f.deps.loggerGetter = loggerGetter
	f.deps.contextRenter = contextRenter

	return f
}

func (f *Factory) Build(cfg handlers.HandlerConfig) (handlers.Handler, error) {
	switch c := cfg.(type) {
	// Ingress handlers
	case ingress_tcp.TCPIngressConfig:
		return ingress_tcp.BuildTCPIngress(c, f.deps.contextRenter, f.deps.loggerGetter)
	case ingress_udp.UDPIngressConfig:
		return ingress_udp.BuildUDPIngress(c, f.deps.contextRenter, f.deps.loggerGetter)
	// Egress handlers
	case egress_tcp.TCPEgressConfig:
		return egress_tcp.BuildTCPEgress(c, f.deps.loggerGetter)
	case egress_udp.UDPEgressConfig:
		return egress_udp.BuildUDPEgress(c, f.deps.loggerGetter)
	default:
		return nil, fmt.Errorf("factory.HandlerFactory.Build: unsupported config type %T", cfg)
	}
}
