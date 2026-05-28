package stream

import (
	"io"
	"net"
	"time"
)

type ByteStream interface {
	io.ReadWriteCloser
	CloseWrite() error
	SetDeadline(t time.Time) error
	Unwrap() (net.Conn, bool)
}
