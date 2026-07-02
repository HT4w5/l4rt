package id

import (
	"fmt"
	"runtime"
	"testing"
)

func BenchmarkSimpleIDCounter(b *testing.B) {
	for i := range runtime.NumCPU() {
		i += 1
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", i), func(b *testing.B) {
			runtime.GOMAXPROCS(i)
			c := SimpleIDCounter{}
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					c.Get()
				}
			})
		})
	}
}

func BenchmarkRandomShardedIDCounter(b *testing.B) {
	for i := range runtime.NumCPU() {
		i += 1
		b.Run(fmt.Sprintf("GOMAXPROCS=%d", i), func(b *testing.B) {
			runtime.GOMAXPROCS(i)
			c := NewRandomShardedIDCounter(i)
			b.RunParallel(func(p *testing.PB) {
				for p.Next() {
					c.Get()
				}
			})
		})
	}
}
