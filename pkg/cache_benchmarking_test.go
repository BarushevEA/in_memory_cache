package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"testing"
	"time"
)

func BenchmarkBatchOperationsBigCacheBatch(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	concurrentCache := NewConcurrentCache[*RealWorldData](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[*RealWorldData](ctx, ttl, ttlDecrement)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	batchSizes := []int{10, 100, 1000, 10000}

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "ConcurrentCache",
			fn: func(b *testing.B) {
				for _, size := range batchSizes {
					// Подготовка данных
					batch := make(map[string]*RealWorldData, size)
					keys := make([]string, 0, size)
					for i := 0; i < size; i++ {
						data := generateRandomData()
						batch[data.TransactionID] = data
						keys = append(keys, data.TransactionID)
					}

					b.Run(fmt.Sprintf("SetBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							_ = concurrentCache.SetBatch(batch)
						}
					})

					b.Run(fmt.Sprintf("GetBatch/%d", size), func(b *testing.B) {
						_ = concurrentCache.SetBatch(batch)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							_, _ = concurrentCache.GetBatch(keys)
						}
					})

					b.Run(fmt.Sprintf("GetBatchWithMetrics/%d", size), func(b *testing.B) {
						_ = concurrentCache.SetBatch(batch)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							_, _ = concurrentCache.GetBatchWithMetrics(keys)
						}
					})

					b.Run(fmt.Sprintf("DeleteBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							b.StopTimer()
							_ = concurrentCache.SetBatch(batch)
							b.StartTimer()
							concurrentCache.DeleteBatch(keys)
						}
					})
				}
			},
		},
		{
			name: "ShardedCache",
			fn: func(b *testing.B) {
				for _, size := range batchSizes {
					batch := make(map[string]*RealWorldData, size)
					keys := make([]string, 0, size)
					for i := 0; i < size; i++ {
						data := generateRandomData()
						batch[data.TransactionID] = data
						keys = append(keys, data.TransactionID)
					}

					b.Run(fmt.Sprintf("SetBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							_ = shardedCache.SetBatch(batch)
						}
					})

					b.Run(fmt.Sprintf("GetBatch/%d", size), func(b *testing.B) {
						_ = shardedCache.SetBatch(batch)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							_, _ = shardedCache.GetBatch(keys)
						}
					})

					b.Run(fmt.Sprintf("GetBatchWithMetrics/%d", size), func(b *testing.B) {
						_ = shardedCache.SetBatch(batch)
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							_, _ = shardedCache.GetBatchWithMetrics(keys)
						}
					})

					b.Run(fmt.Sprintf("DeleteBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							b.StopTimer()
							_ = shardedCache.SetBatch(batch)
							b.StartTimer()
							shardedCache.DeleteBatch(keys)
						}
					})
				}
			},
		},
		{
			name: "BigCache",
			fn: func(b *testing.B) {
				for _, size := range batchSizes {
					// Подготовка данных для BigCache
					data := make(map[string][]byte, size)
					for i := 0; i < size; i++ {
						record := generateRandomData()
						jsonData, _ := json.Marshal(record)
						data[record.TransactionID] = jsonData
					}

					b.Run(fmt.Sprintf("SetBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							for k, v := range data {
								_ = bigCache.Set(k, v)
							}
						}
					})

					b.Run(fmt.Sprintf("GetBatch/%d", size), func(b *testing.B) {
						for k, v := range data {
							_ = bigCache.Set(k, v)
						}
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							for k := range data {
								_, _ = bigCache.Get(k)
							}
						}
					})

					b.Run(fmt.Sprintf("DeleteBatch/%d", size), func(b *testing.B) {
						for i := 0; i < b.N; i++ {
							b.StopTimer()
							for k, v := range data {
								_ = bigCache.Set(k, v)
							}
							b.StartTimer()
							for k := range data {
								_ = bigCache.Delete(k)
							}
						}
					})
				}
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, bm.fn)
	}
}

func BenchmarkParallelBatchOperationsBigCacheBatch(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	concurrentCache := NewConcurrentCache[*RealWorldData](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[*RealWorldData](ctx, ttl, ttlDecrement)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	// Подготовка тестовых данных
	batchSize := 1000
	batch := make(map[string]*RealWorldData, batchSize)
	bigCacheBatch := make(map[string][]byte, batchSize)
	keys := make([]string, 0, batchSize)

	for i := 0; i < batchSize; i++ {
		data := generateRandomData()
		batch[data.TransactionID] = data
		jsonData, _ := json.Marshal(data)
		bigCacheBatch[data.TransactionID] = jsonData
		keys = append(keys, data.TransactionID)
	}

	// Предварительное заполнение кэшей
	_ = concurrentCache.SetBatch(batch)
	_ = shardedCache.SetBatch(batch)
	for k, v := range bigCacheBatch {
		_ = bigCache.Set(k, v)
	}

	b.Run("ConcurrentCache/ParallelSetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = concurrentCache.SetBatch(batch)
			}
		})
	})

	b.Run("ConcurrentCache/ParallelGetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			localKeys := keys[:10] // Используем первые 10 ключей для теста
			for pb.Next() {
				_, _ = concurrentCache.GetBatch(localKeys)
			}
		})
	})

	b.Run("ShardedCache/ParallelSetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = shardedCache.SetBatch(batch)
			}
		})
	})

	b.Run("ShardedCache/ParallelGetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			localKeys := keys[:10]
			for pb.Next() {
				_, _ = shardedCache.GetBatch(localKeys)
			}
		})
	})

	b.Run("BigCache/ParallelSetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			localBatch := make(map[string][]byte)
			for k, v := range bigCacheBatch {
				localBatch[k] = v
				if len(localBatch) >= 10 {
					break
				}
			}
			for pb.Next() {
				for k, v := range localBatch {
					_ = bigCache.Set(k, v)
				}
			}
		})
	})

	b.Run("BigCache/ParallelGetBatch", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			localKeys := keys[:10]
			for pb.Next() {
				for _, k := range localKeys {
					_, _ = bigCache.Get(k)
				}
			}
		})
	})
}
