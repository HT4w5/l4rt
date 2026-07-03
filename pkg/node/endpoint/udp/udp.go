package udp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"runtime"
	"sync"

	"github.com/HT4w5/l4rt/pkg/arena"
	"github.com/HT4w5/l4rt/pkg/log"
	"github.com/HT4w5/l4rt/pkg/node"
	"github.com/HT4w5/l4rt/pkg/node/request"
	udpopts "github.com/HT4w5/l4rt/pkg/transport/udp"
	"github.com/HT4w5/l4rt/pkg/utils/addr"
	"github.com/HT4w5/l4rt/pkg/utils/idc"
	"github.com/rs/zerolog"
)

type Config interface {
	node.Config
	Listen() netip.AddrPort
	NextTag() string
	UDP() udpopts.Config
	BufferSize() int
	NumWorkers() (n int, auto bool)
}

// Implements [node.ActiveNode], [node.Dispatcher]
type UDPEndpoint struct {
	deps struct {
		logger       zerolog.Logger
		next         node.PacketHandler
		listenConfig *net.ListenConfig
		arena        arena.Arena
	}
	workers struct {
		ctx        context.Context
		packetChan chan packet
		cancel     context.CancelFunc
		sync.WaitGroup
	}
	conn *net.UDPConn
	cfg  struct {
		tag        string
		name       string
		listen     netip.AddrPort
		nextTag    string
		bufferSize int
		numWorkers int
	}
}

func NewUDPEndpoint(cfg Config, loggerGetter log.Getter, arena arena.Arena) (*UDPEndpoint, error) {
	ue := &UDPEndpoint{}

	ue.cfg.tag = cfg.Tag()
	ue.cfg.listen = cfg.Listen()
	ue.cfg.nextTag = cfg.NextTag()
	ue.cfg.name = "endpoint/udp:" + ue.cfg.tag

	if n, auto := cfg.NumWorkers(); auto {
		ue.cfg.numWorkers = runtime.NumCPU()
	} else {
		if n <= 0 {
			return nil, errors.New("NewUDPEndpoint: numWorkers must be greater than 0")
		}
		ue.cfg.numWorkers = n
	}

	if l, err := loggerGetter.GetLogger(cfg.Log()); err != nil {
		return nil, fmt.Errorf("NewUDPEndpoint: %w", err)
	} else {
		ue.deps.logger = l.With().Stringer(log.KeyNode, ue).Logger()
	}

	var err error
	ue.deps.listenConfig, err = udpopts.NewListenConfig(cfg.UDP())
	if err != nil {
		return nil, fmt.Errorf("NewUDPEndpoint: failed to create ListenConfig: %w", err)
	}

	ue.deps.arena = arena

	return ue, nil
}

func (ue *UDPEndpoint) Tag() string { return ue.cfg.tag }

func (ue *UDPEndpoint) String() string {
	return ue.cfg.name
}

func (ue *UDPEndpoint) InjectHandlers(inject func(tag string) (node.Node, error)) error {
	next, err := inject(ue.cfg.nextTag)
	if err != nil {
		return err
	}

	var ok bool
	ue.deps.next, ok = next.(node.PacketHandler)
	if !ok {
		return fmt.Errorf("InjectHandlers: node %q does not implement %T", next.Tag(), ue.deps.next)
	}

	return nil
}

func (ue *UDPEndpoint) Handlers() []node.Node {
	return []node.Node{ue.deps.next}
}

func (ue *UDPEndpoint) Start(ctx context.Context) error {
	// Start workers
	ue.workers.packetChan = make(chan packet)
	ue.workers.ctx, ue.workers.cancel = context.WithCancel(context.Background())

	for range ue.cfg.numWorkers {
		ue.workers.Go(ue.handleUDPPacket)
	}

	conn, err := ue.deps.listenConfig.ListenPacket(ctx, "udp", ue.cfg.listen.String())
	if err != nil {
		ue.deps.logger.Error().Stack().Err(err).Stringer("listen", ue.cfg.listen).Msg("listen failed")
		return fmt.Errorf("Start: listen failed: %w", err)
	}

	ue.conn = conn.(*net.UDPConn)

	currentID := idc.Get()
	currentBuf, err := ue.deps.arena.Get(currentID, ue.cfg.bufferSize)
	if err != nil {
		ue.deps.logger.Error().Stack().Err(err).Msg("initial buffer allocation failed")
		return err
	}

	ue.workers.Go(func() {
		for {
			n, ap, err := ue.conn.ReadFromUDPAddrPort(currentBuf)
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					break
				}

				ue.deps.logger.Warn().Err(err).Msg("read packet failed")
				continue
			}

			// Pre-allocate next buffer
			nextID := idc.Get()
			nextBuf, err := ue.deps.arena.Get(nextID, ue.cfg.bufferSize)
			if err != nil {
				if errors.Is(err, arena.ErrOutOfMemory) {
					ue.deps.logger.Warn().Err(err).Msg("packet dropped")
				} else {
					ue.deps.logger.Error().Stack().Err(err).Msg("buffer allocation failed; packet dropped")
				}
				continue
			}

			ue.workers.packetChan <- packet{
				P:   currentBuf[:n],
				Src: ap,
				ID:  currentID,
			}

			currentBuf = nextBuf
			currentID = nextID
		}
		close(ue.workers.packetChan)
		if err := ue.deps.arena.Put(currentID, currentBuf); err != nil {
			ue.deps.logger.Warn().Err(err).Msg("failed to release last buffer")
		}
	})

	return nil
}

type packet struct {
	Src netip.AddrPort
	P   []byte
	ID  uint64
}

func (ue *UDPEndpoint) handleUDPPacket() {
	for pkt := range ue.workers.packetChan {
		req := request.Packet{
			P: pkt.P,
			Metadata: request.PacketMetadata{
				SrcAddr: addr.Addr{
					IPAddr:   pkt.Src.Addr(),
					MuxIndex: pkt.Src.Port(),
					Family:   addr.FamilyUDP,
				},
				DstAddr: addr.Unknown,
			},
			ID: pkt.ID,
		}
		if err := ue.deps.next.HandlePacket(ue.workers.ctx, &req); err != nil {
			ue.deps.logger.Warn().Object(log.KeyRequest, &req).Err(err).Msg("packet request failed")
		}
		if err := ue.deps.arena.Put(req.ID, req.P); err != nil {
			ue.deps.logger.Error().Stack().Err(err).Msg("buffer release failed")
		}
	}
}

func (ue *UDPEndpoint) Stop(ctx context.Context) error {
	if err := ue.conn.Close(); err != nil {
		ue.deps.logger.Warn().Err(err).Msg("failed to close listen")
	}

	ue.workers.cancel()

	done := make(chan struct{})

	go func() {
		ue.workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
