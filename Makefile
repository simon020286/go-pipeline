.PHONY: test test-verbose test-coverage build lint clean help

# Default target
help:
	@echo "Available targets:"
	@echo "  test          - Run all tests"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  build         - Build all packages"
	@echo "  lint          - Run golangci-lint"
	@echo "  clean         - Clean build artifacts"

# Run all tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Build all packages
build:
	@echo "Building packages..."
	go build ./...
	@echo "Building examples..."
	go build ./examples/test_runner
	go build ./examples/webhook
	go build ./examples/webhook_oneshot

# Run linter (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install it from https://golangci-lint.run/welcome/install/" && exit 1)
	golangci-lint run --timeout=5m

# Clean build artifacts
clean:
	@echo "Cleaning..."
	go clean ./...
	rm -f coverage.txt coverage.html
	rm -f examples/test_runner/test_runner
	rm -f examples/webhook/webhook
	rm -f examples/webhook_oneshot/webhook_oneshot
