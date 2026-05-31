package jsonutils

import (
	"encoding/json"
	"fmt"
)

type JSONRouterMapping map[string]func() any

// JSONRouter unmarshals json.RawMessage into target types according to value of specified key.
type JSONRouter struct {
	mapping JSONRouterMapping
	key     string
}

func NewJSONRouter(mapping JSONRouterMapping, key string) *JSONRouter {
	return &JSONRouter{
		mapping: mapping,
		key:     key,
	}
}

func (jr *JSONRouter) Unmarshal(msg json.RawMessage) (obj any, value string, err error) {
	var peek map[string]json.RawMessage
	if err = json.Unmarshal(msg, &peek); err != nil {
		return
	}

	rawValue, ok := peek[jr.key]
	if !ok {
		err = fmt.Errorf("JSONRouter.Unmarshal: %q not in JSON context", jr.key)
		return
	}

	if err = json.Unmarshal(rawValue, &value); err != nil {
		return
	}

	creator, ok := jr.mapping[value]
	if !ok {
		err = fmt.Errorf("JSONRouter.Unmarshal: unknown mapping string %q", value)
		return
	}
	obj = creator()

	err = json.Unmarshal(msg, obj)
	return
}
