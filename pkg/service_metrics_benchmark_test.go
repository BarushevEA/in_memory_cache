package pkg

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkCacheMetrics(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	type testStruct struct {
		ID    int
		Value string
	}

	benchmarks := []struct {
		name     string
		createFn func(context.Context, time.Duration, time.Duration) ICache[testStruct]
	}{
		{
			name:     "ConcurrentCache",
			createFn: NewConcurrentCache[testStruct],
		},
		{
			name:     "ShardedCache",
			createFn: NewShardedCache[testStruct],
		},
	}

	for _, bm := range benchmarks {
		cache := bm.createFn(ctx, ttl, ttlDecrement)
		testData := testStruct{ID: 1, Value: "test"}

		// Подготовка данных для бенчмарков
		for i := 0; i < 1000; i++ {
			_ = cache.Set(fmt.Sprintf("key-%d", i), testStruct{ID: i, Value: fmt.Sprintf("value-%d", i)})
		}

		b.Run(fmt.Sprintf("%s/GetNodeValueWithMetrics", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%1000)
				_, _, _, _, _ = cache.GetNodeValueWithMetrics(key)
			}
		})

		b.Run(fmt.Sprintf("%s/SetWithMetricsTracking", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("benchmark-key-%d", i)
				_ = cache.Set(key, testData)
			}
		})

		b.Run(fmt.Sprintf("%s/GetWithMetricsTracking", bm.name), func(b *testing.B) {
			key := "test-key"
			_ = cache.Set(key, testData)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = cache.Get(key)
			}
		})

		b.Run(fmt.Sprintf("%s/RangeWithMetrics", bm.name), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = cache.RangeWithMetrics(func(key string, value testStruct, createdAt time.Time, setCount, getCount uint32) bool {
					return true
				})
			}
		})

		b.Run(fmt.Sprintf("%s/ParallelMetricsAccess", bm.name), func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					key := fmt.Sprintf("key-%d", i%1000)
					switch i % 3 {
					case 0:
						_ = cache.Set(key, testData)
					case 1:
						_, _ = cache.Get(key)
					case 2:
						_, _, _, _, _ = cache.GetNodeValueWithMetrics(key)
					}
					i++
				}
			})
		})
	}
}

func BenchmarkMetricsWithDifferentPayloads(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	payloadSizes := []int{64, 256, 1024, 4096} // размеры данных в байтах

	for _, size := range payloadSizes {
		data := struct {
			ID    int
			Value string
		}{
			ID:    1,
			Value: generateRandomString(size),
		}

		b.Run(fmt.Sprintf("PayloadSize_%dB", size), func(b *testing.B) {
			benchmarks := []struct {
				name     string
				createFn func(context.Context, time.Duration, time.Duration) ICache[struct {
					ID    int
					Value string
				}]
			}{
				{
					name: "ConcurrentCache",
					createFn: NewConcurrentCache[struct {
						ID    int
						Value string
					}],
				},
				{
					name: "ShardedCache",
					createFn: NewShardedCache[struct {
						ID    int
						Value string
					}],
				},
			}

			for _, bm := range benchmarks {
				cache := bm.createFn(ctx, ttl, ttlDecrement)

				b.Run(fmt.Sprintf("%s/SetGet", bm.name), func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) {
						i := 0
						for pb.Next() {
							key := fmt.Sprintf("key-%d", i)
							if i%2 == 0 {
								_ = cache.Set(key, data)
							} else {
								_, _ = cache.Get(key)
							}
							_, _, _, _, _ = cache.GetNodeValueWithMetrics(key)
							i++
						}
					})
				})
			}
		})
	}
}
