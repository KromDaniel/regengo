# Complex Pattern Performance Analysis

## Findings Summary

After analyzing the 35 complex patterns where stdlib wins, here are the KEY issues:

### 1. **Capture Group Overhead** (CRITICAL)

**Problem:** On every backtracking attempt, we reset ALL capture positions:

```go
for i := range captures {
    captures[i] = 0
}
```

**Impact:**

- Pattern: `^(\w+)@(\w+)\.(\w{2,})$` (3 captures = 6 ints)
- Each backtrack attempt loops through 6 integers
- Patterns with 4+ capture groups are significantly slower
- Stdlib uses copy-on-write or checkpoint-based capture tracking

**Solution:** Implement capture stack/checkpoint system instead of reset-all

### 2. **Nested Repetition Complexity**

**Problem:** Patterns like `^(?:[A-Z][a-z]+\s){2}[A-Z][a-z]+$` generate exponential gotos

**Characteristics:**

- Repetition inside repetition: `(...)+` contains `...+`
- Each level multiplies the state machine complexity
- Generated code has 98+ goto statements for simple nested repeat

**Solution:** Detect nested repetitions and generate specialized loop code instead of state machine

### 3. **Alternation in Loops**

**Problem:** `^(?:foo|bar){2}$` requires stack management per iteration

**Impact:**

- Each alternation adds 2 InstAlt instructions
- Each repetition multiplies the branches
- Stack grows/shrinks on each iteration
- 166 goto statements for pattern with 2 repetitions

**Solution:** Unroll small repetitions (`{2}` → explicit sequence) or use specialized loop handler

## Performance Comparison

### Simple Pattern (No Captures)

```
Pattern: ^[a-z]+@[a-z]+\.[a-z]{2,}$
Regengo: 95% FASTER ✅
Reason: Our optimized character class checks win
```

### Complex Pattern (With Captures)

```
Pattern: ^(\w+)@(\w+)\.(\w{2,})$
Regengo: 10-30% SLOWER ⚠️
Reason: Capture reset overhead on backtracking
```

### Very Complex (Nested + Captures)

```
Pattern: ^(?P<area>\d{3})-(?P<prefix>\d{3})-(?P<line>\d{4})(?: x(?P<ext>\d{2}))?$
Regengo: 20-40% SLOWER ⚠️
Reason: Many captures (8) + optional group = reset loop on every backtrack
```

## Code Size Impact

Generated code analysis:

- **Simple pattern** (`^[a-z]+$`): ~200 lines, 30 gotos
- **With captures** (`^(\w+)$`): ~1460 lines, 318 gotos
- **Nested repetition** (`^(?:...){2}$`): ~800 lines, 166 gotos

## Recommended Optimizations (Priority Order)

### 1. **Capture Checkpoint System** (HIGH IMPACT)

Replace `for i := range captures { captures[i] = 0 }` with:

```go
captureStack := [][]int{} // Stack of capture states
// On backtrack: restore from captureStack instead of resetting
```

**Expected gain:** 20-40% faster for patterns with 3+ capture groups

### 2. **Unroll Small Repetitions** (MEDIUM IMPACT)

Transform `{2}` and `{3}` into explicit sequences:

```go
// Instead of: ^(?:foo|bar){2}$
// Generate:   ^(?:foo|bar)(?:foo|bar)$
```

**Expected gain:** 15-25% faster for patterns with small bounded repetitions

### 3. **Specialized Loop Code** (MEDIUM IMPACT)

For simple repeating groups without captures:

```go
// Pattern: ^(?:[A-Z][a-z]+\s){2}$
// Generate loop instead of state machine:
for count := 0; count < 2; count++ {
    if !matchUpperLowerPlusSpace() { return false }
}
```

**Expected gain:** 10-20% faster for nested repetitions

### 4. **Lazy Capture Allocation** (LOW IMPACT)

Only allocate captures array if actually needed:

```go
var captures []int // Don't allocate upfront
if needsCaptures {
    captures = make([]int, captureCount)
}
```

**Expected gain:** 5-10% faster for MatchString (no captures needed)

## Test Cases for Next Optimization Round

1. **Phone Number:** `^(?P<area>\d{3})-(?P<prefix>\d{3})-(?P<line>\d{4})$`
2. **Email with Captures:** `^(\w+)@(\w+)\.(\w{2,})$`
3. **Repeated Alternation:** `^(?:foo|bar){2,4}$`
4. **Nested Repetition:** `^(?:[A-Z][a-z]+\s){2,}[A-Z][a-z]+$`
5. **Many Captures:** `^(\d+)-(\d+)-(\d+)-(\d+)$`

## Conclusion

Our optimizations are **extremely effective for simple patterns** (95% faster!), but patterns with **capture groups** and **nested structures** suffer from:

1. Naive capture reset strategy (O(n) per backtrack)
2. Over-generation of goto statements for nested structures
3. No specialized handling for common complex pattern types

**Next step:** Implement capture checkpoint system - will likely bring us to parity or ahead of stdlib even for complex patterns.
