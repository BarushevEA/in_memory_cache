package pkg

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestInvalidInputs(t *testing.T) {
	testCases := []struct {
		name    string
		test    func(t *testing.T, cache ICache[string])
		wantErr bool
	}{
		{
			name: "Very long key",
			test: func(t *testing.T, cache ICache[string]) {
				key := make([]byte, 1<<16)
				err := cache.Set(string(key), "value")
				if err != nil {
					t.Errorf("Large key should be handled: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Null bytes in key",
			test: func(t *testing.T, cache ICache[string]) {
				err := cache.Set("key\x00with\x00nulls", "value")
				if err != nil {
					t.Errorf("Null bytes should be handled: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "Unicode key",
			test: func(t *testing.T, cache ICache[string]) {
				err := cache.Set("🔑", "value")
				if err != nil {
					t.Errorf("Unicode key should be handled: %v", err)
				}
			},
			wantErr: false,
		},
	}

	caches := map[string]func() ICache[string]{
		"ConcurrentCache": func() ICache[string] {
			return NewConcurrentCache[string](context.Background(), time.Second, time.Millisecond)
		},
		"ShardedCache": func() ICache[string] {
			return NewShardedCache[string](context.Background(), time.Second, time.Millisecond)
		},
	}

	for cacheName, newCache := range caches {
		for _, tc := range testCases {
			t.Run(cacheName+"/"+tc.name, func(t *testing.T) {
				cache := newCache()
				tc.test(t, cache)
			})
		}
	}
}

func TestMemoryLeaks(t *testing.T) {
	ctx := context.Background()
	cache := NewConcurrentCache[string](ctx, time.Millisecond*100, time.Millisecond*10)

	runtime.GC()
	time.Sleep(time.Millisecond * 100)

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key-%d", i)
		err := cache.Set(key, "value")
		if err != nil {
			t.Fatalf("Failed to set value: %v", err)
		}
		_, _ = cache.Get(key)
		cache.Delete(key)
	}

	cache.Clear()

	time.Sleep(time.Millisecond * 200)
	runtime.GC()
	time.Sleep(time.Millisecond * 100)

	runtime.ReadMemStats(&m2)

	heapDiff := int64(m2.HeapAlloc - m1.HeapAlloc)
	t.Logf("Heap allocation difference: %d bytes", heapDiff)

	objectsDiff := int64(m2.HeapObjects - m1.HeapObjects)
	t.Logf("Heap objects difference: %d", objectsDiff)

	const maxAcceptableBytes = 1 * 1024 * 1024
	if heapDiff > maxAcceptableBytes {
		t.Errorf("Possible memory leak detected: heap grew by %d bytes (max acceptable: %d bytes)",
			heapDiff, maxAcceptableBytes)
	}

	const maxAcceptableObjects = 1000
	if objectsDiff > maxAcceptableObjects {
		t.Errorf("Possible memory leak detected: heap objects grew by %d (max acceptable: %d)",
			objectsDiff, maxAcceptableObjects)
	}
}

func TestErrorRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cache := NewConcurrentCache[string](ctx, time.Second, time.Millisecond)

	cancel()

	newCtx := context.Background()
	cache = NewConcurrentCache[string](newCtx, time.Second, time.Millisecond)

	err := cache.Set("key", "value")
	if err != nil {
		t.Errorf("Cache should recover after context cancellation: %v", err)
	}
}

func TestDifferentTypes(t *testing.T) {
	ctx := context.Background()

	t.Run("Integer", func(t *testing.T) {
		cache := NewConcurrentCache[int](ctx, time.Second, time.Millisecond)
		err := cache.Set("key", 42)
		if err != nil {
			t.Errorf("Failed to set integer: %v", err)
		}
		if val, ok := cache.Get("key"); !ok || val != 42 {
			t.Error("Failed to get integer value")
		}
	})

	type TestStruct struct {
		Field1 string
		Field2 int
	}

	t.Run("Struct", func(t *testing.T) {
		cache := NewConcurrentCache[TestStruct](ctx, time.Second, time.Millisecond)
		value := TestStruct{Field1: "test", Field2: 42}
		err := cache.Set("key", value)
		if err != nil {
			t.Errorf("Failed to set struct: %v", err)
		}
		if val, ok := cache.Get("key"); !ok || val != value {
			t.Error("Failed to get struct value")
		}
	})

	t.Run("Pointer", func(t *testing.T) {
		cache := NewConcurrentCache[*TestStruct](ctx, time.Second, time.Millisecond)
		value := &TestStruct{Field1: "test", Field2: 42}
		err := cache.Set("key", value)
		if err != nil {
			t.Errorf("Failed to set pointer: %v", err)
		}
		if val, ok := cache.Get("key"); !ok || val != value {
			t.Error("Failed to get pointer value")
		}
	})
}

func TestHighLoad(t *testing.T) {
	ctx := context.Background()
	cache := NewShardedCache[string](ctx, time.Second, time.Millisecond)

	const goroutines = 100
	const operationsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	start := time.Now()

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				err := cache.Set(key, "value")
				if err != nil {
					t.Errorf("Set failed: %v", err)
				}
				_, _ = cache.Get(key)
				cache.Delete(key)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("High load test completed in %v", duration)
	t.Logf("Operations per second: %v", float64(goroutines*operationsPerGoroutine)/duration.Seconds())
}
