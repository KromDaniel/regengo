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

- [Performance](#performance)
- [Installation](#installation)
- [Usage](#usage)
- [Generated Output](#generated-output)
- [Capture Groups](#capture-groups)
- [API Comparison](#api-comparison)
- [CLI Options](#cli-options)
- [Detailed Benchmarks](#detailed-benchmarks)
- [License](#license)

## Performance

Regengo outperforms Go's standard `regexp` package across all benchmarks:

| Pattern | Type | stdlib | regengo | Speedup |
|---------|------|--------|---------|---------|
| `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})` | FindString | 99 ns | 18 ns | **5.5x faster** |
| `[\w\.+-]+@[\w\.-]+\.[\w\.-]+` | MatchString | 1549 ns | 488 ns | **3.2x faster** |
| `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)` | FindString | 242 ns | 99 ns | **2.4x faster** |
| `https?://[\w\.-]+(?::\d+)?(?:/[\w\./]*)?` | FindString | 367 ns | 229 ns | **1.6x faster** |

Memory usage is also reduced: **50% fewer allocations** and **50% less bytes per operation** for capture groups.

See [Detailed Benchmarks](#detailed-benchmarks) for complete results.

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

## CLI Options

```
-pattern string    Regex pattern to compile (required)
-name string       Name for generated struct (required)
-output string     Output file path (required)
-package string    Package name (default "main")
-no-pool          Disable sync.Pool optimization
-no-test          Disable test file generation
-test-inputs      Comma-separated test inputs
```

### Example

```bash
regengo -pattern '[\w\.+-]+@[\w\.-]+\.[\w\.-]+' \
        -name Email \
        -output email.go \
        -package myapp \
        -test-inputs 'user@example.com,invalid,test@test.org'
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
| stdlib | 102.8 | 128 | 2 | - |
| regengo | 19.4 | 64 | 1 | **5.3x faster** |
| regengo (reuse) | 7.4 | 0 | 0 | **13.8x faster** |

### EmailCaptureFindString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 294.2 | 128 | 2 | - |
| regengo | 115.5 | 64 | 1 | **2.5x faster** |
| regengo (reuse) | 104.8 | 0 | 0 | **2.8x faster** |

### EmailMatchString

**Pattern:**
```regex
[\w\.+-]+@[\w\.-]+\.[\w\.-]+
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1574.0 | 0 | 0 | - |
| regengo | 499.7 | 0 | 0 | **3.1x faster** |

### GreedyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 750.7 | 0 | 0 | - |
| regengo | 622.4 | 0 | 0 | **1.2x faster** |

### LazyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1252.0 | 0 | 0 | - |
| regengo | 905.8 | 0 | 0 | **1.4x faster** |

### MultiDateFindAllString

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 425.0 | 331 | 3 | - |
| regengo | 80.7 | 106 | 2 | **5.3x faster** |
| regengo (reuse) | 48.3 | 0 | 0 | **8.8x faster** |

### MultiEmailFindAllString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 954.9 | 374 | 4 | - |
| regengo | 398.2 | 133 | 3 | **2.4x faster** |
| regengo (reuse) | 366.3 | 0 | 0 | **2.6x faster** |

### URLCaptureFindString

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 292.0 | 160 | 2 | - |
| regengo | 165.0 | 80 | 1 | **1.8x faster** |
| regengo (reuse) | 151.8 | 0 | 0 | **1.9x faster** |

---

To regenerate these benchmarks: `make bench-readme`

## License

MIT License - see [LICENSE](LICENSE) for details.
