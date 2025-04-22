package src

import (
	"context"
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

	// Проверяем начальное состояние
	if l := cMap.Len(); l != 0 {
		t.Fatalf("Initial length should be 0, got %d", l)
	}

	// Устанавливаем значение
	err := cMap.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Проверяем что значение установлено
	if l := cMap.Len(); l != 1 {
		t.Fatalf("Length should be 1 after Set, got %d", l)
	}

	if val, found := cMap.Get("test-key"); !found || val != "test-value" {
		t.Fatal("Value should exist and match immediately after Set()")
	}

	// Ждем истечения TTL
	time.Sleep(ttl + 3*ttlDecrement)

	// Проверяем что значение удалено
	if val, found := cMap.Get("test-key"); found {
		t.Fatalf("Value should be expired and removed, but got %v", val)
	}

	// Проверяем финальную длину
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
			// Создаем новую карту для каждого тестового случая
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cMap := NewConcurrentMapWithTTL[string](ctx, 5*time.Second, 1*time.Second)

			// Заполняем начальными данными
			cMap.Set("key1", "value1")
			cMap.Set("key2", "value2")

			// Выполняем удаление
			cMap.Delete(tt.keyToDelete)

			// Даем небольшое время для асинхронных операций
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
