package builder

import (
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

// CreateStep creates a step based on type and configuration
func CreateStep(stepType string, stepConfig map[string]any) (models.Step, error) {
	factory, err := GetStepFactory(stepType)
	if err != nil {
		return nil, err
	}

	// Pre-process all configuration values to convert special prefixes
	// ($var:, $secret:, $env:, $js:) into appropriate ValueSpec types
	processedConfig := preprocessStepConfig(stepConfig)

	return factory(processedConfig)
}

// preprocessStepConfig recursively processes configuration values,
// converting strings with special prefixes into ValueSpec types
func preprocessStepConfig(config map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range config {
		result[key] = preprocessValue(value)
	}

	return result
}

// preprocessValue converts a single value, handling nested structures
func preprocessValue(value any) any {
	// If already a ValueSpec, return as-is
	if _, ok := value.(config.ValueSpec); ok {
		return value
	}

	switch v := value.(type) {
	case string:
		// Only convert strings with special prefixes to ValueSpec
		// Keep normal strings as-is for backward compatibility
		if strings.HasPrefix(v, "$js:") || strings.HasPrefix(v, "$var:") ||
			strings.HasPrefix(v, "$secret:") || strings.HasPrefix(v, "$env:") {
			return ParseConfigValue(v)
		}
		// Return normal strings as-is
		return v

	case map[string]any:
		// Recursively process maps
		return preprocessStepConfig(v)

	case []any:
		// Process array elements
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = preprocessValue(item)
		}
		return result

	default:
		// Return other types (numbers, booleans, nil) as-is
		// They will be wrapped in StaticValue by the step factory if needed
		return v
	}
}

// GenerateEventID generates a unique ID for an event
func GenerateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// RegisterDynamicAPIServices registers all services from the registry as step types
func RegisterDynamicAPIServices(serviceRegistry *ServiceRegistry) error {
	for _, serviceName := range serviceRegistry.List() {
		// Capture serviceName for the closure
		svcName := serviceName

		RegisterStepType(svcName, func(cfg map[string]any) (models.Step, error) {
			// Get the service definition
			serviceDef, exists := serviceRegistry.Get(svcName)
			if !exists {
				return nil, fmt.Errorf("service definition not found: %s", svcName)
			}

			// Extract operation from configuration
			operation, ok := cfg["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("missing required field 'operation' for service %s", svcName)
			}

			// Verify the operation exists
			opDef, err := serviceDef.GetOperation(operation)
			if err != nil {
				return nil, fmt.Errorf("invalid operation for service %s: %w", svcName, err)
			}

			// Build config.ValueSpec context from parameters
			// Use all parameters from config except 'operation'
			valueContext := make(map[string]config.ValueSpec)
			for k, v := range cfg {
				if k != "operation" {
					valueContext[k] = ParseConfigValue(v)
				}
			}

			// Build config.ValueSpec for URL (base_url + path + query params)
			urlSpec, err := buildURLSpec(serviceDef, opDef, valueContext)
			if err != nil {
				return nil, fmt.Errorf("failed to build URL spec: %w", err)
			}

			// Build config.ValueSpec for method
			methodSpec := config.NewStaticValue(opDef.Method)

			// Render headers (defaults + auth + operation-specific)
			// TODO: in the future these could also be config.ValueSpec
			headers, err := renderHeaders(serviceDef, opDef, valueContext)
			if err != nil {
				return nil, fmt.Errorf("failed to render headers: %w", err)
			}

			// Build config.ValueSpec for body if present using BodyResolver
			var bodySpec config.ValueSpec
			if opDef.Body != nil {
				resolver := NewBodyResolver(serviceDef, opDef)
				bodySpec, err = resolver.ResolveBody(valueContext)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve body: %w", err)
				}
			}

			// Determine content-type (operation > service defaults > default "application/json")
			contentType := opDef.ContentType
			if contentType == "" {
				contentType = serviceDef.Defaults.ContentType
			}
			if contentType == "" {
				contentType = "application/json"
			}

			// Determine response type
			responseType := opDef.ResponseType
			if responseType == "" {
				responseType = "json"
			}

			// Create config for http_client step
			// Pass config.ValueSpec directly
			httpConfig := map[string]any{
				"url":          urlSpec,
				"method":       methodSpec,
				"headers":      headers,
				"content_type": contentType,
				"response":     responseType,
			}
			if bodySpec != nil {
				httpConfig["body"] = bodySpec
			}

			// Create HTTPClientStep using the registered factory
			return CreateStep("http_client", httpConfig)
		})
	}

	return nil
}

