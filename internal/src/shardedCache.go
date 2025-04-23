package src

import (
	"context"
	"sync"
	"time"
)

type DynamicShardedMapWithTTL[T any] struct {
	sync.RWMutex
	shards map[uint8]ICacheInMemory[T]
	ctx    context.Context
	cancel context.CancelFunc
	//ttl                       time.Duration
	ttlDecrement              time.Duration
	cacheInMemoryTtl          time.Duration
	cacheInMemoryTtlDecrement time.Duration
	isClosed                  bool
	tickerOnce                sync.Once
}

func NewDynamicShardedMapWithTTL[T any](ctx context.Context, cacheInMemoryTtl time.Duration, cacheInMemoryTtlDecrement time.Duration) ICacheInMemory[T] {
	shardedMap := &DynamicShardedMapWithTTL[T]{}
	shardedMap.shards = make(map[uint8]ICacheInMemory[T])
	shardedMap.ctx, shardedMap.cancel = context.WithCancel(ctx)
	shardedMap.cacheInMemoryTtl = cacheInMemoryTtl
	shardedMap.cacheInMemoryTtlDecrement = cacheInMemoryTtlDecrement
	shardedMap.ttlDecrement = 5 * time.Second

	if cacheInMemoryTtl <= 0 || cacheInMemoryTtlDecrement <= 0 || cacheInMemoryTtlDecrement > cacheInMemoryTtl {
		shardedMap.cacheInMemoryTtl = 5 * time.Second
		shardedMap.cacheInMemoryTtlDecrement = 1 * time.Second
	}

	return shardedMap
}

func (d DynamicShardedMapWithTTL[T]) Set(key string, value T) error {
	//TODO implement me
	panic("implement me")
}

func (d DynamicShardedMapWithTTL[T]) Get(key string) (T, bool) {
	//TODO implement me
	panic("implement me")
}

func (d DynamicShardedMapWithTTL[T]) Delete(key string) {
	//TODO implement me
	panic("implement me")
}

func (d DynamicShardedMapWithTTL[T]) Clear() {
	//TODO implement me
	panic("implement me")
}

func (d DynamicShardedMapWithTTL[T]) Len() int {
	//TODO implement me
	panic("implement me")
}

func (d DynamicShardedMapWithTTL[T]) Range(f func(key string, value T) bool) error {
	//TODO implement me
	panic("implement me")
}
