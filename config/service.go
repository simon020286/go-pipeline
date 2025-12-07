package config

import (
	"fmt"
)

// ServiceDefinition represents the complete definition of an API service
type ServiceDefinition struct {
	Service      ServiceInfo             `yaml:"service"`
	Defaults     ServiceDefaults         `yaml:"defaults"`
	GlobalParams map[string]ParameterDef `yaml:"global_params"` // global parameters with defaults for all operations
	Operations   map[string]OperationDef `yaml:"operations"`
}

// ServiceInfo contains the basic service information
type ServiceInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// ServiceDefaults contains default configurations for all operations
type ServiceDefaults struct {
	BaseURL     string            `yaml:"base_url"`
	Headers     map[string]string `yaml:"headers"`
	Auth        *AuthConfig       `yaml:"auth"`
	Timeout     int               `yaml:"timeout"`      // in seconds, default 30
	ContentType string            `yaml:"content_type"` // default content type for requests (e.g., "application/json")
}

// AuthConfig configures authentication for the service
type AuthConfig struct {
	Type   string `yaml:"type"`   // bearer, basic, api_key, custom, none
	Header string `yaml:"header"` // header name (e.g., "Authorization")
	Value  string `yaml:"value"`  // value template (e.g., "Bearer {{.api_token}}")
	// For basic auth
	Username string `yaml:"username"` // username template
	Password string `yaml:"password"` // password template
}

// ParameterDef defines a parameter for an operation
type ParameterDef struct {
	Required    bool   `yaml:"$required"`    // if true, parameter must be provided by user
	Optional    bool   `yaml:"$optional"`    // if true, parameter is optional (equivalent to Required: false)
	Default     any    `yaml:"$default"`     // default value if not provided
	Type        string `yaml:"$type"`        // expected type: string, int, float, bool, object, array
	Description string `yaml:"$description"` // parameter documentation
}

// IsRequired returns true if the parameter is required
func (p ParameterDef) IsRequired() bool {
	return p.Required && !p.Optional
}

// IsOptional returns true if the parameter is optional
func (p ParameterDef) IsOptional() bool {
	return p.Optional || !p.Required
}

// ConditionalDef defines a conditional structure in body
type ConditionalDef struct {
	If   ConditionDef `yaml:"$if"`   // condition to evaluate
	Then any         `yaml:"$then"` // value if condition is true
	Else any         `yaml:"$else"` // value if condition is false
}

// ConditionDef defines a condition for conditional logic
type ConditionDef struct {
	Param     string `yaml:"$param"`     // parameter name to check
	Exists    bool   `yaml:"$exists"`    // check if parameter exists and is not null
	Equals    any    `yaml:"$equals"`    // check if parameter equals this value
	NotEquals any    `yaml:"$not_equals"` // check if parameter does not equal this value
	NotEmpty  bool   `yaml:"$not_empty"` // check if parameter is not empty (string, array, object)
	IsEmpty   bool   `yaml:"$is_empty"`  // check if parameter is empty (string, array, object)
}

// ArrayTemplateDef defines an array template structure
type ArrayTemplateDef struct {
	ForEach  string `yaml:"$for_each"`  // parameter name containing array
	Template any    `yaml:"$template"`  // template for each array item
	ArrayMap  *ArrayMapDef `yaml:"$array_map"` // alternative: map array items
}

// ArrayMapDef defines mapping for array items
type ArrayMapDef struct {
	Param string `yaml:"$param"`     // parameter name containing array
	Transform any `yaml:"$transform"` // transformation to apply to each item
}

// TransformDef defines value transformations
type TransformDef struct {
	Template string         `yaml:"$template"` // template string with {value} placeholder
	Function string         `yaml:"$function"` // built-in function name
	Args    map[string]any `yaml:"$args"`    // function arguments
}

// OperationDef defines a single API operation
type OperationDef struct {
	Description  string                  `yaml:"description"`
	Method       string                  `yaml:"method"`        // GET, POST, PUT, DELETE, PATCH
	Path         string                  `yaml:"path"`          // path template (e.g., "/{{.index}}/_search")
	Headers      map[string]string       `yaml:"headers"`       // additional headers specific to this operation
	Params       map[string]ParameterDef `yaml:"params"`        // operation parameters with type info and defaults
	Body         any                     `yaml:"body"`          // body structure (map, array, or string for backward compatibility)
	ContentType  string                  `yaml:"content_type"`  // content type for this operation (overrides service default)
	ResponseType string                  `yaml:"response_type"` // json, text, raw
	QueryParams  map[string]string       `yaml:"query_params"`  // query parameters template
}

// Validate valida la definizione del servizio
func (sd *ServiceDefinition) Validate() error {
	if sd.Service.Name == "" {
		return fmt.Errorf("service name is required")
	}

	if len(sd.Operations) == 0 {
		return fmt.Errorf("service %s must have at least one operation", sd.Service.Name)
	}

	for opName, op := range sd.Operations {
		if op.Method == "" {
			return fmt.Errorf("operation %s.%s: method is required", sd.Service.Name, opName)
		}
		if op.Path == "" {
			return fmt.Errorf("operation %s.%s: path is required", sd.Service.Name, opName)
		}
		// Validate method
		validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true, "HEAD": true, "OPTIONS": true}
		if !validMethods[op.Method] {
			return fmt.Errorf("operation %s.%s: invalid method %s", sd.Service.Name, opName, op.Method)
		}
	}

	return nil
}

// GetOperation returns an operation by name
func (sd *ServiceDefinition) GetOperation(name string) (*OperationDef, error) {
	op, exists := sd.Operations[name]
	if !exists {
		return nil, fmt.Errorf("operation '%s' not found in service '%s'", name, sd.Service.Name)
	}
	return &op, nil
}
