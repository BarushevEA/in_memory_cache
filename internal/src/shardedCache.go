package src

import (
	"context"
	"errors"
	"github.com/BarushevEA/in_memory_cache/internal/utils"
	"github.com/BarushevEA/in_memory_cache/types"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// DynamicShardedMapWithTTL is a generic, sharded, in-memory cache with a configurable TTL for entries.
// Uses 256 shards for concurrency and stores data with automatic expiration based on the TTL settings.
// Allows operations like Set, Get, Delete, Clear, Len, and Range.
// Provides thread-safe access and automatic TTL decrement on entries to remove expired data.
// Each shard is lazily initialized for efficiency based on usage.
// Maintains internal synchronization to safely handle concurrent operations.
// Once closed, the map discards all shards and prevents further operations.
type DynamicShardedMapWithTTL[T any] struct {
	shards    [256]*types.ICacheInMemory[T]
	ctx       context.Context
	cancel    context.CancelFunc
	ttl       time.Duration
	decrement time.Duration
	isClosed  atomic.Bool
	initOnce  sync.Once
}

// NewDynamicShardedMapWithTTL initializes a sharded in-memory cache with TTL and decrement intervals for cleanup operations.
// ctx is the context that controls the lifetime of the cache and its internal operations.
// ttl specifies the time to live for cache entries before they expire.
// decrement specifies the frequency of cleanup checks for expired keys, must be less than or equal to ttl.
// Returns an implementation of ICacheInMemory[T].
func NewDynamicShardedMapWithTTL[T any](ctx context.Context, ttl, decrement time.Duration) types.ICacheInMemory[T] {
	if ttl <= 0 || decrement <= 0 || decrement > ttl {
		ttl = 5 * time.Second
		decrement = 1 * time.Second
	}

	ctx, cancel := context.WithCancel(ctx)
	m := &DynamicShardedMapWithTTL[T]{
		ctx:       ctx,
		cancel:    cancel,
		ttl:       ttl,
		decrement: decrement,
	}

	return m
}

// getShard returns the shard corresponding to the given hash, creating it if it doesn't already exist.
func (shardMap *DynamicShardedMapWithTTL[T]) getShard(hash uint8) types.ICacheInMemory[T] {
	shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[hash]))))
	if shard != nil {
		return *shard
	}

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	shard = (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[hash]))))
	if shard != nil {
		return *shard
	}

	newShard := NewConcurrentMapWithTTL[T](shardMap.ctx, shardMap.ttl, shardMap.decrement)
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[hash])),
		unsafe.Pointer(&newShard),
	)
	return newShard
}

// Set inserts or updates a key-value pair in the dynamic sharded map. Returns an error if the map is closed.
func (shardMap *DynamicShardedMapWithTTL[T]) Set(key string, value T) error {
	if shardMap.isClosed.Load() {
		return errors.New("cache is closed")
	}

	hash := utils.GetTopHash(key)
	shard := shardMap.getShard(hash)
	return shard.Set(key, value)
}

