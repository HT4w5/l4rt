package stream

import (
	"net"
	"time"
)

// TCPStream implements [ByteStream].
type TCPStream struct {
	conn *net.TCPConn
}

func NewTCPStream(conn *net.TCPConn) *TCPStream {
	return &TCPStream{
		conn: conn,
	}
}

func (s *TCPStream) Read(b []byte) (int, error) { return s.conn.Read(b) }

func (s *TCPStream) Write(b []byte) (int, error) { return s.conn.Write(b) }

func (s *TCPStream) Close() error { return s.conn.Close() }

func (s *TCPStream) CloseWrite() error { return s.conn.CloseWrite() }

func (s *TCPStream) SetDeadline(t time.Time) error { return s.conn.SetDeadline(t) }

func (s *TCPStream) Unwrap() (net.Conn, bool) {
	return s.conn, true
}
