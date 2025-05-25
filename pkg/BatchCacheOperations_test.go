package pkg

import (
	"context"
	"github.com/BarushevEA/in_memory_cache/types"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBatchOperations(t *testing.T) {
	type TestStruct struct {
		ID    int
		Value string
	}

	testData := map[string]TestStruct{
		"key1": {ID: 1, Value: "value1"},
		"key2": {ID: 2, Value: "value2"},
		"key3": {ID: 3, Value: "value3"},
	}

	testKeys := []string{"key1", "key2", "key3", "nonexistent"}

	tests := []struct {
		name      string
		cacheFunc func(context.Context, time.Duration, time.Duration) types.ICacheInMemory[TestStruct]
	}{
		{
			name:      "ConcurrentCache",
			cacheFunc: NewConcurrentCache[TestStruct],
		},
		{
			name:      "ShardedCache",
			cacheFunc: NewShardedCache[TestStruct],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cache := tt.cacheFunc(ctx, time.Second, time.Millisecond*100)

			t.Run("SetBatch", func(t *testing.T) {
				err := cache.SetBatch(testData)
				assert.NoError(t, err)
				assert.Equal(t, len(testData), cache.Len())

				// Проверяем что все значения установлены корректно
				for k, v := range testData {
					val, exists := cache.Get(k)
					assert.True(t, exists)
					assert.Equal(t, v, val)
				}
			})

			t.Run("GetBatch", func(t *testing.T) {
				results, err := cache.GetBatch(testKeys)
				assert.NoError(t, err)
				assert.Len(t, results, len(testKeys))

				// Проверяем результаты
				for _, result := range results {
					if expected, ok := testData[result.Key]; ok {
						assert.True(t, result.Exists)
						assert.Equal(t, expected, result.Value)
					} else {
						assert.False(t, result.Exists)
					}
				}
			})

			t.Run("GetBatchWithMetrics", func(t *testing.T) {
				results, err := cache.GetBatchWithMetrics(testKeys)
				assert.NoError(t, err)
				assert.Len(t, results, len(testKeys))

				for _, result := range results {
					if expected, ok := testData[result.Key]; ok {
						assert.True(t, result.Exists)
						assert.Equal(t, expected, result.Value)
						assert.NotZero(t, result.TimeCreated)
						assert.NotZero(t, result.GetCount) // т.к. мы уже делали Get в предыдущем тесте
						assert.Equal(t, uint32(1), result.SetCount)
					} else {
						assert.False(t, result.Exists)
						assert.Zero(t, result.TimeCreated)
						assert.Zero(t, result.GetCount)
						assert.Zero(t, result.SetCount)
					}
				}
			})

			t.Run("DeleteBatch", func(t *testing.T) {
				// Удаляем часть ключей
				keysToDelete := []string{"key1", "key2"}
				cache.DeleteBatch(keysToDelete)

				// Проверяем что ключи удалены
				for _, key := range keysToDelete {
					_, exists := cache.Get(key)
					assert.False(t, exists)
				}

				// Проверяем что остальные ключи на месте
				remainingKey := "key3"
				val, exists := cache.Get(remainingKey)
				assert.True(t, exists)
				assert.Equal(t, testData[remainingKey], val)
			})
		})
	}
}

func TestBatchOperationsWithClosedCache(t *testing.T) {
	type TestStruct struct {
		ID    int
		Value string
	}

	tests := []struct {
		name      string
		cacheFunc func(context.Context, time.Duration, time.Duration) types.ICacheInMemory[TestStruct]
	}{
		{
			name:      "ConcurrentCache",
			cacheFunc: NewConcurrentCache[TestStruct],
		},
		{
			name:      "ShardedCache",
			cacheFunc: NewShardedCache[TestStruct],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cache := tt.cacheFunc(ctx, time.Second, time.Millisecond*100)
			cache.Clear() // Закрываем кэш

			t.Run("SetBatch", func(t *testing.T) {
				err := cache.SetBatch(map[string]TestStruct{
					"key1": {ID: 1, Value: "value1"},
				})
				assert.Error(t, err)
			})

			t.Run("GetBatch", func(t *testing.T) {
				results, err := cache.GetBatch([]string{"key1"})
				assert.Error(t, err)
				assert.Nil(t, results)
			})

			t.Run("GetBatchWithMetrics", func(t *testing.T) {
				results, err := cache.GetBatchWithMetrics([]string{"key1"})
				assert.Error(t, err)
				assert.Nil(t, results)
			})

			t.Run("DeleteBatch", func(t *testing.T) {
				// Не должно вызывать панику
				cache.DeleteBatch([]string{"key1"})
			})
		})
	}
}
