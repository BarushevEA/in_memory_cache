package main

import (
	"context"
	"github.com/BarushevEA/in_memory_cache/pkg"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"github.com/patrickmn/go-cache"
	"strconv"
	"sync"
	"testing"
	"time"
)

// SafeMap is a threadsafe map that provides synchronized access to a map of string keys and string values.
type SafeMap struct {
	sync.RWMutex
	data map[string]string
}

// NewSafeMap creates and returns a new instance of SafeMap with an initialized internal map.
func NewSafeMap() *SafeMap {
	return &SafeMap{
		data: make(map[string]string),
	}
}

// Set adds or updates a key-value pair in the SafeMap with thread-safe locking.
func (s *SafeMap) Set(key string, value string) error {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
	return nil
}

// Len returns the number of key-value pairs currently stored in the SafeMap. It employs a read lock for thread safety.
func (s *SafeMap) Len() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.data)
}

// Get retrieves the value associated with the provided key and a boolean indicating if the key exists in the map.
func (m *SafeMap) Get(key string) (string, bool) {
	m.RLock()
	defer m.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

// generateKey generates a string key by concatenating "key-" with the string representation of the given integer.
func generateKey(i int) string {
	return "key-" + strconv.Itoa(i)
}

// generateValue generates a value string by appending the integer input to the prefix "value-".
func generateValue(i int) string {
	return "value-" + strconv.Itoa(i)
}

// BenchmarkCaches benchmarks multiple caching implementations for read, write, and mixed operations using the testing framework.
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
	goCache := cache.New(ttl, ttl)

	for i := 0; i < 1000; i++ {
		key, value := generateKey(i), generateValue(i)
		standardMap.Set(key, value)
		bigCache.Set(key, []byte(value))
		freeCache.Set([]byte(key), []byte(value), int(ttl.Seconds()))
		shardedCache.Set(key, value)
		concurrentCache.Set(key, value)
		goCache.Set(key, value, cache.DefaultExpiration)
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
		{
			name: "GoCache_Write",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						goCache.Set(generateKey(i), generateValue(i), cache.DefaultExpiration)
						i++
					}
				})
			},
		},
		{
			name: "GoCache_Read",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = goCache.Get(generateKey(i % 1000))
						i++
					}
				})
			},
		},
		{
			name: "GoCache_Mixed",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						if i%2 == 0 {
							goCache.Set(generateKey(i), generateValue(i), cache.DefaultExpiration)
						} else {
							_, _ = goCache.Get(generateKey(i % 1000))
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

// BenchmarkCachesWithDuplicates tests cache performance with a limited set of keys to simulate data reuse and pooling.
func BenchmarkCachesWithDuplicates(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute // Достаточно большой TTL, чтобы не было вытеснения по времени
	ttlDecrement := 1 * time.Minute
	const numKeys = 1000 // Ограниченное количество уникальных ключей

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(100 * 1024 * 1024)
	standardMap := NewSafeMap()
	shardedCache := pkg.NewShardedCache[string](ctx, ttl, ttlDecrement)
	concurrentCache := pkg.NewConcurrentCache[string](ctx, ttl, ttlDecrement)
	goCache := cache.New(ttl, ttl)

	// Предварительное заполнение кешей ограниченным набором ключей
	for i := 0; i < numKeys; i++ {
		key, value := generateKey(i), generateValue(i)
		standardMap.Set(key, value)
		bigCache.Set(key, []byte(value))
		freeCache.Set([]byte(key), []byte(value), int(ttl.Seconds()))
		shardedCache.Set(key, value)
		concurrentCache.Set(key, value)
		goCache.Set(key, value, cache.DefaultExpiration)
	}

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "StandardMap_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						// Используем оператор по модулю для генерации повторяющихся ключей
						_ = standardMap.Set(generateKey(i%numKeys), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "StandardMap_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = standardMap.Get(generateKey(i % numKeys))
						i++
					}
				})
			},
		},
		{
			name: "BigCache_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = bigCache.Set(generateKey(i%numKeys), []byte(generateValue(i)))
						i++
					}
				})
			},
		},
		{
			name: "BigCache_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = bigCache.Get(generateKey(i % numKeys))
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = freeCache.Set([]byte(generateKey(i%numKeys)), []byte(generateValue(i)), int(ttl.Seconds()))
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = freeCache.Get([]byte(generateKey(i % numKeys)))
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = shardedCache.Set(generateKey(i%numKeys), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = shardedCache.Get(generateKey(i % numKeys))
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_ = concurrentCache.Set(generateKey(i%numKeys), generateValue(i))
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = concurrentCache.Get(generateKey(i % numKeys))
						i++
					}
				})
			},
		},
		{
			name: "GoCache_Write_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						goCache.Set(generateKey(i%numKeys), generateValue(i), cache.DefaultExpiration)
						i++
					}
				})
			},
		},
		{
			name: "GoCache_Read_Duplicates",
			fn: func(b *testing.B) {
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						_, _ = goCache.Get(generateKey(i % numKeys))
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

// BenchmarkCachesWithEviction tests cache performance under eviction scenarios (short TTLs and deletions).
func BenchmarkCachesWithEviction(b *testing.B) {
	ctx := context.Background()
	// Очень короткий TTL для быстрого вытеснения
	ttl := 200 * time.Millisecond
	ttlDecrement := 50 * time.Millisecond // Возможно, не используется всеми кешами, но для полной картины

	const numKeys = 1000 // Количество уникальных ключей для записи

	// Инициализация кешей для каждого бенчмарка, чтобы избежать влияния предыдущих тестов
	// и обеспечить чистые условия для каждого запуска.

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "BigCache_Write_Evict_Read",
			fn: func(b *testing.B) {
				bigcacheConfig := bigcache.DefaultConfig(ttl)
				bigcacheConfig.Verbose = false
				bigcacheConfig.Logger = nil
				bigCache, _ := bigcache.New(ctx, bigcacheConfig)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := []byte(generateValue(i))
						_ = bigCache.Set(key, value)
						// Чтение элемента, который, возможно, уже был вытеснен
						_, _ = bigCache.Get(key)
						i++
					}
				})
				// Небольшая задержка после цикла, чтобы дать кешу время на очистку/пулинг
				time.Sleep(ttl * 2)
			},
		},
		{
			name: "FreeCache_Write_Evict_Read",
			fn: func(b *testing.B) {
				freeCache := freecache.NewCache(100 * 1024 * 1024)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := []byte(generateKey(i % numKeys))
						value := []byte(generateValue(i))
						_ = freeCache.Set(key, value, int(ttl.Seconds()))
						_, _ = freeCache.Get(key)
						i++
					}
				})
				time.Sleep(ttl * 2)
			},
		},
		{
			name: "ShardedCache_Write_Evict_Read",
			fn: func(b *testing.B) {
				shardedCache := pkg.NewShardedCache[string](ctx, ttl, ttlDecrement)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						_ = shardedCache.Set(key, value)
						_, _ = shardedCache.Get(key)
						i++
					}
				})
				time.Sleep(ttl * 2)
			},
		},
		{
			name: "ConcurrentCache_Write_Evict_Read",
			fn: func(b *testing.B) {
				concurrentCache := pkg.NewConcurrentCache[string](ctx, ttl, ttlDecrement)
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						_ = concurrentCache.Set(key, value)
						_, _ = concurrentCache.Get(key)
						i++
					}
				})
				time.Sleep(ttl * 2)
			},
		},
		{
			name: "GoCache_Write_Evict_Read",
			fn: func(b *testing.B) {
				goCache := cache.New(ttl, ttl) // TTL по умолчанию для GoCache - это переданный ttl
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						goCache.Set(key, value, cache.DefaultExpiration)
						_, _ = goCache.Get(key)
						i++
					}
				})
				time.Sleep(ttl * 2)
			},
		},
		// Добавление тестов с явным удалением
		{
			name: "BigCache_Write_Delete_Read",
			fn: func(b *testing.B) {
				bigcacheConfig := bigcache.DefaultConfig(5 * time.Minute) // Большой TTL, чтобы удаление было явным
				bigcacheConfig.Verbose = false
				bigcacheConfig.Logger = nil
				bigCache, _ := bigcache.New(ctx, bigcacheConfig)

				// Предварительное заполнение
				for i := 0; i < numKeys; i++ {
					_ = bigCache.Set(generateKey(i), []byte(generateValue(i)))
				}
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := []byte(generateValue(i))
						_ = bigCache.Set(key, value)
						if i%2 == 0 { // Удаляем половину элементов
							_ = bigCache.Delete(key)
						}
						_, _ = bigCache.Get(key)
						i++
					}
				})
			},
		},
		{
			name: "FreeCache_Write_Delete_Read",
			fn: func(b *testing.B) {
				freeCache := freecache.NewCache(100 * 1024 * 1024)
				// Предварительное заполнение
				for i := 0; i < numKeys; i++ {
					_ = freeCache.Set([]byte(generateKey(i)), []byte(generateValue(i)), 300) // Большой TTL
				}
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := []byte(generateKey(i % numKeys))
						value := []byte(generateValue(i))
						_ = freeCache.Set(key, value, 300)
						if i%2 == 0 {
							freeCache.Del(key)
						}
						_, _ = freeCache.Get(key)
						i++
					}
				})
			},
		},
		{
			name: "ShardedCache_Write_Delete_Read",
			fn: func(b *testing.B) {
				shardedCache := pkg.NewShardedCache[string](ctx, 5*time.Minute, ttlDecrement) // Большой TTL
				// Предварительное заполнение
				for i := 0; i < numKeys; i++ {
					_ = shardedCache.Set(generateKey(i), generateValue(i))
				}
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						_ = shardedCache.Set(key, value)
						if i%2 == 0 {
							shardedCache.Delete(key)
						}
						_, _ = shardedCache.Get(key)
						i++
					}
				})
			},
		},
		{
			name: "ConcurrentCache_Write_Delete_Read",
			fn: func(b *testing.B) {
				concurrentCache := pkg.NewConcurrentCache[string](ctx, 5*time.Minute, ttlDecrement) // Большой TTL
				// Предварительное заполнение
				for i := 0; i < numKeys; i++ {
					_ = concurrentCache.Set(generateKey(i), generateValue(i))
				}
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						_ = concurrentCache.Set(key, value)
						if i%2 == 0 {
							concurrentCache.Delete(key)
						}
						_, _ = concurrentCache.Get(key)
						i++
					}
				})
			},
		},
		{
			name: "GoCache_Write_Delete_Read",
			fn: func(b *testing.B) {
				goCache := cache.New(5*time.Minute, 5*time.Minute) // Большой TTL
				// Предварительное заполнение
				for i := 0; i < numKeys; i++ {
					goCache.Set(generateKey(i), generateValue(i), cache.DefaultExpiration)
				}
				b.ResetTimer()
				b.RunParallel(func(pb *testing.PB) {
					i := 0
					for pb.Next() {
						key := generateKey(i % numKeys)
						value := generateValue(i)
						goCache.Set(key, value, cache.DefaultExpiration)
						if i%2 == 0 {
							goCache.Delete(key)
						}
						_, _ = goCache.Get(key)
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
