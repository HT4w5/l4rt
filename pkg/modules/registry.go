package modules

import (
	"context"
	"fmt"
	"reflect"
)

// BuildFunc builds a module from an untyped config.
type BuildFunc func(ctx context.Context, cfg any) (Module, error)

var buildFuncRegistry = map[reflect.Type]BuildFunc{}

// RegisterModule registers a module by its config type interface or struct.
func RegisterModule(cfgInterfaceOrStruct any, f BuildFunc) error {
	var configType reflect.Type

	t := reflect.TypeOf(cfgInterfaceOrStruct)
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Interface {
		configType = t.Elem()
	} else {
		configType = t
	}

	if _, ok := buildFuncRegistry[configType]; ok {
		return fmt.Errorf("RegisterModule: %s already registered", configType.String())
	}

	buildFuncRegistry[configType] = f
	return nil
}

// BuildModule resolves module type from config and builds the module.
func BuildModule(ctx context.Context, cfg any) (Module, error) {
	concreteType := reflect.TypeOf(cfg)

	if f, ok := buildFuncRegistry[concreteType]; ok {
		return f(ctx, cfg)
	}

	for registeredType, f := range buildFuncRegistry {
		if registeredType.Kind() == reflect.Interface && concreteType.Implements(registeredType) {
			return f(ctx, cfg)
		}
	}

	return nil, fmt.Errorf("BuildModule: no registered handler matches %s", concreteType.String())
}
