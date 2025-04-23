package src

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"testing"
	"time"
)

func TestDynamicShardedMapWithTTL_Set(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	tests := []struct {
		name    string
		key     string
		value   string
		wantErr error
	}{
		{"valid set", "key1", "value1", nil},
		{"duplicate key", "key1", "value2", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.Set(tt.key, tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Set() got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestDynamicShardedMapWithTTL_Get(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")

	tests := []struct {
		name   string
		key    string
		want   string
		wantOk bool
	}{
		{"existing key", "key1", "value1", true},
		{"non-existing key", "key2", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := cache.Get(tt.key)
			if got != tt.want || ok != tt.wantOk {
				t.Errorf("Get() got %v, want %v; gotOk %v, wantOk %v", got, tt.want, ok, tt.wantOk)
			}
		})
	}
}

func TestDynamicShardedMapWithTTL_Delete(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")

	cache.Delete("key1")
	if _, ok := cache.Get("key1"); ok {
		t.Errorf("Delete() did not remove the key")
	}

	cache.Delete("key2") // Test deleting non-existing key
}

func TestDynamicShardedMapWithTTL_Clear(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")
	_ = cache.Set("key2", "value2")

	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("Clear() did not clear the cache")
	}
}

func TestDynamicShardedMapWithTTL_Len(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	if cache.Len() != 0 {
		t.Errorf("Len() on empty cache should return 0")
	}

	_ = cache.Set("key1", "value1")
	_ = cache.Set("key2", "value2")
	if cache.Len() != 2 {
		t.Errorf("Len() got %d, want %d", cache.Len(), 2)
	}

	cache.Delete("key1")
	if cache.Len() != 1 {
		t.Errorf("Len() after delete got %d, want %d", cache.Len(), 1)
	}
}

func TestDynamicShardedMapWithTTL_Range(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")
	_ = cache.Set("key2", "value2")

	var count int
	err := cache.Range(func(k string, v string) bool {
		count++
		return true
	})

	if err != nil {
		t.Errorf("Range() returned an error: %v", err)
	}
	if count != 2 {
		t.Errorf("Range() iterated over %d items, want %d", count, 2)
	}
}

func TestDynamicShardedMapWithTTL_SetOnClosedCache(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	cache.Clear() // This will close the cache

	err := cache.Set("key1", "value1")
	if err == nil {
		t.Errorf("Set() on closed cache did not return an error")
	}
}

func TestDynamicShardedMapWithTTL_TickCollection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewDynamicShardedMapWithTTL[string](ctx, 2*time.Second, 1*time.Second)
	_ = cache.Set("key1", "value1")

	time.Sleep(3 * time.Second)
	cancel()

	if cache.Len() != 0 {
		t.Errorf("tickCollection() did not clear expired items")
	}
}

func TestNewDynamicShardedMapWithTTL_InvalidParams(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		ttl  time.Duration
		decr time.Duration
	}{
		{"negative ttl", -1 * time.Second, 1 * time.Second},
		{"zero ttl", 0, 1 * time.Second},
		{"negative decrement", 5 * time.Second, -1 * time.Second},
		{"zero decrement", 5 * time.Second, 0},
		{"decrement larger than ttl", 5 * time.Second, 6 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewDynamicShardedMapWithTTL[string](ctx, tt.ttl, tt.decr)
			err := cache.Set("test", "value")
			if err != nil {
				t.Errorf("Set() on cache with invalid params failed: %v", err)
			}
		})
	}
}

func TestDynamicShardedMapWithTTL_RangeWithBreak(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	_ = cache.Set("key1", "value1")
	_ = cache.Set("key2", "value2")
	_ = cache.Set("key3", "value3")

	var count int
	err := cache.Range(func(k string, v string) bool {
		count++
		return false
	})

	if err != nil {
		t.Errorf("Range() returned an error: %v", err)
	}
	if count != 1 {
		t.Errorf("Range() with break iterated over %d items, want %d", count, 1)
	}
}

func TestDynamicShardedMapWithTTL_RangeOnClosedCache(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")

	cache.Clear()

	err := cache.Range(func(k string, v string) bool {
		return true
	})

	if err == nil {
		t.Error("Range() on closed cache should return error")
	}
}

