# Benchmark Results

This directory contains benchmark results for the regengo regex compiler, allowing performance tracking over time.

## Files

- **BENCHMARK_RESULTS.txt** - Current benchmark results from mass_generator.go
- **OPTIMIZATION_RESULTS.md** - Detailed analysis of optimization implementations
- **mass_generator.go** - Tool for generating and benchmarking regex patterns

## Running Benchmarks

To generate new benchmark results:

```bash
# Run the mass generator (takes ~1-2 minutes)
go run mass_generator.go > BENCHMARK_RESULTS.txt 2>&1

# Or run in background
go run mass_generator.go > /tmp/mass_bench_new.txt 2>&1 &
```

## Benchmark Categories

The mass generator tests **155 patterns** across 3 categories:

### Simple Patterns (90 patterns)

- Character classes with fixed lengths: `^[a-z]{5}$`
- Digit sequences: `^\d{4}$`
- Hex patterns: `^[a-f0-9]{6}$`
- **Expected:** 95%+ faster than stdlib, zero allocations

### Complex Patterns (45 patterns)

- Alternations with repetitions: `^(?:foo|bar){2}baz\d{4}$`
- Phone numbers with captures: `^(?P<area>\d{3})-(?P<prefix>\d{3})-(?P<line>\d{4})(?: x(?P<ext>\d{3}))?$`
- Name sequences: `^(?:[A-Z][a-z]+\s){4}[A-Z][a-z]+$`
- **Expected:** Competitive with stdlib (within 10%)

### Very Complex Patterns (20 patterns)

- URL patterns with nested repetitions: `^(?P<protocol>https?)://(?P<host>(?:[a-z0-9-]+\.){1}[a-z]{2,})...`
- ISO timestamp patterns: `^(?P<year>\d{4})-(?P<month>0[1-9]|1[0-2])-...`
- API path patterns: `^/api/v1(?:/[a-z]{3,8}){5}/(?P<id>[1-9]\d{3,5})...`
- Key-value patterns: `^(?:[a-z]{3}=[0-9]{2}&){2}[a-z]{3}=[0-9]{2}$`
- **Expected:** 10-15% faster than stdlib for most patterns

## Comparing Results

To compare benchmark results between versions:

```bash
# Run new benchmarks
go run mass_generator.go > /tmp/new_results.txt 2>&1

# Compare key metrics
echo "=== PREVIOUS RESULTS ==="
grep "OVERALL SUMMARY" -A 4 BENCHMARK_RESULTS.txt

echo ""
echo "=== NEW RESULTS ==="
grep "OVERALL SUMMARY" -A 4 /tmp/new_results.txt

# Compare by category
for cat in "simple" "complex" "very_complex"; do
    echo ""
    echo "=== Category: $cat ==="
    echo "Previous:"
    grep -A 4 "Category: $cat" BENCHMARK_RESULTS.txt | head -5
    echo "New:"
    grep -A 4 "Category: $cat" /tmp/new_results.txt | head -5
done
```

## Benchmark Metrics

Each benchmark reports:

- **Avg time (ns/op)** - Average nanoseconds per operation
- **Avg memory (B/op)** - Average bytes allocated per operation
- **Avg allocs (allocs/op)** - Average number of allocations per operation
- **Win rate** - Percentage of patterns faster than stdlib

## Current Results (Latest)

**Date:** 2025-01-05  
**Optimizations:** Capture Checkpoint System + Unroll Small Repetitions

### Overall Performance

- **42.2% faster** than stdlib
- **Win rate:** 132/185 patterns (71.4%)
- **Memory:** 12.8% less than stdlib
- **Allocations:** Trade-off (zero for simple, more for complex with captures)

### By Category

| Category     | Win Rate      | Speed        | Memory    |
| ------------ | ------------- | ------------ | --------- |
| Simple       | 90/90 (100%)  | 95.2% faster | 100% less |
| Complex      | 23/60 (38.3%) | 1.5% slower  | 22% more  |
| Very Complex | 19/35 (54.3%) | 14.9% faster | 133% more |

## Performance Targets

### Minimum Acceptable Performance

- Overall: ≥40% faster than stdlib
- Simple patterns: ≥90% faster
- Complex patterns: Within 5% of stdlib
- Very complex patterns: ≥10% faster

### Regression Detection

A regression is indicated if:

- Overall speedup drops below 40%
- Simple pattern speedup drops below 90%
- Complex pattern slowdown exceeds 10%
- Very complex pattern speedup drops below 5%

## Known Performance Characteristics

### Strengths

✅ Simple patterns without alternations or captures  
✅ Fixed-length character classes  
✅ Patterns with 2-3 repetitions (unroll optimization)  
✅ Very complex patterns with good structure

### Challenges

⚠️ Complex patterns with 4+ nested repetitions  
⚠️ Patterns with many capture groups + optional groups  
⚠️ Deeply nested alternations (5+ levels)

## Optimization History

### Current (v0 + Optimizations #1 and #2)

- **Optimization #1:** Capture Checkpoint System
  - Replaces O(n×c) capture reset with O(1) checkpoint restore
  - Impact: Stabilizes complex pattern performance
- **Optimization #2:** Unroll Small Repetitions
  - AST-level transformation for {2}, {3} repetitions
  - Impact: 10% code reduction, reduced branching

### Baseline (v0)

- Generated Go code with goto-based state machine
- Character class detection optimization
- Backtracking stack with pooling
- BytesView optimization for []byte patterns

## Future Work

### Optimization #3: Specialized Loop Code (Planned)

- Target: Nested repetitions like {4}, {5}, {3,5}
- Expected impact: 10-20% improvement for complex patterns
- Would address 37/60 complex patterns where stdlib currently wins

## Contributing

When adding new optimizations:

1. Run benchmarks before changes: `go run mass_generator.go > before.txt 2>&1`
2. Implement optimization
3. Run benchmarks after changes: `go run mass_generator.go > after.txt 2>&1`
4. Compare results and document in OPTIMIZATION_RESULTS.md
5. Update this file if targets or characteristics change
6. Commit BENCHMARK_RESULTS.txt with the latest results

## Troubleshooting

### Benchmarks failing

- Check that all dependencies are installed: `go mod tidy`
- Verify tests pass first: `go test ./...`
- Check for build errors: `go build ./...`

### Results inconsistent

- CPU governor may affect results (use performance mode)
- Close resource-intensive applications
- Run multiple times and average results
- Use `-benchtime=1x` for consistency (already default in mass_generator)

### Out of memory

- Reduce pattern count in mass_generator.go
- Run categories separately (modify buildPatternSpecs())
- Increase system memory or swap

---

**Last Updated:** 2025-01-05  
**Benchmark Version:** mass_generator.go with 155 patterns  
**Go Version:** go1.x (check with `go version`)  
**Hardware:** Apple M4 Pro (update for your system)
