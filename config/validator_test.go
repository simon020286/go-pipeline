package config

import (
	"strings"
	"testing"
)

func TestValidateServiceDefinition_ValidService(t *testing.T) {
	def := &ServiceDefinition{
		Service: ServiceInfo{
			Name:        "test-service",
			Description: "Test service",
			Version:     "1.0",
		},
		Defaults: ServiceDefaults{
			BaseURL:     "https://api.example.com",
			ContentType: "application/json",
		},
		Operations: map[string]OperationDef{
			"test_op": {
				Description: "Test operation",
				Method:      "POST",
				Path:        "/test",
				Params: map[string]ParameterDef{
					"param1": {
						Required:    true,
						Type:        "string",
						Description: "Test param",
					},
					"param2": {
						Optional:    true,
						Default:     "default_value",
						Type:        "string",
						Description: "Optional param with default",
					},
				},
				Body: map[string]any{
					"field1": map[string]any{
						"$param": "param1",
					},
					"field2": map[string]any{
						"$param": "param2",
					},
				},
			},
		},
	}

	err := ValidateServiceDefinition(def)
	if err != nil {
		t.Errorf("Expected valid service to pass validation, got error: %v", err)
	}
}

func TestValidateServiceDefinition_MissingServiceName(t *testing.T) {
	def := &ServiceDefinition{
		Service: ServiceInfo{
			Description: "Test service",
		},
		Operations: map[string]OperationDef{
			"test_op": {
				Method: "GET",
				Path:   "/test",
			},
		},
	}

	err := ValidateServiceDefinition(def)
	if err == nil {
		t.Error("Expected error for missing service name")
	}
	if !strings.Contains(err.Error(), "service name is required") {
		t.Errorf("Expected error about missing service name, got: %v", err)
	}
}

func TestValidateOperation_RequiredParamWithDefault(t *testing.T) {
	def := &ServiceDefinition{
		Service: ServiceInfo{
			Name: "test-service",
		},
		Operations: map[string]OperationDef{
			"test_op": {
				Method: "POST",
				Path:   "/test",
				Params: map[string]ParameterDef{
					"bad_param": {
						Required:    true,
						Default:     "invalid", // Required param should not have default
						Type:        "string",
						Description: "This should fail validation",
					},
				},
			},
		},
	}

	err := ValidateServiceDefinition(def)
	if err == nil {
		t.Error("Expected error for required param with default value")
	}
	if !strings.Contains(err.Error(), "required but has a default value") {
		t.Errorf("Expected error about required param with default, got: %v", err)
	}
}

func TestValidateBodyReferences_ValidReferences(t *testing.T) {
	params := map[string]ParameterDef{
		"param1": {Required: true, Type: "string"},
		"param2": {Optional: true, Type: "object"},
	}

	body := map[string]any{
		"field1": map[string]any{
			"$param": "param1",
		},
		"nested": map[string]any{
			"field2": map[string]any{
				"$param": "param2",
			},
		},
	}

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected valid body references to pass, got error: %v", err)
	}
}

func TestValidateBodyReferences_UndefinedParam(t *testing.T) {
	params := map[string]ParameterDef{
		"param1": {Required: true, Type: "string"},
	}

	body := map[string]any{
		"field1": map[string]any{
			"$param": "undefined_param", // This param is not defined
		},
	}

	err := validateBodyReferences(body, params)
	if err == nil {
		t.Error("Expected error for undefined param reference")
	}
	if !strings.Contains(err.Error(), "undefined_param") {
		t.Errorf("Expected error to mention 'undefined_param', got: %v", err)
	}
}

func TestValidateBodyReferences_NestedStructure(t *testing.T) {
	params := map[string]ParameterDef{
		"filter": {Optional: true, Type: "object"},
		"sorts":  {Optional: true, Type: "array"},
	}

	body := map[string]any{
		"query": map[string]any{
			"filter": map[string]any{
				"$param": "filter",
			},
			"sort": map[string]any{
				"$param": "sorts",
			},
		},
	}

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected nested body structure to be valid, got error: %v", err)
	}
}

func TestValidateBodyReferences_ArrayBody(t *testing.T) {
	params := map[string]ParameterDef{
		"items": {Required: true, Type: "array"},
	}

	body := []any{
		map[string]any{
			"$param": "items",
		},
	}

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected array body to be valid, got error: %v", err)
	}
}

