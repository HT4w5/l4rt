package factory

import (
	"fmt"

	"github.com/HT4w5/l4rt/pkg/handler"
	egress_tcp "github.com/HT4w5/l4rt/pkg/handler/egress/tcp"
	ingress_tcp "github.com/HT4w5/l4rt/pkg/handler/ingress/tcp"
)

type HandlerFactory struct {
	deps handler.HandlerDeps
}

func NewHandlerFactory(deps handler.HandlerDeps) *HandlerFactory {
	return &HandlerFactory{
		deps: deps,
	}
}

// Build constructs a Handler from the given config using the registered builder.
func (hf *HandlerFactory) Build(cfg handler.HandlerConfig) (handler.Handler, error) {
	switch c := cfg.(type) {
	// Ingress handlers
	case ingress_tcp.TCPIngressConfig:
		return ingress_tcp.BuildTCPIngress(c, hf.deps)
	// Egress handlers
	case egress_tcp.TCPEgressConfig:
		return egress_tcp.BuildTCPEgress(c, hf.deps)
	default:
		return nil, fmt.Errorf("factory.HandlerFactory.Build: unsupported config type %T", cfg)
	}
}
