package context

import (
	"context"

	scontext "github.com/HT4w5/l4rt/pkg/common/context"
)

type Renter interface {
	Rent(parent context.Context) *scontext.Context
	Release(ctx *scontext.Context)
}
