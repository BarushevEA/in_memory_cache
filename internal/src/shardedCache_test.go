package src

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"sync"
	"testing"
	"time"
)

// TestDynamicShardedMapWithTTL_Set tests the Set functionality of DynamicShardedMapWithTTL with TTL and ensures correctness.
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

// TestDynamicShardedMapWithTTL_Get tests the Get method of a DynamicShardedMapWithTTL, verifying correct value retrieval and existence checks.
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

// TestDynamicShardedMapWithTTL_Delete tests the Delete function in the DynamicShardedMapWithTTL for proper key removal.
// Verifies that existing keys are removed successfully and non-existent keys do not cause errors.
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

// TestDynamicShardedMapWithTTL_Clear tests the Clear method of DynamicShardedMapWithTTL to ensure it properly clears all entries.
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

// TestDynamicShardedMapWithTTL_Len verifies the Len method's correctness in counting entries in the dynamic sharded map.
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

// TestDynamicShardedMapWithTTL_Range tests the Range method of the dynamic sharded map with TTL for correct iteration functionality.
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

// TestDynamicShardedMapWithTTL_SetOnClosedCache validates that attempting to set a key-value pair on a closed cache returns an error.
func TestDynamicShardedMapWithTTL_SetOnClosedCache(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 10*time.Second, 2*time.Second)
	cache.Clear() // This will close the cache

	err := cache.Set("key1", "value1")
	if err == nil {
		t.Errorf("Set() on closed cache did not return an error")
	}
}

// TestDynamicShardedMapWithTTL_TickCollection validates the map's ability to periodically remove expired entries using TTL.
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

// TestNewDynamicShardedMapWithTTL_InvalidParams verifies that NewDynamicShardedMapWithTTL handles invalid parameters correctly.
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

// TestDynamicShardedMapWithTTL_RangeWithBreak validates the behavior of Range with early termination using a break condition.
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

// TestDynamicShardedMapWithTTL_RangeOnClosedCache verifies that the Range method returns an error when called on a closed cache.
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

// TestDynamicShardedMapWithTTL_GetOnClosedCache verifies that calling Get on a cache after it is cleared returns zero value and false.
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

// TestDynamicShardedMapWithTTL_TTLExpiration validates the TTL-based expiration functionality of the dynamic sharded map.
// Ensures that values are retrievable before the TTL expires and removed after the TTL duration elapses.
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

// TestDynamicShardedMapWithTTL_ConcurrentAccess tests concurrent access on DynamicShardedMapWithTTL to validate thread safety and TTL behavior.
// It spawns multiple goroutines to perform concurrent Set, Get, and Delete operations with a configured TTL duration.
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

// TestDynamicShardedMapWithTTL_ContextCancellation tests cancellation behavior of context in DynamicShardedMapWithTTL with TTL.
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

// TestDynamicShardedMapWithTTL_EmptyKeysAndValues validates handling of empty keys and values in a sharded map with TTL.
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

// TestDynamicShardedMapWithTTL_MultipleShards verifies the correct functionality of dynamic sharded map with TTL using multiple shards.
// Ensures keys are set correctly, can be retrieved with their values, and handles multiple keys across shards properly.
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

// BenchmarkDynamicShardedMapWithTTL_SetGet benchmarks the Set and Get operations of the dynamic sharded map with TTL in parallel.
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

// BenchmarkDynamicShardedMapWithTTL_HighLoad benchmarks the performance of DynamicShardedMapWithTTL under high-concurrency scenarios.
// It performs random Set, Get, and Delete operations with pre-populated keys while testing under parallel workloads.
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

// BenchmarkDynamicShardedMapWithTTL_Operations evaluates the performance of Set, Get, and Delete operations on a sharded cache with TTL.
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

// BenchmarkDynamicShardedMapWithTTL_Range benchmarks the Range method of DynamicShardedMapWithTTL with 1000 pre-set keys.
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

