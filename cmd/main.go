package main

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/pkg"
	"github.com/BarushevEA/in_memory_cache/types"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"github.com/patrickmn/go-cache"
	"time"
)

// Test represents a simple structure with name and age fields.
type Test struct {
	name string
	age  int
}

func main() {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 500 * time.Millisecond
	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)
	freeCache := freecache.NewCache(100 * 1024 * 1024)

	benchFreeCache(freeCache, ttl, "FreeCache")
	benchBigCache(bigCache, "BigCache")
	benchGoCache(cache.New(ttl, ttlDecrement), "GoCache")
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

func benchGoCache(goCache *cache.Cache, cacheName string) {
	start := time.Now()
	for i := 0; i < 10000000; i++ {
		goCache.Set(fmt.Sprintf("%dkey", i), &Test{"name", 20}, cache.DefaultExpiration)
	}

	fmt.Println(cacheName, "Duration:", time.Since(start), "cache.Len:", len(goCache.Items()))
}

func benchBigCache(bigCache *bigcache.BigCache, cacheName string) {
	start := time.Now()
	for i := 0; i < 10000000; i++ {
		err := bigCache.Set(fmt.Sprintf("%dkey", i), []byte("name:20"))
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(cacheName, "Duration:", time.Since(start), "cache.Len:", bigCache.Len())
}

func benchFreeCache(freeCache *freecache.Cache, ttl time.Duration, cacheName string) {
	start := time.Now()
	for i := 0; i < 10000000; i++ {
		err := freeCache.Set([]byte(fmt.Sprintf("%dkey", i)), []byte("name:20"), int(ttl.Seconds()))
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(cacheName, "Duration:", time.Since(start), "cache.Len:", freeCache.EntryCount())
}
