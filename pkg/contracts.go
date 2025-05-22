package pkg

import "time"

// ICache defines a generic caching interface supporting basic operations for managing key-value pairs.
// It provides methods for adding, retrieving, deleting, and clearing cache entries.
// The interface supports iteration over stored entries using a callback function.
// ICache ensures flexibility with generic types to store any data type while maintaining type safety.
// It also provides utility methods to check the cache size via Len.
type ICache[T any] interface {
	Set(key string, value T) error
	Get(key string) (T, bool)
	Delete(key string)
	Clear()

	Len() int
	Range(func(key string, value T) bool) error
	RangeWithMetrics(callback func(key string, value T, createdAt time.Time, setCount uint32, getCount uint32) bool) error
	GetNodeValueWithMetrics(key string) (T, time.Time, uint32, uint32, bool)
}
