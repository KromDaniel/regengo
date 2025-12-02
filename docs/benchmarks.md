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
| stdlib | 1428.0 | 0 | 0 | - |
| regengo | 493.7 | 0 | 0 | **2.9x faster** |

### Greedy

Greedy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 712.0 | 0 | 0 | - |
| regengo | 152.9 | 0 | 0 | **4.7x faster** |

### Lazy

Lazy quantifier with alternation

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1195.0 | 0 | 0 | - |
| regengo | 148.4 | 0 | 0 | **8.1x faster** |

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
| stdlib | 201.5 | 0 | 0 | - |
| regengo | 12.0 | 0 | 0 | **16.8x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 292.5 | 386 | 6 | - |
| regengo | 56.2 | 192 | 3 | **5.2x faster** |
| regengo (reuse) | 21.0 | 0 | 0 | **13.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 508.1 | 1111 | 9 | - |
| regengo | 89.8 | 216 | 6 | **5.7x faster** |
| regengo (reuse) | 24.7 | 0 | 0 | **20.6x faster** |

### EmailCapture

Email pattern with named capture groups

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 731.3 | 0 | 0 | - |
| regengo | 175.1 | 0 | 0 | **4.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 862.6 | 384 | 6 | - |
| regengo | 351.7 | 192 | 3 | **2.5x faster** |
| regengo (reuse) | 318.9 | 0 | 0 | **2.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1277.0 | 1108 | 9 | - |
| regengo | 417.3 | 216 | 6 | **3.1x faster** |
| regengo (reuse) | 356.3 | 0 | 0 | **3.6x faster** |

### URLCapture

URL pattern with protocol, host, port, and path capture groups

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 763.6 | 0 | 0 | - |
| regengo | 86.0 | 0 | 0 | **8.9x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 865.9 | 481 | 6 | - |
| regengo | 339.5 | 240 | 3 | **2.6x faster** |
| regengo (reuse) | 304.2 | 0 | 0 | **2.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1055.0 | 1205 | 9 | - |
| regengo | 398.8 | 264 | 6 | **2.6x faster** |
| regengo (reuse) | 350.3 | 240 | 3 | **3.0x faster** |

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
| stdlib | 611.0 | 0 | 0 | - |
| regengo | 43.2 | 0 | 0 | **14.2x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 807.4 | 513 | 8 | - |
| regengo | 125.6 | 256 | 4 | **6.4x faster** |
| regengo (reuse) | 91.1 | 64 | 1 | **8.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1315.0 | 1477 | 12 | - |
| regengo | 203.8 | 288 | 8 | **6.5x faster** |
| regengo (reuse) | 125.6 | 0 | 0 | **10.5x faster** |

### MultiEmail

Find multiple email addresses in text

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1465.0 | 0 | 0 | - |
| regengo | 256.7 | 0 | 0 | **5.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1991.0 | 513 | 8 | - |
| regengo | 712.5 | 256 | 4 | **2.8x faster** |
| regengo (reuse) | 690.9 | 64 | 1 | **2.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2880.0 | 1604 | 14 | - |
| regengo | 1330.0 | 368 | 10 | **2.2x faster** |
| regengo (reuse) | 1247.0 | 0 | 0 | **2.3x faster** |

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
| stdlib | 1543.0 | 0 | 0 | - |
| regengo | 135.0 | 0 | 0 | **11.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1755.0 | 866 | 6 | - |
| regengo | 766.1 | 432 | 3 | **2.3x faster** |
| regengo (reuse) | 715.9 | 0 | 0 | **2.5x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1950.0 | 1589 | 9 | - |
| regengo | 794.2 | 456 | 6 | **2.5x faster** |
| regengo (reuse) | 717.1 | 0 | 0 | **2.7x faster** |

### TDFALogParser

Log line parser with multiple optional groups

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 963.4 | 0 | 0 | - |
| regengo | 114.7 | 0 | 0 | **8.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1114.0 | 577 | 6 | - |
| regengo | 331.2 | 288 | 3 | **3.4x faster** |
| regengo (reuse) | 304.4 | 0 | 0 | **3.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1322.0 | 1300 | 9 | - |
| regengo | 364.1 | 312 | 6 | **3.6x faster** |
| regengo (reuse) | 311.7 | 0 | 0 | **4.2x faster** |

