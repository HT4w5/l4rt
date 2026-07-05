package ingress

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync"

	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/node"
	"github.com/HT4w5/l4rt/pkg/node/request"
	tcpopts "github.com/HT4w5/l4rt/pkg/transport/tcp"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/HT4w5/l4rt/pkg/utils/idc"
	"github.com/rs/zerolog"
)

type Config interface {
	node.Config
	Listen() netip.AddrPort
	NextTag() string
	TCP() tcpopts.Config
}

type TCPIngress struct {
	deps struct {
		logger       zerolog.Logger
		next         node.StreamHandler
		listenConfig *net.ListenConfig
	}
	listener net.Listener
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      struct {
		tag     string
		name    string
		listen  netip.AddrPort
		nextTag string
	}
}

func NewTCPIngress(cfg Config, loggerGetter log.Getter) (*TCPIngress, error) {
	ti := &TCPIngress{}

	ti.cfg.tag = cfg.Tag()
	ti.cfg.listen = cfg.Listen()
	ti.cfg.nextTag = cfg.NextTag()
	ti.cfg.name = "endpoint/tcp/ingress:" + ti.cfg.tag

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewTCPIngress: %w", err)
	} else {
		ti.deps.logger = l.With().Stringer(log.KeyNode, ti).Logger()
	}

	var err error
	ti.deps.listenConfig, err = tcpopts.NewListenConfig(cfg.TCP())
	if err != nil {
		return nil, fmt.Errorf("NewTCPIngress: failed to create ListenConfig: %w", err)
	}

	return ti, nil
}

// implement [node.Node]
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

// implement [node.Worker]
func (ti *TCPIngress) Start(ctx context.Context) error {
	listener, err := ti.deps.listenConfig.Listen(ctx, "tcp", ti.cfg.listen.String())
	if err != nil {
		ti.deps.logger.Error().Stack().Err(err).Stringer("listen", ti.cfg.listen).Msg("listen failed")
		return fmt.Errorf("Start: listen failed: %w", err)
	}
	ti.listener = listener
	return nil
}

func (ti *TCPIngress) Run(ctx context.Context) error {
	ti.ctx, ti.cancel = context.WithCancel(ctx)

	var wg sync.WaitGroup

	// watchdog
	go func() {
		<-ti.ctx.Done()
		if err := ti.listener.Close(); err != nil {
			ti.deps.logger.Warn().Err(err).Msg("failed to close TCP listener")
		}
	}()

	for {
		conn, err := ti.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			ti.deps.logger.Warn().Err(err).Msg("accept error")
			continue
		}

		wg.Go(func() {
			ti.handleTCPConn(conn.(*net.TCPConn))
		})
	}

	wg.Wait()
	return nil
}

func (ti *TCPIngress) handleTCPConn(conn *net.TCPConn) {
	srcAddrPort := conn.RemoteAddr().(*net.TCPAddr).AddrPort()

	req := request.Stream{
		Conn: conn,
		Metadata: request.StreamMetadata{
			SrcAddr: addr.Addr{
				IPAddr:   srcAddrPort.Addr(),
				MuxIndex: srcAddrPort.Port(),
				Family:   addr.FamilyTCP,
			},
			DstAddr: addr.Unknown,
		},
		ID: idc.Get(),
	}

	ti.deps.logger.Info().Object(log.KeyRequest, &req).Msg("stream request created")

	if err := ti.deps.next.HandleStream(ti.ctx, &req); err != nil {
		ti.deps.logger.Warn().Object(log.KeyRequest, &req).Err(err).Msg("stream request failed")
	}
}

func (ti *TCPIngress) Stop(ctx context.Context) error {
	ti.cancel()
	return nil
}
