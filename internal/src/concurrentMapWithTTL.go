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
		node.SetTTL(cMap.ttl)
		node.SetTTLDecrement(cMap.ttlDecrement)
	} else {
		//dCache.Save(key, status)
	}

	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) Get(key string) (T, bool) {
	//TODO implement me
	panic("implement me")
}

func (cMap *ConcurrentMapWithTTL[T]) Delete(key string) {
	//TODO implement me
	panic("implement me")
}

func (cMap *ConcurrentMapWithTTL[T]) Clear() {
	//TODO implement me
	panic("implement me")
}

func (cMap *ConcurrentMapWithTTL[T]) Len() int {
	//TODO implement me
	panic("implement me")
}

func (cMap *ConcurrentMapWithTTL[T]) SetTTL(ttl time.Duration) {
	//TODO implement me
	panic("implement me")
}

func (cMap *ConcurrentMapWithTTL[T]) SetTTLDecrement(ttlDecrement time.Duration) {
	//TODO implement me
	panic("implement me")
}
