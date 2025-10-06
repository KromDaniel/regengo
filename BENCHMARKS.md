# Benchmark Results - Quick Start

This file provides quick access to the benchmark results and comparison tools.

## 📊 Latest Benchmark Results

**Current Performance (2025-01-05):**

- **42.2% faster** than stdlib overall
- **132/185 patterns faster** (71.4% win rate)
- **12.8% less memory** than stdlib

### By Category

- **Simple patterns:** 95.2% faster (90/90 wins)
- **Complex patterns:** 1.5% slower avg (23/60 wins)
- **Very complex patterns:** 14.9% faster (19/35 wins)

**Full results:** See `BENCHMARK_RESULTS.txt`

## 🚀 Quick Commands

### Run New Benchmarks

```bash
# Run and save to file (takes ~1-2 minutes)
go run benchmarks/mass_generator.go > /tmp/new_results.txt 2>&1

# View progress in real-time
go run benchmarks/mass_generator.go 2>&1 | tee /tmp/new_results.txt
```

### Compare Results

```bash
# Compare new results against baseline
./benchmarks/compare_benchmarks.sh /tmp/new_results.txt

# Compare two specific files
./benchmarks/compare_benchmarks.sh /tmp/new_results.txt BENCHMARK_RESULTS.txt
```

### Update Baseline

```bash
# After verifying new results are good, update baseline
cp /tmp/new_results.txt BENCHMARK_RESULTS.txt
git add BENCHMARK_RESULTS.txt
git commit -m "Update benchmark results after [optimization/change]"
```

## 📁 File Organization

### Main Files

- **BENCHMARK_RESULTS.txt** - Current benchmark baseline (commit this)
- **BENCHMARKS_README.md** - Detailed guide to benchmarking
- **benchmarks/mass_generator.go** - Benchmark generation tool
- **benchmarks/compare_benchmarks.sh** - Script to compare results

### Documentation

- **OPTIMIZATION_RESULTS.md** - Complete analysis of implemented optimizations
- **docs/OPTIMIZATION_1_CAPTURE_CHECKPOINT.md** - Capture checkpoint system details
- **docs/OPTIMIZATION_2_UNROLL_REPETITIONS.md** - AST unrolling details
- **docs/COMPLETE_OPTIMIZATION_REPORT.md** - Full implementation report

## 🎯 Performance Targets

### Must Maintain

- Overall: ≥40% faster than stdlib ✅
- Simple: ≥90% faster ✅
- Complex: Within 10% of stdlib ✅
- Very complex: ≥10% faster ✅

### Regression Alerts

⚠️ Alert if:

- Overall drops below 40%
- Simple drops below 90%
- Complex slower by >10%
- Very complex slower by >5%

## 🔍 Quick Check

After making changes, run this quick check:

```bash
# Quick benchmark and comparison
go run benchmarks/mass_generator.go > /tmp/quick_bench.txt 2>&1 && \
./benchmarks/compare_benchmarks.sh /tmp/quick_bench.txt && \
echo "✅ Check complete!"
```

Expected output should show:

- ✅ Overall performance maintained or improved
- ✅ Win rate stable or increased
- ✅ No major regressions in any category

## 📈 Benchmark History

### Current (v0 + Opt #1 + Opt #2) - 2025-01-05

- **42.2%** faster overall
- **132/185** wins
- Optimizations: Capture checkpoints + Unroll {2},{3}

### Baseline (v0)

- **46.3%** faster overall (but unstable for complex patterns)
- Had issues with capture reset overhead
- No small repetition optimization

## 🛠️ Troubleshooting

### Benchmarks take too long

```bash
# Run just one category at a time
# Edit benchmarks/mass_generator.go:
# - Comment out generateComplexSpecs() and generateVeryComplexSpecs()
# - Run: go run benchmarks/mass_generator.go > /tmp/simple_only.txt 2>&1
```

### Results seem inconsistent

```bash
# Run multiple times and average
for i in {1..3}; do
    echo "Run $i of 3..."
    go run benchmarks/mass_generator.go > /tmp/bench_run_$i.txt 2>&1
done

# Then manually compare the OVERALL SUMMARY sections
```

### Memory issues

```bash
# Reduce pattern count in benchmarks/mass_generator.go
# In buildPatternSpecs(), change:
#   generateSimpleSpecs(90) -> generateSimpleSpecs(30)
#   generateComplexSpecs(45) -> generateComplexSpecs(15)
#   generateVeryComplexSpecs(20) -> generateVeryComplexSpecs(10)
```

## 📚 Learn More

- **BENCHMARKS_README.md** - Complete benchmarking guide
- **OPTIMIZATION_RESULTS.md** - Optimization analysis and results
- **docs/** - Detailed optimization documentation

## 🤝 Contributing

When submitting PRs with performance changes:

1. Run benchmarks before changes
2. Make your changes
3. Run benchmarks after changes
4. Use `benchmarks/compare_benchmarks.sh` to show improvement
5. Include comparison in PR description
6. Update BENCHMARK_RESULTS.txt if significant improvement

---

**Last Updated:** 2025-01-05  
**Test Count:** 155 patterns (795 test cases)  
**Hardware:** Apple M4 Pro  
**Status:** ✅ All optimizations complete and verified
