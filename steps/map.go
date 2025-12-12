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

type MapStep struct {
	fields map[string]config.ValueSpec
}

func (s *MapStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *MapStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			// Risolvi tutti i campi
			resolvedFields := make(map[string]any)
			for key, iv := range s.fields {
				resolved, err := iv.Resolve(input)
				if err != nil {
					errorChan <- fmt.Errorf("failed to resolve field %s: %w", key, err)
					return
				}
				resolvedFields[key] = resolved
			}

			// Send the result
			select {
			case outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(resolvedFields),
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
	builder.RegisterStepType("map", func(cfg map[string]any) (models.Step, error) {
		fields, ok := cfg["fields"]
		if !ok {
			return nil, errors.New("missing 'fields' in map step")
		}

		fieldList, ok := fields.([]interface{})
		if !ok {
			return nil, fmt.Errorf("fields must be a list of maps, got %T", fields)
		}

		mapStepFields := make(map[string]config.ValueSpec)

		for _, field := range fieldList {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("each field must be a map, got %T", field)
			}

			name, ok := fieldMap["name"].(string)
			if !ok {
				return nil, fmt.Errorf("field map must contain a 'name' key with a string value, got %T", fieldMap["name"])
			}

			value, ok := fieldMap["value"]
			if !ok {
				return nil, fmt.Errorf("field map must contain a 'value' key, got %v", fieldMap)
			}

			// Converti in ValueSpec
			var fieldSpec config.ValueSpec
			if vs, ok := value.(config.ValueSpec); ok {
				fieldSpec = vs
			} else {
				fieldSpec = config.NewStaticValue(value)
			}

			mapStepFields[name] = fieldSpec
		}

		return &MapStep{fields: mapStepFields}, nil
	})
}
