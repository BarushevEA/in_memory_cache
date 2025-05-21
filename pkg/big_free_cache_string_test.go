package pkg

import (
	"bytes"
	"context"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"testing"
	"time"
)

func generateRandomString(size int) string {
	var buffer bytes.Buffer
	for i := 0; i < size; i++ {
		buffer.WriteByte('x')
	}
	return buffer.String()
}

func BenchmarkStringCacheImplementations(b *testing.B) {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 100 * time.Millisecond

	testData := generateRandomString(1024)
	testDataBytes := []byte(testData)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(1024 * 1024 * 10)

	concurrentCache := NewConcurrentCache[string](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[string](ctx, ttl, ttlDecrement)

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "ConcurrentCache",
			fn: func(b *testing.B) {
				b.Run("Set", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_ = concurrentCache.Set(fmt.Sprintf("key-%d", i), testData)
					}
				})
				b.Run("Get", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, _ = concurrentCache.Get(fmt.Sprintf("key-%d", i))
					}
				})
			},
		},
		{
			name: "ShardedCache",
			fn: func(b *testing.B) {
				b.Run("Set", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_ = shardedCache.Set(fmt.Sprintf("key-%d", i), testData)
					}
				})
				b.Run("Get", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, _ = shardedCache.Get(fmt.Sprintf("key-%d", i))
					}
				})
			},
		},
		{
			name: "BigCache",
			fn: func(b *testing.B) {
				b.Run("Set", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_ = bigCache.Set(fmt.Sprintf("key-%d", i), testDataBytes)
					}
				})
				b.Run("Get", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, _ = bigCache.Get(fmt.Sprintf("key-%d", i))
					}
				})
			},
		},
		{
			name: "FreeCache",
			fn: func(b *testing.B) {
				b.Run("Set", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_ = freeCache.Set([]byte(fmt.Sprintf("key-%d", i)), testDataBytes, int(ttl.Seconds()))
					}
				})
				b.Run("Get", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, _ = freeCache.Get([]byte(fmt.Sprintf("key-%d", i)))
					}
				})
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, bm.fn)
	}
}

func BenchmarkStringParallelAccess(b *testing.B) {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 100 * time.Millisecond

	testData := generateRandomString(1024)
	testDataBytes := []byte(testData)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(1024 * 1024 * 10)
	concurrentCache := NewConcurrentCache[string](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[string](ctx, ttl, ttlDecrement)

	b.Run("ConcurrentCache_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i)
				if i%2 == 0 {
					_ = concurrentCache.Set(key, testData)
				} else {
					_, _ = concurrentCache.Get(key)
				}
				i++
			}
		})
	})

	b.Run("ShardedCache_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i)
				if i%2 == 0 {
					_ = shardedCache.Set(key, testData)
				} else {
					_, _ = shardedCache.Get(key)
				}
				i++
			}
		})
	})

	b.Run("BigCache_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i)
				if i%2 == 0 {
					_ = bigCache.Set(key, testDataBytes)
				} else {
					_, _ = bigCache.Get(key)
				}
				i++
			}
		})
	})

	b.Run("FreeCache_Parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				key := fmt.Sprintf("key-%d", i)
				if i%2 == 0 {
					_ = freeCache.Set([]byte(key), testDataBytes, int(ttl.Seconds()))
				} else {
					_, _ = freeCache.Get([]byte(key))
				}
				i++
			}
		})
	})
}
