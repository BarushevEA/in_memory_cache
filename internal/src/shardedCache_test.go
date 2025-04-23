package src

import (
	"context"
	"errors"
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
