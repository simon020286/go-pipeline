# Go Pipeline Orchestration System

A flexible and powerful pipeline orchestration framework for Go that lets you build complex workflows with ease. Chain operations together, run tasks in parallel, handle streaming data, and integrate with APIs using simple YAML configuration or programmatic API.

Perfect for data processing workflows, ETL pipelines, microservice orchestration, webhook handlers, and scheduled batch jobs.

## ✨ Key Features

- **📋 YAML Configuration** - Define complex pipelines in simple YAML files
- **🔄 Dual Execution Modes** - Run once (batch) or continuously (streaming with webhooks/cron)
- **⚡ Smart Execution** - Stages run only when their dependencies are ready, maximizing efficiency
- **🔀 Conditional Branching** - Create if/else flows with `step_id:true` and `step_id:false` dependency syntax
- **🔌 Dynamic Services** - Built-in support for APIs with template-based URL construction
- **📊 Event System** - Monitor pipeline execution in real-time with custom event listeners
- **🧩 Extensible** - Easy to add custom step types for your specific needs
- **🔒 Thread-Safe** - Built with Go's concurrency primitives for safe parallel execution
- **💻 Fluent API** - Clean, type-safe programmatic pipeline construction

## 🚀 Installation

```bash
go get github.com/simon020286/go-pipeline
```

## Quick Start

### CLI Usage

Create a YAML file `my-pipeline.yaml`:

```yaml
name: "data-processor"
description: "Fetch and process data"

stages:
  - id: "fetch"
    step_type: "http_client"
    step_config:
      url: "https://jsonplaceholder.typicode.com/posts/1"
      method: "GET"
    dependencies: []

  - id: "process"
    step_type: "js"
    step_config:
      code: |
        return {
          title: ctx.fetch.Body.title,
          processed: true
        };
    dependencies:
      - "fetch"
```

Run it:
```bash
go run examples/test_runner/main.go my-pipeline.yaml
```

### Library Usage

```go
package main

import (
    "context"
    pipeline "github.com/simon020286/go-pipeline"
    "github.com/simon020286/go-pipeline/builder"
)

func main() {
    // Create pipeline
    p := pipeline.NewPipeline()

    // Create steps
    httpStep, _ := builder.CreateStep("http_client", map[string]any{
        "url":    "https://api.github.com/users/golang",
        "method": "GET",
    })

    jsStep, _ := builder.CreateStep("js", map[string]any{
        "code": "return { name: ctx.fetch.Body.name, repos: ctx.fetch.Body.public_repos };",
    })

    // Create stages and define dependencies
    fetchStage := pipeline.NewStage("fetch", httpStep)
    processStage := pipeline.NewStage("process", jsStep)

    p.AddStage(fetchStage)
    p.AddStage(processStage).After(fetchStage)

    // Execute
    ctx := context.Background()
    p.Start(ctx)
    p.Wait()
}
```

### Loading from YAML

```go
import (
    "os"
    pipeline "github.com/simon020286/go-pipeline"
    "github.com/simon020286/go-pipeline/config"
    "gopkg.in/yaml.v3"
)

data, _ := os.ReadFile("my-pipeline.yaml")
var cfg config.PipelineConfig
yaml.Unmarshal(data, &cfg)

p, _ := pipeline.BuildFromConfig(&cfg)
p.Start(context.Background())
p.Wait()
```

## 🔧 Available Steps

### HTTP Client (`http_client`)

Make HTTP requests to REST APIs.

**Configuration:**
```yaml
step_type: "http_client"
step_config:
  url: "https://api.example.com/endpoint"  # Required
  method: "GET"                             # Required: GET, POST, PUT, DELETE, PATCH
  headers:                                  # Optional: custom headers
    Authorization: "Bearer token"
    Content-Type: "application/json"
  body: '{"key": "value"}'                 # Optional: request body (for POST/PUT/PATCH)
  response: "json"                         # Optional: json (default), text, raw
```

**Dynamic values with JavaScript:**
```yaml
url: "$js: 'https://api.example.com/users/' + ctx.previous_step.Body.user_id"
body: "$js: JSON.stringify({ name: ctx.user.name })"
```

**Output format:**
```go
{
  "StatusCode": 200,
  "Headers": {...},
  "Body": {...}  // Parsed JSON or string based on response type
}
```

### JavaScript Execution (`js`)

Execute JavaScript code to transform data or apply business logic.

**Configuration:**
```yaml
step_type: "js"
step_config:
  code: |
    // Access previous stage outputs via ctx
    const data = ctx.previous_stage.Body;

    // Transform data
    return {
      processed: true,
      result: data.map(item => item.value * 2)
    };
```

**Available context:**
- `ctx.stage_id` - Access output from any previous stage
- `ctx.stage_id.Body` - Access response body (for HTTP steps)
- `ctx._execution.id` - Current execution ID

**JavaScript features:**
- ES6+ syntax support
- JSON manipulation
- Array methods (map, filter, reduce)
- String operations
- Math operations
- Date handling

### JSON Parser (`json`)

Parse JSON strings into structured data.

