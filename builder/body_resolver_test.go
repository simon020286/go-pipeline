package builder

import (
	"testing"

	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

func TestBodyResolver_MergeParams_GlobalDefaults(t *testing.T) {
	serviceDef := &config.ServiceDefinition{
		GlobalParams: map[string]config.ParameterDef{
			"page_size": {
				Default: 100,
				Type:    "int",
			},
		},
	}

	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{},
	}

	resolver := NewBodyResolver(serviceDef, opDef)
	userParams := make(map[string]config.ValueSpec)

	merged, err := resolver.mergeParams(userParams)
	if err != nil {
		t.Fatalf("mergeParams failed: %v", err)
	}

	if len(merged) != 1 {
		t.Errorf("Expected 1 merged param, got %d", len(merged))
	}

	pageSize, exists := merged["page_size"]
	if !exists {
		t.Fatal("Expected page_size in merged params")
	}

	val, isStatic := pageSize.GetStaticValue()
	if !isStatic {
		t.Error("Expected page_size to be static")
	}
	if val != 100 {
		t.Errorf("Expected page_size = 100, got %v", val)
	}
}

func TestBodyResolver_MergeParams_OperationOverridesGlobal(t *testing.T) {
	serviceDef := &config.ServiceDefinition{
		GlobalParams: map[string]config.ParameterDef{
			"page_size": {
				Default: 100,
			},
		},
	}

	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"page_size": {
				Default: 50, // Override global
			},
		},
	}

	resolver := NewBodyResolver(serviceDef, opDef)
	userParams := make(map[string]config.ValueSpec)

	merged, err := resolver.mergeParams(userParams)
	if err != nil {
		t.Fatalf("mergeParams failed: %v", err)
	}

	pageSize := merged["page_size"]
	val, _ := pageSize.GetStaticValue()
	if val != 50 {
		t.Errorf("Expected operation default to override global, got %v instead of 50", val)
	}
}

func TestBodyResolver_MergeParams_UserOverridesAll(t *testing.T) {
	serviceDef := &config.ServiceDefinition{
		GlobalParams: map[string]config.ParameterDef{
			"page_size": {Default: 100},
		},
	}

	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"page_size": {Default: 50},
		},
	}

	resolver := NewBodyResolver(serviceDef, opDef)
	userParams := map[string]config.ValueSpec{
		"page_size": config.StaticValue{Value: 25}, // User value
	}

	merged, err := resolver.mergeParams(userParams)
	if err != nil {
		t.Fatalf("mergeParams failed: %v", err)
	}

	pageSize := merged["page_size"]
	val, _ := pageSize.GetStaticValue()
	if val != 25 {
		t.Errorf("Expected user param to override all, got %v instead of 25", val)
	}
}

