package mem

type Allocator interface {
	Allocate(size int) ([]byte, error)
	Free(b []byte) error
}

var defaultAllocator Allocator

func Allocate(size int) ([]byte, error) {
	return defaultAllocator.Allocate(size)
}

func Free(b []byte) error {
	return defaultAllocator.Free(b)
}

func DefaultAllocator() Allocator {
	return defaultAllocator
}
