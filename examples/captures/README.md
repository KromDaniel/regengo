# Capture Groups Examples

This directory contains examples demonstrating the capture groups feature in regengo.

## What are Capture Groups?

Capture groups allow you to extract specific parts of a matched string. They can be:

- **Named groups**: `(?P<name>...)` - generates fields with the specified name
- **Indexed groups**: `(...)` - generates fields like `Match1`, `Match2`, etc.

## Running the Examples

### Email Pattern with Named Groups

```bash
go run capture_groups.go
```

This example demonstrates:

- Pattern: `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`
- Extracts username, domain, and TLD from email addresses
- Result struct has `User`, `Domain`, and `Tld` fields

### URL Pattern with Mixed Groups

```bash
go run url_captures.go
```

This example demonstrates:

- Pattern: `(?P<protocol>https?)://(\w+)\.(?P<domain>\w+)\.(\w+)`
- Extracts protocol (named), subdomain (indexed), domain (named), and TLD (indexed)
- Result struct has `Protocol`, `Match2`, `Domain`, and `Match4` fields

## Generated Files

The examples use pre-generated matcher files in `test/`:

- `test/EmailCapture.go` - Email matcher with capture groups
- `test/URLCapture.go` - URL matcher with mixed capture groups

## Generating Your Own

To generate a matcher with capture groups:

```bash
./bin/regengo -pattern '(?P<name>\w+)' -name 'MyMatcher' -output 'output.go' -captures
```

The `-captures` flag enables capture group support, which:

1. Extracts named and indexed groups from the pattern
2. Generates a result struct with appropriate fields
3. Creates `FindString` and `FindBytes` functions that return the struct
4. Optimizes the matching logic to track capture positions

## Result Structure

All capture matchers return a struct with:

- `Match` - The full matched string
- `Start` - Start position in the input
- `End` - End position in the input
- Named group fields (e.g., `User`, `Domain`)
- Indexed group fields (e.g., `Match1`, `Match2`)

Example:

```go
result, found := EmailCaptureFindString("user@example.com")
if found {
    fmt.Printf("User: %s, Domain: %s, TLD: %s\n",
        result.User, result.Domain, result.Tld)
}
```
