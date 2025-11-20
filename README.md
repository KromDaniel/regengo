# Regengo 🚀

[![Go Reference](https://pkg.go.dev/badge/github.com/KromDaniel/regengo.svg)](https://pkg.go.dev/github.com/KromDaniel/regengo)
[![Go Report Card](https://goreportcard.com/badge/github.com/KromDaniel/regengo)](https://goreportcard.com/report/github.com/KromDaniel/regengo)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Regengo is a high-performance regex-to-Go code generator that compiles regular expressions into optimized Go functions at build time. By converting regex patterns into native Go code, the Go compiler can apply its full optimization suite, resulting in significantly faster pattern matching compared to traditional runtime regex engines.

## 🎯 Features

- **Compile-Time Optimization**: Convert regex patterns to optimized Go code
- **Zero Runtime Overhead**: No regex engine interpretation at runtime
- **Type-Safe**: Generated functions are type-checked by the Go compiler
- **High Performance**: Benchmarks show significant speedups over `regexp` package
- **Capture Groups**: Extract named and indexed submatches with optimized struct generation
- **FindAll Support**: Extract all matches from input with `FindAllString` and `FindAllBytes`
- **Easy Integration**: Simple API for code generation
- **CLI Tool**: Command-line interface for batch generation

## 📦 Installation

```bash
go get github.com/KromDaniel/regengo
```

## 🚀 Quick Start

### As a Library

```go
package main

import (
    "github.com/KromDaniel/regengo/pkg/regengo"
)

func main() {
    // Compile a regex pattern to Go code
    err := regengo.Compile(regengo.Options{
        Pattern:    `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
        Name:       "Email",
        OutputFile: "./generated/email.go",
        Package:    "generated",
    })
    if err != nil {
        panic(err)
    }
}
```

This generates a file with optimized matcher functions:

```go
package generated

func EmailMatchString(input string) bool {
    // ... optimized matching code ...
}

func EmailMatchBytes(input []byte) bool {
    // ... optimized matching code ...
}
```

### As a CLI Tool

```bash
# Install CLI
go install github.com/KromDaniel/regengo/cmd/regengo@latest

# Generate matcher from pattern
regengo -pattern "[\w\.+-]+@[\w\.-]+\.[\w\.-]+" -name Email -output email.go -package main

# Generate with capture groups (auto-detected from pattern)
regengo -pattern "(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+\.[\w\.-]+)" -name Email -output email.go -package main

# Disable pool optimization if needed
regengo -pattern "[\w\.+-]+@[\w\.-]+\.[\w\.-]+" -name Email -output email.go -package main -no-pool
```

**CLI Flags**:

- `-pattern`: Regex pattern to compile (required)
- `-name`: Function name prefix (required)
- `-output`: Output file path (required)
- `-package`: Go package name (required)
- `-no-pool`: Disable sync.Pool optimization (default: pool enabled)
- `-test`: Generate test file with sample inputs

## 📊 Performance

Regengo provides **dramatic performance improvements** over the standard `regexp` package. All benchmarks below are from actual test runs on Apple M4 Pro.

### 🏆 Mass Benchmark Results

**Regengo wins ALL 185 benchmarks** across simple, complex, and very complex patterns:

| Category      | Patterns | Regengo Wins | Stdlib Wins | Speed Improvement | Memory Improvement | Allocs Improvement |
| ------------- | -------- | ------------ | ----------- | ----------------- | ------------------ | ------------------ |
| Simple        | 90       | **90**       | 0           | **94.6% faster**  | 0 B/op (same)      | 0 allocs (same)    |
| Complex       | 60       | **60**       | 0           | **65.6% faster**  | **50.0% less**     | **50.0% less**     |
| Very Complex  | 35       | **35**       | 0           | **60.3% faster**  | **51.4% less**     | **50.0% less**     |
| **OVERALL**   | **185**  | **185**     | **0**       | **69.1% faster**  | **50.8% less**     | **50.0% less**     |

**Key Takeaways:**
- ✅ **100% win rate**: Regengo outperforms stdlib in every single benchmark
- ✅ **69.1% faster** on average across all pattern types
- ✅ **50.8% less memory** used per operation
- ✅ **50% fewer allocations** for complex patterns
- ✅ **113 ns/op** (regengo) vs **365 ns/op** (stdlib) average time

Run comprehensive benchmarks yourself:
```bash
make mass-bench  # Generates and runs 185 pattern benchmarks
```

### Pattern Matching (MatchString)

| Pattern | Regengo    | Standard regexp | Speedup          | Memory   |
| ------- | ---------- | --------------- | ---------------- | -------- |
| Email   | **480 ns** | 1,505 ns        | **3.1x faster**  | 0 allocs |
| Greedy  | 623 ns     | 738 ns          | **1.2x faster**  | 0 allocs |
| Lazy    | 880 ns     | 1,244 ns        | **1.4x faster**  | 0 allocs |

**🚀 Greedy Loop Optimization**: Regengo includes intelligent auto-detection for greedy quantifiers (`+`, `*`, `{n,}`). Simple character class loops like `[\w]+` or `[a-z]*` are automatically optimized to avoid O(N²) backtracking, resulting in massive speedups for patterns like email matching.

### Capture Groups (FindStringSubmatch)

Regengo generates **optimized structs** with named fields, providing massive speedups:

| Pattern      | Regengo       | Standard regexp | Speedup          | Memory (Regengo)  | Memory (Stdlib)     |
| ------------ | ------------- | --------------- | ---------------- | ----------------- | ------------------- |
| DateCapture  | **19 ns/op**  | 101 ns/op       | **5.3x faster**  | 64 B/op (1 alloc) | 128 B/op (2 allocs) |
| EmailCapture | **243 ns/op** | 249 ns/op       | **1.0x faster**  | 1216 B/op (19 allocs) | 128 B/op (2 allocs) |
| URLCapture   | **226 ns/op** | 183 ns/op       | _0.8x_           | 1200 B/op (15 allocs) | 160 B/op (2 allocs) |

**Key Benefits:**

- ✅ **Up to 5x faster** than standard regexp for simple capture groups
- ✅ **50% less memory** for simple patterns
- ✅ **Type-safe structs** with named fields
- ✅ **Fewer allocations** for simple patterns
- ✅ **Optional groups handled** efficiently (empty strings when not matched)

_Note: Complex patterns with many alternations may have more allocations. Email/URL capture performance is being actively optimized._

### Real-World Examples

```go
// DateCapture: (?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
// Input: "2024-12-25"
result, found := DateCaptureFindString("2024-12-25")
// stdlib: 101 ns/op, 128 B/op, 2 allocs
// regengo: 19 ns/op, 64 B/op, 1 alloc → 5.3x faster

// EmailMatchString: [\w\.+-]+@[\w\.-]+\.[\w\.-]+
// Input: "aaa...aaa| me@myself.com" (100 'a's + email)
result := EmailMatchString(input)
// stdlib: 1,505 ns/op, 0 B/op, 0 allocs
// regengo: 480 ns/op, 0 B/op, 0 allocs → 3.1x faster (greedy loop optimization)
```

_Run `make bench` to see benchmarks on your system. Results from `go test -bench=. ./benchmarks/generated`_

### 🎯 Pool Optimization

The generated code uses `sync.Pool` by default for stack reuse, achieving:

- ✅ **Zero allocations** for backtracking stack management
- ✅ **3-4x faster** than standard regexp for capture groups
- ✅ **Thread-safe** concurrent access
- ✅ **Automatic memory management**

To disable pool optimization, use the `NoPool` option:

```go
err := regengo.Compile(regengo.Options{
    Pattern:    `[\w\.+-]+@[\w\.-]+\.[\w\.-]+`,
    Name:       "Email",
    OutputFile: "./generated/email.go",
    Package:    "generated",
    NoPool:     true, // Disable pool optimization
})
```

**Recommendation**: Keep pool enabled (default) for production deployments and hot paths.

See [Memory Optimization docs](docs/MEMORY_OPTIMIZATION.md) for details.

## 🏗️ How It Works

1. **Parse**: Uses Go's `regexp/syntax` to parse the regex pattern
2. **Simplify**: Optimizes the regex syntax tree
3. **Compile**: Converts to a finite state machine
4. **Generate**: Produces optimized Go code using code generation
5. **Optimize**: Go compiler applies optimizations to the generated code

The generated code uses techniques like:

- Inline state transitions
- Backtracking with explicit stack management
- Direct byte/rune comparisons instead of regex engine interpretation

## 🔧 Configuration

The `Options` struct provides full control over code generation:

```go
type Options struct {
    Pattern    string // Regex pattern to compile (required)
    Name       string // Function name prefix (required)
    OutputFile string // Output file path (required)
    Package    string // Go package name (required)
    NoPool     bool   // Disable sync.Pool optimization (default: false, pool enabled)
}
```

**Important Notes**:

- **Capture groups are auto-detected**: If your pattern has named groups like `(?P<name>...)` or numbered groups `(...)`, capture extraction functions are automatically generated
- **Pool optimization is enabled by default**: Set `NoPool: true` only if you need to disable it
- **Generated functions**:
  - `{Name}MatchString(input string) bool` - Check if pattern matches
  - `{Name}MatchBytes(input []byte) bool` - Check if pattern matches bytes
  - `{Name}FindString(input string) (*{Name}Match, bool)` - Extract captures (if pattern has groups)
  - `{Name}FindBytes(input []byte) (*{Name}Match, bool)` - Extract captures from bytes
  - `{Name}FindAllString(input string, n int) []*{Name}Match` - Find all matches (if pattern has groups)
  - `{Name}FindAllBytes(input []byte, n int) []*{Name}Match` - Find all matches from bytes (if pattern has groups)

### FindAll: Multiple Match Extraction

When your pattern has capture groups, Regengo generates `FindAll` functions to extract **all matches** from the input, similar to Go's stdlib `regexp.FindAllStringSubmatch`:

```go
// Example: Find all dates in a string
pattern := `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`

// Generated functions
func DateCaptureFindAllString(input string, n int) []*DateCaptureMatch
func DateCaptureFindAllBytes(input []byte, n int) []*DateCaptureMatch

// Usage
text := "Dates: 2024-01-15 and 2024-12-25 and 2025-06-30"

// Find all matches
matches := DateCaptureFindAllString(text, -1)
// Returns 3 matches with filled Year, Month, Day fields

// Find up to 2 matches
matches := DateCaptureFindAllString(text, 2)
// Returns 2 matches

// Find no matches
matches := DateCaptureFindAllString(text, 0)
// Returns nil
```

**Parameter `n` controls max matches**:

- `n < 0`: Find all matches (unlimited)
- `n = 0`: Return nil immediately (no search)
- `n > 0`: Return up to n matches

**Features**:

- ✅ **Compatible with stdlib**: Same semantics as `regexp.FindAllStringSubmatch`
- ✅ **Type-safe**: Returns slice of typed structs, not `[][]string`
- ✅ **Zero-width match handling**: Automatically advances to prevent infinite loops
- ✅ **Pool-optimized**: Reuses stack allocations for performance
- ✅ **Non-overlapping**: Finds matches sequentially (standard regex behavior)

### Capture Groups

Regengo automatically detects capture groups in your pattern and generates optimized extraction functions:

```go
// Example pattern with named groups
pattern := `(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)`

// Generated struct
type EmailMatch struct {
    Match  string // Full matched string
    Start  int    // Start position
    End    int    // End position
    User   string // Named capture group
    Domain string // Named capture group
    Tld    string // Named capture group
}

// Generated function
func EmailFindString(input string) (*EmailMatch, bool)
```

**Features**:

- **Auto-detection**: Named groups `(?P<name>...)` and indexed groups `(...)` automatically trigger capture generation
- **Named groups**: `(?P<name>...)` → struct field `Name`
- **Indexed groups**: `(...)` → struct field `Group1`, `Group2`, etc.
- **Optional groups**: Return empty string when not matched (e.g., `(?P<port>\d+)?`)
- **Type-safe**: Compile-time checked struct fields
- **Performance**: 3-4x faster than stdlib with 50% less memory

### BytesView: Zero-Copy []byte Captures

The generated code automatically provides zero-copy `[]byte` capture support through the `FindBytes` function:

```go
// For []byte inputs - zero-copy!
type EmailMatch struct {
    Match  []byte  // Direct slice reference, no allocation
    User   []byte  // Direct slice reference, no allocation
    Domain []byte  // Direct slice reference, no allocation
}

func EmailFindBytes(input []byte) (*EmailMatch, bool)  // Returns []byte fields
```

**Benefits**:

- ✅ **Zero allocations** for capture field slicing
- ✅ **50% less memory** per match (64-80 bytes vs 128-160 bytes)
- ✅ **3-4x faster** for capture extraction
- ✅ **Direct slice references** - no `string()` conversions
- ✅ **Automatic** - just use `FindBytes` instead of `FindString`

**When to use**:

- Processing `[]byte` data (HTTP bodies, file buffers, protocol parsing)
- Performance-critical hot paths
- Want to avoid string conversion allocations

⚠️ **Important**: The returned `[]byte` slices reference the original input. Do not modify the input while using the result.

See [BytesView documentation](docs/BYTES_VIEW.md) for detailed usage and safety considerations.

### Repeating Capture Groups

**Important**: Regex patterns with repeating capture groups (e.g., `(\w)+` or `(\d)*`) follow standard POSIX regex behavior.

Go's `regexp` package (like most regex engines) **captures only the LAST match** from repeating groups:

```go
// Pattern: (\w)+@(\w+)\.com
// Input: "abc@example.com"

// Group 1: (\w)+ matches "a", "b", "c" sequentially
// But only captures: "c" (the last character)

// Generated code includes warning:
// Warning: This pattern contains repeating capture groups (e.g., (\w)+ or (\d)*).
// Go's regex engine (like most regex implementations) captures only the LAST match from repeating groups.
// For example, pattern (\w)+ matching 'abc' will capture 'c', not ['a', 'b', 'c'].

type EmailMatch struct {
    Match  string // Full match: "abc@example.com"
    Group1 string // Only last char: "c"
    Group2 string // Complete match: "example"
}
```

**Examples of repeating captures**:

- `(\w)+` - Matches one or more word characters, captures LAST one
- `(\d)*` - Matches zero or more digits, captures LAST one
- `(\w){3,5}` - Matches 3-5 word characters, captures LAST one
- `(\w)(\d)+` - Group 1 captures normally, Group 2 captures LAST digit

**Non-repeating alternatives**:

- `(\w+)` - Captures ALL matched characters (not repeating the group itself)
- `(?P<user>\w+)@(?P<domain>\w+)` - Each group captures its full match

This is standard regex behavior across all implementations (Perl, Python, JavaScript, etc.), not a regengo limitation. The generated code includes automatic warnings when repeating captures are detected.

See [Repeating Captures Documentation](docs/REPEATING_CAPTURES.md) for detailed explanation and [examples/repeating_captures_demo.go](./examples/repeating_captures_demo.go) for a working demonstration.

## 📝 Examples

Check the [benchmarks/generated](./benchmarks/generated) directory to see actual generated code examples. You can regenerate these by running:

```bash
make bench-gen
```

This will generate matchers for various patterns including:

- Date matching with captures (year, month, day)
- Email matching with captures (user, domain)
- URL matching with captures (protocol, host, port, path)
- Greedy and lazy quantifiers

## 🧪 Testing

```bash
# Run all tests
make test

# Run benchmarks
make bench

# Generate benchmark code
make bench-gen
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built on top of Go's excellent `regexp/syntax` package
- Code generation powered by [jennifer](https://github.com/dave/jennifer)

## ⚠️ Limitations

- Not all regex features are currently supported
- Generated code may be larger than using the `regexp` package
- Best suited for hot-path performance-critical regex matching

## 🗺️ Roadmap

- [x] Support for capture groups
- [ ] More regex operations (FindAll, Replace, etc.)
- [ ] Parallel matching optimization

## 📧 Contact

Daniel Krom - [@KromDaniel](https://github.com/KromDaniel)

Project Link: [https://github.com/KromDaniel/regengo](https://github.com/KromDaniel/regengo)
