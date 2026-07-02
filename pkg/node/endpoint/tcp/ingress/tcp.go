package ingress

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/node"
	"github.com/HT4w5/l4rt/pkg/node/request"
	tcpopts "github.com/HT4w5/l4rt/pkg/transport/tcp"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/HT4w5/l4rt/pkg/utils/id"
	"github.com/rs/zerolog"
)

type Config interface {
	node.Config
	Listen() netip.AddrPort
	NextTag() string
	TCP() tcpopts.Config
}

// Implements [node.ActiveNode], [node.Dispatcher]
type TCPIngress struct {
	cfg struct {
		tag     string
		name    string
		listen  netip.AddrPort
		nextTag string
	}

	deps struct {
		logger       zerolog.Logger
		next         node.StreamHandler
		listenConfig *net.ListenConfig
	}

	pool struct {
		sync.WaitGroup
		listener net.Listener
		ctx      context.Context
		cancel   context.CancelFunc
	}

	stats struct {
		accepted atomic.Int64
	}
}

func NewTCPIngress(cfg Config, loggerGetter log.Getter) (*TCPIngress, error) {
	n := &TCPIngress{}

	n.cfg.tag = cfg.Tag()
	n.cfg.listen = cfg.Listen()
	n.cfg.nextTag = cfg.NextTag()
	n.cfg.name = "endpoint/tcp/ingress:" + n.cfg.tag

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewTCPIngress: %w", err)
	} else {
		n.deps.logger = l.With().Stringer(log.KeyNode, n).Logger()
	}

	var err error
	n.deps.listenConfig, err = tcpopts.NewListenConfig(cfg.TCP())
	if err != nil {
		return nil, fmt.Errorf("NewTCPIngress: failed to create ListenConfig: %w", err)
	}

	n.pool.ctx, n.pool.cancel = context.WithCancel(context.Background())

	return n, nil
}

func (ti *TCPIngress) Tag() string { return ti.cfg.tag }

func (ti *TCPIngress) String() string {
	return ti.cfg.name
}

func (ti *TCPIngress) InjectHandlers(inject func(tag string) (node.Node, error)) error {
	next, err := inject(ti.cfg.nextTag)
	if err != nil {
		return err
	}

	var ok bool
	ti.deps.next, ok = next.(node.StreamHandler)
	if !ok {
		return fmt.Errorf("InjectHandlers: node %q does not implement %T", next.Tag(), ti.deps.next)
	}

	return nil
}

func (ti *TCPIngress) Handlers() []node.Node {
	return []node.Node{ti.deps.next}
}

func (ti *TCPIngress) Start(ctx context.Context) error {
	listener, err := ti.deps.listenConfig.Listen(ctx, "tcp", ti.cfg.listen.String())
	if err != nil {
		ti.deps.logger.Error().Stack().Err(err).Stringer("listen", ti.cfg.listen).Msg("listen failed")
		return fmt.Errorf("Start: listen failed: %w", err)
	}
	ti.pool.listener = listener

	go func() {
		for {
			conn, err := ti.pool.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				ti.deps.logger.Error().Stack().Err(err).Msg("accept error")
				return
			}
			ti.stats.accepted.Add(1)
			ti.pool.Go(func() {
				ti.handleTCPConn(conn.(*net.TCPConn))
			})
		}
	}()

	return nil
}

func (ti *TCPIngress) handleTCPConn(conn *net.TCPConn) {
	srcAddrPort := conn.RemoteAddr().(*net.TCPAddr).AddrPort()

	srcAddr := addr.Addr{
		IPAddr:   srcAddrPort.Addr(),
		MuxIndex: srcAddrPort.Port(),
		Family:   addr.FamilyTCP,
	}

	req := request.Stream{
		Conn: conn,
		Metadata: request.StreamMetadata{
			SrcAddr: srcAddr,
			DstAddr: addr.Unknown,
		},
		ID: id.Get(),
	}

	ti.deps.logger.Info().Object(log.KeyRequest, &req).Msg("stream request created")

	if err := ti.deps.next.HandleStream(ti.pool.ctx, &req); err != nil {
		ti.deps.logger.Warn().Object(log.KeyRequest, &req).Err(err).Msg("stream request failed")
	}
}

func (ti *TCPIngress) Stop(ctx context.Context) error {
	if err := ti.pool.listener.Close(); err != nil {
		ti.deps.logger.Warn().Err(err).Msg("failed to close listener")
	}

	ti.pool.cancel()

	done := make(chan struct{})

	go func() {
		ti.pool.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
