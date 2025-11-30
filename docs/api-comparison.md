# API Comparison: Regengo vs Go stdlib regexp

This document provides a comprehensive comparison between regengo's generated API and Go's standard library `regexp` package.

## Key Differences

| Aspect | stdlib `regexp` | regengo |
|--------|-----------------|---------|
| **Result type** | `[]string` / `[][]string` | Typed struct with named fields |
| **Capture access** | By index: `match[1]` | By name: `result.Year` |
| **Type safety** | None (all strings) | Compile-time field access |
| **Streaming** | First match only | All matches with callback |
| **Memory** | Allocates per call | Zero-alloc `Reuse` variants |

## Generated Types

For a pattern like `(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})` with name `Date`:

```go
// Main matcher type
type Date struct{}
var CompiledDate = Date{}

// String result with typed fields
type DateResult struct {
    Match string   // Full match
    Year  string   // Named capture group
    Month string   // Named capture group
    Day   string   // Named capture group
}

// Bytes result (for []byte input and streaming)
type DateBytesResult struct {
    Match []byte
    Year  []byte
    Month []byte
    Day   []byte
}

// Match length constants
const DateMinMatchLen = 10
const DateMaxMatchLen = 10  // -1 if unbounded
```

## Method Reference

### Matching (bool result)

| stdlib `regexp` | regengo | Notes |
|-----------------|---------|-------|
| `re.MatchString(s)` | `CompiledDate.MatchString(s)` | Identical signature |
| `re.Match(b)` | `CompiledDate.MatchBytes(b)` | Identical signature |

**Example:**

```go
// stdlib
re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
if re.MatchString("2024-12-25") { ... }

// regengo
if CompiledDate.MatchString("2024-12-25") { ... }
```

---

### Finding First Match

| stdlib `regexp` | regengo | Return type |
|-----------------|---------|-------------|
| `re.FindString(s)` | `CompiledDate.FindString(s)` | `string` vs `(*DateResult, bool)` |
| `re.FindStringSubmatch(s)` | `CompiledDate.FindString(s)` | `[]string` vs `(*DateResult, bool)` |
| `re.Find(b)` | `CompiledDate.FindBytes(b)` | `[]byte` vs `(*DateBytesResult, bool)` |
| `re.FindSubmatch(b)` | `CompiledDate.FindBytes(b)` | `[][]byte` vs `(*DateBytesResult, bool)` |
| - | `CompiledDate.FindStringReuse(s, r)` | Zero-alloc with reuse |
| - | `CompiledDate.FindBytesReuse(b, r)` | Zero-alloc with reuse |

**Example - Accessing captures:**

```go
// stdlib - access by index, error-prone
re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
match := re.FindStringSubmatch("2024-12-25")
if match != nil {
    year := match[1]   // Must remember index
    month := match[2]  // Off-by-one errors possible
    day := match[3]
}

// regengo - typed struct, compile-time checked
result, ok := CompiledDate.FindString("2024-12-25")
if ok {
    year := result.Year   // IDE autocomplete
    month := result.Month // Compile-time errors if wrong
    day := result.Day
}
```

**Example - Zero-allocation reuse:**

```go
// regengo only - no stdlib equivalent
var reuse DateResult
for _, input := range largeDataset {
    result, ok := CompiledDate.FindStringReuse(input, &reuse)
    if ok {
        process(result.Year, result.Month, result.Day)
    }
}
```

---

### Finding All Matches

| stdlib `regexp` | regengo | Return type |
|-----------------|---------|-------------|
| `re.FindAllString(s, n)` | `CompiledDate.FindAllString(s, n)` | `[]string` vs `[]*DateResult` |
| `re.FindAllStringSubmatch(s, n)` | `CompiledDate.FindAllString(s, n)` | `[][]string` vs `[]*DateResult` |
| `re.FindAll(b, n)` | `CompiledDate.FindAllBytes(b, n)` | `[][]byte` vs `[]*DateBytesResult` |
| `re.FindAllSubmatch(b, n)` | `CompiledDate.FindAllBytes(b, n)` | `[][][]byte` vs `[]*DateBytesResult` |
| - | `CompiledDate.FindAllStringAppend(s, n, slice)` | Append to existing slice |
| - | `CompiledDate.FindAllBytesAppend(b, n, slice)` | Append to existing slice |

**Example - Finding all matches:**

```go
input := "Dates: 2024-01-15 and 2024-12-25"

// stdlib - nested slices, index-based access
re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
matches := re.FindAllStringSubmatch(input, -1)
for _, match := range matches {
    fmt.Printf("%s-%s-%s\n", match[1], match[2], match[3])
}
// Type: [][]string

// regengo - flat slice of typed structs
results := CompiledDate.FindAllString(input, -1)
for _, r := range results {
    fmt.Printf("%s-%s-%s\n", r.Year, r.Month, r.Day)
}
// Type: []*DateResult
```