func TestBodyResolver_ValidateParams_RequiredPresent(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"database_id": {
				Required: true,
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	params := map[string]config.ValueSpec{
		"database_id": config.StaticValue{Value: "abc123"},
	}

	err := resolver.validateParams(params)
	if err != nil {
		t.Errorf("Expected validation to pass with required param present, got error: %v", err)
	}
}

func TestBodyResolver_ValidateParams_RequiredMissing(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"database_id": {
				Required: true,
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	params := map[string]config.ValueSpec{} // Missing required param

	err := resolver.validateParams(params)
	if err == nil {
		t.Error("Expected validation error for missing required param")
	}
}

func TestBodyResolver_ResolveBodyField_StaticValue(t *testing.T) {
	resolver := NewBodyResolver(&config.ServiceDefinition{}, &config.OperationDef{})
	params := make(map[string]config.ValueSpec)

	// Test string
	result, err := resolver.resolveBodyField("static string", params)
	if err != nil {
		t.Fatalf("resolveBodyField failed: %v", err)
	}
	val, isStatic := result.GetStaticValue()
	if !isStatic || val != "static string" {
		t.Errorf("Expected static string value, got %v (static=%v)", val, isStatic)
	}

	// Test int
	result, err = resolver.resolveBodyField(42, params)
	if err != nil {
		t.Fatalf("resolveBodyField failed: %v", err)
	}
	val, isStatic = result.GetStaticValue()
	if !isStatic || val != 42 {
		t.Errorf("Expected static int value 42, got %v", val)
	}

	// Test bool
	result, err = resolver.resolveBodyField(true, params)
	if err != nil {
		t.Fatalf("resolveBodyField failed: %v", err)
	}
	val, isStatic = result.GetStaticValue()
	if !isStatic || val != true {
		t.Errorf("Expected static bool value true, got %v", val)
	}
}

func TestBodyResolver_ResolveBodyMap_ParamReference(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"filter": {Required: true},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	params := map[string]config.ValueSpec{
		"filter": config.StaticValue{Value: map[string]any{"status": "active"}},
	}

	bodyMap := map[string]any{
		"$param": "filter",
	}

	result, err := resolver.resolveBodyMap(bodyMap, params)
	if err != nil {
		t.Fatalf("resolveBodyMap failed: %v", err)
	}

	val, isStatic := result.GetStaticValue()
	if !isStatic {
		t.Error("Expected param reference to resolve to static value")
	}

	filterMap, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("Expected map, got %T", val)
	}
	if filterMap["status"] != "active" {
		t.Errorf("Expected status=active, got %v", filterMap["status"])
	}
}

func TestBodyResolver_ResolveBodyMap_OptionalParamMissing(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"filter": {Optional: true},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	params := map[string]config.ValueSpec{} // filter not provided

	bodyMap := map[string]any{
		"$param": "filter",
	}

	result, err := resolver.resolveBodyMap(bodyMap, params)
	if err != nil {
		t.Fatalf("resolveBodyMap failed: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil for missing optional param, got %v", result)
	}
}

func TestBodyResolver_ResolveBodyMap_NestedObject(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"filter":    {Required: true},
			"page_size": {Default: 100},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	params := map[string]config.ValueSpec{
		"filter":    config.StaticValue{Value: map[string]any{"status": "active"}},
		"page_size": config.StaticValue{Value: 50},
	}

	bodyMap := map[string]any{
		"query": map[string]any{
			"filter": map[string]any{
				"$param": "filter",
			},
		},
		"pagination": map[string]any{
			"page_size": map[string]any{
				"$param": "page_size",
			},
		},
	}

	result, err := resolver.resolveBodyMap(bodyMap, params)
	if err != nil {
		t.Fatalf("resolveBodyMap failed: %v", err)
	}

	val, isStatic := result.GetStaticValue()
	if !isStatic {
		t.Error("Expected nested static object")
	}

	resultMap := val.(map[string]any)
	queryMap := resultMap["query"].(map[string]any)
	filterMap := queryMap["filter"].(map[string]any)
	if filterMap["status"] != "active" {
		t.Errorf("Expected nested filter to be resolved")
	}
}

func TestBodyResolver_ResolveBody_CompleteFlow(t *testing.T) {
	serviceDef := &config.ServiceDefinition{
		GlobalParams: map[string]config.ParameterDef{
			"page_size": {Default: 100},
		},
	}

	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"filter":    {Optional: true},
			"sorts":     {Optional: true},
			"page_size": {Optional: true},
		},
		Body: map[string]any{
			"filter": map[string]any{
				"$param": "filter",
			},
			"sorts": map[string]any{
				"$param": "sorts",
			},
			"page_size": map[string]any{
				"$param": "page_size",
			},
		},
	}

	resolver := NewBodyResolver(serviceDef, opDef)
	userParams := map[string]config.ValueSpec{
		"filter": config.StaticValue{Value: map[string]any{"status": "done"}},
		// sorts not provided (should be omitted)
		// page_size uses default from global
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, isStatic := bodySpec.GetStaticValue()
	if !isStatic {
		t.Error("Expected body to be static")
	}

	bodyMap := val.(map[string]any)

	// Check filter is present
	if _, exists := bodyMap["filter"]; !exists {
		t.Error("Expected filter in body")
	}

	// Check sorts is omitted (optional and not provided)
	if _, exists := bodyMap["sorts"]; exists {
		t.Error("Expected sorts to be omitted (optional param not provided)")
	}

	// Check page_size uses default
	if bodyMap["page_size"] != 100 {
		t.Errorf("Expected page_size=100 (global default), got %v", bodyMap["page_size"])
	}
}

