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
}

// ICacheInMemory defines an in-memory cache interface supporting generic types and basic operations with TTL support.
// Set stores a key-value pair in the cache or updates an existing key with the provided value.
// Get retrieves the value associated with a key and indicates if the key exists in the cache.
// Delete removes a key-value pair from the cache if the key exists.
// Clear removes all key-value pairs from the cache, resetting its state.
// Len returns the total number of key-value pairs currently stored in the cache.
// Range iterates over all key-value pairs, with the provided function executed for each one.
// SetTTL assigns a time-to-live duration for entries in the cache, after which they expire.
type ICacheInMemory[T any] interface {
	Set(key string, value T) error
	Get(key string) (T, bool)
	Delete(key string)
	Clear()

	Len() int
	Range(func(key string, value T) bool) error
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
}