// buildURLSpec builds a config.ValueSpec for the complete URL (base_url + path + query params)
func buildURLSpec(serviceDef *config.ServiceDefinition, opDef *config.OperationDef, context map[string]config.ValueSpec) (config.ValueSpec, error) {
	// Check if there are dynamic values in the context
	hasDynamic := config.HasDynamicValues(context)

	if !hasDynamic {
		// All values are static: render complete URL as static string
		staticContext := config.ExtractStaticValues(context)

		// Render base_url
		baseURL, err := renderGoTemplate(serviceDef.Defaults.BaseURL, staticContext)
		if err != nil {
			return nil, fmt.Errorf("failed to render base_url: %w", err)
		}

		// Render path
		path, err := renderGoTemplate(opDef.Path, staticContext)
		if err != nil {
			return nil, fmt.Errorf("failed to render path: %w", err)
		}

		// Combine base URL and path
		baseURLStr := strings.TrimRight(baseURL, "/")
		pathStr := strings.TrimLeft(path, "/")
		url := baseURLStr + "/" + pathStr

		// Render query parameters if present
		if len(opDef.QueryParams) > 0 {
			queryParts := []string{}
			for key, valueTemplate := range opDef.QueryParams {
				value, err := renderGoTemplate(valueTemplate, staticContext)
				if err != nil {
					return nil, fmt.Errorf("failed to render query param %s: %w", key, err)
				}
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, value))
			}
			url += "?" + strings.Join(queryParts, "&")
		}

		return config.NewStaticValue(url), nil
	}

	// At least one value is dynamic: build a JavaScript expression
	baseURL, err := renderTemplate(serviceDef.Defaults.BaseURL, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render base_url: %w", err)
	}

	path, err := renderTemplate(opDef.Path, context)
	if err != nil {
		return nil, fmt.Errorf("failed to render path: %w", err)
	}

	// Combine base URL and path
	baseURLStr := strings.TrimRight(baseURL, "/")
	pathStr := strings.TrimLeft(path, "/")

	var urlExpr string
	// If path is a JS expression (starts with quote), concatenate as JS
	if len(pathStr) > 0 && pathStr[0] == '\'' {
		urlExpr = jsStringLiteral(baseURLStr) + " + " + pathStr
	} else {
		// Static path, but baseURL might be dynamic
		urlExpr = jsStringLiteral(baseURLStr + "/" + pathStr)
	}

	// TODO: handle dynamic query parameters
	if len(opDef.QueryParams) > 0 {
		// For now we don't support dynamic query params
		// In the future could do: urlExpr + " + '?key=' + ctx.value"
	}

	return config.DynamicValue{
		Language:   "js",
		Expression: urlExpr,
	}, nil
}

// renderHeaders builds all headers as ValueSpec
func renderHeaders(serviceDef *config.ServiceDefinition, opDef *config.OperationDef, context map[string]config.ValueSpec) (map[string]config.ValueSpec, error) {
	headers := make(map[string]config.ValueSpec)

	// Add default headers
	for k, v := range serviceDef.Defaults.Headers {
		headerSpec, err := buildHeaderValueSpec(v, context)
		if err != nil {
			return nil, fmt.Errorf("failed to build default header %s: %w", k, err)
		}
		headers[k] = headerSpec
	}

	// Add authentication
	if serviceDef.Defaults.Auth != nil {
		authHeaders, err := renderAuthHeaders(serviceDef.Defaults.Auth, context)
		if err != nil {
			return nil, fmt.Errorf("failed to render auth headers: %w", err)
		}
		for k, v := range authHeaders {
			headers[k] = v
		}
	}

	// Add operation-specific headers (can override defaults)
	for k, v := range opDef.Headers {
		headerSpec, err := buildHeaderValueSpec(v, context)
		if err != nil {
			return nil, fmt.Errorf("failed to build operation header %s: %w", k, err)
		}
		headers[k] = headerSpec
	}

	return headers, nil
}

