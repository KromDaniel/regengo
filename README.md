# Regengo

[![Go Reference](https://pkg.go.dev/badge/github.com/KromDaniel/regengo.svg)](https://pkg.go.dev/github.com/KromDaniel/regengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/KromDaniel/regengo)](https://goreportcard.com/report/github.com/KromDaniel/regengo)
[![codecov](https://codecov.io/github/KromDaniel/regengo/branch/main/graph/badge.svg?token=CHGHDKQ0XX)](https://codecov.io/github/KromDaniel/regengo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<p align="center">
  <img src="assets/logo.png" alt="Regengo - Go Gopher with Regex" width="400">
</p>

Regengo is a **compile-time finite state machine generator** for regular expressions. It converts regex patterns into optimized Go code, leveraging the Go compiler's optimizations to eliminate runtime interpretation overhead.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Performance](#performance)
- [Smart Analysis](#smart-analysis)
- [Complexity Guarantees](#complexity-guarantees)
- [Advanced Options](#advanced-options)
- [Generated Output](#generated-output)
- [Generated Tests & Benchmarks](#generated-tests--benchmarks)
- [Capture Groups](#capture-groups)
- [API Comparison](#api-comparison)
- [CLI Reference](#cli-reference)
- [Detailed Benchmarks](#detailed-benchmarks)
- [License](#license)

## Installation

```bash
go install github.com/KromDaniel/regengo/cmd/regengo@latest
```

## Usage

### CLI

```bash
regengo -pattern '(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})' \
        -name Date \
        -output date.go \
        -package main
```

### Library

```go
package main

import "github.com/KromDaniel/regengo/pkg/regengo"

func main() {
    err := regengo.Compile(regengo.Options{
        Pattern:    `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`,
        Name:       "Date",
        OutputFile: "date.go",
        Package:    "main",
    })
    if err != nil {
        panic(err)
    }
}
```

### Options

```go
type Options struct {
    Pattern          string   // Regex pattern to compile (required)
    Name             string   // Name for generated struct (required)
    OutputFile       string   // Output file path (required)
    Package          string   // Package name (required)
    NoPool           bool     // Disable sync.Pool optimization
    GenerateTestFile bool     // Generate test file with benchmarks
    TestFileInputs   []string // Test inputs for generated tests
}
```

## Performance

Regengo consistently outperforms Go's standard `regexp` package:

### Best Results

| Pattern | Method | stdlib | regengo | Speedup |
|---------|--------|--------|---------|---------|
| Date `\d{4}-\d{2}-\d{2}` | FindString | 105 ns | 7 ns | **14x faster** |
| Multi-date extraction | FindAllString | 431 ns | 49 ns | **8.9x faster** |
| Date capture | FindString | 105 ns | 20 ns | **5.3x faster** |

### Typical Results

| Pattern | Method | stdlib | regengo | Speedup |
|---------|--------|--------|---------|---------|
| Email validation | MatchString | 1554 ns | 507 ns | **3x faster** |
| Email capture | FindString | 296 ns | 127 ns | **2.3x faster** |
| Log parser (TDFA) | FindString | 399 ns | 121 ns | **3.3x faster** |

### Memory Efficiency

- **50-100% fewer allocations** per operation
- **Zero allocations** with `Reuse` variants
- Typed structs instead of `[]string` slices

See [Detailed Benchmarks](#detailed-benchmarks) for complete results.

## Smart Analysis

Regengo automatically analyzes your pattern and selects the optimal matching engine:

### Supported Algorithms

| Algorithm | Use Case | Complexity | Status |
|-----------|----------|------------|--------|
| **Backtracking DFA** | Simple patterns | O(n) typical | ✅ Default |
| **Thompson NFA** | Patterns at risk of catastrophic backtracking | O(n×m) guaranteed | ✅ Supported |
| **Tagged DFA (TDFA)** | Capture groups with complex patterns | O(n) guaranteed | ✅ Supported |
| **Bit-vector Memoization** | Nested quantifiers with captures | O(n×m) with caching | ✅ Supported |

### Auto-Detection Examples

```
Pattern: [\w\.+-]+@[\w\.-]+
 → Backtracking DFA (simple, fast)

Pattern: (a+)+b
 → Thompson NFA (prevents exponential backtracking)

Pattern: (?P<user>\w+)@(?P<domain>\w+)
 → Tagged DFA (O(n) captures)

Pattern: (?P<outer>(?P<inner>a+)+)b
 → TDFA + Memoization (complex nested captures)
```

### Verbose Mode

See analysis decisions with `-verbose`:

```bash
regengo -pattern '(a+)+b' -name Test -output test.go -verbose
```

```
=== Pattern Analysis ===
Pattern: (a+)+b
NFA states: 8
Has nested quantifiers: true
Has catastrophic risk: true
→ Using Thompson NFA for MatchString
→ Using Tagged DFA for FindString
```

## Complexity Guarantees

### Runtime Complexity

| Operation | Go stdlib | Regengo | Notes |
|-----------|-----------|---------|-------|
| Simple match | O(n) | O(n) | Both efficient |
| Nested quantifiers `(a+)+` | **O(n×m)** guaranteed | **O(n×m)** guaranteed | Both use Thompson NFA construction |
| Captures | O(n) typical | O(n) guaranteed | TDFA eliminates backtracking overhead |
| Complex captures | **O(n×m)** guaranteed | **O(n×m)** with memoization | Both safe, Regengo uses bit-vector caching |

### Memory Complexity

| Aspect | Go stdlib | Regengo |
|--------|-----------|---------|
| Per-match allocation | 2 allocs (128-192 B) | 1 alloc (64-96 B) |
| With reuse API | N/A | **0 allocs** |
| Result storage | `[]string` slices | Typed structs |
| Backtracking stack | Dynamic allocation | `sync.Pool` reuse |

### Where Regengo May Be Slower

| Scenario | Reason | Mitigation |
|----------|--------|------------|
| Patterns with many optional groups | TDFA state explosion | Increase `-tdfa-threshold` or pattern redesign |
| Non-matching pathological patterns | Memoization overhead in capture groups | Use stdlib or reduce capture complexity |
| First cold call | No JIT, but consistent performance | Warm up in init() if needed |

> **Note:** Regengo trades compilation time for runtime performance. The generated code is optimized by the Go compiler, giving consistent, predictable performance without runtime interpretation overhead.

## Advanced Options

For fine-grained control over the compilation engine, use these advanced options:

### Library API

```go
type Options struct {
    // ... basic options ...

    // Engine selection
    ForceThompson bool // Force Thompson NFA for all match functions
    ForceTNFA     bool // Force Tagged NFA for capture functions
    ForceTDFA     bool // Force Tagged DFA for capture functions

    // Tuning parameters
    TDFAThreshold int  // Max DFA states before fallback (default: 500)
    Verbose       bool // Print analysis decisions to stderr
}
```

### When to Use Each Engine

| Option | Use Case |
|--------|----------|
| `ForceThompson` | Guaranteed O(n×m) for untrusted patterns |
| `ForceTNFA` | Debugging, or patterns with many optional groups |
| `ForceTDFA` | Maximize capture performance, predictable patterns |
| `TDFAThreshold` | Increase for complex patterns with known bounds |
| `Verbose` | Debug engine selection, understand analysis decisions |

### Example: Force TDFA for Trusted Patterns

```go
err := regengo.Compile(regengo.Options{
    Pattern:    `(?P<key>\w+)=(?P<value>[^;]+)`,
    Name:       "KeyValue",
    OutputFile: "keyvalue.go",
    Package:    "parser",
    ForceTDFA:  true,  // Skip analysis, use O(n) TDFA directly
})
```

### Example: Safe Mode for User Input

```go
err := regengo.Compile(regengo.Options{
    Pattern:       userPattern,
    Name:          "UserRegex",
    OutputFile:    "user_regex.go",
    Package:       "validator",
    ForceThompson: true,  // Guaranteed O(n×m), prevents ReDoS
})
```

## Generated Output

The above generates:

```go
package main

type Date struct{}

var CompiledDate = Date{}

type DateResult struct {
    Match string
    Year  string
    Month string
    Day   string
}

func (Date) MatchString(input string) bool { /* ... */ }
func (Date) MatchBytes(input []byte) bool { /* ... */ }
func (Date) FindString(input string) (*DateResult, bool) { /* ... */ }
func (Date) FindBytes(input []byte) (*DateBytesResult, bool) { /* ... */ }
func (Date) FindAllString(input string, n int) []*DateResult { /* ... */ }
func (Date) FindAllBytes(input []byte, n int) []*DateBytesResult { /* ... */ }
func (Date) FindAllStringAppend(input string, n int, s []*DateResult) []*DateResult { /* ... */ }
func (Date) FindAllBytesAppend(input []byte, n int, s []*DateBytesResult) []*DateBytesResult { /* ... */ }
```

### Usage of Generated Code

Both string and `[]byte` variants are auto-generated for each method.

```go
// String variant
if CompiledDate.MatchString("2024-12-25") {
    result, ok := CompiledDate.FindString("2024-12-25")
    if ok {
        fmt.Printf("Year: %s, Month: %s, Day: %s\n", result.Year, result.Month, result.Day)
    }
}

// Bytes variant
data := []byte("2024-12-25")
if CompiledDate.MatchBytes(data) {
    result, ok := CompiledDate.FindBytes(data)
    if ok {
        fmt.Printf("Year: %s, Month: %s, Day: %s\n", result.Year, result.Month, result.Day)
    }
}

// Find all matches
matches := CompiledDate.FindAllString("Dates: 2024-01-15 and 2024-12-25", -1)
for _, m := range matches {
    fmt.Println(m.Match)
}
```

## Generated Tests & Benchmarks

Regengo automatically generates a `_test.go` file alongside your output file (unless disabled). This file contains:

1. **Correctness Tests**: Verifies that Regengo's output matches `regexp` stdlib exactly for provided inputs.
   - `Test...MatchString`: Validates boolean matching.
   - `Test...MatchBytes`: Validates byte-slice matching.
   - `Test...FindString`: Validates capture groups (if present), checking both the full match and every individual captured group against stdlib's `FindStringSubmatch`.
   - `Test...FindAllString`: Validates all matches and their captures against stdlib's `FindAllStringSubmatch`.

2. **Benchmarks**: Comparison benchmarks to measure speedup vs stdlib.
   - `Benchmark...MatchString`: Performance of simple matching.
   - `Benchmark...FindString`: Performance of capture extraction (if applicable).

### Customizing Tests

You can provide specific test inputs to verify your pattern against real-world data:

**CLI:**
```bash
# Generates date.go and date_test.go
regengo -pattern '...' -name Date -output date.go -test-inputs "2024-01-01,2025-12-31"
```

**Library:**
```go
regengo.Options{
    // ...
    GenerateTestFile: true, // Required: Library defaults to false
    TestFileInputs:   []string{"2024-01-01", "2025-12-31"},
}
```

### Running Benchmarks

Run the generated benchmarks using standard Go tooling:

```bash
go test -bench=. -benchmem
```

## Capture Groups

Regengo generates dedicated structs for match groups, avoiding runtime slice allocations used by stdlib's `[]string` returns. Both string and `[]byte` variants are auto-generated.

**Note:** Result strings and bytes are slices into the original input, not copies. This improves performance but means results are only valid while the input remains unchanged.

### Named Groups

Pattern: `(?P<user>\w+)@(?P<domain>\w+)`

```go
// For FindString
type EmailResult struct {
    Match  string
    User   string  // from (?P<user>...)
    Domain string  // from (?P<domain>...)
}

// For FindBytes
type EmailBytesResult struct {
    Match  []byte
    User   []byte  // from (?P<user>...)
    Domain []byte  // from (?P<domain>...)
}
```

### Unnamed Groups

Pattern: `(\d{4})-(\d{2})-(\d{2})`

```go
// For FindString
type DateResult struct {
    Match  string
    Group1 string  // first capture
    Group2 string  // second capture
    Group3 string  // third capture
}

// For FindBytes
type DateBytesResult struct {
    Match  []byte
    Group1 []byte  // first capture
    Group2 []byte  // second capture
    Group3 []byte  // third capture
}
```

### Usage Example

```go
// String variant
result, ok := CompiledEmail.FindString("user@example.com")
if ok {
    fmt.Println(result.User, result.Domain)  // "user" "example"
}

// Bytes variant - result fields are slices into original input
data := []byte("user@example.com")
result, ok := CompiledEmail.FindBytes(data)
if ok {
    fmt.Println(result.User, result.Domain)  // "user" "example"
}
```

### Slice Reuse

For high-performance scenarios, use the `Append` variants to reuse slices and reduce allocations:

```go
// Pre-allocate a slice with capacity
results := make([]*EmailCaptureResult, 0, 100)

for _, input := range inputs {
    // Reuse the same backing array by resetting length to 0
    results = CompiledEmail.FindAllStringAppend(input, -1, results[:0])

    for _, match := range results {
        // Process matches...
        fmt.Println(match.User, match.Domain)
    }
}
```

The `Append` methods:
- Reuse existing slice capacity when possible
- Reset and reuse existing struct pointers within capacity
- Only allocate new elements when capacity is exceeded

This is particularly useful when processing many inputs in a loop, as it avoids repeated slice and struct allocations.

## API Comparison

### API Stability
❕ Regengo is still beta, API might change on minor versions ❕

| stdlib `regexp` | regengo | Notes |
|-----------------|---------|-------|
| `MatchString(s string) bool` | `MatchString(s string) bool` | ✅ Identical |
| `Match(b []byte) bool` | `MatchBytes(b []byte) bool` | ✅ Identical |
| `FindStringSubmatch(s string) []string` | `FindString(s string) (*Result, bool)` | ✅ Returns typed struct instead of []string |
| `FindSubmatch(b []byte) [][]byte` | `FindBytes(b []byte) (*BytesResult, bool)` | ✅ Returns typed struct instead of [][]byte |
| `FindAllStringSubmatch(s string, n int) [][]string` | `FindAllString(s string, n int) []*Result` | ✅ Returns []*Result instead of [][]string |
| `FindAllSubmatch(b []byte, n int) [][][]byte` | `FindAllBytes(b []byte, n int) []*BytesResult` | ✅ Returns []*BytesResult instead of [][][]byte |
| - | `FindStringReuse(s string, r *Result) (*Result, bool)` | Zero-alloc reuse variant |
| - | `FindBytesReuse(b []byte, r *BytesResult) (*BytesResult, bool)` | Zero-alloc reuse variant |
| - | `FindAllStringAppend(s string, n int, slice []*Result) []*Result` | Append to existing slice |
| - | `FindAllBytesAppend(b []byte, n int, slice []*BytesResult) []*BytesResult` | Append to existing slice |
| `FindString(s string) string` | - | Use FindString().Match |
| `Find(b []byte) []byte` | - | Use FindBytes().Match |
| `ReplaceAllString(s, repl string) string` | - | Not implemented |
| `Split(s string, n int) []string` | - | Not implemented |
| `FindStringIndex(s string) []int` | - | Not implemented |

## CLI Reference

```
Required:
  -pattern string    Regex pattern to compile
  -name string       Name for generated struct
  -output string     Output file path

Basic:
  -package string    Package name (default "main")
  -no-pool           Disable sync.Pool optimization
  -no-test           Disable test file generation
  -test-inputs       Comma-separated test inputs

Advanced (Engine Selection):
  -force-thompson    Force Thompson NFA for match functions
  -force-tnfa        Force Tagged NFA for capture functions
  -force-tdfa        Force Tagged DFA for capture functions

Tuning:
  -tdfa-threshold int  Max DFA states before fallback (default 500)
  -verbose             Print analysis decisions to stderr
```

### Basic Example

```bash
regengo -pattern '[\w\.+-]+@[\w\.-]+\.[\w\.-]+' \
        -name Email \
        -output email.go \
        -package myapp
```

### Advanced Example: Verbose Analysis

```bash
regengo -pattern '(a+)+b' \
        -name Dangerous \
        -output dangerous.go \
        -verbose
```

Output:
```
=== Pattern Analysis ===
Pattern: (a+)+b
NFA states: 8
Has nested quantifiers: true
→ Using Thompson NFA for MatchString (prevents ReDoS)
→ Using Tagged DFA for FindString
```

### Advanced Example: Force Specific Engine

```bash
# Force TDFA for maximum capture performance
regengo -pattern '(?P<k>\w+)=(?P<v>[^;]+)' \
        -name KV \
        -output kv.go \
        -force-tdfa

# Force Thompson NFA for untrusted patterns
regengo -pattern "$USER_PATTERN" \
        -name UserRegex \
        -output user.go \
        -force-thompson
```

## Detailed Benchmarks

Benchmarks run on Apple M4 Pro. Each benchmark shows performance for Go stdlib vs regengo.

### DateCaptureFindString

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 105.3 | 128 | 2 | - |
| regengo | 19.7 | 64 | 1 | **5.3x faster** |
| regengo (reuse) | 7.3 | 0 | 0 | **14.4x faster** |

### EmailCaptureFindString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 295.9 | 128 | 2 | - |
| regengo | 127.3 | 64 | 1 | **2.3x faster** |
| regengo (reuse) | 115.1 | 0 | 0 | **2.6x faster** |

### EmailMatchString

**Pattern:**
```regex
[\w\.+-]+@[\w\.-]+\.[\w\.-]+
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1554.0 | 0 | 0 | - |
| regengo | 507.3 | 0 | 0 | **3.1x faster** |

### GreedyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 751.4 | 0 | 0 | - |
| regengo | 475.3 | 0 | 0 | **1.6x faster** |

### LazyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1260.0 | 0 | 0 | - |
| regengo | 478.0 | 0 | 0 | **2.6x faster** |

### MultiDateFindAllString

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 431.4 | 331 | 3 | - |
| regengo | 82.2 | 106 | 2 | **5.2x faster** |
| regengo (reuse) | 48.6 | 0 | 0 | **8.9x faster** |

### MultiEmailFindAllString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 978.1 | 374 | 4 | - |
| regengo | 478.1 | 133 | 3 | **2.0x faster** |
| regengo (reuse) | 438.5 | 0 | 0 | **2.2x faster** |

### TDFAComplexURLFindString

**Pattern:**
```regex
(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 598.7 | 288 | 2 | - |
| regengo | 285.4 | 421 | 2 | **2.1x faster** |
| regengo (reuse) | 263.7 | 277 | 1 | **2.3x faster** |

### TDFALogParserFindString

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 398.8 | 192 | 2 | - |
| regengo | 120.9 | 96 | 1 | **3.3x faster** |
| regengo (reuse) | 106.0 | 0 | 0 | **3.8x faster** |

### TDFANestedWordFindString

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 654.9 | 112 | 2 | - |
| regengo | 403.6 | 128 | 1 | **1.6x faster** |
| regengo (reuse) | 394.6 | 80 | 0 | **1.7x faster** |

### TDFAPathologicalFindString

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 492.3 | 84 | 1 | - |
| regengo | 896.5 | 80 | 1 | **0.5x faster** |
| regengo (reuse) | 888.9 | 44 | 0 | **0.6x faster** |

### TDFASemVerFindString

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 211.3 | 192 | 2 | - |
| regengo | 72.6 | 96 | 1 | **2.9x faster** |
| regengo (reuse) | 57.1 | 0 | 0 | **3.7x faster** |

### URLCaptureFindString

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 293.8 | 160 | 2 | - |
| regengo | 116.6 | 80 | 1 | **2.5x faster** |
| regengo (reuse) | 112.7 | 0 | 0 | **2.6x faster** |

---

To regenerate these benchmarks: `make bench-readme`

## License

MIT License - see [LICENSE](LICENSE) for details.
