// stepgen parses Go source files looking for step config structs
// and generates JSON metadata and registry code.
//
// Usage: go run ./codegen/cmd/stepgen ./steps
package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// StepMetadata represents the complete metadata for a step type
type StepMetadata struct {
	Name        string      `json:"name"`
	Category    string      `json:"category"`
	Description string      `json:"description"`
	Inputs      []InputMeta `json:"inputs"`
}

// InputMeta represents an input parameter metadata
type InputMeta struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// StepsRegistry holds all discovered step metadata
type StepsRegistry struct {
	Steps   []StepMetadata `json:"steps"`
	Version string         `json:"version"`
}

// stepCommentRegex matches @step comments
// Format: @step name=xxx category=xxx description=xxx
var stepCommentRegex = regexp.MustCompile(`@step\s+(.+)`)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <directory>\n", os.Args[0])
		os.Exit(1)
	}

	dir := os.Args[1]
	steps, err := parseDirectory(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing directory: %v\n", err)
		os.Exit(1)
	}

	if len(steps) == 0 {
		fmt.Println("No step configs found")
		return
	}

	registry := StepsRegistry{
		Steps:   steps,
		Version: "1.0.0",
	}

	// // Generate JSON file
	// jsonPath := filepath.Join(dir, "steps_metadata.json")
	// if err := writeJSON(jsonPath, registry); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error writing JSON: %v\n", err)
	// 	os.Exit(1)
	// }
	// fmt.Printf("Generated %s\n", jsonPath)

	// Generate Go registry file
	goPath := filepath.Join(dir, "steps_registry_gen.go")
	if err := writeGoRegistry(goPath, registry); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing Go file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated %s\n", goPath)
}

func parseDirectory(dir string) ([]StepMetadata, error) {
	fset := token.NewFileSet()
	var steps []StepMetadata

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		// Skip generated files
		if strings.HasSuffix(entry.Name(), "_gen.go") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		fileSteps, err := parseFile(fset, filePath)
		if err != nil {
			return nil, fmt.Errorf("error parsing %s: %w", filePath, err)
		}
		steps = append(steps, fileSteps...)
	}

	return steps, nil
}

func parseFile(fset *token.FileSet, filePath string) ([]StepMetadata, error) {
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var steps []StepMetadata

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Only process structs ending with "Config"
			if !strings.HasSuffix(typeSpec.Name.Name, "Config") {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Look for @step comment in the doc comments
			stepMeta := parseStepComment(genDecl.Doc)
			if stepMeta == nil {
				continue
			}

			// Parse struct fields
			stepMeta.Inputs = parseStructFields(structType)
			steps = append(steps, *stepMeta)
		}
	}

	return steps, nil
}

func parseStepComment(doc *ast.CommentGroup) *StepMetadata {
	if doc == nil {
		return nil
	}

	for _, comment := range doc.List {
		text := strings.TrimPrefix(comment.Text, "//")
		text = strings.TrimSpace(text)

		match := stepCommentRegex.FindStringSubmatch(text)
		if match == nil {
			continue
		}

		// Parse key=value pairs from the comment
		meta := &StepMetadata{}
		params := match[1]

		// Simple parser for key=value pairs
		meta.Name = extractValue(params, "name")
		meta.Category = extractValue(params, "category")
		meta.Description = extractValue(params, "description")

		if meta.Name != "" {
			return meta
		}
	}

	return nil
}

func extractValue(params, key string) string {
	// Look for key=value pattern
	prefix := key + "="
	idx := strings.Index(params, prefix)
	if idx == -1 {
		return ""
	}

	// Find the start of the value
	start := idx + len(prefix)
	if start >= len(params) {
		return ""
	}

	// Find the end of the value (next space followed by key= or end of string)
	rest := params[start:]

	// Look for next key= pattern
	nextKeyPatterns := []string{" name=", " category=", " description="}
	end := len(rest)
	for _, pattern := range nextKeyPatterns {
		if idx := strings.Index(rest, pattern); idx != -1 && idx < end {
			end = idx
		}
	}

	return strings.TrimSpace(rest[:end])
}

func parseStructFields(structType *ast.StructType) []InputMeta {
	var inputs []InputMeta

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		fieldType := typeToString(field.Type)

		// Parse the step tag
		input := InputMeta{
			Name: toSnakeCase(fieldName),
			Type: fieldType,
		}

		if field.Tag != nil {
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			stepTag := tag.Get("step")
			if stepTag != "" {
				parseStepTag(stepTag, &input)
			}
		}

		inputs = append(inputs, input)
	}

	return inputs
}

