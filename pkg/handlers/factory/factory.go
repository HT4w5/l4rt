package factory

import (
	"fmt"

	"github.com/HT4w5/l4rt/pkg/handlers"
	egress_tcp "github.com/HT4w5/l4rt/pkg/handlers/egress/tcp"
	egress_udp "github.com/HT4w5/l4rt/pkg/handlers/egress/udp"
	ingress_tcp "github.com/HT4w5/l4rt/pkg/handlers/ingress/tcp"
	ingress_udp "github.com/HT4w5/l4rt/pkg/handlers/ingress/udp"
)

type HandlerFactory struct {
	deps handlers.HandlerDeps
}

func NewHandlerFactory(deps handlers.HandlerDeps) *HandlerFactory {
	return &HandlerFactory{
		deps: deps,
	}
}

// Build constructs a Handler from the given config using the registered builder.
func (hf *HandlerFactory) Build(cfg handlers.HandlerConfig) (handlers.Handler, error) {
	switch c := cfg.(type) {
	// Ingress handlers
	case ingress_tcp.TCPIngressConfig:
		return ingress_tcp.BuildTCPIngress(c, hf.deps)
	case ingress_udp.UDPIngressConfig:
		return ingress_udp.BuildUDPIngress(c, hf.deps)
	// Egress handlers
	case egress_tcp.TCPEgressConfig:
		return egress_tcp.BuildTCPEgress(c, hf.deps)
	case egress_udp.UDPEgressConfig:
		return egress_udp.BuildUDPEgress(c, hf.deps)
	default:
		return nil, fmt.Errorf("factory.HandlerFactory.Build: unsupported config type %T", cfg)
	}
}