### TDFANestedWord

Nested quantifiers with word boundaries

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1758.0 | 0 | 0 | - |
| regengo | 289.9 | 0 | 0 | **6.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1942.0 | 336 | 6 | - |
| regengo | 1157.0 | 144 | 3 | **1.7x faster** |
| regengo (reuse) | 1128.0 | 0 | 0 | **1.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2127.0 | 1011 | 9 | - |
| regengo | 1382.0 | 168 | 6 | **1.5x faster** |
| regengo (reuse) | 1326.0 | 0 | 0 | **1.6x faster** |

### TDFAPathological

Classic (a+)+b pattern - O(2^n) without TDFA, O(n) with TDFA

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1660.0 | 0 | 0 | - |
| regengo | 476.5 | 0 | 0 | **3.5x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1849.0 | 336 | 6 | - |
| regengo | 3999.0 | 144 | 3 | **2.2x slower** |
| regengo (reuse) | 3231.0 | 48 | 1 | **1.7x slower** |

> **Note:** For FindString on this pathological pattern, regengo's TDFA memoization adds overhead when tracking capture group positions. This is a known tradeoff — memoization prevents exponential O(2^n) blowup on matching inputs but adds constant overhead. MatchString (which doesn't track captures) is 3.5x faster. See [Analysis](analysis.md#where-regengo-may-be-slower) for details.

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 2023.0 | 1010 | 9 | - |
| regengo | 1169.0 | 168 | 6 | **1.7x faster** |
| regengo (reuse) | 1101.0 | 0 | 0 | **1.8x faster** |

### TDFASemVer

Semantic version with optional pre-release and build metadata

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 642.4 | 0 | 0 | - |
| regengo | 48.9 | 0 | 0 | **13.1x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 811.2 | 770 | 8 | - |
| regengo | 285.4 | 384 | 4 | **2.8x faster** |
| regengo (reuse) | 221.3 | 0 | 0 | **3.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1101.0 | 1737 | 12 | - |
| regengo | 325.9 | 416 | 8 | **3.4x faster** |
| regengo (reuse) | 289.1 | 384 | 4 | **3.8x faster** |

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
| stdlib | 388.7 | 0 | 0 | - |
| regengo | 42.0 | 0 | 0 | **9.3x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 454.3 | 224 | 4 | - |
| regengo | 260.0 | 96 | 2 | **1.7x faster** |
| regengo (reuse) | 243.8 | 0 | 0 | **1.9x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 578.7 | 675 | 6 | - |
| regengo | 315.2 | 112 | 4 | **1.8x faster** |
| regengo (reuse) | 272.6 | 0 | 0 | **2.1x faster** |

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
| stdlib | 365.2 | 0 | 0 | - |
| regengo | 21.0 | 0 | 0 | **17.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 512.6 | 385 | 6 | - |
| regengo | 76.9 | 192 | 3 | **6.7x faster** |
| regengo (reuse) | 43.3 | 0 | 0 | **11.8x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 14128.0 | 21650 | 213 | - |
| regengo | 2410.0 | 8792 | 114 | **5.9x faster** |
| regengo (reuse) | 850.0 | 0 | 0 | **16.6x faster** |

**Replace `$month/$day/$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 18610.0 | 11332 | 123 | - |
| regengo | 3729.0 | 5728 | 29 | **5.0x faster** |
| regengo (precompiled) | 2945.0 | 3424 | 14 | **6.3x faster** |
| regengo (reuse) | 2097.0 | 1208 | 3 | **8.9x faster** |

**Replace `[DATE]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 10837.0 | 2789 | 18 | - |
| regengo | 2099.0 | 2272 | 18 | **5.2x faster** |
| regengo (precompiled) | 1827.0 | 1984 | 12 | **5.9x faster** |
| regengo (reuse) | 1660.0 | 1208 | 3 | **6.5x faster** |

**Replace `$year`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 13757.0 | 8305 | 120 | - |
| regengo | 2047.0 | 1376 | 17 | **6.7x faster** |
| regengo (precompiled) | 1845.0 | 1088 | 11 | **7.5x faster** |
| regengo (reuse) | 1761.0 | 1208 | 3 | **7.8x faster** |

### ReplaceEmail

Email replacement with capture group references

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1174.0 | 0 | 0 | - |
| regengo | 206.6 | 0 | 0 | **5.7x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1516.0 | 641 | 10 | - |
| regengo | 611.3 | 320 | 5 | **2.5x faster** |
| regengo (reuse) | 564.2 | 0 | 0 | **2.7x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 15082.0 | 11700 | 116 | - |
| regengo | 5930.0 | 4508 | 65 | **2.5x faster** |
| regengo (reuse) | 5129.0 | 0 | 0 | **2.9x faster** |

**Replace `$user@REDACTED.$tld`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 16603.0 | 8282 | 84 | - |
| regengo | 6921.0 | 5496 | 40 | **2.4x faster** |
| regengo (precompiled) | 6199.0 | 3573 | 20 | **2.7x faster** |
| regengo (reuse) | 5901.0 | 1001 | 5 | **2.8x faster** |

**Replace `[EMAIL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 12751.0 | 1742 | 25 | - |
| regengo | 6097.0 | 1642 | 25 | **2.1x faster** |
| regengo (precompiled) | 5871.0 | 1161 | 15 | **2.2x faster** |
| regengo (reuse) | 5790.0 | 1001 | 5 | **2.2x faster** |

**Replace `$0`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 15151.0 | 6648 | 79 | - |
| regengo | 6140.0 | 2579 | 25 | **2.5x faster** |
| regengo (precompiled) | 6060.0 | 2099 | 15 | **2.5x faster** |
| regengo (reuse) | 5844.0 | 1001 | 5 | **2.6x faster** |

### ReplaceURL

URL redaction with selective capture group output

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**MatchString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 576.4 | 0 | 0 | - |
| regengo | 170.9 | 0 | 0 | **3.4x faster** |

**FindString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 688.0 | 481 | 6 | - |
| regengo | 257.8 | 240 | 3 | **2.7x faster** |
| regengo (reuse) | 215.8 | 0 | 0 | **3.2x faster** |

**FindAllString:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1504.0 | 1846 | 17 | - |
| regengo | 659.4 | 936 | 16 | **2.3x faster** |
| regengo (reuse) | 569.7 | 800 | 10 | **2.6x faster** |

**Replace `$protocol://$host[REDACTED]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1925.0 | 1327 | 26 | - |
| regengo | 1086.0 | 1640 | 25 | **1.8x faster** |
| regengo (precompiled) | 689.2 | 488 | 13 | **2.8x faster** |
| regengo (reuse) | 544.6 | 176 | 3 | **3.5x faster** |

**Replace `[URL]`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1359.0 | 338 | 15 | - |
| regengo | 724.7 | 488 | 15 | **1.9x faster** |
| regengo (precompiled) | 601.4 | 200 | 9 | **2.3x faster** |
| regengo (reuse) | 532.4 | 176 | 3 | **2.6x faster** |

**Replace `$host`:**

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1689.0 | 932 | 22 | - |
| regengo | 766.3 | 512 | 15 | **2.2x faster** |
| regengo (precompiled) | 620.3 | 224 | 9 | **2.7x faster** |
| regengo (reuse) | 534.2 | 176 | 3 | **3.2x faster** |

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
go test ./benchmarks/curated/... -bench="Input\[0\]" -benchmem
```

### Aggregating Results

Use the aggregation script for summary statistics across all inputs:

```bash
# Aggregate results for a pattern
go test ./benchmarks/curated/... -bench="BenchmarkDateCapture" -benchmem | go run scripts/curated/aggregate.go

# Example output:
# Pattern: DateCapture
#   Category: Match
#     stdlib:            avg=   73.86 ns  min=   73.15  max=   74.28  allocs=0
#     regengo:           avg=    3.91 ns  min=    3.86  max=    4.01  allocs=0  (18.9x faster)
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