func parseStepTag(tag string, input *InputMeta) {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)

		if part == "required" {
			input.Required = true
			continue
		}

		if strings.HasPrefix(part, "name=") {
			input.Name = strings.TrimPrefix(part, "name=")
			continue
		}

		if strings.HasPrefix(part, "default=") {
			input.Default = strings.TrimPrefix(part, "default=")
			continue
		}

		if strings.HasPrefix(part, "desc=") {
			input.Description = strings.TrimPrefix(part, "desc=")
			continue
		}
	}
}

func typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.InterfaceType:
		return "any"
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	default:
		return "any"
	}
}

func toSnakeCase(s string) string {
	// Handle common acronyms that should stay together
	acronyms := map[string]string{
		"URL":  "url",
		"HTTP": "http",
		"API":  "api",
		"ID":   "id",
		"JSON": "json",
		"XML":  "xml",
		"SQL":  "sql",
		"TCP":  "tcp",
		"UDP":  "udp",
		"IP":   "ip",
	}

	// Replace acronyms with placeholders
	result := s
	for acronym, replacement := range acronyms {
		result = strings.ReplaceAll(result, acronym, "_"+replacement+"_")
	}

	// Convert remaining camelCase to snake_case
	var sb strings.Builder
	for i, r := range result {
		if unicode.IsUpper(r) {
			if i > 0 {
				sb.WriteRune('_')
			}
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}

	// Clean up multiple underscores and leading/trailing underscores
	output := sb.String()
	for strings.Contains(output, "__") {
		output = strings.ReplaceAll(output, "__", "_")
	}
	output = strings.Trim(output, "_")

	return output
}

func writeJSON(path string, registry StepsRegistry) error {
	data, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func writeGoRegistry(path string, registry StepsRegistry) error {
	jsonData, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return err
	}

	code := fmt.Sprintf(`// Code generated by stepgen. DO NOT EDIT.
package steps

import (
	"encoding/json"
)

// StepMetadata represents the complete metadata for a step type
type StepMetadata struct {
	Name        string      %[1]sjson:"name"%[1]s
	Category    string      %[1]sjson:"category"%[1]s
	Description string      %[1]sjson:"description"%[1]s
	Inputs      []InputMeta %[1]sjson:"inputs"%[1]s
}

// InputMeta represents an input parameter metadata
type InputMeta struct {
	Name        string %[1]sjson:"name"%[1]s
	Type        string %[1]sjson:"type"%[1]s
	Required    bool   %[1]sjson:"required"%[1]s
	Default     string %[1]sjson:"default,omitempty"%[1]s
	Description string %[1]sjson:"description,omitempty"%[1]s
}

// stepsMetadataJSON contains the embedded JSON metadata
var stepsMetadataJSON = %[1]s%[2]s%[1]s

var stepsMetadata []StepMetadata

func init() {
	var registry struct {
		Steps []StepMetadata %[1]sjson:"steps"%[1]s
	}
	if err := json.Unmarshal([]byte(stepsMetadataJSON), &registry); err == nil {
		stepsMetadata = registry.Steps
	}
}

// GetStepsMetadata returns the metadata for all registered steps
func GetStepsMetadata() []StepMetadata {
	return stepsMetadata
}

// GetStepMetadata returns the metadata for a specific step by name
func GetStepMetadata(name string) (StepMetadata, bool) {
	for _, step := range stepsMetadata {
		if step.Name == name {
			return step, true
		}
	}
	return StepMetadata{}, false
}

// GetStepsMetadataJSON returns the raw JSON metadata
func GetStepsMetadataJSON() string {
	return stepsMetadataJSON
}

// GetStepsByCategory returns all steps in a given category
func GetStepsByCategory(category string) []StepMetadata {
	var result []StepMetadata
	for _, step := range stepsMetadata {
		if step.Category == category {
			result = append(result, step)
		}
	}
	return result
}

// GetCategories returns all unique categories
func GetCategories() []string {
	seen := make(map[string]bool)
	var categories []string
	for _, step := range stepsMetadata {
		if !seen[step.Category] {
			seen[step.Category] = true
			categories = append(categories, step.Category)
		}
	}
	return categories
}
`, "`", string(jsonData))

	return os.WriteFile(path, []byte(code), 0644)
}
