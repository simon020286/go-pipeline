package builder

import (
	"fmt"

	"github.com/simon020286/go-pipeline/config"
	"github.com/simon020286/go-pipeline/models"
)

// BodyResolver gestisce la risoluzione del body da template YAML a struttura Go
type BodyResolver struct {
	serviceDef *config.ServiceDefinition
	opDef      *config.OperationDef
}

// NewBodyResolver crea un nuovo BodyResolver
func NewBodyResolver(serviceDef *config.ServiceDefinition, opDef *config.OperationDef) *BodyResolver {
	return &BodyResolver{
		serviceDef: serviceDef,
		opDef:      opDef,
	}
}

// ResolveBody costruisce il body risolvendo i parametri
// Ritorna un config.ValueSpec che rappresenta il body completo
func (br *BodyResolver) ResolveBody(userParams map[string]config.ValueSpec) (config.ValueSpec, error) {
	// 1. Merge params (global + operation defaults + user)
	mergedParams, err := br.mergeParams(userParams)
	if err != nil {
		return nil, err
	}

	// 2. Validate required params
	if err := br.validateParams(mergedParams); err != nil {
		return nil, err
	}

	// 3. Build body structure
	if br.opDef.Body == nil {
		return nil, nil
	}

	bodyValue, err := br.resolveBodyField(br.opDef.Body, mergedParams)
	if err != nil {
		return nil, err
	}

	return bodyValue, nil
}

// mergeParams unisce global, operation e user params
// Priority: userParams > opDef.Params (con defaults) > globalParams (con defaults)
func (br *BodyResolver) mergeParams(userParams map[string]config.ValueSpec) (map[string]config.ValueSpec, error) {
	merged := make(map[string]config.ValueSpec)

	// 1. Start with global params defaults
	if br.serviceDef.GlobalParams != nil {
		for name, paramDef := range br.serviceDef.GlobalParams {
			if paramDef.Default != nil {
				merged[name] = config.NewStaticValue(paramDef.Default)
			}
		}
	}

	// 2. Apply operation params defaults
	if br.opDef.Params != nil {
		for name, paramDef := range br.opDef.Params {
			if paramDef.Default != nil {
				merged[name] = config.NewStaticValue(paramDef.Default)
			}
		}
	}

	// 3. Override with user params
	for name, value := range userParams {
		merged[name] = value
	}

	return merged, nil
}

// validateParams valida che tutti i parametri required siano presenti
func (br *BodyResolver) validateParams(params map[string]config.ValueSpec) error {
	// Check operation required params
	if br.opDef.Params != nil {
		for name, paramDef := range br.opDef.Params {
			if paramDef.IsRequired() {
				if _, exists := params[name]; !exists {
					return fmt.Errorf("required parameter '%s' not provided", name)
				}
			}
		}
	}

	return nil
}

// resolveBodyField risolve un singolo campo del body (ricorsivo per strutture annidate)
func (br *BodyResolver) resolveBodyField(field any, params map[string]config.ValueSpec) (config.ValueSpec, error) {
	switch v := field.(type) {
	case map[string]any:
		return br.resolveBodyMap(v, params)
	case []any:
		return br.resolveBodyArray(v, params)
	case string, int, float64, bool, nil:
		// Valore statico primitivo
		return config.NewStaticValue(v), nil
	default:
		// Per altri tipi, trattali come statici
		return config.NewStaticValue(v), nil
	}
}

