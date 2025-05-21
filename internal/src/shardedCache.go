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

type DynamicShardedMapWithTTL[T any] struct {
	shards    [256]*ICacheInMemory[T]
	ctx       context.Context
	cancel    context.CancelFunc
	ttl       time.Duration
	decrement time.Duration
	isClosed  atomic.Bool
	initOnce  sync.Once
}

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

func (m *DynamicShardedMapWithTTL[T]) Set(key string, value T) error {
	if m.isClosed.Load() {
		return errors.New("cache is closed")
	}

	hash := utils.GetTopHash(key)
	shard := m.getShard(hash)
	return shard.Set(key, value)
}

func (m *DynamicShardedMapWithTTL[T]) Get(key string) (T, bool) {
	if m.isClosed.Load() {
		return *new(T), false
	}

	hash := utils.GetTopHash(key)
	shard := m.getShard(hash)
	return shard.Get(key)
}

func (m *DynamicShardedMapWithTTL[T]) Delete(key string) {
	if m.isClosed.Load() {
		return
	}

	hash := utils.GetTopHash(key)
	if shard := (*ICacheInMemory[T])(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.shards[hash])))); shard != nil {
		(*shard).Delete(key)
	}
}

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
