# In-Memory Cache

A Go library providing two thread-safe cache implementations with TTL support for different use cases.

## Implementations

### ConcurrentMapWithTTL

A thread-safe map with automatic expiration of values. Optimized for:
- Range operations (4-5x faster than sharded version)
- Stable read performance
- Predictable behavior with full data scans
- Simpler architecture

### DynamicShardedMapWithTTL

A sharded thread-safe map with automatic expiration. Features dynamic shard management and optimized for:
- Write operations under parallel load
- Large datasets
- Memory efficiency
- Automatic adaptation to load patterns

## Implementation Choice Guide

> **Important**: Performance characteristics heavily depend on your specific use case, data size, access patterns, and system configuration. We recommend running benchmarks in conditions similar to your production environment.

Choose **ConcurrentMapWithTTL** when:
- You need predictable Range operation performance
- Your workload is read-heavy
- You perform frequent full data scans
- Simpler architecture is preferred

Choose **DynamicShardedMapWithTTL** when:
- Your workload is write-heavy
- You work with large datasets
- Memory efficiency is critical
- You need automatic adaptation to varying loads

## Usage Examples

### ConcurrentMapWithTTL

```go
ctx := context.Background() cache := NewConcurrentMapWithTTL[string](ctx, 5_time.Minute, 1_time.Minute)
// Set value cache.Set("key", "value")
// Get value value, exists := cache.Get("key")
// Delete value cache.Delete("key")
// Iterate over elements cache.Range(func(key string, value string) bool { // Process key-value pair return true // continue iteration })
```

### DynamicShardedMapWithTTL

```go
ctx := context.Background() cache := NewDynamicShardedMapWithTTL[string](ctx, 5_time.Minute, 1_time.Minute)
// Same API as ConcurrentMapWithTTL
```


## Dynamic Sharding Process

The DynamicShardedMapWithTTL automatically manages shards based on usage:

1. Initial state: Empty map
2. As data is added: Creates shards as needed
3. Under load: Distributes data across shards
4. After cleanup: Removes empty shards

### Key Distribution

The performance of DynamicShardedMapWithTTL depends on key distribution:
- Use diverse key prefixes
- Aim for uniform distribution
- Avoid sequential patterns

Example key patterns:

```go
"user:uuid" 
// User data "session:ts" 
//Session data "product:sku" 
//Product data "order:id" 
//Order data
```

## Performance Considerations

Performance varies based on several factors:
- Hardware configuration
- Operating system
- Load patterns
- Data size
- Key distribution
- TTL settings

For accurate performance assessment:
1. Run benchmarks in your environment
2. Test with your typical data patterns
3. Simulate your expected load
4. Monitor memory usage
5. Verify TTL behavior


## Environment Impact

Performance can vary significantly based on:
- Number of CPU cores
- Memory architecture
- Operating system scheduling
- Virtualization overhead
- Hardware specifications

Always test in an environment similar to your production setup.