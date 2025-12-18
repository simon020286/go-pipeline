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

// @step name=delay category=flow description=Pauses pipeline execution for a specified duration
type DelayConfig struct {
	Ms int `step:"name=ms,required,desc=Delay duration in milliseconds"`
}

type DelayStep struct {
	delay config.ValueSpec
}

func (s *DelayStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *DelayStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			delayResolved, err := s.delay.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve delay: %w", err)
				return
			}

			// Converti a int
			var delayMS int
			switch v := delayResolved.(type) {
			case int:
				delayMS = v
			case float64:
				delayMS = int(v)
			case int64:
				delayMS = int(v)
			default:
				errorChan <- fmt.Errorf("delay must be a number, got %T", delayResolved)
				return
			}

			// Simula elaborazione
			select {
			case <-time.After(time.Duration(delayMS) * time.Millisecond):
				result := "delay completed successfully"

				outputChan <- models.StepOutput{
					Data:      models.CreateDefaultResultData(result),
					EventID:   input.EventID,
					Timestamp: time.Now(),
				}
			case <-ctx.Done():
				errorChan <- errors.New("step cancelled")
				return
			}
		}
	}()

	return outputChan, errorChan
}

func init() {
	builder.RegisterStepType("delay", func(cfg map[string]any) (models.Step, error) {
		ms, ok := cfg["ms"]
		if !ok {
			return nil, errors.New("missing 'ms' in delay step")
		}
		rawInputs, _ := cfg["inputs"].([]any)
		inputs := make([]string, len(rawInputs))
		for i, v := range rawInputs {
			inputs[i], ok = v.(string)
			if !ok {
				return nil, errors.New("invalid input name")
			}
		}
		// Converti in ValueSpec
		var delaySpec config.ValueSpec
		if vs, ok := ms.(config.ValueSpec); ok {
			delaySpec = vs
		} else {
			// Wrappa come StaticValue
			delaySpec = config.NewStaticValue(ms)
		}

		return &DelayStep{delay: delaySpec}, nil
	})
}
