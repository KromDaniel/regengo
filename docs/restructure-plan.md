# Regengo Project Restructuring Plan

## Current State Analysis

### Directory Structure Issues

```
CURRENT                          PROBLEMS
├── e2e/                         Main e2e tests location
│   ├── streaming/               Streaming tests
│   └── testdata.json            Pattern test data
├── test/e2e/streaming/          DUPLICATE - empty dirs, confusing
├── tests/integration/           DUPLICATE - findall_test.go same as benchmarks/
├── benchmarks/
│   ├── generated/               Curated benchmark patterns
│   ├── findall_test.go          DUPLICATE with tests/integration/
│   └── compare_benchmarks.sh    Unclear purpose
├── cmd/
│   ├── regengo/                 Main CLI (clear)
│   ├── mass_generator/          Unclear - generates patterns for benchmarking
│   ├── curated_generator/       Unclear - generates benchmark patterns
│   └── bench_to_readme/         Unclear - formats benchmark output
├── scripts/
│   ├── manage_e2e_test.py       Manages e2e tests
│   └── migrate_testdata.py      One-time migration (can delete)
└── docs/                        EMPTY
```

### README Issues
- **937 lines** - far too long for a README
- Mixes quickstart, deep technical details, and API reference
- Jumps between unrelated topics
- Hard to find what you need

---

## Proposed Structure

### New Directory Layout

```
regengo/
├── cmd/
│   └── regengo/                 # Main CLI only
├── pkg/regengo/                 # Public library API
├── internal/
│   ├── compiler/                # Core compilation logic
│   └── codegen/                 # Code generation
├── stream/                      # Streaming API package
├── testdata/                    # All test data (renamed from e2e/)
│   ├── patterns.json            # Pattern test cases
│   └── streaming/               # Streaming test patterns
├── benchmarks/
│   ├── generated/               # Generated benchmark patterns
│   └── results/                 # Benchmark result artifacts
├── examples/
│   └── streaming/               # Working examples
├── scripts/
│   ├── generate_benchmarks.go   # Consolidate generators
│   └── benchmark_chart.py       # Generate performance charts
├── docs/
│   ├── streaming.md             # Streaming API guide
│   ├── analysis.md              # Analysis & complexity
│   ├── unicode.md               # Unicode support
│   ├── benchmarks.md            # Detailed benchmarks
│   └── testing.md               # Testing guide
└── README.md                    # Concise (~200 lines)
```

---

## Phase 1: Clean Up Duplicates & Dead Code

### 1.1 Remove Duplicate Test Directories
```bash
# Remove empty/duplicate directories
rm -rf test/                     # Empty nested structure
rm -rf tests/                    # Duplicate of benchmarks/
```

### 1.2 Rename e2e/ to testdata/
- `e2e/` → `testdata/` (more idiomatic Go naming)
- Update all imports

### 1.3 Consolidate cmd/ Tools
**Keep:**
- `cmd/regengo/` - Main CLI

**Merge into scripts/:**
- `cmd/mass_generator/` → Delete (duplicates e2e logic)
- `cmd/curated_generator/` → `scripts/generate_benchmarks.go`
- `cmd/bench_to_readme/` → `scripts/format_benchmarks.go`

### 1.4 Clean Up Scripts
```bash
# Remove one-time migration script
rm scripts/migrate_testdata.py

# Move benchmark shell script
mv benchmarks/compare_benchmarks.sh scripts/
```

---

## Phase 2: Documentation Restructure

### 2.1 New README.md (~200 lines)

```markdown
# Regengo

[badges]
[logo]

Brief description (2-3 sentences).

## Installation
## Quick Start
## Performance
  - Summary table (5-7 patterns)
  - Performance chart image
  - Link to detailed benchmarks
## Usage
  - CLI example
  - Library example
## Generated Output
  - Basic methods
  - Streaming methods
## Capture Groups
## Documentation
  - [Streaming API](docs/streaming.md)
  - [Analysis & Complexity](docs/analysis.md)
  - [Unicode Support](docs/unicode.md)
  - [Detailed Benchmarks](docs/benchmarks.md)
  - [Testing Guide](docs/testing.md)
## License
```

### 2.2 docs/streaming.md
- Complete streaming API guide
- Configuration options
- Memory usage patterns
- Examples with io.Reader sources

### 2.3 docs/analysis.md
Consolidate from current README:
- Smart Analysis section
- Complexity Guarantees section
- Advanced Options section
- `-analyze` and `-verbose` flags

