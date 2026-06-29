package tcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/nodes/node"
	tcpopts "github.com/HT4w5/l4rt/pkg/transport/tcp"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	uctx "github.com/HT4w5/l4rt/pkg/utils/context"
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
	}

	ctxPool sync.Pool

	stats struct {
		accepted atomic.Int64
	}
}

func NewTCPIngress(cfg Config, loggerGetter log.Getter) (*TCPIngress, error) {
	n := &TCPIngress{}

	n.cfg.tag = cfg.Tag()
	n.cfg.listen = cfg.Listen()
	n.cfg.nextTag = cfg.NextTag()
	n.cfg.name = "ingress/tcp:" + n.cfg.tag

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewTCPIngress: %w", err)
	} else {
		n.deps.logger = l.With().Stringer(log.Node, n).Logger()
	}

	var err error
	n.deps.listenConfig, err = tcpopts.NewListenConfig(cfg.TCP())
	if err != nil {
		return nil, fmt.Errorf("NewTCPIngress: failed to create ListenConfig: %w", err)
	}

	n.ctxPool = sync.Pool{
		New: func() any {
			return &TCPConnCtx{}
		},
	}

	return n, nil
}

func (n *TCPIngress) Tag() string { return n.cfg.tag }

func (n *TCPIngress) String() string {
	return n.cfg.name
}

func (n *TCPIngress) InjectHandlers(inject func(tag string) (node.Node, error)) error {
	next, err := inject(n.cfg.nextTag)
	if err != nil {
		return err
	}

	var ok bool
	n.deps.next, ok = next.(node.StreamHandler)
	if !ok {
		return fmt.Errorf("InjectHandlers: node %q does not implement %T", next.Tag(), n.deps.next)
	}

	return nil
}

func (n *TCPIngress) Handlers() []node.Node {
	return []node.Node{n.deps.next}
}

func (n *TCPIngress) Start(ctx context.Context) error {
	listener, err := n.deps.listenConfig.Listen(ctx, "tcp", n.cfg.listen.String())
	if err != nil {
		return fmt.Errorf("Start: listen failed: %w", err)
	}
	n.pool.listener = listener

	go func() {
		for {
			conn, err := n.pool.listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				n.deps.logger.Error().Stack().Err(err).Msg("accept error")
				return
			}
			n.stats.accepted.Add(1)
			n.pool.Go(func() {
				n.handleTCPConn(conn.(*net.TCPConn))
			})
		}
	}()

	return nil
}

func (n *TCPIngress) handleTCPConn(conn *net.TCPConn) {
	srcAddrPort := conn.RemoteAddr().(*net.TCPAddr).AddrPort()

	srcAddr := addr.Addr{
		IPAddr:   srcAddrPort.Addr(),
		MuxIndex: srcAddrPort.Port(),
		Family:   addr.FamilyTCP,
	}

	ctx := n.ctxPool.Get().(*TCPConnCtx)
	defer n.ctxPool.Put(ctx)

	ctx.Init(uctx.GetID(), conn, &srcAddr, &addr.Unknown)

	n.deps.logger.Info().EmbedObject(ctx).Msg("stream created")

	if err := n.deps.next.HandleStream(ctx); err != nil {
		n.deps.logger.Warn().EmbedObject(ctx).Err(err).Msg("stream failed")
	}
}