// buildHeaderValueSpec converts a header template to a ValueSpec
func buildHeaderValueSpec(tmpl string, context map[string]config.ValueSpec) (config.ValueSpec, error) {
	// If no template markers, return static value
	if !strings.Contains(tmpl, "{{") {
		return config.NewStaticValue(tmpl), nil
	}

	// Check if there are dynamic values in context
	hasDynamic := config.HasDynamicValues(context)

	if !hasDynamic {
		// All static: render with Go template
		staticContext := config.ExtractStaticValues(context)
		rendered, err := renderGoTemplate(tmpl, staticContext)
		if err != nil {
			return nil, err
		}
		return config.NewStaticValue(rendered), nil
	}

	// Has dynamic values: convert to JavaScript expression
	jsExpr, err := convertGoTemplateToJS(tmpl, context)
	if err != nil {
		return nil, err
	}

	return config.DynamicValue{
		Language:   "js",
		Expression: jsExpr,
	}, nil
}

// renderAuthHeaders builds authentication headers as ValueSpec
func renderAuthHeaders(auth *config.AuthConfig, context map[string]config.ValueSpec) (map[string]config.ValueSpec, error) {
	headers := make(map[string]config.ValueSpec)

	switch auth.Type {
	case "bearer", "api_key", "custom":
		valueSpec, err := buildHeaderValueSpec(auth.Value, context)
		if err != nil {
			return nil, fmt.Errorf("failed to build auth value: %w", err)
		}
		headers[auth.Header] = valueSpec

	case "basic":
		// For basic auth, build a JavaScript expression that encodes credentials
		usernameSpec, err := buildHeaderValueSpec(auth.Username, context)
		if err != nil {
			return nil, fmt.Errorf("failed to build auth username: %w", err)
		}
		passwordSpec, err := buildHeaderValueSpec(auth.Password, context)
		if err != nil {
			return nil, fmt.Errorf("failed to build auth password: %w", err)
		}

		// If both are static, encode now
		if usernameSpec.IsStatic() && passwordSpec.IsStatic() {
			username, _ := usernameSpec.GetStaticValue()
			password, _ := passwordSpec.GetStaticValue()
			credentials := fmt.Sprintf("%s:%s", username, password)
			encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
			headers["Authorization"] = config.NewStaticValue("Basic " + encoded)
		} else {
			// At least one is dynamic - create a JS expression
			// Note: This would require btoa() in JavaScript for base64 encoding
			// For now, return an error as this is a complex case
			return nil, fmt.Errorf("dynamic basic auth is not yet supported - use bearer or api_key auth with $var:/$secret: instead")
		}
	}

	return headers, nil
}

// renderTemplate renders a Go template using config.ValueSpec context
// If all values are static, uses standard Go template
// If at least one value is dynamic, converts Go template to JavaScript expression
func renderTemplate(tmplStr string, context map[string]config.ValueSpec) (string, error) {
	// If it doesn't contain template markers, return as string
	if !strings.Contains(tmplStr, "{{") {
		return tmplStr, nil
	}

	// Check if there are dynamic values
	hasDynamic := config.HasDynamicValues(context)

	if !hasDynamic {
		// All values are static: use standard Go template
		staticContext := config.ExtractStaticValues(context)
		return renderGoTemplate(tmplStr, staticContext)
	} else {
		// At least one value is dynamic: convert to JavaScript expression
		return convertGoTemplateToJS(tmplStr, context)
	}
}

// renderGoTemplate uses Go's standard template engine
func renderGoTemplate(tmplStr string, context map[string]any) (string, error) {
	tmpl, err := template.New("tpl").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, context); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// convertGoTemplateToJS converts a Go template to a JavaScript expression
