package main

import (
	"context"
	"fmt"
	"github.com/BarushevEA/in_memory_cache/pkg"
	"time"
)

type Test struct {
	name string
	age  int
}

func main() {
	ctx := context.Background()
	ttl := 1 * time.Second
	ttlDecrement := 500 * time.Millisecond

	cache := pkg.NewShardedCache[*Test](ctx, ttl, ttlDecrement)
	//cache := pkg.NewConcurrentCache[*Test](ctx, ttl, ttlDecrement)

	start := time.Now()
	for i := 0; i < 10000000; i++ {
		err := cache.Set(fmt.Sprintf("%dkey", i), &Test{"name", 20})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println(time.Since(start), cache.Len())
}
