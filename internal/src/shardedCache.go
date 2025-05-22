package src

import (
	"context"
	"errors"
	"github.com/BarushevEA/in_memory_cache/internal/utils"
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
	shards    [256]*ICacheInMemory[T]
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
func NewDynamicShardedMapWithTTL[T any](ctx context.Context, ttl, decrement time.Duration) ICacheInMemory[T] {
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
func (m *DynamicShardedMapWithTTL[T]) getShard(hash uint8) ICacheInMemory[T] {
	shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[hash]))))
	if shard != nil {
		return *shard
	}

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	shard = (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[hash]))))
	if shard != nil {
		return *shard
	}

	newShard := NewConcurrentMapWithTTL[T](m.ctx, m.ttl, m.decrement)
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&m.shards[hash])),
		unsafe.Pointer(&newShard),
	)
	return newShard
}

// Set inserts or updates a key-value pair in the dynamic sharded map. Returns an error if the map is closed.
func (m *DynamicShardedMapWithTTL[T]) Set(key string, value T) error {
	if m.isClosed.Load() {
		return errors.New("cache is closed")
	}

	hash := utils.GetTopHash(key)
	shard := m.getShard(hash)
	return shard.Set(key, value)
}

// Get retrieves the value associated with the given key from the dynamic sharded map and its existence status.
func (m *DynamicShardedMapWithTTL[T]) Get(key string) (T, bool) {
	if m.isClosed.Load() {
		return *new(T), false
	}

	hash := utils.GetTopHash(key)
	shard := m.getShard(hash)
	return shard.Get(key)
}

// Delete removes an entry with the specified key from the map if it exists and the map is not closed.
func (m *DynamicShardedMapWithTTL[T]) Delete(key string) {
	if m.isClosed.Load() {
		return
	}

	hash := utils.GetTopHash(key)
	if shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[hash])))); shard != nil {
		(*shard).Delete(key)
	}
}

// Clear removes all data from the map, clears underlying shards, and cancels the internal context, marking the map as closed.
func (m *DynamicShardedMapWithTTL[T]) Clear() {
	if !m.isClosed.CompareAndSwap(false, true) {
		return
	}

	for i := range m.shards {
		if shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[i])))); shard != nil {
			(*shard).Clear()
			atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[i])), nil)
		}
	}
	m.cancel()
}

// Len returns the total number of entries across all shards in the map or 0 if the map is closed.
func (m *DynamicShardedMapWithTTL[T]) Len() int {
	if m.isClosed.Load() {
		return 0
	}

	total := 0
	for i := range m.shards {
		if shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[i])))); shard != nil {
			total += (*shard).Len()
		}
	}
	return total
}

// Range iterates over all key-value pairs in the map, applying the provided callback function.
// Returns an error if the map is closed or if an error occurs during shard iteration.
func (m *DynamicShardedMapWithTTL[T]) Range(callback func(key string, value T) bool) error {
	if m.isClosed.Load() {
		return errors.New("cache is closed")
	}

	for i := range m.shards {
		if shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[i])))); shard != nil {
			if err := (*shard).Range(callback); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *DynamicShardedMapWithTTL[T]) GetNodeMetrics(key string) (time.Time, uint32, uint32, bool) {
	var (
		timeCreated time.Time
		setCount    uint32
		getCount    uint32
		exists      bool
	)

	if m.isClosed.Load() {
		return timeCreated, setCount, getCount, exists
	}

	hash := utils.GetTopHash(key)
	shard := m.getShard(hash)

	timeCreated, setCount, getCount, exists = shard.GetNodeMetrics(key)
	return timeCreated, setCount, getCount, exists
}
