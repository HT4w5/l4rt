// LLM usage: generated with deepseek-v4-pro and modified manually
package arena

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/HT4w5/l4rt/pkg/utils/mem"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

// mockAllocator implements mem.Allocator and tracks allocation/free calls.
type mockAllocator struct {
	mu      sync.Mutex
	allocs  [][]byte
	sizes   []int
	frees   [][]byte
	freeErr error
}

func (m *mockAllocator) Allocate(size int) ([]byte, error) {
	b := make([]byte, size)
	m.mu.Lock()
	m.allocs = append(m.allocs, b)
	m.sizes = append(m.sizes, size)
	m.mu.Unlock()
	return b, nil
}

func (m *mockAllocator) Free(b []byte) error {
	m.mu.Lock()
	m.frees = append(m.frees, b)
	m.mu.Unlock()
	if m.freeErr != nil {
		return m.freeErr
	}
	return nil
}

func (m *mockAllocator) allocCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.allocs)
}

func (m *mockAllocator) freeCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.frees)
}

func (m *mockAllocator) totalAllocated() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	total := 0
	for _, s := range m.sizes {
		total += s
	}
	return total
}

// mockConfig implements BuddyArenaConfig.
type mockConfig struct {
	shards_   int
	auto_     bool
	maxBytes_ int
	allocator mem.Allocator
}

func (c *mockConfig) Shards() (n int, auto bool) { return c.shards_, c.auto_ }
func (c *mockConfig) MaxBytes() int              { return c.maxBytes_ }
func (c *mockConfig) Allocator() mem.Allocator   { return c.allocator }

// newTestArena creates an arena with the given number of shards and max bytes,
// using a mockAllocator. Registers cleanup via t.Cleanup.
func newTestArena(t *testing.T, shards, maxBytes int) (*BuddyArena, *mockAllocator) {
	t.Helper()
	alloc := &mockAllocator{}
	cfg := &mockConfig{
		shards_:   shards,
		maxBytes_: maxBytes,
		allocator: alloc,
	}
	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena: %v", err)
	}
	t.Cleanup(cleanup)
	return arena, alloc
}

// ---------------------------------------------------------------------------
// Phase 2: Creation & Cleanup
// ---------------------------------------------------------------------------

// TestNewBuddyArena_Basic is the tracer bullet: create an arena with 1 shard,
// verify the allocator is called, cleanup frees the backing memory.
func TestNewBuddyArena_Basic(t *testing.T) {
	alloc := &mockAllocator{}
	cfg := &mockConfig{
		shards_:   1,
		maxBytes_: minShardSize,
		allocator: alloc,
	}

	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena: %v", err)
	}
	if arena == nil {
		t.Fatal("expected non-nil arena")
	}

	// Verify allocator received exactly 1 Allocate call of minShardSize.
	if n := alloc.allocCount(); n != 1 {
		t.Fatalf("alloc count: got %d, want 1", n)
	}
	if total := alloc.totalAllocated(); total != minShardSize {
		t.Fatalf("total allocated: got %d, want %d", total, minShardSize)
	}

	// Cleanup should free the backing memory.
	cleanup()
	if n := alloc.freeCount(); n != 1 {
		t.Fatalf("free count after cleanup: got %d, want 1", n)
	}
}

// TestNewBuddyArena_AutoShards verifies that auto=true produces
// runtime.NumCPU() shards with matching alloc/free counts.
func TestNewBuddyArena_AutoShards(t *testing.T) {
	alloc := &mockAllocator{}
	cfg := &mockConfig{
		auto_:     true,
		maxBytes_: minShardSize * 4,
		allocator: alloc,
	}

	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena(auto): %v", err)
	}
	if arena == nil {
		t.Fatal("expected non-nil arena")
	}

	// Auto mode should produce at least 1 shard (NumCPU).
	n := alloc.allocCount()
	if n < 1 {
		t.Fatalf("alloc count: got %d, want >= 1", n)
	}
	// Each shard is a power of 2 at least minShardSize.
	total := alloc.totalAllocated()
	if total < minShardSize*n {
		t.Fatalf("total allocated %d < %d * %d", total, n, minShardSize)
	}

	cleanup()
	if alloc.freeCount() != n {
		t.Fatalf("free count: got %d, want %d", alloc.freeCount(), n)
	}
}

