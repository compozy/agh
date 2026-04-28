package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
)

const schemaTypeObject = "object"

func normalizeCallInput(input json.RawMessage) json.RawMessage {
	if len(bytes.TrimSpace(input)) == 0 {
		return json.RawMessage(`{}`)
	}
	return cloneRawMessage(input)
}

func validateCallInput(d Descriptor, input json.RawMessage) error {
	normalized := normalizeCallInput(input)
	if !json.Valid(normalized) {
		return NewToolError(
			ErrorCodeInvalidInput,
			d.ID,
			fmt.Sprintf("tool %q input is not valid JSON", d.ID),
			ErrToolInvalidInput,
			ReasonSchemaInvalid,
		)
	}
	if err := validateJSONSchemaValue(d.ID, d.InputSchema, normalized); err != nil {
		return err
	}
	return nil
}

func validateJSONSchemaValue(id ToolID, schema json.RawMessage, raw json.RawMessage) error {
	var schemaValue map[string]json.RawMessage
	if err := json.Unmarshal(schema, &schemaValue); err != nil {
		return NewToolError(
			ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("tool %q input schema is invalid", id),
			fmt.Errorf("%w: %w", ErrToolInvalidInput, err),
			ReasonSchemaInvalid,
		)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return NewToolError(
			ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("tool %q input is invalid", id),
			fmt.Errorf("%w: %w", ErrToolInvalidInput, err),
			ReasonSchemaInvalid,
		)
	}
	if err := validateSchemaNode("$", schemaValue, value); err != nil {
		return NewToolError(
			ErrorCodeInvalidInput,
			id,
			fmt.Sprintf("tool %q input failed schema validation", id),
			fmt.Errorf("%w: %w", ErrToolInvalidInput, err),
			ReasonSchemaInvalid,
		)
	}
	return nil
}

func validateSchemaNode(path string, schema map[string]json.RawMessage, value any) error {
	types, err := schemaTypes(schema["type"])
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if len(types) > 0 && !jsonValueMatchesAnyType(value, types) {
		return fmt.Errorf("%s: expected %v, got %s", path, types, jsonValueType(value))
	}
	if !slices.Contains(types, schemaTypeObject) && jsonValueType(value) != schemaTypeObject {
		return nil
	}
	objectValue, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	required, err := schemaStringArray(schema["required"])
	if err != nil {
		return fmt.Errorf("%s.required: %w", path, err)
	}
	for _, field := range required {
		if _, ok := objectValue[field]; !ok {
			return fmt.Errorf("%s.%s: required field missing", path, field)
		}
	}
	properties, err := schemaProperties(schema["properties"])
	if err != nil {
		return fmt.Errorf("%s.properties: %w", path, err)
	}
	if len(properties) > 0 {
		for key, childSchema := range properties {
			childValue, ok := objectValue[key]
			if !ok {
				continue
			}
			if err := validateSchemaNode(path+"."+key, childSchema, childValue); err != nil {
				return err
			}
		}
	}
	if additionalPropertiesFalse(schema["additionalProperties"]) {
		for key := range objectValue {
			if _, ok := properties[key]; !ok {
				return fmt.Errorf("%s.%s: additional property is not allowed", path, key)
			}
		}
	}
	return nil
}

func schemaTypes(raw json.RawMessage) ([]string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err != nil {
		return nil, fmt.Errorf("type must be a string or string array: %w", err)
	}
	return many, nil
}

func schemaStringArray(raw json.RawMessage) ([]string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func schemaProperties(raw json.RawMessage) (map[string]map[string]json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var values map[string]map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func additionalPropertiesFalse(raw json.RawMessage) bool {
	if len(bytes.TrimSpace(raw)) == 0 {
		return false
	}
	var allowed bool
	if err := json.Unmarshal(raw, &allowed); err != nil {
		return false
	}
	return !allowed
}

func jsonValueMatchesAnyType(value any, types []string) bool {
	for _, item := range types {
		if jsonValueMatchesType(value, item) {
			return true
		}
	}
	return false
}

func jsonValueMatchesType(value any, schemaType string) bool {
	switch schemaType {
	case schemaTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == float64(int64(number))
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func jsonValueType(value any) string {
	switch value.(type) {
	case map[string]any:
		return schemaTypeObject
	case []any:
		return "array"
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
