# Transform API

The Transform API enables **streaming transformations** of pattern matches. It processes data incrementally with constant memory usage, returning an `io.Reader` that can be seamlessly composed with standard Go I/O utilities.

## Quick Reference

All methods are generated on the compiled pattern struct (e.g., `CompiledEmail`).

| Method | Description | Use Case |
|--------|-------------|----------|
| **`ReplaceReader`** | Replaces matches with a template string | Simple redaction, data masking |
| **`SelectReader`** | Outputs **only** matches (discards non-matches) | Extracting data (emails, URLs) |
| **`RejectReader`** | Outputs everything **except** matches | Removing sensitive data |
| **`NewTransformReader`** | Full control (callback-based) | Complex logic, conditional emit, 1-to-N expansion |

## Usage Examples

### 1. Replace (Redaction)

```go
file, _ := os.Open("server.log")

// Simple string replacement
r := CompiledEmail.ReplaceReader(file, "[REDACTED]")
// Input:  "Contact user@example.com for help"
// Output: "Contact [REDACTED] for help"

// Template replacement with capture groups
// $1 = first capture group, $name = named capture
r2 := CompiledEmail.ReplaceReader(file, "[$name at $domain]")
// Input:  "Contact john@example.com for help"
// Output: "Contact [john at example.com] for help"
```

### 2. Extract (Select)

Keeps only the text that matches the pattern. Non-matching text is discarded.

```go
// Extract all emails
r := CompiledEmail.SelectReader(file, func(m *EmailBytesResult) bool {
    return true // Keep all
})
// Input:  "Contact john@a.com and jane@b.org"
// Output: "john@a.comjane@b.org"

// Extract only .com emails
r = CompiledEmail.SelectReader(file, func(m *EmailBytesResult) bool {
    return bytes.HasSuffix(m.Domain, []byte(".com"))
})
// Input:  "john@a.com and jane@b.org"
// Output: "john@a.com"
```

### 3. Filter (Reject)

Removes text that matches the pattern. Non-matching text is preserved.

```go
// Remove all emails
r := CompiledEmail.RejectReader(file, func(m *EmailBytesResult) bool {
    return true // Reject all
})
// Input:  "Contact user@example.com for help"
// Output: "Contact  for help"

// Remove only "spam.com" emails
r = CompiledEmail.RejectReader(file, func(m *EmailBytesResult) bool {
    return bytes.Equal(m.Domain, []byte("spam.com"))
})
// Input:  "good@keep.com vs bad@spam.com"
// Output: "good@keep.com vs "
```

### 4. Pipeline Composition

Since all methods return `io.Reader`, you can chain them to perform multiple transformations in a single pass.

```go
func ProcessLog(input io.Reader) io.Reader {
    var r io.Reader = input

    // 1. Mask Emails
    r = CompiledEmail.ReplaceReader(r, "[EMAIL]")

    // 2. Mask IPs
    r = CompiledIP.ReplaceReader(r, "[IP]")

    // 3. Filter out DEBUG lines (using generic helper)
    r = stream.LineFilter(r, func(line []byte) bool {
        return !bytes.HasPrefix(line, []byte("DEBUG"))
    })

    return r
}

// Input:
// DEBUG: starting up
// INFO: user@test.com logged in from 192.168.1.1

// Output:
// INFO: [EMAIL] logged in from [IP]
```

## Advanced Usage

### NewTransformReader (Low-Level)

For maximum control, use `NewTransformReader`. The `emit` callback allows you to output zero, one, or multiple segments per match.

```go
r := CompiledEmail.NewTransformReader(input, stream.DefaultTransformConfig(),
    func(m *EmailBytesResult, emit func([]byte)) {
        // Example: Expand match into XML
        emit([]byte("<email>"))
        emit(m.Match)
        emit([]byte("</email>"))
    })
// Input:  "user@example.com"
// Output: "<email>user@example.com</email>"
```

### Configuration & Context

Use `stream.TransformConfig` to control buffering or enable cancellation.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cfg := stream.TransformConfig{
    Context:         ctx,        // Enable cancellation
    BufferSize:      256 * 1024, // 256KB input buffer
    MaxOutputBuffer: 0,          // Unlimited output buffer
}

r := CompiledEmail.NewTransformReader(file, cfg, onMatchCallback)
```

## Line Helpers

Generic utilities for line-based processing (works on any `io.Reader`).

*   **`stream.LineFilter(r, pred)`**: Keep lines where `pred` returns true.
*   **`stream.LineTransform(r, fn)`**: Transform each line byte-by-byte.

```go
// Keep only ERROR lines
r = stream.LineFilter(r, func(line []byte) bool {
    return bytes.Contains(line, []byte("ERROR"))
})
```

## Performance

*   **Throughput:** ~90 MB/s for typical replacement operations.
*   **Memory:** Constant memory usage regardless of input size.
*   **Pooling:** Internal buffers are pooled and reused.

For benchmarks, see [`benchmarks/streams/`](../benchmarks/streams/).
