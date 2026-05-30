package udp

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/HT4w5/l4rt/pkg/common/addr"
	"github.com/HT4w5/l4rt/pkg/common/constants"
	scontext "github.com/HT4w5/l4rt/pkg/common/context"
	"github.com/HT4w5/l4rt/pkg/common/stream"
	"github.com/HT4w5/l4rt/pkg/handler"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type UDPEngressConfig interface {
	handler.HandlerConfig
}

func BuildUDPEgress(cfg UDPEngressConfig, deps handler.HandlerDeps) (*UDPEgress, error) {
	logger, err := deps.LoggerGetter.GetLogger(cfg.LogConfig(), "handler/"+cfg.Tag())
	if err != nil {
		return nil, fmt.Errorf("BuildUDPEgress: failed to get logger: %w", err)
	}
	h := new(UDPEgress)

	h.cfg.tag = cfg.Tag()

	h.deps.logger = logger

	return h, nil
}

// UDPEgress forwards packets from a PacketStream.
//
// UDPEgress implements [github.com/HT4w5/l4rt/pkg/handler.PacketHandler].
type UDPEgress struct {
	cfg struct {
		tag string
	}

	deps struct {
		logger zerolog.Logger
	}

	stats struct {
		rx     atomic.Int64
		tx     atomic.Int64
		rxPkts atomic.Int64
		txPkts atomic.Int64
	}
}

// Implement Handler

func (h *UDPEgress) Tag() string {
	return h.cfg.tag
}

func (h *UDPEgress) Stats() map[string]any {
	return map[string]any{
		"rx":      h.stats.rx.Load(),
		"tx":      h.stats.tx.Load(),
		"rx_pkts": h.stats.rxPkts.Load(),
		"tx_pkts": h.stats.txPkts.Load(),
	}
}

func (h *UDPEgress) HandlePacket(ctx *scontext.Context, s stream.PacketStream) error {
	laddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		h.deps.logger.Err(err).Msg("failed to resolve UDP address")
		return err
	}

	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return err
	}

	return h.copy(s, conn)
}

func (h *UDPEgress) copy(s stream.PacketStream, conn *net.UDPConn) error {
	eg := new(errgroup.Group)

	// local -> remote
	eg.Go(func() error {
		defer conn.Close() // Unblock remote -> local.
		for {
			pkt, _, dst, err := s.ReadPacket()
			if err != nil {
				return err
			}

			ap, err := dst.AssertUDPIPAddr()
			if err != nil {
				continue
			}

			n, err := conn.WriteToUDPAddrPort(pkt, ap)
			if err != nil {
				return err
			}

			h.stats.tx.Add(int64(n))
			h.stats.txPkts.Add(1)
		}
	})

	// remote -> local
	eg.Go(func() error {
		defer s.Close()
		buf := make([]byte, constants.MaxUDPPayloadSize) // TODO: implement pooling
		for {
			n, src, err := conn.ReadFromUDPAddrPort(buf)
			if err != nil {
				return err
			}

			srcAddr, err := addr.FromAddrPort(src, addr.ProtoUDP)
			if err != nil {
				continue
			}

			err = s.WritePacket(buf[:n], srcAddr, addr.UnknownAddr)
			if err != nil {
				return err
			}

			h.stats.rx.Add(int64(n))
			h.stats.rxPkts.Add(1)
		}
	})

	return eg.Wait()
}