func TestDynamicShardedMapWithTTL_GetOnClosedCache(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	_ = cache.Set("key1", "value1")

	cache.Clear()

	value, ok := cache.Get("key1")
	if ok || value != "" {
		t.Errorf("Get() on closed cache should return zero value and false, got %v, %v", value, ok)
	}
}

func TestDynamicShardedMapWithTTL_TTLExpiration(t *testing.T) {
	ctx := context.Background()
	ttl := 2 * time.Second
	cache := NewDynamicShardedMapWithTTL[string](ctx, ttl, 1*time.Second)

	_ = cache.Set("key1", "value1")

	if val, ok := cache.Get("key1"); !ok || val != "value1" {
		t.Errorf("Value should be available immediately after set")
	}

	time.Sleep(ttl / 2)

	if val, ok := cache.Get("key1"); !ok || val != "value1" {
		t.Errorf("Value should be available before TTL expires")
	}

	timer := time.NewTimer(ttl + 500*time.Millisecond)
	<-timer.C

	maxAttempts := 5
	for i := 0; i < maxAttempts; i++ {
		if _, ok := cache.Get("key1"); !ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Errorf("Value should be removed after TTL expiration")
}

func TestDynamicShardedMapWithTTL_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				err := cache.Set(key, "value")
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}

				if _, ok := cache.Get(key); !ok {
					t.Errorf("Get failed for key: %s", key)
				}

				cache.Delete(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestDynamicShardedMapWithTTL_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	_ = cache.Set("key1", "value1")
	_ = cache.Set("key2", "value2")

	cancel()

	time.Sleep(100 * time.Millisecond)

	if cache.Len() != 0 {
		t.Errorf("Cache should be empty after context cancellation")
	}

	err := cache.Set("key3", "value3")
	if err == nil {
		t.Error("Should not be able to set values after context cancellation")
	}
}

func TestDynamicShardedMapWithTTL_EmptyKeysAndValues(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	err := cache.Set("", "value")
	if err != nil {
		t.Errorf("Setting empty key should be allowed: %v", err)
	}

	err = cache.Set("key", "")
	if err != nil {
		t.Errorf("Setting empty value should be allowed: %v", err)
	}

	if val, ok := cache.Get(""); !ok {
		t.Error("Should be able to get value for empty key")
	} else if val != "value" {
		t.Errorf("Got wrong value for empty key: %s", val)
	}
}

func TestDynamicShardedMapWithTTL_MultipleShards(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)

	keys := []string{
		"a1", "b1", "c1", "d1", "e1",
		"a2", "b2", "c2", "d2", "e2",
	}

	for _, key := range keys {
		err := cache.Set(key, "value-"+key)
		if err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	for _, key := range keys {
		if val, ok := cache.Get(key); !ok {
			t.Errorf("Failed to get key %s", key)
		} else if val != "value-"+key {
			t.Errorf("Wrong value for key %s: got %s, want %s", key, val, "value-"+key)
		}
	}
}

func BenchmarkDynamicShardedMapWithTTL_SetGet(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewDynamicShardedMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			err := cache.Set(key, "value")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = cache.Get(key)
			i++
		}
	})
}

func BenchmarkDynamicShardedMapWithTTL_HighLoad(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewDynamicShardedMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("init-key-%d", i), "value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localID := rand.Int()
		counter := 0

		for pb.Next() {
			key := fmt.Sprintf("key-%d-%d", localID, counter%100)
			switch counter % 3 {
			case 0:
				cache.Set(key, "new-value")
			case 1:
				cache.Get(key)
			case 2:
				cache.Delete(key)
			}
			counter++
		}
	})
}

func BenchmarkDynamicShardedMapWithTTL_Operations(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewDynamicShardedMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("init-key-%d", i), "value")
	}

	b.ResetTimer()

	b.Run("Set", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			id := rand.Int()
			i := 0
			for pb.Next() {
				cache.Set(fmt.Sprintf("set-key-%d-%d", id, i), "value")
				i++
			}
		})
	})

	b.Run("Get", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				cache.Get(fmt.Sprintf("init-key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				cache.Delete(fmt.Sprintf("init-key-%d", i%1000))
				i++
			}
		})
	})
}

func BenchmarkDynamicShardedMapWithTTL_Range(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewDynamicShardedMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("key-%d", i), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Range(func(key string, value string) bool {
			return true
		})
	}
}
