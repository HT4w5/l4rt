//go:build unix

package mem

import (
	"golang.org/x/sys/unix"
)

func Allocate(size int) (b []byte, err error) {
	b, err = unix.Mmap(
		-1, 0, size,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE,
	)
	return
}

func Free(b []byte) error {
	return unix.Munmap(b)
}