func TestDynamicShardedMapWithTTL_Metrics(t *testing.T) {
	t.Run("should track metrics across shards", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		beforeCreate := time.Now()
		err := cache.Set("key1", "value1")
		afterCreate := time.Now()
		require.NoError(t, err)

		// Act
		value, createdAt, setCount, getCount, exists := cache.GetNodeValueWithMetrics("key1")

		// Assert
		assert.True(t, exists)
		assert.Equal(t, "value1", value)
		assert.True(t, createdAt.After(beforeCreate) || createdAt.Equal(beforeCreate))
		assert.True(t, createdAt.Before(afterCreate) || createdAt.Equal(afterCreate))
		assert.Equal(t, uint32(1), setCount)
		assert.Equal(t, uint32(0), getCount)
	})

	t.Run("should correctly count operations for multiple keys", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		// Act
		err := cache.Set("key1", "value1")
		require.NoError(t, err)
		err = cache.Set("key2", "value2")
		require.NoError(t, err)

		_, _ = cache.Get("key1")
		_, _ = cache.Get("key1")
		_, _ = cache.Get("key2")

		// Assert
		_, _, setCount1, getCount1, _ := cache.GetNodeValueWithMetrics("key1")
		_, _, setCount2, getCount2, _ := cache.GetNodeValueWithMetrics("key2")

		assert.Equal(t, uint32(1), setCount1)
		assert.Equal(t, uint32(2), getCount1)
		assert.Equal(t, uint32(1), setCount2)
		assert.Equal(t, uint32(1), getCount2)
	})

	t.Run("should maintain metrics when using RangeWithMetrics", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		err := cache.Set("key1", "value1")
		require.NoError(t, err)
		_, _ = cache.Get("key1")

		collected := make(map[string]struct {
			value     string
			setCount  uint32
			getCount  uint32
			createdAt time.Time
		})

		// Act
		err = cache.RangeWithMetrics(func(key string, value string, createdAt time.Time, setCount uint32, getCount uint32) bool {
			collected[key] = struct {
				value     string
				setCount  uint32
				getCount  uint32
				createdAt time.Time
			}{
				value:     value,
				setCount:  setCount,
				getCount:  getCount,
				createdAt: createdAt,
			}
			return true
		})

		// Assert
		require.NoError(t, err)
		assert.Len(t, collected, 1)
		metrics := collected["key1"]
		assert.Equal(t, "value1", metrics.value)
		assert.Equal(t, uint32(1), metrics.setCount)
		assert.Equal(t, uint32(1), metrics.getCount)
		assert.False(t, metrics.createdAt.IsZero())
	})

	t.Run("should handle metrics for non-existent keys", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		// Act
		value, createdAt, setCount, getCount, exists := cache.GetNodeValueWithMetrics("non-existent")

		// Assert
		assert.False(t, exists)
		assert.Empty(t, value)
		assert.True(t, createdAt.IsZero())
		assert.Equal(t, uint32(0), setCount)
		assert.Equal(t, uint32(0), getCount)
	})

	t.Run("should reset metrics on Clear", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

		err := cache.Set("key1", "value1")
		require.NoError(t, err)
		_, _ = cache.Get("key1")

		// Act
		cache.Clear()
		value, createdAt, setCount, getCount, exists := cache.GetNodeValueWithMetrics("key1")

		// Assert
		assert.False(t, exists)
		assert.Empty(t, value)
		assert.True(t, createdAt.IsZero())
		assert.Equal(t, uint32(0), setCount)
		assert.Equal(t, uint32(0), getCount)
	})

	t.Run("should handle concurrent metric operations", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)
		const goroutines = 10
		const operationsPerGoroutine = 50

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			key := fmt.Sprintf("key%d", i)
			err := cache.Set(key, "initial")
			require.NoError(t, err)
		}

		// Act
		for i := 0; i < goroutines; i++ {
			go func(routineID int) {
				defer wg.Done()
				key := fmt.Sprintf("key%d", routineID)

				for j := 0; j < operationsPerGoroutine; j++ {
					if j%2 == 0 {
						err := cache.Set(key, fmt.Sprintf("value%d-%d", routineID, j))
						require.NoError(t, err)
					} else {
						_, ok := cache.Get(key)
						require.True(t, ok)
					}
					time.Sleep(time.Microsecond)
				}
			}(i)
		}

		wg.Wait()

		// Assert
		for i := 0; i < goroutines; i++ {
			key := fmt.Sprintf("key%d", i)
			_, _, setCount, getCount, exists := cache.GetNodeValueWithMetrics(key)
			assert.True(t, exists)
			expectedOps := uint32(operationsPerGoroutine / 2)
			assert.GreaterOrEqual(t, setCount, expectedOps)
			assert.GreaterOrEqual(t, getCount, expectedOps)
		}
	})
}

func TestDynamicShardedMapWithTTL_SetBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := cache.SetBatch(tt.batch)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			// Verify all values were set correctly
			for k, v := range tt.batch {
				val, exists := cache.Get(k)
				assert.True(t, exists)
				assert.Equal(t, v, val)
			}
		})
	}

	// Test with closed cache
	t.Run("closed cache", func(t *testing.T) {
		cache.Clear()
		err := cache.SetBatch(map[string]string{"key": "value"})
		assert.Error(t, err)
		assert.Equal(t, "cache is closed", err.Error())
	})
}

