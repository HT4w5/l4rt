package iox

import (
	"errors"
	"io"
	"net"

	uctx "github.com/HT4w5/l4rt/pkg/utils/context"
)

var (
	ErrNilBuffer    = errors.New("nil buffer")
	ErrInvalidWrite = errors.New("invalid write result")
)

func CopyStreamToConn(ctx uctx.StreamCtx, conn net.Conn, buf []byte) (n int64, err error) {
	if buf == nil {
		return 0, ErrNilBuffer
	}

	for {
		if err = ctx.Err(); err != nil {
			return
		}
		nr, er := ctx.Read(buf)
		if nr > 0 {
			nw, ew := conn.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = ErrInvalidWrite
				}
			}
			n += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}

func CopyConnToStream(conn net.Conn, ctx uctx.StreamCtx, buf []byte) (n int64, err error) {
	if buf == nil {
		return 0, ErrNilBuffer
	}

	for {
		if err = ctx.Err(); err != nil {
			return
		}
		nr, er := conn.Read(buf)
		if nr > 0 {
			nw, ew := ctx.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = ErrInvalidWrite
				}
			}
			n += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}
