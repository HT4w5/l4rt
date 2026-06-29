package tcp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/arena"
	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/nodes/node"
	uctx "github.com/HT4w5/l4rt/pkg/utils/context"
	"github.com/HT4w5/l4rt/pkg/utils/iox"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Config interface {
	node.Config
	FixedRaddr() (netip.AddrPort, bool)
	BindLaddr() (netip.AddrPort, bool)
	BufferSize() int
}

type TCPEgress struct {
	stats struct {
		rx atomic.Int64
		tx atomic.Int64
	}

	cfg struct {
		tag           string
		name          string
		hasFixedRaddr bool
		fixedRaddr    netip.AddrPort
		hasBindLaddr  bool
		bindLaddr     netip.AddrPort
		bufferSize    int
	}

	deps struct {
		logger zerolog.Logger
		arena  arena.Arena
		dialer net.Dialer
	}
}

func NewTCPEgress(cfg Config, loggerGetter log.Getter, arena arena.Arena) (*TCPEgress, error) {
	te := &TCPEgress{}

	te.cfg.tag = cfg.Tag()
	te.cfg.name = "egress/tcp:" + te.cfg.tag
	if addr, ok := cfg.FixedRaddr(); ok {
		te.cfg.hasFixedRaddr = true
		te.cfg.fixedRaddr = addr
	}
	if addr, ok := cfg.BindLaddr(); ok {
		te.cfg.hasBindLaddr = true
		te.cfg.bindLaddr = addr
	}
	te.cfg.bufferSize = cfg.BufferSize()

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewTCPEgress: %w", err)
	} else {
		te.deps.logger = l.With().Stringer(log.Node, te).Logger()
	}

	te.deps.arena = arena
	te.deps.dialer = net.Dialer{} // TODO: add options

	return te, nil
}

func (te *TCPEgress) Tag() string { return te.cfg.tag }

func (te *TCPEgress) String() string {
	return te.cfg.name
}

func (te *TCPEgress) HandleStream(ctx uctx.StreamCtx) error {
	var raddr netip.AddrPort
	if te.cfg.hasFixedRaddr {
		raddr = te.cfg.fixedRaddr
	} else {
		dstAddr := ctx.DstAddr()
		raddr = netip.AddrPortFrom(dstAddr.IPAddr, dstAddr.MuxIndex)
	}

	var laddr netip.AddrPort
	if te.cfg.hasBindLaddr {
		laddr = te.cfg.bindLaddr
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	conn, err := te.deps.dialer.DialTCP(context.Background(), "tcp", laddr, raddr)
	if err != nil {
		te.deps.logger.Warn().Err(err).Stringer("laddr", laddr).Stringer("raddr", raddr).Msg("dial failed")
		return err
	}

	id := ctx.ID()
	inBuf, err := te.deps.arena.Get(id, te.cfg.bufferSize)
	if err != nil {
		return err
	}
	defer te.deps.arena.Put(id, inBuf)

	outBuf, err := te.deps.arena.Get(id, te.cfg.bufferSize)
	if err != nil {
		return err
	}
	defer te.deps.arena.Put(id, outBuf)

	eg := new(errgroup.Group)

	// Stream -> TCPConn
	eg.Go(func() (err error) {
		var n int64
		n, err = iox.CopyStreamToConn(ctx, conn, outBuf)
		te.stats.tx.Add(n)
		conn.CloseWrite()

		if err != nil {
			te.deps.logger.Warn().EmbedObject(ctx).Err(err).Send()
		}
		return
	})

	eg.Go(func() (err error) {
		var n int64
		n, err = iox.CopyConnToStream(conn, ctx, inBuf)
		te.stats.rx.Add(n)
		ctx.CloseWrite()

		if err != nil {
			te.deps.logger.Warn().EmbedObject(ctx).Err(err).Send()
		}
		return
	})

	return eg.Wait()
}
