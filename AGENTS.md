# AGENTS.md

## Build & Test Commands

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run single test with verbose output
go test -v ./path/to/package -run TestSpecificFunction

# Run tests with coverage
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Lint (requires golangci-lint)
golangci-lint run --timeout=5m

# Format code
go fmt ./...

# Run pipeline examples
go run examples/test_runner/main.go examples/PIPELINE_NAME.yaml
```

## Code Style Guidelines

### Imports
- Group imports: stdlib, third-party, project packages
- Use absolute imports for project packages: `github.com/simon020286/go-pipeline/...`

### Naming
- PascalCase for exported types, functions, constants
- camelCase for unexported
- Use descriptive names: `HTTPClientStep` not `HttpStep`

### Error Handling
- Always handle errors explicitly
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Use domain-specific errors from `models/errors.go`

### Types & Interfaces
- Implement `models.Step` interface for all steps
- Use `config.ValueSpec` for dynamic vs static values
- Export types with JSON tags for serialization

### Step Implementation Pattern
- All steps must implement `IsContinuous()` and `Run()`
- Use channel-based communication: `inputs <-chan`, `outputs <-chan`
- Close channels in defer statements
- Process all inputs in for-range loop

### Dynamic Services
- Use Go template syntax in service definitions: `{{.param}}`
- Use `$js:` prefix for JavaScript expressions in pipeline configs
- Register new steps via `builder.RegisterStepType()` in init()