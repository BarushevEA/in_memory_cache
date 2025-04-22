package pkg

import "time"

type ICache[T any] interface {
	Set(key string, value T) error
	Get(key string) (T, bool)
	Delete(key string)
	Clear()

	Len() int
	Range(func(key string, value T) bool) error
	SetTTL(ttl time.Duration)
	SetTTLDecrement(ttlDecrement time.Duration)
}
