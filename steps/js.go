package steps

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

type JsStep struct {
	code string
}

func (s *JsStep) IsContinuous() bool {
	return false // Batch step, executes and terminates
}

func (s *JsStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Process ALL incoming inputs
		for input := range inputs {
			// Prepare JavaScript context
			runtime := goja.New()
			jsCtx := make(map[string]any)

			input.Lock()
			for stepName, outputs := range input.Data {
				// If there's only one "default" output, use the value directly
				if len(outputs) == 1 {
					if data, ok := outputs["default"]; ok {
						jsCtx[stepName] = data.Value
						continue
					}
				}

				// Otherwise create a map of all outputs
				stepCtx := make(map[string]any)
				for outName, data := range outputs {
					stepCtx[outName] = data.Value
				}
				jsCtx[stepName] = stepCtx
			}

			// Add execution metadata
			if input.EventID != "" {
				jsCtx["_execution"] = map[string]any{
					"id": input.EventID,
				}
			}
			input.Unlock()

			// Set the context in the JavaScript runtime
			if err := runtime.Set("ctx", jsCtx); err != nil {
				errorChan <- fmt.Errorf("failed to set context in JavaScript runtime: %w", err)
				return
			}

			// Wrap the code in an anonymous function to allow return usage
			// The user can write: return { key: "value" };
			// It gets transformed to: (function() { return { key: "value" }; })()
			wrappedCode := "(function() {\n" + s.code + "\n})()"

			// Execute the wrapped JavaScript code
			result, err := runtime.RunString(wrappedCode)
			if err != nil {
				errorChan <- fmt.Errorf("JavaScript execution error: %w", err)
				return
			}

			// Export the result
			output := result.Export()

			// Send the result
			select {
			case outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(output),
				EventID:   input.EventID,
				Timestamp: time.Now(),
			}:
			case <-ctx.Done():
				errorChan <- errors.New("step cancelled")
				return
			}
		}
	}()

	return outputChan, errorChan
}

func init() {
	builder.RegisterStepType("js", func(cfg map[string]any) (models.Step, error) {
		code, ok := cfg["code"].(string)
		if !ok {
			return nil, errors.New("missing 'code' in js step")
		}

		return &JsStep{
			code: code,
		}, nil
	})
}
