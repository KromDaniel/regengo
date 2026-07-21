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
| stdlib | 1478.0 | 0 | 0 | - |
| regengo | 523.6 | 0 | 0 | **2.8x faster** |

### Greedy

Greedy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 749.0 | 0 | 0 | - |
| regengo | 166.6 | 0 | 0 | **4.5x faster** |

### Lazy

Lazy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1253.0 | 0 | 0 | - |
| regengo | 153.9 | 0 | 0 | **8.1x faster** |

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
| stdlib | 65.6 | 0 | 0 | - |
| regengo | 3.8 | 0 | 0 | **17.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 98.5 | 128 | 2 | - |
| regengo | 18.9 | 64 | 1 | **5.2x faster** |
| regengo (reuse) | 7.4 | 0 | 0 | **13.3x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 169.1 | 369 | 3 | - |
| regengo | 28.9 | 72 | 2 | **5.9x faster** |
| regengo (reuse) | 8.4 | 0 | 0 | **20.1x faster** |

### EmailCapture

Email pattern with named capture groups

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 236.3 | 0 | 0 | - |
| regengo | 64.0 | 0 | 0 | **3.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 281.6 | 128 | 2 | - |
| regengo | 125.8 | 64 | 1 | **2.2x faster** |
| regengo (reuse) | 115.4 | 0 | 0 | **2.4x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 352.0 | 368 | 3 | - |
| regengo | 148.5 | 72 | 2 | **2.4x faster** |
| regengo (reuse) | 129.9 | 0 | 0 | **2.7x faster** |

### URLCapture

URL pattern with protocol, host, port, and path capture groups

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 238.2 | 0 | 0 | - |
| regengo | 28.9 | 0 | 0 | **8.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 279.6 | 160 | 2 | - |
| regengo | 107.4 | 80 | 1 | **2.6x faster** |
| regengo (reuse) | 95.2 | 0 | 0 | **2.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 345.3 | 401 | 3 | - |
| regengo | 130.9 | 88 | 2 | **2.6x faster** |
| regengo (reuse) | 121.8 | 80 | 1 | **2.8x faster** |

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
| stdlib | 124.0 | 0 | 0 | - |
| regengo | 7.5 | 0 | 0 | **16.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 167.1 | 102 | 1 | - |
| regengo | 26.5 | 51 | 0 | **6.3x faster** |
| regengo (reuse) | 17.0 | 0 | 0 | **9.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 272.8 | 295 | 2 | - |
| regengo | 44.8 | 57 | 1 | **6.1x faster** |
| regengo (reuse) | 27.5 | 0 | 0 | **9.9x faster** |

### MultiEmail

Find multiple email addresses in text

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 281.9 | 0 | 0 | - |
| regengo | 57.2 | 0 | 0 | **4.9x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 395.8 | 102 | 1 | - |
| regengo | 154.1 | 51 | 0 | **2.6x faster** |
| regengo (reuse) | 143.9 | 0 | 0 | **2.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 573.9 | 321 | 2 | - |
| regengo | 289.6 | 73 | 2 | **2.0x faster** |
| regengo (reuse) | 268.2 | 0 | 0 | **2.1x faster** |

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
| stdlib | 478.9 | 0 | 0 | - |
| regengo | 43.5 | 0 | 0 | **11.0x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 571.3 | 288 | 2 | - |
| regengo | 260.4 | 144 | 1 | **2.2x faster** |
| regengo (reuse) | 243.6 | 0 | 0 | **2.3x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 635.1 | 529 | 3 | - |
| regengo | 273.7 | 152 | 2 | **2.3x faster** |
| regengo (reuse) | 245.9 | 0 | 0 | **2.6x faster** |

### TDFALogParser

Log line parser with multiple optional groups

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 311.2 | 0 | 0 | - |
| regengo | 40.4 | 0 | 0 | **7.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 357.7 | 192 | 2 | - |
| regengo | 113.5 | 96 | 1 | **3.2x faster** |
| regengo (reuse) | 105.2 | 0 | 0 | **3.4x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 431.5 | 433 | 3 | - |
| regengo | 124.3 | 104 | 2 | **3.5x faster** |
| regengo (reuse) | 106.9 | 0 | 0 | **4.0x faster** |

### TDFANestedWord

Nested quantifiers with word boundaries

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 555.5 | 0 | 0 | - |
| regengo | 92.6 | 0 | 0 | **6.0x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 621.6 | 112 | 2 | - |
| regengo | 397.1 | 48 | 1 | **1.6x faster** |
| regengo (reuse) | 387.7 | 0 | 0 | **1.6x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 691.0 | 336 | 3 | - |
| regengo | 480.4 | 56 | 2 | **1.4x faster** |
| regengo (reuse) | 459.5 | 0 | 0 | **1.5x faster** |

### TDFAPathological

Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 395.5 | 0 | 0 | - |
| regengo | 111.1 | 0 | 0 | **3.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 472.6 | 84 | 1 | - |
| regengo | 883.4 | 36 | 0 | **1.9x slower** |
| regengo (reuse) | 874.1 | 0 | 0 | **1.8x slower** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 517.1 | 252 | 2 | - |
| regengo | 315.3 | 42 | 1 | **1.6x faster** |
| regengo (reuse) | 302.3 | 0 | 0 | **1.7x faster** |

### TDFASemVer