**Configuration:**
```yaml
step_type: "json"
step_config:
  data: '{"key": "value"}'  # JSON string to parse
```

**With dynamic values:**
```yaml
data: "$js: ctx.previous_step.Body.json_string"
```

### File Reader (`file`)

Read file contents from the filesystem.

**Configuration:**
```yaml
step_type: "file"
step_config:
  path: "/path/to/file.txt"  # Absolute or relative path
```

**With dynamic paths:**
```yaml
path: "$js: './data/' + ctx.config.filename"
```

**Output:** File contents as string

### ForEach Iterator (`foreach`)

Iterate over a collection and process each item.

**Configuration:**
```yaml
step_type: "foreach"
step_config:
  list: "$js: ctx.fetch_users.Body"  # Array to iterate
  step_type: "js"                     # Step type for each item
  step_config:
    code: |
      return {
        id: ctx.item.id,
        processed: true
      };
```

**Output format:**
```go
{
  "items": [...],  // Array of processed results
  "count": 10      // Number of items processed
}
```

### Map Transform (`map`)

Transform multiple fields using dynamic expressions.

**Configuration:**
```yaml
step_type: "map"
step_config:
  fields:
    name: "$js: ctx.user.Body.full_name"
    age: "$js: new Date().getFullYear() - ctx.user.Body.birth_year"
    email: "$js: ctx.user.Body.email.toLowerCase()"
```

**Output:** Object with transformed fields

### Delay (`delay`)

Wait for a specified duration before continuing.

**Configuration:**
```yaml
step_type: "delay"
step_config:
  ms: 1000  # Milliseconds to wait
```

**Use cases:** Rate limiting, throttling, waiting for external systems

### Webhook Listener (`webhook`)

Listen for incoming HTTP webhooks (streaming mode).

**Configuration:**
```yaml
step_type: "webhook"
step_config:
  path: "/webhook"           # HTTP path to listen on
  method: "POST"             # HTTP method
  continuous: true           # true = continuous, false = one-shot
```

**Usage:**
```bash
# Start pipeline with webhook
go run examples/test_runner/main.go webhook-pipeline.yaml

# Trigger webhook
curl -X POST http://localhost:8080/webhook -d '{"data":"value"}'
```

**Output:** Webhook payload as received

### Cron Scheduler (`cron`)

Schedule recurring pipeline executions (streaming mode).

**Configuration:**
```yaml
step_type: "cron"
step_config:
  schedule: "*/5 * * * *"  # Cron expression (every 5 minutes)
```

**Cron format:**
```
* * * * *
│ │ │ │ │
│ │ │ │ └─── Day of week (0-6, Sunday=0)
│ │ │ └───── Month (1-12)
│ │ └─────── Day of month (1-31)
│ └───────── Hour (0-23)
└─────────── Minute (0-59)
```

**Examples:**
- `* * * * *` - Every minute
- `*/5 * * * *` - Every 5 minutes
- `0 * * * *` - Every hour
- `0 0 * * *` - Every day at midnight
- `0 9 * * 1` - Every Monday at 9:00 AM

### Conditional Branching (`if`)

Execute different branches based on a boolean condition.

**Configuration:**
```yaml
step_type: "if"
step_config:
  condition: true  # Static boolean or dynamic expression
```

**With dynamic condition:**
```yaml
step_type: "if"
step_config:
  condition: "$js: ctx.previous_step.Body.status === 'active'"
```

**Output:** The step produces an output with key `"true"` or `"false"` based on the condition result.

**Branch dependencies:** Use the `step_id:branch` syntax in dependencies to create conditional flows:

```yaml
stages:
  - id: "check_status"
    step_type: "if"
    step_config:
      condition: "$js: ctx.user.Body.is_premium"
    dependencies:
      - "user"

  # Executed ONLY when condition is TRUE
  - id: "premium_flow"
    step_type: "js"
    step_config:
      code: "return { message: 'Welcome premium user!' };"
    dependencies:
      - "check_status:true"  # Note the :true suffix

  # Executed ONLY when condition is FALSE
  - id: "free_flow"
    step_type: "js"
    step_config:
      code: "return { message: 'Upgrade to premium!' };"
    dependencies:
      - "check_status:false"  # Note the :false suffix

  # Executed regardless of condition (receives any output)
  - id: "always_run"
    step_type: "delay"
    step_config:
      ms: 100
    dependencies:
      - "check_status"  # No suffix = accepts any output
```

**Dependency syntax:**
- `"step_id"` - Receives all outputs (backward compatible)
- `"step_id:true"` - Receives output only when condition is true
- `"step_id:false"` - Receives output only when condition is false
- `"step_id:custom_branch"` - For custom branch names (extensible)

### Dynamic Service Steps

Pre-configured API integrations with template support.

**HackerNews API:**
```yaml
# Get top stories
step_type: "hackernews.get_top_stories"
step_config:
  operation: "get_top_stories"

# Get specific item
step_type: "hackernews.get_item"
step_config:
  operation: "get_item"
  item_id: "$js: ctx.top_stories.Body[0]"
```

