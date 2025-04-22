package src

import (
	"context"
	"fmt"
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

func TestConcurrentMapWithTTL_Concurrent(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 200*time.Millisecond, 50*time.Millisecond)

	// Тестируем конкурентные операции записи/чтения
	const goroutines = 10
	const operationsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 2) // для writers и readers

	// Writers
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

	// Readers
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

	// Тест 1: Значение должно существовать до истечения TTL
	cMap.Set("test1", "value1")
	time.Sleep(ttl / 2)
	if val, exists := cMap.Get("test1"); !exists {
		t.Error("Value should exist before TTL expiration")
	} else if val != "value1" {
		t.Errorf("Expected value1, got %v", val)
	}

	// Тест 2: Значение должно исчезнуть после TTL
	time.Sleep(ttl)
	if _, exists := cMap.Get("test1"); exists {
		t.Error("Value should be removed after TTL expiration")
	}

	// Тест 3: Обновление существующего значения должно сбросить TTL
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

	// Заполняем данными
	cMap.Set("key1", "value1")
	cMap.Set("key2", "value2")

	// Отменяем контекст
	cancel()

	// Даем время на очистку
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что карта пуста после отмены контекста
	if l := cMap.Len(); l != 0 {
		t.Errorf("Map should be empty after context cancellation, got length %d", l)
	}

	// Проверяем, что новые записи отвергаются
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

	// Добавляем множество элементов с разным временем жизни
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

	// Проверяем, что все элементы добавлены
	initialLen := cMap.Len()
	if initialLen != numItems {
		t.Errorf("Expected %d items, got %d", numItems, initialLen)
	}

	// Ждем достаточно времени для истечения TTL
	time.Sleep(ttl + 2*decrement)

	// Проверяем, что все элементы удалены
	finalLen := cMap.Len()
	if finalLen != 0 {
		t.Errorf("Expected 0 items after TTL, got %d", finalLen)
	}
}

func TestConcurrentMapWithTTL_ZeroTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем карту с нулевым TTL
	cMap := NewConcurrentMapWithTTL[string](ctx, 0, time.Millisecond)

	// Пробуем добавить значение
	err := cMap.Set("key", "value")
	if err != nil {
		t.Errorf("Set with zero TTL should not return error, got: %v", err)
	}

	// Значение должно быть доступно сразу после установки
	if val, exists := cMap.Get("key"); !exists || val != "value" {
		t.Error("Value should be accessible immediately after Set with zero TTL")
	}

	// Ждем некоторое время и проверяем, что значение все еще доступно
	time.Sleep(100 * time.Millisecond)
	if val, exists := cMap.Get("key"); !exists || val != "value" {
		t.Error("Value should persist with zero TTL")
	}
}

func TestConcurrentMapWithTTL_NegativeTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Создаем карту с отрицательным TTL
	cMap := NewConcurrentMapWithTTL[string](ctx, -time.Second, time.Millisecond)

	// Проверяем, что карта работает с дефолтными значениями
	err := cMap.Set("key", "value")
	if err != nil {
		t.Error("Set should work with default TTL values")
	}

	// Проверяем, что значение установлено
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
				time.Sleep(time.Microsecond) // Небольшая задержка для создания "реальных" условий
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

	// Устанавливаем начальное значение
	cMap.Set("key", "initial")

	// Ждем почти до истечения TTL
	time.Sleep(ttl - 50*time.Millisecond)

	// Обновляем значение
	cMap.Set("key", "updated")

	// Ждем больше половины TTL
	time.Sleep(ttl/2 + 10*time.Millisecond)

	// Значение должно все еще существовать
	if val, exists := cMap.Get("key"); !exists || val != "updated" {
		t.Error("Value should exist with updated content after TTL reset")
	}

	// Ждем до полного истечения нового TTL с дополнительным запасом
	time.Sleep(ttl + 100*time.Millisecond)

	// Проверяем несколько раз с небольшими интервалами
	for i := 0; i < 3; i++ {
		if _, exists := cMap.Get("key"); exists {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		// Если значение удалено, тест пройден
		return
	}
	t.Error("Value should be removed after full TTL period")
}