Semantic version with optional pre-release and build metadata

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 158.7 | 0 | 0 | - |
| regengo | 12.9 | 0 | 0 | **12.3x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 204.6 | 192 | 2 | - |
| regengo | 66.9 | 96 | 1 | **3.1x faster** |
| regengo (reuse) | 53.2 | 0 | 0 | **3.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 278.8 | 433 | 3 | - |
| regengo | 79.4 | 104 | 2 | **3.5x faster** |
| regengo (reuse) | 68.8 | 96 | 1 | **4.1x faster** |

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
| stdlib | 195.8 | 0 | 0 | - |
| regengo | 21.8 | 0 | 0 | **9.0x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 232.9 | 112 | 2 | - |
| regengo | 139.7 | 48 | 1 | **1.7x faster** |
| regengo (reuse) | 132.4 | 0 | 0 | **1.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 291.1 | 337 | 3 | - |
| regengo | 167.1 | 56 | 2 | **1.7x faster** |
| regengo (reuse) | 148.2 | 0 | 0 | **2.0x faster** |

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
| stdlib | 129.2 | 0 | 0 | - |
| regengo | 7.0 | 0 | 0 | **18.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 176.1 | 128 | 2 | - |
| regengo | 26.6 | 64 | 1 | **6.6x faster** |
| regengo (reuse) | 15.4 | 0 | 0 | **11.4x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 4841.1 | 7212 | 71 | - |
| regengo | 870.7 | 2930 | 38 | **5.6x faster** |
| regengo (reuse) | 326.7 | 0 | 0 | **14.8x faster** |

**Replace `$month/$day/$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 6047.8 | 3771 | 41 | - |
| regengo | 1333.3 | 1909 | 9 | **4.5x faster** |
| regengo (precompiled) | 898.7 | 1141 | 4 | **6.7x faster** |
| regengo (reuse) | 688.4 | 0 | 0 | **8.8x faster** |

**Replace `[DATE]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3657.6 | 927 | 6 | - |
| regengo | 729.5 | 757 | 6 | **5.0x faster** |
| regengo (precompiled) | 664.2 | 661 | 4 | **5.5x faster** |
| regengo (reuse) | 550.5 | 0 | 0 | **6.6x faster** |

**Replace `$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 4747.3 | 2764 | 40 | - |
| regengo | 729.2 | 458 | 5 | **6.5x faster** |
| regengo (precompiled) | 648.9 | 362 | 3 | **7.3x faster** |
| regengo (reuse) | 584.7 | 0 | 0 | **8.1x faster** |

### ReplaceEmail

Email replacement with capture group references

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 223.6 | 0 | 0 | - |
| regengo | 45.6 | 0 | 0 | **4.9x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 308.1 | 128 | 2 | - |
| regengo | 131.6 | 64 | 1 | **2.3x faster** |
| regengo (reuse) | 120.8 | 0 | 0 | **2.6x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3183.9 | 2338 | 23 | - |
| regengo | 1278.0 | 901 | 13 | **2.5x faster** |
| regengo (reuse) | 1129.0 | 0 | 0 | **2.8x faster** |

**Replace `$user@REDACTED.$tld`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3427.3 | 1652 | 16 | - |
| regengo | 1452.6 | 1098 | 8 | **2.4x faster** |
| regengo (precompiled) | 1326.6 | 714 | 4 | **2.6x faster** |
| regengo (reuse) | 1253.7 | 0 | 0 | **2.7x faster** |

**Replace `[EMAIL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2547.9 | 347 | 5 | - |
| regengo | 1284.3 | 328 | 5 | **2.0x faster** |
| regengo (precompiled) | 1247.4 | 232 | 3 | **2.0x faster** |
| regengo (reuse) | 1224.3 | 0 | 0 | **2.1x faster** |

**Replace `$0`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3139.5 | 1328 | 15 | - |
| regengo | 1293.8 | 515 | 5 | **2.4x faster** |
| regengo (precompiled) | 1261.2 | 419 | 3 | **2.5x faster** |
| regengo (reuse) | 1229.6 | 0 | 0 | **2.6x faster** |

### ReplaceURL

URL redaction with selective capture group output

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 184.0 | 0 | 0 | - |
| regengo | 54.3 | 0 | 0 | **3.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 225.1 | 160 | 2 | - |
| regengo | 82.4 | 80 | 1 | **2.7x faster** |
| regengo (reuse) | 70.9 | 0 | 0 | **3.2x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 500.3 | 615 | 5 | - |
| regengo | 210.9 | 312 | 5 | **2.4x faster** |
| regengo (reuse) | 183.8 | 266 | 3 | **2.7x faster** |

**Replace `$protocol://$host[REDACTED]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 617.9 | 441 | 8 | - |
| regengo | 350.2 | 546 | 8 | **1.8x faster** |
| regengo (precompiled) | 211.7 | 162 | 4 | **2.9x faster** |
| regengo (reuse) | 162.9 | 0 | 0 | **3.8x faster** |

**Replace `[URL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 446.0 | 112 | 5 | - |
| regengo | 223.5 | 162 | 5 | **2.0x faster** |
| regengo (precompiled) | 189.2 | 66 | 3 | **2.4x faster** |
| regengo (reuse) | 154.6 | 0 | 0 | **2.9x faster** |

**Replace `$host`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 532.3 | 310 | 7 | - |
| regengo | 232.4 | 170 | 5 | **2.3x faster** |
| regengo (precompiled) | 191.8 | 74 | 3 | **2.8x faster** |
| regengo (reuse) | 155.8 | 0 | 0 | **3.4x faster** |

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
