# Detailed Benchmarks

Each benchmark shows performance for Go stdlib vs regengo.

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
| stdlib | 2975.0 | 0 | 0 | - |
| regengo | 164.6 | 0 | 0 | **18.1x faster** |

### Greedy

Greedy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2493.0 | 0 | 0 | - |
| regengo | 162.9 | 0 | 0 | **15.3x faster** |

### Lazy

Lazy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 3950.0 | 0 | 0 | - |
| regengo | 163.4 | 0 | 0 | **24.2x faster** |

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
| stdlib | 153.9 | 0 | 0 | - |
| regengo | 10.4 | 0 | 0 | **14.8x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 274.0 | 128 | 2 | - |
| regengo | 37.8 | 64 | 1 | **7.3x faster** |
| regengo (reuse) | 9.0 | 0 | 0 | **30.5x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 389.7 | 152 | 3 | - |
| regengo | 58.5 | 72 | 2 | **6.7x faster** |
| regengo (reuse) | 9.8 | 0 | 0 | **39.8x faster** |

### EmailCapture

Email pattern with named capture groups

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 507.6 | 0 | 0 | - |
| regengo | 16.1 | 0 | 0 | **31.5x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 666.8 | 128 | 2 | - |
| regengo | 253.6 | 64 | 1 | **2.6x faster** |
| regengo (reuse) | 231.4 | 0 | 0 | **2.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 776.7 | 152 | 3 | - |
| regengo | 293.1 | 72 | 2 | **2.6x faster** |
| regengo (reuse) | 248.7 | 0 | 0 | **3.1x faster** |

### URLCapture

URL pattern with protocol, host, port, and path capture groups

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 555.8 | 0 | 0 | - |
| regengo | 14.0 | 0 | 0 | **39.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 729.3 | 160 | 2 | - |
| regengo | 226.4 | 80 | 1 | **3.2x faster** |
| regengo (reuse) | 192.0 | 0 | 0 | **3.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 823.4 | 184 | 3 | - |
| regengo | 249.6 | 88 | 2 | **3.3x faster** |
| regengo (reuse) | 230.1 | 80 | 1 | **3.6x faster** |

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
| stdlib | 250.5 | 0 | 0 | - |
| regengo | 11.3 | 0 | 0 | **22.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 437.7 | 102 | 1 | - |
| regengo | 38.1 | 51 | 0 | **11.5x faster** |
| regengo (reuse) | 17.0 | 0 | 0 | **25.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 642.0 | 121 | 2 | - |
| regengo | 62.5 | 57 | 1 | **10.3x faster** |
| regengo (reuse) | 26.3 | 0 | 0 | **24.4x faster** |

### MultiEmail

Find multiple email addresses in text

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 602.9 | 0 | 0 | - |
| regengo | 16.7 | 0 | 0 | **36.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 931.3 | 102 | 1 | - |
| regengo | 308.5 | 51 | 0 | **3.0x faster** |
| regengo (reuse) | 287.9 | 0 | 0 | **3.2x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1281.7 | 144 | 2 | - |
| regengo | 566.4 | 73 | 2 | **2.3x faster** |
| regengo (reuse) | 516.4 | 0 | 0 | **2.5x faster** |

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
| stdlib | 1236.8 | 0 | 0 | - |
| regengo | 16.9 | 0 | 0 | **73.0x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1565.0 | 288 | 2 | - |
| regengo | 694.0 | 144 | 1 | **2.3x faster** |
| regengo (reuse) | 588.2 | 0 | 0 | **2.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1734.7 | 313 | 3 | - |
| regengo | 727.0 | 152 | 2 | **2.4x faster** |
| regengo (reuse) | 609.5 | 0 | 0 | **2.8x faster** |

### TDFALogParser

Log line parser with multiple optional groups

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 856.3 | 0 | 0 | - |
| regengo | 90.6 | 0 | 0 | **9.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1081.9 | 192 | 2 | - |
| regengo | 335.8 | 96 | 1 | **3.2x faster** |
| regengo (reuse) | 279.3 | 0 | 0 | **3.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1216.3 | 217 | 3 | - |
| regengo | 351.2 | 104 | 2 | **3.5x faster** |
| regengo (reuse) | 279.9 | 0 | 0 | **4.3x faster** |

### TDFANestedWord

Nested quantifiers with word boundaries

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1468.5 | 0 | 0 | - |
| regengo | 67.5 | 0 | 0 | **21.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1839.3 | 112 | 2 | - |
| regengo | 987.4 | 48 | 1 | **1.9x faster** |
| regengo (reuse) | 967.7 | 0 | 0 | **1.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1836.5 | 120 | 3 | - |
| regengo | 1088.6 | 56 | 2 | **1.7x faster** |
| regengo (reuse) | 1042.4 | 0 | 0 | **1.8x faster** |

### TDFAPathological

Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 963.7 | 0 | 0 | - |
| regengo | 54.2 | 0 | 0 | **17.8x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1426.5 | 84 | 1 | - |
| regengo | 2098.9 | 36 | 0 | **1.5x slower** |
| regengo (reuse) | 2098.4 | 0 | 0 | **1.5x slower** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1511.1 | 90 | 2 | - |
| regengo | 688.2 | 42 | 1 | **2.2x faster** |
| regengo (reuse) | 656.4 | 0 | 0 | **2.3x faster** |

### TDFASemVer

