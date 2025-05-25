package types

import "time"

// Metric represents a generic structure to store metadata and access statistics for a key-value pair in a cache.
type Metric[T any] struct {
	Key         string
	SetCount    uint32
	GetCount    uint32
	Value       T
	TimeCreated time.Time
	Exists      bool
}

// BatchNode represents a node used in batch operations with key, value, and existence flag.
type BatchNode[T any] struct {
	Key    string
	Value  T
	Exists bool
}
