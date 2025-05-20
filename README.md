# In-Memory Cache

The library provides two implementations of a thread-safe cache with TTL for different use cases.

## Choosing an Implementation

### ConcurrentMapWithTTL

ConcurrentMapWithTTL represents a thread-safe map with automatic removal of expired values. 

 Recommended when:
- Frequent iteration over all elements (Range is ~88% faster)
- Scenarios requiring frequent full data scans
- Simpler architecture is preferred
- Predictable performance for Range operations is critical

Performance (operations per second): 
- Get: ~9.3M ops/sec
- Delete: ~231K ops/sec
- Set: ~267K ops/sec
- Range: ~1.7K ops/sec

### DynamicShardedMapWithTTL

DynamicShardedMapWithTTL represents a sharded thread-safe map 
with automatic removal of expired values. 

Recommended when:
- Write operations predominate (Set is ~91% faster)
- Read operations are frequent (Get is ~25% faster)
- Delete operations are critical (Delete is ~470% faster)
- High parallel load
- Memory usage is critical (fewer allocations)

Performance (operations per second): 
- Get: ~11.7M ops/sec
- Delete: ~1.3M ops/sec
- Set: ~511K ops/sec
- Range: ~900 ops/sec

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

Recent benchmark results (Windows, bare metal):

| Operation | ConcurrentMap | DynamicShardedMap | Ops/Sec Conversion |
|-----------|---------------|-------------------|-------------------|
| Set | 3741 ns/op | 1956 ns/op | ~267K vs ~511K |
| Get | 107.3 ns/op | 85.58 ns/op | ~9.3M vs ~11.7M |
| Delete | 4321 ns/op | 753 ns/op | ~231K vs ~1.3M |
| Range | 588859 ns/op | 1107338 ns/op | ~1.7K vs ~900 |

## Detailed Comparison

### ConcurrentMapWithTTL Advantages
- 88% faster in range operations
- Simpler internal structure
- More predictable Range performance
- Better for full data scans

### DynamicShardedMapWithTTL Advantages
- 91% faster in write operations
- 25% faster in read operations
- 470% faster in delete operations
- Better scalability under load
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
- Operating System: Windows
- CPU: AMD Ryzen 9 3950X (32 logical cores available)
- Architecture: amd64
- Running on bare metal (not virtualized)

### Key Distribution Impact
The performance of DynamicShardedMapWithTTL significantly depends on key distribution:
- Diverse key prefixes improve shard distribution
- Better performance with uniformly distributed keys
- Avoid sequential or similar key patterns
- Consider key design in your application for optimal performance

Example of good key patterns:
```go
"user:uuid"
// User-related data "session:timestamp" 
//Session data "product:sku" 
//Product data "order:orderid" 
//Order data
```

### Important Notes
⚠️ Performance metrics were obtained in a virtualized environment. Results on physical hardware may vary:
- Higher performance potential on bare metal
- Possibly better results with more available CPU cores
- Different I/O and memory access patterns
- Less overhead without virtualization layer

### Performance Notes
All performance metrics were obtained on Windows with 32 available cores,
which differs from previous Linux virtualized environment tests. Key differences:
- Higher parallelization potential with 32 vs 16 cores
- No virtualization overhead
- Different OS scheduling characteristics
- Different memory management patterns

For the most accurate performance assessment in your environment,
we recommend running the benchmarks in conditions matching your production setup.

Benchmark command used:
```shell
shell go test -bench=BenchmarkCache_Operations -benchmem
```

### Scaling Expectations
- Performance may scale better on systems with >16 cores
- Bare metal deployments likely to show improved throughput
- Memory access patterns could be more efficient
- Lower latency in non-virtualized environments
- Performance metrics are highly dependent on key distribution patterns
- Real-world performance may vary based on your key design
- Consider running benchmarks with your specific use case


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

