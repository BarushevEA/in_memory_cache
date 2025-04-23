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
	if shardedMap.isClosed {
		return *new(T), false
	}

	shardedMap.RLock()
	defer shardedMap.RUnlock()
	hash := utils.GetTopHash(key)
	if shard, ok := shardedMap.shards[hash]; ok {
		return shard.Get(key)
	}

	return *new(T), false
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Delete(key string) {
	if shardedMap.isClosed {
		return
	}

	shardedMap.Lock()
	defer shardedMap.Unlock()
	hash := utils.GetTopHash(key)
	if shard, ok := shardedMap.shards[hash]; ok {
		shard.Delete(key)
		if shard.Len() == 0 {
			delete(shardedMap.shards, hash)
		}
	}
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Clear() {
	if shardedMap.isClosed {
		return
	}

	shardedMap.isClosed = true

	shardedMap.Lock()
	defer shardedMap.Unlock()
	for key, shard := range shardedMap.shards {
		shard.Clear()
		delete(shardedMap.shards, key)
	}
	shardedMap.shards = make(map[uint8]ICacheInMemory[T])
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Len() int {
	if shardedMap.isClosed {
		return 0
	}

	shardedMap.RLock()
	defer shardedMap.RUnlock()
	var count int
	for _, shard := range shardedMap.shards {
		count += shard.Len()
	}
	return count
}

func (shardedMap *DynamicShardedMapWithTTL[T]) Range(callback func(key string, value T) bool) error {
	if shardedMap.isClosed {
		return errors.New("DynamicShardedMapWithTTL.Range ERROR: cannot perform operation on closed cache")
	}

	shardedMap.RLock()
	defer shardedMap.RUnlock()
	for _, shard := range shardedMap.shards {
		err := shard.Range(callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func (shardedMap *DynamicShardedMapWithTTL[T]) tickCollection() {
	if shardedMap.isClosed {
		return
	}

	ticker := time.NewTicker(shardedMap.ttlDecrement)
	defer ticker.Stop()

	for {
		select {
		case <-shardedMap.ctx.Done():
			shardedMap.Clear()
		}

		select {
		case <-ticker.C:
			if shardedMap.isClosed {
				return
			}

			shardedMap.Lock()
			for key, shard := range shardedMap.shards {
				if shard.Len() == 0 {
					shard.Clear()
					delete(shardedMap.shards, key)
				}
			}
			shardedMap.Unlock()
		}
	}
}
