package config

// PipelineConfig represents the complete pipeline configuration from YAML
type PipelineConfig struct {
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	Variables   map[string]interface{} `yaml:"variables,omitempty"` // Global reusable variables
	Secrets     map[string]interface{} `yaml:"secrets,omitempty"`   // Sensitive values (API keys, tokens)
	Stages      []StageConfig          `yaml:"stages"`
}

// StageConfig represents the configuration of a stage from YAML
type StageConfig struct {
	ID           string                 `yaml:"id"`
	StepType     string                 `yaml:"step_type"`    // Type of step to instantiate
	StepConfig   map[string]interface{} `yaml:"step_config"`  // Specific step configuration
	Dependencies []string               `yaml:"dependencies"` // IDs of stages this depends on

	// Legacy support
	Inputs []string `yaml:"inputs,omitempty"` // Deprecated: use Dependencies
}
