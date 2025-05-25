package src

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/types"
	"github.com/stretchr/testify/assert"
	"math/rand/v2"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestConcurrentMapWithTTL_SetAndGet(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		ttl          time.Duration
		ttlDecrement time.Duration
		expectFound  bool
	}{
		{"valid key-value", "key1", "value1", 5 * time.Second, 1 * time.Second, true},
		//{"expired key-value", "key2", "value2", 1 * time.Second, 500 * time.Millisecond, false},
		{"set key after closing", "key3", "value3", 5 * time.Second, 1 * time.Second, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cMap := NewConcurrentMapWithTTL[string](ctx, tt.ttl, tt.ttlDecrement)

			err := cMap.Set(tt.key, tt.value)
			if err != nil {
				t.Errorf("Set() error = %v", err)
			}

			if tt.name == "set key after closing" {
				cMap.Clear()
				err := cMap.Set(tt.key, tt.value)
				if err == nil {
					t.Errorf("Set() should return an error when the cache is closed")
				}
				return
			}

			if tt.name == "expired key-value" {
				time.Sleep(2 * time.Second)
			}

			gotValue, found := cMap.Get(tt.key)
			if found != tt.expectFound {
				t.Errorf("Get() found = %v; expected found = %v", found, tt.expectFound)
			}
			if found && gotValue != tt.value {
				t.Errorf("Get() got = %v; expected = %v", gotValue, tt.value)
			}
		})
	}
}

func TestConcurrentMapWithTTL_ExpiredValue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ttl := 100 * time.Millisecond
	ttlDecrement := 50 * time.Millisecond

	cMap := NewConcurrentMapWithTTL[string](ctx, ttl, ttlDecrement)

	if l := cMap.Len(); l != 0 {
		t.Fatalf("Initial length should be 0, got %d", l)
	}

	err := cMap.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	if l := cMap.Len(); l != 1 {
		t.Fatalf("Length should be 1 after Set, got %d", l)
	}

	if val, found := cMap.Get("test-key"); !found || val != "test-value" {
		t.Fatal("Value should exist and match immediately after Set()")
	}

	time.Sleep(ttl + 3*ttlDecrement)

	if val, found := cMap.Get("test-key"); found {
		t.Fatalf("Value should be expired and removed, but got %v", val)
	}

	if l := cMap.Len(); l != 0 {
		t.Fatalf("Final length should be 0, got %d", l)
	}
}

func TestConcurrentMapWithTTL_Delete(t *testing.T) {
	tests := []struct {
		name        string
		keyToDelete string
		remaining   int
	}{
		{"delete existing key", "key1", 1},
		{"delete non-existing key", "key3", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cMap := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

			cMap.Set("key1", "value1")
			cMap.Set("key2", "value2")

			cMap.Delete(tt.keyToDelete)

			time.Sleep(50 * time.Millisecond)

			if got := cMap.Len(); got != tt.remaining {
				t.Errorf("Len() = %v; expected = %v", got, tt.remaining)
			}
		})
	}
}

func TestConcurrentMapWithTTL_Clear(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cMap := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
	cMap.Set("key1", "value1")
	cMap.Set("key2", "value2")

	cMap.Clear()
	if got := cMap.Len(); got != 0 {
		t.Errorf("Clear() Len() = %v; expected = 0", got)
	}
}

func TestConcurrentMapWithTTL_Range(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cMap := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
	cMap.Set("key1", "value1")
	cMap.Set("key2", "value2")

	keys := make(map[string]bool)
	err := cMap.Range(func(key string, value string) bool {
		keys[key] = true
		return true
	})

	if err != nil {
		t.Errorf("Range() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Range() keys count = %v; expected = 2", len(keys))
	}
}

func TestConcurrentMapWithTTL_Len(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cMap := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	if got := cMap.Len(); got != 0 {
		t.Errorf("Len() = %v; expected = 0", got)
	}

	cMap.Set("key1", "value1")
	cMap.Set("key2", "value2")

	if got := cMap.Len(); got != 2 {
		t.Errorf("Len() = %v; expected = 2", got)
	}
}

