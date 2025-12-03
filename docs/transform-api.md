# Transform API

The Transform API enables streaming transformations of pattern matches. Unlike batch operations that load entire files into memory, transform readers process data incrementally with constant memory usage.

## Overview

Transform readers implement `io.Reader`, enabling standard Go composition:

```go
// Chain transformations with io.Copy, io.Pipe, http handlers, etc.
file, _ := os.Open("data.log")
r := CompiledEmail.ReplaceReader(file, "[REDACTED]")
io.Copy(os.Stdout, r)
```

**Key Features:**
- **Streaming**: Process files of any size with constant memory
- **Composable**: Chain multiple transformers using standard `io.Reader` composition
- **Efficient**: Uses buffer pooling and minimizes allocations
- **Type-safe**: Access capture groups via typed struct fields

## Quick Start

### Replace Matches

```go
// Replace all emails with "[EMAIL]"
file, _ := os.Open("logs.txt")
r := CompiledEmail.ReplaceReader(file, "[EMAIL]")
io.Copy(os.Stdout, r)

// Use capture groups in replacement
r := CompiledEmail.ReplaceReader(file, "[$name at $domain]")
// "john@example.com" -> "[john at example.com]"
```

### Extract Matches

```go
// Output only email matches (no surrounding text)
r := CompiledEmail.SelectReader(file, func(m *EmailBytesResult) bool {
    return true // keep all matches
})

// Filter by domain
r := CompiledEmail.SelectReader(file, func(m *EmailBytesResult) bool {
    return bytes.HasSuffix(m.Domain, []byte(".com"))
})
```

### Remove Matches

```go
// Remove all emails from output (keep surrounding text)
r := CompiledEmail.RejectReader(file, func(m *EmailBytesResult) bool {
    return true // reject all matches
})

// Remove only specific emails
r := CompiledEmail.RejectReader(file, func(m *EmailBytesResult) bool {
    return bytes.Equal(m.Domain, []byte("spam.com"))
})
```

## API Reference

### NewTransformReader

Full control over match transformation with emit callback.

```go
func (Pattern) NewTransformReader(
    r io.Reader,
    cfg stream.TransformConfig,
    onMatch func(match *PatternBytesResult, emit func([]byte)),
) io.Reader
```

**Parameters:**
- `r`: Source reader
- `cfg`: Configuration (buffer size, context cancellation)
- `onMatch`: Called for each match; call `emit` zero or more times to produce output

**Behavior:**
- Non-matching text passes through unchanged
- Each match invokes `onMatch`; caller controls output via `emit`
- Not calling `emit` drops the match (filter behavior)
- Calling `emit` multiple times expands the match (1-to-N)

**Example:**
```go
r := CompiledEmail.NewTransformReader(input, stream.DefaultTransformConfig(),
    func(m *EmailBytesResult, emit func([]byte)) {
        // Transform: "john@example.com" -> "<john AT example.com>"
        emit([]byte("<"))
        emit(m.Name)
        emit([]byte(" AT "))
        emit(m.Domain)
        emit([]byte(">"))
    })
```

### ReplaceReader

Replace matches using a template string.

```go
func (Pattern) ReplaceReader(r io.Reader, template string) io.Reader
```

**Template Syntax:**
| Syntax | Description |
|--------|-------------|
| `$0` | Full match |
| `$1`, `$2` | Capture by index |
| `$name` | Capture by name |
| `$$` | Literal `$` |

**Example:**
```go
// Pattern: (?P<user>\w+)@(?P<domain>\w+\.\w+)
r := CompiledEmail.ReplaceReader(file, "[$user at $domain]")
// Input:  "Contact john@example.com"
// Output: "Contact [john at example.com]"
```

### SelectReader

Output only matches that satisfy a predicate.

```go
func (Pattern) SelectReader(r io.Reader, pred func(*PatternBytesResult) bool) io.Reader
```

Non-matching text is discarded. Only matches where `pred` returns `true` are output.

**Example:**
```go
// Extract all .com emails
r := CompiledEmail.SelectReader(file, func(m *EmailBytesResult) bool {
    return bytes.HasSuffix(m.Domain, []byte(".com"))
})
// Input:  "john@example.com and jane@example.org"
// Output: "john@example.com"
```

### RejectReader

Remove matches that satisfy a predicate.

```go
func (Pattern) RejectReader(r io.Reader, pred func(*PatternBytesResult) bool) io.Reader
```

Non-matching text passes through. Matches where `pred` returns `true` are removed.

**Example:**
```go
// Remove spam domain emails
r := CompiledEmail.RejectReader(file, func(m *EmailBytesResult) bool {
    return bytes.Equal(m.Domain, []byte("spam.com"))
})
// Input:  "good@example.com and bad@spam.com"
// Output: "good@example.com and "
```

## Composition

Chain multiple transformations using standard Go patterns:

```go
func ProcessLog(input io.Reader) io.Reader {
    var r io.Reader = input

    // 1. Mask sensitive data
    r = CompiledEmail.ReplaceReader(r, "[EMAIL]")
    r = CompiledIP.ReplaceReader(r, "[IP]")
    r = CompiledSSN.ReplaceReader(r, "[SSN]")

    // 2. Filter debug lines
    r = stream.LineFilter(r, func(line []byte) bool {
        return !bytes.HasPrefix(line, []byte("DEBUG"))
    })

    // 3. Add timestamps
    r = stream.LineTransform(r, func(line []byte) []byte {
        return append([]byte(time.Now().Format(time.RFC3339)+" "), line...)
    })

    return r
}

// Usage
file, _ := os.Open("app.log")
processed := ProcessLog(file)
io.Copy(os.Stdout, processed)
```

## Line Helpers

Generic line-based utilities for any `io.Reader`:

### LineFilter