func TestValidateBodyReferences_StringBody(t *testing.T) {
	params := map[string]ParameterDef{
		"data": {Required: true, Type: "string"},
	}

	body := "simple string body"

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected string body to be valid (backward compatibility), got error: %v", err)
	}
}

func TestParameterDef_IsRequired(t *testing.T) {
	tests := []struct {
		name     string
		param    ParameterDef
		expected bool
	}{
		{
			name:     "Required true, Optional false",
			param:    ParameterDef{Required: true, Optional: false},
			expected: true,
		},
		{
			name:     "Required false, Optional true",
			param:    ParameterDef{Required: false, Optional: true},
			expected: false,
		},
		{
			name:     "Both true (Optional takes precedence)",
			param:    ParameterDef{Required: true, Optional: true},
			expected: false,
		},
		{
			name:     "Both false (defaults to not required)",
			param:    ParameterDef{Required: false, Optional: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.param.IsRequired()
			if result != tt.expected {
				t.Errorf("IsRequired() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParameterDef_IsOptional(t *testing.T) {
	tests := []struct {
		name     string
		param    ParameterDef
		expected bool
	}{
		{
			name:     "Optional true",
			param:    ParameterDef{Optional: true},
			expected: true,
		},
		{
			name:     "Required false (defaults to optional)",
			param:    ParameterDef{Required: false},
			expected: true,
		},
		{
			name:     "Required true, Optional false",
			param:    ParameterDef{Required: true, Optional: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.param.IsOptional()
			if result != tt.expected {
				t.Errorf("IsOptional() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// ============================================================================
// Conditional Validation Tests
// ============================================================================

func TestValidateConditionalDef_ValidConditional(t *testing.T) {
	params := map[string]ParameterDef{
		"database_id": {Optional: true, Type: "string"},
		"page_id":     {Optional: true, Type: "string"},
	}

	cond := map[string]any{
		"$if": map[string]any{
			"$param":  "database_id",
			"$exists": true,
		},
		"$then": map[string]any{
			"type":        "database_id",
			"database_id": map[string]any{"$param": "database_id"},
		},
		"$else": map[string]any{
			"type":    "page_id",
			"page_id": map[string]any{"$param": "page_id"},
		},
	}

	err := validateConditionalDef(cond, params)
	if err != nil {
		t.Errorf("Expected valid conditional to pass, got error: %v", err)
	}
}

func TestValidateConditionalDef_UndefinedParamInCondition(t *testing.T) {
	params := map[string]ParameterDef{
		"page_id": {Optional: true, Type: "string"},
	}

	cond := map[string]any{
		"$if": map[string]any{
			"$param":  "undefined_param", // Not defined
			"$exists": true,
		},
		"$then": "something",
	}

	err := validateConditionalDef(cond, params)
	if err == nil {
		t.Error("Expected error for undefined param in condition")
	}
	if !strings.Contains(err.Error(), "undefined_param") {
		t.Errorf("Expected error to mention 'undefined_param', got: %v", err)
	}
}

func TestValidateConditionalDef_UndefinedParamInThenBranch(t *testing.T) {
	params := map[string]ParameterDef{
		"database_id": {Optional: true, Type: "string"},
	}

	cond := map[string]any{
		"$if": map[string]any{
			"$param":  "database_id",
			"$exists": true,
		},
		"$then": map[string]any{
			"value": map[string]any{"$param": "undefined_in_then"}, // Not defined
		},
	}

	err := validateConditionalDef(cond, params)
	if err == nil {
		t.Error("Expected error for undefined param in then branch")
	}
}

func TestValidateConditionDef_MissingOperator(t *testing.T) {
	params := map[string]ParameterDef{
		"test_param": {Optional: true, Type: "string"},
	}

	cond := map[string]any{
		"$param": "test_param",
		// Missing operator like $exists, $equals, etc.
	}

	err := validateConditionDef(cond, params)
	if err == nil {
		t.Error("Expected error for condition without operator")
	}
	if !strings.Contains(err.Error(), "must have at least one operator") {
		t.Errorf("Expected error about missing operator, got: %v", err)
	}
}

func TestValidateConditionDef_MultipleOperators(t *testing.T) {
	params := map[string]ParameterDef{
		"test_param": {Optional: true, Type: "string"},
	}

	cond := map[string]any{
		"$param":  "test_param",
		"$exists": true,
		"$equals": "value", // Multiple operators not allowed
	}

	err := validateConditionDef(cond, params)
	if err == nil {
		t.Error("Expected error for condition with multiple operators")
	}
	if !strings.Contains(err.Error(), "can only have one operator") {
		t.Errorf("Expected error about multiple operators, got: %v", err)
	}
}

func TestValidateConditionDef_ValidWithEquals(t *testing.T) {
	params := map[string]ParameterDef{
		"parent_type": {Required: true, Type: "string"},
	}

	cond := map[string]any{
		"$param":  "parent_type",
		"$equals": "database",
	}

	err := validateConditionDef(cond, params)
	if err != nil {
		t.Errorf("Expected valid condition with $equals to pass, got error: %v", err)
	}
}

func TestValidateConditionDef_ValidWithNotEmpty(t *testing.T) {
	params := map[string]ParameterDef{
		"tags": {Optional: true, Type: "array"},
	}

	cond := map[string]any{
		"$param":     "tags",
		"$not_empty": true,
	}

	err := validateConditionDef(cond, params)
	if err != nil {
		t.Errorf("Expected valid condition with $not_empty to pass, got error: %v", err)
	}
}

// ============================================================================
// Array Template Validation Tests
// ============================================================================

func TestValidateArrayTemplateDef_ValidTemplate(t *testing.T) {
	params := map[string]ParameterDef{
		"items": {Required: true, Type: "array"},
	}

	arrTmpl := map[string]any{
		"$for_each": "items",
		"$template": map[string]any{
			"value": "item_value",
		},
	}

	err := validateArrayTemplateDef(arrTmpl, params)
	if err != nil {
		t.Errorf("Expected valid array template to pass, got error: %v", err)
	}
}

func TestValidateArrayTemplateDef_MissingForEach(t *testing.T) {
	params := map[string]ParameterDef{}

	arrTmpl := map[string]any{
		"$template": map[string]any{
			"value": "item_value",
		},
	}

	err := validateArrayTemplateDef(arrTmpl, params)
	if err == nil {
		t.Error("Expected error for missing $for_each")
	}
	if !strings.Contains(err.Error(), "$for_each") {
		t.Errorf("Expected error about $for_each, got: %v", err)
	}
}

func TestValidateArrayTemplateDef_MissingTemplate(t *testing.T) {
	params := map[string]ParameterDef{
		"items": {Required: true, Type: "array"},
	}

	arrTmpl := map[string]any{
		"$for_each": "items",
		// Missing $template or $array_map
	}

	err := validateArrayTemplateDef(arrTmpl, params)
	if err == nil {
		t.Error("Expected error for missing $template")
	}
	if !strings.Contains(err.Error(), "$template") || !strings.Contains(err.Error(), "$array_map") {
		t.Errorf("Expected error about $template or $array_map, got: %v", err)
	}
}

func TestValidateArrayTemplateDef_BothTemplateAndArrayMap(t *testing.T) {
	params := map[string]ParameterDef{
		"items": {Required: true, Type: "array"},
	}

	arrTmpl := map[string]any{
		"$for_each":  "items",
		"$template":  "value",
		"$array_map": "other", // Both not allowed
	}

	err := validateArrayTemplateDef(arrTmpl, params)
	if err == nil {
		t.Error("Expected error for having both $template and $array_map")
	}
	if !strings.Contains(err.Error(), "cannot have both") {
		t.Errorf("Expected error about both not allowed, got: %v", err)
	}
}

// ============================================================================
// Body References with Conditionals
// ============================================================================

func TestValidateBodyReferences_WithConditional(t *testing.T) {
	params := map[string]ParameterDef{
		"database_id": {Optional: true, Type: "string"},
		"page_id":     {Optional: true, Type: "string"},
	}

	body := map[string]any{
		"parent": map[string]any{
			"$if": map[string]any{
				"$param":  "database_id",
				"$exists": true,
			},
			"$then": map[string]any{
				"type":        "database_id",
				"database_id": map[string]any{"$param": "database_id"},
			},
			"$else": map[string]any{
				"type":    "page_id",
				"page_id": map[string]any{"$param": "page_id"},
			},
		},
	}

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected body with conditional to be valid, got error: %v", err)
	}
}

func TestValidateBodyReferences_WithArrayTemplate(t *testing.T) {
	params := map[string]ParameterDef{
		"items": {Required: true, Type: "array"},
	}

	body := map[string]any{
		"data": map[string]any{
			"$for_each": "items",
			"$template": map[string]any{
				"name": "item_name",
			},
		},
	}

	err := validateBodyReferences(body, params)
	if err != nil {
		t.Errorf("Expected body with array template to be valid, got error: %v", err)
	}
}
