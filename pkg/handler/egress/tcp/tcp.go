package tcp

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/HT4w5/l4rt/pkg/common/addr"
	scontext "github.com/HT4w5/l4rt/pkg/common/context"
	"github.com/HT4w5/l4rt/pkg/common/stream"
	"github.com/HT4w5/l4rt/pkg/handler"
	"github.com/HT4w5/l4rt/pkg/utils/pipe"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

var (
	ErrNotTCPIPAddress = errors.New("not a TCP/IP address")
)

type TCPEgressConfig interface {
	handler.HandlerConfig
	DialTimeout() time.Duration
}

// TCPEgress forwards a byte stream to a target tcp address.
//
// TCPEgress implements [github.com/HT4w5/l4rt/pkg/handler.ByteStreamHandler].
type TCPEgress struct {
	cfg struct {
		tag string
	}

	deps struct {
		dialer net.Dialer
		logger zerolog.Logger
	}

	stats struct {
		rx      atomic.Int64
		tx      atomic.Int64
		handled atomic.Int64
	}
}

func BuildTCPEgress(cfg TCPEgressConfig, deps handler.HandlerDeps) (*TCPEgress, error) {
	logger, err := deps.LoggerGetter.GetLogger(cfg.LogConfig(), "")
	if err != nil {
		return nil, fmt.Errorf("egress.BuildTCPEgress: failed to get logger: %w", err)
	}
	h := new(TCPEgress)

	h.cfg.tag = cfg.Tag()

	h.deps.dialer = net.Dialer{
		Timeout: cfg.DialTimeout(), // TODO: general dialer config
	}

	h.deps.logger = logger

	return h, nil
}

func (h *TCPEgress) Tag() string {
	return h.cfg.tag
}

func (h *TCPEgress) Stats() map[string]any {
	return map[string]any{
		"rx":      h.stats.rx.Load(),
		"tx":      h.stats.tx.Load(),
		"handled": h.stats.handled.Load(),
	}
}

func (h *TCPEgress) HandleStream(ctx *scontext.Context, s stream.ByteStream) error {
	h.stats.handled.Add(1)
	// Dst must be a valid TCP/IP address
	if !ctx.Dst.IsIPAddr() || ctx.Dst.Proto != addr.ProtoTCP {
		return ErrNotTCPIPAddress
	}

	// TODO: dial options (timeout, redirect, laddr, etc.)
	conn, err := h.deps.dialer.DialContext(ctx.Ctx, "tcp", ctx.Dst.String())
	if err != nil {
		return fmt.Errorf("dial %s: %w", ctx.Dst.URI(), err)
	}

	eg := new(errgroup.Group)

	eg.Go(func() error {
		n, err := pipe.Copy(s, conn) // From remote to local stream
		if n > 0 {
			h.stats.rx.Add(n)
		}
		s.CloseWrite()
		return err
	})

	eg.Go(func() error {
		n, err := pipe.Copy(conn, s) // From local stream to remote
		if n > 0 {
			h.stats.tx.Add(n)
		}
		conn.(*net.TCPConn).CloseWrite()
		return err
	})

	return eg.Wait()
}
