package pipe

import "io"

// TODOs:
// 1. manage buffer manually for reuse
// 2. implement idle timeout
// 3. unwrap connections for wt/rf optimizations
func Copy(dst io.Writer, src io.Reader) (n int64, err error) {
	if wt, ok := src.(io.WriterTo); ok {
		n, err = wt.WriteTo(dst)
		return
	}

	if rf, ok := dst.(io.ReaderFrom); ok {
		n, err = rf.ReadFrom(src)
		return
	}

	n, err = io.Copy(dst, src)
	return
}
