package egress

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/arena"
	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/node"
	"github.com/HT4w5/l4rt/pkg/node/request"
	tcpopts "github.com/HT4w5/l4rt/pkg/transport/tcp"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/HT4w5/l4rt/pkg/utils/iox"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type Config interface {
	node.Config
	FixedRaddr() (netip.AddrPort, bool)
	BindLaddr() (netip.AddrPort, bool)
	BufferSize() int
	TCP() tcpopts.Config
}

type TCPEgress struct {
	stats struct {
		rx atomic.Int64
		tx atomic.Int64
	}

	cfg struct {
		tag           string
		name          string
		hasFixedLaddr bool
		fixedLaddr    netip.AddrPort
		bufferSize    int
	}

	deps struct {
		logger zerolog.Logger
		arena  arena.Arena
		dialer *net.Dialer
	}
}

func NewTCPEgress(cfg Config, loggerGetter log.Getter, arena arena.Arena) (*TCPEgress, error) {
	te := &TCPEgress{}

	te.cfg.tag = cfg.Tag()
	te.cfg.name = "endpoint/tcp/egress:" + te.cfg.tag
	if addr, ok := cfg.BindLaddr(); ok {
		te.cfg.hasFixedLaddr = true
		te.cfg.fixedLaddr = addr
	}
	te.cfg.bufferSize = cfg.BufferSize()

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewTCPEgress: %w", err)
	} else {
		te.deps.logger = l.With().Stringer(log.KeyNode, te).Logger()
	}

	te.deps.arena = arena
	var err error
	te.deps.dialer, err = tcpopts.NewDialer(cfg.TCP())
	if err != nil {
		return nil, fmt.Errorf("NewTCPEgress: failed to create dialer: %w", err)
	}

	return te, nil
}

func (te *TCPEgress) Tag() string { return te.cfg.tag }

func (te *TCPEgress) String() string {
	return te.cfg.name
}

func (te *TCPEgress) HandleStream(ctx context.Context, req *request.Stream) error {
	logger := te.deps.logger.With().Object(log.KeyRequest, req).Logger()

	if req.Metadata.DstAddr.Family != addr.FamilyTCP {
		return addr.ErrFamilyNotSupported
	}
	raddr := netip.AddrPortFrom(req.Metadata.DstAddr.IPAddr, req.Metadata.DstAddr.MuxIndex)

	var laddr netip.AddrPort
	if te.cfg.hasFixedLaddr {
		laddr = te.cfg.fixedLaddr
	}

	conn, err := te.deps.dialer.DialTCP(ctx, "tcp", laddr, raddr)
	if err != nil {
		logger.Warn().Err(err).Stringer("laddr", laddr).Stringer("raddr", raddr).Msg("dial failed")
		return err
	}
	defer conn.Close()

	id := req.ID
	inBuf, err := te.deps.arena.Get(id, te.cfg.bufferSize)
	if err != nil {
		logger.Warn().Err(err).Int("size", te.cfg.bufferSize).Msg("failed to get buffer from arena")
		return err
	}
	defer te.deps.arena.Put(id, inBuf)

	outBuf, err := te.deps.arena.Get(id, te.cfg.bufferSize)
	if err != nil {
		logger.Warn().Err(err).Int("size", te.cfg.bufferSize).Msg("failed to get buffer from arena")
		return err
	}
	defer te.deps.arena.Put(id, outBuf)

	eg, egCtx := errgroup.WithContext(ctx)

	// watchdog
	eg.Go(func() error {
		<-egCtx.Done()
		conn.Close()
		req.Conn.Close()
		return nil
	})

	// Local -> Remote
	eg.Go(func() error {
		n, err := io.CopyBuffer(conn, req.Conn, outBuf)
		te.stats.tx.Add(n)
		if err != nil {
			logger.Warn().Err(err).Msg("local -> remote: copy error")
			return err
		}
		if err := conn.CloseWrite(); err != nil {
			logger.Warn().Err(err).Msg("local -> remote: failed to close write")
		}
		return nil
	})

	// Remote -> Local
	eg.Go(func() error {
		n, err := io.CopyBuffer(req.Conn, conn, inBuf)
		te.stats.rx.Add(n)
		if err != nil {
			logger.Warn().Err(err).Msg("remote -> local: copy error")
			return err
		}
		wc, ok := request.GetCapability[iox.WriteCloser](req)
		if ok {
			if err := wc.CloseWrite(); err != nil {
				logger.Warn().Err(err).Msg("remote -> local: failed to close write")
			}
		} else {
			logger.Warn().Msg("upstream connection does not support close write; falling back to full close")
			if err := req.Conn.Close(); err != nil {
				logger.Warn().Err(err).Msg("remote -> local: failed to close connection")
			}
		}
		return nil
	})

	return eg.Wait()
}
