package utils

import "github.com/cespare/xxhash/v2"

// GetTopHash calculates and returns the top 8 bits of a hash value for the given key string.
// For keys shorter than or equal to 32 characters, a multiplicative hashing strategy is used.
// For longer keys, an additive hashing strategy with bit-shifting is applied.
func GetTopHash(key string) uint8 {
	hash := xxhash.Sum64String(key)
	return uint8(hash >> 56)
}
