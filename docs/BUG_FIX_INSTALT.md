# Bug Fix: InstAlt Bounds Checking

## Issue Description

**Date**: December 2024  
**Severity**: Critical - Incorrect Match Results  
**Affected Patterns**: Any pattern with alternation at the end (e.g., email validation)

### Symptoms

The email pattern `[\w\.+-]+@[\w\.-]+\.[\w\.-]+` incorrectly returned `false` for input `"a@b.c"` when it should return `true`.

```go
// Expected: true
// Actual: false
generated.EmailMatchString("a@b.c")
```

### Root Cause

The generated code was performing bounds checking **before** executing `InstAlt` (alternation) instructions. This prevented the instruction from pushing alternatives onto the backtracking stack when at the end of input.

#### Problematic Flow

1. At offset=5 (EOF), the matcher reached instruction `Ins8` (InstAlt with alternatives: continue to `Ins5`, or jump to `Ins9` MATCH)
2. The bounds check `if offset >= len(input)` triggered **before** saving the MATCH alternative
3. The goto statement jumped to the backtracking logic
4. Since the MATCH alternative was never saved to the stack, it was never tried
5. Result: false negative

#### Why This Is Wrong

`InstAlt` is a **control flow instruction** that doesn't consume input. It only:

- Pushes an alternative instruction pointer onto the backtracking stack
- Continues execution to the next instruction

Since it doesn't consume input, it should be allowed to execute even when `offset >= len(input)`.

## Solution

### Code Change

Modified `internal/compiler/compiler.go` in the `generateInstructions()` method:

```go
// Before (lines ~157-160)
if inst.Op != syntax.InstMatch {
    f.If(jen.Id(codegen.OffsetName).Op(">=").Len(jen.Id(codegen.InputName))).Block(
        jen.Goto().Id(codegen.BacktrackLabel),
    )
}

// After
if inst.Op != syntax.InstMatch &&
   inst.Op != syntax.InstAlt &&
   inst.Op != syntax.InstEmptyWidth {
    f.If(jen.Id(codegen.OffsetName).Op(">=").Len(jen.Id(codegen.InputName))).Block(
        jen.Goto().Id(codegen.BacktrackLabel),
    )
}
```

### Rationale

**Instructions excluded from bounds checking**:

- `InstMatch`: Terminal instruction, always succeeds
- `InstAlt`: Saves alternatives, doesn't consume input
- `InstEmptyWidth`: Zero-width assertions (e.g., `^`, `$`, `\b`), don't consume input

**Instructions that need bounds checking**:

- `InstRune`, `InstRune1`: Consume a single character
- `InstRuneAny`, `InstRuneAnyNotNL`: Consume a single character
  All other consuming instructions

## Validation

### Test Results

Created comprehensive test suite comparing regengo output against Go's standard `regexp`:

```
Pattern: [\w\.+-]+@[\w\.-]+\.[\w\.-]+
✓ "test@example.com" -> true
✓ "a@b.c" -> true (previously failed)
✓ "user.name+tag@domain.co.uk" -> true
✓ "invalid" -> false
✓ "@example.com" -> false
... (33 total test cases)

Result: All tests passed!
```

### Performance Impact

No performance regression. In fact, benchmarks show regengo is faster than standard regexp:

```
BenchmarkEmailStdRegexp-12    1338194    865.4 ns/op    0 B/op    0 allocs/op
BenchmarkEmailRegengo-12      2027746    585.4 ns/op    2352 B/op 23 allocs/op

BenchmarkURLStdRegexp-12      2144304    554.3 ns/op    0 B/op    0 allocs/op
BenchmarkURLRegengo-12        3036224    392.0 ns/op    2544 B/op 13 allocs/op

BenchmarkIPv4StdRegexp-12     2505442    481.2 ns/op    0 B/op    0 allocs/op
BenchmarkIPv4Regengo-12       4577810    252.6 ns/op    768 B/op  14 allocs/op
```

- Email: **1.48x faster**
- URL: **1.41x faster**
- IPv4: **1.90x faster**

## Lessons Learned

1. **Instruction Semantics Matter**: Not all instructions consume input. Control flow and zero-width assertions must be treated specially.

2. **Test Edge Cases**: Short inputs at the minimal length for a pattern (like `"a@b.c"`) are critical test cases.

3. **Backtracking Semantics**: The backtracking stack must be populated **before** checking if input is available. The alternative exists regardless of whether we can continue on the current path.

4. **RE2 Compliance**: Any deviation from standard regex behavior is a bug. Always validate against the standard library.

## Related Files

- `internal/compiler/compiler.go` - Fix applied here
- `examples/generated/Email.go` - Regenerated with fix
- `test/benchmarks/benchmark_test.go` - Performance validation
- Test scripts (removed): `test_comparison.go`, `debug_check.go`, `trace_debug.go`

## References

- Go `regexp/syntax` package documentation
- RE2 instruction set semantics
- Backtracking algorithm design patterns
