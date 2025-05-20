package pkg

import (
	"context"
	"github.com/BarushevEA/in_memory_cache/internal/src"
	"time"
)

func NewConcurrentCache[T any](ctx context.Context, ttl, ttlDecrement time.Duration) ICache[T] {
	return src.NewConcurrentMapWithTTL[T](ctx, ttl, ttlDecrement)
}

func NewShardedCache[T any](ctx context.Context, ttl, ttlDecrement time.Duration) ICache[T] {
	return src.NewDynamicShardedMapWithTTL[T](ctx, ttl, ttlDecrement)
}
