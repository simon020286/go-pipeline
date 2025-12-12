package config

import (
	"fmt"
	"os"

	"github.com/dop251/goja"
	"github.com/simon020286/go-pipeline/models"
)

// ValueSpec represents a value that can be static or dynamic
type ValueSpec interface {
	IsStatic() bool
	GetStaticValue() (any, bool)
	GetDynamicExpression() (DynamicValue, bool)
	// Resolve resolves the value using the pipeline context
	Resolve(state *models.StepInput) (any, error)
}

// StaticValue represents a literal value (number, string, bool, etc.)
type StaticValue struct {
	Value any
}

func NewStaticValue(value any) StaticValue {
	return StaticValue{
		Value: value,
	}
}

func (s StaticValue) IsStatic() bool {
	return true
}

func (s StaticValue) GetStaticValue() (any, bool) {
	return s.Value, true
}

func (s StaticValue) GetDynamicExpression() (DynamicValue, bool) {
	return DynamicValue{}, false
}

func (s StaticValue) Resolve(state *models.StepInput) (any, error) {
	// Static values always return themselves
	return s.Value, nil
}

// DynamicValue represents an expression to be evaluated at runtime
type DynamicValue struct {
	Language   string // "js", "python", "lua", etc.
	Expression string // the expression to evaluate
	Type       string // optional: "string", "number", "boolean", etc.
}

func (d DynamicValue) IsStatic() bool {
	return false
}

func (d DynamicValue) GetStaticValue() (any, bool) {
	return nil, false
}

func (d DynamicValue) GetDynamicExpression() (DynamicValue, bool) {
	return d, true
}

func (d DynamicValue) Resolve(state *models.StepInput) (any, error) {
	// Evaluate the expression using the appropriate runtime
	switch d.Language {
	case "js", "javascript", "":
		return d.resolveJS(state)
	// Future: other languages
	// case "py", "python":
	//     return d.resolvePython(state)
	// case "lua":
	//     return d.resolveLua(state)
	default:
		return nil, fmt.Errorf("unsupported language: %s", d.Language)
	}
}

// resolveJS evaluates a JavaScript expression using Goja
func (d DynamicValue) resolveJS(state *models.StepInput) (any, error) {
	runtime := goja.New()

	// Build the ctx context from pipeline state
	ctx := make(map[string]any)
	state.Lock()

	for stepName, outputs := range state.Data {
		// If there's only a "default" output, use the value directly
		if len(outputs) == 1 {
			if data, ok := outputs["default"]; ok {
				ctx[stepName] = data.Value
				continue
			}
		}

		// Otherwise create a map of all outputs
		stepCtx := make(map[string]any)
		for outName, data := range outputs {
			stepCtx[outName] = data.Value
		}
		ctx[stepName] = stepCtx
	}

	// Add execution metadata
	if state.EventID != "" {
		ctx["_execution"] = map[string]any{
			"id": state.EventID,
		}
	}

	state.Unlock()

	// Set the context in the JS runtime
	if err := runtime.Set("ctx", ctx); err != nil {
		return nil, fmt.Errorf("failed to set context: %w", err)
	}

	// Set global variables and secrets in the JS runtime
	if state.GlobalVariables != nil {
		if err := runtime.Set("$vars", state.GlobalVariables); err != nil {
			return nil, fmt.Errorf("failed to set global variables: %w", err)
		}
	}

	if state.GlobalSecrets != nil {
		if err := runtime.Set("$secrets", state.GlobalSecrets); err != nil {
			return nil, fmt.Errorf("failed to set global secrets: %w", err)
		}
	}

	wrappedCode := "(function() {\n return " + d.Expression + "\n})()"

	// Execute the expression
	result, err := runtime.RunString(wrappedCode)
	if err != nil {
		return nil, fmt.Errorf("failed to execute JS expression '%s': %w", d.Expression, err)
	}

	// Return the native Go value
	return result.Export(), nil
}

// HasDynamicValues checks if at least one value is dynamic
func HasDynamicValues(values map[string]ValueSpec) bool {
	for _, v := range values {
		if !v.IsStatic() {
			return true
		}
	}
	return false
}

// ExtractStaticValues extracts only static values into a map[string]any
// Useful for passing to standard Go templates
func ExtractStaticValues(values map[string]ValueSpec) map[string]any {
	result := make(map[string]any)
	for k, v := range values {
		if staticVal, ok := v.GetStaticValue(); ok {
			result[k] = staticVal
		}
	}
	return result
}

// VariableReference represents a reference to a global pipeline variable ($var:name)
type VariableReference struct {
	Name string
}

func (v VariableReference) IsStatic() bool {
	return false
}

func (v VariableReference) GetStaticValue() (any, bool) {
	return nil, false
}

func (v VariableReference) GetDynamicExpression() (DynamicValue, bool) {
	return DynamicValue{}, false
}

func (v VariableReference) Resolve(state *models.StepInput) (any, error) {
	if state.GlobalVariables == nil {
		return nil, fmt.Errorf("variable '%s' not found: no global variables defined", v.Name)
	}

	value, exists := state.GlobalVariables[v.Name]
	if !exists {
		return nil, fmt.Errorf("variable '%s' not found in global variables", v.Name)
	}

	return value, nil
}

// SecretReference represents a reference to a global pipeline secret ($secret:name)
type SecretReference struct {
	Name string
}

func (s SecretReference) IsStatic() bool {
	return false
}

func (s SecretReference) GetStaticValue() (any, bool) {
	return nil, false
}

func (s SecretReference) GetDynamicExpression() (DynamicValue, bool) {
	return DynamicValue{}, false
}

func (s SecretReference) Resolve(state *models.StepInput) (any, error) {
	if state.GlobalSecrets == nil {
		return nil, fmt.Errorf("secret '%s' not found: no global secrets defined", s.Name)
	}

	value, exists := state.GlobalSecrets[s.Name]
	if !exists {
		return nil, fmt.Errorf("secret '%s' not found in global secrets", s.Name)
	}

	return value, nil
}

// String returns a masked representation of the secret for logging
func (s SecretReference) String() string {
	return fmt.Sprintf("$secret:%s=***", s.Name)
}

// EnvReference represents a reference to an environment variable ($env:NAME)
type EnvReference struct {
	Name string
}

func (e EnvReference) IsStatic() bool {
	return false
}

func (e EnvReference) GetStaticValue() (any, bool) {
	return nil, false
}

func (e EnvReference) GetDynamicExpression() (DynamicValue, bool) {
	return DynamicValue{}, false
}

func (e EnvReference) Resolve(state *models.StepInput) (any, error) {
	value := os.Getenv(e.Name)
	if value == "" {
		return nil, fmt.Errorf("environment variable '%s' is not set or is empty", e.Name)
	}

	return value, nil
}
