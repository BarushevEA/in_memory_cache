package main

import (
	"context"
	"github.com/BarushevEA/in_memory_cache/pkg"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"strconv"
	"sync"
	"testing"
	"time"
)

// SafeMap представляет потокобезопасную обертку над map
type SafeMap struct {
	sync.RWMutex
	data map[string]string
}

// NewSafeMap создает новый экземпляр SafeMap
func NewSafeMap() *SafeMap {
	return &SafeMap{
		data: make(map[string]string),
	}
}

// Set добавляет значение в мапу
func (s *SafeMap) Set(key string, value string) error {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
	return nil
}

// Len возвращает количество элементов в мапе
func (s *SafeMap) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.data)
}

func (m *SafeMap) Get(key string) (string, bool) {
	m.RLock()
	defer m.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func generateKey(i int) string {
	return "key-" + strconv.Itoa(i)
}

func generateValue(i int) string {
	return "value-" + strconv.Itoa(i)
}

func BenchmarkCaches(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Minute

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(100 * 1024 * 1024)
	standardMap := NewSafeMap()
	shardedCache := pkg.NewShardedCache[string](ctx, ttl, ttlDecrement)
	concurrentCache := pkg.NewConcurrentCache[string](ctx, ttl, ttlDecrement)

	// Предварительное заполнение кешей данными для тестов чтения
	for i := 0; i < 1000; i++ {
		key, value := generateKey(i), generateValue(i)
		standardMap.Set(key, value)
		bigCache.Set(key, []byte(value))
		freeCache.Set([]byte(key), []byte(value), int(ttl.Seconds()))
		shardedCache.Set(key, value)
		concurrentCache.Set(key, value)
	}

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "StandardMap_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = standardMap.Set(generateKey(i), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "StandardMap_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = standardMap.Get(generateKey(i % 1000))
						i++
					}
				})
			},
		},
		{
			name: "StandardMap_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							_ = standardMap.Set(generateKey(i), generateValue(i))
						} else {
							_, _ = standardMap.Get(generateKey(i % 1000))
						}
						i++
					}
				})
			},
		},
		{
			name: "BigCache_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = bigCache.Set(generateKey(i), []byte(generateValue(i)))
						i++
					}
				})
			},
		},
		{
			name: "BigCache_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = bigCache.Get(generateKey(i % 1000))
						i++
					}
				})
			},
		},
		{
			name: "BigCache_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							_ = bigCache.Set(generateKey(i), []byte(generateValue(i)))
						} else {
							_, _ = bigCache.Get(generateKey(i % 1000))
						}
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = freeCache.Set([]byte(generateKey(i)), []byte(generateValue(i)), int(ttl.Seconds()))
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = freeCache.Get([]byte(generateKey(i % 1000)))
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							_ = freeCache.Set([]byte(generateKey(i)), []byte(generateValue(i)), int(ttl.Seconds()))
						} else {
							_, _ = freeCache.Get([]byte(generateKey(i % 1000)))
						}
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = shardedCache.Set(generateKey(i), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = shardedCache.Get(generateKey(i % 1000))
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							_ = shardedCache.Set(generateKey(i), generateValue(i))
						} else {
							_, _ = shardedCache.Get(generateKey(i % 1000))
						}
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = concurrentCache.Set(generateKey(i), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = concurrentCache.Get(generateKey(i % 1000))
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							_ = concurrentCache.Set(generateKey(i), generateValue(i))
						} else {
							_, _ = concurrentCache.Get(generateKey(i % 1000))
						}
						i++
					}
				})
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, bm.fn)
	}
}
