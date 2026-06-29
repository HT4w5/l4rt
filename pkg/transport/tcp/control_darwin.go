//go:build darwin

package tcp

import "syscall"

func newControlFunc(cfg Config) (ControlFunc, error) {
	return func(network, address string, c syscall.RawConn) error { return nil }, nil
}
