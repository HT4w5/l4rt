package udp

import (
	"net"
	"syscall"
)

type Config interface {
}

type ControlFunc = func(network, address string, c syscall.RawConn) error

func NewListenConfig(cfg Config) (*net.ListenConfig, error) {
	lc := &net.ListenConfig{}

	f, err := newControlFunc(cfg)
	if err != nil {
		return nil, err
	}

	lc.Control = f

	return lc, nil
}
