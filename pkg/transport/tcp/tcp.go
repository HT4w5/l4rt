package tcp

import (
	"net"
	"syscall"
	"time"
)

type Config interface {
	DialTimeout() time.Duration
	KeepAlive() (enable bool, idle time.Duration, interval time.Duration, count int)
	DialFallbackDelay() time.Duration
}

type ControlFunc = func(network, address string, c syscall.RawConn) error

func NewDialer(cfg Config) (*net.Dialer, error) {
	enable, idle, interval, count := cfg.KeepAlive()
	d := &net.Dialer{
		Timeout:   cfg.DialTimeout(),
		KeepAlive: -1,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   enable,
			Idle:     idle,
			Interval: interval,
			Count:    count,
		},
		FallbackDelay: cfg.DialFallbackDelay(),
	}

	f, err := newControlFunc(cfg)
	if err != nil {
		return nil, err
	}

	d.Control = f

	return d, nil
}

func NewListenConfig(cfg Config) (*net.ListenConfig, error) {
	enable, idle, interval, count := cfg.KeepAlive()
	lc := &net.ListenConfig{
		KeepAlive: -1,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   enable,
			Idle:     idle,
			Interval: interval,
			Count:    count,
		},
	}

	f, err := newControlFunc(cfg)
	if err != nil {
		return nil, err
	}

	lc.Control = f

	return lc, nil
}