func TestConcurrentMapWithTTL_Concurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 200*time.Millisecond, 50*time.Millisecond)

	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				err := cMap.Set(key, "value")
				if err != nil {
					t.Errorf("Set error: %v", err)
				}
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, _ = cMap.Get(key)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentMapWithTTL_TTLBehavior(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ttl := 100 * time.Millisecond
	decrement := 20 * time.Millisecond
	cMap := NewConcurrentMapWithTTL[string](ctx, ttl, decrement)

	cMap.Set("test1", "value1")
	time.Sleep(ttl / 2)
	if val, exists := cMap.Get("test1"); !exists {
		t.Error("Value should exist before TTL expiration")
	} else if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	time.Sleep(ttl)
	if _, exists := cMap.Get("test1"); exists {
		t.Error("Value should be removed after TTL expiration")
	}

	cMap.Set("test2", "initial")
	time.Sleep(ttl / 2)
	cMap.Set("test2", "updated")
	time.Sleep(ttl / 2)
	if val, exists := cMap.Get("test2"); !exists || val != "updated" {
		t.Error("Value should exist with updated content after TTL reset")
	}
}

func TestConcurrentMapWithTTL_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	cMap.Set("key1", "value1")
	cMap.Set("key2", "value2")

	cancel()

	time.Sleep(200 * time.Millisecond)

	if l := cMap.Len(); l != 0 {
		t.Errorf("Map should be empty after context cancellation, got length %d", l)
	}

	err := cMap.Set("new-key", "new-value")
	if err == nil {
		t.Error("Set should return error after context cancellation")
	}
}

func TestConcurrentMapWithTTL_ConcurrentExpiration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ttl := 150 * time.Millisecond
	decrement := 50 * time.Millisecond
	cMap := NewConcurrentMapWithTTL[string](ctx, ttl, decrement)

	const numItems = 100
	var wg sync.WaitGroup
	wg.Add(numItems)

	for i := 0; i < numItems; i++ {
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			cMap.Set(key, fmt.Sprintf("value-%d", id))
		}(i)
	}

	wg.Wait()

	initialLen := cMap.Len()
	if initialLen != numItems {
		t.Errorf("Expected %d items, got %d", numItems, initialLen)
	}

	time.Sleep(ttl + 2*decrement)

	finalLen := cMap.Len()
	if finalLen != 0 {
		t.Errorf("Expected 0 items after TTL, got %d", finalLen)
	}
}

func TestConcurrentMapWithTTL_ZeroTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 0, time.Millisecond)

	err := cMap.Set("key", "value")
	if err != nil {
		t.Errorf("Set with zero TTL should not return error, got: %v", err)
	}

	if val, exists := cMap.Get("key"); !exists || val != "value" {
		t.Error("Value should be accessible immediately after Set with zero TTL")
	}

	time.Sleep(100 * time.Millisecond)
	if val, exists := cMap.Get("key"); !exists || val != "value" {
		t.Error("Value should persist with zero TTL")
	}
}

func TestConcurrentMapWithTTL_NegativeTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, -time.Second, time.Millisecond)

	err := cMap.Set("key", "value")
	if err != nil {
		t.Error("Set should work with default TTL values")
	}

	if val, exists := cMap.Get("key"); !exists || val != "value" {
		t.Error("Value should be retrievable after Set with default TTL")
	}
}

func TestConcurrentMapWithTTL_Stress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	const (
		numGoroutines = 20
		numOperations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // writers, readers, deleters

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_ = cMap.Set(key, "value")
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, _ = cMap.Get(key)
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Deleters
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Delete(key)
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()
}

func TestConcurrentMapWithTTL_TTLReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ttl := 200 * time.Millisecond
	cMap := NewConcurrentMapWithTTL[string](ctx, ttl, 50*time.Millisecond)

	cMap.Set("key", "initial")

	time.Sleep(ttl - 50*time.Millisecond)

	cMap.Set("key", "updated")

	time.Sleep(ttl/2 + 10*time.Millisecond)

	if val, exists := cMap.Get("key"); !exists || val != "updated" {
		t.Error("Value should exist with updated content after TTL reset")
	}

	time.Sleep(ttl + 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		if _, exists := cMap.Get("key"); exists {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		return
	}
	t.Error("Value should be removed after full TTL period")
}

