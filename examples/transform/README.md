# Transform API Example

This example demonstrates the Transform API for streaming transformations.

## Running the Example

```bash
# Generate the email pattern
go run generate.go

# Run the example
go run main.go email.go
```

## What This Demonstrates

1. **ReplaceReader** - Simple template-based replacement
2. **ReplaceReader with captures** - Using `$user` and `$domain` in templates
3. **SelectReader** - Extract only matching content
4. **SelectReader with predicate** - Filter matches by domain
5. **RejectReader** - Remove matching content
6. **NewTransformReader** - Custom multi-emit transformation
7. **Conditional transformation** - Different output per match
8. **Chaining** - Combine LineFilter with ReplaceReader

## Key Concepts

### io.Reader Composition

All transform methods return `io.Reader`, enabling standard Go composition:

```go
var r io.Reader = file
r = CompiledEmail.ReplaceReader(r, "[EMAIL]")
r = CompiledIP.ReplaceReader(r, "[IP]")
r = stream.LineFilter(r, filterFunc)
io.Copy(os.Stdout, r)
```

### Emit Callback

`NewTransformReader` gives full control via the emit callback:

- Call `emit(data)` zero times to drop the match
- Call `emit(data)` once to replace the match
- Call `emit(data)` multiple times for 1-to-N expansion

### Template Syntax

| Syntax | Description |
|--------|-------------|
| `$0` | Full match |
| `$1`, `$2` | Capture by index |
| `$name` | Capture by name |
| `$$` | Literal `$` |

## Expected Output

```
=== Original Input ===
Server logs from 2024-12-01:
DEBUG: Starting server
INFO: User john@example.com logged in
ERROR: Failed connection from admin@internal.corp
INFO: User jane@company.org performed action
DEBUG: Heartbeat check
INFO: User john@example.com logged out

=== 1. ReplaceReader: Mask all emails ===
Server logs from 2024-12-01:
DEBUG: Starting server
INFO: User [REDACTED] logged in
ERROR: Failed connection from [REDACTED]
INFO: User [REDACTED] performed action
DEBUG: Heartbeat check
INFO: User [REDACTED] logged out

=== 2. ReplaceReader: Format emails ===
Server logs from 2024-12-01:
DEBUG: Starting server
INFO: User <john AT example.com> logged in
ERROR: Failed connection from <admin AT internal.corp>
INFO: User <jane AT company.org> performed action
DEBUG: Heartbeat check
INFO: User <john AT example.com> logged out

... (additional examples)
```
