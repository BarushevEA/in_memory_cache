package src

import "time"

// IMapNode defines an interface for managing nodes with generic data, time-to-live, and lifecycle management functionality.
// SetTTL sets the time-to-live duration for the node.
// SetTTLDecrement defines the decrement duration for the TTL on each tick operation.
// SetRemoveCallback sets a callback function triggered when the node expires or is removed.
// Tick decreases the TTL by the specified decrement and triggers removal if the TTL reaches zero.
// GetData retrieves the stored data from the node, resetting its duration to the original TTL.
// SetData assigns the specified data to the node.
// Clear resets the node's state, removing data, TTL values, and any associated callbacks.
type IMapNode[T any] interface {
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
	SetRemoveCallback(remove func())
	Tick()
	GetData() T
	SetData(data T)
	Clear()
	GetMetrics() (time.Time, uint32, uint32)
}

// ICacheInMemory defines a generic interface for in-memory caching functionality with CRUD operations and iteration support.
// Set stores or updates a key-value pair in the cache and returns an error if the operation fails.
// Get retrieves a value associated with the given key and a boolean indicating if the key exists in the cache.
// Delete removes the entry corresponding to the provided key from the cache.
// Clear removes all entries from the cache.
// Len returns the current number of entries in the cache.
// Range iterates over key-value pairs, applying the provided callback function, and halts if the callback returns false.
type ICacheInMemory[T any] interface {
	Set(key string, value T) error
	Get(key string) (T, bool)
	Delete(key string)
	Clear()

	Len() int
	Range(func(key string, value T) bool) error
	GetNodeMetrics(key string) (time.Time, uint32, uint32, bool)
}
