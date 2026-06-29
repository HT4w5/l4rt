//go:build !windows && !linux && !darwin

package udp

import "syscall"

func newControlFunc(Config) (ControlFunc, error) {
	return func(network, address string, c syscall.RawConn) error { return nil }, nil
}