func TestStructuredBody_Resolve_Object(t *testing.T) {
	sb := &StructuredBody{
		Fields: map[string]config.ValueSpec{
			"static_field":  config.StaticValue{Value: "hello"},
			"dynamic_field": config.DynamicValue{Expression: "ctx.value", Language: "js"},
		},
	}

	// Create a mock StepInput
	state := &models.StepInput{
		Data: map[string]map[string]*models.Data{
			"value": {
				"default": &models.Data{Value: "world"},
			},
		},
	}

	result, err := sb.Resolve(state)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["static_field"] != "hello" {
		t.Errorf("Expected static_field=hello, got %v", resultMap["static_field"])
	}

	if resultMap["dynamic_field"] != "world" {
		t.Errorf("Expected dynamic_field=world (from ctx.value), got %v", resultMap["dynamic_field"])
	}
}

func TestStructuredBody_Resolve_Array(t *testing.T) {
	sb := &StructuredBody{
		Array: []config.ValueSpec{
			config.StaticValue{Value: "item1"},
			config.StaticValue{Value: "item2"},
		},
	}

	state := &models.StepInput{
		Data: make(map[string]map[string]*models.Data),
	}

	result, err := sb.Resolve(state)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	resultArray, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected array result, got %T", result)
	}

	if len(resultArray) != 2 {
		t.Errorf("Expected 2 items, got %d", len(resultArray))
	}

	if resultArray[0] != "item1" || resultArray[1] != "item2" {
		t.Errorf("Expected [item1, item2], got %v", resultArray)
	}
}

// ============================================================================
// Conditional Tests
// ============================================================================

func TestBodyResolver_Conditional_ParamExists_True(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"database_id": {Optional: true},
			"page_id":     {Optional: true},
		},
		Body: map[string]any{
			"$if": map[string]any{
				"$param":  "database_id",
				"$exists": true,
			},
			"$then": map[string]any{
				"database_id": map[string]any{"$param": "database_id"},
			},
			"$else": map[string]any{
				"page_id": map[string]any{"$param": "page_id"},
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"database_id": config.StaticValue{Value: "db-123"},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, isStatic := bodySpec.GetStaticValue()
	if !isStatic {
		t.Error("Expected static result")
	}

	bodyMap := val.(map[string]any)
	if bodyMap["database_id"] != "db-123" {
		t.Errorf("Expected database_id=db-123, got %v", bodyMap["database_id"])
	}
	if _, exists := bodyMap["page_id"]; exists {
		t.Error("Expected page_id to be absent (else branch not taken)")
	}
}

func TestBodyResolver_Conditional_ParamExists_False(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"database_id": {Optional: true},
			"page_id":     {Optional: true},
		},
		Body: map[string]any{
			"$if": map[string]any{
				"$param":  "database_id",
				"$exists": true,
			},
			"$then": map[string]any{
				"database_id": map[string]any{"$param": "database_id"},
			},
			"$else": map[string]any{
				"page_id": map[string]any{"$param": "page_id"},
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"page_id": config.StaticValue{Value: "page-456"},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, isStatic := bodySpec.GetStaticValue()
	if !isStatic {
		t.Error("Expected static result")
	}

	bodyMap := val.(map[string]any)
	if bodyMap["page_id"] != "page-456" {
		t.Errorf("Expected page_id=page-456, got %v", bodyMap["page_id"])
	}
	if _, exists := bodyMap["database_id"]; exists {
		t.Error("Expected database_id to be absent (then branch not taken)")
	}
}

func TestBodyResolver_Conditional_ParamEquals(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"parent_type": {Required: true},
			"parent_id":   {Required: true},
		},
		Body: map[string]any{
			"parent": map[string]any{
				"$if": map[string]any{
					"$param":  "parent_type",
					"$equals": "database",
				},
				"$then": map[string]any{
					"type":        "database_id",
					"database_id": map[string]any{"$param": "parent_id"},
				},
				"$else": map[string]any{
					"type":    "page_id",
					"page_id": map[string]any{"$param": "parent_id"},
				},
			},
		},
	}

	// Test with parent_type = "database"
	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"parent_type": config.StaticValue{Value: "database"},
		"parent_id":   config.StaticValue{Value: "db-123"},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	bodyMap := val.(map[string]any)
	parentMap := bodyMap["parent"].(map[string]any)

	if parentMap["type"] != "database_id" {
		t.Errorf("Expected type=database_id, got %v", parentMap["type"])
	}
	if parentMap["database_id"] != "db-123" {
		t.Errorf("Expected database_id=db-123, got %v", parentMap["database_id"])
	}
}

