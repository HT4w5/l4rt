package tcp

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	uctx "github.com/HT4w5/l4rt/pkg/utils/context"
	"github.com/rs/zerolog"
)

type TCPConnCtx struct {
	conn             *net.TCPConn
	id               uint64
	deadlineUnixNano atomic.Int64
	canceled         atomic.Bool
	srcAddr          *addr.Addr
	dstAddr          *addr.Addr
}

func (ctx *TCPConnCtx) Init(id uint64, conn *net.TCPConn, srcAddr *addr.Addr, dstAddr *addr.Addr) {
	ctx.conn = conn
	ctx.id = id
	ctx.deadlineUnixNano.Store(0)
	ctx.canceled.Store(false)
	ctx.srcAddr = srcAddr
	ctx.dstAddr = dstAddr
}

func (ctx *TCPConnCtx) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64(log.ID, ctx.id).
		Stringer(log.SourceAddr, ctx.srcAddr).
		Stringer(log.DestAddr, ctx.dstAddr)
}

func (ctx *TCPConnCtx) ID() uint64 {
	return ctx.id
}

func (ctx *TCPConnCtx) Err() error {
	if ctx.canceled.Load() {
		return uctx.ErrCanceled
	}

	ddl := ctx.deadlineUnixNano.Load()
	if ddl != 0 && ddl < time.Now().UnixNano() {
		return uctx.ErrDeadlineExceeded
	}

	return nil
}

func (ctx *TCPConnCtx) Deadline() (ddl time.Time, ok bool) {
	t := ctx.deadlineUnixNano.Load()
	if t == 0 {
		return
	}

	ddl = time.Unix(0, t)
	ok = true
	return
}

func (ctx *TCPConnCtx) SetDeadline(t time.Time) error {
	ctx.deadlineUnixNano.Store(t.UnixNano())
	return ctx.conn.SetDeadline(t)
}

func (ctx *TCPConnCtx) Cancel() error {
	ctx.canceled.Store(true)
	return ctx.conn.Close()
}

func (ctx *TCPConnCtx) Read(p []byte) (n int, err error) {
	return ctx.conn.Read(p)
}

func (ctx *TCPConnCtx) Write(p []byte) (n int, err error) {
	return ctx.conn.Write(p)
}

func (ctx *TCPConnCtx) CloseWrite() error {
	return ctx.conn.CloseWrite()
}

func (ctx *TCPConnCtx) SetSrcAddr(addr *addr.Addr) {
	ctx.srcAddr = addr
}

func (ctx *TCPConnCtx) SrcAddr() *addr.Addr {
	return ctx.srcAddr
}

func (ctx *TCPConnCtx) SetDstAddr(addr *addr.Addr) {
	ctx.dstAddr = addr
}

func (ctx *TCPConnCtx) DstAddr() *addr.Addr {
	return ctx.dstAddr
}
