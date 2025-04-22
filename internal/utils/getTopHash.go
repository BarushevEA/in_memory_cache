package utils

// getTopHash calculates and returns the top 8 bits of a hash value for the given key string.
// For keys shorter than or equal to 32 characters, a multiplicative hashing strategy is used.
// For longer keys, an additive hashing strategy with bit-shifting is applied.
func getTopHash(key string) uint8 {
	if len(key) <= 32 {
		var hash uint64
		for i := 0; i < len(key); i++ {
			hash = hash*0x9E3779B97F4A7C15 + uint64(key[i])
		}
		return uint8(hash >> 56)
	}

	var hash uint64
	for i := 0; i < len(key); i++ {
		hash = (hash << 5) + hash + uint64(key[i])
	}
	return uint8(hash >> 56)
}
