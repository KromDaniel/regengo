# Testing Regengo

This guide covers testing the regengo project itself. For documentation on the test files regengo generates for users, see [Generated Tests](../testing.md).

## Running Tests

### All Tests

```bash
go test ./...
```

### Specific Packages

```bash
go test ./internal/compiler/...
go test ./pkg/regengo/...
go test ./stream/...
```

### With Options

```bash
# Verbose output
go test -v ./...

# Race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.txt -covermode=atomic ./...
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
├── tests/
│   └── e2e/
│       ├── e2e_test.go   # End-to-end pattern tests
│       └── testdata.json # Test pattern definitions
└── benchmarks/
    ├── generated/        # Generated benchmark patterns
    └── findall_test.go   # FindAll comparison tests
```

## End-to-End Tests

The `tests/e2e/` directory contains comprehensive e2e tests that:

1. Generate code for each pattern in `testdata.json`
2. Run the generated tests
3. Verify correctness against stdlib

### Running E2E Tests

```bash
# Run all e2e tests
go test ./tests/e2e/...

# Run with label filter
go test ./tests/e2e/... -run "TDFA"
go test ./tests/e2e/... -run "Captures.*WordBoundary"
```

### Adding Test Patterns

#### Via Script (Recommended)

```bash
# Add a new pattern with test inputs
python scripts/manage_e2e_test.py -p '(?P<name>\w+)@\w+' -i '["test@example.com", "user@domain"]'

# Update existing pattern labels
python scripts/manage_e2e_test.py -p 'existing-pattern'
```

#### Via testdata.json

Edit `tests/e2e/testdata.json`:

```json
{
  "pattern": "your-pattern-here",
  "inputs": ["matching-input", "non-matching"],
  "feature_labels": ["Captures", "CharClass"],
  "engine_labels": ["TDFA"]
}
```

## Benchmarks

### Project Benchmarks

```bash
# Generate and run curated benchmarks
make bench

# Run with analysis summary
make bench-analyze

# Generate markdown output
make bench-readme
```

### Specific Benchmark

```bash
go test -bench=BenchmarkDate -benchmem ./benchmarks/generated/
```

## Coverage

### Generate Report

```bash
make coverage
```

This generates and opens `coverage.html`.

### CI Coverage

The CI pipeline runs with coverage and uploads to Codecov:

```bash
go test -v -race -coverprofile=coverage.txt -covermode=atomic \
    -coverpkg=./internal/...,./pkg/...,./tests/...,./benchmarks/... \
    ./internal/... ./pkg/... ./tests/... ./benchmarks/...
```

## CI/CD Pipeline

The GitHub Actions workflow (`.github/workflows/ci.yml`) runs:

1. **Format check**: `gofmt -s -d .`
2. **Vet**: `go vet ./...`
3. **Lint**: `golangci-lint run`
4. **Build**: `go build -v ./cmd/regengo`
5. **CLI test**: Generate and verify a test pattern
6. **Tests**: Full test suite with race detection and coverage

## Pre-commit Hooks

The project uses git hooks for quality checks:

```bash
# Install hooks
make setup-hooks
```

This runs formatting, vet, and lint checks before each commit.

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make test` | Run all tests |
| `make bench` | Run benchmarks |
| `make bench-analyze` | Run benchmarks with analysis |
| `make coverage` | Generate coverage report |
| `make lint` | Run linter |
| `make fmt` | Format code |
| `make ci` | Run full CI pipeline locally |
