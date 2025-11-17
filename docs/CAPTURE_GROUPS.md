# Capture Groups

Regengo supports capture groups for extracting submatches from regex patterns. This feature generates optimized structs with type-safe fields for each captured group.

## Features

- ✅ **Named capture groups**: `(?P<name>...)` → struct field with capitalized name
- ✅ **Indexed capture groups**: `(...)` → struct fields `Match1`, `Match2`, etc.
- ✅ **Optional groups**: Return empty string when group doesn't match
- ✅ **Full match tracking**: Includes `Match`, `Start`, and `End` fields
- ✅ **Type-safe**: Compile-time checked struct fields

## Usage

### CLI

```bash
regengo -pattern '(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)' \
        -name 'Email' \
        -output 'email.go' \
        -package 'main' \
        -captures
```

### Library

```go
err := regengo.Compile(regengo.Options{
    Pattern:      `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
    Name:         "Email",
    OutputFile:   "email.go",
    Package:      "main",
    WithCaptures: true,
})
```

## Generated Code

For the pattern `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`, Regengo generates:

```go
// EmailMatch holds the result of a successful match with capture groups.
type EmailMatch struct {
    Match  string // The full matched string
    Start  int    // Start position in input
    End    int    // End position in input
    User   string // Capture group 1
    Domain string // Capture group 2
    Tld    string // Capture group 3
}

func EmailFindString(input string) (*EmailMatch, bool) {
    // ... optimized matching code ...
}

func EmailFindBytes(input []byte) (*EmailMatch, bool) {
    // ... optimized matching code ...
}

func EmailFindAllString(input string, n int) []*EmailMatch {
    // ... find all matches ...
}

func EmailFindAllBytes(input []byte, n int) []*EmailMatch {
    // ... find all matches from bytes ...
}
```

## Examples

### FindAll: Multiple Matches

```go
// Pattern: (?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
text := "Dates: 2024-01-15 and 2024-12-25 and 2025-06-30"

// Find all matches (n = -1 means unlimited)
matches := DateCaptureFindAllString(text, -1)
for _, m := range matches {
    fmt.Printf("%s: Year=%s, Month=%s, Day=%s\n", m.Match, m.Year, m.Month, m.Day)
}
// Output:
// 2024-01-15: Year=2024, Month=01, Day=15
// 2024-12-25: Year=2024, Month=12, Day=25
// 2025-06-30: Year=2025, Month=06, Day=30

// Find up to 2 matches
matches := DateCaptureFindAllString(text, 2)
// Returns first 2 matches only

// Find no matches
matches := DateCaptureFindAllString(text, 0)
// Returns nil immediately
```

**Parameter `n` controls max matches**:

- `n < 0`: Find all matches (unlimited)
- `n = 0`: Return nil immediately (no search)
- `n > 0`: Return up to n matches

**Behavior**:

- ✅ Returns slice of match pointers (`[]*EmailMatch`)
- ✅ Non-overlapping matches (standard regex behavior)
- ✅ Advances past each match to find the next
- ✅ Handles zero-width matches (advances by 1 to prevent infinite loops)
- ✅ Compatible with stdlib `regexp.FindAllStringSubmatch` semantics

### Named Groups

```go
result, found := EmailFindString("user@example.com")
if found {
    fmt.Printf("User: %s\n", result.User)       // "user"
    fmt.Printf("Domain: %s\n", result.Domain)   // "example"
    fmt.Printf("TLD: %s\n", result.Tld)         // "com"
    fmt.Printf("Full: %s\n", result.Match)      // "user@example.com"
}
```

### Indexed Groups (Unnamed)

Pattern: `(\w+)@(\w+)\.(\w+)`

```go
result, found := EmailFindString("user@example.com")
if found {
    fmt.Printf("Group 1: %s\n", result.Match1)  // "user"
    fmt.Printf("Group 2: %s\n", result.Match2)  // "example"
    fmt.Printf("Group 3: %s\n", result.Match3)  // "com"
}
```

### Optional Groups

Pattern: `(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?`

```go
// With port
result, found := URLFindString("http://example.com:8080")
// result.Protocol = "http"
// result.Host = "example.com"
// result.Port = "8080"

// Without port
result, found := URLFindString("http://example.com")
// result.Protocol = "http"
// result.Host = "example.com"
// result.Port = ""  // Empty string for unmatched optional group
```

## Performance

Capture groups in Regengo are significantly faster than the standard library:

| Pattern      | Regengo    | Standard regexp | Speedup         |
| ------------ | ---------- | --------------- | --------------- |
| DateCapture  | 20.7 ns/op | 101 ns/op       | **5x faster**   |
| EmailCapture | 123 ns/op  | 246 ns/op       | **2x faster**   |
| URLCapture   | 118 ns/op  | 196 ns/op       | **1.7x faster** |

### Why Faster?

1. **Direct struct allocation**: No slice allocation for submatches
2. **Inline string conversion**: Efficient byte-to-string conversion
3. **Optimized bounds checking**: Smart handling of optional groups
4. **Zero regex interpretation**: All logic compiled to native Go

## Limitations

- Backreferences are not supported (e.g., `\1`, `\2`)
- Nested groups may have complex numbering (use named groups for clarity)
- Very large numbers of capture groups may impact performance

## Best Practices

1. **Use named groups** for clarity and maintainability
2. **Check for empty strings** when using optional groups
3. **Combine with `-pool`** flag for maximum performance (note: pool optimization applies to stack management, not struct allocation)
4. **Prefer specific patterns** over overly generic ones

## See Also

- [Examples](../examples/captures/) - Working examples with capture groups
- [Memory Optimization](./MEMORY_OPTIMIZATION.md) - Pool optimization details
- [Benchmarks](../benchmarks/) - Performance comparison code
