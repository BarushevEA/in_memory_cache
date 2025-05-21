package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"testing"
	"time"
)

type TestStruct struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"created_at"`
	Data      []byte `json:"data"`
}

func BenchmarkCacheImplementations(b *testing.B) {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 100 * time.Millisecond

	testData := &TestStruct{
		ID:        1,
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: time.Now().Unix(),
		Data:      make([]byte, 100),
	}

	jsonData, _ := json.Marshal(testData)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(1024 * 1024 * 10)

	concurrentCache := NewConcurrentCache[*TestStruct](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[*TestStruct](ctx, ttl, ttlDecrement)

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
						_ = bigCache.Set(fmt.Sprintf("key-%d", i), jsonData)
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
						_ = freeCache.Set([]byte(fmt.Sprintf("key-%d", i)), jsonData, int(ttl.Seconds()))
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

func BenchmarkParallelAccess(b *testing.B) {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 100 * time.Millisecond

	testData := &TestStruct{
		ID:        1,
		Name:      "Test User",
		Email:     "test@example.com",
		CreatedAt: time.Now().Unix(),
		Data:      make([]byte, 100),
	}

	jsonData, _ := json.Marshal(testData)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(1024 * 1024 * 10)
	concurrentCache := NewConcurrentCache[*TestStruct](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[*TestStruct](ctx, ttl, ttlDecrement)

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
					_ = bigCache.Set(key, jsonData)
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
					_ = freeCache.Set([]byte(key), jsonData, int(ttl.Seconds()))
				} else {
					_, _ = freeCache.Get([]byte(key))
				}
				i++
			}
		})
	})
}
