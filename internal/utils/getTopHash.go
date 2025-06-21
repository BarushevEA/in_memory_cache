package utils

import "github.com/cespare/xxhash/v2"

// GetTopHash computes and returns the most significant 8 bits of the xxHash64 checksum for the given key.
func GetTopHash(key string) uint8 {
	hash := xxhash.Sum64String(key)
	return uint8(hash >> 56)
}
