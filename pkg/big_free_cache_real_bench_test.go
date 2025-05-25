package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"math/rand"
	"testing"
	"time"
)

// RealWorldData represents information related to financial transactions in a real-world scenario.
// ID is the unique identifier of the record.
// UserID indicates the user associated with the transaction.
// TransactionID is the unique identification string for the transaction.
// Amount denotes the monetary value of the transaction.
// Currency is the currency code (ISO 4217) of the transaction.
// Status represents the current state of the transaction (e.g., pending, completed, failed).
// CreatedAt indicates when the record was created.
// UpdatedAt indicates when the record was last updated.
// Metadata contains additional information such as IP and tags for the record.
type RealWorldData struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Metadata      Metadata  `json:"metadata"`
}

// Metadata represents additional information associated with an entity such as IP address, user agent, tags, and extra data.
type Metadata struct {
	IP        string            `json:"ip"`
	UserAgent string            `json:"user_agent"`
	Tags      []string          `json:"tags"`
	Extra     map[string]string `json:"extra"`
}

// generateRandomData generates and returns a pointer to a RealWorldData object populated with random data values.
func generateRandomData() *RealWorldData {
	currencies := []string{"USD", "EUR", "GBP", "JPY"}
	statuses := []string{"pending", "completed", "failed", "cancelled"}
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	}

	now := time.Now()
	return &RealWorldData{
		ID:            rand.Int63(),
		UserID:        rand.Int63n(1000000),
		TransactionID: fmt.Sprintf("trx_%d", rand.Int63()),
		Amount:        rand.Float64() * 1000,
		Currency:      currencies[rand.Intn(len(currencies))],
		Status:        statuses[rand.Intn(len(statuses))],
		CreatedAt:     now.Add(-time.Duration(rand.Intn(24)) * time.Hour),
		UpdatedAt:     now,
		Metadata: Metadata{
			IP:        fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			UserAgent: userAgents[rand.Intn(len(userAgents))],
			Tags:      []string{"tag1", "tag2", "tag3"}[0 : rand.Intn(3)+1],
			Extra: map[string]string{
				"source":    fmt.Sprintf("source_%d", rand.Intn(5)),
				"processor": fmt.Sprintf("processor_%d", rand.Intn(3)),
			},
		},
	}
}

// BenchmarkRealWorldScenario measures the performance of multiple caching implementations in various real-world scenarios.
// It benchmarks write, read, and mixed operations using ConcurrentCache, ShardedCache, BigCache, and FreeCache.
// The function also evaluates parallel mixed workloads with varying operation distributions for comprehensive analysis.
func BenchmarkRealWorldScenario(b *testing.B) {
	ctx := context.Background()
	ttl := 5 * time.Minute
	ttlDecrement := 1 * time.Second

	concurrentCache := NewConcurrentCache[*RealWorldData](ctx, ttl, ttlDecrement)
	shardedCache := NewShardedCache[*RealWorldData](ctx, ttl, ttlDecrement)

	bigcacheConfig := bigcache.DefaultConfig(ttl)
	bigcacheConfig.Verbose = false
	bigcacheConfig.Logger = nil
	bigCache, _ := bigcache.New(ctx, bigcacheConfig)

	freeCache := freecache.NewCache(1024 * 1024 * 100)

	benchmarks := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{
			name: "ConcurrentCache_RealWorld",
			fn: func(b *testing.B) {
				b.Run("Write", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						data := generateRandomData()
						_ = concurrentCache.Set(data.TransactionID, data)
					}
				})

				b.Run("Read", func(b *testing.B) {
					data := generateRandomData()
					_ = concurrentCache.Set(data.TransactionID, data)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = concurrentCache.Get(data.TransactionID)
					}
				})

				b.Run("Mixed", func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							if rand.Float32() < 0.7 {
								data := generateRandomData()
								if rand.Float32() < 0.3 {
									_ = concurrentCache.Set(data.TransactionID, data)
								} else {
									_, _ = concurrentCache.Get(data.TransactionID)
								}
							}
						}
					})
				})
			},
		},
		{
			name: "ShardedCache_RealWorld",
			fn: func(b *testing.B) {
				b.Run("Write", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						data := generateRandomData()
						_ = shardedCache.Set(data.TransactionID, data)
					}
				})

				b.Run("Read", func(b *testing.B) {
					data := generateRandomData()
					_ = shardedCache.Set(data.TransactionID, data)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = shardedCache.Get(data.TransactionID)
					}
				})

				b.Run("Mixed", func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							if rand.Float32() < 0.7 {
								data := generateRandomData()
								if rand.Float32() < 0.3 {
									_ = shardedCache.Set(data.TransactionID, data)
								} else {
									_, _ = shardedCache.Get(data.TransactionID)
								}
							}
						}
					})
				})
			},
		},
		{
			name: "BigCache_RealWorld",
			fn: func(b *testing.B) {
				b.Run("Write", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						data := generateRandomData()
						jsonData, _ := json.Marshal(data)
						_ = bigCache.Set(data.TransactionID, jsonData)
					}
				})

				b.Run("Read", func(b *testing.B) {
					data := generateRandomData()
					jsonData, _ := json.Marshal(data)
					_ = bigCache.Set(data.TransactionID, jsonData)
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = bigCache.Get(data.TransactionID)
					}
				})

				b.Run("Mixed", func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							if rand.Float32() < 0.7 {
								data := generateRandomData()
								if rand.Float32() < 0.3 {
									jsonData, _ := json.Marshal(data)
									_ = bigCache.Set(data.TransactionID, jsonData)
								} else {
									_, _ = bigCache.Get(data.TransactionID)
								}
							}
						}
					})
				})
			},
		},
		{
			name: "FreeCache_RealWorld",
			fn: func(b *testing.B) {
				b.Run("Write", func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						data := generateRandomData()
						jsonData, _ := json.Marshal(data)
						_ = freeCache.Set([]byte(data.TransactionID), jsonData, int(ttl.Seconds()))
					}
				})

				b.Run("Read", func(b *testing.B) {
					data := generateRandomData()
					jsonData, _ := json.Marshal(data)
					_ = freeCache.Set([]byte(data.TransactionID), jsonData, int(ttl.Seconds()))
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = freeCache.Get([]byte(data.TransactionID))
					}
				})

				b.Run("Mixed", func(b *testing.B) {
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							if rand.Float32() < 0.7 {
								data := generateRandomData()
								if rand.Float32() < 0.3 {
									jsonData, _ := json.Marshal(data)
									_ = freeCache.Set([]byte(data.TransactionID), jsonData, int(ttl.Seconds()))
								} else {
									_, _ = freeCache.Get([]byte(data.TransactionID))
								}
							}
						}
					})
				})
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, bm.fn)
	}
}
