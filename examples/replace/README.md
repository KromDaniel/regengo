# Replace API Example

This example demonstrates the Replace API for string replacement with capture group references.

## Running the Example

```bash
# Generate the pattern code
go run generate.go

# Run the example
go run main.go email.go
```

## Expected Output

```
Input: Contact support@example.com or sales@company.org for help

=== Pre-compiled Replacers ===
Mask domain: Contact support@REDACTED.com or sales@REDACTED.org for help
Full redact: Contact [EMAIL REMOVED] or [EMAIL REMOVED] for help
Partial mask: Contact support@***.com or sales@***.org for help

=== Runtime Replacer ===
Custom format: Contact [support AT example] or [sales AT company] for help
Wrap match: Contact <support@example.com> or <sales@company.org> for help
First only: Contact [FIRST EMAIL] or sales@company.org for help

=== Zero-Allocation ===
  user1@a.com -> user1@REDACTED.com
  user2@b.org -> user2@REDACTED.org
  user3@c.net -> user3@REDACTED.net
```

## Key Concepts

### Pre-compiled vs Runtime

| Type | Method | Speed | Flexibility |
|------|--------|-------|-------------|
| Pre-compiled | `ReplaceAllString0()` | Fastest | Template fixed at generation |
| Runtime | `ReplaceAllString(template)` | Fast | Any template at runtime |

### Template Syntax

- `$0` - Full match
- `$1`, `$2` - Capture by index
- `$name` - Capture by name
- `$$` - Literal dollar sign

### Zero-Allocation

Use `ReplaceAllBytesAppend` with a pre-allocated buffer for high-throughput scenarios:

```go
buf := make([]byte, 0, 4096)
for _, input := range inputs {
    buf = Pattern.ReplaceAllBytesAppend0(input, buf)
    process(buf)
    buf = buf[:0]
}
```