**JSONPlaceholder API:**
```yaml
# List posts
step_type: "jsonplaceholder.list_posts"
step_config:
  operation: "list_posts"

# Get specific post
step_type: "jsonplaceholder.get_post"
step_config:
  operation: "get_post"
  post_id: "1"
```

**Add custom services:** Create YAML definitions in `builder/services/` directory.

## 📝 YAML Configuration

### Basic Structure

```yaml
name: "pipeline-name"
description: "Pipeline description"

stages:
  - id: "unique-stage-id"
    step_type: "step-type"
    step_config:
      # Step-specific configuration
    dependencies:
      - "previous-stage-id"
```

### Dynamic Values

Use `$js:` prefix to create dynamic expressions:

```yaml
# Access previous stage output
field: "$js: ctx.stage_id.Body.field"

# String concatenation
url: "$js: 'https://api.example.com/users/' + ctx.user_id"

# Array operations
items: "$js: ctx.data.Body.map(item => item.id)"

# Conditional logic
status: "$js: ctx.response.StatusCode === 200 ? 'success' : 'failed'"

# JSON serialization
body: "$js: JSON.stringify({ data: ctx.values })"
```

### Complete Example

```yaml
name: "user-processor"
description: "Fetch users and send notifications"

stages:
  # Fetch user data
  - id: "fetch_users"
    step_type: "http_client"
    step_config:
      url: "https://jsonplaceholder.typicode.com/users"
      method: "GET"
    dependencies: []

  # Process each user
  - id: "process"
    step_type: "foreach"
    step_config:
      list: "$js: ctx.fetch_users.Body"
      step_type: "js"
      step_config:
        code: |
          return {
            id: ctx.item.id,
            email: ctx.item.email,
            domain: ctx.item.email.split('@')[1]
          };
    dependencies:
      - "fetch_users"

  # Send notification
  - id: "notify"
    step_type: "http_client"
    step_config:
      url: "https://api.slack.com/webhook"
      method: "POST"
      body: "$js: JSON.stringify({ text: 'Processed ' + ctx.process.count + ' users' })"
    dependencies:
      - "process"
```

## 📊 Event System

Monitor pipeline execution with custom event listeners:

```go
import "github.com/simon020286/go-pipeline/models"

// Add event listener
pipeline.AddListener(models.EventListenerFunc(func(event models.Event) {
    switch event.Type {
    case models.EventPipelineStarted:
        log.Println("Pipeline started")
    case models.EventStageOutput:
        stageID := event.Data["stage_id"].(string)
        log.Printf("Stage '%s' completed", stageID)
    case models.EventStageError:
        stageID := event.Data["stage_id"].(string)
        err := event.Data["error"].(string)
        log.Printf("Stage '%s' failed: %s", stageID, err)
    }
}))
```

**Available events:**
- `pipeline.started` - Pipeline execution started
- `pipeline.completed` - Pipeline execution completed
- `pipeline.error` - Pipeline error occurred
- `stage.started` - Stage execution started
- `stage.output` - Stage produced output
- `stage.error` - Stage error occurred

**Use cases:**
- Custom logging (console, files, database)
- Metrics collection (Prometheus, StatsD)
- Real-time dashboards (WebSocket)
- Alerting systems (Slack, email, PagerDuty)
- Audit trails

## 🔨 Creating Custom Steps

```go
package mysteps

import (
    "context"
    "time"
    "github.com/simon020286/go-pipeline/builder"
    "github.com/simon020286/go-pipeline/models"
)

type MyStep struct {
    config string
}

func (s *MyStep) IsContinuous() bool {
    return false // true for streaming steps
}

func (s *MyStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
    outputChan := make(chan models.StepOutput, 1)
    errorChan := make(chan error, 1)

    go func() {
        defer close(outputChan)
        defer close(errorChan)

        for input := range inputs {
            // Process input
            result := processData(input)

            // Send output
            outputChan <- models.StepOutput{
                Data:      models.CreateDefaultResultData(result),
                EventID:   input.EventID,
                Timestamp: time.Now(),
            }
        }
    }()

    return outputChan, errorChan
}

// Register step type
func init() {
    builder.RegisterStepType("my_step", func(cfg map[string]any) (models.Step, error) {
        return &MyStep{config: cfg["config"].(string)}, nil
    })
}
```

## 📚 Examples

The `examples/` directory contains working examples:

- `hackernews_dynamic_pipeline.yaml` - HackerNews API integration
- `http_client_dynamic_pipeline.yaml` - HTTP client with dynamic values
- `foreach_pipeline.yaml` - Collection processing
- `cron_pipeline.yaml` - Scheduled execution
- `if_pipeline.yaml` - Conditional branching with true/false flows
- `examples/webhook/main.go` - Webhook handler

Run any example:
```bash
go run examples/test_runner/main.go examples/<example>.yaml
```

## 📖 Technical Documentation

For architecture details and implementation specifics, see [CLAUDE.md](CLAUDE.md).

## 📄 License

MIT License - see LICENSE file for details.

---

**Built with ❤️ using Go**
