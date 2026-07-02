package iox

type Conn interface {
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
}

type Unwrapper interface {
	Unwrap() Conn
}

type WriteCloser interface {
	CloseWrite() error
}

type ReadCloser interface {
	CloseRead() error
}
