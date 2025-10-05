# Performance Benchmark Comparison

## Visual Comparison

### Email Pattern: `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`

```
Standard regexp:    ████████████████████████████████████████ 882 ns/op
Regengo (no pool):  ██████████████████████ 585 ns/op (1.5x faster)
Regengo (pooled):   ██████████ 289 ns/op (3.1x faster) ⚡️
```

**Memory Allocations**:
```
Standard regexp:    ✓ 0 allocs/op
Regengo (no pool):  ✗ 23 allocs/op, 2352 B/op
Regengo (pooled):   ✓ 0 allocs/op, 0 B/op ⚡️
```

---

### URL Pattern: `https?://[^\s]+`

```
Standard regexp:    ████████████████████████████████ 549 ns/op
Regengo (no pool):  ████████████████████ 373 ns/op (1.5x faster)
Regengo (pooled):   █████ 121 ns/op (4.5x faster) ⚡️⚡️
```

**Memory Allocations**:
```
Standard regexp:    ✓ 0 allocs/op
Regengo (no pool):  ✗ 13 allocs/op, 2544 B/op
Regengo (pooled):   ✓ 0 allocs/op, 0 B/op ⚡️
```

---

### IPv4 Pattern: `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`

```
Standard regexp:    ███████████████████████████ 470 ns/op
Regengo (no pool):  ████████████ 237 ns/op (2.0x faster)
Regengo (pooled):   ████ 96 ns/op (4.9x faster) ⚡️⚡️⚡️
```

**Memory Allocations**:
```
Standard regexp:    ✓ 0 allocs/op
Regengo (no pool):  ✗ 14 allocs/op, 768 B/op
Regengo (pooled):   ✓ 0 allocs/op, 0 B/op ⚡️
```

---

## Summary Table

| Metric | Standard regexp | Regengo (no pool) | Regengo (pooled) |
|--------|----------------|-------------------|------------------|
| **Speed** | Baseline | 1.5-2.0x faster | **3.1-4.9x faster** |
| **Memory** | 0 allocs | 768-2352 B | **0 allocs** |
| **GC Pressure** | None | High | **None** |
| **Thread Safety** | ✓ | ✓ | ✓ |
| **Complexity** | Low | Low | Low |

## Recommendation

🎯 **Use Regengo with `-pool` for production**

The pooled version delivers:
- ✅ Up to 4.9x faster than standard regexp
- ✅ Zero heap allocations
- ✅ No GC pressure
- ✅ Thread-safe concurrent access
- ✅ Simple API (just add `-pool` flag)

## Test It Yourself

```bash
# Clone the repository
git clone https://github.com/KromDaniel/regengo
cd regengo

# Run benchmarks
make bench

# Or run specific benchmark
go test -bench=BenchmarkEmail -benchmem ./test/benchmarks/
```

---

**Platform**: Apple M4 Pro (arm64)  
**Go Version**: 1.21+  
**Date**: October 2025
