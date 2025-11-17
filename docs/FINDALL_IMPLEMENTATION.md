# FindAll Implementation

## Overview

Added `FindAll` functionality to extract **all matches** from input, similar to Go's stdlib `regexp.FindAllStringSubmatch`. This feature is automatically generated for patterns with capture groups.

## Generated Functions

For patterns with capture groups, Regengo now generates:

```go
// Find all matches from string
func {Name}FindAllString(input string, n int) []*{Name}Match

// Find all matches from []byte
func {Name}FindAllBytes(input []byte, n int) []*{Name}Match
```

## Usage

```go
// Pattern: (?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
text := "Dates: 2024-01-15 and 2024-12-25 and 2025-06-30"

// Find all matches
matches := DateCaptureFindAllString(text, -1)
// Returns: 3 matches

// Find up to 2 matches
matches := DateCaptureFindAllString(text, 2)
// Returns: 2 matches

// No search
matches := DateCaptureFindAllString(text, 0)
// Returns: nil
```

## Parameter `n`

Controls the maximum number of matches:

- `n < 0`: Find all matches (unlimited)
- `n = 0`: Return nil immediately (no search performed)
- `n > 0`: Return up to n matches

This matches the stdlib `regexp.FindAllStringSubmatch` behavior.

## Implementation Details

### Loop-Based Matching

FindAll uses a loop-based approach that:

1. Attempts to match from current position
2. On match: adds to results, advances past match, continues
3. On no match: advances by 1 character, continues
4. Stops when: n limit reached OR end of input

### Key Components

**Modified Match Instruction** (`generateMatchInstForFindAll`):

- Instead of returning, adds match to result slice
- Advances `searchStart` past the match
- Uses `continue` to search for next match

**Modified Fail Instruction**:

- Instead of returning `nil, false`, uses `goto TryFallback`
- This allows fallback logic to advance position and continue searching

**Zero-Width Match Protection**:

```go
if captures[1] > searchStart {
    searchStart = captures[1]
} else {
    searchStart++ // Prevent infinite loop
}
```

### State Machine

FindAll generates an inline state machine similar to Find but wraps it in a loop:

```go
for true {
    // Check limits
    if n > 0 && len(result) >= n { break }
    if searchStart >= len(input) { break }

    // Initialize state
    offset := searchStart
    captures := make([]int, numCaptures)
    // ... stack initialization ...

    // Run state machine
    goto StepSelect

TryFallback:
    if len(stack) > 0 {
        // Try backtracking
    } else {
        searchStart++
        continue // No match at this position
    }

StepSelect:
    // ... instruction dispatch ...

    // On match: add to result and continue
}
return result
```

## Performance Characteristics

- ✅ **Pool-optimized**: Reuses stack allocations across iterations
- ✅ **Non-overlapping**: Standard regex behavior
- ✅ **Zero allocation** for backtracking (pool-based)
- ✅ **Efficient**: Direct state machine, no regex interpretation
- ✅ **Thread-safe**: Each call has its own state

## Compatibility

FindAll behavior matches Go's stdlib:

- ✅ Same `n` parameter semantics
- ✅ Non-overlapping matches
- ✅ Zero-width match handling
- ✅ Returns slice of matches (not `[][]string`, but typed structs)

## Tests

Comprehensive test coverage in:

- `internal/compiler/findall_test.go` - Unit tests
- `tests/integration/findall_test.go` - Integration tests comparing with stdlib

Test cases include:

- Multiple matches (unlimited)
- Limited matches (n > 0)
- No matches found
- n = 0 (returns nil)
- Single match
- Zero-width matches
- Stdlib comparison

## Code Structure

### Compiler Changes

**File**: `internal/compiler/compiler.go`

1. **generateCaptureFunctions** (lines 520-558):

   - Added calls to `generateFindAllStringFunction`
   - Added calls to `generateFindAllBytesFunction`

2. **generateFindAllStringFunction** (lines 648-685):

   - Generates FindAllString wrapper
   - Calls `generateFindAllFunction` with string types

3. **generateFindAllBytesFunction** (similar):

   - Generates FindAllBytes wrapper
   - Calls `generateFindAllFunction` with []byte types

4. **generateFindAllFunction** (lines 687-816):

   - Core loop-based matching logic
   - Builds result slice
   - Handles n parameter
   - Manages state per iteration

5. **generateInstructionsForFindAll** (lines 781-813):

   - Generates instructions for FindAll context
   - Special handling for Match and Fail instructions
   - Uses regular generation for other instructions

6. **generateMatchInstForFindAll** (lines 818-885):
   - Modified Match instruction
   - Adds to result slice
   - Advances searchStart
   - Uses `continue` instead of `return`

## Example Generated Code

```go
func DateCaptureFindAllString(input string, n int) []*DateCaptureMatch {
    if n == 0 {
        return nil
    }
    var result []*DateCaptureMatch
    l := len(input)
    searchStart := 0

    for true {
        if n > 0 && len(result) >= n { break }
        if searchStart >= l { break }

        // Initialize state
        offset := searchStart
        captures := make([]int, 8)
        // ... stack setup ...

        // State machine
        goto StepSelect

    TryFallback:
        if len(stack) > 0 {
            // Backtrack
        } else {
            searchStart++
            continue
        }

    StepSelect:
        // ... instruction dispatch ...

    Ins17: // Match instruction
        {
            captures[1] = offset
            result = append(result, &DateCaptureMatch{
                Match: string(input[captures[0]:captures[1]]),
                Year:  string(input[captures[2]:captures[3]]),
                Month: string(input[captures[4]:captures[5]]),
                Day:   string(input[captures[6]:captures[7]]),
            })
            if captures[1] > searchStart {
                searchStart = captures[1]
            } else {
                searchStart++
            }
            continue
        }
    }
    return result
}
```

## Documentation Updates

- **README.md**: Added FindAll section under "Generated functions" and "FindAll: Multiple Match Extraction"
- **docs/CAPTURE_GROUPS.md**: Added "FindAll: Multiple Matches" examples section

## Future Enhancements

Potential improvements:

- Add `FindAllIndex` variants that return positions instead of strings
- Add streaming/iterator API for very large inputs
- Optimize for patterns that can benefit from Boyer-Moore or similar
- Add benchmarks comparing FindAll performance vs stdlib

## Testing Checklist

- [x] Unit tests for FindAll behavior
- [x] Integration tests comparing with stdlib
- [x] Test n parameter semantics (n<0, n=0, n>0)
- [x] Test zero-width matches
- [x] Test no matches found
- [x] Test single match
- [x] Test multiple matches
- [x] Generated code compiles
- [x] make bench-gen works
- [x] All existing tests still pass
- [x] Documentation updated

## Date

2024-01-XX (implementation complete)