// resolveBodyMap risolve un map (oggetto) nel body
func (br *BodyResolver) resolveBodyMap(m map[string]any, params map[string]config.ValueSpec) (config.ValueSpec, error) {
	// Check for conditional structure
	if _, hasIf := m["$if"]; hasIf {
		return br.resolveConditional(m, params)
	}

	// Check for array template
	if _, hasForEach := m["$for_each"]; hasForEach {
		return br.resolveArrayTemplate(m, params)
	}

	// Check if it's a $param reference
	if paramName, ok := m["$param"].(string); ok {
		// Reference to a parameter
		paramValue, exists := params[paramName]
		if !exists {
			// Check if it's optional
			if br.opDef.Params != nil {
				if paramDef, found := br.opDef.Params[paramName]; found && paramDef.IsOptional() {
					// Optional param not provided -> return nil to omit field
					return nil, nil
				}
			}
			return nil, fmt.Errorf("parameter '%s' not found", paramName)
		}
		return paramValue, nil
	}

	// Otherwise, it's a nested object - resolve all fields
	resolvedFields := make(map[string]config.ValueSpec)
	hasDynamic := false

	for key, value := range m {
		resolved, err := br.resolveBodyField(value, params)
		if err != nil {
			return nil, fmt.Errorf("field '%s': %w", key, err)
		}

		// Omit nil values (optional params not provided)
		if resolved == nil {
			continue
		}

		// Check if dynamic
		if !resolved.IsStatic() {
			hasDynamic = true
		}

		resolvedFields[key] = resolved
	}

	// If all values are static, return as static map
	if !hasDynamic {
		staticMap := make(map[string]any)
		for k, v := range resolvedFields {
			if staticVal, isStatic := v.GetStaticValue(); isStatic {
				staticMap[k] = staticVal
			}
		}
		return config.NewStaticValue(staticMap), nil
	}

	// Otherwise, return a structured body with mix of static/dynamic fields
	return &StructuredBody{Fields: resolvedFields}, nil
}

// resolveConditional risolve una struttura condizionale nel body
func (br *BodyResolver) resolveConditional(cond map[string]any, params map[string]config.ValueSpec) (config.ValueSpec, error) {
	// Extract condition
	ifCond, ok := cond["$if"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid conditional structure: $if must be a map")
	}

	// Evaluate condition
	conditionMet, err := br.evaluateCondition(ifCond, params)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate condition: %w", err)
	}

	// Return appropriate branch
	if conditionMet {
		if thenVal, hasThen := cond["$then"]; hasThen {
			return br.resolveBodyField(thenVal, params)
		}
	} else {
		if elseVal, hasElse := cond["$else"]; hasElse {
			return br.resolveBodyField(elseVal, params)
		}
	}

	// If no matching branch, return nil (omit field)
	return nil, nil
}

// evaluateCondition valuta una condizione
func (br *BodyResolver) evaluateCondition(cond map[string]any, params map[string]config.ValueSpec) (bool, error) {
	// Get the parameter name (required for all condition types)
	paramName, hasParam := cond["$param"].(string)
	if !hasParam {
		return false, fmt.Errorf("condition must have $param")
	}

	// Check for $exists operator (doesn't need parameter value)
	if exists, ok := cond["$exists"].(bool); ok {
		_, paramExists := params[paramName]
		if exists {
			// Condition is true if parameter exists
			return paramExists, nil
		} else {
			// Condition is true if parameter doesn't exist
			return !paramExists, nil
		}
	}

	// For other operators, we need to resolve the parameter value
	paramValue, paramExists := params[paramName]
	if !paramExists {
		return false, nil // Parameter not provided = condition false
	}

	resolved, err := paramValue.Resolve(&models.StepInput{Data: make(map[string]map[string]*models.Data)})
	if err != nil {
		return false, fmt.Errorf("failed to resolve parameter '%s': %w", paramName, err)
	}

	// Check for $equals operator
	if equals, ok := cond["$equals"]; ok {
		return resolved == equals, nil
	}

	// Check for $not_empty operator
	if notEmpty, ok := cond["$not_empty"].(bool); ok {
		if notEmpty {
			return br.valueExists(resolved), nil
		} else {
			return !br.valueExists(resolved), nil
		}
	}

	// Check for $is_empty operator
	if isEmpty, ok := cond["$is_empty"].(bool); ok {
		if isEmpty {
			return !br.valueExists(resolved), nil
		} else {
			return br.valueExists(resolved), nil
		}
	}

	// Default: just check if parameter has a value (implicit $not_empty: true)
	return br.valueExists(resolved), nil
}

// valueExists checks if a value is not null/empty
func (br *BodyResolver) valueExists(value any) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	default:
		return true // For numbers, booleans, etc.
	}
}

