package arena

import (
	"errors"
	"fmt"
	"math/bits"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/HT4w5/l4rt/pkg/utils/mem"
)

var (
	ErrTooLarge     = errors.New("size is too large")
	ErrOutOfMemory  = errors.New("out of memory")
	ErrInvalidSlice = errors.New("foreign or invalid slice")
)

const (
	minBlockSizeBits = 10
	minBlockSize     = 1 << minBlockSizeBits
	minShardSizeBits = 16
	minShardSize     = 1 << minShardSizeBits
)

type blockNode struct {
	prev int // previous free block (-1 if none)
	next int // next free block (-1 if none)
}

type buddyShard struct {
	b         []byte
	buddyBits [][]uint64
	nodes     []blockNode
	freeLists []int
	startPtr  uintptr
	maxOrder  int
	used      atomic.Int64
	mu        sync.Mutex
}

func newBuddyShard(size int, allocator mem.Allocator) (buddyShard, error) {
	if size < minShardSize || (size&(size-1)) != 0 {
		return buddyShard{}, fmt.Errorf("newBuddyShard: size must be power of 2 and at least %d", minShardSize)
	}

	b, err := allocator.Allocate(size)
	if err != nil {
		return buddyShard{}, err
	}

	totalBaseBlocks := size / minBlockSize
	maxOrder := bits.Len(uint(totalBaseBlocks)) - 1

	buddyBits := make([][]uint64, maxOrder)
	freeLists := make([]int, maxOrder+1)

	for i := range freeLists {
		freeLists[i] = -1
	}

	for order := range maxOrder {
		blocks := totalBaseBlocks >> order
		pairs := blocks / 2
		words := (pairs + 63) / 64
		buddyBits[order] = make([]uint64, words)
	}

	nodes := make([]blockNode, totalBaseBlocks)
	for i := range nodes {
		nodes[i] = blockNode{prev: -1, next: -1}
	}

	s := buddyShard{
		b:         b,
		buddyBits: buddyBits,
		nodes:     nodes,
		freeLists: freeLists,
		startPtr:  uintptr(unsafe.Pointer(&b[0])),
		maxOrder:  maxOrder,
	}

	s.addToFreeList(0, maxOrder)

	return s, nil
}

func destroyBuddyShard(shard *buddyShard, allocator mem.Allocator) error {
	shard.mu.Lock()
	defer shard.mu.Unlock()

	used := shard.used.Load()
	if used > 0 {
		return errors.New("shard still in use")
	}

	for i := range len(shard.freeLists) {
		shard.freeLists[i] = -1
	}

	return allocator.Free(shard.b)
}

func (s *buddyShard) get(size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("size must be greater than 0")
	}

	targetSize := minBlockSize
	order := 0
	for targetSize < size {
		targetSize <<= 1
		order++
	}

	if order > s.maxOrder {
		return nil, ErrTooLarge
	}

	s.mu.Lock()

	allocOrder := order
	for s.freeLists[allocOrder] == -1 {
		allocOrder++
		if allocOrder > s.maxOrder {
			s.mu.Unlock()
			return nil, ErrOutOfMemory
		}
	}

	blockIdx := s.freeLists[allocOrder]
	s.removeFromFreeList(blockIdx, allocOrder)

	if allocOrder < s.maxOrder {
		s.flipBuddyBit(blockIdx, allocOrder)
	}

	for allocOrder > order {
		allocOrder--
		// Left -> utilize; right -> freeList
		rightIdx := (blockIdx << 1) + 1
		s.addToFreeList(rightIdx, allocOrder)

		s.flipBuddyBit(blockIdx<<1, allocOrder)
		blockIdx <<= 1
	}

	offset := blockIdx * (minBlockSize << order)
	s.mu.Unlock()
	s.used.Add(int64(targetSize))
	return s.b[offset : offset+size : offset+targetSize], nil
}

