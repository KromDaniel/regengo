# Regengo vs Go stdlib regex: Complexity Analysis Report

## Executive Summary

This report compares the runtime and memory complexity of regengo (compile-time regex code generation) versus Go's standard library `regexp` package (runtime interpretation).

**Key Findings:**
- regengo is **2-16x faster** for most patterns
- regengo uses **50-100% less memory** per operation
- Thompson NFA prevents catastrophic backtracking (O(2^n) → O(n*m))
- Trade-off: Patterns with nested quantifiers + captures have memoization overhead

---

## 1. Theoretical Complexity Analysis

### 1.1 Runtime Complexity

| Engine | Pattern Type | Time Complexity | Notes |
|--------|--------------|-----------------|-------|
| **Go stdlib** | All patterns | O(n*m) | Uses Thompson-like NFA internally |
| **regengo (backtracking)** | Simple patterns | O(n) to O(n*m) | Faster due to compiled code |
| **regengo (Thompson)** | Pathological patterns | O(n*m) | Guaranteed polynomial |
| **regengo (memoized)** | Complex + end anchor | O(n*m) | Polynomial with overhead |

Where:
- `n` = input string length
- `m` = number of NFA states (pattern complexity)

### 1.2 Memory Complexity

| Engine | Match Operations | Capture Operations |
|--------|------------------|-------------------|
| **Go stdlib** | O(m) per call | O(m + k) per call |
| **regengo (backtracking)** | O(1) amortized* | O(k) amortized* |
| **regengo (Thompson)** | O(1) | N/A (uses backtracking for captures) |
| **regengo (memoized)** | O(n*m) worst case | O(n*m + k) worst case |

Where:
- `k` = number of capture groups
- `*` = with sync.Pool enabled (default)

---

## 2. Benchmark Results

### 2.1 Date Pattern (Simple, No Catastrophic Risk)

Pattern: `(\d{4})-(\d{2})-(\d{2})`

| Operation | stdlib | regengo | regengo (reuse) | Speedup |
|-----------|--------|---------|-----------------|---------|
| FindString | 110 ns | 19 ns | 7 ns | **5.8x - 15.7x** |
| Memory | 128 B/op | 64 B/op | 0 B/op | **50-100% less** |
| Allocations | 2 | 1 | 0 | **50-100% less** |

### 2.2 Email Pattern (Medium Complexity)

Pattern: `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`

| Operation | stdlib | regengo | regengo (reuse) | Speedup |
|-----------|--------|---------|-----------------|---------|
| FindString | 240 ns | 92 ns | 83 ns | **2.6x - 2.9x** |
| MatchString | 1473 ns | 484 ns | - | **3.0x** |
| Memory | 128 B/op | 64 B/op | 0 B/op | **50-100% less** |

### 2.3 Multi-Match Operations

Pattern: Date pattern with multiple matches

| Operation | stdlib | regengo | regengo (reuse) | Speedup |
|-----------|--------|---------|-----------------|---------|
| FindAllString (3 matches) | 697 ns | 142 ns | 70 ns | **4.9x - 10x** |
| Memory | 627 B/op | 248 B/op | 0 B/op | **60-100% less** |
| Allocations | 7 | 6 | 0 | **14-100% less** |

### 2.4 Greedy vs Lazy Quantifiers

| Pattern Type | stdlib | regengo | Speedup |
|--------------|--------|---------|---------|
| Greedy (`.*`) | 734 ns | 467 ns | **1.6x** |
| Lazy (`.*?`) | 1237 ns | 456 ns | **2.7x** |

### 2.5 Pathological Pattern (Catastrophic Backtracking Risk)

Pattern: `(a+)+b` - Classic ReDoS pattern

**Input: 30 'a' characters followed by 'c' (non-matching)**

| Engine | Time | Theoretical without protection |
|--------|------|-------------------------------|
| regengo (Thompson) | 953 ns | Would be ~2^30 operations (hours) |
| Go stdlib | 725 ns | Same protection internally |

**Input: Simple "example" string**

| Operation | stdlib | regengo | Speedup |
|-----------|--------|---------|---------|
| MatchString | 39 ns | 20 ns | **2.0x** |

