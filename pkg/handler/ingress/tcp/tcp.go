package tcp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/common/addr"
	scontext "github.com/HT4w5/l4rt/pkg/common/context"
	"github.com/HT4w5/l4rt/pkg/common/stream"
	"github.com/HT4w5/l4rt/pkg/handler"
	"github.com/rs/zerolog"
)

type TCPIngressConfig interface {
	handler.HandlerConfig
	Listen() netip.AddrPort
	Next() string
}

// TCPIngress listens for TCP connections.
//
// TCPIngress implements [github.com/HT4w5/l4rt/pkg/handler.IngressHandler].
type TCPIngress struct {
	cfg struct {
		tag    string
		next   string
		listen netip.AddrPort
	}

	deps struct {
		ctxr   scontext.ContextRenter
		next   handler.ByteStreamHandler
		logger zerolog.Logger
	}

	stats struct {
		accepted atomic.Int64
	}
}

func BuildTCPIngress(cfg TCPIngressConfig, deps handler.HandlerDeps) (*TCPIngress, error) {
	logger, err := deps.LoggerGetter.GetLogger(cfg.LogConfig(), "handler/"+cfg.Tag())
	if err != nil {
		return nil, fmt.Errorf("ingress.BuildTCPIngress: failed to get logger: %w", err)
	}
	h := new(TCPIngress)

	h.cfg.tag = cfg.Tag()
	h.cfg.next = cfg.Next()
	h.cfg.listen = cfg.Listen()

	h.deps.ctxr = deps.ContextRenter
	h.deps.logger = logger
	return h, nil
}

func (ig *TCPIngress) Tag() string {
	return ig.cfg.tag
}

func (ig *TCPIngress) Stats() map[string]any {
	return map[string]any{
		"accepted": ig.stats.accepted.Load(),
	}
}

func (ig *TCPIngress) Start(ctx context.Context) error {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", ig.cfg.listen.String())
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		ig.stats.accepted.Add(1)
		go ig.handleConn(ctx, conn)
	}
}

func (ig *TCPIngress) handleConn(ctx context.Context, conn net.Conn) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sctx := ig.deps.ctxr.Rent(streamCtx)

	sctx.HandlerStack = append(sctx.HandlerStack, ig.cfg.tag)

	if srcAddr, err := addr.FromAddrPort(conn.RemoteAddr().(*net.TCPAddr).AddrPort(), addr.ProtoTCP); err != nil {
		ig.deps.logger.Error().Err(err).Msg("invalid source address")
		conn.Close()
		return
	} else {
		sctx.Src = srcAddr
	}
	if dstAddr, err := addr.FromAddrPort(conn.LocalAddr().(*net.TCPAddr).AddrPort(), addr.ProtoTCP); err != nil {
		ig.deps.logger.Error().Err(err).Msg("invalid destination address")
		conn.Close()
		return
	} else {
		sctx.Dst = dstAddr
	}

	tcpConn := conn.(*net.TCPConn)

	ig.deps.next.HandleStream(sctx, stream.NewTCPStream(tcpConn))
}