func TestConcurrentMapWithTTL_RangeEarlyStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	// Заполняем данными
	expectedKeys := []string{"key1", "key2", "key3", "key4"}
	for _, key := range expectedKeys {
		cMap.Set(key, "value")
	}

	visitedKeys := make([]string, 0)
	err := cMap.Range(func(key string, value string) bool {
		visitedKeys = append(visitedKeys, key)
		return len(visitedKeys) < 2 // Останавливаемся после двух элементов
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

	// Заполняем начальными данными
	cMap.Set("key1", "value1")

	// Закрываем карту
	cMap.Clear()

	// Проверяем операции после закрытия
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

	// Проверяем Get после закрытия
	if _, found := cMap.Get("key1"); found {
		t.Error("Get should return not found after cache is closed")
	}

	// Проверяем Len после закрытия
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

			// Проверяем, что карта использует дефолтные значения
			err := cMap.Set("test", "value")
			if err != nil {
				t.Errorf("Set() failed with error: %v", err)
			}

			// Проверяем, что значение доступно
			if val, exists := cMap.Get("test"); !exists || val != "value" {
				t.Error("Value should be accessible after Set with default TTL values")
			}

			// Ждем немного меньше дефолтного TTL
			time.Sleep(4 * time.Second)

			// Значение все еще должно быть доступно
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

	// Устанавливаем значение для запуска tickCollection
	err := cMap.Set("key1", "value1")
	if err != nil {
		t.Fatalf("Failed to set initial value: %v", err)
	}

	// Проверяем, что значение существует
	if _, exists := cMap.Get("key1"); !exists {
		t.Fatal("Initial value should exist")
	}

	// Ждем один тик
	time.Sleep(decrement + 10*time.Millisecond)

	// Значение должно все еще существовать
	if _, exists := cMap.Get("key1"); !exists {
		t.Error("Value should exist after one tick")
	}

	// Ждем полного TTL
	time.Sleep(ttl + decrement)

	// Значение должно быть удалено
	if _, exists := cMap.Get("key1"); exists {
		t.Error("Value should be removed after TTL expiration")
	}

	// Проверяем, что tickCollection все еще работает
	err = cMap.Set("key2", "value2")
	if err != nil {
		t.Fatalf("Failed to set second value: %v", err)
	}

	time.Sleep(ttl + decrement)

	if _, exists := cMap.Get("key2"); exists {
		t.Error("Second value should be removed after TTL expiration")
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

func BenchmarkConcurrentMapWithTTL_Range(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, time.Second, 100*time.Millisecond)

	// Предварительное заполнение данными
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

func TestConcurrentMapWithTTL_MemoryLeak(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory leak test in short mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	// Добавляем первый элемент и ждем инициализации тикера
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

		// Увеличиваем время ожидания
		time.Sleep(250 * time.Millisecond)

		// Несколько попыток проверки
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

	wg.Add(goroutines * 4) // для Set, Get, Delete и Range операций

	// Set operations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Set(key, "value")
			}
		}(i)
	}

	// Get operations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Get(key)
			}
		}(i)
	}

	// Delete operations
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cMap.Delete(key)
			}
		}(i)
	}

	// Range operations
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

	// Инициализация тикера
	err := cMap.Set("init", "value")
	if err != nil {
		t.Fatal("Failed to initialize map")
	}
	time.Sleep(50 * time.Millisecond)

	// Создаем фиксированный набор ключей
	const keysCount = 1000
	keys := make([]string, keysCount)
	for i := 0; i < keysCount; i++ {
		keys[i] = fmt.Sprintf("key-%d", i)
	}

	// Обновляем значения многократно
	for i := 0; i < 100; i++ {
		// Обновляем все ключи
		for _, key := range keys {
			cMap.Set(key, fmt.Sprintf("value-%d", i))
		}

		// Ждем истечения TTL
		time.Sleep(250 * time.Millisecond)

		// Проверяем очистку
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

	// Инициализация
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
	wg.Add(goroutines * 2) // для updaters и deleters

	// Updaters
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d", j%100) // Переиспользуем 100 ключей
				cMap.Set(key, fmt.Sprintf("value-%d-%d", id, j))
				time.Sleep(time.Millisecond) // Небольшая задержка
			}
		}(i)
	}

	// Deleters
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operations; j++ {
				key := fmt.Sprintf("key-%d", j%100)
				cMap.Delete(key)
				time.Sleep(time.Millisecond) // Небольшая задержка
			}
		}()
	}

	// Дополнительная горутина для проверки размера
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

	// Финальная проверка
	time.Sleep(250 * time.Millisecond)
	if size := cMap.Len(); size > 0 {
		t.Errorf("Expected empty map after all operations, got %d items", size)
	}
}

func BenchmarkConcurrentMapWithTTL_HighLoad(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cMap := NewConcurrentMapWithTTL[string](ctx, 100*time.Millisecond, 20*time.Millisecond)

	// Предварительное заполнение
	for i := 0; i < 1000; i++ {
		cMap.Set(fmt.Sprintf("init-key-%d", i), "value")
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Каждая горутина работает со своим диапазоном ключей
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

	// Предварительное заполнение для операций Get/Delete
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
