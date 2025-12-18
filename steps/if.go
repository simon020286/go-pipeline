package steps

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/simon020286/go-pipeline/config"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

// @step name=if category=flow description=Conditional branching step that evaluates a boolean condition
type IfConfig struct {
	Condition bool `step:"required,desc=Boolean condition to evaluate (use $js: for dynamic expressions)"`
}

type IfStep struct {
	condition config.ValueSpec
}

func (s *IfStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *IfStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			condResolved, err := s.condition.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve condition: %w", err)
				return
			}

			// Converti a bool
			condition, ok := condResolved.(bool)
			if !ok {
				errorChan <- fmt.Errorf("condition must be a boolean, got %T", condResolved)
				return
			}

			// Crea output con chiave "true" o "false" basato sulla condizione
			var outputData map[string]*models.Data
			if condition {
				outputData = models.CreateResultData("true", nil)
			} else {
				outputData = models.CreateResultData("false", nil)
			}

			select {
			case outputChan <- models.StepOutput{
				Data:      outputData,
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
	builder.RegisterStepType("if", func(cfg map[string]any) (models.Step, error) {
		condition, ok := cfg["condition"]
		if !ok {
			return nil, errors.New("missing 'condition' in if step")
		}

		// Converti in ValueSpec
		var condSpec config.ValueSpec
		if vs, ok := condition.(config.ValueSpec); ok {
			condSpec = vs
		} else {
			condSpec = config.NewStaticValue(condition)
		}

		return &IfStep{condition: condSpec}, nil
	})
}
