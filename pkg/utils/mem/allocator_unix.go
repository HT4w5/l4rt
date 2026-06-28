//go:build unix

package mem

import (
	"golang.org/x/sys/unix"
)

func init() {
	defaultAllocator = &UnixMmapAllocator{}
}

type UnixMmapAllocator struct{}

func (a *UnixMmapAllocator) Allocate(size int) (b []byte, err error) {
	b, err = unix.Mmap(
		-1, 0, size,
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_ANON|unix.MAP_PRIVATE,
	)
	return
}

func (a *UnixMmapAllocator) Free(b []byte) error {
	return unix.Munmap(b)
}
