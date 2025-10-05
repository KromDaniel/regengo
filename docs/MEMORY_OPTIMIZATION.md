# Memory Optimization Strategies for Regengo

## Current State

Regengo currently allocates a new backtracking stack for every match operation:

```go
func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0
    stack := make([][2]int, 0)  // ← New allocation every call
    // ...
}
```

**Benchmark Results** (current):

```
BenchmarkEmailRegengo-12    2027746    585.4 ns/op    2352 B/op    23 allocs/op
BenchmarkURLRegengo-12      3036224    392.0 ns/op    2544 B/op    13 allocs/op
BenchmarkIPv4Regengo-12     4577810    252.6 ns/op     768 B/op    14 allocs/op
```

## Problem Analysis

1. **Per-Call Allocation**: Each match creates a new slice for the backtracking stack
2. **GC Pressure**: Frequent allocations create garbage collection overhead
3. **Memory Waste**: Stack capacity is often larger than needed after append operations

## Optimization Strategies

### Strategy 1: Stack Pooling with sync.Pool (Recommended)

Use Go's `sync.Pool` to reuse stack slices across multiple match operations.

**Implementation**:

```go
package generated

import "sync"

var emailStackPool = sync.Pool{
    New: func() interface{} {
        stack := make([][2]int, 0, 32) // Pre-allocate reasonable capacity
        return &stack
    },
}

func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0

    // Get stack from pool
    stackPtr := emailStackPool.Get().(*[][2]int)
    stack := (*stackPtr)[:0] // Reset length, keep capacity

    nextInstruction := 1

    // Ensure stack is returned to pool
    defer func() {
        // Clear references to prevent memory leaks
        for i := range stack {
            stack[i] = [2]int{0, 0}
        }
        *stackPtr = stack[:0]
        emailStackPool.Put(stackPtr)
    }()

    // ... rest of matching logic
}
```

**Benefits**:

- ✅ Zero allocations for typical cases (stack size ≤ 32)
- ✅ Thread-safe (sync.Pool handles concurrency)
- ✅ Automatic memory management
- ✅ Works across goroutines

**Expected Improvement**: 50-80% reduction in allocations

### Strategy 2: Pre-allocated Fixed-Size Stack

For patterns with known maximum backtracking depth, use a fixed-size stack on the function stack.

**Implementation**:

```go
func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0

    // Stack allocated on function stack (no heap allocation)
    var stackArray [64][2]int
    stack := stackArray[:0]

    nextInstruction := 1

    // ... rest of matching logic
    // Note: If stack exceeds 64, fall back to append (rare)
}
```

**Benefits**:

- ✅ Zero heap allocations for patterns with shallow backtracking
- ✅ Simple implementation
- ✅ No synchronization overhead

**Drawbacks**:

- ❌ Increases function stack size
- ❌ Potential stack overflow for deeply nested patterns
- ❌ Wastes stack space if not fully used

**Expected Improvement**: 90-100% reduction in allocations (for patterns staying within limit)

### Strategy 3: Hybrid Approach (Best of Both Worlds)

Combine fixed-size stack with pool fallback:

```go
func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0

    // Try stack allocation first
    var stackArray [32][2]int
    stack := stackArray[:0]

    usePool := false
    var poolStack *[][2]int

    nextInstruction := 1

    defer func() {
        if usePool {
            *poolStack = (*poolStack)[:0]
            emailStackPool.Put(poolStack)
        }
    }()

    goto StepSelect

TryFallback:
    // If stack exceeds capacity, switch to pool
    if len(stack) >= 32 && !usePool {
        poolStack = emailStackPool.Get().(*[][2]int)
        *poolStack = append((*poolStack)[:0], stack...)
        stack = *poolStack
        usePool = true
    }
    // ... rest of logic
}
```

**Benefits**:

- ✅ Fast path uses zero heap allocations
- ✅ Fallback for complex patterns
- ✅ Optimal for most use cases

**Expected Improvement**: 80-95% reduction in allocations

### Strategy 4: Context-Based Stack (For API Users)

Allow users to provide their own stack buffer:

```go
type MatchContext struct {
    stack [][2]int
}

func NewMatchContext() *MatchContext {
    return &MatchContext{
        stack: make([][2]int, 0, 64),
    }
}

func (ctx *MatchContext) EmailMatchString(input string) bool {
    l := len(input)
    offset := 0
    stack := ctx.stack[:0] // Reuse context's stack

    // ... matching logic

    ctx.stack = stack // Save for next call
    return result
}
```

**Benefits**:

- ✅ User controls allocation strategy
- ✅ Perfect for hot paths in user code
- ✅ Can be cached per-goroutine

**Expected Improvement**: 100% elimination of allocations (user manages lifecycle)

## Recommended Implementation Plan

### Phase 1: Add sync.Pool Support (Immediate)

1. Add `--pool` flag to CLI to generate pooled versions
2. Update compiler to generate pool-based code
3. Run benchmarks to measure improvement

**Code Changes**:

- `pkg/regengo/regengo.go`: Add `UsePool bool` to Options
- `internal/compiler/compiler.go`: Generate pool code when enabled
- `cmd/regengo/main.go`: Add `-pool` flag

### Phase 2: Analyze Pattern Depth (Short-term)

1. Add analysis pass to determine maximum backtracking depth
2. Use fixed-size stack when depth < 32
3. Use pool for deeper patterns

### Phase 3: Hybrid Strategy (Medium-term)

1. Always start with fixed-size stack
2. Automatically upgrade to pool if exceeded
3. Make this the default behavior

### Phase 4: Context API (Long-term)

1. Add optional Context-based API
2. Keep simple API for casual users
3. Document performance characteristics

## Benchmark Projections

Based on similar optimizations in other projects:

| Strategy    | Allocs/op | Memory/op | Speedup  |
| ----------- | --------- | --------- | -------- |
| Current     | 23        | 2352 B    | 1.0x     |
| Pool        | 1-5       | 128 B     | 1.2-1.5x |
| Fixed Stack | 0-1       | 0-64 B    | 1.3-1.8x |
| Hybrid      | 0-2       | 0-128 B   | 1.4-2.0x |
| Context     | 0         | 0 B       | 1.5-2.5x |

## Implementation Priority

1. **High Priority**: sync.Pool (Strategy 1)

   - Easy to implement
   - Significant gains
   - No API changes needed

2. **Medium Priority**: Hybrid (Strategy 3)

   - Best overall performance
   - Moderate complexity
   - Compatible with existing code

3. **Low Priority**: Context API (Strategy 4)
   - Advanced use case
   - Breaking API change
   - Overkill for most users

## Next Steps

1. Implement sync.Pool support in compiler
2. Add benchmarks comparing strategies
3. Document memory characteristics in README
4. Consider making pool the default in v2.0

## References

- Go sync.Pool: https://pkg.go.dev/sync#Pool
- Stack vs Heap allocation: https://go.dev/blog/stack-allocation
- Regex engine optimization: https://swtch.com/~rsc/regexp/