// Example: "/item/{{.item_id}}.json" with item_id="ctx.step.id" → "'/item/' + ctx.step.id + '.json'"
func convertGoTemplateToJS(tmplStr string, context map[string]config.ValueSpec) (string, error) {
	var parts []string
	remaining := tmplStr

	for {
		// Find the next placeholder {{.name}}
		startIdx := strings.Index(remaining, "{{")
		if startIdx == -1 {
			// No more placeholders, add the rest as string literal
			if remaining != "" {
				parts = append(parts, jsStringLiteral(remaining))
			}
			break
		}

		// Add the static part before the placeholder
		if startIdx > 0 {
			parts = append(parts, jsStringLiteral(remaining[:startIdx]))
		}

		// Find the end of the placeholder
		endIdx := strings.Index(remaining[startIdx:], "}}")
		if endIdx == -1 {
			return "", fmt.Errorf("unclosed template marker in: %s", tmplStr)
		}
		endIdx += startIdx + 2

		// Extract variable name (remove {{ . }})
		placeholder := remaining[startIdx:endIdx]
		varName := strings.TrimSpace(placeholder[2 : len(placeholder)-2])
		varName = strings.TrimPrefix(varName, ".")

		// Look up value in context
		valueSpec, exists := context[varName]
		if !exists {
			return "", fmt.Errorf("template variable '%s' not found in context", varName)
		}

		// Add the appropriate JavaScript expression
		if staticVal, ok := valueSpec.GetStaticValue(); ok {
			// Static value: convert to JS string literal
			parts = append(parts, jsStringLiteral(fmt.Sprintf("%v", staticVal)))
		} else if dynVal, ok := valueSpec.GetDynamicExpression(); ok {
			// Dynamic value: use JavaScript expression directly
			parts = append(parts, dynVal.Expression)
		} else if varRef, ok := valueSpec.(config.VariableReference); ok {
			// Variable reference: generate $vars.name expression
			parts = append(parts, fmt.Sprintf("$vars.%s", varRef.Name))
		} else if secretRef, ok := valueSpec.(config.SecretReference); ok {
			// Secret reference: generate $secrets.name expression
			parts = append(parts, fmt.Sprintf("$secrets.%s", secretRef.Name))
		} else if envRef, ok := valueSpec.(config.EnvReference); ok {
			// Environment variable reference: these should be resolved at build time
			// This shouldn't happen in practice since env vars are resolved early,
			// but handle it gracefully by generating a runtime error
			return "", fmt.Errorf("environment variable reference '%s' should be resolved before template rendering", envRef.Name)
		} else {
			return "", fmt.Errorf("unsupported ValueSpec type for template variable '%s'", varName)
		}

		// Continue with the rest
		remaining = remaining[endIdx:]
	}

	// If there's only one part and it's not a string literal, return as is
	if len(parts) == 1 && !strings.HasPrefix(parts[0], "'") {
		return parts[0], nil
	}

	// Otherwise concatenate with +
	return strings.Join(parts, " + "), nil
}

// jsStringLiteral converts a Go string to a JavaScript string literal
func jsStringLiteral(s string) string {
	if s == "" {
		return "''"
	}

	// Escape special characters for JavaScript
	escaped := strings.ReplaceAll(s, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")

	return fmt.Sprintf("'%s'", escaped)
}

// parseConfigValue converts a configuration value to config.ValueSpec
// Recognizes special prefixes:
// - "$js:" for dynamic JavaScript expressions
// - "$var:" for global variable references
// - "$secret:" for global secret references
// - "$env:" for environment variable references
func ParseConfigValue(v any) config.ValueSpec {
	// If it's already a ValueSpec, return it as-is (idempotent)
	if vs, ok := v.(config.ValueSpec); ok {
		return vs
	}

	// If it's a string, check for special prefixes
	if str, ok := v.(string); ok {
		// Check for $js: prefix (dynamic JavaScript expression)
		if strings.HasPrefix(str, "$js:") {
			expr := strings.TrimPrefix(str, "$js:")
			expr = strings.TrimSpace(expr)
			return config.DynamicValue{
				Language:   "js",
				Expression: expr,
			}
		}

		// Check for $var: prefix (global variable reference)
		if strings.HasPrefix(str, "$var:") {
			varName := strings.TrimPrefix(str, "$var:")
			varName = strings.TrimSpace(varName)
			return config.VariableReference{Name: varName}
		}

		// Check for $secret: prefix (global secret reference)
		if strings.HasPrefix(str, "$secret:") {
			secretName := strings.TrimPrefix(str, "$secret:")
			secretName = strings.TrimSpace(secretName)
			return config.SecretReference{Name: secretName}
		}

		// Check for $env: prefix (environment variable reference)
		if strings.HasPrefix(str, "$env:") {
			envName := strings.TrimPrefix(str, "$env:")
			envName = strings.TrimSpace(envName)
			return config.EnvReference{Name: envName}
		}
	}

	// Otherwise it's a static value
	return config.NewStaticValue(v)
}

// GetRegisteredStepTypes returns all registered step types
func GetRegisteredStepTypes() []string {
	return ListStepTypes()
}
