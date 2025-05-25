package src

import (
	"time"
)

// IMapNode represents an interface for managing nodes with TTL, data handling, lifecycle control, and metrics tracking.
type IMapNode[T any] interface {
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
	SetRemoveCallback(remove func())
	Tick()
	GetData() T
	SetData(data T)
	Clear()
	GetDataWithMetrics() (T, time.Time, uint32, uint32)
}

// ICacheInMemory defines a generic in-memory cache interface with type-safe operations and optional metadata support.
// It provides methods for managing individual and batch key-value pairs, along with cache metadata and metrics.
// The interface ensures thread-safe operations for scenarios requiring concurrency and real-time updates.
type ICacheInMemory[T any] interface {
	Set(key string, value T) error
	SetBatch(batch map[string]T) error

	Get(key string) (T, bool)
	GetNodeValueWithMetrics(key string) (T, time.Time, uint32, uint32, bool)
	GetBatch(keys []string) ([]*BatchNode[T], error)
	GetBatchWithMetrics(keys []string) ([]*Metric[T], error)

	Delete(key string)
	DeleteBatch(keys []string)

	Clear()
	Len() int

	Range(func(key string, value T) bool) error
	RangeWithMetrics(callback func(key string, value T, createdAt time.Time, setCount uint32, getCount uint32) bool) error
}
