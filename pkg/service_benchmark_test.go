package pkg

import (
	"context"
	"fmt"
	"math/rand/v2"
	"runtime"
	"testing"
	"time"
)

// Тесты производительности (benchmarks)
func BenchmarkCache(b *testing.B) {
	ctx := context.Background()
	caches := map[string]ICache[string]{
		"ConcurrentCache": NewConcurrentCache[string](ctx, time.Second, time.Millisecond),
		"ShardedCache":    NewShardedCache[string](ctx, time.Second, time.Millisecond),
	}

	for name, cache := range caches {
		b.Run(name+"/Set", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache.Set("key", "value")
			}
		})

		b.Run(name+"/Get", func(b *testing.B) {
			cache.Set("key", "value")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cache.Get("key")
			}
		})

		b.Run(name+"/SetGet", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					cache.Set("key", "value")
					cache.Get("key")
				}
			})
		})
	}
}

// Добавьте эти тесты в конец файла service_test.go

// Тест на большие объемы данных
func BenchmarkCache_LargeDataSet(b *testing.B) {
	ctx := context.Background()
	caches := map[string]ICache[string]{
		"ConcurrentCache": NewConcurrentCache[string](ctx, time.Second, time.Millisecond),
		"ShardedCache":    NewShardedCache[string](ctx, time.Second, time.Millisecond),
	}

	// Генерируем большой набор данных
	const dataSize = 100_000
	data := make(map[string]string, dataSize)
	for i := 0; i < dataSize; i++ {
		data[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	for name, cache := range caches {
		b.Run(name, func(b *testing.B) {
			// Заполнение кеша
			for k, v := range data {
				if err := cache.Set(k, v); err != nil {
					b.Fatal(err)
				}
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					key := fmt.Sprintf("key-%d", rand.Int()%dataSize)
					cache.Get(key)
				}
			})
		})
	}
}

// Тест на устойчивость к нагрузке
func BenchmarkCache_StressTest(b *testing.B) {
	ctx := context.Background()
	caches := map[string]ICache[string]{
		"ConcurrentCache": NewConcurrentCache[string](ctx, time.Second, time.Millisecond),
		"ShardedCache":    NewShardedCache[string](ctx, time.Second, time.Millisecond),
	}

	operations := []struct {
		name     string
		weight   int
		function func(cache ICache[string], key string)
	}{
		{
			name:   "Set",
			weight: 25,
			function: func(cache ICache[string], key string) {
				_ = cache.Set(key, "value")
			},
		},
		{
			name:   "Get",
			weight: 50,
			function: func(cache ICache[string], key string) {
				_, _ = cache.Get(key)
			},
		},
		{
			name:   "Delete",
			weight: 15,
			function: func(cache ICache[string], key string) {
				cache.Delete(key)
			},
		},
		{
			name:   "Range",
			weight: 10,
			function: func(cache ICache[string], _ string) {
				_ = cache.Range(func(k string, v string) bool {
					return true
				})
			},
		},
	}

	for name, cache := range caches {
		b.Run(name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))

				for pb.Next() {
					key := fmt.Sprintf("key-%d", localRand.Uint64()%1000)
					opIndex := weightedRandomChoice(operations, localRand)
					operations[opIndex].function(cache, key)
				}
			})
		})
	}
}

// weightedRandomChoice выбирает операцию с учетом её веса
func weightedRandomChoice(operations []struct {
	name     string
	weight   int
	function func(cache ICache[string], key string)
}, r *rand.PCG) int {
	totalWeight := 0
	for _, op := range operations {
		totalWeight += op.weight
	}

	n := int(r.Uint64() % uint64(totalWeight))
	for i, op := range operations {
		n -= op.weight
		if n < 0 {
			return i
		}
	}
	return len(operations) - 1
}

// Тест на работу с разными типами данных
func BenchmarkCache_DifferentTypes(b *testing.B) {
	ctx := context.Background()

	type ComplexStruct struct {
		ID        int
		Name      string
		Data      []byte
		Timestamp time.Time
		Map       map[string]interface{}
	}

	b.Run("IntCache", func(b *testing.B) {
		cache := NewConcurrentCache[int](ctx, time.Second, time.Millisecond)
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("test", 42)
				cache.Get("test")
			}
		})
	})

	b.Run("StructCache", func(b *testing.B) {
		cache := NewConcurrentCache[ComplexStruct](ctx, time.Second, time.Millisecond)
		value := ComplexStruct{
			ID:        1,
			Name:      "test",
			Data:      make([]byte, 1024),
			Timestamp: time.Now(),
			Map:       make(map[string]interface{}),
		}
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				cache.Set("test", value)
				cache.Get("test")
			}
		})
	})
}

// Тест на утечку памяти при длительной работе
func BenchmarkCache_MemoryLeakLongRun(b *testing.B) {
	ctx := context.Background()
	cache := NewConcurrentCache[string](ctx, time.Millisecond*100, time.Millisecond*10)

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := fmt.Sprintf("key-%d", rand.Int()%1000)
			cache.Set(key, "test-value")
			cache.Get(key)
			cache.Delete(key)
		}
	})

	runtime.GC()
	runtime.ReadMemStats(&m2)

	b.ReportMetric(float64(m2.Alloc-m1.Alloc), "B/op")
	b.ReportMetric(float64(m2.NumGC-m1.NumGC), "GCs")
}

// Тест на восстановление после сбоев
func BenchmarkCache_Recovery(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewConcurrentCache[string](ctx, time.Second, time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Float64() < 0.001 { // 0.1% шанс отмены контекста
				cancel()
				ctx, cancel = context.WithCancel(context.Background())
				cache = NewConcurrentCache[string](ctx, time.Second, time.Millisecond)
			}

			key := fmt.Sprintf("key-%d", rand.Int()%1000)
			cache.Set(key, "test-value")
			cache.Get(key)
		}
	})
	cancel()
}
