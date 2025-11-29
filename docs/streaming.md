# Streaming API

Regengo supports streaming regex matching on `io.Reader` inputs, enabling efficient processing of arbitrarily large files and network streams with **constant memory usage**.

## Generated Streaming Methods

When you generate code with regengo, the following streaming methods are automatically generated:

```go
// FindReader calls onMatch for each match found in the reader.
// Returns error from the reader, or nil on success.
// Matches are delivered via callback to avoid buffering.
func (Date) FindReader(
    r io.Reader,
    cfg stream.Config,
    onMatch func(m stream.Match[*DateBytesResult]) bool,
) error

// FindReaderCount returns the total number of matches in the reader.
func (Date) FindReaderCount(r io.Reader, cfg stream.Config) (int64, error)

// FindReaderFirst returns the first match found, or nil if no match.
// The int64 return is the stream offset of the match (-1 if no match).
func (Date) FindReaderFirst(r io.Reader, cfg stream.Config) (*DateBytesResult, int64, error)

// MatchLengthInfo returns the minimum and maximum match lengths for the pattern.
// maxLen is -1 for unbounded patterns (e.g., `a+`).
func (Date) MatchLengthInfo() (minLen, maxLen int)
```

## Basic Usage

```go
import (
    "os"
    stream "github.com/KromDaniel/regengo/stream"
)

// Process a large log file
file, _ := os.Open("server.log")
defer file.Close()

err := CompiledDate.FindReader(file, stream.Config{}, func(m stream.Match[*DateBytesResult]) bool {
    fmt.Printf("Found date at offset %d: %s\n", m.StreamOffset, m.Result.Match)
    return true // continue processing
})
```

## Configuration

```go
type Config struct {
    // BufferSize is the chunk size for reading from the io.Reader.
    // Default: 64KB. Larger values reduce syscall overhead.
    BufferSize int

    // MaxLeftover limits bytes kept between chunks when no match is found.
    // Prevents unbounded memory growth on long non-matching sections.
    // Default: computed from pattern analysis.
    MaxLeftover int
}
```

### Default Values

| Setting | Default | Notes |
|---------|---------|-------|
| BufferSize | 64KB | Minimum enforced: 64KB or 2*MaxMatchLength (whichever is larger) |
| MaxLeftover | Pattern-dependent | Bounded: 10*MaxMatchLength (clamped to 1KB-1MB). Unbounded: 1MB |

> **Warning for unbounded patterns:** For patterns like `.*` or `.+`, matches longer than `MaxLeftover` (default 1MB) that cross chunk boundaries may be truncated or missed entirely. If you expect very long matches, increase `MaxLeftover` accordingly.

### Buffer Size Guidelines

| Use Case | Recommended Size |
|----------|-----------------|
| Small files (<1MB) | 64KB (default) |
| Large files (>100MB) | 1-4MB |
| Network streams | 64KB-256KB |
| Memory constrained | 64KB (minimum enforced) |

## Callback-Based Streaming

The callback receives a `stream.Match` containing:

```go
type Match[T any] struct {
    Result       T       // Pattern-specific result (*DateBytesResult, etc.)
    StreamOffset int64   // Absolute byte position in stream
    ChunkIndex   int     // Which chunk the match was found in
}
```

**Important:** The callback's `Result` byte slices point into a reusable buffer. You **must copy** any data you need to retain after the callback returns:

```go
var savedMatches []string

err := pattern.FindReader(reader, cfg, func(m stream.Match[*ResultType]) bool {
    // This creates a copy - which is what you need:
    savedMatches = append(savedMatches, string(m.Result.Match))

    // Or explicitly copy bytes:
    matchCopy := append([]byte{}, m.Result.Match...)

    return true
})
```

## Early Termination

Return `false` from the callback to stop processing early:

```go
var firstTen []string

err := pattern.FindReader(reader, stream.Config{}, func(m stream.Match[*ResultType]) bool {
    firstTen = append(firstTen, string(m.Result.Match))
    return len(firstTen) < 10 // stop after 10 matches
})
```

## Counting Matches

For large files where you only need the count:

```go
count, err := CompiledDate.FindReaderCount(file, stream.Config{})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d dates in file\n", count)
```

## Finding First Match

When you only need the first match:

```go
result, offset, err := CompiledDate.FindReaderFirst(file, stream.Config{})
if err != nil {
    log.Fatal(err)
}
if result != nil {
    fmt.Printf("First date at offset %d: %s\n", offset, result.Match)
}
```

## Memory Characteristics

| Aspect | Behavior |
|--------|----------|
| Memory usage | Constant (~BufferSize) regardless of input size |
| Allocations | One buffer allocation, results reused |
| Large files | Process GB+ files without loading into memory |
| Network streams | Works with any io.Reader |

## When to Use Streaming

| Scenario | Use Streaming | Use In-Memory |
|----------|---------------|---------------|
| File > available RAM | Yes | No |
| Network streams | Yes | No (must buffer) |
| Small strings (<1MB) | No (overhead) | Yes |
| Need all matches at once | No | Yes |
| Processing one match at a time | Yes | Either |

## Error Handling

```go
err := pattern.FindReader(reader, cfg, callback)
if err != nil {
    if errors.Is(err, io.ErrUnexpectedEOF) {
        // Handle truncated input
    }
    // Handle other io errors
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "os"
    "github.com/KromDaniel/regengo/stream"
)

func main() {
    file, err := os.Open("access.log")
    if err != nil {
        panic(err)
    }
    defer file.Close()

    cfg := stream.Config{
        BufferSize: 2 * 1024 * 1024, // 2MB chunks
    }

    var matchCount int64
    err = CompiledLogPattern.FindReader(file, cfg, func(m stream.Match[*LogPatternBytesResult]) bool {
        matchCount++

        // Process match
        fmt.Printf("Match #%d at offset %d: %s\n",
            matchCount, m.StreamOffset, m.Result.Match)

        return true // continue
    })

    if err != nil {
        panic(err)
    }

    fmt.Printf("Total matches: %d\n", matchCount)
}
```