func TestBodyResolver_Conditional_ParamEquals_ElseBranch(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"parent_type": {Required: true},
			"parent_id":   {Required: true},
		},
		Body: map[string]any{
			"parent": map[string]any{
				"$if": map[string]any{
					"$param":  "parent_type",
					"$equals": "database",
				},
				"$then": map[string]any{
					"type":        "database_id",
					"database_id": map[string]any{"$param": "parent_id"},
				},
				"$else": map[string]any{
					"type":    "page_id",
					"page_id": map[string]any{"$param": "parent_id"},
				},
			},
		},
	}

	// Test with parent_type = "page" (else branch)
	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"parent_type": config.StaticValue{Value: "page"},
		"parent_id":   config.StaticValue{Value: "page-456"},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	bodyMap := val.(map[string]any)
	parentMap := bodyMap["parent"].(map[string]any)

	if parentMap["type"] != "page_id" {
		t.Errorf("Expected type=page_id, got %v", parentMap["type"])
	}
	if parentMap["page_id"] != "page-456" {
		t.Errorf("Expected page_id=page-456, got %v", parentMap["page_id"])
	}
}

func TestBodyResolver_Conditional_NestedInObject(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"include_meta": {Optional: true},
			"meta_value":   {Optional: true},
		},
		Body: map[string]any{
			"data": "some data",
			"meta": map[string]any{
				"$if": map[string]any{
					"$param":  "include_meta",
					"$exists": true,
				},
				"$then": map[string]any{
					"value": map[string]any{"$param": "meta_value"},
				},
			},
		},
	}

	// Test without include_meta (should omit meta field)
	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	bodyMap := val.(map[string]any)

	if bodyMap["data"] != "some data" {
		t.Errorf("Expected data='some data', got %v", bodyMap["data"])
	}
	if _, exists := bodyMap["meta"]; exists {
		t.Error("Expected meta to be omitted when condition is false")
	}
}

func TestBodyResolver_Conditional_NotEmpty(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"tags": {Optional: true},
		},
		Body: map[string]any{
			"$if": map[string]any{
				"$param":     "tags",
				"$not_empty": true,
			},
			"$then": map[string]any{
				"tags": map[string]any{"$param": "tags"},
			},
			"$else": map[string]any{
				"tags": []any{},
			},
		},
	}

	// Test with non-empty tags
	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"tags": config.StaticValue{Value: []any{"tag1", "tag2"}},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	bodyMap := val.(map[string]any)
	tags := bodyMap["tags"].([]any)

	if len(tags) != 2 || tags[0] != "tag1" {
		t.Errorf("Expected tags=[tag1, tag2], got %v", tags)
	}
}

// ============================================================================
// Array Template Tests
// ============================================================================

func TestBodyResolver_ArrayTemplate_ForEach(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"items": {Required: true},
		},
		Body: map[string]any{
			"$for_each": "items",
			"$template": map[string]any{
				"value": map[string]any{"$param": "$item"},
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"items": config.StaticValue{Value: []any{"a", "b", "c"}},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, isStatic := bodySpec.GetStaticValue()
	if !isStatic {
		t.Error("Expected static result")
	}

	resultArray := val.([]any)
	if len(resultArray) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(resultArray))
	}

	for i, item := range resultArray {
		itemMap := item.(map[string]any)
		expected := []string{"a", "b", "c"}[i]
		if itemMap["value"] != expected {
			t.Errorf("Item %d: expected value=%s, got %v", i, expected, itemMap["value"])
		}
	}
}

