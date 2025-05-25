package pkg

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/types"
	"math/rand/v2"
	"runtime"
	"testing"
	"time"
)

// BenchmarkCache performs benchmark tests for different cache implementations by measuring Set, Get, and SetGet operations.
// It uses the testing.B framework to evaluate cache performance under different scenarios and with varying workloads.
// The function initializes multiple cache types and benchmarks their operations in both serial and parallel contexts.
func BenchmarkCache(b *testing.B) {
	ctx := context.Background()
	caches := map[string]types.ICacheInMemory[string]{
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

// BenchmarkCache_LargeDataSet benchmarks cache implementations with a large dataset under concurrent access conditions.
// It evaluates performance for storing, retrieving, and concurrency handling across different cache types.
// The benchmark uses 100,000 key-value pairs and measures performance with parallel access scenarios.
func BenchmarkCache_LargeDataSet(b *testing.B) {
	ctx := context.Background()
	caches := map[string]types.ICacheInMemory[string]{
		"ConcurrentCache": NewConcurrentCache[string](ctx, time.Second, time.Millisecond),
		"ShardedCache":    NewShardedCache[string](ctx, time.Second, time.Millisecond),
	}

	const dataSize = 100_000
	data := make(map[string]string, dataSize)
	for i := 0; i < dataSize; i++ {
		data[fmt.Sprintf("key-%d", i)] = fmt.Sprintf("value-%d", i)
	}

	for name, cache := range caches {
		b.Run(name, func(b *testing.B) {
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

// BenchmarkCache_StressTest performs a stress test benchmarking various cache implementations using randomized operations.
func BenchmarkCache_StressTest(b *testing.B) {
	ctx := context.Background()
	caches := map[string]types.ICacheInMemory[string]{
		"ConcurrentCache": NewConcurrentCache[string](ctx, time.Second, time.Millisecond),
		"ShardedCache":    NewShardedCache[string](ctx, time.Second, time.Millisecond),
	}

	const (
		initialSize = 100000
		keySpace    = 200000
		getWeight   = 80
		setWeight   = 15
		delWeight   = 4
		rangeWeight = 1
	)

	operations := []struct {
		name     string
		weight   int
		function func(cache types.ICacheInMemory[string], key string)
	}{
		{
			name:   "Get",
			weight: getWeight,
			function: func(cache types.ICacheInMemory[string], key string) {
				_, _ = cache.Get(key)
			},
		},
		{
			name:   "Set",
			weight: setWeight,
			function: func(cache types.ICacheInMemory[string], key string) {
				_ = cache.Set(key, fmt.Sprintf("value-%s", key))
			},
		},
		{
			name:   "Delete",
			weight: delWeight,
			function: func(cache types.ICacheInMemory[string], key string) {
				cache.Delete(key)
			},
		},
		{
			name:   "Range",
			weight: rangeWeight,
			function: func(cache types.ICacheInMemory[string], _ string) {
				_ = cache.Range(func(k string, v string) bool {
					return true
				})
			},
		},
	}

	for name, cache := range caches {
		b.Run(name, func(b *testing.B) {
			for i := 0; i < initialSize; i++ {
				_ = cache.Set(fmt.Sprintf("init-key-%d", i), fmt.Sprintf("init-value-%d", i))
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))

				for pb.Next() {
					key := fmt.Sprintf("key-%d", localRand.Uint64()%uint64(keySpace))
					opIndex := weightedRandomChoice(operations, localRand)
					operations[opIndex].function(cache, key)
				}
			})
		})
	}
}

// weightedRandomChoice selects an index from the operations slice based on the provided weights and randomness.
// It uses a specified random number generator to determine the selection probability.
// The function assumes all weights in the operations slice are non-negative integers.
// It returns the index of the selected operation within the slice.
func weightedRandomChoice(operations []struct {
	name     string
	weight   int
	function func(cache types.ICacheInMemory[string], key string)
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

// BenchmarkCache_DifferentTypes benchmarks the performance of a concurrent cache using different data types with TTL.
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

// BenchmarkCache_MemoryLeakLongRun measures memory allocation and garbage collections of the cache over a long run.
// It simulates concurrent cache operations including setting, getting, and deleting keys to identify potential memory leaks.
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

// BenchmarkCache_Recovery benchmarks the recovery and performance of the cache under rapid context cancellations and re-creation.
func BenchmarkCache_Recovery(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewConcurrentCache[string](ctx, time.Second, time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if rand.Float64() < 0.001 {
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

// BenchmarkCache_Operations measures the performance of various cache operations in concurrent and sharded cache implementations.
func BenchmarkCache_Operations(b *testing.B) {
	ctx := context.Background()
	dataSize := 100000

	prefixes := []string{
		"user:", "session:", "post:", "comment:",
		"product:", "order:", "cache:", "token:",
		"config:", "metric:", "log:", "event:",
		"data:", "temp:", "queue:", "task:",
	}

	operations := []struct {
		name string
		op   func(cache types.ICacheInMemory[string], r *rand.PCG, prefix string)
	}{
		{"Set", func(cache types.ICacheInMemory[string], r *rand.PCG, prefix string) {
			key := fmt.Sprintf("%s%x", prefix, r.Uint64())
			cache.Set(key, "new-value")
		}},
		{"Get", func(cache types.ICacheInMemory[string], r *rand.PCG, prefix string) {
			key := fmt.Sprintf("%s%x", prefix, r.Uint64())
			cache.Get(key)
		}},
		{"Delete", func(cache types.ICacheInMemory[string], r *rand.PCG, prefix string) {
			key := fmt.Sprintf("%s%x", prefix, r.Uint64())
			cache.Delete(key)
		}},
		{"Range", func(cache types.ICacheInMemory[string], _ *rand.PCG, _ string) {
			cache.Range(func(k string, v string) bool {
				return true
			})
		}},
	}

	for _, op := range operations {
		b.Run("ConcurrentCache_"+op.name, func(b *testing.B) {
			cache := NewConcurrentCache[string](ctx, time.Second, time.Millisecond)
			r := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))

			for i := 0; i < dataSize; i++ {
				prefix := prefixes[i%len(prefixes)]
				cache.Set(fmt.Sprintf("%s%x", prefix, r.Uint64()), "value")
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))
				prefix := prefixes[localRand.Uint64()%uint64(len(prefixes))]
				for pb.Next() {
					op.op(cache, localRand, prefix)
				}
			})
		})

		b.Run("ShardedCache_"+op.name, func(b *testing.B) {
			cache := NewShardedCache[string](ctx, time.Second, time.Millisecond)
			r := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))

			for i := 0; i < dataSize; i++ {
				prefix := prefixes[i%len(prefixes)]
				cache.Set(fmt.Sprintf("%s%x", prefix, r.Uint64()), "value")
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				localRand := rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano()))
				prefix := prefixes[localRand.Uint64()%uint64(len(prefixes))]
				for pb.Next() {
					op.op(cache, localRand, prefix)
				}
			})
		})
	}
}