func TestDynamicShardedMapWithTTL_GetBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Initialize test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	tests := []struct {
		name     string
		keys     []string
		expected []*BatchNode[string]
		wantErr  bool
	}{
		{
			name: "get existing keys",
			keys: []string{"key1", "key2"},
			expected: []*BatchNode[string]{
				{Key: "key1", Value: "value1", Exists: true},
				{Key: "key2", Value: "value2", Exists: true},
			},
			wantErr: false,
		},
		{
			name: "get non-existent key",
			keys: []string{"nonexistent"},
			expected: []*BatchNode[string]{
				{Key: "nonexistent", Value: "", Exists: false},
			},
			wantErr: false,
		},
		{
			name:     "empty keys list",
			keys:     []string{},
			expected: []*BatchNode[string]{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := cache.GetBatch(tt.keys)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(result))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.Key, result[i].Key)
				assert.Equal(t, expected.Value, result[i].Value)
				assert.Equal(t, expected.Exists, result[i].Exists)
			}
		})
	}

	// Test with closed cache
	t.Run("closed cache", func(t *testing.T) {
		cache.Clear()
		result, err := cache.GetBatch([]string{"key1"})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "cache is closed", err.Error())
	})
}

func TestDynamicShardedMapWithTTL_GetBatchWithMetrics(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Initialize test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	// Generate some metrics
	_, _ = cache.Get("key1")
	_, _ = cache.Get("key1")

	tests := []struct {
		name        string
		keys        []string
		checkResult func(t *testing.T, metrics []*Metric[string])
		wantErr     bool
	}{
		{
			name: "check metrics for existing keys",
			keys: []string{"key1", "key2"},
			checkResult: func(t *testing.T, metrics []*Metric[string]) {
				assert.Equal(t, 2, len(metrics))

				// Check key1 with multiple gets
				key1 := metrics[0]
				assert.Equal(t, "key1", key1.Key)
				assert.Equal(t, "value1", key1.Value)
				assert.True(t, key1.Exists)
				assert.False(t, key1.TimeCreated.IsZero())
				assert.Equal(t, uint32(1), key1.SetCount)
				assert.Equal(t, uint32(2), key1.GetCount)

				// Check key2 with no gets
				key2 := metrics[1]
				assert.Equal(t, "key2", key2.Key)
				assert.Equal(t, "value2", key2.Value)
				assert.True(t, key2.Exists)
				assert.False(t, key2.TimeCreated.IsZero())
				assert.Equal(t, uint32(1), key2.SetCount)
				assert.Equal(t, uint32(0), key2.GetCount)
			},
			wantErr: false,
		},
		{
			name: "check non-existent key",
			keys: []string{"nonexistent"},
			checkResult: func(t *testing.T, metrics []*Metric[string]) {
				assert.Equal(t, 1, len(metrics))
				metric := metrics[0]
				assert.Equal(t, "nonexistent", metric.Key)
				assert.False(t, metric.Exists)
				assert.Empty(t, metric.Value)
				assert.True(t, metric.TimeCreated.IsZero())
				assert.Equal(t, uint32(0), metric.SetCount)
				assert.Equal(t, uint32(0), metric.GetCount)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := cache.GetBatchWithMetrics(tt.keys)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			tt.checkResult(t, metrics)
		})
	}

	// Test with closed cache
	t.Run("closed cache", func(t *testing.T) {
		cache.Clear()
		metrics, err := cache.GetBatchWithMetrics([]string{"key1"})
		assert.Error(t, err)
		assert.Nil(t, metrics)
		assert.Equal(t, "cache is closed", err.Error())
	})
}

func TestDynamicShardedMapWithTTL_DeleteBatch(t *testing.T) {
	ctx := context.Background()
	cache := NewDynamicShardedMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

	// Initialize test data
	initialData := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := cache.SetBatch(initialData)
	assert.NoError(t, err)

	tests := []struct {
		name         string
		keysToDelete []string
		checkResult  func(t *testing.T, c ICacheInMemory[string])
	}{
		{
			name:         "delete existing keys",
			keysToDelete: []string{"key1", "key2"},
			checkResult: func(t *testing.T, c ICacheInMemory[string]) {
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
			checkResult: func(t *testing.T, c ICacheInMemory[string]) {
				// Verify state remains unchanged
				val, exists := c.Get("key3")
				assert.True(t, exists)
				assert.Equal(t, "value3", val)
				assert.Equal(t, 1, c.Len())
			},
		},
		{
			name:         "empty delete list",
			keysToDelete: []string{},
			checkResult: func(t *testing.T, c ICacheInMemory[string]) {
				assert.Equal(t, 1, c.Len())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.DeleteBatch(tt.keysToDelete)
			tt.checkResult(t, cache)
		})
	}

	// Test with closed cache
	t.Run("closed cache", func(t *testing.T) {
		cache.Clear()
		cache.DeleteBatch([]string{"key1"})
		// Should not panic and should be a no-op
		assert.Equal(t, 0, cache.Len())
	})
}
