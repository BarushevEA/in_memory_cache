package pkg

import (
	"context"
	"github.com/BarushevEA/in_memory_cache/internal/src"
	"github.com/BarushevEA/in_memory_cache/types"
	"time"
)

// NewConcurrentCache creates a concurrent cache with TTL support and automatic TTL decrement in the background.
func NewConcurrentCache[T any](ctx context.Context, ttl, ttlDecrement time.Duration) types.ICacheInMemory[T] {
	return src.NewConcurrentMapWithTTL[T](ctx, ttl, ttlDecrement)
}

// NewShardedCache initializes a sharded in-memory cache with specified TTL and TTL decrement for cleanup operations.
// ctx controls the lifetime of the cache and its background tasks.
// ttl defines the time-to-live duration for cache entries before expiration.
// ttlDecrement specifies the frequency of cleanup checks for expired entries.
func NewShardedCache[T any](ctx context.Context, ttl, ttlDecrement time.Duration) types.ICacheInMemory[T] {
	return src.NewDynamicShardedMapWithTTL[T](ctx, ttl, ttlDecrement)
}
