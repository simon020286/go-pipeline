package pipeline

import (
	"fmt"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/config"
)

// BuildFromConfig builds a pipeline from a configuration
func BuildFromConfig(cfg *config.PipelineConfig) (*Pipeline, error) {
	pipeline := NewPipeline()

	// Temporary map to resolve dependencies
	stageMap := make(map[string]*Stage)

	// Phase 1: Create all stages without dependencies
	for _, stageConfig := range cfg.Stages {
		// Create the step using the factory
		step, err := builder.CreateStep(stageConfig.StepType, stageConfig.StepConfig)
		if err != nil {
			return nil, err
		}

		// Create the stage (without dependencies)
		stage := NewStage(stageConfig.ID, step)
		stageMap[stageConfig.ID] = stage

		// Add the stage to the pipeline
		pipeline.AddStage(stage)
	}

	// Phase 2: Resolve dependencies from IDs to *Stage references
	for _, stageConfig := range cfg.Stages {
		// Support both Dependencies and Inputs (legacy)
		dependencies := stageConfig.Dependencies
		if len(dependencies) == 0 {
			dependencies = stageConfig.Inputs
		}

		if len(dependencies) == 0 {
			continue
		}

		stage := stageMap[stageConfig.ID]

		// Resolve dependency IDs to *Stage references
		var deps []*Stage
		for _, depID := range dependencies {
			depStage, exists := stageMap[depID]
			if !exists {
				return nil, fmt.Errorf("stage '%s' depends on non-existent stage '%s'", stageConfig.ID, depID)
			}
			deps = append(deps, depStage)
		}

		// Use the After API to set dependencies
		builder := &StageBuilder{pipeline: pipeline, stage: stage}
		if err := builder.After(deps...); err != nil {
			return nil, err
		}
	}

	return pipeline, nil
}
