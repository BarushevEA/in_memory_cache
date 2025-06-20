package src

import "time"

// MapNode represents a generic node with a time-to-live (TTL) mechanism, handling data, lifecycle, and expiration.
type MapNode[T any] struct {
	data         T
	ttl          time.Duration
	duration     time.Duration
	ttlDecrement time.Duration
	remove       func()

	createdAt time.Time
	setCount  uint32
	getCount  uint32
}

// NewMapNode creates a new instance of a MapNode with the given data, returning it as an implementation of IMapNode.
func NewMapNode[T any](data T) IMapNode[T] {
	node := &MapNode[T]{}
	node.data = data
	node.createdAt = time.Now()
	node.setCount = 1
	return node
}

// SetRemoveCallback sets the callback function to be invoked when the node needs to be removed due to TTL expiration.
func (node *MapNode[T]) SetRemoveCallback(remove func()) {
	node.remove = remove
}

// SetTTL sets the time-to-live (TTL) duration for the MapNode instance.
func (node *MapNode[T]) SetTTL(ttl time.Duration) {
	node.ttl = ttl
	node.duration = ttl
}

// SetTTLDecrement sets the time duration to decrement from the node's TTL on each tick operation.
func (node *MapNode[T]) SetTTLDecrement(ttlDecrement time.Duration) {
	node.ttlDecrement = ttlDecrement
}

// Tick decreases the node's remaining time-to-live by the decrement value and invokes the removal callback if expired.
func (node *MapNode[T]) Tick() {
	node.duration -= node.ttlDecrement
	if node.duration > 0 {
		return
	}
	if node.remove == nil {
		return
	}

	node.remove()
	node.remove = nil
}

// GetData resets the node's duration to its ttl value and returns the data stored in the node.
func (node *MapNode[T]) GetData() T {
	node.duration = node.ttl
	node.getCount++
	return node.data
}

// SetData sets the data for the MapNode instance.
func (node *MapNode[T]) SetData(data T) {
	node.duration = node.ttl
	node.data = data
	node.setCount++
}

// Clear resets all fields of the MapNode to their zero values, effectively clearing its state and binding.
func (node *MapNode[T]) Clear() {
	node.duration = 0
	node.remove = nil
	node.ttl = 0
	node.ttlDecrement = 0
	node.data = *new(T)
	node.setCount = 0
	node.getCount = 0
}

// GetMetrics returns the creation time, set count, and get count for the MapNode instance.
func (node *MapNode[T]) GetDataWithMetrics() (T, time.Time, uint32, uint32) {
	return node.data, node.createdAt, node.setCount, node.getCount
}
