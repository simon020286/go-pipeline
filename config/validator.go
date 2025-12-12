package config

import "fmt"

// ValidateServiceDefinition valida l'intera definizione del servizio
// Questa funzione estende la validazione base in service.go con validazioni più approfondite
func ValidateServiceDefinition(def *ServiceDefinition) error {
	// First run basic validation
	if err := def.Validate(); err != nil {
		return err
	}

	// Validate each operation in detail
	for opName, opDef := range def.Operations {
		if err := validateOperation(opName, opDef); err != nil {
			return fmt.Errorf("invalid operation '%s': %w", opName, err)
		}
	}

	return nil
}

// validateOperation valida una singola operazione
func validateOperation(_ string, op OperationDef) error {
	// Validate that required params don't have defaults
	for paramName, paramDef := range op.Params {
		if paramDef.IsRequired() && paramDef.Default != nil {
			return fmt.Errorf("parameter '%s' is marked as required but has a default value", paramName)
		}
	}

	// Validate body references if body is a map
	if op.Body != nil {
		if err := validateBodyReferences(op.Body, op.Params); err != nil {
			return fmt.Errorf("invalid body: %w", err)
		}
	}

	return nil
}

// validateBodyReferences verifica che tutti i $param nel body esistano in params
func validateBodyReferences(body any, params map[string]ParameterDef) error {
	switch v := body.(type) {
	case map[string]any:
		// Check for conditional structure
		if _, hasIf := v["$if"]; hasIf {
			return validateConditionalDef(v, params)
		}

		// Check for array template
		if _, hasForEach := v["$for_each"]; hasForEach {
			return validateArrayTemplateDef(v, params)
		}

		// Check for $param reference
		if paramName, ok := v["$param"].(string); ok {
			if _, exists := params[paramName]; !exists {
				return fmt.Errorf("parameter '%s' referenced in body but not defined in params", paramName)
			}
			return nil
		}

		// Recurse into nested fields
		for key, value := range v {
			if err := validateBodyReferences(value, params); err != nil {
				return fmt.Errorf("field '%s': %w", key, err)
			}
		}

	case []any:
		// Validate array elements
		for i, item := range v {
			if err := validateBodyReferences(item, params); err != nil {
				return fmt.Errorf("array item %d: %w", i, err)
			}
		}

	case string:
		// String body is OK (backward compatibility or simple text)
		return nil
	}

	return nil
}

// validateConditionalDef validates a conditional structure
func validateConditionalDef(cond map[string]any, params map[string]ParameterDef) error {
	// Validate condition
	if ifCond, ok := cond["$if"].(map[string]any); ok {
		if err := validateConditionDef(ifCond, params); err != nil {
			return fmt.Errorf("invalid condition: %w", err)
		}
	}

	// Validate then branch
	if thenVal, hasThen := cond["$then"]; hasThen {
		if err := validateBodyReferences(thenVal, params); err != nil {
			return fmt.Errorf("invalid then branch: %w", err)
		}
	}

	// Validate else branch (optional)
	if elseVal, hasElse := cond["$else"]; hasElse {
		if err := validateBodyReferences(elseVal, params); err != nil {
			return fmt.Errorf("invalid else branch: %w", err)
		}
	}

	return nil
}

// validateConditionDef validates a condition definition
func validateConditionDef(cond map[string]any, params map[string]ParameterDef) error {
	// Check for $param
	if paramName, ok := cond["$param"].(string); ok {
		if _, exists := params[paramName]; !exists {
			return fmt.Errorf("condition references undefined parameter '%s'", paramName)
		}
	}

	// Validate condition operators
	operators := 0
	if _, hasExists := cond["$exists"]; hasExists {
		operators++
	}
	if _, hasEquals := cond["$equals"]; hasEquals {
		operators++
	}
	if _, hasNotEquals := cond["$not_equals"]; hasNotEquals {
		operators++
	}
	if _, hasNotEmpty := cond["$not_empty"]; hasNotEmpty {
		operators++
	}
	if _, hasIsEmpty := cond["$is_empty"]; hasIsEmpty {
		operators++
	}

	if operators == 0 {
		return fmt.Errorf("condition must have at least one operator ($exists, $equals, $not_equals, $not_empty, $is_empty)")
	}

	if operators > 1 {
		return fmt.Errorf("condition can only have one operator")
	}

	return nil
}

// validateArrayTemplateDef validates an array template structure
func validateArrayTemplateDef(arrTmpl map[string]any, _ map[string]ParameterDef) error {
	// Check for $for_each
	if _, ok := arrTmpl["$for_each"].(string); !ok {
		return fmt.Errorf("array template must have $for_each parameter")
	}

	// Check for $template or $array_map
	hasTemplate := false
	hasArrayMap := false

	if _, hasTemplate = arrTmpl["$template"]; hasTemplate {
		hasTemplate = true
	}

	if _, hasArrayMap = arrTmpl["$array_map"]; hasArrayMap {
		hasArrayMap = true
	}

	if !hasTemplate && !hasArrayMap {
		return fmt.Errorf("array template must have either $template or $array_map")
	}

	if hasTemplate && hasArrayMap {
		return fmt.Errorf("array template cannot have both $template and $array_map")
	}

	return nil
}
