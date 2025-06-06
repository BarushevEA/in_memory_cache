package pkg

import (
	"bytes"
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/types"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"testing"
	"time"
)

// generateRandomString generates a random string of the specified size using placeholder characters.
func generateRandomString(size int) string {
	var buffer bytes.Buffer
	for i := 0; i < size; i++ {
		buffer.WriteByte('x')
	}
	return buffer.String()
}

// BenchmarkStringCacheImplementations benchmarks various string cache implementations for performance and efficiency.
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

// BenchmarkStringParallelAccess performs parallel benchmarking tests for various cache implementations with string data.
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

func BenchmarkStringDeletionOperations(b *testing.B) {
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

	// Предварительное заполнение кэшей
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		concurrentCache.Set(key, testData)
		shardedCache.Set(key, testData)
		bigCache.Set(key, testDataBytes)
		freeCache.Set([]byte(key), testDataBytes, int(ttl.Seconds()))
	}

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "ConcurrentCache_Delete",
			fn: func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					concurrentCache.Delete(fmt.Sprintf("key-%d", i%1000))
				}
			},
		},
		{
			name: "ShardedCache_Delete",
			fn: func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					shardedCache.Delete(fmt.Sprintf("key-%d", i%1000))
				}
			},
		},
		{
			name: "BigCache_Delete",
			fn: func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = bigCache.Delete(fmt.Sprintf("key-%d", i%1000))
				}
			},
		},
		{
			name: "FreeCache_Delete",
			fn: func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = freeCache.Del([]byte(fmt.Sprintf("key-%d", i%1000)))
				}
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, bm.fn)
	}
}

// Добавим также тест на параллельное удаление
func BenchmarkStringParallelDelete(b *testing.B) {
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

	// Предварительное заполнение
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		concurrentCache.Set(key, testData)
		shardedCache.Set(key, testData)
		bigCache.Set(key, testDataBytes)
		freeCache.Set([]byte(key), testDataBytes, int(ttl.Seconds()))
	}

	b.Run("ConcurrentCache_ParallelDelete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				concurrentCache.Delete(fmt.Sprintf("key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("ShardedCache_ParallelDelete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				shardedCache.Delete(fmt.Sprintf("key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("BigCache_ParallelDelete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_ = bigCache.Delete(fmt.Sprintf("key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("FreeCache_ParallelDelete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				_ = freeCache.Del([]byte(fmt.Sprintf("key-%d", i%1000)))
				i++
			}
		})
	})
}

func BenchmarkSingleVsBatchOperations(b *testing.B) {
	type TestStruct struct {
		ID    int
		Value string
		Data  []byte
	}

	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second
	batchSizes := []int{10, 100, 1000, 10000}

	for _, implementation := range []struct {
		name     string
		newCache func(context.Context, time.Duration, time.Duration) types.ICacheInMemory[*TestStruct]
	}{
		{"ConcurrentCache", NewConcurrentCache[*TestStruct]},
		{"ShardedCache", NewShardedCache[*TestStruct]},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			for _, size := range batchSizes {
				cache := implementation.newCache(ctx, ttl, ttlDecrement)

				// Подготовка данных
				batchData := make(map[string]*TestStruct, size)
				keys := make([]string, 0, size)
				for i := 0; i < size; i++ {
					key := fmt.Sprintf("key-%d", i)
					keys = append(keys, key)
					batchData[key] = &TestStruct{
						ID:    i,
						Value: fmt.Sprintf("value-%d", i),
						Data:  make([]byte, 100),
					}
				}

				// Тест операций записи
				b.Run(fmt.Sprintf("Set_%d/Single", size), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						for k, v := range batchData {
							_ = cache.Set(k, v)
						}
					}
				})

				b.Run(fmt.Sprintf("Set_%d/Batch", size), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = cache.SetBatch(batchData)
					}
				})

				// Предварительное заполнение для тестов чтения
				_ = cache.SetBatch(batchData)

				// Тест операций чтения
				b.Run(fmt.Sprintf("Get_%d/Single", size), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						for _, k := range keys {
							_, _ = cache.Get(k)
						}
					}
				})

				b.Run(fmt.Sprintf("Get_%d/Batch", size), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = cache.GetBatch(keys)
					}
				})

				// Тест операций удаления
				b.Run(fmt.Sprintf("Delete_%d/Single", size), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_ = cache.SetBatch(batchData)
						b.StartTimer()
						for _, k := range keys {
							cache.Delete(k)
						}
					}
				})

				b.Run(fmt.Sprintf("Delete_%d/Batch", size), func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_ = cache.SetBatch(batchData)
						b.StartTimer()
						cache.DeleteBatch(keys)
					}
				})
			}
		})
	}
}