func TestBodyResolver_ArrayTemplate_NestedInBody(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"names": {Required: true},
		},
		Body: map[string]any{
			"type": "list",
			"items": map[string]any{
				"$for_each": "names",
				"$template": map[string]any{
					"name": map[string]any{"$param": "$item"},
					"type": "user",
				},
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"names": config.StaticValue{Value: []any{"Alice", "Bob"}},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	bodyMap := val.(map[string]any)

	if bodyMap["type"] != "list" {
		t.Errorf("Expected type=list, got %v", bodyMap["type"])
	}

	items := bodyMap["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	alice := items[0].(map[string]any)
	if alice["name"] != "Alice" || alice["type"] != "user" {
		t.Errorf("Expected {name: Alice, type: user}, got %v", alice)
	}

	bob := items[1].(map[string]any)
	if bob["name"] != "Bob" || bob["type"] != "user" {
		t.Errorf("Expected {name: Bob, type: user}, got %v", bob)
	}
}

func TestBodyResolver_ArrayTemplate_EmptyArray(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"items": {Required: true},
		},
		Body: map[string]any{
			"$for_each": "items",
			"$template": map[string]any{
				"value": map[string]any{"$param": "$item"},
			},
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"items": config.StaticValue{Value: []any{}}, // Empty array
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	resultArray := val.([]any)
	if len(resultArray) != 0 {
		t.Errorf("Expected empty array, got %v", resultArray)
	}
}

func TestBodyResolver_ArrayTemplate_WithStaticTemplate(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"ids": {Required: true},
		},
		Body: map[string]any{
			"$for_each": "ids",
			"$template": "static_value", // Template is just a static string
		},
	}

	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"ids": config.StaticValue{Value: []any{1, 2, 3}},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	resultArray := val.([]any)

	for _, item := range resultArray {
		if item != "static_value" {
			t.Errorf("Expected static_value for each item, got %v", item)
		}
	}
}

// ============================================================================
// Combined Conditional + Array Template Tests
// ============================================================================

func TestBodyResolver_ConditionalWithArrayTemplate(t *testing.T) {
	opDef := &config.OperationDef{
		Params: map[string]config.ParameterDef{
			"use_list": {Optional: true},
			"items":    {Optional: true},
		},
		Body: map[string]any{
			"$if": map[string]any{
				"$param":  "use_list",
				"$exists": true,
			},
			"$then": map[string]any{
				"$for_each": "items",
				"$template": map[string]any{
					"item": map[string]any{"$param": "$item"},
				},
			},
			"$else": map[string]any{
				"single": "value",
			},
		},
	}

	// Test with use_list = true
	resolver := NewBodyResolver(&config.ServiceDefinition{}, opDef)
	userParams := map[string]config.ValueSpec{
		"use_list": config.StaticValue{Value: true},
		"items":    config.StaticValue{Value: []any{"x", "y"}},
	}

	bodySpec, err := resolver.ResolveBody(userParams)
	if err != nil {
		t.Fatalf("ResolveBody failed: %v", err)
	}

	val, _ := bodySpec.GetStaticValue()
	resultArray := val.([]any)
	if len(resultArray) != 2 {
		t.Fatalf("Expected 2 items when use_list is provided, got %d", len(resultArray))
	}
}

// ============================================================================
// valueExists Tests
// ============================================================================

func TestBodyResolver_ValueExists(t *testing.T) {
	resolver := NewBodyResolver(&config.ServiceDefinition{}, &config.OperationDef{})

	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"nil", nil, false},
		{"empty string", "", false},
		{"non-empty string", "hello", true},
		{"empty array", []any{}, false},
		{"non-empty array", []any{1, 2}, true},
		{"empty map", map[string]any{}, false},
		{"non-empty map", map[string]any{"key": "val"}, true},
		{"zero int", 0, true},
		{"non-zero int", 42, true},
		{"false bool", false, true},
		{"true bool", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.valueExists(tt.value)
			if result != tt.expected {
				t.Errorf("valueExists(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}
