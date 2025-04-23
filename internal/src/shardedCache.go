package src

import (
	"context"
	"errors"
	"github.com/BarushevEA/in_memory_cache/internal/utils"
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

func (shardedMap *DynamicShardedMapWithTTL[T]) Set(key string, value T) error {
	if shardedMap.isClosed {
		return errors.New("DynamicShardedMapWithTTL.Set ERROR: cannot perform operation on closed cache")
	}

	defer func() {
		shardedMap.tickerOnce.Do(func() {
			go shardedMap.tickCollection()
		})
	}()

	shardedMap.Lock()
	defer shardedMap.Unlock()
	hash := utils.GetTopHash(key)
	if shard, ok := shardedMap.shards[hash]; ok {
		err := shard.Set(key, value)
		if err != nil {
			return err
		}
	} else {
		newShard := NewConcurrentMapWithTTL[T](
			shardedMap.ctx,
			shardedMap.cacheInMemoryTtl,
			shardedMap.cacheInMemoryTtlDecrement)
		err := newShard.Set(key, value)
		if err != nil {
			return err
		}
		shardedMap.shards[hash] = newShard
	}

	return nil
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Get(key string) (T, bool) {
	//TODO implement me
	panic("implement me")
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Delete(key string) {
	//TODO implement me
	panic("implement me")
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Clear() {
	//TODO implement me
	panic("implement me")
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Len() int {
	//TODO implement me
	panic("implement me")
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Range(f func(key string, value T) bool) error {
	//TODO implement me
	panic("implement me")
}

func (shardedMap *DynamicShardedMapWithTTL[T]) tickCollection() {

}
