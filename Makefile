.PHONY: all build test bench bench-gen clean fmt lint install help mass-gen mass-bench mass-delete mass-build mass-workflow setup-hooks

# Variables
BINARY_NAME=regengo
CMD_PATH=./cmd/regengo
MASS_GEN_BINARY=bin/mass_generator
MASS_GEN_SOURCE=./cmd/mass_generator
PKG_LIST=$$(go list ./... | grep -v /vendor/ | grep -v /benchmarks/generated)

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

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic $(PKG_LIST)

## bench: Run benchmarks
bench: bench-gen
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./benchmarks/generated/

## bench-readme: Generate benchmark markdown for README
bench-readme: bench-gen
	@echo "Generating benchmark markdown..."
	@go test -bench=. -benchmem ./benchmarks/generated/ 2>&1 | go run ./cmd/bench_to_readme

## bench-gen: Generate benchmark code
bench-gen:
	@echo "Generating benchmark code..."
	@rm -rf ./benchmarks/generated
	@mkdir -p ./benchmarks/generated
	@go run ./cmd/curated_generator
	@gofmt -s -w ./benchmarks/generated

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
	@rm -rf benchmarks/mass_generated/
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

## mass-build: Build the mass_generator binary
mass-build:
	@echo "Building mass_generator..."
	@mkdir -p bin
	@go build -o $(MASS_GEN_BINARY) $(MASS_GEN_SOURCE)

## mass-gen: Generate mass test files (compiles binary if needed)
mass-gen: 
	@if [ ! -f $(MASS_GEN_BINARY) ]; then \
		echo "mass_generator binary not found, building..."; \
		$(MAKE) mass-build; \
	fi
	@echo "Generating mass tests..."
	@$(MASS_GEN_BINARY) -command=generate

## mass-bench: Run mass benchmarks (compiles binary if needed)
mass-bench:
	@if [ ! -f $(MASS_GEN_BINARY) ]; then \
		echo "mass_generator binary not found, building..."; \
		$(MAKE) mass-build; \
	fi
	@echo "Running mass benchmarks..."
	@$(MASS_GEN_BINARY) -command=benchmark

## mass-delete: Delete generated mass tests (compiles binary if needed)
mass-delete:
	@if [ ! -f $(MASS_GEN_BINARY) ]; then \
		echo "mass_generator binary not found, building..."; \
		$(MAKE) mass-build; \
	fi
	@echo "Deleting mass tests..."
	@$(MASS_GEN_BINARY) -command=delete

## mass-workflow: Run complete mass test workflow (generate -> benchmark -> delete)
mass-workflow: mass-gen mass-bench mass-delete
	@echo "Mass test workflow completed!"