**Example - Slice reuse for reduced allocations:**

```go
// regengo only - reuse slice across calls
var results []*DateResult
for _, input := range inputs {
    results = CompiledDate.FindAllStringAppend(input, -1, results[:0])
    for _, r := range results {
        process(r)
    }
}
```

---

### Streaming (io.Reader)

| stdlib `regexp` | regengo | Notes |
|-----------------|---------|-------|
| `re.FindReaderIndex(r)` | `CompiledDate.FindReader(r, cfg, cb)` | **First only** vs **all matches** |
| - | `CompiledDate.FindReaderCount(r, cfg)` | Count matches |
| - | `CompiledDate.FindReaderFirst(r, cfg)` | First match with result |

**Key difference:** stdlib's `FindReaderIndex` only finds the **first** match. Regengo finds **all matches** via callback.

**Example - Streaming all matches:**

```go
// stdlib - only finds FIRST match, returns indices only
re := regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)
loc := re.FindReaderIndex(reader)  // []int{start, end} or nil
// Cannot get subsequent matches without buffering entire stream!

// regengo - finds ALL matches with callback
err := CompiledDate.FindReader(reader, stream.Config{}, func(m stream.Match[*DateBytesResult]) bool {
    fmt.Printf("Match at offset %d: %s\n", m.StreamOffset, m.Result.Match)
    return true // continue to next match
})
```

**Example - Count matches in stream:**

```go
// stdlib - must read entire file into memory
data, _ := io.ReadAll(reader)
count := len(re.FindAllString(string(data), -1))

// regengo - constant memory
count, err := CompiledDate.FindReaderCount(reader, stream.Config{})
```

**Example - Find first match in stream:**

```go
// stdlib
loc := re.FindReaderIndex(reader)  // []int indices only, no captures

// regengo - full result with captures
result, offset, err := CompiledDate.FindReaderFirst(reader, stream.Config{})
if result != nil {
    fmt.Printf("First at %d: year=%s\n", offset, result.Year)
}
```

See [Streaming API](streaming.md) for complete documentation.

---

### Match Length Info

| stdlib `regexp` | regengo | Notes |
|-----------------|---------|-------|
| - | `CompiledDate.MatchLengthInfo()` | Returns `(minLen, maxLen int)` |
| - | `DateMinMatchLen` | Constant |
| - | `DateMaxMatchLen` | Constant (-1 if unbounded) |
| - | `CompiledDate.DefaultMaxLeftover()` | Recommended streaming buffer |

**Example:**

```go
// regengo only - useful for buffer sizing
minLen, maxLen := CompiledDate.MatchLengthInfo()
fmt.Printf("Matches are %d-%d bytes\n", minLen, maxLen)

// Use constants directly
if len(input) < DateMinMatchLen {
    return // Can't possibly match
}
```

---

### Replace

| stdlib `regexp` | regengo | Notes |
|-----------------|---------|-------|
| `re.ReplaceAllString(s, repl)` | `CompiledDate.ReplaceAllString(s, tmpl)` | Runtime template parsing |
| - | `CompiledDate.ReplaceAllString0(s)` | Pre-compiled template (fastest) |
| - | `CompiledDate.CompileReplaceTemplate(tmpl)` | Compile once, use many times |
| `re.ReplaceAllLiteralString(s, repl)` | Use template without `$` refs | Same effect |

**Template syntax** is fully compatible with stdlib: `$1`, `${name}`, `$0` (full match), `$$` (literal `$`).

**Example - Runtime replace:**

```go
// stdlib
re := regexp.MustCompile(`(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})`)
result := re.ReplaceAllString("2024-12-25", "$month/$day/$year")

// regengo - same template syntax
result := CompiledDate.ReplaceAllString("2024-12-25", "$month/$day/$year")
```

**Example - Pre-compiled (2-3x faster than stdlib):**

```go
// Generate with: -replacer '$month/$day/$year'
result := CompiledDate.ReplaceAllString0("2024-12-25")
// Result: "12/25/2024"
```

**Example - Compiled template (for config-driven templates):**

```go
tmpl, err := CompiledDate.CompileReplaceTemplate("$month/$day/$year")
if err != nil {
    log.Fatal(err)
}
// Use many times without re-parsing
for _, input := range inputs {
    result := tmpl.ReplaceAllString(input)
}
```

See [Replace API](replace-api.md) for complete documentation.

---

## Methods Not in stdlib

