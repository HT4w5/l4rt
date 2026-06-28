//go:build windows

package mem

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

func Allocate(size int) ([]byte, error) {
	size64 := uint64(size)
	fmap, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		uint32(size64>>32),
		uint32(size64),
		nil,
	)
	if err != nil {
		return nil, err
	}

	defer windows.CloseHandle(fmap)

	ptr, err := windows.MapViewOfFile(
		fmap,
		windows.FILE_MAP_WRITE,
		0,
		0,
		uintptr(size),
	)
	if err != nil {
		return nil, err
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size), nil
}

func Free(b []byte) error {
	if b == nil {
		return nil
	}
	return windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&b[0])))
}
