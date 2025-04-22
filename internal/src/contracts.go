package src

import "time"

// IMapNode represents a generic node interface used for managing elements with a time-to-live (TTL) mechanism.
// SetTTL sets the time-to-live for the node, specifying its lifespan duration.
// SetTTLDecrement defines the decrement interval for the TTL, influencing the ticking behavior.
// SetRemoveCallback assigns a function to be invoked when the node's TTL expires.
// Tick decreases the node's TTL by the defined decrement interval, triggering removal when TTL is zero or negative.
// GetData retrieves the data associated with the node, resetting its TTL to the original value.
type IMapNode[T any] interface {
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
	SetRemoveCallback(remove func())
	Tick()
	GetData() T
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
}