Semantic version with optional pre-release and build metadata

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 367.0 | 0 | 0 | - |
| regengo | 6.0 | 0 | 0 | **61.5x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 541.4 | 192 | 2 | - |
| regengo | 165.7 | 96 | 1 | **3.3x faster** |
| regengo (reuse) | 115.8 | 0 | 0 | **4.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 670.2 | 216 | 3 | - |
| regengo | 187.4 | 104 | 2 | **3.6x faster** |
| regengo (reuse) | 164.9 | 96 | 1 | **4.1x faster** |

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
| stdlib | 477.4 | 0 | 0 | - |
| regengo | 40.6 | 0 | 0 | **11.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 607.4 | 112 | 2 | - |
| regengo | 330.9 | 48 | 1 | **1.8x faster** |
| regengo (reuse) | 298.2 | 0 | 0 | **2.0x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 683.1 | 120 | 3 | - |
| regengo | 380.0 | 56 | 2 | **1.8x faster** |
| regengo (reuse) | 333.5 | 0 | 0 | **2.0x faster** |

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
| stdlib | 248.1 | 0 | 0 | - |
| regengo | 12.2 | 0 | 0 | **20.3x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 423.6 | 128 | 2 | - |
| regengo | 42.0 | 64 | 1 | **10.1x faster** |
| regengo (reuse) | 14.8 | 0 | 0 | **28.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 11021.9 | 4826 | 39 | - |
| regengo | 1611.4 | 2930 | 38 | **6.8x faster** |
| regengo (reuse) | 354.0 | 0 | 0 | **31.1x faster** |

**Replace `$month/$day/$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 12928.8 | 3775 | 41 | - |
| regengo | 2103.8 | 1909 | 9 | **6.1x faster** |
| regengo (precompiled) | 1582.4 | 1141 | 4 | **8.2x faster** |
| regengo (reuse) | 1149.0 | 0 | 0 | **11.3x faster** |

**Replace `[DATE]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 8061.8 | 929 | 6 | - |
| regengo | 1196.3 | 757 | 6 | **6.7x faster** |
| regengo (precompiled) | 1058.3 | 661 | 4 | **7.6x faster** |
| regengo (reuse) | 830.0 | 0 | 0 | **9.7x faster** |

**Replace `$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 11256.4 | 2772 | 40 | - |
| regengo | 1112.0 | 458 | 5 | **10.1x faster** |
| regengo (precompiled) | 1001.2 | 362 | 3 | **11.2x faster** |
| regengo (reuse) | 850.6 | 0 | 0 | **13.2x faster** |

### ReplaceEmail

Email replacement with capture group references

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 492.8 | 0 | 0 | - |
| regengo | 12.6 | 0 | 0 | **39.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 729.0 | 128 | 2 | - |
| regengo | 267.3 | 64 | 1 | **2.7x faster** |
| regengo (reuse) | 241.4 | 0 | 0 | **3.0x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 7120.5 | 1466 | 14 | - |
| regengo | 2476.0 | 902 | 13 | **2.9x faster** |
| regengo (reuse) | 2123.8 | 0 | 0 | **3.4x faster** |

**Replace `$user@REDACTED.$tld`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 7575.8 | 1653 | 16 | - |
| regengo | 2944.2 | 1099 | 8 | **2.6x faster** |
| regengo (precompiled) | 2716.5 | 714 | 4 | **2.8x faster** |
| regengo (reuse) | 2475.8 | 0 | 0 | **3.1x faster** |

**Replace `[EMAIL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 5840.6 | 349 | 5 | - |
| regengo | 2662.0 | 328 | 5 | **2.2x faster** |
| regengo (precompiled) | 2573.2 | 232 | 3 | **2.3x faster** |
| regengo (reuse) | 2460.5 | 0 | 0 | **2.4x faster** |

**Replace `$0`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 7549.5 | 1330 | 15 | - |
| regengo | 2723.0 | 516 | 5 | **2.8x faster** |
| regengo (precompiled) | 2652.1 | 419 | 3 | **2.8x faster** |
| regengo (reuse) | 2528.3 | 0 | 0 | **3.0x faster** |

### ReplaceURL

URL redaction with selective capture group output

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 428.8 | 0 | 0 | - |
| regengo | 14.5 | 0 | 0 | **29.6x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 583.6 | 160 | 2 | - |
| regengo | 169.0 | 80 | 1 | **3.5x faster** |
| regengo (reuse) | 131.3 | 0 | 0 | **4.4x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1244.4 | 404 | 5 | - |
| regengo | 402.9 | 312 | 5 | **3.1x faster** |
| regengo (reuse) | 341.7 | 266 | 3 | **3.6x faster** |

**Replace `$protocol://$host[REDACTED]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1400.4 | 442 | 8 | - |
| regengo | 625.9 | 546 | 8 | **2.2x faster** |
| regengo (precompiled) | 383.1 | 162 | 4 | **3.7x faster** |
| regengo (reuse) | 276.5 | 0 | 0 | **5.1x faster** |

**Replace `[URL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1067.0 | 112 | 5 | - |
| regengo | 441.8 | 162 | 5 | **2.4x faster** |
| regengo (precompiled) | 344.3 | 66 | 3 | **3.1x faster** |
| regengo (reuse) | 260.6 | 0 | 0 | **4.1x faster** |

**Replace `$host`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1425.8 | 311 | 7 | - |
| regengo | 454.5 | 170 | 5 | **3.1x faster** |
| regengo (precompiled) | 370.6 | 74 | 3 | **3.8x faster** |
| regengo (reuse) | 279.7 | 0 | 0 | **5.1x faster** |

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
