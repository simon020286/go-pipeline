package pipeline

import (
	"fmt"

	"github.com/simon020286/go-pipeline/builder"
	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

// BuildFromConfig builds a pipeline from a configuration
func BuildFromConfig(cfg *config.PipelineConfig) (*Pipeline, error) {
	pipeline := NewPipeline()

	// Process global variables
	if cfg.Variables != nil {
		resolvedVars, err := resolveGlobalConfig(cfg.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve global variables: %w", err)
		}
		pipeline.SetGlobalVariables(resolvedVars)
	}

	// Process global secrets
	if cfg.Secrets != nil {
		resolvedSecrets, err := resolveGlobalConfig(cfg.Secrets)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve global secrets: %w", err)
		}
		pipeline.SetGlobalSecrets(resolvedSecrets)
	}

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
		stageBuilder := &StageBuilder{pipeline: pipeline, stage: stage}

		// Resolve dependency IDs to *Stage references
		// Support format: "stage_id" or "stage_id:branch" for conditional branching
		for _, depStr := range dependencies {
			// Parse the dependency string to extract stage ID and optional branch
			depRef := config.ParseDependency(depStr)

			depStage, exists := stageMap[depRef.StageID]
			if !exists {
				return nil, fmt.Errorf("stage '%s' depends on non-existent stage '%s'", stageConfig.ID, depRef.StageID)
			}

			// Use AfterWithBranch if there's a branch filter, otherwise use After
			if depRef.Branch != "" {
				if err := stageBuilder.AfterWithBranch(depStage, depRef.Branch); err != nil {
					return nil, err
				}
			} else {
				if err := stageBuilder.After(depStage); err != nil {
					return nil, err
				}
			}
		}
	}

	return pipeline, nil
}

// resolveGlobalConfig resolves global configuration values (variables or secrets)
// Supports $env: references to environment variables
func resolveGlobalConfig(configMap map[string]interface{}) (map[string]any, error) {
	resolved := make(map[string]any)

	// Create an empty StepInput for resolution (no pipeline context needed for globals)
	emptyInput := &models.StepInput{
		Data: make(map[string]map[string]*models.Data),
	}

	for key, value := range configMap {
		// Parse the value to ValueSpec (handles $env: and other prefixes)
		valueSpec := builder.ParseConfigValue(value)

		// Resolve the value
		resolvedValue, err := valueSpec.Resolve(emptyInput)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve '%s': %w", key, err)
		}

		resolved[key] = resolvedValue
	}

	return resolved, nil
}