func TestConcurrentMapWithTTL_RangeEarlyStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	expectedKeys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range expectedKeys {
		cMap.Set(key, "value")
	}

	visitedKeys := make([]string, 0)
	err := cMap.Range(func(key string, value string) bool {
		visitedKeys = append(visitedKeys, key)
		return len(visitedKeys) < 2
	})

	if err != nil {
		t.Errorf("Range() returned error: %v", err)
	}

	if len(visitedKeys) != 2 {
		t.Errorf("Range() should stop after 2 elements, visited %d elements", len(visitedKeys))
	}
}

func TestConcurrentMapWithTTL_OperationsAfterClose(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	cMap.Set("key1", "value1")

	cMap.Clear()

	tests := []struct {
		name string
		op   func() error
	}{
		{
			name: "Set after close",
			op: func() error {
				return cMap.Set("key2", "value2")
			},
		},
		{
			name: "Range after close",
			op: func() error {
				return cMap.Range(func(key string, value string) bool {
					return true
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.op()
			if err == nil {
				t.Error("Expected error for operation on closed cache")
			}
		})
	}

	if _, found := cMap.Get("key1"); found {
		t.Error("Get should return not found after cache is closed")
	}

	if l := cMap.Len(); l != 0 {
		t.Errorf("Len should return 0 after close, got %d", l)
	}
}

func TestConcurrentMapWithTTL_InvalidTTLParameters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tests := []struct {
		name         string
		ttl          time.Duration
		ttlDecrement time.Duration
	}{
		{"negative TTL", -time.Second, time.Second},
		{"zero TTL", 0, time.Second},
		{"negative decrement", time.Second, -time.Second},
		{"zero decrement", time.Second, 0},
		{"decrement larger than TTL", time.Second, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cMap := NewConcurrentMapWithTTL[string](ctx, tt.ttl, tt.ttlDecrement)

			err := cMap.Set("test", "value")
			if err != nil {
				t.Errorf("Set() failed with error: %v", err)
			}

			if val, exists := cMap.Get("test"); !exists || val != "value" {
				t.Error("Value should be accessible after Set with default TTL values")
			}

			time.Sleep(4 * time.Second)

			if _, exists := cMap.Get("test"); !exists {
				t.Error("Value should still exist before default TTL expiration")
			}
		})
	}
}

func TestConcurrentMapWithTTL_TickCollection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ttl := 300 * time.Millisecond
	decrement := 100 * time.Millisecond
	cMap := NewConcurrentMapWithTTL[string](ctx, ttl, decrement)

	err := cMap.Set("key1", "value1")
	if err != nil {
		t.Fatalf("Failed to set initial value: %v", err)
	}

	if _, exists := cMap.Get("key1"); !exists {
		t.Fatal("Initial value should exist")
	}

	time.Sleep(decrement + 10*time.Millisecond)

	if _, exists := cMap.Get("key1"); !exists {
		t.Error("Value should exist after one tick")
	}

	time.Sleep(ttl + decrement)

	if _, exists := cMap.Get("key1"); exists {
		t.Error("Value should be removed after TTL expiration")
	}

	err = cMap.Set("key2", "value2")
	if err != nil {
		t.Fatalf("Failed to set second value: %v", err)
	}

	time.Sleep(ttl + decrement)

	if _, exists := cMap.Get("key2"); exists {
		t.Error("Second value should be removed after TTL expiration")
	}
}

func TestConcurrentMapWithTTL_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	err := cMap.Set("init", "value")
	if err != nil {
		t.Fatal("Failed to initialize map")
	}
	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 100; i++ {
		for j := 0; j < 1000; j++ {
			key := fmt.Sprintf("key-%d-%d", i, j)
			cMap.Set(key, "value")
		}

		time.Sleep(250 * time.Millisecond)

		for attempt := 0; attempt < 3; attempt++ {
			if l := cMap.Len(); l > 0 {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}

		if l := cMap.Len(); l > 0 {
			t.Errorf("Iteration %d: Expected map to be empty, got %d items", i, l)
		}

		runtime.GC()
	}
}