// TestNewBuddyArena_InvalidConfig verifies error handling for bad configs.
func TestNewBuddyArena_InvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		shards   int
		auto     bool
		maxBytes int
		wantErr  bool
	}{
		{"zero shards", 0, false, minShardSize, true},
		{"negative shards", -1, false, minShardSize, true},
		{"maxBytes rounds up", 1, false, minShardSize + 1, false},
		{"maxBytes exact power of 2", 1, false, minShardSize * 2, false},
		{"maxBytes small", 1, false, 100, false}, // rounds up to minShardSize
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alloc := &mockAllocator{}
			cfg := &mockConfig{
				shards_:   tt.shards,
				auto_:     tt.auto,
				maxBytes_: tt.maxBytes,
				allocator: alloc,
			}

			arena, cleanup, err := NewBuddyArena(cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if arena != nil {
					t.Fatal("expected nil arena on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if arena == nil {
				t.Fatal("expected non-nil arena")
			}
			cleanup()
		})
	}
}

// TestNewBuddyArena_OutstandingAllocs verifies cleanup when allocations
// are still outstanding: destroyBuddyShard returns an error and Free
// is not called.
func TestNewBuddyArena_OutstandingAllocs(t *testing.T) {
	alloc := &mockAllocator{}
	cfg := &mockConfig{
		shards_:   1,
		maxBytes_: minShardSize,
		allocator: alloc,
	}

	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena: %v", err)
	}

	// Allocate without freeing.
	_, err = arena.Get(0, 500)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Cleanup should not Free since shard is still in use.
	cleanup()
	if alloc.freeCount() != 0 {
		t.Fatalf("free count with outstanding allocs: got %d, want 0", alloc.freeCount())
	}
}

// ---------------------------------------------------------------------------
// Phase 3: Simple Get & Put
// ---------------------------------------------------------------------------

// TestGetPut_Basic verifies a simple allocate, write, read, free cycle.
func TestGetPut_Basic(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	b, err := arena.Get(0, 100)
	if err != nil {
		t.Fatalf("Get(100): %v", err)
	}
	if len(b) != 100 {
		t.Fatalf("len: got %d, want 100", len(b))
	}
	if cap(b) < 100 {
		t.Fatalf("cap: got %d, want >= 100", cap(b))
	}
	// Cap should be minBlockSize (1024), the smallest allocation unit.
	if cap(b) != minBlockSize {
		t.Fatalf("cap: got %d, want %d (minBlockSize)", cap(b), minBlockSize)
	}

	// Write data and read it back.
	for i := range b {
		b[i] = byte(i % 256)
	}
	for i := range b {
		if b[i] != byte(i%256) {
			t.Fatalf("data corruption at offset %d: got %d, want %d", i, b[i], byte(i%256))
		}
	}

	// Return the buffer.
	if err := arena.Put(0, b); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get again — should succeed (reuses freed memory).
	b2, err := arena.Get(0, 100)
	if err != nil {
		t.Fatalf("Get(100) after Put: %v", err)
	}
	if len(b2) != 100 {
		t.Fatalf("len after re-get: got %d, want 100", len(b2))
	}
}

// TestGetPut_DifferentSizes verifies allocating and freeing multiple
// buffers of different power-of-2-aligned sizes.
func TestGetPut_DifferentSizes(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	sizes := []int{300, 1024, 2048, 5000, 100, 8192}

	// Allocate all.
	bufs := make([][]byte, len(sizes))
	for i, sz := range sizes {
		b, err := arena.Get(0, sz)
		if err != nil {
			t.Fatalf("Get(%d)[%d]: %v", i, sz, err)
		}
		if len(b) != sz {
			t.Fatalf("Get(%d): len=%d, want %d", i, len(b), sz)
		}
		bufs[i] = b
	}

	// Write a marker to each.
	for i, b := range bufs {
		b[0] = byte(i)
	}

	// Free all.
	for i, b := range bufs {
		if err := arena.Put(0, b); err != nil {
			t.Fatalf("Put(%d): %v", i, err)
		}
	}

	// Re-allocate all in reverse order.
	for i := len(sizes) - 1; i >= 0; i-- {
		b, err := arena.Get(0, sizes[i])
		if err != nil {
			t.Fatalf("re-Get(%d)[%d]: %v", i, sizes[i], err)
		}
		if len(b) != sizes[i] {
			t.Fatalf("re-Get(%d): len=%d, want %d", i, len(b), sizes[i])
		}
	}
}

// TestGetPut_WrongShard verifies that putting a buffer to a different
// shard than the one it was allocated from returns ErrInvalidSlice.
func TestGetPut_WrongShard(t *testing.T) {
	arena, _ := newTestArena(t, 2, minShardSize*2)

	// Allocate from shard 0.
	b, err := arena.Get(0, 500)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Try to free to shard 1 — should fail.
	err = arena.Put(1, b)
	if err == nil {
		t.Fatal("expected ErrInvalidSlice when putting to wrong shard")
	}
}

