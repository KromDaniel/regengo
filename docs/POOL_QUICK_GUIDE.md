# Memory Optimization Summary

## Quick Comparison

### Before Optimization

```go
func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0
    stack := make([][2]int, 0)  // ← New allocation every call
    // ... matching logic
}
```

**Performance**: 585 ns/op, 2352 B/op, 23 allocs/op

### After Optimization (with -pool flag)

```go
var emailStackPool = sync.Pool{
    New: func() interface{} {
        stack := make([][2]int, 0, 32)
        return &stack
    },
}

func EmailMatchString(input string) bool {
    l := len(input)
    offset := 0
    stackPtr := emailStackPool.Get().(*[][2]int)
    stack := (*stackPtr)[:0]
    defer func() {
        for i := range stack {
            stack[i] = [2]int{0, 0}
        }
        *stackPtr = stack[:0]
        emailStackPool.Put(stackPtr)
    }()
    // ... matching logic
}
```

**Performance**: 289 ns/op, 0 B/op, 0 allocs/op

## Impact

| Metric          | Before       | After       | Improvement              |
| --------------- | ------------ | ----------- | ------------------------ |
| **Speed**       | 585 ns/op    | 289 ns/op   | **2.0x faster**          |
| **Memory**      | 2352 B/op    | 0 B/op      | **100% reduction**       |
| **Allocations** | 23 allocs/op | 0 allocs/op | **100% reduction**       |
| **vs regexp**   | 1.5x faster  | 3.1x faster | **2.1x additional gain** |

## Usage

Simply add the `-pool` flag when generating code:

```bash
# Without pool
regengo -pattern '[\w\.+-]+@[\w\.-]+\.[\w\.-]+' -name Email -output email.go

# With pool (recommended)
regengo -pattern '[\w\.+-]+@[\w\.-]+\.[\w\.-]+' -name Email -output email.go -pool
```

Or programmatically:

```go
opts := regengo.Options{
    Pattern:    `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
    Name:       "Email",
    OutputFile: "email.go",
    Package:    "matcher",
    UsePool:    true,  // ← Enable pool
}
```

## When to Use

✅ **Always use `-pool` for:**

- Production deployments
- High-frequency matching
- Hot paths in your application
- Multi-threaded/concurrent environments
- When performance matters

⚠️ **Consider non-pooled only for:**

- Quick prototypes/scripts
- One-time matches
- Extreme memory constraints (embedded systems)

## Technical Details

The optimization works by:

1. Creating a `sync.Pool` that maintains a set of reusable stack buffers
2. Getting a buffer from the pool at function start
3. Resetting the buffer length to 0 (keeps capacity)
4. Using the buffer for backtracking
5. Clearing references and returning buffer to pool via `defer`

The pool automatically:

- Manages buffer lifecycle
- Handles concurrent access
- Clears unused buffers during GC
- Scales with application load

## Benchmark Results

See [POOL_OPTIMIZATION_RESULTS.md](POOL_OPTIMIZATION_RESULTS.md) for complete benchmark data.

**Key Findings**:

- Email pattern: 3.1x faster than standard regexp
- URL pattern: 4.5x faster than standard regexp
- IPv4 pattern: 4.9x faster than standard regexp
- All with 0 allocations per operation
