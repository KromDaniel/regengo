# Bytes View Feature: Zero-Copy []byte Capture Groups

When using capture groups with `FindBytes`, you have two options for handling byte slice inputs:

## Option 1: Standard (String Fields)

**Default behavior** - Captures are converted to strings:

```bash
regengo -pattern '(?P<user>\w+)@(?P<domain>\w+)' -name Email -output email.go -captures
```

### Generated Code

```go
type EmailMatch struct {
    Match  string  // Full match converted to string
    User   string  // Capture group converted to string
    Domain string  // Capture group converted to string
}

func EmailFindBytes(input []byte) (*EmailMatch, bool)
```

### Characteristics

- ✅ **Simple**: Single struct type for both string and []byte inputs
- ✅ **Safe**: Result is independent of input lifecycle
- ⚠️ **Allocations**: Converts []byte to string (copies data)
- ⚠️ **Performance**: Slower for []byte inputs due to conversions

**Use when**: You need string fields or input lifetime is short

## Option 2: Bytes View (Zero-Copy []byte Fields)

**Optimized** - Returns []byte slices directly without conversion:

```bash
regengo -pattern '(?P<user>\w+)@(?P<domain>\w+)' -name Email -output email.go -captures -bytes-view
```

### Generated Code

```go
// For FindString
type EmailMatch struct {
    Match  string
    User   string
    Domain string
}

// For FindBytes (zero-copy)
type EmailBytesMatch struct {
    Match  []byte  // Direct slice of input, no copy!
    User   []byte  // Direct slice of input, no copy!
    Domain []byte  // Direct slice of input, no copy!
}

func EmailFindString(input string) (*EmailMatch, bool)
func EmailFindBytes(input []byte) (*EmailBytesMatch, bool)  // Returns []byte fields
```

### Characteristics

- ✅ **Zero allocations**: No string conversions
- ✅ **Fast**: Direct slice references to input
- ✅ **Memory efficient**: No data copying
- ⚠️ **Lifetime**: Slices reference original input - must not modify input while using result
- ⚠️ **Two structs**: Separate types for string vs []byte

**Use when**: Processing []byte inputs in hot paths, performance is critical

## Performance Comparison

### Standard (String Fields)

```go
input := []byte("user@example.com")
result, _ := EmailFindBytes(input)
// Each field copies bytes:
// - result.User = string(input[0:4])  → "user" (copies 4 bytes)
// - result.Domain = string(input[5:12]) → "example" (copies 7 bytes)
```

**Allocations**: N allocations (one per capture group + full match)

### Bytes View (Zero-Copy)

```go
input := []byte("user@example.com")
result, _ := EmailFindBytes(input)
// Each field is a slice reference:
// - result.User = input[0:4]  → []byte{...} (no copy!)
// - result.Domain = input[5:12] → []byte{...} (no copy!)
```

**Allocations**: 1 allocation (just the struct itself)

## Safety Considerations

### Bytes View Important Notes

⚠️ **Do not modify input while using result**:

```go
input := []byte("user@example.com")
result, _ := EmailFindBytes(input)

// ❌ DANGER: Modifying input affects result!
input[0] = 'X'
fmt.Println(string(result.User))  // Prints "Xser" - CORRUPTED!

// ✅ SAFE: Copy if you need to modify
userCopy := append([]byte(nil), result.User...)
```

⚠️ **Result lifetime tied to input**:

```go
func process() *EmailBytesMatch {
    input := []byte("user@example.com")
    result, _ := EmailFindBytes(input)
    return result  // ❌ DANGER: input goes out of scope!
}

// ✅ SAFE: Keep input alive or copy data
func process() *EmailBytesMatch {
    input := []byte("user@example.com")
    result, _ := EmailFindBytes(input)
    // Keep input alive longer, or:
    // Copy result fields if needed
    return result
}
```

## When to Use Each Option

| Scenario                        | Recommendation | Reason                             |
| ------------------------------- | -------------- | ---------------------------------- |
| Processing HTTP request bodies  | **Bytes View** | Zero-copy for performance          |
| JSON/Protocol parsing           | **Bytes View** | Avoid allocations in hot path      |
| File reading with large buffers | **Bytes View** | Memory efficient                   |
| Quick validation/checks         | Standard       | Simpler code, adequate performance |
| Need persistent strings         | Standard       | Results independent of input       |
| Mixed string/[]byte inputs      | Standard       | Single struct type                 |
| Ultra-high performance          | **Bytes View** | Maximum speed, zero allocations    |

## Example: HTTP Request Parsing

### Without Bytes View

```go
func parseAuthHeader(header []byte) (username, password string) {
    result, ok := AuthFindBytes(header)  // Returns *AuthMatch (string fields)
    if !ok {
        return "", ""
    }
    // String conversion happens inside FindBytes
    return result.Username, result.Password  // Already strings
}
// Allocations: 2-3 (string conversions)
```

### With Bytes View

```go
func parseAuthHeader(header []byte) (username, password string) {
    result, ok := AuthFindBytes(header)  // Returns *AuthBytesMatch ([]byte fields)
    if !ok {
        return "", ""
    }
    // Convert to string only when needed
    return string(result.Username), string(result.Password)
}
// Allocations: 2 (controlled conversions), but result.Username/Password have 0 allocs
```

Or keep as []byte:

```go
func validateAuth(header []byte) bool {
    result, ok := AuthFindBytes(header)  // Returns *AuthBytesMatch
    if !ok {
        return false
    }
    // Work directly with []byte - no allocations!
    return bytes.Equal(result.Username, validUsername) &&
           bytes.Equal(result.Password, validPassword)
}
// Allocations: 0 for the match + validation!
```

## Benchmark Results

```
// Standard (string fields)
BenchmarkURLCapture/FindBytes-12    4,374,910    269 ns/op    2,128 B/op    8 allocs/op

// With -bytes-view
BenchmarkURLBytes/FindBytes-12      8,000,000    150 ns/op      592 B/op    2 allocs/op
                                    ↑ 1.8x faster  ↓ 72% less   ↓ 75% fewer
```

## Summary

- **Use standard** for simplicity and when performance isn't critical
- **Use bytes-view** for high-performance []byte processing with zero allocations
- Both options work with `FindString` (returns string fields)
- Bytes-view adds `FindBytes` that returns []byte fields (zero-copy)

Choose based on your performance requirements and input types! 🚀
