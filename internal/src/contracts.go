package src

import (
	"time"
)

// IMapNode represents an interface for managing nodes with TTL, data handling, lifecycle control, and metrics tracking.
type IMapNode[T any] interface {
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
	SetRemoveCallback(remove func())
	Tick()
	GetData() T
	SetData(data T)
	Clear()
	GetDataWithMetrics() (T, time.Time, uint32, uint32)
}
