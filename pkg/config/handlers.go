package config

import (
	"encoding/json"

	"github.com/HT4w5/l4rt/pkg/config/jsonutils"
)

// handlerTypeMap contains mappings from handler type strings to config creators.
var handlerTypeMap = jsonutils.JSONRouterMapping{
	// Ingress
	"tcp-ingress": func() any { return new(TCPIngressConfig) },
	"udp-ingress": func() any { return new(UDPIngressConfig) },
	// Egress
	"tcp-egress": func() any { return new(TCPEgressConfig) },
	"udp-egress": func() any { return new(UDPEgressConfig) },
}

var handlerRouter = jsonutils.NewJSONRouter(handlerTypeMap, "type")

type HandlerList []any

func (hl *HandlerList) UnmarshalJSON(b []byte) error {
	var list []json.RawMessage
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}

	hlist := make([]any, 0, len(list))

	for _, msg := range list {
		obj, _, err := handlerRouter.Unmarshal(msg)
		if err != nil {
			return err
		}
		hlist = append(hlist, obj)
	}

	*hl = hlist
	return nil
}