### 2.4 docs/unicode.md
- Unicode character class support
- Multibyte handling
- Examples

### 2.5 docs/benchmarks.md
- Detailed benchmark tables
- Methodology
- How to run benchmarks
- Historical performance data

### 2.6 docs/testing.md
- How to run tests
- Test structure explanation
- Adding new test patterns
- CI/CD testing

---

## Phase 3: Scripts Consolidation

### 3.1 scripts/generate_benchmarks.go
Consolidate `curated_generator` and `mass_generator`:
```go
// Single tool with subcommands:
// go run scripts/generate_benchmarks.go curated    # For benchmarks/generated/
// go run scripts/generate_benchmarks.go mass       # For stress testing
```

**IMPORTANT**: Preserve benchmark analysis/summary logic from mass_generator:
- `printSummary()` - Category breakdown (simple/complex/very_complex/tdfa)
- `printBenchmarkAnalysis()` - Regengo vs stdlib comparison
- `analyzeBenchmarks()` - Parse benchmark output, compute stats

### 3.2 scripts/benchmark_chart.py
New script to generate performance chart for README:
```python
# Reads benchmark output, generates PNG chart
# Output: assets/benchmark_chart.png
```

### 3.3 scripts/format_benchmarks.go
Renamed from `bench_to_readme`:
```go
// Formats benchmark output for docs/benchmarks.md
```

---

## Phase 4: Final Cleanup

### 4.1 Files to Delete
```
coverage.txt                     # Generated, add to .gitignore
e2e_final_coverage.txt           # Generated, add to .gitignore
regengo (binary)                 # Add to .gitignore
streaming (binary)               # Add to .gitignore
```

### 4.2 Update .gitignore
```gitignore
# Add
coverage.txt
*_coverage.txt
/regengo
/streaming
```

### 4.3 Update Makefile
- Remove references to deleted commands
- Add `make docs` target
- Add `make benchmark-chart` target

---

## Migration Checklist

- [ ] **Phase 1.1**: Remove `test/` and `tests/` directories
- [ ] **Phase 1.2**: Rename `e2e/` to `testdata/`, update imports
- [ ] **Phase 1.3**: Delete `cmd/mass_generator/`, merge `curated_generator` and `bench_to_readme` to scripts/
- [ ] **Phase 1.4**: Clean up scripts, delete `migrate_testdata.py`
- [ ] **Phase 2.1**: Create concise README.md
- [ ] **Phase 2.2**: Create docs/streaming.md
- [ ] **Phase 2.3**: Create docs/analysis.md
- [ ] **Phase 2.4**: Create docs/unicode.md
- [ ] **Phase 2.5**: Create docs/benchmarks.md
- [ ] **Phase 2.6**: Create docs/testing.md
- [ ] **Phase 3.1**: Create scripts/generate_benchmarks.go
- [ ] **Phase 3.2**: Create scripts/benchmark_chart.py
- [ ] **Phase 4**: Final cleanup, update .gitignore, Makefile

---

## Before/After Comparison

### Before (confusing)
```
cmd/mass_generator/          # What is this?
cmd/curated_generator/       # How is this different?
cmd/bench_to_readme/         # What does this do?
e2e/                         # Tests? Benchmarks?
test/e2e/                    # More tests?
tests/integration/           # Same as benchmarks?
docs/                        # Empty!
README.md                    # 937 lines, overwhelming
```

### After (clear)
```
cmd/regengo/                 # The CLI
scripts/                     # Development tools
testdata/                    # Test fixtures
benchmarks/                  # Performance testing
docs/                        # Complete documentation
README.md                    # Concise entry point
```

---

## Phase 5: CI/CD Updates

### 5.1 Update .github/workflows/ci.yml
Current issue: References `./tests/...` which will be deleted.

```yaml
# Before:
go test ... ./tests/...

# After:
go test ... ./benchmarks/...
```

---

## Risk Mitigation

1. **Breaking imports**: Create git tag before restructure
2. **Lost functionality**: Test all scripts before deleting cmd tools
3. **Documentation gaps**: Review current README sections against new docs
4. **CI/CD breakage**: Update GitHub workflows in same PR

---

## Estimated Effort

| Phase | Tasks | Complexity |
|-------|-------|------------|
| Phase 1 | Clean duplicates | Low |
| Phase 2 | Documentation | Medium |
| Phase 3 | Scripts | Medium |
| Phase 4 | Cleanup | Low |

**Total**: Can be done incrementally over 2-3 PRs