func TestConcurrentMapWithTTL_RaceCondition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 200*time.Millisecond, 50*time.Millisecond)

	var wg sync.WaitGroup
	operations := 1000
	goroutines := 10

	wg.Add(goroutines * 4)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Set(key, "value")
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Get(key)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Delete(key)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operations/10; j++ { // Меньше Range операций, так как они тяжелее
				cMap.Range(func(key string, value string) bool {
					return true
				})
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentMapWithTTL_UpdateMemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping update memory leak test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	err := cMap.Set("init", "value")
	if err != nil {
		t.Fatal("Failed to initialize map")
	}
	time.Sleep(50 * time.Millisecond)

	const keysCount = 1000
	keys := make([]string, keysCount)
	for i := 0; i < keysCount; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	for i := 0; i < 100; i++ {
		for _, key := range keys {
			cMap.Set(key, fmt.Sprintf("value-%d", i))
		}

		time.Sleep(250 * time.Millisecond)

		for attempt := 0; attempt < 3; attempt++ {
			if l := cMap.Len(); l > 0 {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			break
		}

		if l := cMap.Len(); l > 0 {
			t.Errorf("Iteration %d: Expected map to be empty after updates, got %d items", i, l)
		}

		runtime.GC()
	}
}

func TestConcurrentMapWithTTL_ConcurrentUpdateDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	err := cMap.Set("init", "value")
	if err != nil {
		t.Fatal("Failed to initialize map")
	}
	time.Sleep(50 * time.Millisecond)

	const (
		goroutines = 10
		operations = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d", j%100)
				cMap.Set(key, fmt.Sprintf("value-%d-%d", id, j))
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d", j%100)
				cMap.Delete(key)
				time.Sleep(time.Millisecond)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		maxSize := 0
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				t.Logf("Maximum observed size: %d", maxSize)
				return
			case <-ticker.C:
				if size := cMap.Len(); size > maxSize {
					maxSize = size
				}
			}
		}
	}()

	wg.Wait()
	close(done)

	time.Sleep(250 * time.Millisecond)
	if size := cMap.Len(); size > 0 {
		t.Errorf("Expected empty map after all operations, got %d items", size)
	}
}

func BenchmarkConcurrentMapWithTTL_SetGet(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key-%d", i)
			err := cMap.Set(key, "value")
			if err != nil {
				b.Fatal(err)
			}
			_, _ = cMap.Get(key)
			i++
		}
	})
}

func BenchmarkConcurrentMapWithTTL_HighLoad(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cMap.Set(fmt.Sprintf("init-key-%d", i), "value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		localID := rand.Int()
		counter := 0

		for pb.Next() {
			key := fmt.Sprintf("key-%d-%d", localID, counter%100)
			switch counter % 3 {
			case 0:
				// Set operation
				cMap.Set(key, "new-value")
			case 1:
				// Get operation
				cMap.Get(key)
			case 2:
				// Delete operation
				cMap.Delete(key)
			}
			counter++
		}
	})
}

func BenchmarkConcurrentMapWithTTL_Operations(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cMap.Set(fmt.Sprintf("init-key-%d", i), "value")
	}

	b.ResetTimer()

	b.Run("Set", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			id := rand.Int()
			i := 0
			for pb.Next() {
				cMap.Set(fmt.Sprintf("set-key-%d-%d", id, i), "value")
				i++
			}
		})
	})

	b.Run("Get", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				cMap.Get(fmt.Sprintf("init-key-%d", i%1000))
				i++
			}
		})
	})

	b.Run("Delete", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				cMap.Delete(fmt.Sprintf("init-key-%d", i%1000))
				i++
			}
		})
	})
}

