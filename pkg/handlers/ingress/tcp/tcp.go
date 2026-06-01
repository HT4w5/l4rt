package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/common/addr"
	"github.com/HT4w5/l4rt/pkg/common/stream"
	"github.com/HT4w5/l4rt/pkg/handlers"
	mctx "github.com/HT4w5/l4rt/pkg/modules/context"
	"github.com/HT4w5/l4rt/pkg/modules/log"
	"github.com/rs/zerolog"
)

type TCPIngressConfig interface {
	handlers.HandlerConfig
	Listen() netip.AddrPort
	Next() string
	IsTCPIngressConfig() // For interface uniqueness
}

func BuildTCPIngress(cfg TCPIngressConfig, contextRenter mctx.Renter, loggerGetter log.Getter) (*TCPIngress, error) {
	logger, err := loggerGetter.GetLogger(cfg.LogConfig(), "handler/"+cfg.Tag())
	if err != nil {
		return nil, fmt.Errorf("BuildTCPIngress: failed to get logger: %w", err)
	}
	h := new(TCPIngress)

	h.cfg.tag = cfg.Tag()
	h.cfg.next = cfg.Next()
	h.cfg.listen = cfg.Listen()

	h.deps.ctxr = contextRenter
	h.deps.logger = logger

	return h, nil
}

// TCPIngress listens for TCP connections.
//
// TCPIngress implements [github.com/HT4w5/l4rt/pkg/handlers.IngressHandler].
type TCPIngress struct {
	cfg struct {
		tag    string
		next   string
		listen netip.AddrPort
	}

	deps struct {
		ctxr   mctx.Renter
		next   handlers.StreamHandler
		logger zerolog.Logger
	}

	pool struct {
		sync.WaitGroup
		listener  net.Listener
		closeOnce sync.Once
	}

	stats struct {
		accepted atomic.Int64
	}
}

// Implement Handler

func (ig *TCPIngress) Tag() string {
	return ig.cfg.tag
}

func (ig *TCPIngress) Stats() map[string]any {
	return map[string]any{
		"accepted": ig.stats.accepted.Load(),
	}
}

// Implement Wirer

func (ig *TCPIngress) Wire(getHandler handlers.WireFunc) error {
	h, ok := getHandler(ig.cfg.next)
	if !ok {
		return fmt.Errorf("TCPIngress.Wire: no handler with tag %q", ig.cfg.next)
	}
	sh, ok := h.(handlers.StreamHandler)
	if !ok {
		return fmt.Errorf("TCPIngress.Wire: expected %q to be StreamHandler, got %T", ig.cfg.next, h)
	}
	ig.deps.next = sh
	return nil
}

// Implement IngressHandler

func (ig *TCPIngress) Start(ctx context.Context) error {
	lc := net.ListenConfig{}
	var err error
	ig.pool.listener, err = lc.Listen(ctx, "tcp", ig.cfg.listen.String())
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		ig.pool.closeOnce.Do(func() {
			if ig.pool.listener != nil {
				ig.pool.listener.Close()
			}
		})
	}()

	for {
		conn, err := ig.pool.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			return err
		}
		ig.stats.accepted.Add(1)
		ig.pool.Go(func() {
			ig.handleConn(ctx, conn)
		})
	}
}

func (ig *TCPIngress) handleConn(ctx context.Context, conn net.Conn) {
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sctx := ig.deps.ctxr.Rent(streamCtx)
	defer ig.deps.ctxr.Release(sctx)

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

func (ig *TCPIngress) Shutdown(ctx context.Context) error {
	ig.pool.closeOnce.Do(func() {
		if ig.pool.listener != nil {
			ig.pool.listener.Close()
		}
	})
	done := make(chan struct{})
	go func() {
		ig.pool.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
