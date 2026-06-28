// LLM usage: generated with deepseek-v4-pro and modified manually
package arena

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmark allocator (make()-backed, no build-tag restrictions)
// ---------------------------------------------------------------------------

// benchAllocator uses make() for allocation — identical to HeapAllocator
// but always available regardless of OS build tags. This ensures
// apples-to-apples comparison with the BenchmarkMake baseline.
type benchAllocator struct{}

func (benchAllocator) Allocate(size int) ([]byte, error) { return make([]byte, size), nil }
func (benchAllocator) Free([]byte) error                 { return nil }

// ---------------------------------------------------------------------------
// Benchmark helpers
// ---------------------------------------------------------------------------

// newBenchArena creates an arena for benchmarking backed by make().
// Registers cleanup via b.Cleanup.
func newBenchArena(b *testing.B, shards, maxBytes int) *BuddyArena {
	b.Helper()
	cfg := &mockConfig{
		shards_:   shards,
		maxBytes_: maxBytes,
		allocator: benchAllocator{},
	}
	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		b.Fatalf("NewBuddyArena: %v", err)
	}
	b.Cleanup(cleanup)
	return arena
}

// benchSizes is the set of allocation sizes used across benchmarks.
var benchSizes = []struct {
	name string
	size int
}{
	{"128B", 128},
	{"1K", 1 << 10},
	{"4K", 4 << 10},
	{"16K", 16 << 10},
	{"64K", 64 << 10},
}

// ---------------------------------------------------------------------------
// Phase 1: Get+Put cycle vs make()
// ---------------------------------------------------------------------------

// BenchmarkBuddyArena_GetPut measures the full allocate-use-free cycle
// through the buddy arena: Get → touch → Put.
func BenchmarkBuddyArena_GetPut(b *testing.B) {
	// 64 MiB per shard, NumCPU shards — large enough to avoid OOM.
	arena := newBenchArena(b, runtime.NumCPU(), 64<<20)
	b.ResetTimer()
	b.ReportAllocs()

	for _, bs := range benchSizes {
		b.Run(bs.name, func(b *testing.B) {
			for b.Loop() {
				buf, err := arena.Get(0, bs.size)
				if err != nil {
					b.Fatalf("Get(%d): %v", bs.size, err)
				}
				// Touch first and last byte to prevent optimizer
				// from eliminating the allocation.
				buf[0] = 0xFF
				buf[len(buf)-1] = 0xFF
				if err := arena.Put(0, buf); err != nil {
					b.Fatalf("Put: %v", err)
				}
			}
		})
	}
}

// BenchmarkMake measures raw Go allocation throughput via make().
// Compare against BenchmarkBuddyArena_GetPut to quantify the arena's
// locking and bookkeeping overhead.
func BenchmarkMake(b *testing.B) {
	b.ReportAllocs()

	for _, bs := range benchSizes {
		b.Run(bs.name, func(b *testing.B) {
			for b.Loop() {
				buf := make([]byte, bs.size)
				buf[0] = 0xFF
				buf[len(buf)-1] = 0xFF
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 2: Steady-state rotation
// ---------------------------------------------------------------------------

// BenchmarkBuddyArena_GetPut_Rotating measures steady-state throughput
// by pre-allocating buffers and rotating: Put(old) → Get(new) → touch.
// This avoids OOM and measures sustained performance.
func BenchmarkBuddyArena_GetPut_Rotating(b *testing.B) {
	// Use a larger per-shard arena (256 MiB) so we can pre-allocate enough
	// buffers even for the 64 KiB size without exhausting the shard.
	arena := newBenchArena(b, runtime.NumCPU(), 256<<20)
	b.ReportAllocs()

	for _, bs := range benchSizes {
		b.Run(bs.name, func(b *testing.B) {
			// Adaptive pool: fewer buffers for large allocations to avoid OOM.
			poolSize := max(4, (64<<10)/bs.size)
			pool := make([][]byte, poolSize)
			for i := range pool {
				var err error
				pool[i], err = arena.Get(0, bs.size)
				if err != nil {
					b.Fatalf("pre-alloc Get(%d): %v", bs.size, err)
				}
			}

			b.ResetTimer()
			for b.Loop() {
				i := b.N % poolSize
				// Return old buffer.
				if err := arena.Put(0, pool[i]); err != nil {
					b.Fatalf("Put: %v", err)
				}
				// Get new buffer.
				var err error
				pool[i], err = arena.Get(0, bs.size)
				if err != nil {
					b.Fatalf("Get: %v", err)
				}
				pool[i][0] = 0xFF
				pool[i][len(pool[i])-1] = 0xFF
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 3: Concurrency
// ---------------------------------------------------------------------------

// BenchmarkBuddyArena_GetPut_Parallel measures concurrent throughput
// across NumCPU shards. Each goroutine uses its own ID → no contention.
func BenchmarkBuddyArena_GetPut_Parallel(b *testing.B) {
	arena := newBenchArena(b, runtime.NumCPU(), 64<<20)
	b.ReportAllocs()

	for _, bs := range benchSizes {
		bs := bs
		b.Run(bs.name, func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				// Use goroutine-local ID to avoid shard contention.
				id := uint64(0)
				for p.Next() {
					buf, err := arena.Get(id, bs.size)
					if err != nil {
						b.Fatalf("Get: %v", err)
					}
					buf[0] = 0xFF
					buf[len(buf)-1] = 0xFF
					if err := arena.Put(id, buf); err != nil {
						b.Fatalf("Put: %v", err)
					}
					id++
				}
			})
		})
	}
}

// BenchmarkBuddyArena_GetPut_Contended measures throughput under
// maximum contention: single shard, all goroutines use same ID.
func BenchmarkBuddyArena_GetPut_Contended(b *testing.B) {
	arena := newBenchArena(b, 1, 64<<20)
	b.ReportAllocs()

	for _, bs := range benchSizes {
		bs := bs
		b.Run(bs.name, func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					buf, err := arena.Get(0, bs.size)
					if err != nil {
						b.Fatalf("Get: %v", err)
					}
					buf[0] = 0xFF
					buf[len(buf)-1] = 0xFF
					if err := arena.Put(0, buf); err != nil {
						b.Fatalf("Put: %v", err)
					}
				}
			})
		})
	}
}

// BenchmarkMake_Parallel measures concurrent make() throughput.
func BenchmarkMake_Parallel(b *testing.B) {
	b.ReportAllocs()

	for _, bs := range benchSizes {
		bs := bs
		b.Run(bs.name, func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					buf := make([]byte, bs.size)
					buf[0] = 0xFF
					buf[len(buf)-1] = 0xFF
				}
			})
		})
	}
}

