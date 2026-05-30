package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	scontext "github.com/HT4w5/l4rt/pkg/common/context"
	"github.com/HT4w5/l4rt/pkg/common/stream"
	"github.com/HT4w5/l4rt/pkg/handlers"
	"github.com/rs/zerolog"
)

type UDPIngressConfig interface {
	handlers.HandlerConfig
	Listen() netip.AddrPort
	Next() string
}

func BuildUDPIngress(cfg UDPIngressConfig, deps handlers.HandlerDeps) (*UDPIngress, error) {
	logger, err := deps.LoggerGetter.GetLogger(cfg.LogConfig(), "handler/"+cfg.Tag())
	if err != nil {
		return nil, fmt.Errorf("BuildUDPIngress: failed to get logger: %w", err)
	}
	h := new(UDPIngress)

	h.cfg.tag = cfg.Tag()
	h.cfg.listen = cfg.Listen()
	h.cfg.next = cfg.Next()

	h.deps.ctxr = deps.ContextRenter
	h.deps.logger = logger

	h.conn.done = make(chan struct{})

	return h, nil
}

// UDPIngress listens for UDP packets.
type UDPIngress struct {
	cfg struct {
		tag    string
		next   string
		listen netip.AddrPort
	}

	deps struct {
		ctxr   scontext.ContextRenter
		next   handlers.PacketHandler
		logger zerolog.Logger
	}

	conn struct {
		net.PacketConn
		closeOnce sync.Once
		done      chan struct{}
	}
}

// Implement Handler

func (ui *UDPIngress) Tag() string {
	return ui.cfg.tag
}

func (ui *UDPIngress) Stats() map[string]any {
	return map[string]any{}
}

// Implement Wirer

func (ui *UDPIngress) Wire(getHandler handlers.WireFunc) error {
	h, ok := getHandler(ui.cfg.next)
	if !ok {
		return fmt.Errorf("UDPIngress.Wire: no handler with tag %q", ui.cfg.next)
	}
	ph, ok := h.(handlers.PacketHandler)
	if !ok {
		return fmt.Errorf("UDPIngress.Wire: expected %q to be StreamHandler, got %T", ui.cfg.next, h)
	}
	ui.deps.next = ph
	return nil
}

// Implement IngressHandler

func (ui *UDPIngress) Start(ctx context.Context) error {
	defer close(ui.conn.done)
	lc := net.ListenConfig{}
	var err error
	ui.conn.PacketConn, err = lc.ListenPacket(ctx, "udp", ui.cfg.listen.String())
	if err != nil {
		ui.deps.logger.Err(err).Str("listen", ui.cfg.listen.String()).Msg("UDP listen failure")
		return err
	}

	go func() {
		<-ctx.Done()
		ui.conn.closeOnce.Do(func() {
			if ui.conn.PacketConn != nil {
				ui.conn.Close()
			}
		})
	}()

	sctx := ui.deps.ctxr.Rent(ctx)
	defer ui.deps.ctxr.Release(sctx)

	sctx.HandlerStack = append(sctx.HandlerStack, ui.cfg.tag)
	sctx.IsPacket = true

	err = ui.deps.next.HandlePacket(sctx, stream.NewUDPPacketStream(ui.conn.PacketConn.(*net.UDPConn)))
	if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
		return nil
	}
	return err
}

func (ui *UDPIngress) Shutdown(ctx context.Context) error {
	ui.conn.closeOnce.Do(func() {
		if ui.conn.PacketConn != nil {
			ui.conn.Close()
		}
	})
	select {
	case <-ui.conn.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
