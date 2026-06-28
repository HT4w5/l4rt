package arena

type Arena interface {
	Get(id uint64, size int) ([]byte, error)
	Put(id uint64, b []byte) error
}