These regengo methods have no stdlib equivalent:

| Method | Purpose |
|--------|---------|
| `FindStringReuse(s, r)` | Zero-allocation find with result reuse |
| `FindBytesReuse(b, r)` | Zero-allocation find with result reuse |
| `FindAllStringAppend(s, n, slice)` | Append results to existing slice |
| `FindAllBytesAppend(b, n, slice)` | Append results to existing slice |
| `FindReader(r, cfg, cb)` | Stream all matches via callback |
| `FindReaderCount(r, cfg)` | Count matches in stream |
| `FindReaderFirst(r, cfg)` | First match with full result |
| `ReplaceAllString0(s)` | Pre-compiled replace (type-safe) |
| `ReplaceFirstString(s, tmpl)` | Replace first match only |
| `ReplaceAllBytesAppend(b, tmpl, buf)` | Zero-alloc replace with buffer |
| `CompileReplaceTemplate(tmpl)` | Compile template for reuse |
| `MatchLengthInfo()` | Get min/max match lengths |
| `DefaultMaxLeftover()` | Recommended streaming buffer size |

---

## Methods Not in regengo

These stdlib methods are not generated by regengo:

| stdlib Method | Why Not Included |
|---------------|------------------|
| `FindStringIndex(s)` | Use `FindString()` - result contains match position implicitly |
| `FindAllStringIndex(s, n)` | Use `FindAllString()` - positions derivable from Match field |
| `ReplaceAllStringFunc(s, f)` | Not yet implemented |
| `Split(s, n)` | Not yet implemented |
| `Expand(...)` | Template expansion not supported |
| `SubexpNames()` | Use struct field names directly |
| `SubexpIndex(name)` | Not needed - fields are named |
| `NumSubexp()` | Use reflection on result struct if needed |
| `Longest()` | No leftmost-longest mode |
| `Copy()` | Patterns are value types |

---

## Quick Reference Card

```go
// ═══════════════════════════════════════════════════════════════════
// MATCHING
// ═══════════════════════════════════════════════════════════════════
CompiledDate.MatchString(s string) bool
CompiledDate.MatchBytes(b []byte) bool

// ═══════════════════════════════════════════════════════════════════
// FINDING FIRST
// ═══════════════════════════════════════════════════════════════════
CompiledDate.FindString(s string) (*DateResult, bool)
CompiledDate.FindStringReuse(s string, r *DateResult) (*DateResult, bool)
CompiledDate.FindBytes(b []byte) (*DateBytesResult, bool)
CompiledDate.FindBytesReuse(b []byte, r *DateBytesResult) (*DateBytesResult, bool)

// ═══════════════════════════════════════════════════════════════════
// FINDING ALL
// ═══════════════════════════════════════════════════════════════════
CompiledDate.FindAllString(s string, n int) []*DateResult
CompiledDate.FindAllStringAppend(s string, n int, slice []*DateResult) []*DateResult
CompiledDate.FindAllBytes(b []byte, n int) []*DateBytesResult
CompiledDate.FindAllBytesAppend(b []byte, n int, slice []*DateBytesResult) []*DateBytesResult

// ═══════════════════════════════════════════════════════════════════
// STREAMING
// ═══════════════════════════════════════════════════════════════════
CompiledDate.FindReader(r io.Reader, cfg stream.Config,
    onMatch func(stream.Match[*DateBytesResult]) bool) error
CompiledDate.FindReaderCount(r io.Reader, cfg stream.Config) (int64, error)
CompiledDate.FindReaderFirst(r io.Reader, cfg stream.Config) (*DateBytesResult, int64, error)

// ═══════════════════════════════════════════════════════════════════
// REPLACE
// ═══════════════════════════════════════════════════════════════════
CompiledDate.ReplaceAllString(s string, tmpl string) string
CompiledDate.ReplaceFirstString(s string, tmpl string) string
CompiledDate.ReplaceAllBytes(b []byte, tmpl string) []byte
CompiledDate.ReplaceAllBytesAppend(b []byte, tmpl string, buf []byte) []byte
CompiledDate.ReplaceAllString0(s string) string       // pre-compiled
CompiledDate.ReplaceAllBytesAppend0(b []byte, buf []byte) []byte
CompiledDate.CompileReplaceTemplate(tmpl string) (*DateReplaceTemplate, error)

// ═══════════════════════════════════════════════════════════════════
// INTROSPECTION
// ═══════════════════════════════════════════════════════════════════
CompiledDate.MatchLengthInfo() (minLen, maxLen int)
CompiledDate.DefaultMaxLeftover() int
DateMinMatchLen  // const
DateMaxMatchLen  // const (-1 if unbounded)
```
