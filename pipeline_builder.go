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
