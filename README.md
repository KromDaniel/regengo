# Regengo

[![Go Reference](https://pkg.go.dev/badge/github.com/KromDaniel/regengo.svg)](https://pkg.go.dev/github.com/KromDaniel/regengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/KromDaniel/regengo)](https://goreportcard.com/report/github.com/KromDaniel/regengo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

<p align="center">
  <img src="assets/logo.png" alt="Regengo - Go Gopher with Regex" width="400">
</p>

Regengo is a **compile-time finite state machine generator** for regular expressions. It converts regex patterns into optimized Go code, leveraging the Go compiler's optimizations to eliminate runtime interpretation overhead.

## Performance

Regengo outperforms Go's standard `regexp` package across all benchmarks:

| Pattern | Type | stdlib | regengo | Speedup |
|---------|------|--------|---------|---------|
| `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})` | FindString | 99 ns | 18 ns | **5.5x faster** |
| `[\w\.+-]+@[\w\.-]+\.[\w\.-]+` | MatchString | 1549 ns | 488 ns | **3.2x faster** |
| `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)` | FindString | 242 ns | 99 ns | **2.4x faster** |
| `https?://[\w\.-]+(?::\d+)?(?:/[\w\./]*)?` | FindString | 367 ns | 229 ns | **1.6x faster** |

Memory usage is also reduced: **50% fewer allocations** and **50% less bytes per operation** for capture groups.

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

## API Comparison

| stdlib `regexp` | regengo | Status |
|-----------------|---------|--------|
| `MatchString(s string) bool` | `MatchString(s string) bool` | Implemented |
| `Match(b []byte) bool` | `MatchBytes(b []byte) bool` | Implemented |
| `FindString(s string) string` | - | Not implemented |
| `FindStringSubmatch(s string) []string` | `FindString(s string) (*Result, bool)` | Implemented (typed struct) |
| `Find(b []byte) []byte` | - | Not implemented |
| `FindSubmatch(b []byte) [][]byte` | `FindBytes(b []byte) (*BytesResult, bool)` | Implemented (typed struct) |
| `FindAllStringSubmatch(s string, n int) [][]string` | `FindAllString(s string, n int) []*Result` | Implemented |
| `FindAllSubmatch(b []byte, n int) [][][]byte` | `FindAllBytes(b []byte, n int) []*BytesResult` | Implemented |
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

## License

MIT License - see [LICENSE](LICENSE) for details.
