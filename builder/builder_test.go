package builder

import (
	"testing"

	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

func TestParseConfigValue_StaticString(t *testing.T) {
	result := ParseConfigValue("hello")

	if !result.IsStatic() {
		t.Error("Expected static value")
	}

	val, ok := result.GetStaticValue()
	if !ok {
		t.Fatal("GetStaticValue should return true")
	}

	if val != "hello" {
		t.Errorf("Expected 'hello', got %v", val)
	}
}

func TestParseConfigValue_StaticNumber(t *testing.T) {
	result := ParseConfigValue(42)

	if !result.IsStatic() {
		t.Error("Expected static value")
	}

	val, ok := result.GetStaticValue()
	if !ok {
		t.Fatal("GetStaticValue should return true")
	}

	if val != 42 {
		t.Errorf("Expected 42, got %v", val)
	}
}

func TestParseConfigValue_DynamicJS(t *testing.T) {
	result := ParseConfigValue("$js: ctx.step1.value")

	if result.IsStatic() {
		t.Error("Expected dynamic value")
	}

	dv, ok := result.GetDynamicExpression()
	if !ok {
		t.Fatal("GetDynamicExpression should return true")
	}

	if dv.Language != "js" {
		t.Errorf("Expected language 'js', got %s", dv.Language)
	}

	if dv.Expression != "ctx.step1.value" {
		t.Errorf("Expected expression 'ctx.step1.value', got %s", dv.Expression)
	}
}

func TestParseConfigValue_DynamicJSWithSpaces(t *testing.T) {
	result := ParseConfigValue("$js:   ctx.step1 + 10  ")

	dv, ok := result.GetDynamicExpression()
	if !ok {
		t.Fatal("Expected dynamic value")
	}

	// Expression should be trimmed
	if dv.Expression != "ctx.step1 + 10" {
		t.Errorf("Expected trimmed expression, got '%s'", dv.Expression)
	}
}

func TestParseConfigValue_Bool(t *testing.T) {
	result := ParseConfigValue(true)

	if !result.IsStatic() {
		t.Error("Expected static value")
	}

	val, ok := result.GetStaticValue()
	if !ok {
		t.Fatal("GetStaticValue should return true")
	}

	if val != true {
		t.Errorf("Expected true, got %v", val)
	}
}

func TestConvertGoTemplateToJS_SimpleParam(t *testing.T) {
	template := "/api/{{.user_id}}"
	params := map[string]config.ValueSpec{
		"user_id": config.DynamicValue{
			Language:   "js",
			Expression: "ctx.step1.id",
		},
	}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/api/' + ctx.step1.id"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvertGoTemplateToJS_MultipleParams(t *testing.T) {
	template := "/users/{{.user_id}}/posts/{{.post_id}}"
	params := map[string]config.ValueSpec{
		"user_id": config.DynamicValue{
			Language:   "js",
			Expression: "ctx.user.id",
		},
		"post_id": config.DynamicValue{
			Language:   "js",
			Expression: "ctx.post.id",
		},
	}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/users/' + ctx.user.id + '/posts/' + ctx.post.id"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvertGoTemplateToJS_StaticParam(t *testing.T) {
	template := "/api/{{.version}}/users"
	params := map[string]config.ValueSpec{
		"version": config.NewStaticValue("v1"),
	}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/api/' + 'v1' + '/users'"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvertGoTemplateToJS_NoParams(t *testing.T) {
	template := "/api/static/path"
	params := map[string]config.ValueSpec{}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/api/static/path'"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvertGoTemplateToJS_MissingParam(t *testing.T) {
	template := "/api/{{.user_id}}"
	params := map[string]config.ValueSpec{
		// user_id is missing
	}

	_, err := convertGoTemplateToJS(template, params)
	if err == nil {
		t.Error("Expected error for missing parameter")
	}
}

func TestStepRegistry(t *testing.T) {
	// Test that we can register and retrieve a step type
	called := false
	testFactory := func(cfg map[string]any) (models.Step, error) {
		called = true
		return nil, nil
	}

	RegisterStepType("test_step", testFactory)

	factory, err := GetStepFactory("test_step")
	if err != nil {
		t.Fatalf("Step type not registered: %v", err)
	}

	factory(nil)
	if !called {
		t.Error("Factory function not called")
	}
}