// resolveArrayTemplate risolve un template per array
func (br *BodyResolver) resolveArrayTemplate(arrTmpl map[string]any, params map[string]config.ValueSpec) (config.ValueSpec, error) {
	forEach, ok := arrTmpl["$for_each"].(string)
	if !ok {
		return nil, fmt.Errorf("array template must have $for_each parameter as string")
	}

	template, hasTemplate := arrTmpl["$template"]
	if !hasTemplate {
		return nil, fmt.Errorf("array template must have $template")
	}

	// Get array parameter
	arrayParam, exists := params[forEach]
	if !exists {
		return nil, fmt.Errorf("array parameter '%s' not found", forEach)
	}

	// Resolve array value
	resolvedArray, err := arrayParam.Resolve(&models.StepInput{Data: make(map[string]map[string]*models.Data)})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve array parameter '%s': %w", forEach, err)
	}

	array, ok := resolvedArray.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter '%s' is not an array", forEach)
	}

	// Apply template to each item
	resolvedItems := make([]config.ValueSpec, 0, len(array))
	hasDynamic := false

	for i, item := range array {
		// Create context with $item
		itemParams := make(map[string]config.ValueSpec)
		for k, v := range params {
			itemParams[k] = v
		}
		itemParams["$item"] = config.NewStaticValue(item)

		// Resolve template with item context
		resolved, err := br.resolveBodyField(template, itemParams)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve template for array item %d: %w", i, err)
		}

		if resolved == nil {
			continue
		}

		if !resolved.IsStatic() {
			hasDynamic = true
		}

		resolvedItems = append(resolvedItems, resolved)
	}

	// If all items are static, return as static array
	if !hasDynamic {
		staticArray := make([]any, 0, len(resolvedItems))
		for _, v := range resolvedItems {
			if staticVal, isStatic := v.GetStaticValue(); isStatic {
				staticArray = append(staticArray, staticVal)
			}
		}
		return config.NewStaticValue(staticArray), nil
	}

	// Otherwise, return a structured body representing array
	return &StructuredBody{Array: resolvedItems}, nil
}

// resolveBodyArray risolve un array nel body
func (br *BodyResolver) resolveBodyArray(arr []any, params map[string]config.ValueSpec) (config.ValueSpec, error) {
	resolvedItems := make([]config.ValueSpec, 0, len(arr))
	hasDynamic := false

	for i, item := range arr {
		resolved, err := br.resolveBodyField(item, params)
		if err != nil {
			return nil, fmt.Errorf("array item %d: %w", i, err)
		}

		// Omit nil items
		if resolved == nil {
			continue
		}

		if !resolved.IsStatic() {
			hasDynamic = true
		}

		resolvedItems = append(resolvedItems, resolved)
	}

	// If all items are static, return as static array
	if !hasDynamic {
		staticArr := make([]any, 0, len(resolvedItems))
		for _, v := range resolvedItems {
			if staticVal, isStatic := v.GetStaticValue(); isStatic {
				staticArr = append(staticArr, staticVal)
			}
		}
		return config.NewStaticValue(staticArr), nil
	}

	// Otherwise, return a structured body representing the array
	return &StructuredBody{Array: resolvedItems}, nil
}

// StructuredBody rappresenta un body con campi strutturati (mix di static e dynamic)
// Implementa config.ValueSpec
type StructuredBody struct {
	Fields map[string]config.ValueSpec // Per oggetti
	Array  []config.ValueSpec          // Per array
}

// IsStatic ritorna sempre false perché StructuredBody contiene almeno un valore dinamico
func (sb *StructuredBody) IsStatic() bool {
	return false
}

// GetStaticValue ritorna nil perché StructuredBody non è statico
func (sb *StructuredBody) GetStaticValue() (any, bool) {
	return nil, false
}

// GetDynamicExpression ritorna false perché StructuredBody non è una semplice espressione JS
func (sb *StructuredBody) GetDynamicExpression() (config.DynamicValue, bool) {
	return config.DynamicValue{}, false
}

// Resolve risolve tutti i campi/elementi usando lo stato della pipeline
func (sb *StructuredBody) Resolve(state *models.StepInput) (any, error) {
	// Se è un oggetto
	if sb.Fields != nil {
		result := make(map[string]any)

		for key, valueSpec := range sb.Fields {
			resolved, err := valueSpec.Resolve(state)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve field '%s': %w", key, err)
			}
			result[key] = resolved
		}

		return result, nil
	}

	// Se è un array
	if sb.Array != nil {
		result := make([]any, 0, len(sb.Array))

		for i, valueSpec := range sb.Array {
			resolved, err := valueSpec.Resolve(state)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve array item %d: %w", i, err)
			}
			result = append(result, resolved)
		}

		return result, nil
	}

	return nil, fmt.Errorf("StructuredBody has neither Fields nor Array")
}
