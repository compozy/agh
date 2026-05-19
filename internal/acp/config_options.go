package acp

import (
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"
)

const (
	booleanTrueText = "true"
)

func sessionConfigOptionsFromSDK(options []acpsdk.SessionConfigOption) []SessionConfigOption {
	if len(options) == 0 {
		return nil
	}
	converted := make([]SessionConfigOption, 0, len(options))
	for _, option := range options {
		if convertedOption, ok := sessionConfigOptionFromSDK(option); ok {
			converted = append(converted, convertedOption)
		}
	}
	return converted
}

func sessionConfigOptionFromSDK(option acpsdk.SessionConfigOption) (SessionConfigOption, bool) {
	switch {
	case option.Select != nil:
		selectOption := option.Select
		id := strings.TrimSpace(string(selectOption.Id))
		if id == "" {
			return SessionConfigOption{}, false
		}
		return SessionConfigOption{
			ID:          id,
			Label:       strings.TrimSpace(selectOption.Name),
			Description: trimStringPointer(selectOption.Description),
			Kind:        SessionConfigOptionKindSelect,
			Current:     strings.TrimSpace(string(selectOption.CurrentValue)),
			Values:      sessionConfigValuesFromSDK(selectOption.Options),
		}, true
	case option.Boolean != nil:
		booleanOption := option.Boolean
		id := strings.TrimSpace(string(booleanOption.Id))
		if id == "" {
			return SessionConfigOption{}, false
		}
		current := "false"
		if booleanOption.CurrentValue {
			current = booleanTrueText
		}
		return SessionConfigOption{
			ID:          id,
			Label:       strings.TrimSpace(booleanOption.Name),
			Description: trimStringPointer(booleanOption.Description),
			Kind:        SessionConfigOptionKindBoolean,
			Current:     current,
		}, true
	default:
		return SessionConfigOption{}, false
	}
}

func sessionConfigValuesFromSDK(options acpsdk.SessionConfigSelectOptions) []SessionConfigOptionValue {
	values := make([]SessionConfigOptionValue, 0)
	if options.Ungrouped != nil {
		for _, value := range *options.Ungrouped {
			values = append(values, sessionConfigValueFromSDK(value))
		}
	}
	if options.Grouped != nil {
		for _, group := range *options.Grouped {
			for _, value := range group.Options {
				values = append(values, sessionConfigValueFromSDK(value))
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func sessionConfigValueFromSDK(value acpsdk.SessionConfigSelectOption) SessionConfigOptionValue {
	return SessionConfigOptionValue{
		Value:       strings.TrimSpace(string(value.Value)),
		Label:       strings.TrimSpace(value.Name),
		Description: trimStringPointer(value.Description),
	}
}

func trimStringPointer(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func findModelConfigOption(options []SessionConfigOption) (SessionConfigOption, bool) {
	return findSelectConfigOption(options, "model")
}

func findReasoningConfigOption(options []SessionConfigOption) (SessionConfigOption, bool) {
	return findSelectConfigOption(options, "reasoning_effort", "effort")
}

func findSelectConfigOption(options []SessionConfigOption, candidateIDs ...string) (SessionConfigOption, bool) {
	for _, candidateID := range candidateIDs {
		for _, option := range options {
			if option.Kind != SessionConfigOptionKindSelect {
				continue
			}
			if strings.TrimSpace(option.ID) == candidateID {
				return option, true
			}
		}
	}
	return SessionConfigOption{}, false
}

func configOptionAllowsValue(option SessionConfigOption, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || option.Kind != SessionConfigOptionKindSelect {
		return false
	}
	for _, candidate := range option.Values {
		if strings.TrimSpace(candidate.Value) == value {
			return true
		}
	}
	return false
}

func legacyModelStateAllows(caps Caps, modelID string) bool {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return false
	}
	for _, candidate := range caps.SupportedModels {
		if strings.TrimSpace(candidate) == modelID {
			return true
		}
	}
	return false
}
