package src

import (
	"context"
	"sync"
	"time"
)

type ConcurrentMapWithTTL[T any] struct {
	sync.RWMutex
	data         map[string]IMapNode[T]
	ctx          context.Context
	cancel       context.CancelFunc
	ttl          time.Duration
	ttlDecrement time.Duration
}

func NewConcurrentMapWithTTL[T any](ctx context.Context) ICacheInMemory[T] {
	cMap := &ConcurrentMapWithTTL[T]{}
	cMap.data = make(map[string]IMapNode[T])
	cMap.ctx, cMap.cancel = context.WithCancel(ctx)

	return cMap
}

func (cMap *ConcurrentMapWithTTL[T]) Range(callback func(key string, value T) bool) error {
	cMap.RLock()
	defer cMap.RUnlock()
	for key, node := range cMap.data {
		if !callback(key, node.GetData()) {
			return nil
		}
	}
	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) Set(key string, value T) error {
	cMap.Lock()
	defer cMap.Unlock()

	if node, ok := cMap.data[key]; ok {
		node.SetData(value)
	} else {
		newNode := NewMapNode[T](value)
		newNode.SetTTL(cMap.ttl)
		newNode.SetTTLDecrement(cMap.ttlDecrement)
		newNode.SetRemoveCallback(func() {
			cMap.Delete(key)
		})
		cMap.data[key] = newNode
	}

	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) Delete(key string) {
	cMap.Lock()
	defer cMap.Unlock()
	if node, ok := cMap.data[key]; ok {
		node.Clear()
		delete(cMap.data, key)
	}
}

func (cMap *ConcurrentMapWithTTL[T]) Get(key string) (T, bool) {
	cMap.RLock()
	defer cMap.RUnlock()

	if node, ok := cMap.data[key]; ok {
		return node.GetData(), true
	}

	return *new(T), false
}

func (cMap *ConcurrentMapWithTTL[T]) Clear() {
	cMap.Lock()
	defer cMap.Unlock()
	for key, node := range cMap.data {
		node.Clear()
		delete(cMap.data, key)
	}
}

func (cMap *ConcurrentMapWithTTL[T]) Len() int {
	cMap.RLock()
	defer cMap.RUnlock()
	return len(cMap.data)
}

func (cMap *ConcurrentMapWithTTL[T]) SetTTL(ttl time.Duration) {
	cMap.Lock()
	defer cMap.Unlock()
	cMap.ttl = ttl
	for _, node := range cMap.data {
		node.SetTTL(ttl)
	}
}

func (cMap *ConcurrentMapWithTTL[T]) SetTTLDecrement(ttlDecrement time.Duration) {
	cMap.Lock()
	defer cMap.Unlock()
	cMap.ttlDecrement = ttlDecrement
	for _, node := range cMap.data {
		node.SetTTLDecrement(ttlDecrement)
	}
}

func (cMap *ConcurrentMapWithTTL[T]) Tick() {
	cMap.Lock()
	defer cMap.Unlock()
	for _, node := range cMap.data {
		node.Tick()
	}
}
