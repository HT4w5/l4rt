//go:build !unix && !windows

package mem

func init() {
	defaultAllocator = &HeapAllocator{}
}

type HeapAllocator struct{}

func (a *HeapAllocator) Allocate(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func (a *HeapAllocator) Free(b []byte) error {
	return nil
}
