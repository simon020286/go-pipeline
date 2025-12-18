package steps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/simon020286/go-pipeline/config"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

// @step name=json category=data description=Parses a JSON string into a structured object
type JsonConfig struct {
	Data string `step:"required,desc=JSON string to parse (supports variable interpolation)"`
}

type JsonStep struct {
	data config.ValueSpec
}

func (s *JsonStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *JsonStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			// Risolvi la stringa JSON (supporta interpolazione)
			dataResolved, err := s.data.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve data: %w", err)
				return
			}
			dataString := fmt.Sprintf("%v", dataResolved)

			// Parsifica la stringa JSON (supporta oggetti, array, e valori primitivi)
			var jsonData any
			if err := json.Unmarshal([]byte(dataString), &jsonData); err != nil {
				errorChan <- fmt.Errorf("failed to unmarshal JSON data: %w", err)
				return
			}

			// Send the result
			select {
			case outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(jsonData),
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
	builder.RegisterStepType("json", func(cfg map[string]any) (models.Step, error) {
		data, ok := cfg["data"]
		if !ok {
			return nil, errors.New("missing 'data' in json step")
		}

		// Converti in ValueSpec
		var dataSpec config.ValueSpec
		if vs, ok := data.(config.ValueSpec); ok {
			dataSpec = vs
		} else {
			dataSpec = config.NewStaticValue(data)
		}

		return &JsonStep{
			data: dataSpec,
		}, nil
	})
}
