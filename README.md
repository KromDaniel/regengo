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

# Generate basic matcher
regengo -pattern "[\w\.+-]+@[\w\.-]+\.[\w\.-]+" -name Email -output email.go -package main

# Generate with capture groups
regengo -pattern "(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)" -name Email -output email.go -package main -captures

# Generate with pool optimization (recommended for production)
regengo -pattern "[\w\.+-]+@[\w\.-]+\.[\w\.-]+" -name Email -output email.go -package main -pool

# Generate with zero-copy []byte captures (best performance for byte inputs)
regengo -pattern "(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)" -name Email -output email.go -package main -captures -bytes-view

# Combine optimizations for maximum performance
regengo -pattern "(?P<protocol>https?)://(?P<host>[\w\.-]+)" -name URL -output url.go -package main -pool -captures -bytes-view
```

## 📊 Performance

Regengo provides **dramatic performance improvements** over the standard `regexp` package, especially with pool optimization enabled:

### Basic Matching (MatchString)

#### With Pool Optimization (`-pool` flag)

| Pattern | Regengo (pooled) | Standard regexp | Speedup         |
| ------- | ---------------- | --------------- | --------------- |
| Email   | 289 ns/op        | 882 ns/op       | **3.1x faster** |
| URL     | 121 ns/op        | 549 ns/op       | **4.5x faster** |
| IPv4    | 96 ns/op         | 470 ns/op       | **4.9x faster** |

**Memory**: 0 allocations/op with pool (vs 0 for standard regexp, 14-23 for non-pooled)

#### Without Pool Optimization

| Pattern | Regengo   | Standard regexp | Speedup         |
| ------- | --------- | --------------- | --------------- |
| Email   | 585 ns/op | 882 ns/op       | **1.5x faster** |
| URL     | 373 ns/op | 549 ns/op       | **1.5x faster** |
| IPv4    | 237 ns/op | 470 ns/op       | **2.0x faster** |

### Capture Groups (FindStringSubmatch)

#### Standard (String Fields)

| Pattern      | Regengo    | Standard regexp | Speedup         | Allocations |
| ------------ | ---------- | --------------- | --------------- | ----------- |
| DateCapture  | 20.7 ns/op | 101 ns/op       | **5x faster**   | 1 vs 2      |
| EmailCapture | 123 ns/op  | 246 ns/op       | **2x faster**   | 6 vs 2      |
| URLCapture   | 118 ns/op  | 196 ns/op       | **1.7x faster** | 6 vs 2      |

#### BytesView (Zero-Copy []byte Fields)

For `[]byte` inputs, BytesView eliminates string conversion allocations:

| Pattern    | Standard (string) | BytesView ([]byte) | Improvement          |
| ---------- | ----------------- | ------------------ | -------------------- |
| URLCapture | 269 ns/op         | 150 ns/op          | **1.8x faster**      |
|            | 2,128 B/op        | 592 B/op           | **72% less memory**  |
|            | 8 allocs/op       | 2 allocs/op        | **75% fewer allocs** |

**Generate with**: `regengo -pattern "..." -name URL -captures -bytes-view`

**Note**: Standard capture groups generate optimized structs with named fields. Optional groups are handled efficiently, returning empty strings (or `nil` for []byte) when not matched.

_Run `make bench` to see benchmarks on your system._

### 🎯 Pool Optimization

The `-pool` flag enables `sync.Pool` for stack reuse, achieving:

- ✅ **Zero allocations** per match operation
- ✅ **3-5x faster** than standard regexp
- ✅ **Thread-safe** concurrent access
- ✅ **Automatic memory management**

**Recommendation**: Use `-pool` for production deployments and hot paths.

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
    Pattern      string // Regex pattern to compile
    Name         string // Function name prefix
    OutputFile   string // Output file path
    Package      string // Go package name
    UsePool      bool   // Enable sync.Pool optimization (recommended)
    WithCaptures bool   // Generate capture group extraction
    BytesView    bool   // Generate zero-copy []byte capture structs (requires WithCaptures)
}
```

**Option Details**:

- `UsePool`: Use `sync.Pool` for stack reuse (0 allocs, 3-5x faster)
- `WithCaptures`: Extract named/indexed submatches into type-safe structs
- `BytesView`: Generate `[]byte` fields for zero-copy captures (72% less memory for byte inputs)

### Capture Groups

When `WithCaptures` is enabled, Regengo generates optimized functions that extract submatches:

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

- Named groups: `(?P<name>...)` → struct field `Name`
- Indexed groups: `(...)` → struct field `Match1`, `Match2`, etc.
- Optional groups: Return empty string when not matched
- Type-safe: Compile-time checked struct fields

### BytesView: Zero-Copy []byte Captures

For maximum performance with `[]byte` inputs, use the `-bytes-view` flag to generate zero-copy capture structs:

```bash
regengo -pattern "(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)" -name Email -output email.go -captures -bytes-view
```

This generates **two struct types**:

```go
// For string inputs (FindString)
type EmailMatch struct {
    Match  string
    User   string
    Domain string
}

// For []byte inputs (FindBytes) - zero-copy!
type EmailBytesMatch struct {
    Match  []byte  // Direct slice reference, no allocation
    User   []byte  // Direct slice reference, no allocation
    Domain []byte  // Direct slice reference, no allocation
}

func EmailFindString(input string) (*EmailMatch, bool)
func EmailFindBytes(input []byte) (*EmailBytesMatch, bool)  // Returns []byte fields
```

**Benefits**:

- ✅ **Zero allocations** for capture field slicing
- ✅ **72% less memory** per match (vs string conversion)
- ✅ **1.8x faster** for []byte inputs
- ✅ **Direct slice references** - no `string()` conversions

**When to use**:

- Processing `[]byte` data (HTTP bodies, file buffers, protocol parsing)
- Performance-critical hot paths
- Want to avoid string conversion allocations

⚠️ **Important**: The returned `[]byte` slices reference the original input. Do not modify the input while using the result.

See [BytesView documentation](docs/BYTES_VIEW.md) for detailed usage and safety considerations.

## 📝 Examples

See the [examples](./examples) directory for more usage patterns:

- Email validation
- URL matching
- Capture groups (named and indexed)
- Custom patterns

Capture group examples can be found in [examples/captures](./examples/captures).

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
- [ ] Web-based playground
- [ ] VSCode extension

## 📧 Contact

Daniel Krom - [@KromDaniel](https://github.com/KromDaniel)

Project Link: [https://github.com/KromDaniel/regengo](https://github.com/KromDaniel/regengo)
