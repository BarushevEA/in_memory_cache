# In-Memory Cache

The library provides two implementations of a thread-safe cache with TTL for different use cases.

## Choosing an Implementation

### ConcurrentMapWithTTL

ConcurrentMapWithTTL represents a thread-safe map with automatic removal of expired values. 

 Recommended when:
- Read operations predominate (Get is 37.9% faster) 
- Frequent deletion operations (Delete is 30.7% faster)
- Frequent iteration over all elements (Range is 20.8% faster) 
- Predictable operation performance is important 

Performance (operations per second): 
- Get: ~20.6M ops/sec 
- Delete: ~7.1M ops/sec 
- Set: ~197K ops/sec 
- Range: ~74K ops/sec

### DynamicShardedMapWithTTL

DynamicShardedMapWithTTL represents a sharded thread-safe map 
with automatic removal of expired values. 

Recommended when:
- Write operations predominate (Set is 165.8% faster) 
- High parallel load (HighLoad is 30.2% faster) 
- Mixed read/write operations (SetGet is 47.5% faster) 
- Memory usage is critical (fewer allocations) 

Performance (operations per second): 
- Get: ~12.8M ops/sec 
- Delete: ~4.9M ops/sec 
- Set: ~523K ops/sec 
- Range: ~58K ops/sec

## Implementation Choice Guide

### ConcurrentMapWithTTL
Optimized for:
- ✅ Frequent read operations
- ✅ Frequent delete operations
- ✅ Iteration over all elements
- ✅ Predictable performance

```go
cache := NewConcurrentMapWithTTL[string](ctx, ttl, cleanupInterval)
```

### DynamicShardedMapWithTTL
Optimized for:
- ✅ Frequent write operations
- ✅ High parallel load
- ✅ Mixed read/write operations
- ✅ Memory efficiency

#### 🔄 Adaptive Sharding
- Automatically adjusts the number of shards based on usage patterns
- Self-optimizing performance under varying loads
- No manual configuration needed for shard count
- Reduces memory overhead by cleaning up unused shards
- Adapts to changing access patterns in real-time

#### 🎯 Ideal Use Cases
- Systems with varying load patterns
- Applications where access patterns change over time
- Microservices with unpredictable usage spikes
- Environments where manual tuning is impractical


```go
cache := NewDynamicShardedMapWithTTL[string](ctx, ttl, cleanupInterval)
```

Dynamic Sharding Process:

Initial State:
[Empty Map]

After First Writes:
[Shard 1] --> Data
[Shard 2] --> Data

Under Heavy Load:
[Shard 1] --> Data
[Shard 2] --> Data
[Shard 3] --> Data
[Shard 4] --> Data
...

After Data Cleanup:
[Shard 1] --> Data
[Shard 2] --> Data
(empty shards automatically removed)

## Performance Comparison

| Operation | ConcurrentMap | DynamicShardedMap | Advantage |
|-----------|---------------|-------------------|-----------|
| Get | 20.6M ops/sec | 12.8M ops/sec | ConcurrentMap |
| Set | 197K ops/sec | 523K ops/sec | DynamicShardedMap |
| Delete | 7.1M ops/sec | 4.9M ops/sec | ConcurrentMap |
| Range | 74K ops/sec | 58K ops/sec | ConcurrentMap |
| SetGet | 1.7M ops/sec | 2.5M ops/sec | DynamicShardedMap |
| HighLoad | 1.8M ops/sec | 2.4M ops/sec | DynamicShardedMap |

## Detailed Comparison

### ConcurrentMapWithTTL Advantages
- 37.9% faster in read operations
- 30.7% faster in delete operations
- 20.8% faster in range operations
- More predictable performance across operations

### DynamicShardedMapWithTTL Advantages
- 165.8% faster in write operations
- 47.5% faster in combined read/write operations
- 30.2% faster under high parallel load
- More efficient memory usage with fewer allocations

## Usage Examples

### Basic Usage with ConcurrentMapWithTTL

```go
ctx := context.Background() 
cache := NewConcurrentMapWithTTL[string](ctx, 5_time.Minute, 1_time.Minute)

// Set value cache.Set("key", "value")
// Get value value, exists := cache.Get("key")
// Delete value cache.Delete("key")
// Iterate over all elements cache.Range(func(key string, value string) bool { // Process key-value pair return true })
```
### Basic Usage with DynamicShardedMapWithTTL

```go
ctx := context.Background() 
cache := NewDynamicShardedMapWithTTL[string](ctx, 5_time.Minute, 1_time.Minute)
// Set value cache.Set("key", "value")
// Get value value, exists := cache.Get("key")
// Delete value cache.Delete("key")
// Iterate over all elements cache.Range(func(key string, value string) bool { // Process key-value pair return true })

```
## Benchmark Environment

### Test Environment Specifications
- Virtual Machine Environment
- CPU: AMD Ryzen 9 3950X (16 logical cores allocated)
- OS: Linux
- Architecture: amd64

### Important Notes
⚠️ Performance metrics were obtained in a virtualized environment. Results on physical hardware may vary:
- Higher performance potential on bare metal
- Possibly better results with more available CPU cores
- Different I/O and memory access patterns
- Less overhead without virtualization layer

### Scaling Expectations
- Performance may scale better on systems with >16 cores
- Bare metal deployments likely to show improved throughput
- Memory access patterns could be more efficient
- Lower latency in non-virtualized environments

```go
// Example of configuration considering environment
cache := NewDynamicShardedMapWithTTL[string](
    ctx,
    ttl,
    cleanupInterval,
)

// The cache will automatically adapt to available resources
// No manual tuning needed for different environments
```