func BenchmarkConcurrentMapWithTTL_Range(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	for i := 0; i < 1000; i++ {
		cMap.Set(fmt.Sprintf("key-%d", i), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cMap.Range(func(key string, value string) bool {
			return true
		})
	}
}

func TestConcurrentMapWithTTL_Metrics(t *testing.T) {
	t.Run("should track creation time for new entries", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		beforeSet := time.Now()

		// Act
		_ = cache.Set("test-key", "test-value")
		time.Sleep(time.Millisecond)
		value, createdAt, _, _, exists := cache.GetNodeValueWithMetrics("test-key")

		// Assert
		assert.True(t, exists)
		assert.Equal(t, "test-value", value)
		assert.True(t, createdAt.After(beforeSet) || createdAt.Equal(beforeSet))
		assert.True(t, createdAt.Before(time.Now()))
	})

	t.Run("should correctly count get operations", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		_ = cache.Set("test-key", "test-value")

		// Act
		for i := 0; i < 5; i++ {
			_, _ = cache.Get("test-key")
		}
		_, _, _, getCount, exists := cache.GetNodeValueWithMetrics("test-key")

		// Assert
		assert.True(t, exists)
		assert.Equal(t, uint32(5), getCount)
	})

	t.Run("should correctly count set operations", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		// Act
		_ = cache.Set("test-key", "value1")
		_ = cache.Set("test-key", "value2")
		_ = cache.Set("test-key", "value3")
		_, _, setCount, _, exists := cache.GetNodeValueWithMetrics("test-key")

		// Assert
		assert.True(t, exists)
		assert.Equal(t, uint32(3), setCount)
	})

	t.Run("should track separate metrics for different keys", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		// Act
		_ = cache.Set("key1", "value1")
		_ = cache.Set("key2", "value2")

		_, _ = cache.Get("key1")
		_, _ = cache.Get("key1")
		_, _ = cache.Get("key2")

		_, _, setCount1, getCount1, _ := cache.GetNodeValueWithMetrics("key1")
		_, _, setCount2, getCount2, _ := cache.GetNodeValueWithMetrics("key2")

		// Assert
		assert.Equal(t, uint32(1), setCount1)
		assert.Equal(t, uint32(2), getCount1)
		assert.Equal(t, uint32(1), setCount2)
		assert.Equal(t, uint32(1), getCount2)
	})

	t.Run("should correctly handle metrics with RangeWithMetrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		_ = cache.Set("key1", "value1")
		_ = cache.Set("key1", "value2") // Увеличивает счетчик setCount
		_, _ = cache.Get("key1")

		// Act
		var foundKey string
		var foundValue string
		var foundSetCount uint32
		var foundGetCount uint32

		_ = cache.RangeWithMetrics(func(key string, value string, createdAt time.Time, setCount uint32, getCount uint32) bool {
			foundKey = key
			foundValue = value
			foundSetCount = setCount
			foundGetCount = getCount
			return true
		})

		// Assert
		assert.Equal(t, "key1", foundKey)
		assert.Equal(t, "value2", foundValue)
		assert.Equal(t, uint32(2), foundSetCount)
		assert.Equal(t, uint32(1), foundGetCount)
	})

	t.Run("should reset metrics on Delete", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		_ = cache.Set("test-key", "test-value")
		_, _ = cache.Get("test-key")

		// Act
		cache.Delete("test-key")
		_, _, _, _, exists := cache.GetNodeValueWithMetrics("test-key")

		// Assert
		assert.False(t, exists)
	})

	t.Run("should reset metrics on Clear", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		_ = cache.Set("key1", "value1")
		_ = cache.Set("key2", "value2")

		// Act
		cache.Clear()
		_, _, _, _, exists1 := cache.GetNodeValueWithMetrics("key1")
		_, _, _, _, exists2 := cache.GetNodeValueWithMetrics("key2")

		// Assert
		assert.False(t, exists1)
		assert.False(t, exists2)
		assert.Equal(t, 0, cache.Len())
	})

	t.Run("should not affect metrics when getting non-existent key", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		_ = cache.Set("existing-key", "value")
		_ = cache.Set("existing-key", "new-value")

		// Act
		_, existingOk := cache.Get("non-existent-key")
		_, _, setCount, getCount, _ := cache.GetNodeValueWithMetrics("existing-key")

		// Assert
		assert.False(t, existingOk)
		assert.Equal(t, uint32(2), setCount)
		assert.Equal(t, uint32(0), getCount)
	})
}

func TestConcurrentMapWithTTL_SetBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	testCases := []struct {
		name    string
		batch   map[string]string
		wantErr bool
	}{
		{
			name: "successful multiple items addition",
			batch: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			wantErr: false,
		},
		{
			name:    "empty batch",
			batch:   map[string]string{},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := cache.SetBatch(tc.batch)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify all values were set correctly
			for k, v := range tc.batch {
				val, exists := cache.Get(k)
				assert.True(t, exists)
				assert.Equal(t, v, val)
			}
		})
	}
}

