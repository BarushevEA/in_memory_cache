package src

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ConcurrentMapWithTTL provides a thread-safe map with support for time-to-live (TTL) for its entries.
type ConcurrentMapWithTTL[T any] struct {
	sync.RWMutex
	data         map[string]IMapNode[T]
	ctx          context.Context
	cancel       context.CancelFunc
	ttl          time.Duration
	ttlDecrement time.Duration
	isClosed     bool
	tickerOnce   sync.Once

	keysForDelete         []string
	keysForDeleteSync     sync.Mutex
	maxKeysForDeleteUsage int
	maxKeysForDeleteCount int
}

// NewConcurrentMapWithTTL creates a new concurrent map with TTL support and starts a background TTL management goroutine.
func NewConcurrentMapWithTTL[T any](ctx context.Context, ttl, ttlDecrement time.Duration) ICacheInMemory[T] {
	cMap := &ConcurrentMapWithTTL[T]{}
	cMap.data = make(map[string]IMapNode[T])
	cMap.maxKeysForDeleteUsage = 10000
	cMap.keysForDelete = make([]string, 100, cMap.maxKeysForDeleteUsage)
	cMap.ctx, cMap.cancel = context.WithCancel(ctx)
	cMap.ttl = ttl
	cMap.ttlDecrement = ttlDecrement

	if ttl <= 0 || ttlDecrement <= 0 || ttlDecrement > ttl {
		cMap.ttl = 5 * time.Second
		cMap.ttlDecrement = 1 * time.Second
	}

	return cMap
}

// Range iterates over all key-value pairs in the map, executing the provided callback function for each pair.
// The iteration stops if the callback function returns false.
// Returns an error if the operation cannot be performed.
func (cMap *ConcurrentMapWithTTL[T]) Range(callback func(key string, value T) bool) error {
	if cMap.isClosed {
		return errors.New("ConcurrentMapWithTTL.Range ERROR: cannot perform operation on closed cache")
	}

	cMap.RLock()
	for key, node := range cMap.data {
		if !callback(key, node.GetData()) {
			cMap.RUnlock()
			return nil
		}
	}
	cMap.RUnlock()

	return nil
}

// Set adds or updates a key-value pair in the map, initializing a new node if the key does not exist.
func (cMap *ConcurrentMapWithTTL[T]) Set(key string, value T) error {
	if cMap.isClosed {
		return errors.New("ConcurrentMapWithTTL.Set ERROR: cannot perform operation on closed cache")
	}

	cMap.Lock()
	if node, ok := cMap.data[key]; ok {
		node.SetData(value)
		cMap.Unlock()
		return nil
	}

	newNode := NewMapNode[T](value)
	newNode.SetTTL(cMap.ttl)
	newNode.SetTTLDecrement(cMap.ttlDecrement)
	newNode.SetRemoveCallback(func() {
		cMap.keysForDeleteSync.Lock()
		cMap.keysForDelete = append(cMap.keysForDelete, key)
		cMap.maxKeysForDeleteCount++
		cMap.keysForDeleteSync.Unlock()
	})

	cMap.data[key] = newNode
	cMap.tickerOnce.Do(func() {
		go cMap.tickCollection()
	})
	cMap.Unlock()

	return nil
}

// Get retrieves the value associated with the given key and a boolean indicating if the key exists in the map.
func (cMap *ConcurrentMapWithTTL[T]) Get(key string) (T, bool) {
	if cMap.isClosed {
		return *new(T), false
	}

	cMap.RLock()
	node, ok := cMap.data[key]
	cMap.RUnlock()

	if ok {
		return node.GetData(), true
	}

	return *new(T), false
}

// Delete removes a key and its associated data from the map, clearing the node before deletion if it exists.
func (cMap *ConcurrentMapWithTTL[T]) Delete(key string) {
	if cMap.isClosed {
		return
	}

	cMap.Lock()
	if node, ok := cMap.data[key]; ok {
		node.Clear()
		delete(cMap.data, key)
	}
	cMap.Unlock()
}

// Clear removes all elements from the map and clears their associated nodes.
func (cMap *ConcurrentMapWithTTL[T]) Clear() {
	if cMap.isClosed {
		return
	}

	cMap.isClosed = true

	cMap.Lock()
	for key, node := range cMap.data {
		node.Clear()
		delete(cMap.data, key)
	}
	cMap.data = make(map[string]IMapNode[T])

	cMap.cancel()
	cMap.Unlock()
}

// Len returns the number of elements in the map. It is safe for concurrent access.
func (cMap *ConcurrentMapWithTTL[T]) Len() int {
	if cMap.isClosed {
		return 0
	}

	cMap.RLock()
	defer cMap.RUnlock()
	return len(cMap.data)
}

// tickCollection periodically decrements the TTL of each node and removes expired nodes until the context is canceled.
func (cMap *ConcurrentMapWithTTL[T]) tickCollection() {
	if cMap.isClosed {
		return
	}

	ticker := time.NewTicker(cMap.ttlDecrement)
	defer ticker.Stop()

	isProcessed := false

	for {
		select {
		case <-cMap.ctx.Done():
			cMap.Clear()
			return
		case <-ticker.C:
			if cMap.isClosed {
				return
			}
			if isProcessed {
				continue
			}

			isProcessed = true

			nodes := make([]IMapNode[T], 0, len(cMap.data))

			cMap.RLock()
			for _, node := range cMap.data {
				nodes = append(nodes, node)
			}
			cMap.RUnlock()

			for i := 0; i < len(nodes); i++ {
				nodes[i].Tick()
			}

			if len(cMap.keysForDelete) > 0 {
				go func() {
					cMap.keysForDeleteSync.Lock()
					for _, key := range cMap.keysForDelete {
						cMap.Delete(key)
					}
					cMap.keysForDelete = cMap.keysForDelete[:0]
					if cMap.maxKeysForDeleteCount > cMap.maxKeysForDeleteUsage {
						cMap.maxKeysForDeleteCount = 0
						cMap.keysForDelete = make([]string, 100, 10000)
					}
					cMap.keysForDeleteSync.Unlock()
				}()
			}

			nodes = nil
			isProcessed = false
		}
	}
}

func (cMap *ConcurrentMapWithTTL[T]) GetNodeMetrics(key string) (time.Time, uint32, uint32, bool) {
	var (
		timeCreated time.Time
		setCount    uint32
		getCount    uint32
	)

	if cMap.isClosed {
		return timeCreated, setCount, getCount, false
	}

	cMap.RLock()
	node, exists := cMap.data[key]
	cMap.RUnlock()

	if !exists {
		return timeCreated, setCount, getCount, false
	}

	timeCreated, setCount, getCount = node.GetMetrics()

	return timeCreated, setCount, getCount, true
}
