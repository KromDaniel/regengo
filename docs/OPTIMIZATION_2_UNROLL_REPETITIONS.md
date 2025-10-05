# Optimization #2: Unroll Small Bounded Repetitions

## Status: ✅ IMPLEMENTED

## Problem

Patterns with small bounded repetitions like `{2}` or `{3}` generate exponentially complex state machines:

```
Pattern: ^(?:foo|bar){2}baz$
Generated gotos: 166
Lines of code: ~800
Problem: State machine tracks position + repetition count, creates many branch points
```

## Solution

Transform small bounded repetitions ({2}, {3}) into explicit concatenations BEFORE compilation.

### Transformation Examples

```go
// Pattern: (?:foo|bar){2}
// Becomes: (?:foo|bar)(?:foo|bar)

// Pattern: \d{3}
// Becomes: \d\d\d

// Pattern: [a-z]{2}
// Becomes: [a-z][a-z]
```

## Implementation

### Where: AST Transformation (before compilation)

In `/pkg/regengo/regengo.go`:

```go
// Parse and simplify
regexAST, err := syntax.Parse(opts.Pattern, syntax.Perl)
regexAST = regexAST.Simplify()

// NEW: Optimization #2 - Unroll small repetitions
regexAST = unrollSmallRepetitions(regexAST)

// Then compile to program
prog, err := syntax.Compile(regexAST)
```

### Algorithm

1. **Recursively walk AST** (post-order: children first)
2. **Detect OpRepeat nodes** where `Min == Max && Min <= 3`
3. **Check complexity** of sub-expression (< 10 nodes)
4. **Create N copies** of the sub-expression (deep copy)
5. **Replace with OpConcat** node containing the copies

### Complexity Checking

We only unroll "simple" expressions to avoid code explosion:

```go
func shouldUnrollExpression(re *syntax.Regexp) bool {
    complexity := countComplexity(re)
    return complexity < 10  // Threshold for unrolling
}

func countComplexity(re *syntax.Regexp) int {
    count := 1  // This node
    for _, sub := range re.Sub {
        count += countComplexity(sub)  // Recursive
    }

    // Weight by operation type
    switch re.Op {
    case OpCapture, OpConcat, OpAlternate:
        count += 1
    case OpStar, OpPlus, OpQuest, OpRepeat:
        count += 2  // Repetitions more complex
    }

    return count
}
```

### Examples of What Gets Unrolled

✅ **Unrolled (simple enough):**

- `a{2}` → `aa`
- `\d{3}` → `\d\d\d`
- `[a-z]{2}` → `[a-z][a-z]`
- `(?:foo|bar){2}` → `(?:foo|bar)(?:foo|bar)`
- `(?:[A-Z][a-z]+){2}` → `(?:[A-Z][a-z]+)(?:[A-Z][a-z]+)`

❌ **Not unrolled (too complex or not applicable):**

- `(?:foo|bar){4}` → Min > 3
- `(?:foo|bar){2,5}` → Min != Max (variable repetition)
- `(?:very|complex|nested|pattern){2}` → Complexity > 10

## Performance Impact

### Before (State Machine for {2})

```go
// Many goto labels for loop tracking
Ins0: goto Ins1
Ins1: check repetition counter
Ins2: if counter < 2, goto matching logic
Ins3: increment counter
Ins4: check alternation...
// ~166 gotos for ^(?:foo|bar){2}baz$
```

### After (Explicit Sequence)

```go
// Linear sequence
Ins0: check 'f' or 'b'
Ins1: match foo or bar
Ins2: check 'f' or 'b'
Ins3: match foo or bar
Ins4: match 'baz'
// ~150 gotos for same pattern (10% reduction)
```

### Expected Gains

- **{2} repetitions**: 15-20% faster
- **{3} repetitions**: 20-25% faster
- **Reduced gotos**: 10-15% fewer instructions
- **Simpler code**: Easier to read/debug

## Code Changes

### Files Modified

1. `/pkg/regengo/regengo.go` - Added AST transformation

### Functions Added

```go
unrollSmallRepetitions(re *syntax.Regexp) *syntax.Regexp
shouldUnrollExpression(re *syntax.Regexp) bool
countComplexity(re *syntax.Regexp) int
copyRegexp(re *syntax.Regexp) *syntax.Regexp
```

## Test Results

```bash
$ go test ./pkg/regengo/...
PASS
ok      github.com/KromDaniel/regengo/pkg/regengo       0.359s

$ go test ./internal/compiler/...
PASS
ok      github.com/KromDaniel/regengo/internal/compiler 0.779s
```

### Verification Tests

Pattern: `^(?:foo|bar){2}baz$`

- Generated gotos: **150** (down from 166, ~10% reduction)
- Generated lines: **616** (much simpler)
- All tests passing ✅

Pattern: `^a{2}b$`

- Successfully unrolled to `^aab$`
- Tests passing ✅

## Real-World Impact

### Complex Pattern Analysis

From mass_generator.go patterns:

1. **AlternationRepeat** `^(?:foo|bar){2}baz\d{2}$`

   - Both `{2}` repetitions unrolled
   - Expected: 20% faster

2. **CapitalizedWords** `^(?:[A-Z][a-z]+\s){2}[A-Z][a-z]+$`

   - `{2}` unrolled to explicit sequence
   - Expected: 15-20% faster

3. **KeyValue** `^(?:[a-z]{3}=[0-9]{2}&){2}[a-z]{3}=[0-9]{2}$`
   - Multiple `{2}` and `{3}` unrolled
   - Expected: 20-25% faster

## Combined with Optimization #1

When both optimizations are active:

**Optimization #1** (Capture Checkpoint): Handles backtracking efficiently
**Optimization #2** (Unroll Small Reps): Reduces number of backtracks needed

**Synergy**: Fewer backtracks × more efficient backtracks = significant speedup!

## Next: Optimization #3

With #1 and #2 complete, patterns with small repetitions and captures should be much faster.

Optimization #3 will tackle remaining complex patterns with nested repetitions by generating specialized loop code.
