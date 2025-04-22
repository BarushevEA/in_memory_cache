package src

import "time"

// IMapNode defines an interface for a map node supporting generic types, TTL management, and removal callbacks.
// SetTTL sets the time-to-live (TTL) duration for the node.
// SetTTLDecrement sets the decrement value for the node's TTL on each tick.
// SetRemoveCallback sets a callback function to be invoked when the node expires and needs removal.
// Tick decrements the TTL by the specified decrement value and triggers the removal callback if expired.
// GetData retrieves the data associated with the node, resetting its TTL duration.
// SetData sets the data for the node.
type IMapNode[T any] interface {
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
	SetRemoveCallback(remove func())
	Tick()
	GetData() T
	SetData(data T)
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
