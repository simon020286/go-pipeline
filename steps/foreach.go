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

// @step name=foreach category=flow description=Iterates over a list and emits each item with its index
type ForeachConfig struct {
	List any `step:"required,desc=The list to iterate over"`
}

type ForeachStep struct {
	list config.ValueSpec
}

func (s *ForeachStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *ForeachStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			// Risolvi la lista da iterare
			listResolved, err := s.list.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve list: %w", err)
				return
			}

			// Converti a []any
			list, ok := listResolved.([]any)
			if !ok {
				errorChan <- fmt.Errorf("list must be an array, got %T", listResolved)
				return
			}

			// Crea il map di output che conterrà tutti i risultati
			outputData := make(map[string]*models.Data)

			// Raccogli tutti i risultati per l'output finale
			allResults := make([]any, 0, len(list))

			// Emetti un output per ogni elemento della lista
			for i, item := range list {
				iterationResult := map[string]any{
					"item":  item,
					"index": i,
				}

				// Aggiungi l'output dell'iterazione con chiave "iteration_N"
				outputData[fmt.Sprintf("iteration_%d", i)] = &models.Data{
					Value: iterationResult,
				}

				// Aggiungi ai risultati finali
				allResults = append(allResults, iterationResult)
			}

			// Aggiungi l'output finale aggregato con chiave "default"
			outputData["default"] = &models.Data{
				Value: map[string]any{
					"items": allResults,
					"count": len(list),
				},
			}

			// Send the result
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
	builder.RegisterStepType("foreach", func(cfg map[string]any) (models.Step, error) {
		list, ok := cfg["list"]
		if !ok {
			return nil, errors.New("missing 'list' in foreach step")
		}

		// Converti in ValueSpec
		var listSpec config.ValueSpec
		if vs, ok := list.(config.ValueSpec); ok {
			listSpec = vs
		} else {
			listSpec = config.NewStaticValue(list)
		}

		return &ForeachStep{
			list: listSpec,
		}, nil
	})
}