func (s *buddyShard) put(b []byte) error {
	ptr := uintptr(unsafe.Pointer(&b[0]))
	if ptr < s.startPtr || ptr >= s.startPtr+uintptr(len(s.b)) {
		return ErrInvalidSlice
	}

	offset := int(ptr - s.startPtr)
	if offset%minBlockSize != 0 {
		return ErrInvalidSlice
	}

	blockSize := cap(b)
	if blockSize < minBlockSize || (blockSize&(blockSize-1)) != 0 {
		return ErrInvalidSlice
	}
	order := bits.Len(uint(blockSize/minBlockSize)) - 1

	blockIdx := offset / blockSize

	s.mu.Lock()

	for order < s.maxOrder {
		if s.flipBuddyBit(blockIdx, order) {
			// Buddy occupied
			s.addToFreeList(blockIdx, order)
			s.mu.Unlock()
			s.used.Add(-int64(blockSize))
			return nil
		}

		buddyIdx := blockIdx ^ 1

		s.removeFromFreeList(buddyIdx, order)

		blockIdx >>= 1
		order++
	}

	s.addToFreeList(blockIdx, s.maxOrder)
	s.mu.Unlock()
	s.used.Add(-int64(blockSize))
	return nil
}

func (s *buddyShard) flipBuddyBit(blockIdx int, order int) bool {
	pairIdx := blockIdx >> 1
	wordIdx := pairIdx >> 6
	bitPos := pairIdx & 63
	mask := uint64(1 << bitPos)
	word := &s.buddyBits[order][wordIdx]
	old := *word
	*word = old ^ mask
	return (old & mask) == 0
}

func (s *buddyShard) addToFreeList(blockIdx int, order int) {
	baseIdx := blockIdx << order

	head := s.freeLists[order]
	s.nodes[baseIdx].prev = -1
	s.nodes[baseIdx].next = head

	if head != -1 {
		headBaseIdx := head << order
		s.nodes[headBaseIdx].prev = blockIdx
	}
	s.freeLists[order] = blockIdx
}

func (s *buddyShard) removeFromFreeList(blockIdx int, order int) {
	baseIdx := blockIdx << order
	node := s.nodes[baseIdx]

	if s.freeLists[order] == blockIdx {
		s.freeLists[order] = node.next
	}
	if node.next != -1 {
		nextBaseIdx := node.next << order
		s.nodes[nextBaseIdx].prev = node.prev
	}
	if node.prev != -1 {
		prevBaseIdx := node.prev << order
		s.nodes[prevBaseIdx].next = node.next
	}

	s.nodes[baseIdx].prev = -1
	s.nodes[baseIdx].next = -1
}

type BuddyArenaConfig interface {
	Shards() (n int, auto bool)
	MaxBytes() (n int)
	Allocator() mem.Allocator
}

type BuddyArena struct {
	cfg struct {
		size int
	}

	allocator mem.Allocator

	shards []buddyShard
}

func NewBuddyArena(cfg BuddyArenaConfig) (*BuddyArena, func(), error) {
	numShards, auto := cfg.Shards()
	if auto {
		numShards = runtime.NumCPU()
	} else {
		if numShards <= 0 {
			return nil, nil, fmt.Errorf("NewBuddyArena: number of shards must be greater than 0")
		}
	}

	maxBytesPerShard := cfg.MaxBytes() / numShards

	shardSize := minShardSize
	for shardSize < maxBytesPerShard {
		shardSize <<= 1
	}

	arena := &BuddyArena{}

	arena.cfg.size = numShards * shardSize

	arena.allocator = cfg.Allocator()

	arena.shards = make([]buddyShard, 0, numShards)
	var shardError error
	for range numShards {
		s, err := newBuddyShard(shardSize, arena.allocator)
		if err != nil {
			shardError = fmt.Errorf("NewBuddyArena: failed to create shard: %w", err)
			break
		}
		arena.shards = append(arena.shards, s)
	}

	if shardError != nil {
		for i := range len(arena.shards) {
			destroyBuddyShard(&arena.shards[i], arena.allocator)
		}
		return nil, nil, shardError
	}

	return arena, func() {
		for i := range len(arena.shards) {
			destroyBuddyShard(&arena.shards[i], arena.allocator)
		}
	}, nil
}

func (a *BuddyArena) Get(id uint64, size int) ([]byte, error) {
	idx := int(id) % len(a.shards)
	return a.shards[idx].get(size)
}

func (a *BuddyArena) Put(id uint64, b []byte) error {
	idx := int(id) % len(a.shards)
	return a.shards[idx].put(b)
}
