package src

import (
	"context"
	"errors"
	"github.com/BarushevEA/in_memory_cache/types"
	"runtime"
	"sync"
	"time"
)

// ConcurrentMapWithTTL provides a thread-safe map with support for time-to-live (TTL) for its entries.
type ConcurrentMapWithTTL[T any] struct {
	sync.RWMutex
	data         map[string]*MapNode[T]
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
	deleteCount           int
}

// rangeEntry represents a key-value entry with a key of type string and a node implementing the IMapNode interface.
type rangeEntry[T any] struct {
	key  string
	node *MapNode[T]
}

// NewConcurrentMapWithTTL creates a new concurrent map with TTL support and starts a background TTL management goroutine.
func NewConcurrentMapWithTTL[T any](ctx context.Context, ttl, ttlDecrement time.Duration) types.ICacheInMemory[T] {
	cMap := &ConcurrentMapWithTTL[T]{}
	cMap.data = make(map[string]*MapNode[T])
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

	entries := make([]*rangeEntry[T], 0, len(cMap.data))
	cMap.RLock()
	for key, node := range cMap.data {
		entries = append(entries, &rangeEntry[T]{key: key, node: node})
	}
	cMap.RUnlock()

	for _, entry := range entries {
		if entry.node.remove == nil {
			continue
		}

		if !callback(entry.key, entry.node.GetData()) {
			return nil
		}
	}

	entries = nil

	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) RangeWithMetrics(callback func(key string, value T, createdAt time.Time, setCount uint32, getCount uint32) bool) error {
	if cMap.isClosed {
		return errors.New("ConcurrentMapWithTTL.Range ERROR: cannot perform operation on closed cache")
	}

	entries := make([]*rangeEntry[T], 0, len(cMap.data))
	cMap.RLock()
	for key, node := range cMap.data {
		entries = append(entries, &rangeEntry[T]{key: key, node: node})
	}
	cMap.RUnlock()

	for _, entry := range entries {
		if entry.node.remove == nil {
			continue
		}

		value, createdAt, setCount, getCount := entry.node.GetDataWithMetrics()
		if !callback(entry.key, value, createdAt, setCount, getCount) {
			return nil
		}
	}

	entries = nil

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

// SetBatch adds multiple key-value pairs to the map by invoking the Set method for each entry in the provided batch map.
func (cMap *ConcurrentMapWithTTL[T]) SetBatch(batch map[string]T) error {
	if cMap.isClosed {
		return errors.New("ConcurrentMapWithTTL.Set ERROR: cannot perform operation on closed cache")
	}

	for key, value := range batch {
		err := cMap.Set(key, value)
		if err != nil {
			return err
		}
	}

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

// GetNodeValueWithMetrics retrieves the value, creation time, set count, and get count for a key, along with its existence status.
func (cMap *ConcurrentMapWithTTL[T]) GetNodeValueWithMetrics(key string) (T, time.Time, uint32, uint32, bool) {
	var (
		timeCreated time.Time
		setCount    uint32
		getCount    uint32
		value       T
	)

	if cMap.isClosed {
		return value, timeCreated, setCount, getCount, false
	}

	cMap.RLock()
	node, exists := cMap.data[key]
	cMap.RUnlock()

	if !exists {
		return value, timeCreated, setCount, getCount, false
	}

	value, timeCreated, setCount, getCount = node.GetDataWithMetrics()

	return value, timeCreated, setCount, getCount, true
}

// GetBatch retrieves a batch of values corresponding to the provided keys from the ConcurrentMapWithTTL.
// It returns a slice of BatchNode containing the values, existence flags, or an error if the map is closed.
func (cMap *ConcurrentMapWithTTL[T]) GetBatch(keys []string) ([]*types.BatchNode[T], error) {
	if cMap.isClosed {
		return nil, errors.New("ConcurrentMapWithTTL.Get ERROR: cannot perform operation on closed cache")
	}

	batch := make([]*types.BatchNode[T], len(keys))
	cMap.RLock()
	for i, key := range keys {
		batch[i] = &types.BatchNode[T]{Key: key}
		if mapNode, ok := cMap.data[key]; ok {
			batch[i].Value = mapNode.GetData()
			batch[i].Exists = true
		}
	}
	cMap.RUnlock()

	return batch, nil
}

// GetBatchWithMetrics retrieves detailed metrics for a batch of keys, returning a slice of Metric objects or an error.
func (cMap *ConcurrentMapWithTTL[T]) GetBatchWithMetrics(keys []string) ([]*types.Metric[T], error) {
	if cMap.isClosed {
		return nil, errors.New("ConcurrentMapWithTTL.Get ERROR: cannot perform operation on closed cache")
	}

	result := make([]*types.Metric[T], 0, len(keys))
	for _, key := range keys {
		metric := &types.Metric[T]{}
		metric.Key = key

		cMap.RLock()
		node, exists := cMap.data[key]
		cMap.RUnlock()

		if !exists {
			result = append(result, metric)
			continue
		}

		metric.Value,
			metric.TimeCreated,
			metric.SetCount,
			metric.GetCount = node.GetDataWithMetrics()
		metric.Exists = true

		result = append(result, metric)
	}

	return result, nil
}

// Delete removes a key and its associated data from the map, clearing the node before deletion if it exists.
func (cMap *ConcurrentMapWithTTL[T]) Delete(key string) {
	if cMap.isClosed {
		return
	}

	cMap.Lock()
	node, ok := cMap.data[key]
	if ok {
		delete(cMap.data, key)
	}
	cMap.Unlock()

	if ok {
		node.Clear()
		cMap.deleteCount++
		if cMap.deleteCount > 1000 {
			runtime.GC()
			cMap.deleteCount = 0
		}
	}
}

// DeleteBatch removes multiple keys and their associated data from the map. Clears each node before deletion if it exists.
func (cMap *ConcurrentMapWithTTL[T]) DeleteBatch(keys []string) {
	if cMap.isClosed {
		return
	}

	deletionsCount := 0

	for _, key := range keys {
		cMap.Lock()
		node, ok := cMap.data[key]
		if ok {
			delete(cMap.data, key)
			deletionsCount++
		}
		cMap.Unlock()

		if ok {
			node.Clear()
		}
	}

	if deletionsCount > 0 && deletionsCount%10 == 0 {
		runtime.GC()
	}
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
	cMap.data = make(map[string]*MapNode[T])

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
