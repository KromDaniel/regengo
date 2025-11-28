# Testing Guide

## Generated Tests & Benchmarks

Regengo automatically generates a `_test.go` file alongside your output file (unless disabled). This file contains:

### Correctness Tests

Verifies that Regengo's output matches `regexp` stdlib exactly for provided inputs:

- `Test...MatchString`: Validates boolean matching
- `Test...MatchBytes`: Validates byte-slice matching
- `Test...FindString`: Validates capture groups (if present), checking both the full match and every individual captured group against stdlib's `FindStringSubmatch`
- `Test...FindAllString`: Validates all matches and their captures against stdlib's `FindAllStringSubmatch`

### Benchmarks

Comparison benchmarks to measure speedup vs stdlib:

- `Benchmark...MatchString`: Performance of simple matching
- `Benchmark...FindString`: Performance of capture extraction (if applicable)

## Customizing Tests

You can provide specific test inputs to verify your pattern against real-world data:

### CLI

```bash
# Generates date.go and date_test.go
regengo -pattern '...' -name Date -output date.go -test-inputs "2024-01-01,2025-12-31"
```

### Library

```go
regengo.Options{
    // ...
    GenerateTestFile: true, // Required: Library defaults to false
    TestFileInputs:   []string{"2024-01-01", "2025-12-31"},
}
```

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Specific Package Tests

```bash
go test ./internal/compiler/...
go test ./pkg/regengo/...
```

### Run with Verbose Output

```bash
go test -v ./...
```

### Run with Race Detection

```bash
go test -race ./...
```

## Running Benchmarks

### Generated Benchmarks

Run the generated benchmarks using standard Go tooling:

```bash
go test -bench=. -benchmem
```

### Project Benchmarks

```bash
# Generate and run curated benchmarks
make bench

# Run with analysis summary
make bench-analyze
```

### Specific Benchmark

```bash
go test -bench=BenchmarkDate -benchmem ./benchmarks/generated/
```

## Test Structure

```
regengo/
├── internal/
│   ├── compiler/
│   │   └── *_test.go     # Unit tests for compiler
│   └── codegen/
│       └── *_test.go     # Unit tests for code generation
├── pkg/regengo/
│   └── *_test.go         # Public API tests
├── stream/
│   └── *_test.go         # Streaming API tests
├── testdata/
│   ├── e2e_test.go       # End-to-end pattern tests
│   ├── testdata.json     # Test pattern definitions
│   └── streaming/        # Streaming-specific tests
└── benchmarks/
    ├── generated/        # Generated benchmark patterns
    └── findall_test.go   # FindAll comparison tests
```

## Adding New Test Patterns

### Via testdata.json

Edit `testdata/testdata.json` to add new patterns:

```json
{
  "patterns": [
    {
      "regex": "your-pattern-here",
      "inputs": [
        {"text": "matching-input", "shouldMatch": true},
        {"text": "non-matching", "shouldMatch": false}
      ]
    }
  ]
}
```

### Via Test Script

```bash
python scripts/manage_e2e_test.py add "your-pattern" "test-input-1" "test-input-2"
```

## Coverage

### Generate Coverage Report

```bash
make coverage
```

This generates and opens `coverage.html`.

### Coverage in CI

The CI pipeline runs with coverage and uploads to Codecov:

```bash
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

## CI/CD Testing

The GitHub Actions workflow runs:

1. **Format check**: `gofmt -s -d .`
2. **Vet**: `go vet ./...`
3. **Lint**: `golangci-lint run`
4. **Build**: `go build -v ./cmd/regengo`
5. **CLI test**: Generate and verify a test pattern
6. **Tests**: `go test -v -race -coverprofile=coverage.txt ./...`

## Debugging Test Failures

### Verbose Analysis

Use `-verbose` to see engine selection:

```bash
regengo -pattern 'your-pattern' -name Test -output test.go -verbose
```

### Analyze Mode

Check pattern characteristics without generating code:

```bash
regengo -analyze -pattern 'your-pattern'
```

### Compare with Stdlib

All generated tests compare against `regexp` stdlib. If a test fails, it indicates a behavioral difference between regengo and stdlib.