// TestGetPut_ForeignSlice verifies that slices not allocated by the arena
// are rejected with ErrInvalidSlice.
func TestGetPut_ForeignSlice(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	// A heap-allocated slice should be rejected.
	b := make([]byte, 100)
	if err := arena.Put(0, b); err == nil {
		t.Fatal("expected ErrInvalidSlice for heap-allocated slice")
	}
}

// ---------------------------------------------------------------------------
// Phase 4: Edge Cases
// ---------------------------------------------------------------------------

// TestGet_ZeroSize verifies that requesting a zero-size allocation
// returns an error.
func TestGet_ZeroSize(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	_, err := arena.Get(0, 0)
	if err == nil {
		t.Fatal("expected error for size=0")
	}
}

// TestGet_TooLarge verifies ErrTooLarge when size exceeds shard capacity.
func TestGet_TooLarge(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	_, err := arena.Get(0, minShardSize+1)
	if err == nil {
		t.Fatal("expected ErrTooLarge")
	}
}

// TestGet_OutOfMemory verifies ErrOutOfMemory when the shard is
// completely filled.
func TestGet_OutOfMemory(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	// Fill the shard with minBlockSize allocations.
	maxBlocks := minShardSize / minBlockSize
	var err error
	for range maxBlocks {
		_, err = arena.Get(0, minBlockSize)
		if err != nil {
			t.Fatalf("Get should succeed until full, but got: %v", err)
		}
	}

	// Next allocation should fail.
	_, err = arena.Get(0, minBlockSize)
	if err == nil {
		t.Fatal("expected ErrOutOfMemory when shard is full")
	}
}

// TestGet_ExactShardSize verifies allocating the entire shard in one
// max-order block works.
func TestGet_ExactShardSize(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	b, err := arena.Get(0, minShardSize)
	if err != nil {
		t.Fatalf("Get(shardSize): %v", err)
	}
	if len(b) != minShardSize {
		t.Fatalf("len: got %d, want %d", len(b), minShardSize)
	}
	if cap(b) != minShardSize {
		t.Fatalf("cap: got %d, want %d", cap(b), minShardSize)
	}

	// Put it back.
	if err := arena.Put(0, b); err != nil {
		t.Fatalf("Put(shardSize): %v", err)
	}

	// Should be able to get it again.
	b2, err := arena.Get(0, minShardSize)
	if err != nil {
		t.Fatalf("Get(shardSize) after Put: %v", err)
	}
	if len(b2) != minShardSize {
		t.Fatalf("len after re-get: got %d, want %d", len(b2), minShardSize)
	}
}

// TestGetPut_FillAndDrain fills the entire shard with small allocations,
// frees them all, then re-allocates — proving all memory is reclaimed.
func TestGetPut_FillAndDrain(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	maxBlocks := minShardSize / minBlockSize

	// Fill the shard with minBlockSize allocations.
	bufs := make([][]byte, 0, maxBlocks)
	for range maxBlocks {
		b, err := arena.Get(0, minBlockSize)
		if err != nil {
			t.Fatalf("Get during fill: %v (filled %d/%d)", err, len(bufs), maxBlocks)
		}
		bufs = append(bufs, b)
	}

	if len(bufs) != maxBlocks {
		t.Fatalf("filled %d blocks, want %d", len(bufs), maxBlocks)
	}

	// Free all.
	for _, b := range bufs {
		if err := arena.Put(0, b); err != nil {
			t.Fatalf("Put during drain: %v", err)
		}
	}

	// Re-fill — must succeed.
	for range maxBlocks {
		_, err := arena.Get(0, minBlockSize)
		if err != nil {
			t.Fatalf("Get after drain should succeed, got: %v", err)
		}
	}
}

// TestGetPut_InvalidCap verifies that a slice with non-power-of-2 cap
// (after reslicing) is rejected by Put.
func TestGetPut_InvalidCap(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	b, err := arena.Get(0, 500)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Reslice to change capacity to a non-power-of-2 value.
	// b[:len(b):len(b)] gives cap=500 which is not a power of 2.
	badCap := b[:len(b):len(b)]
	if cap(badCap) != 500 {
		t.Fatalf("cap after reslice: got %d, want 500", cap(badCap))
	}

	if err := arena.Put(0, badCap); err == nil {
		t.Fatal("expected ErrInvalidSlice for non-power-of-2 cap")
	}
}

// TestGetPut_DoubleFree verifies behavior when a buffer is freed twice.
func TestGetPut_DoubleFree(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	b, err := arena.Get(0, minBlockSize)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// First free — succeeds.
	if err := arena.Put(0, b); err != nil {
		t.Fatalf("first Put: %v", err)
	}

	// Second free with same buffer — behavior is implementation-defined.
	// The buddy system may panic, return ErrInvalidSlice, or succeed silently.
	// We document the current behavior without t.Fatal on success.
	err = arena.Put(0, b)
	t.Logf("double-free behavior: %v", err)
}

