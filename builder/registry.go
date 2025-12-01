package builder

import (
	"fmt"
	"sync"

	"github.com/simon020286/go-pipeline/models"
)

// StepFactory is a function that creates a Step from a configuration
type StepFactory func(config map[string]any) (models.Step, error)

var (
	// registry contains all registered factories by step type
	registry = make(map[string]StepFactory)
	mu       sync.RWMutex
)

// RegisterStepType registers a factory for a step type
// This function is called by init() in step packages
func RegisterStepType(stepType string, factory StepFactory) {
	mu.Lock()
	defer mu.Unlock()
	registry[stepType] = factory
}

// GetStepFactory returns the factory for a step type
func GetStepFactory(stepType string) (StepFactory, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, exists := registry[stepType]
	if !exists {
		return nil, fmt.Errorf("unknown step type: %s", stepType)
	}
	return factory, nil
}

// ListStepTypes returns all registered step types
func ListStepTypes() []string {
	mu.RLock()
	defer mu.RUnlock()

	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