```go
func LineFilter(r io.Reader, pred func(line []byte) bool) io.Reader
```

Output only lines where predicate returns true.

```go
// Keep ERROR lines only
r := stream.LineFilter(file, func(line []byte) bool {
    return bytes.Contains(line, []byte("ERROR"))
})
```

### LineTransform

```go
func LineTransform(r io.Reader, fn func(line []byte) []byte) io.Reader
```

Transform each line. Return `nil` to drop the line.

```go
// Add line numbers
lineNum := 0
r := stream.LineTransform(file, func(line []byte) []byte {
    lineNum++
    return append([]byte(fmt.Sprintf("%4d: ", lineNum)), line...)
})
```

## Configuration

### TransformConfig

```go
type TransformConfig struct {
    Config  // Embedded base config

    // MaxOutputBuffer limits internal output buffer size.
    // Default: 0 (unlimited, grows as needed)
    MaxOutputBuffer int

    // Context for cancellation support.
    // Default: nil (no cancellation)
    Context context.Context
}
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

cfg := stream.DefaultTransformConfig()
cfg.Context = ctx

r := CompiledEmail.NewTransformReader(file, cfg,
    func(m *EmailBytesResult, emit func([]byte)) {
        emit([]byte("[EMAIL]"))
    })

_, err := io.Copy(os.Stdout, r)
if err == context.DeadlineExceeded {
    log.Println("Processing timed out")
}
```

### Buffer Size Tuning

```go
cfg := stream.DefaultTransformConfig()
cfg.BufferSize = 256 * 1024  // 256KB input buffer
cfg.MaxLeftover = 128 * 1024 // Max bytes kept for boundary matching

r := CompiledEmail.NewTransformReader(file, cfg, onMatch)
```

## Performance

### Streaming Benchmarks

Benchmarks are available in [`benchmarks/streams/`](../benchmarks/streams/):

| Benchmark | Throughput | Notes |
|-----------|------------|-------|
| ReplaceReader (1KB) | ~40 MB/s | Small input overhead |
| ReplaceReader (1MB) | ~90 MB/s | Steady-state throughput |
| ReplaceReader (10MB) | ~95 MB/s | Large input |
| Pipeline (2-stage) | ~70 MB/s | Email + IP replacement |
| Pipeline (5-stage) | ~40 MB/s | Multi-stage pipeline |

Run benchmarks:
```bash
go test -bench=. ./benchmarks/streams/...
```

**Note:** These are baseline benchmarks for tracking regressions. stdlib `regexp` doesn't have streaming transform methods, so direct comparison isn't possible. For in-memory Replace benchmarks (where stdlib comparison is valid), see [`benchmarks/curated/`](../benchmarks/curated/).

### High-Throughput: Pooled Transformer

For scenarios with many short-lived transformers, use pooled buffers:

```go
// In stream package (internal use)
tr := stream.NewTransformerPooled(source, cfg, processor, onMatch)
defer tr.Close() // MUST call Close to return buffers to pool
```

Note: The generated `ReplaceReader`, `SelectReader`, and `RejectReader` methods use the default (non-pooled) transformer. For high-throughput scenarios with many concurrent streams, use `NewTransformReader` with custom configuration.

## Use Cases

### Log Redaction

```go
// Mask PII in logs before forwarding
func RedactLogs(input io.Reader, output io.Writer) error {
    var r io.Reader = input
    r = CompiledEmail.ReplaceReader(r, "[EMAIL]")
    r = CompiledPhone.ReplaceReader(r, "[PHONE]")
    r = CompiledSSN.ReplaceReader(r, "[SSN]")
    _, err := io.Copy(output, r)
    return err
}
```

### Data Extraction

```go
// Extract all URLs from HTML
func ExtractURLs(html io.Reader) ([]string, error) {
    r := CompiledURL.SelectReader(html, func(m *URLBytesResult) bool {
        return true
    })

    data, err := io.ReadAll(r)
    if err != nil {
        return nil, err
    }

    // Split by URL (each match is separated by newline or original delimiter)
    return strings.Fields(string(data)), nil
}
```

### HTTP Middleware

```go
// Middleware that masks emails in responses
func EmailMaskingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        pr, pw := io.Pipe()

        go func() {
            defer pw.Close()
            // Capture response
            rec := &responseRecorder{ResponseWriter: w, body: pw}
            next.ServeHTTP(rec, r)
        }()

        // Stream masked response
        masked := CompiledEmail.ReplaceReader(pr, "[EMAIL]")
        io.Copy(w, masked)
    })
}
```

## Error Handling

Transform readers propagate errors from the source:

```go
r := CompiledEmail.ReplaceReader(source, "[EMAIL]")

buf := make([]byte, 4096)
for {
    n, err := r.Read(buf)
    if n > 0 {
        process(buf[:n])
    }
    if err == io.EOF {
        break // Normal completion
    }
    if err != nil {
        log.Printf("Read error: %v", err)
        break
    }
}
```

With context cancellation:
```go
cfg := stream.DefaultTransformConfig()
cfg.Context = ctx

r := CompiledEmail.NewTransformReader(source, cfg, onMatch)
_, err := io.Copy(dest, r)
if errors.Is(err, context.Canceled) {
    // Context was canceled
}
```

## Best Practices

1. **Use ReplaceReader for simple replacements** - It handles template parsing automatically

2. **Use NewTransformReader for complex logic** - When you need conditional processing, multi-emit, or access to capture group data

3. **Chain transformers for multiple patterns** - Each transformer adds minimal overhead

4. **Consider buffer size for large files** - Larger buffers reduce syscalls but use more memory

5. **Use context for long-running operations** - Enable cancellation for user-facing applications

6. **Close pooled transformers** - When using `NewTransformerPooled`, always call `Close()`
