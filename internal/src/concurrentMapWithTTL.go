package src

import (
	"context"
	"errors"
	"github.com/BarushevEA/in_memory_cache/types"
	"sync"
	"sync/atomic"
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
	isClosed     atomic.Bool
	tickerOnce   sync.Once

	nodeBuffer        []*MapNode[T]
	nodeBufferLock    sync.RWMutex
	maxNodeBufferSize int

	keysForDelete         map[string]struct{}
	keysForDeleteSync     sync.RWMutex
	maxKeysForDeleteUsage int
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
	cMap.keysForDelete = make(map[string]struct{}, cMap.maxKeysForDeleteUsage)
	cMap.ctx, cMap.cancel = context.WithCancel(ctx)
	cMap.ttl = ttl
	cMap.ttlDecrement = ttlDecrement
	cMap.isClosed.Store(false)
	cMap.maxNodeBufferSize = 10000
	cMap.nodeBuffer = make([]*MapNode[T], 0, cMap.maxNodeBufferSize)

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
	if cMap.isClosed.Load() {
		return errors.New("ConcurrentMapWithTTL.Range ERROR: cannot perform operation on closed cache")
	}

	entries := make([]*rangeEntry[T], 0, len(cMap.data))
	cMap.RLock()
	for key, node := range cMap.data {
		entries = append(entries, &rangeEntry[T]{key: key, node: node})
	}
	cMap.RUnlock()

	for _, entry := range entries {
		if !callback(entry.key, entry.node.GetData()) {
			return nil
		}
	}

	entries = nil

	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) RangeWithMetrics(callback func(key string, value T, createdAt time.Time, setCount uint32, getCount uint32) bool) error {
	if cMap.isClosed.Load() {
		return errors.New("ConcurrentMapWithTTL.Range ERROR: cannot perform operation on closed cache")
	}

	entries := make([]*rangeEntry[T], 0, len(cMap.data))
	cMap.RLock()
	for key, node := range cMap.data {
		entries = append(entries, &rangeEntry[T]{key: key, node: node})
	}
	cMap.RUnlock()

	for _, entry := range entries {
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
	if cMap.isClosed.Load() {
		return errors.New("ConcurrentMapWithTTL.Set ERROR: cannot perform operation on closed cache")
	}

	cMap.Lock()
	if node, ok := cMap.data[key]; ok && !node.IsDeleted() {
		node.SetData(value)
		cMap.Unlock()
		return nil
	} else if ok {
		node.reset(value)
		node.SetTTL(cMap.ttl)
		node.SetTTLDecrement(cMap.ttlDecrement)
		// No need for a remove callback as we now check for expired keys in tickCollection
		cMap.Unlock()
		return nil
	}

	newNode := cMap.getNode(value)
	newNode.SetTTL(cMap.ttl)
	newNode.SetTTLDecrement(cMap.ttlDecrement)
	// No need for a remove callback as we now check for expired keys in tickCollection

	cMap.data[key] = newNode
	cMap.tickerOnce.Do(func() {
		go cMap.tickCollection()
	})
	cMap.Unlock()

	return nil
}

func (cMap *ConcurrentMapWithTTL[T]) getNode(value T) *MapNode[T] {
	cMap.nodeBufferLock.RLock()
	if len(cMap.nodeBuffer) > 0 {
		node := cMap.nodeBuffer[0]
		cMap.nodeBuffer = cMap.nodeBuffer[1:]
		cMap.nodeBufferLock.RUnlock()
		return node
	}
	cMap.nodeBufferLock.RUnlock()
	return NewMapNode[T](value)
}

// markForDelete is now deprecated as we delete keys immediately
// This method is kept for backward compatibility with existing code
//func (cMap *ConcurrentMapWithTTL[T]) markForDelete(key string, node *MapNode[T]) {
//	// Instead of marking for deletion, delete immediately
//	cMap.Delete(key)
//}

// SetBatch adds multiple key-value pairs to the map by invoking the Set method for each entry in the provided batch map.
func (cMap *ConcurrentMapWithTTL[T]) SetBatch(batch map[string]T) error {
	if cMap.isClosed.Load() {
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
	if cMap.isClosed.Load() {
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

	if cMap.isClosed.Load() {
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
	if cMap.isClosed.Load() {
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
	if cMap.isClosed.Load() {
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
	if cMap.isClosed.Load() {
		return
	}

	// First try with a read lock to check if the key exists
	cMap.RLock()
	node, ok := cMap.data[key]
	cMap.RUnlock()

	if !ok {
		return
	}

	// If the key exists, acquire a write lock and delete it immediately
	cMap.Lock()
	// Check again in case the key was deleted between the read lock and write lock
	if node, ok = cMap.data[key]; ok {
		delete(cMap.data, key)
		node.Clear()
		// Add the node to the buffer for reuse if there's space
		if len(cMap.nodeBuffer) < cMap.maxNodeBufferSize {
			cMap.nodeBufferLock.Lock()
			cMap.nodeBuffer = append(cMap.nodeBuffer, node)
			cMap.nodeBufferLock.Unlock()
		}
	}
	cMap.Unlock()
}

// DeleteBatch removes multiple keys and their associated data from the map. Clears each node before deletion if it exists.
func (cMap *ConcurrentMapWithTTL[T]) DeleteBatch(keys []string) {
	if cMap.isClosed.Load() {
		return
	}

	// First collect all the keys that exist in the map
	keysToDelete := make([]string, 0, len(keys))
	nodesToClear := make([]*MapNode[T], 0, len(keys))

	cMap.RLock()
	for _, key := range keys {
		if node, ok := cMap.data[key]; ok {
			keysToDelete = append(keysToDelete, key)
			nodesToClear = append(nodesToClear, node)
		}
	}
	cMap.RUnlock()

	if len(keysToDelete) == 0 {
		return
	}

	// Then delete them all at once with a single write lock
	cMap.Lock()
	for i, key := range keysToDelete {
		// Check again in case the key was deleted between the read lock and write lock
		if node, ok := cMap.data[key]; ok {
			delete(cMap.data, key)
			node.Clear()
			// Add the node to the buffer for reuse if there's space
			if len(cMap.nodeBuffer) < cMap.maxNodeBufferSize {
				cMap.nodeBufferLock.Lock()
				cMap.nodeBuffer = append(cMap.nodeBuffer, node)
				cMap.nodeBufferLock.Unlock()
			}
		} else {
			// If the key was deleted, clear the node we collected earlier
			nodesToClear[i].Clear()
		}
	}
	cMap.Unlock()
}

// Clear removes all elements from the map and clears their associated nodes.
func (cMap *ConcurrentMapWithTTL[T]) Clear() {
	if cMap.isClosed.Load() {
		return
	}

	cMap.isClosed.Store(true)

	cMap.Lock()
	for key, node := range cMap.data {
		node.Clear()
		delete(cMap.data, key)
	}
	cMap.data = make(map[string]*MapNode[T])

	cMap.cancel()
	cMap.Unlock()

	cMap.keysForDeleteSync.Lock()
	cMap.keysForDelete = make(map[string]struct{}, cMap.maxKeysForDeleteUsage)
	cMap.tickerOnce = sync.Once{}
	cMap.keysForDeleteSync.Unlock()
}

// Len returns the number of elements in the map. It is safe for concurrent access.
func (cMap *ConcurrentMapWithTTL[T]) Len() int {
	if cMap.isClosed.Load() {
		return 0
	}

	cMap.RLock()
	length := len(cMap.data)
	cMap.RUnlock()

	return length
}

// tickCollection periodically decrements the TTL of each node and removes expired nodes until the context is canceled.
func (cMap *ConcurrentMapWithTTL[T]) tickCollection() {
	if cMap.isClosed.Load() {
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
			if cMap.isClosed.Load() {
				return
			}
			if isProcessed {
				continue
			}

			isProcessed = true

			// Collect keys and nodes that need to be processed
			keysToProcess := make([]string, 0, len(cMap.data))
			nodesToProcess := make([]*MapNode[T], 0, len(cMap.data))

			cMap.RLock()
			for key, node := range cMap.data {
				keysToProcess = append(keysToProcess, key)
				nodesToProcess = append(nodesToProcess, node)
			}
			cMap.RUnlock()

			// Process TTL for each node
			expiredKeys := make([]string, 0)
			for i, node := range nodesToProcess {
				// Decrement TTL
				node.Tick()

				// If TTL is expired, mark the key for deletion
				if node.duration <= 0 {
					expiredKeys = append(expiredKeys, keysToProcess[i])
				}
			}

			// Delete expired keys
			if len(expiredKeys) > 0 {
				cMap.DeleteBatch(expiredKeys)
			}

			// Clean up
			keysToProcess = nil
			nodesToProcess = nil
			expiredKeys = nil
			isProcessed = false
		}
	}
}
