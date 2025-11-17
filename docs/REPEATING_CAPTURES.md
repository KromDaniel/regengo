# Repeating Capture Groups Support

## Overview

Regengo now automatically detects and documents repeating capture groups in regex patterns. This feature helps users understand the behavior of patterns like `(\w)+` or `(\d)*` where the capture group itself is in a repeating context.

## Standard Regex Behavior

Go's `regexp` package (like all standard regex engines) captures **only the LAST match** from repeating groups:

```go
pattern := `(\w)+`
input := "abc"

// The pattern matches "a", "b", "c" sequentially
// But group 1 captures only: "c" (the last character)
// Result: ["abc", "c"]  (full match, then last capture)
```

This is standard POSIX regex behavior, **not a regengo limitation**.

## Detection

Regengo detects the following repeating contexts at compile time:

### True Repeating Groups

- `(\w)*` - Zero or more (captures last)
- `(\w)+` - One or more (captures last)
- `(\w){3}` - Fixed repeat (captures last)
- `(\w){2,5}` - Range repeat (captures last)

### Optional Groups

- `(\w)?` - Optional (captures empty string when not matched)
- `(?::(?P<port>\d+))?` - Capture in optional group

### Non-Repeating (No Warning)

- `(\w+)` - Capture wraps repeating content (captures all)
- `(\w)(\d)` - Normal captures

## Generated Warnings

When repeating captures are detected, regengo adds informative comments:

```go
// Note: This pattern contains capture groups in repeating/optional context.
// Go's regex engine captures only the LAST match from repeating groups (* + {n,m}).
// For example: (\w)+ matching 'abc' captures 'c', not ['a','b','c'].
// Optional groups (?) return empty string when not matched.

type EmailMatch struct {
    Match  string // Full match
    Group1 string // Only last character from (\w)+
    Group2 string // Complete match from (\w+)
}
```

## Examples

### Example 1: Repeating Group

```go
// Pattern: (\w)+@(\w+)\.com
// Input: "abc@example.com"

// Stdlib behavior:
matches := regexp.MustCompile(`(\w)+@(\w+)\.com`).FindStringSubmatch("abc@example.com")
// Result: ["abc@example.com", "c", "example"]
//         full match          ↑ last char only

// Regengo generates:
type EmailMatch struct {
    Match  string // "abc@example.com"
    Group1 string // "c" (last char from (\w)+)
    Group2 string // "example" (all chars from (\w+))
}
```

### Example 2: Optional Group

```go
// Pattern: (?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?
// Input: "https://example.com"

// The port group is optional and not matched:
result, found := URLCaptureFindString("https://example.com")
// result.Port == "" (empty string when optional group not matched)
```

### Example 3: Non-Repeating

```go
// Pattern: (\w+)@(\w+)\.com
// Input: "abc@example.com"

// No warning - these are normal captures:
type EmailMatch struct {
    Match  string // "abc@example.com"
    Group1 string // "abc" (all chars)
    Group2 string // "example" (all chars)
}
```

## Testing

Comprehensive tests verify the detection logic:

```bash
# Run repeating capture tests
go test -v ./internal/compiler -run TestRepeating

# All tests include:
# - Detection of repeating contexts
# - Stdlib behavior verification
# - Edge cases (nested, mixed, optional)
```

## Implementation Details

### Detection Algorithm

The `hasRepeatingCaptures()` function walks the regex AST:

1. Track when entering repeating contexts (OpStar, OpPlus, OpQuest, OpRepeat)
2. If a capture (OpCapture) is found in a repeating context, flag it
3. Recursively check all sub-expressions

### AST Structure

```
Pattern: (?::(?P<port>\d+))?
Op: OpQuest (optional)
  Op: OpConcat
    Op: OpLiteral (:)
    Op: OpCapture (port) ← Inside Quest, so flagged
      Op: OpPlus
        Op: OpCharClass
```

### Code Generation

When `hasRepeatingCaptures` is true:

1. Add informative comment block before struct
2. Document that:
   - Repeating groups capture last match
   - Optional groups return empty when not matched
3. Generate same struct fields (no API changes)

## Recommendations

### When to Use Repeating Captures

✅ **Good uses:**

- Optional groups: `(?P<port>\d+)?`
- Last occurrence: `(\w)+` to get last word char
- Count-limited: `(\d){4}` for last of 4 digits

❌ **Avoid when:**

- You need all matches (use `FindAll` methods instead)
- You need intermediate matches
- Pattern should be `(\w+)` not `(\w)+`

### Rewriting Patterns

Instead of:

```go
(\w)+@(\w)+\.com  // Captures last chars only
```

Use:

```go
(\w+)@(\w+)\.com  // Captures all chars
```

## References

- [Go regexp documentation](https://pkg.go.dev/regexp)
- [POSIX regex standards](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap09.html)
- [Repeating captures demo](../examples/repeating_captures_demo.go)

## Related Files

- `internal/compiler/compiler.go` - Detection and generation logic
- `internal/compiler/repeating_test.go` - Comprehensive test suite
- `examples/repeating_captures_demo.go` - Working demonstration
- `README.md` - User documentation
