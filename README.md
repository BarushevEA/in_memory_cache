# In-Memory Cache

A high-performance caching library for Go that provides two thread-safe implementations with smart TTL support. Matches BigCache and FreeCache performance while offering:
- Direct struct support without serialization
- Generic type system integration
- Clean and intuitive API

## Features

- Thread-safe operations
- Smart TTL management with auto-extension
- Generic type support
- Configurable cleanup intervals
- Zero external dependencies

## Implementations

### ConcurrentCache

A thread-safe cache with unified storage space. Optimized for:
- Sequential read operations
- Read-heavy scenarios
- Cases where operation time predictability is important
- Situations requiring simple and clear architecture

### ShardedCache

A segmented cache with dynamic shard management. Optimized for:
- Parallel read/write operations
- Working with large datasets
- High concurrency scenarios
- Efficient memory usage

## When to Use What

Choose **ConcurrentCache** when:
- Read operations dominate
- Operation time predictability is crucial
- Simple debugging and monitoring is required
- Dataset size is relatively small

Choose **ShardedCache** when:
- High parallel load is expected
- Working with large volumes of data
- Write performance is critical
- Memory efficiency is important

## Smart TTL Management

Both cache implementations feature an advanced TTL (Time-To-Live) system:

### Smart TTL Extension
- **Auto-renewal**: TTL automatically resets on each access
- **Active Data Protection**: Frequently accessed items never expire
- **Usage-based Expiration**: Only truly inactive data gets removed

### How it works:
1. When an item is added to cache:
   - Initial TTL is set
   - Expiration countdown begins
2. On each access (Get/Set operations):
   - TTL is reset to its initial value
   - Item gets "fresh" lifetime
3. For inactive items:
   - TTL decrements gradually
   - Removal occurs only after full TTL period without access

## Usage Examples

### Basic Usage

```go
// Create 
cache ctx := context.Background() 
cache := NewShardedCache[string](ctx, 5 * time.Minute, 1 * time.Minute)

// Basic operations 
cache.Set("key", "value") 
value, exists := cache.Get("key") 
cache.Delete("key")

// Iterate over elements 
cache.Range(func(key string, value string) bool { 
//Process key-value pair return true 
//continue iteration 
})
```

### With Custom Type
```go
type User struct { 
	ID int 
	Name string 
}

cache := NewShardedCache[*User](ctx, 5 * time.Minute, 1 * time.Minute) 
cache.Set("user:1", &User{ID: 1, Name: "John"})
```

## Metrics and Advanced Features
The library provides extended methods for retrieving metrics on cache elements, allowing for deeper analysis of their usage and behavior.

### RangeWithMetrics
This method allows you to iterate over all cache elements, obtaining not only the key and value but also additional metrics for each element:

- createdAt time.Time: The time the item was added to the cache.
- setCount uint32: The number of Set operations for this item (including the initial addition).
- getCount uint32: The number of Get operations for this item.

## Usage Example:
```go
cache.RangeWithMetrics(
	func(key, value, createdAt, setCount, getCount) bool {
    fmt.Printf(
		"Key: %s, Value: %v, Created At: %s, Set Count: %d, Get Count: %d\n", 
		key, 
		value, 
		createdAt.Format(time.RFC3339), 
		setCount, 
		getCount)
    return true // Continue iteration
})
```

### GetNodeValueWithMetrics
This method allows you to retrieve the value of an element by key along with its metrics without incrementing getCount:

- value T: The value of the element.
- createdAt time.Time: The creation time of the element.
- setCount uint32: The number of Set operations for this element.
- getCount uint32: The number of Get operations for this element.
- found bool: A flag indicating whether the element was found in the cache.

## Usage Example:
```go
value, createdAt, setCount, getCount, exists := cache.GetNodeValueWithMetrics("key")
if exists {
    fmt.Printf(
		"Value: %v, Created At: %s, Set Count: %d, Get Count: %d\n", 
		value, 
		createdAt.Format(time.RFC3339), 
		setCount, 
		getCount)
}
```

## Performance

The library is optimized for various use cases:
- Parallel read/write operations
- Large dataset handling
- Efficient memory management
- Predictable response time

> **Important**: Actual performance depends on specific use case, data size, access patterns, and system configuration. Testing in production-like conditions is recommended.

## Memory Management

Both cache types feature automatic TTL management:
- Configurable item lifetime
- Periodic cleanup of expired data
- Efficient memory deallocation

## Usage Recommendations

1. **Implementation Choice**:
    - Test both implementations
    - Consider data access patterns
    - Evaluate memory requirements

2. **TTL Configuration**:
    - Choose TTL based on data relevance
    - Consider update frequency
    - Balance between freshness and performance

3. **Monitoring**:
    - Track memory usage
    - Monitor operation execution time
    - Keep track of cache item count

## Benchmarking

Our benchmarks show that:
- ShardedCache excels in parallel workloads
- ConcurrentCache provides stable read performance
- Both implementations outperform standard map with mutex
- Memory usage scales efficiently with data size

> Note: Run your own benchmarks to validate performance in your specific use case

## Thread Safety

Both implementations provide comprehensive thread safety:
- All operations are atomic
- No external synchronization required
- Safe for concurrent access from multiple goroutines

## Error Handling

The library implements robust error handling:
- Clear error types
- Context cancellation support
- Proper cleanup on errors

## Requirements

- Go 1.24 or higher
- Generics support

## Best Practices

1. **Key Design**:
    - Use consistent key naming patterns
    - Avoid overly long keys
    - Consider key distribution for ShardedCache

2. **Resource Management**:
    - Set appropriate TTL values
    - Configure cleanup intervals
    - Monitor memory usage

3. **Performance Optimization**:
    - Choose appropriate cache size
    - Balance between memory and performance
    - Consider your access patterns