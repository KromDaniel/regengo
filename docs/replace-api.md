# Replace API

The Replace API provides high-performance string replacement with regex capture group support. It offers both runtime flexibility and compile-time optimization for maximum performance.

## Table of Contents

- [Overview](#overview)
- [Template Syntax](#template-syntax)
- [Runtime Replace](#runtime-replace)
- [Compiled Templates](#compiled-templates)
- [Pre-compiled Replace](#pre-compiled-replace)
- [Zero-Allocation Patterns](#zero-allocation-patterns)
- [Performance](#performance)
- [Examples](#examples)
- [Migration from stdlib](#migration-from-stdlib)

## Overview

Regengo generates three types of Replace methods:

**Compile-time safety:** Pre-compiled replacer templates are validated during code generation. References to non-existent capture groups (e.g., `$invalid` or `$3` when only 2 groups exist) cause a compile error—you'll know immediately if a template is invalid, not at runtime.

| Type | Methods | Template Validation | Error Handling | Best For |
|------|---------|---------------------|----------------|----------|
| **Runtime** | `ReplaceAllString`, `ReplaceFirstString` | Every call | Panics | Dynamic templates, flexibility |
| **Compiled** | `CompileReplaceTemplate` → `ReplaceAllString` | Once at startup | Returns error | Config-driven templates, graceful error handling |
| **Pre-compiled** | `ReplaceAllString0`, `ReplaceAllString1`, ... | At code generation | Compile error | Type safety, no runtime surprises |

**Performance note:** Compiled and pre-compiled templates have nearly identical performance—both skip template parsing. The main benefit of pre-compiled is **compile-time type safety**: invalid templates cause build failures, not runtime errors.

All types support:
- Full match reference (`$0`)
- Indexed captures (`$1`, `$2`, ...)
- Named captures (`$name`, `${name}`)
- Literal dollar signs (`$$`)

## Template Syntax

The template syntax is **fully compatible with Go's stdlib `regexp`**. Templates that work with `regexp.ReplaceAllString` will work identically with regengo.

| Syntax | Description | Example |
|--------|-------------|---------|
| `$0` | Full match | `"[$0]"` → `"[full match]"` |
| `$1`, `$2` | Capture by index (1-based) | `"$1-$2"` → `"first-second"` |
| `$name` | Capture by name | `"$user@$domain"` → `"alice@example"` |
| `${name}` | Explicit boundary | `"${user}name"` → `"alicename"` |
| `${1}` | Explicit index | `"${1}st"` → `"1st"` |
| `$$` | Literal `$` | `"$$100"` → `"$100"` |

### Template Examples

```go
// Pattern: (?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)
// Input: "alice@example.com"

"$user@REDACTED.$tld"     → "alice@REDACTED.com"
"[$0]"                     → "[alice@example.com]"
"$1 at $2 dot $3"         → "alice at example dot com"
"Email: ${user}"          → "Email: alice"
"Cost: $$50"              → "Cost: $50"
```

## Runtime Replace

Runtime Replace methods accept a template string at call time. The template is parsed on each call, providing flexibility at a small performance cost.

**Error handling:** Invalid templates cause a panic, consistent with Go's stdlib `regexp` behavior. This ensures template errors are caught immediately rather than silently ignored.

### Generated Methods

```go
// Replace all matches
func (Pattern) ReplaceAllString(input, template string) string
func (Pattern) ReplaceAllBytes(input []byte, template string) []byte
func (Pattern) ReplaceAllBytesAppend(input []byte, template string, buf []byte) []byte

// Replace first match only
func (Pattern) ReplaceFirstString(input, template string) string
func (Pattern) ReplaceFirstBytes(input []byte, template string) []byte
```

### Usage

```go
// Generate with captures
// regengo -pattern '(?P<user>\w+)@(?P<domain>\w+)' -name Email -output email.go

input := "Contact alice@example or bob@test"

// Replace all matches
result := CompiledEmail.ReplaceAllString(input, "$user@HIDDEN")
// Result: "Contact alice@HIDDEN or bob@HIDDEN"

// Replace first match only
result := CompiledEmail.ReplaceFirstString(input, "[$0]")
// Result: "Contact [alice@example] or bob@test"

// Dynamic template from user input
template := getUserTemplate() // e.g., "$user at $domain"
result := CompiledEmail.ReplaceAllString(input, template)
```

## Compiled Templates

Compiled Templates provide a middle ground between runtime and pre-compiled approaches. The template is parsed once at runtime with proper error handling, then reused for multiple replacements without re-parsing.

**Use Cases:**
- Templates loaded from configuration files
- Templates stored in databases
- User-provided templates that need validation before use
- High-throughput scenarios where the same dynamic template is used repeatedly

### Generated Types and Methods

```go
// Generated struct to hold a compiled template
type EmailReplaceTemplate struct {
    // internal fields
}

// Compile a template with validation (returns error, doesn't panic)
func (Email) CompileReplaceTemplate(template string) (*EmailReplaceTemplate, error)

// Replace methods on the compiled template
func (t *EmailReplaceTemplate) ReplaceAllString(input string) string
func (t *EmailReplaceTemplate) ReplaceAllBytes(input []byte) []byte
func (t *EmailReplaceTemplate) ReplaceAllBytesAppend(input []byte, buf []byte) []byte
func (t *EmailReplaceTemplate) ReplaceFirstString(input string) string
func (t *EmailReplaceTemplate) ReplaceFirstBytes(input []byte) []byte

// Access to the original template string
func (t *EmailReplaceTemplate) String() string
```

### Usage

```go
// Load template from config (might be invalid)
templateStr := config.Get("email.redaction.template")

// Compile with error handling - doesn't panic on invalid template
tmpl, err := CompiledEmail.CompileReplaceTemplate(templateStr)
if err != nil {
    log.Fatalf("Invalid email redaction template: %v", err)
}

// Use many times without re-parsing
for _, input := range inputs {
    result := tmpl.ReplaceAllString(input)
    process(result)
}

// Zero-allocation variant
buf := make([]byte, 0, 4096)
for _, input := range inputBytes {
    buf = tmpl.ReplaceAllBytesAppend(input, buf)
    process(buf)
    buf = buf[:0]
}
```

### Error Handling

`CompileReplaceTemplate` returns an error for:
- Malformed template syntax (e.g., `${unclosed`)
- Invalid capture index (e.g., `$3` when pattern has only 2 captures)
- Invalid capture name (e.g., `$invalid` when no such named group exists)

```go
tmpl, err := CompiledEmail.CompileReplaceTemplate("$invalid")
// err: capture group "invalid" not found in pattern

tmpl, err := CompiledEmail.CompileReplaceTemplate("$5")
// err: capture group 5 out of range (pattern has 2 capture groups)

tmpl, err := CompiledEmail.CompileReplaceTemplate("${unclosed")
// err: at position 0: unclosed ${
```

### When to Use Compiled Templates

| Scenario | Recommended Approach |
|----------|---------------------|
| Template known at build time, want type safety | Pre-compiled (`-replacer` flag) |
| Template from config/DB, need error handling | Compiled (`CompileReplaceTemplate`) |
| Template from user input, one-time use | Runtime (`ReplaceAllString`) |
| Need to validate template before use | Compiled (`CompileReplaceTemplate`) |

## Pre-compiled Replace

Pre-compiled Replace methods have templates specified at code generation time. Invalid templates cause **build failures**, not runtime errors—guaranteeing type safety and eliminating runtime surprises.

### CLI Usage

```bash
regengo -pattern '(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)' \
        -name Email \
        -replacer '$user@REDACTED.$tld' \
        -replacer '[$0]' \
        -replacer 'EMAIL_REMOVED' \
        -output email.go
```

### Library Usage

```go
import "github.com/KromDaniel/regengo"

err := regengo.Compile(regengo.Options{
    Pattern:    `(?P<user>\w+)@(?P<domain>\w+)\.(?P<tld>\w+)`,
    Name:       "Email",
    OutputFile: "email.go",
    Package:    "main",
    Replacers:  []string{
        "$user@REDACTED.$tld",  // Generates ReplaceAllString0
        "[$0]",                  // Generates ReplaceAllString1
        "EMAIL_REMOVED",         // Generates ReplaceAllString2
    },
})
```

### Generated Methods

For each replacer template, these methods are generated:

```go
// For Replacers[0]: "$user@REDACTED.$tld"
func (Email) ReplaceAllString0(input string) string
func (Email) ReplaceAllBytes0(input []byte) []byte
func (Email) ReplaceAllBytesAppend0(input []byte, buf []byte) []byte
func (Email) ReplaceFirstString0(input string) string
func (Email) ReplaceFirstBytes0(input []byte) []byte

// For Replacers[1]: "[$0]"
func (Email) ReplaceAllString1(input string) string
// ... etc
```

### Usage

```go
input := "Contact alice@example.com"

// Use pre-compiled replacer 0
result := CompiledEmail.ReplaceAllString0(input)
// Result: "Contact alice@REDACTED.com"

// Use pre-compiled replacer 1
result := CompiledEmail.ReplaceAllString1(input)
// Result: "Contact [alice@example.com]"
```

## Zero-Allocation Patterns

The `Append` variants allow buffer reuse for zero-allocation hot paths:

```go
// Pre-allocate buffer once
buf := make([]byte, 0, 4096)

// Process many inputs without allocation
for _, input := range inputs {
    buf = CompiledEmail.ReplaceAllBytesAppend0(input, buf)
    process(buf)
    buf = buf[:0] // Reset for next iteration
}
```

### When to Use

| Method | Allocations | Use Case |
|--------|-------------|----------|
| `ReplaceAllString` | New string each call | Simple usage, readability |
| `ReplaceAllBytes` | New slice each call | Byte slice workflows |
| `ReplaceAllBytesAppend` | Zero (with pre-sized buf) | High-throughput processing |

## Performance

Regengo Replace methods are significantly faster than stdlib:

| Method | vs stdlib | Notes |
|--------|-----------|-------|
| Compiled / Pre-compiled | ~2-3x faster | Template parsed once or at build time |
| Runtime | ~1.5-2x faster | Template parsed each call |
| Zero-alloc (BytesAppend) | ~3-4x faster | Buffer reuse eliminates GC pressure |

### Benchmark Results

```
BenchmarkCompileTemplateComparison/runtime-12      490 ns/op   8 allocs/op
BenchmarkCompileTemplateComparison/compiled-12     390 ns/op   4 allocs/op
BenchmarkCompileTemplateComparison/precompiled-12  387 ns/op   4 allocs/op
```

**Key insight:** Compiled and pre-compiled have nearly identical performance. Choose based on your needs:
- **Pre-compiled**: Compile-time type safety, no runtime template errors possible
- **Compiled**: Runtime flexibility with graceful error handling

### Optimization Tips

1. **Use compiled or pre-compiled** - Both skip template parsing (~20% faster than runtime)
2. **Use Append variants for throughput** - Pre-allocate buffers for hot paths
3. **Literal-only templates are fastest** - `"REDACTED"` skips capture extraction
4. **Full-match templates are fast** - `"[$0]"` only needs match bounds

## Examples

### Email Masking

```go
// Pattern: (?P<user>[\w.+-]+)@(?P<domain>[\w.-]+)\.(?P<tld>\w+)
// Replacer: "$user@REDACTED.$tld"

input := "Contact support@example.com for help"
result := CompiledEmail.ReplaceAllString0(input)
// Result: "Contact support@REDACTED.com for help"
```

### Log Redaction

```go
// Pattern: (?P<key>password|secret|token)=(?P<value>\S+)
// Replacer: "$key=***"

log := "user=alice password=secret123 token=abc"
result := CompiledSecret.ReplaceAllString0(log)
// Result: "user=alice password=*** token=***"
```

### URL Rewriting

```go
// Pattern: https?://(?P<domain>[^/]+)(?P<path>/\S*)
// Replacer: "https://cdn.example.com$path"

html := `<img src="http://old.site/img.png">`
result := CompiledURL.ReplaceAllString0(html)
// Result: `<img src="https://cdn.example.com/img.png">`
```

### High-Throughput Processing

```go
// Process millions of log lines with zero allocation
buf := make([]byte, 0, 8192)
scanner := bufio.NewScanner(file)

for scanner.Scan() {
    line := scanner.Bytes()
    buf = CompiledPattern.ReplaceAllBytesAppend0(line, buf)
    writer.Write(buf)
    buf = buf[:0]
}
```

## Migration from stdlib

### Import Changes

```go
// Before
import "regexp"

// After
import "github.com/KromDaniel/regengo"
// Plus your generated package
```

### Template Syntax

| stdlib | regengo | Notes |
|--------|---------|-------|
| `${1}` | `$1` or `${1}` | Both work in regengo |
| `${name}` | `$name` or `${name}` | Both work in regengo |
| `$1` | `$1` | Same |
| `$$` | `$$` | Same |

### Code Changes

```go
// Before (stdlib)
re := regexp.MustCompile(`(?P<user>\w+)@(?P<domain>\w+)`)
result := re.ReplaceAllString(input, "${user}@HIDDEN")

// After (regengo) - Runtime
result := CompiledPattern.ReplaceAllString(input, "$user@HIDDEN")

// After (regengo) - Pre-compiled (faster)
// Generate with: -replacer '$user@HIDDEN'
result := CompiledPattern.ReplaceAllString0(input)
```

### Key Differences

1. **Compile-time validation** - Invalid templates caught during code generation
2. **Named capture shortcuts** - Use `$name` instead of `${name}`
3. **No regex compilation at runtime** - Pattern compiled at code generation
4. **Pre-compiled option** - Templates can be optimized at generation time
