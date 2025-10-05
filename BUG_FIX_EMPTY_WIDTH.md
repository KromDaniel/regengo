# Bug Fix: Empty Width Assertions (^ and $)

## Problem Summary

The mass generator was failing **151 out of 155 test packages**. All failures were related to patterns with end-of-string anchors (`$`) and some with beginning-of-string anchors (`^`).

### Failing Patterns Examples:

- `^[a-z]{3}$` - Should reject "aaaa" (4 chars) but was matching it
- `^\d{4}$` - Should reject "12345" (5 digits) but was matching it
- `^(?:foo|bar){2}baz\d{2}$` - Should reject "foofoobaz77extra" but was matching it

## Root Cause

In `/Users/dkrom/Dev/regengo/internal/compiler/compiler.go`, the `generateEmptyWidthInst()` function was not implementing position assertions. It simply skipped over them:

```go
func (c *Compiler) generateEmptyWidthInst(label *jen.Statement, inst *syntax.Inst) ([]jen.Code, error) {
	// For now, simple implementation - just continue
	return []jen.Code{
		label,
		jen.Block(
			jen.Goto().Id(codegen.InstructionName(inst.Out)),
		),
	}, nil
}
```

This meant that:

- `^` (BeginText) was not checking if `offset == 0`
- `$` (EndText) was not checking if `offset == length`
- `\n` line boundaries were not being checked

## Solution

Implemented proper empty width assertion checks in `generateEmptyWidthInst()`:

1. **BeginText (`^`)**: Check that `offset == 0`
2. **EndText (`$`)**: Check that `offset == l` (end of input)
3. **BeginLine**: Check that `offset == 0` OR previous char is `\n`
4. **EndLine**: Check that `offset == l` OR current char is `\n`

The function now examines the `EmptyOp` flags in `inst.Arg` and generates appropriate checks.

## Testing

### Before Fix:

- **151 test packages FAILED**
- All failures were "matches when it shouldn't" due to missing end-anchor checks

### After Fix:

- **155/155 test packages PASS** ✅
- All tests complete in ~50 seconds

### Example Generated Code (for `^[a-z]{3}$`):

```go
Ins1:
	{
		if offset != 0 {  // Check ^ anchor
			goto TryFallback
		}
		goto Ins2
	}
// ... [a-z] matching ...
Ins5:
	{
		if offset != l {  // Check $ anchor
			goto TryFallback
		}
		goto Ins6
	}
Ins6:
	{
		return true
	}
```

## Files Modified

- `/Users/dkrom/Dev/regengo/internal/compiler/compiler.go` - Fixed `generateEmptyWidthInst()` function
- `/Users/dkrom/Dev/regengo/mass_generator.go` - Added flag to preserve test directory on failure and skip benchmarks during debugging

## Future Work

Word boundary assertions (`\b`, `\B`) are not yet implemented and will return an error if encountered.
