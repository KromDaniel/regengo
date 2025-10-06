# Regengo Optimizations

This document describes the optimizations implemented in regengo to improve performance.

## Summary of Optimizations

Based on benchmark analysis showing regengo was 14.8% slower than stdlib for simple patterns, we implemented several key optimizations:

### 1. Conditional Backtracking Stack (✅ Implemented)

**Problem**: Every generated function initialized a backtracking stack, even for patterns without alternations.

**Solution**:

- Added `needsBacktracking` detection that scans for `InstAlt` instructions
- Skip stack initialization when not needed
- Skip stack pool generation when not needed
- Generate simpler fallback code for non-backtracking patterns

**Impact**:

- Eliminates unnecessary ~32 int array allocation for simple patterns
- Removes stack manipulation overhead
- Reduces generated code size

**Example**:

```go
// Before: Pattern "^\d{3}$" still had stack initialization
var simpleStackPool = sync.Pool{...}
stack := simpleStackPool.Get()...

// After: No stack at all
func SimpleMatchString(input string) bool {
    l := len(input)
    offset := 0
    // ... direct matching logic
}
```

### 2. Optimized Character Class Matching (✅ Implemented)

**Problem**: Character classes like `\w`, `\d`, `\s` generated long chains of OR conditions.

**Solution**:

- Detect common character classes: `\w`, `\d`, `\s`, `[a-z]`, `[A-Z]`, `[a-zA-Z]`
- Use optimized range checks for these patterns
- Reduce code complexity and improve branch prediction

**Impact**:

- Faster character class matching
- Shorter generated code
- Better CPU cache utilization

**Example**:

```go
// Before: \w check (long OR chain)
if (input[offset] < 0x30 || input[offset] > 0x39) &&
   (input[offset] < 0x41 || input[offset] > 0x5a) &&
   input[offset] != 0x5f &&
   (input[offset] < 0x61 || input[offset] > 0x7a) {
    goto TryFallback
}

// After: Optimized \w check
if input[offset] < 0x30 ||
   (input[offset] > 0x39 && input[offset] < 0x41) ||
   (input[offset] > 0x5a && input[offset] < 0x5f) ||
   (input[offset] > 0x5f && input[offset] < 0x61) ||
   input[offset] > 0x7a {
    goto TryFallback
}
```

### 3. Simplified Alternation Stack (✅ Implemented)

**Problem**: Stack push operations used complex capacity checking with if/else branches.

**Solution**:

- Simplified to use `append()` directly
- Go's runtime handles capacity growth efficiently
- Eliminates branching overhead

**Impact**:

- Faster stack operations
- Simpler generated code
- Better compiler optimization

**Example**:

```go
// Before: Complex capacity check
if cap(stack) > len(stack) {
    stack = stack[:len(stack)+1]
    stack[len(stack)-1][0] = offset
    stack[len(stack)-1][1] = nextInst
} else {
    stack = append(stack, [2]int{offset, nextInst})
}

// After: Simple append
stack = append(stack, [2]int{offset, nextInst})
```

### 4. Small Set Optimization (✅ Implemented)

**Problem**: Character sets with few values still used range checks.

**Solution**:

- For sets with 3 or fewer distinct characters, use direct equality checks
- More efficient for CPU branch prediction

**Example**:

```go
// Pattern: [abc]
// Optimized to: input[offset] != 'a' && input[offset] != 'b' && input[offset] != 'c'
```

### 5. Large Character Class Grouping (✅ Implemented)

**Problem**: Very large character classes generated extremely long condition chains.

**Solution**:

- Group ranges more efficiently
- Reduce condition length while maintaining correctness

**Impact**:

- Shorter generated code
- Faster compilation
- Better readability

## Performance Results

### Before Optimizations (Baseline)

```
Category: simple
  Regengo faster:  29  |  Stdlib faster:  61
  Avg time:       15620 ns/op (regengo) vs 13604 ns/op (stdlib) [14.8% slower]
  Avg memory:     2337 B/op (regengo) vs 1898 B/op (stdlib) [23.2% more]

Category: complex
  Regengo faster:  23  |  Stdlib faster:  37
  Avg time:       14084 ns/op (regengo) vs 14896 ns/op (stdlib) [5.5% faster]

Category: very_complex
  Regengo faster:  21  |  Stdlib faster:  14
  Avg time:       21383 ns/op (regengo) vs 25674 ns/op (stdlib) [16.7% faster]

Overall: 0.6% faster, 20.2% more memory
```

### After Optimizations (Expected)

_Run `go run benchmarks/mass_generator.go` to measure improvements_

Expected improvements:

- Simple patterns: 10-15% faster (eliminate stack overhead)
- Character class patterns: 5-10% faster (optimized checks)
- Memory usage: 15-20% reduction (no stack for simple patterns)
- Overall: 5-10% faster across all categories

## Technical Details

### Detection Functions

**detectCharacterClass**: Identifies common patterns by checking rune ranges
**allSingleChars**: Checks if all ranges are single characters
**needsBacktracking**: Scans program for `InstAlt` instructions

### Code Generation Changes

**generateRuneCheck**: Main dispatcher for character class optimization
**generateOptimizedCharClassCheck**: Generates optimized code for detected classes
**generateSmallSetCheck**: Handles small character sets
**generateLargeCharClassCheck**: Handles large character classes

### Modified Functions

- `New()`: Added backtracking detection
- `Generate()`: Conditional pool generation
- `generateMatchFunction()`: Conditional stack initialization
- `generateFindFunction()`: Conditional stack initialization
- `generateFindAllFunction()`: Conditional stack initialization
- `generateAltInst()`: Simplified stack push

## Future Optimization Opportunities

### Not Yet Implemented

1. **Loop Unrolling**: For patterns like `\d{3}`, generate 3 inline checks instead of a loop
2. **Bounds Check Elimination**: Combine multiple bounds checks into one
3. **Jump Table Optimization**: For large switch statements, consider jump tables
4. **SIMD Instructions**: Use SIMD for parallel byte matching in long strings
5. **Literal String Optimization**: For patterns with literal prefixes, use optimized string search
6. **Capture Array Pooling**: Reuse capture arrays across FindAll iterations

## Testing

Run the comprehensive test suite:

```bash
# Unit tests
go test ./internal/compiler/... ./pkg/regengo/...

# Integration tests
go test ./tests/integration/...

# Benchmark suite
go run benchmarks/mass_generator.go

# Quick pattern tests
go run cmd/regengo/main.go -name Test -pattern '\w+' -output /tmp/test.go
```

## Validation

All optimizations maintain:

- ✅ Correctness: All test cases pass
- ✅ Compatibility: No API changes
- ✅ Coverage: Optimizations apply to all code paths
- ✅ Safety: No undefined behavior introduced
