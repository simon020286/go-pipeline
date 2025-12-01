# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go-based pipeline orchestration system supporting both **batch** (one-shot) and **streaming** (continuous) execution modes with dynamic service integration.

## Common Commands

### Build & Test
```bash
# Build entire project
go build ./...

# Run a pipeline from YAML
go run examples/test_runner/main.go examples/PIPELINE_NAME.yaml

# Run with verbose output
go run examples/test_runner/main.go -v examples/PIPELINE_NAME.yaml

# Run webhook example
go run examples/webhook/main.go
# In another terminal:
curl http://localhost:8080/webhook
```

### Key Examples
- `examples/hackernews_dynamic_pipeline.yaml` - Dynamic service with interpolation
- `examples/http_client_dynamic_pipeline.yaml` - HTTP with $js: expressions
- `examples/foreach_pipeline.yaml` - Iteration over collections
- `examples/cron_pipeline.yaml` - Streaming mode with cron trigger

## Architecture

### Core Components

**Pipeline Execution Model** (output-based, not input-based):
- **Lazy execution**: Stages start only when dependencies complete
- **On-demand triggering**: Stages notified when predecessors produce output
- **Channel-based communication**: Async data flow between stages
- **Dual execution modes**: Batch (one-shot) vs Streaming (continuous)

**Package Structure**:
- **`builder/`** - Step factory and dynamic service registration
  - `builder.go`: Core factory, `RegisterDynamicAPIServices()`, Go template → JS conversion
  - `registry.go`: Step type registry
  - `loader.go`: YAML service definition loader
  - `services/*.yaml`: Service definitions (hackernews, jsonplaceholder, etc.)

- **`config/`** - Configuration and value handling
  - `value.go`: `ValueSpec` interface - distinguishes static vs dynamic values
  - `service.go`: Service definition structs (`ServiceDefinition`, `OperationDef`, `AuthConfig`)
  - `pipeline.go`: Pipeline configuration structs

- **`models/`** - Core interfaces
  - `step.go`: `Step` interface, `StepInput`, execution context
  - `step_output.go`: Output data structures
  - `errors.go`: Domain-specific errors

- **`steps/`** - Built-in step implementations
  - All steps implement `models.Step` interface
  - `IsContinuous()` distinguishes batch vs streaming steps
  - `Run(ctx, inputs <-chan) (outputs <-chan, errors <-chan)` - channel-based execution

- **Root package** - Pipeline orchestration
  - `pipeline.go`: Core `Pipeline` type, stage management, lifecycle
  - `pipeline_builder.go`: `BuildFromConfig()` - YAML → Pipeline construction
  - `event_bus.go`: Event system for monitoring

### Dynamic Service System

**How it works**:
1. Service definitions in `builder/services/*.yaml` use **Go template syntax** (`{{.param}}`)
2. Pipeline YAML config provides parameter values with **`$js:` prefix** for dynamic expressions
3. `builder.RegisterDynamicAPIServices()` converts Go templates → JavaScript expressions
4. `config.ValueSpec` handles static vs dynamic value resolution at runtime

**Example flow**:
```yaml
# Service definition (builder/services/hackernews.yaml)
path: "/item/{{.item_id}}.json"

# Pipeline config
item_id: "$js: ctx.get_best_stories.Body[0]"

# Conversion result
URL: '/item/' + ctx.get_best_stories.Body[0] + '.json'
```

**Key functions**:
- `parseConfigValue()`: Recognizes `$js:` prefix, creates `DynamicValue`
- `convertGoTemplateToJS()`: Parses Go template, substitutes with JS expressions
- `ValueSpec.Resolve()`: Evaluates dynamic expressions at runtime using Goja (JS engine)

### Value Resolution System

**`config.ValueSpec` interface**:
- `StaticValue`: Literal values (strings, numbers, bools) - resolved at build time
- `DynamicValue`: JavaScript expressions - resolved at runtime with pipeline context

**Context structure** (`ctx` in JS expressions):
- `ctx.step_id`: Access outputs from previous stages
- `ctx.step_id.Body[0]`: Nested access to response data
- `ctx._execution.id`: Execution metadata

### Step Implementation Pattern

All steps follow this pattern:
```go
type MyStep struct {
    // Use config.ValueSpec for dynamic resolution
    field config.ValueSpec
}

func (s *MyStep) IsContinuous() bool {
    return false // true for streaming steps (webhook, cron)
}

func (s *MyStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
    outputChan := make(chan models.StepOutput, 1)
    errorChan := make(chan error, 1)

    go func() {
        defer close(outputChan)
        defer close(errorChan)

        for input := range inputs {
            // Resolve dynamic values
            resolved, err := s.field.Resolve(input)

            // Execute logic
            result := doWork(resolved)

            // Send output
            outputChan <- models.StepOutput{
                Data: models.CreateDefaultResultData(result),
                EventID: input.EventID,
                Timestamp: time.Now(),
            }
        }
    }()

    return outputChan, errorChan
}
```

### Pipeline Builder Registration

**Adding new step types**:
```go
// In steps/my_step.go
func init() {
    builder.RegisterStepType("my_step", func(config map[string]any) (models.Step, error) {
        // Parse config, create step instance
        return &MyStep{...}, nil
    })
}
```

**Dynamic services** are registered via `builder.RegisterDynamicAPIServices(serviceRegistry)` which creates HTTP client steps with pre-configured URLs, headers, auth from service definitions.

## Important Patterns

### Go Template → JavaScript Conversion
When adding features to dynamic services:
- Service YAML uses Go templates: `"{{.param}}"`
- Pipeline YAML uses `$js:` prefix: `"$js: ctx.step.field"`
- Builder converts: `renderTemplate()` → `convertGoTemplateToJS()` → produces JS expression
- Runtime evaluates with Goja engine in `DynamicValue.Resolve()`

### Event System Usage
```go
// Add listener to pipeline
pipeline.AddListener(models.EventListenerFunc(func(event models.Event) {
    switch event.Type {
    case models.EventStageOutput:
        stageID := event.Data["stage_id"].(string)
        // Process output
    }
}))
```

### Stage Dependencies (Fluent API)
```go
stage1 := pipeline.NewStage("s1", step1)
stage2 := pipeline.NewStage("s2", step2)

pipeline.AddStage(stage1)
pipeline.AddStage(stage2).After(stage1) // stage2 depends on stage1
```

## Testing Dynamic Services

When testing changes to the dynamic service system:
1. Create/modify service definition in `builder/services/`
2. Create test pipeline YAML with `$js:` expressions
3. Run: `go run examples/test_runner/main.go -v path/to/test.yaml`
4. Check URL construction in error messages (shows resolved URLs)

## Critical Files for Common Tasks

**Adding new built-in step type**:
- Create in `steps/my_step.go`
- Implement `models.Step` interface
- Register in `init()` with `builder.RegisterStepType()`

**Modifying dynamic service behavior**:
- `builder/builder.go`: `RegisterDynamicAPIServices()`, template conversion
- `config/value.go`: Value resolution logic

**Changing pipeline execution**:
- `pipeline.go`: Core execution, stage triggering

**Adding new service definition**:
- Create YAML in `builder/services/`
- Embedded via `//go:embed` in `builder/init.go`