func TestConcurrentMapWithTTL_GetBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Prepare test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	testCases := []struct {
		name     string
		keys     []string
		expected []*types.BatchNode[string]
	}{
		{
			name: "get existing keys",
			keys: []string{"key1", "key2"},
			expected: []*types.BatchNode[string]{
				{Key: "key1", Value: "value1", Exists: true},
				{Key: "key2", Value: "value2", Exists: true},
			},
		},
		{
			name: "get non-existent key",
			keys: []string{"nonexistent"},
			expected: []*types.BatchNode[string]{
				{Key: "nonexistent", Value: "", Exists: false},
			},
		},
		{
			name:     "empty keys list",
			keys:     []string{},
			expected: []*types.BatchNode[string]{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := cache.GetBatch(tc.keys)
			assert.NoError(t, err)
			assert.Equal(t, len(tc.expected), len(result))

			for i, expected := range tc.expected {
				assert.Equal(t, expected.Key, result[i].Key)
				assert.Equal(t, expected.Value, result[i].Value)
				assert.Equal(t, expected.Exists, result[i].Exists)
			}
		})
	}
}

func TestConcurrentMapWithTTL_GetBatchWithMetrics(t *testing.T) {
	ctx := context.Background()
	cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Prepare test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	// Make some accesses to generate metrics
	cache.Get("key1")
	cache.Get("key1")

	testCases := []struct {
		name        string
		keys        []string
		checkResult func(t *testing.T, metrics []*types.Metric[string])
	}{
		{
			name: "check metrics for existing keys",
			keys: []string{"key1", "key2"},
			checkResult: func(t *testing.T, metrics []*types.Metric[string]) {
				assert.Equal(t, 2, len(metrics))

				// Check key1
				assert.Equal(t, "key1", metrics[0].Key)
				assert.Equal(t, "value1", metrics[0].Value)
				assert.True(t, metrics[0].Exists)
				assert.False(t, metrics[0].TimeCreated.IsZero())
				assert.Equal(t, uint32(1), metrics[0].SetCount)
				assert.Equal(t, uint32(2), metrics[0].GetCount)

				// Check key2
				assert.Equal(t, "key2", metrics[1].Key)
				assert.Equal(t, "value2", metrics[1].Value)
				assert.True(t, metrics[1].Exists)
				assert.False(t, metrics[1].TimeCreated.IsZero())
				assert.Equal(t, uint32(1), metrics[1].SetCount)
				assert.Equal(t, uint32(0), metrics[1].GetCount)
			},
		},
		{
			name: "check non-existent key",
			keys: []string{"nonexistent"},
			checkResult: func(t *testing.T, metrics []*types.Metric[string]) {
				assert.Equal(t, 1, len(metrics))
				assert.Equal(t, "nonexistent", metrics[0].Key)
				assert.False(t, metrics[0].Exists)
				assert.Empty(t, metrics[0].Value)
				assert.True(t, metrics[0].TimeCreated.IsZero())
				assert.Equal(t, uint32(0), metrics[0].SetCount)
				assert.Equal(t, uint32(0), metrics[0].GetCount)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			metrics, err := cache.GetBatchWithMetrics(tc.keys)
			assert.NoError(t, err)
			tc.checkResult(t, metrics)
		})
	}
}

func TestConcurrentMapWithTTL_DeleteBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Prepare test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	testCases := []struct {
		name         string
		keysToDelete []string
		checkResult  func(t *testing.T, c types.ICacheInMemory[string])
	}{
		{
			name:         "delete existing keys",
			keysToDelete: []string{"key1", "key2"},
			checkResult: func(t *testing.T, c types.ICacheInMemory[string]) {
				// Verify keys are deleted
				_, exists := c.Get("key1")
				assert.False(t, exists)
				_, exists = c.Get("key2")
				assert.False(t, exists)
				// Verify remaining key
				val, exists := c.Get("key3")
				assert.True(t, exists)
				assert.Equal(t, "value3", val)
			},
		},
		{
			name:         "delete non-existent keys",
			keysToDelete: []string{"nonexistent1", "nonexistent2"},
			checkResult: func(t *testing.T, c types.ICacheInMemory[string]) {
				// Verify remaining key still exists
				val, exists := c.Get("key3")
				assert.True(t, exists)
				assert.Equal(t, "value3", val)
				assert.Equal(t, 1, c.Len())
			},
		},
		{
			name:         "empty keys list",
			keysToDelete: []string{},
			checkResult: func(t *testing.T, c types.ICacheInMemory[string]) {
				assert.Equal(t, 1, c.Len())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache.DeleteBatch(tc.keysToDelete)
			tc.checkResult(t, cache)
		})
	}
}
