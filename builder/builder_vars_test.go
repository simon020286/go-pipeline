package builder

import (
	"testing"

	"github.com/simon020286/go-pipeline/config"
)

func TestConvertGoTemplateToJS_VariableReference(t *testing.T) {
	template := "/databases/{{.database_id}}/query"
	params := map[string]config.ValueSpec{
		"database_id": config.VariableReference{Name: "notion_db"},
	}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/databases/' + $vars.notion_db + '/query'"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvertGoTemplateToJS_SecretReference(t *testing.T) {
	template := "/api/{{.token}}/data"
	params := map[string]config.ValueSpec{
		"token": config.SecretReference{Name: "api_token"},
	}

	result, err := convertGoTemplateToJS(template, params)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := "'/api/' + $secrets.api_token + '/data'"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