// ---------------------------------------------------------------------------
// Phase 4: Scaling & workload
// ---------------------------------------------------------------------------

// BenchmarkBuddyArena_GetPut_Scaling measures throughput at different
// GOMAXPROCS levels with matching shard count, showing how sharding
// scales with parallelism.
func BenchmarkBuddyArena_GetPut_Scaling(b *testing.B) {
	for i := range runtime.NumCPU() {
		procs := i + 1
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", procs), func(b *testing.B) {
			runtime.GOMAXPROCS(procs)
			arena := newBenchArena(b, procs, 256<<20)
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(p *testing.PB) {
				const size = 4 << 10 // 4 KiB
				id := uint64(0)
				for p.Next() {
					buf, err := arena.Get(id, size)
					if err != nil {
						b.Fatalf("Get: %v", err)
					}
					buf[0] = 0xFF
					buf[len(buf)-1] = 0xFF
					if err := arena.Put(id, buf); err != nil {
						b.Fatalf("Put: %v", err)
					}
					id++
				}
			})
		})
	}
}

// workloadSizes defines a realistic mixed-size distribution for
// workload benchmarks: many small headers, some MTU-sized packets,
// occasional jumbo frames.
var workloadSizes = []struct {
	name  string
	size  int
	ratio int // relative frequency
}{
	{"64B", 64, 30},
	{"256B", 256, 25},
	{"1500B", 1500, 25},
	{"4K", 4 << 10, 10},
	{"16K", 16 << 10, 7},
	{"64K", 64 << 10, 3},
}

// BenchmarkBuddyArena_Workload simulates a network buffer pool with
// mixed allocation sizes in a realistic ratio.
func BenchmarkBuddyArena_Workload(b *testing.B) {
	arena := newBenchArena(b, runtime.NumCPU(), 64<<20)
	b.ReportAllocs()

	// Compute cumulative distribution.
	totalRatio := 0
	for _, ws := range workloadSizes {
		totalRatio += ws.ratio
	}

	b.ResetTimer()
	var counter atomic.Int64
	b.RunParallel(func(p *testing.PB) {
		id := uint64(0)
		for p.Next() {
			c := int(counter.Add(1)) % totalRatio
			// Pick size by weighted distribution.
			sz := workloadSizes[0].size
			accum := 0
			for _, ws := range workloadSizes {
				accum += ws.ratio
				if c < accum {
					sz = ws.size
					break
				}
			}

			buf, err := arena.Get(id, sz)
			if err != nil {
				b.Fatalf("Get(%d): %v", sz, err)
			}
			buf[0] = 0xFF
			buf[len(buf)-1] = 0xFF
			if err := arena.Put(id, buf); err != nil {
				b.Fatalf("Put: %v", err)
			}
			id++
		}
	})
}

// BenchmarkMake_Workload is the make() baseline for the mixed-size
// workload distribution.
func BenchmarkMake_Workload(b *testing.B) {
	b.ReportAllocs()

	totalRatio := 0
	for _, ws := range workloadSizes {
		totalRatio += ws.ratio
	}

	b.ResetTimer()
	var counter atomic.Int64
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			c := int(counter.Add(1)) % totalRatio
			sz := workloadSizes[0].size
			accum := 0
			for _, ws := range workloadSizes {
				accum += ws.ratio
				if c < accum {
					sz = ws.size
					break
				}
			}

			buf := make([]byte, sz)
			buf[0] = 0xFF
			buf[len(buf)-1] = 0xFF
		}
	})
}
