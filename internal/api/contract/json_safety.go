package contract

import "encoding/json"

type jsonSafetyKeyPredicate func(string) bool
type jsonSafetyStringPredicate func(string) bool

func containsUnsafeJSON(
	data []byte,
	keyMatches jsonSafetyKeyPredicate,
	stringMatches jsonSafetyStringPredicate,
) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	return containsUnsafeJSONValue(value, keyMatches, stringMatches)
}

func containsUnsafeJSONValue(
	value any,
	keyMatches jsonSafetyKeyPredicate,
	stringMatches jsonSafetyStringPredicate,
) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if keyMatches != nil && keyMatches(key) {
				return true
			}
			if containsUnsafeJSONValue(child, keyMatches, stringMatches) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsUnsafeJSONValue(child, keyMatches, stringMatches) {
				return true
			}
		}
	case string:
		return stringMatches != nil && stringMatches(typed)
	}
	return false
}
