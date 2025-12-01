package steps

import (
	"context"
	"errors"
	"fmt"
	"github.com/simon020286/go-pipeline/config"
	"os"
	"time"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/models"
)

type FileStep struct {
	path config.ValueSpec
}

func (s *FileStep) IsContinuous() bool {
	return false // Step batch, esegue e termina
}

func (s *FileStep) Run(ctx context.Context, inputs <-chan *models.StepInput) (<-chan models.StepOutput, <-chan error) {
	outputChan := make(chan models.StepOutput, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errorChan)

		// Processa TUTTI gli input in arrivo
		for input := range inputs {
			pathResolved, err := s.path.Resolve(input)
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve path: %w", err)
				return
			}
			filePath := fmt.Sprintf("%v", pathResolved)

			// Legge il contenuto del file
			content, err := os.ReadFile(filePath)
			if err != nil {
				errorChan <- err
				return
			}

			// Send the result as string
			select {
			case outputChan <- models.StepOutput{
				Data:      models.CreateDefaultResultData(string(content)),
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
	builder.RegisterStepType("file", func(cfg map[string]any) (models.Step, error) {
		path, ok := cfg["path"]
		if !ok {
			return nil, errors.New("missing 'path' in file step")
		}
		// Converti in ValueSpec
		var pathSpec config.ValueSpec
		if vs, ok := path.(config.ValueSpec); ok {
			pathSpec = vs
		} else {
			pathSpec = config.StaticValue{Value: path}
		}

		return &FileStep{path: pathSpec}, nil
	})
}