### 2.6 URL Pattern (Nested Quantifiers + Captures)

Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?`

This pattern triggers Thompson NFA for Match and memoization for captures.

| Operation | stdlib | regengo | Notes |
|-----------|--------|---------|-------|
| FindString (short URL) | 177 ns | 315 ns | **0.6x** (memoization overhead) |
| FindString (long URL) | 349 ns | 1159 ns | **0.3x** (memoization overhead) |
| Memory | 160 B/op | 408-2202 B/op | Higher due to memoization map |

**Trade-off Analysis:** The URL pattern has nested quantifiers `[\w\.-]+` inside optional groups, triggering catastrophic backtracking protection. This adds overhead but guarantees polynomial time even for adversarial inputs.

---

## 3. Memory Analysis

### 3.1 Per-Operation Allocations

| Pattern | stdlib | regengo | regengo (pool) |
|---------|--------|---------|----------------|
| Date | 2 allocs | 1 alloc | 0 allocs |
| Email | 2 allocs | 1 alloc | 0 allocs |
| Multi-match | 7 allocs | 6 allocs | 0 allocs |
| URL (memoized) | 2 allocs | 4-8 allocs | 3-7 allocs |

### 3.2 Bytes Per Operation

| Pattern | stdlib | regengo | Reduction |
|---------|--------|---------|-----------|
| Date | 128 B | 64 B → 0 B | 50-100% |
| Email | 128 B | 64 B → 0 B | 50-100% |
| Multi-match | 627 B | 248 B → 0 B | 60-100% |
| URL (memoized) | 160 B | 408-2202 B | -155% to -1276% |

### 3.3 sync.Pool Effectiveness

When `sync.Pool` is enabled (default), regengo achieves **zero allocations** for repeated operations by reusing:
- Backtracking stacks
- Capture checkpoint arrays
- Result structs

---

## 4. Complexity by Engine Type

### 4.1 Standard Backtracking (Most Patterns)

```
Time:   O(n) average, O(n*m) worst case
Space:  O(d) where d = backtrack depth
        O(1) with sync.Pool amortized
```

**Used when:** Pattern has no nested quantifiers or catastrophic risk.

### 4.2 Thompson NFA (Pathological Patterns)

```
Time:   O(n*m) guaranteed
Space:  O(m) for state bitset (m ≤ 64 states)
        O(1) allocation (bitset is stack-allocated)
```

**Used when:** Pattern has nested quantifiers like `(a+)+`, `(a*)*`, `(a|b)+*`.

### 4.3 Memoized Backtracking (Complex + End Anchor)

```
Time:   O(n*m) guaranteed (polynomial)
Space:  O(n*m) for visited state map
```

**Used when:** Pattern has catastrophic risk but also has `$` anchor (Thompson can't handle end anchors yet).

---

## 5. Decision Matrix

| Pattern Characteristics | Engine Selected | Performance vs stdlib |
|------------------------|-----------------|----------------------|
| Simple literals | Backtracking | **2-5x faster** |
| Character classes | Backtracking | **2-3x faster** |
| Quantifiers (no nesting) | Backtracking | **1.5-3x faster** |
| Alternation | Backtracking | **1.5-2x faster** |
| Nested quantifiers (no $) | Thompson NFA | **Similar speed, guaranteed safe** |
| Nested quantifiers + $ | Memoized | **Slower, but safe** |
| Nested quantifiers + captures | Memoized captures | **Slower, but safe** |

---

## 6. Recommendations

### When to Use regengo

✅ **High-throughput matching** - 2-16x faster for hot paths
✅ **Memory-constrained environments** - Zero allocations with pool
✅ **Untrusted input patterns** - Thompson NFA prevents ReDoS
✅ **Known patterns at compile time** - Full optimization

### When stdlib May Be Better

⚠️ **Dynamic patterns** - regengo requires compile-time generation
⚠️ **Patterns with nested quantifiers + captures** - Memoization adds overhead
⚠️ **Very complex patterns with end anchors** - Trade-off for safety

---

## 7. Conclusion

regengo provides significant performance improvements for most regex use cases:

| Metric | Improvement |
|--------|-------------|
| **Average speedup** | 2-5x faster |
| **Best case speedup** | 16x faster (date patterns with pool) |
| **Memory reduction** | 50-100% less allocations |
| **Worst case** | ~3x slower (memoized captures) |
| **Safety guarantee** | Polynomial time for all patterns |

The implementation successfully balances performance optimization with safety guarantees against catastrophic backtracking attacks.

---

*Report generated for regengo with Thompson NFA support*
*Platform: Apple M4 Pro, darwin/arm64*
*Go version: 1.24*
