package pkg

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/types"
	"testing"
	"time"
)

func BenchmarkBatchOperations(b *testing.B) {
	type TestStruct struct {
		ID    int
		Value string
		Data  []byte
	}

	// Подготавливаем тестовые данные
	batchSizes := []int{10, 100, 1000, 10000}
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	for _, implementation := range []struct {
		name     string
		newCache func(context.Context, time.Duration, time.Duration) types.ICacheInMemory[*TestStruct]
	}{
		{"ConcurrentCache", NewConcurrentCache[*TestStruct]},
		{"ShardedCache", NewShardedCache[*TestStruct]},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			for _, batchSize := range batchSizes {
				// Подготовка данных для SetBatch
				setBatchData := make(map[string]*TestStruct, batchSize)
				keys := make([]string, 0, batchSize)
				for i := 0; i < batchSize; i++ {
					key := fmt.Sprintf("key-%d", i)
					keys = append(keys, key)
					setBatchData[key] = &TestStruct{
						ID:    i,
						Value: fmt.Sprintf("value-%d", i),
						Data:  make([]byte, 100), // имитируем полезную нагрузку
					}
				}

				cache := implementation.newCache(ctx, ttl, ttlDecrement)

				b.Run(fmt.Sprintf("SetBatch/%d", batchSize), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_ = cache.SetBatch(setBatchData)
					}
				})

				// Заполняем кэш данными для тестов чтения
				_ = cache.SetBatch(setBatchData)

				b.Run(fmt.Sprintf("GetBatch/%d", batchSize), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = cache.GetBatch(keys)
					}
				})

				b.Run(fmt.Sprintf("GetBatchWithMetrics/%d", batchSize), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = cache.GetBatchWithMetrics(keys)
					}
				})

				b.Run(fmt.Sprintf("DeleteBatch/%d", batchSize), func(b *testing.B) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						b.StopTimer()
						_ = cache.SetBatch(setBatchData) // Восстанавливаем данные перед удалением
						b.StartTimer()
						cache.DeleteBatch(keys)
					}
				})
			}
		})
	}
}

// BenchmarkParallelBatchOperations тестирует производительность батч-операций при параллельном доступе
func BenchmarkParallelBatchOperations(b *testing.B) {
	type TestStruct struct {
		ID    int
		Value string
		Data  []byte
	}

	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second
	batchSize := 1000

	// Подготовка тестовых данных
	setBatchData := make(map[string]*TestStruct, batchSize)
	keys := make([]string, 0, batchSize)
	for i := 0; i < batchSize; i++ {
		key := fmt.Sprintf("key-%d", i)
		keys = append(keys, key)
		setBatchData[key] = &TestStruct{
			ID:    i,
			Value: fmt.Sprintf("value-%d", i),
			Data:  make([]byte, 100),
		}
	}

	for _, implementation := range []struct {
		name     string
		newCache func(context.Context, time.Duration, time.Duration) types.ICacheInMemory[*TestStruct]
	}{
		{"ConcurrentCache", NewConcurrentCache[*TestStruct]},
		{"ShardedCache", NewShardedCache[*TestStruct]},
	} {
		b.Run(implementation.name, func(b *testing.B) {
			cache := implementation.newCache(ctx, ttl, ttlDecrement)
			_ = cache.SetBatch(setBatchData) // Предварительно заполняем кэш

			b.Run("ParallelSetBatch", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_ = cache.SetBatch(setBatchData)
					}
				})
			})

			b.Run("ParallelGetBatch", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_, _ = cache.GetBatch(keys)
					}
				})
			})

			b.Run("ParallelGetBatchWithMetrics", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						_, _ = cache.GetBatchWithMetrics(keys)
					}
				})
			})

			b.Run("ParallelMixed", func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					localBatch := make(map[string]*TestStruct, 10)
					localKeys := make([]string, 0, 10)
					for i := 0; i < 10; i++ {
						key := fmt.Sprintf("local-key-%d", i)
						localKeys = append(localKeys, key)
						localBatch[key] = &TestStruct{
							ID:    i,
							Value: fmt.Sprintf("local-value-%d", i),
							Data:  make([]byte, 100),
						}
					}

					for pb.Next() {
						switch b.N % 3 {
						case 0:
							_ = cache.SetBatch(localBatch)
						case 1:
							_, _ = cache.GetBatch(localKeys)
						case 2:
							_, _ = cache.GetBatchWithMetrics(localKeys)
						}
					}
				})
			})
		})
	}
}