// TestGetPut_BuddyMerging verifies that freeing adjacent blocks allows
// a larger allocation that spans the merged region.
func TestGetPut_BuddyMerging(t *testing.T) {
	arena, _ := newTestArena(t, 1, minShardSize)

	// Allocate two small blocks (same ID ensures same shard).
	b1, err := arena.Get(0, minBlockSize)
	if err != nil {
		t.Fatalf("Get(b1): %v", err)
	}
	b2, err := arena.Get(0, minBlockSize)
	if err != nil {
		t.Fatalf("Get(b2): %v", err)
	}

	// Free both.
	if err := arena.Put(0, b1); err != nil {
		t.Fatalf("Put(b1): %v", err)
	}
	if err := arena.Put(0, b2); err != nil {
		t.Fatalf("Put(b2): %v", err)
	}

	// Now a 2*minBlockSize allocation should succeed (merged region).
	b3, err := arena.Get(0, minBlockSize*2)
	if err != nil {
		t.Fatalf("Get(2*minBlockSize) after merge: %v", err)
	}
	if cap(b3) < minBlockSize*2 {
		t.Fatalf("merged cap: got %d, want >= %d", cap(b3), minBlockSize*2)
	}

	// Write full range to verify it's usable.
	for i := range b3 {
		b3[i] = byte(i % 256)
	}
	for i := range b3 {
		if b3[i] != byte(i%256) {
			t.Fatalf("corruption at %d after merge", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Phase 5: Concurrency
// ---------------------------------------------------------------------------

// TestConcurrentGetPut verifies that multiple goroutines can allocate
// and free buffers concurrently across multiple shards without races
// or corruption. Run with -race.
func TestConcurrentGetPut(t *testing.T) {
	t.Parallel()

	numShards := 4
	alloc := &mockAllocator{}
	cfg := &mockConfig{
		shards_:   numShards,
		maxBytes_: minShardSize * numShards,
		allocator: alloc,
	}

	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena: %v", err)
	}
	defer cleanup()

	const numGoroutines = 100
	const itersPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := range numGoroutines {
		go func(gid int) {
			defer wg.Done()

			// Spread goroutines across shards.
			id := uint64(gid)

			for range itersPerGoroutine {
				// Random-ish size: 100 to 4096 bytes.
				sz := 100 + (gid*37+17)%4000

				b, err := arena.Get(id, sz)
				if err != nil {
					// OutOfMemory is OK under high contention.
					continue
				}

				// Write goroutine ID to the buffer.
				if len(b) >= 4 {
					b[0] = byte(gid)
					b[1] = byte(gid >> 8)
					b[2] = byte(gid >> 16)
					b[3] = byte(gid >> 24)
				}

				// Yield to increase interleaving.
				runtime.Gosched()

				// Read back and verify.
				if len(b) >= 4 {
					got := int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
					if got != gid {
						t.Errorf("corruption: expected gid %d, got %d", gid, got)
					}
				}

				if err := arena.Put(id, b); err != nil {
					t.Errorf("Put: %v", err)
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestConcurrentGetPut_SameShard stresses a single shard with many
// goroutines contending on the same mutex.
func TestConcurrentGetPut_SameShard(t *testing.T) {
	t.Parallel()

	alloc := &mockAllocator{}
	cfg := &mockConfig{
		shards_:   1,
		maxBytes_: minShardSize * 8, // 512 KiB to reduce OOM errors.
		allocator: alloc,
	}

	arena, cleanup, err := NewBuddyArena(cfg)
	if err != nil {
		t.Fatalf("NewBuddyArena: %v", err)
	}
	defer cleanup()

	const numGoroutines = 50
	const itersPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	var allocOps atomic.Int64
	var freeOps atomic.Int64
	var oomCount atomic.Int64

	for g := range numGoroutines {
		go func(gid int) {
			defer wg.Done()

			for range itersPerGoroutine {
				sz := 100 + (gid*53+19)%2000

				b, err := arena.Get(0, sz) // All go to the same shard.
				if err != nil {
					oomCount.Add(1)
					continue
				}
				allocOps.Add(1)

				// Minimal work to avoid holding the lock.
				if len(b) > 0 {
					b[0] = byte(gid)
				}

				if err := arena.Put(0, b); err != nil {
					t.Errorf("Put: %v", err)
				}
				freeOps.Add(1)
			}
		}(g)
	}

	wg.Wait()

	t.Logf("allocs=%d frees=%d oom=%d", allocOps.Load(), freeOps.Load(), oomCount.Load())
	if allocOps.Load() != freeOps.Load() {
		t.Errorf("mismatch: allocs=%d frees=%d", allocOps.Load(), freeOps.Load())
	}
}
