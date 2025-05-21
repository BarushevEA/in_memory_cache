package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"math/rand"
	"strings"
	"testing"
	"time"
)

//type RealWorldData struct {
//	ID            int64     `json:"id"`
//	UserID        int64     `json:"user_id"`
//	TransactionID string    `json:"transaction_id"`
//	Amount        float64   `json:"amount"`
//	Currency      string    `json:"currency"`
//	Status        string    `json:"status"`
//	CreatedAt     time.Time `json:"created_at"`
//	UpdatedAt     time.Time `json:"updated_at"`
//	Metadata      Metadata  `json:"metadata"`
//}
//
//type Metadata struct {
//	IP        string            `json:"ip"`
//	UserAgent string            `json:"user_agent"`
//	Tags      []string          `json:"tags"`
//	Extra     map[string]string `json:"extra"`
//}

func generateRandomDataWithKey(keyLength string) *RealWorldData {
	currencies := []string{"USD", "EUR", "GBP", "JPY"}
	statuses := []string{"pending", "completed", "failed", "cancelled"}
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X)",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	}

	now := time.Now()
	transactionID := ""

	switch keyLength {
	case "short": // ~20 байт
		transactionID = fmt.Sprintf("trx_%d", rand.Int63())
	case "medium": // ~50 байт
		transactionID = fmt.Sprintf("transaction_%d_%s_%s",
			rand.Int63(),
			time.Now().Format(time.RFC3339),
			currencies[rand.Intn(len(currencies))])
	case "long": // >100 байт
		padding := strings.Repeat("pad_", 25) // 100 байт падинга
		transactionID = fmt.Sprintf("long_transaction_%d_%s_%s_%s_%s",
			rand.Int63(),
			time.Now().Format(time.RFC3339),
			currencies[rand.Intn(len(currencies))],
			statuses[rand.Intn(len(statuses))],
			padding)
	}

	return &RealWorldData{
		ID:            rand.Int63(),
		UserID:        rand.Int63n(1000000),
		TransactionID: transactionID,
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

func BenchmarkRealWorldScenario_key_test(b *testing.B) {
	keyLengths := []string{"short", "medium", "long"}

	for _, keyLength := range keyLengths {
		b.Run(fmt.Sprintf("KeyLength_%s", keyLength), func(b *testing.B) {
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
								data := generateRandomDataWithKey(keyLength)
								_ = concurrentCache.Set(data.TransactionID, data)
							}
						})

						b.Run("Read", func(b *testing.B) {
							data := generateRandomDataWithKey(keyLength)
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
										data := generateRandomDataWithKey(keyLength)
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
								data := generateRandomDataWithKey(keyLength)
								_ = shardedCache.Set(data.TransactionID, data)
							}
						})

						b.Run("Read", func(b *testing.B) {
							data := generateRandomDataWithKey(keyLength)
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
										data := generateRandomDataWithKey(keyLength)
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
								data := generateRandomDataWithKey(keyLength)
								jsonData, _ := json.Marshal(data)
								_ = bigCache.Set(data.TransactionID, jsonData)
							}
						})

						b.Run("Read", func(b *testing.B) {
							data := generateRandomDataWithKey(keyLength)
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
										data := generateRandomDataWithKey(keyLength)
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
								data := generateRandomDataWithKey(keyLength)
								jsonData, _ := json.Marshal(data)
								_ = freeCache.Set([]byte(data.TransactionID), jsonData, int(ttl.Seconds()))
							}
						})

						b.Run("Read", func(b *testing.B) {
							data := generateRandomDataWithKey(keyLength)
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
										data := generateRandomDataWithKey(keyLength)
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
		})
	}
}
