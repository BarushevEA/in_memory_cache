package pkg

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/types"
	"testing"
	"time"
)

func BenchmarkSingleVsBatchOperationsDetailed(b *testing.B) {
	type TestStruct struct {
		ID   int
		Data []byte
	}

	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second
	size := 100 // фиксированный размер для наглядного сравнения

	// Подготовка тестовых данных
	testData := make(map[string]*TestStruct, size)
	keys := make([]string, 0, size)
	for i := 0; i < size; i++ {
		key := fmt.Sprintf("key-%d", i)
		keys = append(keys, key)
		testData[key] = &TestStruct{
			ID:   i,
			Data: make([]byte, 100),
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

			// Тест операций Set
			b.Run("Set_Single", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for k, v := range testData {
						_ = cache.Set(k, v)
					}
				}
			})

			b.Run("Set_Batch", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_ = cache.SetBatch(testData)
				}
			})

			// Предварительное заполнение для тестов Get
			_ = cache.SetBatch(testData)

			// Тест операций Get
			b.Run("Get_Single", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for _, k := range keys {
						_, _ = cache.Get(k)
					}
				}
			})

			b.Run("Get_Batch", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, _ = cache.GetBatch(keys)
				}
			})

			// Тест операций Delete
			b.Run("Delete_Single", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					_ = cache.SetBatch(testData)
					b.StartTimer()
					for _, k := range keys {
						cache.Delete(k)
					}
				}
			})

			b.Run("Delete_Batch", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer()
					_ = cache.SetBatch(testData)
					b.StartTimer()
					cache.DeleteBatch(keys)
				}
			})
		})
	}
}
