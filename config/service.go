package config

import (
	"fmt"
)

// ServiceDefinition represents the complete definition of an API service
type ServiceDefinition struct {
	Service    ServiceInfo              `yaml:"service"`
	Defaults   ServiceDefaults          `yaml:"defaults"`
	Operations map[string]OperationDef  `yaml:"operations"`
}

// ServiceInfo contains the basic service information
type ServiceInfo struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// ServiceDefaults contains default configurations for all operations
type ServiceDefaults struct {
	BaseURL string            `yaml:"base_url"`
	Headers map[string]string `yaml:"headers"`
	Auth    *AuthConfig       `yaml:"auth"`
	Timeout int               `yaml:"timeout"` // in seconds, default 30
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

// OperationDef defines a single API operation
type OperationDef struct {
	Description  string            `yaml:"description"`
	Method       string            `yaml:"method"`        // GET, POST, PUT, DELETE, PATCH
	Path         string            `yaml:"path"`          // path template (e.g., "/{{.index}}/_search")
	Headers      map[string]string `yaml:"headers"`       // additional headers specific to this operation
	Body         string            `yaml:"body"`          // body template
	ResponseType string            `yaml:"response_type"` // json, text, raw
	QueryParams  map[string]string `yaml:"query_params"`  // query parameters template
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
