# Smart Analysis & Complexity Guarantees

Regengo automatically analyzes your pattern and selects the optimal matching engine for both performance and safety.

## Supported Engines

Regengo uses different strategies for boolean matching vs. capture extraction:

### Match Engines (MatchString/MatchBytes)

| Algorithm | Use Case | Complexity | Constraints |
|-----------|----------|------------|-------------|
| **Backtracking NFA** | Simple patterns | O(n) typical | Default engine |
| **Pure DFA** | ASCII patterns with catastrophic risk | O(n) single pass | ASCII only, no `.` or `\b` |
| **Thompson NFA** | Patterns with catastrophic risk | O(n×m) guaranteed | ≤ 64 NFA states |

### Capture Engines (FindString/FindBytes)

| Algorithm | Use Case | Complexity | Constraints |
|-----------|----------|------------|-------------|
| **Tagged DFA (TDFA)** | Patterns with catastrophic risk | O(n) guaranteed | ≤ 500 states (configurable) |
| **Memoized Backtracking** | Complex captures, TDFA fallback | O(n×m) with caching | Bit-vector memoization |
| **Backtracking** | Simple patterns | O(n) typical | Default for low-risk patterns |

## Auto-Detection Examples

```
Pattern: [\w\.+-]+@[\w\.-]+
 → Backtracking NFA for Match (simple, no catastrophic risk)
 → Backtracking for Captures

Pattern: (a+)+b
 → DFA for Match (catastrophic risk, ASCII-compatible)
 → TDFA for Captures (O(n) guaranteed)

Pattern: (?P<user>\w+)@(?P<domain>\w+)
 → Backtracking for Match (no catastrophic risk)
 → Backtracking for Captures (simple pattern)

Pattern: (?P<outer>(?P<inner>a+)+)b
 → DFA/Thompson for Match (catastrophic risk)
 → TDFA for Captures (O(n) guaranteed)
```

## Match Engine Selection Priority

1. `ForceThompson` → Thompson NFA (if pattern has ≤ 64 states)
2. `ForceDFA` or catastrophic risk detected → Try DFA (ASCII patterns only, no `.` or `\b`)
3. If DFA unavailable but risk exists → Thompson NFA (if ≤ 64 states)
4. If end anchor `$` present with risk → Memoized Backtracking
5. Default → Backtracking NFA

## Capture Engine Selection Priority

1. `ForceTDFA` or catastrophic risk detected → Try TDFA (if ≤ threshold states)
2. If TDFA unavailable → Memoized Backtracking
3. Default → Backtracking

## Verbose Mode

See analysis decisions with `-verbose`:

```bash
regengo -pattern '(a+)+b' -name Test -output test.go -verbose
```

```
[regengo] === Pattern Analysis ===
[regengo] Pattern: (a+)+b
[regengo] NFA states: 8
[regengo] Min match length: 2 bytes
[regengo] Max match length: unbounded
[regengo] Has nested quantifiers: true
[regengo] Has nested loops: true
[regengo] Has end anchor ($): false
[regengo] Has captures: true

[regengo] === Engine Selection ===
[regengo] Match engine: DFA (O(n) single pass)
[regengo] DFA info: DFA constructed with 3 states
[regengo] Capture engine: Tagged DFA (O(n) guaranteed, catastrophic risk mitigated)
```

## Runtime Complexity

| Operation | Go stdlib | Regengo | Notes |
|-----------|-----------|---------|-------|
| Simple match | O(n) | O(n) | Both efficient (Backtracking NFA) |
| Nested quantifiers `(a+)+` | **O(n×m)** guaranteed | **O(n)** with DFA | DFA provides O(n) single-pass matching for ASCII patterns |
| Thompson NFA | **O(n×m)** guaranteed | **O(n×m)** guaranteed | Used when DFA not suitable and pattern ≤ 64 states |
| Captures | O(n) typical | O(n) guaranteed | TDFA eliminates backtracking overhead |
| Complex captures | **O(n×m)** guaranteed | **O(n×m)** with memoization | Both safe, Regengo uses bit-vector caching |

> **Note on DFA Complexity:** Pure DFA matching is a single O(n) pass — unanchored restart positions are embedded in the automaton states (RE2-style), so no per-position re-scan occurs.

> **Note on Thompson NFA:** Thompson NFA is only available for patterns with ≤ 64 NFA states (uses uint64 bitset). For larger patterns, memoized backtracking is used instead.

## Memory Complexity

| Aspect | Go stdlib | Regengo |
|--------|-----------|---------|
| Per-match allocation | 2 allocs typical | 1 alloc or less |
| With reuse API | N/A | **0 allocs** |
| Result storage | `[]string` slices | Typed structs |
| Backtracking stack | Dynamic allocation | `sync.Pool` reuse |

> Run `go test -bench=. -benchmem ./benchmarks/curated/...` to see exact allocation counts for specific patterns.

## Where Regengo May Be Slower

| Scenario | Reason | Mitigation |
|----------|--------|------------|
| Patterns with many optional groups | TDFA state explosion | Increase `-tdfa-threshold` or pattern redesign |
| Non-matching pathological inputs | Memoization overhead in nested capture groups | Use stdlib for patterns expected to frequently not match, or reduce capture nesting |
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
    ForceDFA      bool // Force pure DFA for match functions (ASCII patterns only)
    ForceTNFA     bool // Force Tagged NFA for capture functions
    ForceTDFA     bool // Force Tagged DFA for capture functions

    // Tuning parameters
    TDFAThreshold int  // Max TDFA states before fallback (default: 500)
    DFAThreshold  int  // Max DFA states before fallback (default: 500)
    Verbose       bool // Print analysis decisions to stderr
}
```

### CLI Flags

```
Advanced (Engine Selection):
  -force-thompson    Force Thompson NFA for match functions
  -force-dfa         Force pure DFA for match functions (ASCII patterns only)
  -force-tnfa        Force Tagged NFA for capture functions
  -force-tdfa        Force Tagged DFA for capture functions

Tuning:
  -tdfa-threshold int  Max TDFA states before fallback (default 500)
  -dfa-threshold int   Max DFA states before fallback (default 500)
  -verbose             Print analysis decisions to stderr
```

### When to Use Each Engine

| Option | Use Case |
|--------|----------|
| `ForceThompson` | Guaranteed O(n×m) for untrusted patterns |
| `ForceDFA` | O(n) matching for ASCII patterns, maximum performance |
| `ForceTNFA` | Debugging, or patterns with many optional groups |
| `ForceTDFA` | Maximize capture performance, predictable patterns |
| `TDFAThreshold` | Increase for complex patterns with known bounds |
| `DFAThreshold` | Increase for patterns that benefit from larger DFA |
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
  "nfa_states": 11,
  "min_match_length": 3,
  "max_match_length": -1
}
```

The labels indicate:
- **feature_labels**: Pattern characteristics (Anchored, Alternation, Captures, CharClass, Multibyte, NonCapturing, Quantifiers, Simple, UnicodeCharClass, WordBoundary)
- **engine_labels**: Which engines will be used (DFA, Thompson, TDFA, TNFA, Memoization, Backtracking)
