package main

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/pkg"
	"github.com/BarushevEA/in_memory_cache/types"
	"time"
)

// Test represents a simple structure with name and age fields.
type Test struct {
	name string
	age  int
}

// main acts as the program's entry point, benchmarking different cache implementations with specified TTL configurations.
func main() {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 500 * time.Millisecond

	bench(pkg.NewShardedCache[*Test](ctx, ttl, ttlDecrement), "ShardedCache")
	bench(pkg.NewConcurrentCache[*Test](ctx, ttl, ttlDecrement), "ConcurrentCache")
}

// bench measures and prints the duration of setting 10 million entries in a cache implementing the ICache interface.
func bench(cache types.ICacheInMemory[*Test], cacheName string) {
	start := time.Now()
	for i := 0; i < 10000000; i++ {
		err := cache.Set(fmt.Sprintf("%dkey", i), &Test{"name", 20})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(cacheName, "Duration:", time.Since(start), "cache.Len:", cache.Len())
}
