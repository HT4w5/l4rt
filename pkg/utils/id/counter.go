package id

import (
	"math/rand/v2"
	"runtime"
	"sync/atomic"
)

func init() {
	n := runtime.NumCPU()
	if n <= 4 {
		globalIDCounter = &SimpleIDCounter{}
	} else {
		globalIDCounter = NewRandomShardedIDCounter(n)
	}
}

type IDCounter interface {
	Get() uint64
}

var globalIDCounter IDCounter

func Get() uint64 {
	return globalIDCounter.Get()
}

type SimpleIDCounter struct{ atomic.Uint64 }

func (c *SimpleIDCounter) Get() uint64 {
	return c.Add(1)
}

type shard struct {
	value atomic.Uint64
	_     [64 - 8]byte
}

type RandomShardedIDCounter struct {
	shards []shard
	mask   int
	shifts int
}

func NewRandomShardedIDCounter(n int) *RandomShardedIDCounter {
	if n <= 0 {
		n = 1
	}

	numShards := 1
	shifts := 1

	for numShards < n {
		numShards <<= 1
		shifts++
	}

	return &RandomShardedIDCounter{
		shards: make([]shard, numShards),
		mask:   numShards - 1,
		shifts: shifts,
	}
}

func (c *RandomShardedIDCounter) Get() uint64 {
	idx := rand.Int() & c.mask

	val := c.shards[idx].value.Add(1)

	return (val << uint64(c.shifts)) | uint64(idx)
}
