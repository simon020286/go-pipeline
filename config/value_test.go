package config

import (
	"testing"

	"github.com/simon020286/go-pipeline/models"
)

func TestStaticValue_IsStatic(t *testing.T) {
	sv := NewStaticValue("test")
	if !sv.IsStatic() {
		t.Error("StaticValue should return true for IsStatic()")
	}
}

func TestStaticValue_GetStaticValue(t *testing.T) {
	expected := "test value"
	sv := NewStaticValue(expected)

	val, ok := sv.GetStaticValue()
	if !ok {
		t.Error("GetStaticValue should return true for StaticValue")
	}
	if val != expected {
		t.Errorf("Expected %v, got %v", expected, val)
	}
}

func TestStaticValue_GetDynamicExpression(t *testing.T) {
	sv := NewStaticValue("test")

	_, ok := sv.GetDynamicExpression()
	if ok {
		t.Error("GetDynamicExpression should return false for StaticValue")
	}
}

func TestStaticValue_Resolve(t *testing.T) {
	expected := 42
	sv := NewStaticValue(expected)

	input := &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}

	result, err := sv.Resolve(input)
	if err != nil {
		t.Errorf("StaticValue.Resolve should not return error: %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

func TestDynamicValue_IsStatic(t *testing.T) {
	dv := DynamicValue{Language: "js", Expression: "1 + 1"}
	if dv.IsStatic() {
		t.Error("DynamicValue should return false for IsStatic()")
	}
}

func TestDynamicValue_GetDynamicExpression(t *testing.T) {
	dv := DynamicValue{Language: "js", Expression: "1 + 1"}

	expr, ok := dv.GetDynamicExpression()
	if !ok {
		t.Error("GetDynamicExpression should return true for DynamicValue")
	}
	if expr.Expression != "1 + 1" {
		t.Errorf("Expected expression '1 + 1', got '%s'", expr.Expression)
	}
}

func TestDynamicValue_GetStaticValue(t *testing.T) {
	dv := DynamicValue{Language: "js", Expression: "1 + 1"}

	_, ok := dv.GetStaticValue()
	if ok {
		t.Error("GetStaticValue should return false for DynamicValue")
	}
}

func TestDynamicValue_ResolveJS_SimpleExpression(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "2 + 2",
	}

	input := &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}

	result, err := dv.Resolve(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != int64(4) {
		t.Errorf("Expected 4, got %v", result)
	}
}

func TestDynamicValue_ResolveJS_WithContext(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "ctx.step1.value * 2",
	}

	input := &models.StepInput{
		Data: map[string]map[string]*models.Data{
			"step1": {
				"default": &models.Data{
					Value: map[string]any{"value": 10},
				},
			},
		},
		EventID: "test-event",
	}

	result, err := dv.Resolve(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != int64(20) {
		t.Errorf("Expected 20, got %v", result)
	}
}

func TestDynamicValue_ResolveJS_WithArrayAccess(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "ctx.step1[0]",
	}

	input := &models.StepInput{
		Data: map[string]map[string]*models.Data{
			"step1": {
				"default": &models.Data{
					Value: []any{100, 200, 300},
				},
			},
		},
		EventID: "test-event",
	}

	result, err := dv.Resolve(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != int64(100) {
		t.Errorf("Expected 100, got %v", result)
	}
}

func TestDynamicValue_ResolveJS_StringConcatenation(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "'Hello ' + ctx.step1",
	}

	input := &models.StepInput{
		Data: map[string]map[string]*models.Data{
			"step1": {
				"default": &models.Data{
					Value: "World",
				},
			},
		},
		EventID: "test-event",
	}

	result, err := dv.Resolve(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "Hello World" {
		t.Errorf("Expected 'Hello World', got %v", result)
	}
}

func TestDynamicValue_ResolveJS_WithExecutionID(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "ctx._execution.id",
	}

	input := &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "exec-123",
	}

	result, err := dv.Resolve(input)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "exec-123" {
		t.Errorf("Expected 'exec-123', got %v", result)
	}
}

func TestDynamicValue_ResolveJS_InvalidExpression(t *testing.T) {
	dv := DynamicValue{
		Language:   "js",
		Expression: "invalid syntax !!!",
	}

	input := &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}

	_, err := dv.Resolve(input)
	if err == nil {
		t.Error("Expected error for invalid expression")
	}
}

func TestDynamicValue_Resolve_UnsupportedLanguage(t *testing.T) {
	dv := DynamicValue{
		Language:   "python",
		Expression: "1 + 1",
	}

	input := &models.StepInput{
		Data:    make(map[string]map[string]*models.Data),
		EventID: "test-event",
	}

	_, err := dv.Resolve(input)
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestHasDynamicValues(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]ValueSpec
		expected bool
	}{
		{
			name: "all static",
			values: map[string]ValueSpec{
				"key1": NewStaticValue("val1"),
				"key2": NewStaticValue(42),
			},
			expected: false,
		},
		{
			name: "mixed static and dynamic",
			values: map[string]ValueSpec{
				"key1": NewStaticValue("val1"),
				"key2": DynamicValue{Language: "js", Expression: "1+1"},
			},
			expected: true,
		},
		{
			name: "all dynamic",
			values: map[string]ValueSpec{
				"key1": DynamicValue{Language: "js", Expression: "ctx.step1"},
				"key2": DynamicValue{Language: "js", Expression: "ctx.step2"},
			},
			expected: true,
		},
		{
			name:     "empty map",
			values:   map[string]ValueSpec{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasDynamicValues(tt.values)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractStaticValues(t *testing.T) {
	values := map[string]ValueSpec{
		"static1": NewStaticValue("hello"),
		"static2": NewStaticValue(42),
		"dynamic": DynamicValue{Language: "js", Expression: "ctx.step1"},
	}

	result := ExtractStaticValues(values)

	if len(result) != 2 {
		t.Errorf("Expected 2 static values, got %d", len(result))
	}

	if result["static1"] != "hello" {
		t.Errorf("Expected 'hello', got %v", result["static1"])
	}

	if result["static2"] != 42 {
		t.Errorf("Expected 42, got %v", result["static2"])
	}

	if _, exists := result["dynamic"]; exists {
		t.Error("Dynamic value should not be in extracted static values")
	}
}
