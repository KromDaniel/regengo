# Smart Analysis & Complexity Guarantees

Regengo automatically analyzes your pattern and selects the optimal matching engine for both performance and safety.

## Supported Algorithms

| Algorithm | Use Case | Complexity | Status |
|-----------|----------|------------|--------|
| **Backtracking DFA** | Simple patterns | O(n) typical | Default |
| **Thompson NFA** | Patterns at risk of catastrophic backtracking | O(n×m) guaranteed | Supported |
| **Tagged DFA (TDFA)** | Capture groups with complex patterns | O(n) guaranteed | Supported |
| **Bit-vector Memoization** | Nested quantifiers with captures | O(n×m) with caching | Supported |

## Auto-Detection Examples

```
Pattern: [\w\.+-]+@[\w\.-]+
 → Backtracking DFA (simple, fast)

Pattern: (a+)+b
 → Thompson NFA (prevents exponential backtracking)

Pattern: (?P<user>\w+)@(?P<domain>\w+)
 → Tagged DFA (O(n) captures)

Pattern: (?P<outer>(?P<inner>a+)+)b
 → TDFA + Memoization (complex nested captures)
```

## Verbose Mode

See analysis decisions with `-verbose`:

```bash
regengo -pattern '(a+)+b' -name Test -output test.go -verbose
```

```
=== Pattern Analysis ===
Pattern: (a+)+b
NFA states: 8
Has nested quantifiers: true
Has catastrophic risk: true
→ Using Thompson NFA for MatchString
→ Using Tagged DFA for FindString
```

## Runtime Complexity

| Operation | Go stdlib | Regengo | Notes |
|-----------|-----------|---------|-------|
| Simple match | O(n) | O(n) | Both efficient |
| Nested quantifiers `(a+)+` | **O(n×m)** guaranteed | **O(n×m)** guaranteed | Both use Thompson NFA construction |
| Captures | O(n) typical | O(n) guaranteed | TDFA eliminates backtracking overhead |
| Complex captures | **O(n×m)** guaranteed | **O(n×m)** with memoization | Both safe, Regengo uses bit-vector caching |

## Memory Complexity

| Aspect | Go stdlib | Regengo |
|--------|-----------|---------|
| Per-match allocation | 2 allocs (128-192 B) | 1 alloc (64-96 B) |
| With reuse API | N/A | **0 allocs** |
| Result storage | `[]string` slices | Typed structs |
| Backtracking stack | Dynamic allocation | `sync.Pool` reuse |

## Where Regengo May Be Slower

| Scenario | Reason | Mitigation |
|----------|--------|------------|
| Patterns with many optional groups | TDFA state explosion | Increase `-tdfa-threshold` or pattern redesign |
| Non-matching pathological inputs | Memoization overhead in nested capture groups (e.g., `(?P<outer>(?P<inner>a+)+)b` with input `"aaa...c"`) — up to **2.8x slower** | Use stdlib for patterns expected to frequently not match, or reduce capture nesting |
| First cold call | No JIT, but consistent performance | Warm up in init() if needed |

> **Note:** Regengo trades compilation time for runtime performance. The generated code is optimized by the Go compiler, giving consistent, predictable performance without runtime interpretation overhead.

## Advanced Options

For fine-grained control over the compilation engine:

### Library API

```go
type Options struct {
    // ... basic options ...

    // Engine selection
    ForceThompson bool // Force Thompson NFA for all match functions
    ForceTNFA     bool // Force Tagged NFA for capture functions
    ForceTDFA     bool // Force Tagged DFA for capture functions

    // Tuning parameters
    TDFAThreshold int  // Max DFA states before fallback (default: 500)
    Verbose       bool // Print analysis decisions to stderr
}
```

### CLI Flags

```
Advanced (Engine Selection):
  -force-thompson    Force Thompson NFA for match functions
  -force-tnfa        Force Tagged NFA for capture functions
  -force-tdfa        Force Tagged DFA for capture functions

Tuning:
  -tdfa-threshold int  Max DFA states before fallback (default 500)
  -verbose             Print analysis decisions to stderr
```

### When to Use Each Engine

| Option | Use Case |
|--------|----------|
| `ForceThompson` | Guaranteed O(n×m) for untrusted patterns |
| `ForceTNFA` | Debugging, or patterns with many optional groups |
| `ForceTDFA` | Maximize capture performance, predictable patterns |
| `TDFAThreshold` | Increase for complex patterns with known bounds |
| `Verbose` | Debug engine selection, understand analysis decisions |

### Example: Force TDFA for Trusted Patterns

```go
err := regengo.Compile(regengo.Options{
    Pattern:    `(?P<key>\w+)=(?P<value>[^;]+)`,
    Name:       "KeyValue",
    OutputFile: "keyvalue.go",
    Package:    "parser",
    ForceTDFA:  true,  // Skip analysis, use O(n) TDFA directly
})
```

### Example: Safe Mode for User Input

```go
err := regengo.Compile(regengo.Options{
    Pattern:       userPattern,
    Name:          "UserRegex",
    OutputFile:    "user_regex.go",
    Package:       "validator",
    ForceThompson: true,  // Guaranteed O(n×m), prevents ReDoS
})
```

## Analyze Mode (No Code Generation)

Use `-analyze` to inspect pattern characteristics without generating code:

```bash
regengo -analyze -pattern '(?P<user>\w+)@(?P<domain>\w+)'
```

Output:
```json
{
  "feature_labels": ["Captures", "CharClass", "Quantifiers"],
  "engine_labels": ["Backtracking"],
  "has_captures": true,
  "has_catastrophic_risk": false,
  "has_end_anchor": false,
  "nfa_states": 11
}
```

The labels indicate:
- **feature_labels**: Pattern characteristics (Anchored, Alternation, Captures, CharClass, Multibyte, NonCapturing, Quantifiers, Simple, UnicodeCharClass, WordBoundary)
- **engine_labels**: Which engines will be used (Thompson, TDFA, TNFA, Memoization, Backtracking)
