.PHONY: all build test bench bench-gen bench-chart clean fmt lint install help setup-hooks

# Variables
BINARY_NAME=regengo
CMD_PATH=./cmd/regengo

# Default target
all: fmt lint test build

## help: Display this help message
help:
	@echo "Regengo - Makefile commands:"
	@echo ""
	@grep -E '^##' Makefile | sed 's/##//'

## build: Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o bin/$(BINARY_NAME) $(CMD_PATH)

## install: Install the CLI binary
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(CMD_PATH)

## test: Run all tests with coverage
test:
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

## bench: Run benchmarks
bench: bench-gen
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./benchmarks/generated/

## bench-readme: Generate benchmark markdown for README
bench-readme: bench-gen
	@echo "Generating benchmark markdown..."
	@go test -bench=. -benchmem ./benchmarks/generated/ 2>&1 | go run ./scripts/format_benchmarks.go

## bench-analyze: Analyze benchmark results with comparison summary
bench-analyze: bench-gen
	@echo "Running benchmarks with analysis..."
	@go test -bench=. -benchmem ./benchmarks/generated/ 2>&1 | go run ./scripts/analyze_benchmarks.go

## bench-gen: Generate benchmark code
bench-gen:
	@echo "Generating benchmark code..."
	@rm -rf ./benchmarks/generated
	@mkdir -p ./benchmarks/generated
	@go run ./scripts/generate_benchmarks.go
	@gofmt -s -w ./benchmarks/generated

## bench-chart: Generate performance comparison chart
bench-chart: bench-gen
	@echo "Generating performance chart..."
	@go test -bench=. -benchmem ./benchmarks/generated/ 2>&1 | python3 scripts/benchmark_chart.py

## coverage: Generate and open coverage report
coverage: test
	@echo "Generating coverage report..."
	@go tool cover -html=coverage.txt -o coverage.html
	@open coverage.html || xdg-open coverage.html

## fmt: Format all Go files
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

## lint: Run linter
lint:
	@echo "Running linter..."
	@PATH="$$PATH:$$(go env GOPATH)/bin" golangci-lint run ./...

## clean: Clean build artifacts and generated files
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf benchmarks/generated/
	@rm -rf output/
	@rm -f coverage.txt coverage.html
	@go clean

## tidy: Tidy and verify dependencies
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy
	@go mod verify

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download

## update-deps: Update all dependencies
update-deps:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

## ci: Run CI pipeline (fmt, lint, test)
ci: fmt lint test
	@echo "CI pipeline completed successfully!"

## setup-hooks: Install git hooks and dependencies
setup-hooks:
	@echo "Installing dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing git hooks..."
	@chmod +x .githooks/pre-commit
	@git config core.hooksPath .githooks
	@echo "Setup completed successfully!"

## version: Display Go version
version:
	@go version
