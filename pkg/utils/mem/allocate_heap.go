//go:build !unix && !windows

package mem

func Allocate(size int) ([]byte, error) {
	return make([]byte, size), nil
}

func Free(b []byte) error {
	return nil
}
