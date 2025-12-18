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

// DependencyRef represents a parsed dependency reference
// Format: "stage_id" or "stage_id:branch" where branch filters the output key
// Examples:
//   - "process" -> depends on all outputs from "process" stage
//   - "check:true" -> depends only on outputs where the key is "true"
//   - "check:false" -> depends only on outputs where the key is "false"
type DependencyRef struct {
	StageID string // The ID of the dependency stage
	Branch  string // Optional: filter outputs by this key (empty = accept all)
}

// ParseDependency parses a dependency string into a DependencyRef
// Supports format: "stage_id" or "stage_id:branch"
func ParseDependency(dep string) DependencyRef {
	// Find the last colon (to support stage IDs with colons, though not recommended)
	lastColon := -1
	for i := len(dep) - 1; i >= 0; i-- {
		if dep[i] == ':' {
			lastColon = i
			break
		}
	}

	if lastColon == -1 || lastColon == len(dep)-1 {
		// No colon or colon at the end: no branch filter
		return DependencyRef{StageID: dep, Branch: ""}
	}

	return DependencyRef{
		StageID: dep[:lastColon],
		Branch:  dep[lastColon+1:],
	}
}
