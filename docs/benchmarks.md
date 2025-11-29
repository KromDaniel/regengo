# Detailed Benchmarks

Benchmarks run on Apple M4 Pro. Each benchmark shows performance for Go stdlib vs regengo.

## Summary

| Pattern Type | Typical Speedup | Memory Reduction |
|--------------|-----------------|------------------|
| Simple match | 2-3x faster | 0 allocs |
| Capture groups | 2-5x faster | 50% fewer allocs |
| FindAll | 5-9x faster | 50-100% fewer allocs |
| With reuse API | 10-15x faster | 0 allocs |

## DateCaptureFindString

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 105.3 | 128 | 2 | - |
| regengo | 19.7 | 64 | 1 | **5.3x faster** |
| regengo (reuse) | 7.3 | 0 | 0 | **14.4x faster** |

## EmailCaptureFindString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 295.9 | 128 | 2 | - |
| regengo | 127.3 | 64 | 1 | **2.3x faster** |
| regengo (reuse) | 115.1 | 0 | 0 | **2.6x faster** |

## EmailMatchString

**Pattern:**
```regex
[\w\.+-]+@[\w\.-]+\.[\w\.-]+
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1554.0 | 0 | 0 | - |
| regengo | 507.3 | 0 | 0 | **3.1x faster** |

## GreedyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)*abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 751.4 | 0 | 0 | - |
| regengo | 475.3 | 0 | 0 | **1.6x faster** |

## LazyMatchString

**Pattern:**
```regex
(?:(?:a|b)|(?:k)+)+?abcd
```

**Method:** `MatchString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 1260.0 | 0 | 0 | - |
| regengo | 478.0 | 0 | 0 | **2.6x faster** |

## MultiDateFindAllString

**Pattern:**
```regex
(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 431.4 | 331 | 3 | - |
| regengo | 82.2 | 106 | 2 | **5.2x faster** |
| regengo (reuse) | 48.6 | 0 | 0 | **8.9x faster** |

## MultiEmailFindAllString

**Pattern:**
```regex
(?P<user>[\w\.+-]+)@(?P<domain>[\w\.-]+)\.(?P<tld>[\w\.-]+)
```

**Method:** `FindAllString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 978.1 | 374 | 4 | - |
| regengo | 478.1 | 133 | 3 | **2.0x faster** |
| regengo (reuse) | 438.5 | 0 | 0 | **2.2x faster** |

## TDFAComplexURLFindString

**Pattern:**
```regex
(?P<scheme>https?)://(?P<auth>(?P<user>[\w.-]+)(?::(?P<pass>[\w.-]+))?@)?(?P<host>[\w.-]+)(?::(?P<port>\d+))?(?P<path>/[\w./-]*)?(?:\?(?P<query>[\w=&.-]+))?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 598.7 | 288 | 2 | - |
| regengo | 285.4 | 421 | 2 | **2.1x faster** |
| regengo (reuse) | 263.7 | 277 | 1 | **2.3x faster** |

## TDFALogParserFindString

**Pattern:**
```regex
(?P<timestamp>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})(?:\.(?P<ms>\d{3}))?(?P<tz>Z|[+-]\d{2}:\d{2})?\s+\[(?P<level>\w+)\]\s+(?P<message>.+)
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 398.8 | 192 | 2 | - |
| regengo | 120.9 | 96 | 1 | **3.3x faster** |
| regengo (reuse) | 106.0 | 0 | 0 | **3.8x faster** |

## TDFANestedWordFindString

**Pattern:**
```regex
(?P<words>(?P<word>\w+\s*)+)end
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 654.9 | 112 | 2 | - |
| regengo | 403.6 | 128 | 1 | **1.6x faster** |
| regengo (reuse) | 394.6 | 80 | 0 | **1.7x faster** |

## TDFAPathologicalFindString

**Pattern:**
```regex
(?P<outer>(?P<inner>a+)+)b
```

**Method:** `FindString`

This benchmark tests nested quantifier patterns with captures. Regengo wins on matching inputs but loses on non-matching pathological cases:

| Input | Variant | ns/op | B/op | allocs/op | vs stdlib |
|-------|---------|------:|-----:|----------:|----------:|
| matching | stdlib | 210 | 112 | 2 | - |
| matching | regengo | 119 | 48 | 1 | **1.8x faster** |
| matching | regengo (reuse) | 110 | 0 | 0 | **1.9x faster** |
| non-matching | stdlib | 1076 | 0 | 0 | - |
| non-matching | regengo | 3007 | 0 | 0 | **2.8x slower** |
| non-matching | regengo (reuse) | 3005 | 0 | 0 | **2.8x slower** |

> **Note:** For pathological non-matching inputs like `"aaaaaaaaaaaaaaaaaaaac"`, the memoization overhead in capture groups causes regengo to be slower. See [Analysis](analysis.md#where-regengo-may-be-slower).

## TDFASemVerFindString

**Pattern:**
```regex
(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:-(?P<prerelease>[\w.-]+))?(?:\+(?P<build>[\w.-]+))?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 211.3 | 192 | 2 | - |
| regengo | 72.6 | 96 | 1 | **2.9x faster** |
| regengo (reuse) | 57.1 | 0 | 0 | **3.7x faster** |

## URLCaptureFindString

**Pattern:**
```regex
(?P<protocol>https?)://(?P<host>[\w\.-]+)(?::(?P<port>\d+))?(?P<path>/[\w\./]*)?
```

**Method:** `FindString`

| Variant | ns/op | B/op | allocs/op | vs stdlib |
|---------|------:|-----:|----------:|----------:|
| stdlib | 293.8 | 160 | 2 | - |
| regengo | 116.6 | 80 | 1 | **2.5x faster** |
| regengo (reuse) | 112.7 | 0 | 0 | **2.6x faster** |

---

## Running Benchmarks

To run benchmarks yourself:

```bash
# Generate benchmark code
make bench-gen

# Run benchmarks
make bench

# Generate markdown output
make bench-readme

# Analyze benchmark results
make bench-analyze
```

## Regenerating Results

To regenerate these benchmark tables:

```bash
make bench-readme
```
