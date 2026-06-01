package context

import (
	"context"

	cctx "github.com/HT4w5/l4rt/pkg/common/context"
)

type Renter interface {
	Rent(parent context.Context) *cctx.Context
	Release(ctx *cctx.Context)
}
