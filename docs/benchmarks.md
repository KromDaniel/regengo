# Detailed Benchmarks

Benchmarks run on Apple M4 Pro. Each benchmark shows performance for Go stdlib vs regengo.

## Summary

| Pattern Type | Typical Speedup | Memory Reduction |
|--------------|-----------------|------------------|
| Simple match | 2-3x faster | 0 allocs |
| Capture groups | 2-5x faster | 50% fewer allocs |
| FindAll | 5-9x faster | 50-100% fewer allocs |
| With reuse API | 10-15x faster | 0 allocs |

## Match Patterns (No Captures)

Simple pattern matching without capture groups - uses DFA for O(n) performance.

### Email

Simple email matching without capture groups

**Pattern:**
```regex
[\w\.+-]+@[\w\.-]+\.[\w\.-]+
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1523.0 | 0 | 0 | - |
| regengo | 540.6 | 0 | 0 | **2.8x faster** |

### Greedy

Greedy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 759.7 | 0 | 0 | - |
| regengo | 173.5 | 0 | 0 | **4.4x faster** |

### Lazy

Lazy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1288.0 | 0 | 0 | - |
| regengo | 180.5 | 0 | 0 | **7.1x faster** |

## Capture Patterns

Patterns with named capture groups - uses TDFA or Thompson NFA.

### DateCapture

ISO date pattern with year, month, day capture groups

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 68.2 | 0 | 0 | - |
| regengo | 4.1 | 0 | 0 | **16.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 107.6 | 128 | 2 | - |
| regengo | 20.7 | 64 | 1 | **5.2x faster** |
| regengo (reuse) | 7.5 | 0 | 0 | **14.3x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 189.6 | 370 | 3 | - |
| regengo | 32.0 | 72 | 2 | **5.9x faster** |
| regengo (reuse) | 9.0 | 0 | 0 | **21.0x faster** |

### EmailCapture

Email pattern with named capture groups

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 247.0 | 0 | 0 | - |
| regengo | 66.2 | 0 | 0 | **3.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 298.1 | 128 | 2 | - |
| regengo | 133.0 | 64 | 1 | **2.2x faster** |
| regengo (reuse) | 118.4 | 0 | 0 | **2.5x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 389.0 | 369 | 3 | - |
| regengo | 157.5 | 72 | 2 | **2.5x faster** |
| regengo (reuse) | 134.0 | 0 | 0 | **2.9x faster** |

### URLCapture

URL pattern with protocol, host, port, and path capture groups

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 245.6 | 0 | 0 | - |
| regengo | 30.3 | 0 | 0 | **8.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 299.4 | 160 | 2 | - |
| regengo | 116.8 | 80 | 1 | **2.6x faster** |
| regengo (reuse) | 99.2 | 0 | 0 | **3.0x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 378.6 | 401 | 3 | - |
| regengo | 140.6 | 88 | 2 | **2.7x faster** |
| regengo (reuse) | 131.9 | 80 | 1 | **2.9x faster** |

## FindAll Patterns

Finding multiple matches in text.

### MultiDate

Find multiple dates in text

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 127.3 | 0 | 0 | - |
| regengo | 7.9 | 0 | 0 | **16.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 175.7 | 102 | 1 | - |
| regengo | 27.7 | 51 | 0 | **6.4x faster** |
| regengo (reuse) | 17.1 | 0 | 0 | **10.3x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 301.9 | 295 | 2 | - |
| regengo | 50.5 | 57 | 1 | **6.0x faster** |
| regengo (reuse) | 30.9 | 0 | 0 | **9.8x faster** |

### MultiEmail

Find multiple email addresses in text

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 301.2 | 0 | 0 | - |
| regengo | 60.7 | 0 | 0 | **5.0x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 423.8 | 102 | 1 | - |
| regengo | 165.7 | 51 | 0 | **2.6x faster** |
| regengo (reuse) | 151.0 | 0 | 0 | **2.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 627.6 | 321 | 2 | - |
| regengo | 311.5 | 73 | 2 | **2.0x faster** |
| regengo (reuse) | 281.3 | 0 | 0 | **2.2x faster** |

## TDFA Patterns (Catastrophic Backtracking Prevention)

These patterns have nested quantifiers + captures which would cause exponential backtracking without TDFA's O(n) guarantee.

### TDFAComplexURL

Complex URL with optional components

**Pattern:**
```regex
(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 505.8 | 0 | 0 | - |
| regengo | 47.1 | 0 | 0 | **10.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 608.3 | 288 | 2 | - |
| regengo | 277.9 | 144 | 1 | **2.2x faster** |
| regengo (reuse) | 251.3 | 0 | 0 | **2.4x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 693.8 | 529 | 3 | - |
| regengo | 292.5 | 152 | 2 | **2.4x faster** |
| regengo (reuse) | 254.3 | 0 | 0 | **2.7x faster** |

### TDFALogParser

Log line parser with multiple optional groups

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 322.3 | 0 | 0 | - |
| regengo | 42.2 | 0 | 0 | **7.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 379.8 | 192 | 2 | - |
| regengo | 122.6 | 96 | 1 | **3.1x faster** |
| regengo (reuse) | 108.5 | 0 | 0 | **3.5x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 477.8 | 433 | 3 | - |
| regengo | 133.9 | 104 | 2 | **3.6x faster** |
| regengo (reuse) | 110.6 | 0 | 0 | **4.3x faster** |

### TDFANestedWord

Nested quantifiers with word boundaries

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 578.9 | 0 | 0 | - |
| regengo | 99.4 | 0 | 0 | **5.8x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 660.3 | 112 | 2 | - |
| regengo | 414.6 | 48 | 1 | **1.6x faster** |
| regengo (reuse) | 400.5 | 0 | 0 | **1.6x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 756.2 | 336 | 3 | - |
| regengo | 507.5 | 56 | 2 | **1.5x faster** |
| regengo (reuse) | 483.3 | 0 | 0 | **1.6x faster** |

### TDFAPathological

Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 407.3 | 0 | 0 | - |
| regengo | 127.0 | 0 | 0 | **3.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 494.6 | 84 | 1 | - |
| regengo | 917.7 | 36 | 0 | **1.9x slower** |
| regengo (reuse) | 909.5 | 0 | 0 | **1.8x slower** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 556.6 | 253 | 2 | - |
| regengo | 333.3 | 42 | 1 | **1.7x faster** |
| regengo (reuse) | 316.3 | 0 | 0 | **1.8x faster** |

### TDFASemVer

Semantic version with optional pre-release and build metadata

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 164.7 | 0 | 0 | - |
| regengo | 13.4 | 0 | 0 | **12.3x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 225.8 | 192 | 2 | - |
| regengo | 74.0 | 96 | 1 | **3.0x faster** |
| regengo (reuse) | 55.8 | 0 | 0 | **4.0x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 314.4 | 434 | 3 | - |
| regengo | 86.2 | 104 | 2 | **3.6x faster** |
| regengo (reuse) | 75.6 | 96 | 1 | **4.2x faster** |

## TNFA Patterns (Thompson NFA with Memoization)

Patterns forced to use Thompson NFA with memoization for testing.

### TNFAPathological

Pathological pattern forced to use TNFA with memoization

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 202.8 | 0 | 0 | - |
| regengo | 23.4 | 0 | 0 | **8.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 248.4 | 112 | 2 | - |
| regengo | 148.8 | 48 | 1 | **1.7x faster** |
| regengo (reuse) | 137.2 | 0 | 0 | **1.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 325.0 | 337 | 3 | - |
| regengo | 178.8 | 56 | 2 | **1.8x faster** |
| regengo (reuse) | 154.2 | 0 | 0 | **2.1x faster** |

## Replace Patterns

String replacement with precompiled replacer templates.

### ReplaceDate

Date format conversion using capture groups

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 134.7 | 0 | 0 | - |
| regengo | 7.7 | 0 | 0 | **17.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 195.9 | 128 | 2 | - |
| regengo | 33.4 | 64 | 1 | **5.9x faster** |
| regengo (reuse) | 16.7 | 0 | 0 | **11.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 5513.0 | 7219 | 71 | - |
| regengo | 932.1 | 2930 | 38 | **5.9x faster** |
| regengo (reuse) | 335.4 | 0 | 0 | **16.4x faster** |

**Replace `$month/$day/$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 6647.5 | 3776 | 41 | - |
| regengo | 1501.4 | 1909 | 9 | **4.4x faster** |
| regengo (precompiled) | 986.1 | 1141 | 4 | **6.7x faster** |
| regengo (reuse) | 726.4 | 0 | 0 | **9.2x faster** |

**Replace `[DATE]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3911.9 | 928 | 6 | - |
| regengo | 784.4 | 757 | 6 | **5.0x faster** |
| regengo (precompiled) | 702.6 | 661 | 4 | **5.6x faster** |
| regengo (reuse) | 571.9 | 0 | 0 | **6.8x faster** |

**Replace `$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 5016.8 | 2768 | 40 | - |
| regengo | 770.6 | 458 | 5 | **6.5x faster** |
| regengo (precompiled) | 694.9 | 362 | 3 | **7.2x faster** |
| regengo (reuse) | 600.5 | 0 | 0 | **8.4x faster** |

### ReplaceEmail

Email replacement with capture group references

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 230.2 | 0 | 0 | - |
| regengo | 47.2 | 0 | 0 | **4.9x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 325.1 | 128 | 2 | - |
| regengo | 140.0 | 64 | 1 | **2.3x faster** |
| regengo (reuse) | 124.7 | 0 | 0 | **2.6x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3404.9 | 2341 | 23 | - |
| regengo | 1368.1 | 901 | 13 | **2.5x faster** |
| regengo (reuse) | 1165.9 | 0 | 0 | **2.9x faster** |

**Replace `$user@REDACTED.$tld`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3598.3 | 1654 | 16 | - |
| regengo | 1572.4 | 1098 | 8 | **2.3x faster** |
| regengo (precompiled) | 1421.1 | 714 | 4 | **2.5x faster** |
| regengo (reuse) | 1297.2 | 0 | 0 | **2.8x faster** |

**Replace `[EMAIL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2679.6 | 348 | 5 | - |
| regengo | 1363.9 | 328 | 5 | **2.0x faster** |
| regengo (precompiled) | 1317.4 | 232 | 3 | **2.0x faster** |
| regengo (reuse) | 1285.7 | 0 | 0 | **2.1x faster** |

**Replace `$0`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3314.2 | 1329 | 15 | - |
| regengo | 1376.8 | 515 | 5 | **2.4x faster** |
| regengo (precompiled) | 1336.6 | 419 | 3 | **2.5x faster** |
| regengo (reuse) | 1274.1 | 0 | 0 | **2.6x faster** |

### ReplaceURL

URL redaction with selective capture group output

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 190.1 | 0 | 0 | - |
| regengo | 55.0 | 0 | 0 | **3.5x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 240.7 | 160 | 2 | - |
| regengo | 90.7 | 80 | 1 | **2.7x faster** |
| regengo (reuse) | 73.4 | 0 | 0 | **3.3x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 560.4 | 615 | 5 | - |
| regengo | 241.3 | 312 | 5 | **2.3x faster** |
| regengo (reuse) | 208.1 | 266 | 3 | **2.7x faster** |

**Replace `$protocol://$host[REDACTED]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 683.3 | 442 | 8 | - |
| regengo | 405.8 | 546 | 8 | **1.7x faster** |
| regengo (precompiled) | 240.1 | 162 | 4 | **2.8x faster** |
| regengo (reuse) | 171.6 | 0 | 0 | **4.0x faster** |

**Replace `[URL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 477.9 | 112 | 5 | - |
| regengo | 247.4 | 162 | 5 | **1.9x faster** |
| regengo (precompiled) | 207.4 | 66 | 3 | **2.3x faster** |
| regengo (reuse) | 163.5 | 0 | 0 | **2.9x faster** |

**Replace `$host`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 586.1 | 310 | 7 | - |
| regengo | 260.9 | 170 | 5 | **2.2x faster** |
| regengo (precompiled) | 209.7 | 74 | 3 | **2.8x faster** |
| regengo (reuse) | 166.9 | 0 | 0 | **3.5x faster** |

---

## Running Benchmarks

### Benchmark Structure

Benchmarks use a nested structure for clear comparison:

```
Benchmark{Pattern}/
├── Match/Input[i]/{stdlib,regengo}
├── FindFirst/Input[i]/{stdlib,regengo,regengo_reuse}
├── FindAll/Input[i]/{stdlib,regengo,regengo_append}
└── Replace/Template[j]/Input[i]/{stdlib,regengo_runtime,regengo,regengo_append}
```

### Running Specific Benchmarks

```bash
# Run all benchmarks for a pattern
go test ./benchmarks/curated/... -bench="BenchmarkDateCapture" -benchmem

# Run only Match benchmarks
go test ./benchmarks/curated/... -bench="Match" -benchmem

# Run only regengo_reuse variants
go test ./benchmarks/curated/... -bench="regengo_reuse" -benchmem

# Run specific input
go test ./benchmarks/curated/... -bench="Input\\[0\\]" -benchmem
```

### Aggregating Results

Use the aggregation script for summary statistics across all inputs:

```bash
# Aggregate results for a pattern
go test ./benchmarks/curated/... -bench="BenchmarkDateCapture" -benchmem | go run scripts/curated/aggregate.go
```

### Make Targets

```bash
# Run benchmarks (generates and runs curated benchmarks)
make bench

# Analyze benchmark results with comparison summary
make bench-analyze

# Generate markdown output
make bench-format

# Generate performance chart
make bench-chart
```

## Regenerating Results

To regenerate benchmark files after code changes:

```bash
# Regenerate curated benchmark code
go run scripts/curated/generate.go scripts/curated/cases.go

# Or use make
make bench-format
```
