package handler

import "github.com/HT4w5/l4rt/pkg/handlers"

type Builder interface {
	Build(cfg handlers.HandlerConfig) (handlers.Handler, error)
}