// SetBatch inserts or updates multiple key-value pairs in the dynamic sharded map and returns an error if the map is closed.
func (shardMap *DynamicShardedMapWithTTL[T]) SetBatch(batch map[string]T) error {
	if shardMap.isClosed.Load() {
		return errors.New("cache is closed")
	}

	for key, value := range batch {
		hash := utils.GetTopHash(key)
		shard := shardMap.getShard(hash)
		if err := shard.Set(key, value); err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves the value associated with the given key from the dynamic sharded map and its existence status.
func (shardMap *DynamicShardedMapWithTTL[T]) Get(key string) (T, bool) {
	if shardMap.isClosed.Load() {
		return *new(T), false
	}

	hash := utils.GetTopHash(key)
	shard := shardMap.getShard(hash)
	return shard.Get(key)
}

// GetNodeValueWithMetrics retrieves the value and associated metadata for a specific key if it exists in the map.
// Returns the value, the time it was created, the number of times it's been set, the number of times it's been retrieved, and a boolean indicating existence.
func (shardMap *DynamicShardedMapWithTTL[T]) GetNodeValueWithMetrics(key string) (T, time.Time, uint32, uint32, bool) {
	var (
		timeCreated time.Time
		setCount    uint32
		getCount    uint32
		exists      bool
		value       T
	)

	if shardMap.isClosed.Load() {
		return value, timeCreated, setCount, getCount, exists
	}

	hash := utils.GetTopHash(key)
	shard := shardMap.getShard(hash)

	value, timeCreated, setCount, getCount, exists = shard.GetNodeValueWithMetrics(key)
	return value, timeCreated, setCount, getCount, exists
}

// GetBatch retrieves a batch of key-value pairs from the map for the provided keys. Returns error if the map is closed.
func (shardMap *DynamicShardedMapWithTTL[T]) GetBatch(keys []string) ([]*types.BatchNode[T], error) {
	if shardMap.isClosed.Load() {
		return nil, errors.New("cache is closed")
	}

	var batch = make([]*types.BatchNode[T], 0, len(keys))
	for _, key := range keys {
		node := &types.BatchNode[T]{}
		node.Key = key
		node.Value, node.Exists = shardMap.Get(key)

		batch = append(batch, node)
	}

	return batch, nil
}

// GetBatchWithMetrics retrieves a batch of metrics for the given keys, including value, creation time, and access stats.
// Returns an error if the map is closed.
func (shardMap *DynamicShardedMapWithTTL[T]) GetBatchWithMetrics(keys []string) ([]*types.Metric[T], error) {
	if shardMap.isClosed.Load() {
		return nil, errors.New("cache is closed")
	}

	var batch = make([]*types.Metric[T], 0, len(keys))
	for _, key := range keys {
		node := &types.Metric[T]{}
		node.Key = key
		node.Value, node.TimeCreated, node.SetCount, node.GetCount, node.Exists = shardMap.GetNodeValueWithMetrics(key)

		batch = append(batch, node)
	}

	return batch, nil
}

// Delete removes an entry with the specified key from the map if it exists and the map is not closed.
func (shardMap *DynamicShardedMapWithTTL[T]) Delete(key string) {
	if shardMap.isClosed.Load() {
		return
	}

	hash := utils.GetTopHash(key)
	if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[hash])))); shard != nil {
		(*shard).Delete(key)
	}
}

// DeleteBatch removes multiple entries from the map for the provided keys if they exist and the map is not closed.
func (shardMap *DynamicShardedMapWithTTL[T]) DeleteBatch(keys []string) {
	if shardMap.isClosed.Load() {
		return
	}

	for _, key := range keys {
		hash := utils.GetTopHash(key)
		if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[hash])))); shard != nil {
			(*shard).Delete(key)
		}
	}
}

// Clear removes all data from the map, clears underlying shards, and cancels the internal context, marking the map as closed.
func (shardMap *DynamicShardedMapWithTTL[T]) Clear() {
	if !shardMap.isClosed.CompareAndSwap(false, true) {
		return
	}

	for i := range shardMap.shards {
		if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[i])))); shard != nil {
			(*shard).Clear()
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[i])), nil)
		}
	}
	shardMap.cancel()
}

// Len returns the total number of entries across all shards in the map or 0 if the map is closed.
func (shardMap *DynamicShardedMapWithTTL[T]) Len() int {
	if shardMap.isClosed.Load() {
		return 0
	}

	total := 0
	for i := range shardMap.shards {
		if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[i])))); shard != nil {
			total += (*shard).Len()
		}
	}
	return total
}

// Range iterates over all key-value pairs in the map, applying the provided callback function.
// Returns an error if the map is closed or if an error occurs during shard iteration.
func (shardMap *DynamicShardedMapWithTTL[T]) Range(callback func(key string, value T) bool) error {
	if shardMap.isClosed.Load() {
		return errors.New("cache is closed")
	}

	for i := range shardMap.shards {
		if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[i])))); shard != nil {
			if err := (*shard).Range(callback); err != nil {
				return err
			}
		}
	}
	return nil
}

func (shardMap *DynamicShardedMapWithTTL[T]) RangeWithMetrics(callback func(key string, value T, createdAt time.Time, setCount uint32, getCount uint32) bool) error {
	if shardMap.isClosed.Load() {
		return errors.New("cache is closed")
	}

	for i := range shardMap.shards {
		if shard := (*types.ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&shardMap.shards[i])))); shard != nil {
			if err := (*shard).RangeWithMetrics(callback); err != nil {
				return err
			}
		}
	}
	return nil
}
