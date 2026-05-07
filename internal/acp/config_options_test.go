package acp

import (
	"slices"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
)

func TestSessionConfigOptionsFromSDK(t *testing.T) {
	t.Run("Should convert select boolean and grouped config options", func(t *testing.T) {
		t.Parallel()

		description := "Choose active model"
		booleanDescription := "Enable verbose output"
		grouped := acpsdk.SessionConfigSelectOptionsGrouped{
			{
				Group: "frontier",
				Name:  "Frontier",
				Options: []acpsdk.SessionConfigSelectOption{
					{Value: "model-a", Name: "Model A", Description: &description},
				},
			},
		}

		options := sessionConfigOptionsFromSDK([]acpsdk.SessionConfigOption{
			{
				Select: &acpsdk.SessionConfigOptionSelect{
					Id:           "model",
					Name:         "Model",
					Description:  &description,
					CurrentValue: "model-a",
					Options: acpsdk.SessionConfigSelectOptions{
						Grouped: &grouped,
					},
					Type: "select",
				},
			},
			{
				Boolean: &acpsdk.SessionConfigOptionBoolean{
					Id:           "verbose",
					Name:         "Verbose",
					Description:  &booleanDescription,
					CurrentValue: true,
					Type:         "boolean",
				},
			},
			{
				Select: &acpsdk.SessionConfigOptionSelect{
					Id:   " ",
					Name: "Ignored",
					Type: "select",
				},
			},
		})

		if len(options) != 2 {
			t.Fatalf("sessionConfigOptionsFromSDK() len = %d, want 2: %#v", len(options), options)
		}
		model := options[0]
		if model.ID != "model" || model.Label != "Model" || model.Description != description ||
			model.Kind != SessionConfigOptionKindSelect || model.Current != "model-a" {
			t.Fatalf("model option = %#v", model)
		}
		if len(model.Values) != 1 || model.Values[0].Value != "model-a" ||
			model.Values[0].Description != description {
			t.Fatalf("model values = %#v", model.Values)
		}
		boolean := options[1]
		if boolean.ID != "verbose" || boolean.Kind != SessionConfigOptionKindBoolean || boolean.Current != "true" ||
			boolean.Description != booleanDescription {
			t.Fatalf("boolean option = %#v", boolean)
		}
	})

	t.Run("Should return nil for absent options", func(t *testing.T) {
		t.Parallel()

		if got := sessionConfigOptionsFromSDK(nil); got != nil {
			t.Fatalf("sessionConfigOptionsFromSDK(nil) = %#v, want nil", got)
		}
	})
}

func TestConfigOptionMatching(t *testing.T) {
	t.Parallel()

	options := []SessionConfigOption{
		{ID: "verbose", Kind: SessionConfigOptionKindBoolean, Current: "true"},
		{
			ID:      "model",
			Kind:    SessionConfigOptionKindSelect,
			Current: "model-a",
			Values:  []SessionConfigOptionValue{{Value: "model-a"}, {Value: "model-b"}},
		},
		{
			ID:      "effort",
			Kind:    SessionConfigOptionKindSelect,
			Current: "low",
			Values:  []SessionConfigOptionValue{{Value: "low"}, {Value: "high"}},
		},
	}

	t.Run("Should find model and reasoning config options", func(t *testing.T) {
		t.Parallel()

		model, ok := findModelConfigOption(options)
		if !ok || model.ID != "model" {
			t.Fatalf("findModelConfigOption() = %#v, %v", model, ok)
		}
		reasoning, ok := findReasoningConfigOption(options)
		if !ok || reasoning.ID != "effort" {
			t.Fatalf("findReasoningConfigOption() = %#v, %v", reasoning, ok)
		}
	})

	t.Run("Should allow only advertised select option values", func(t *testing.T) {
		t.Parallel()

		model, ok := findModelConfigOption(options)
		if !ok {
			t.Fatal("findModelConfigOption() ok = false, want true")
		}
		if !configOptionAllowsValue(model, "model-b") {
			t.Fatal("configOptionAllowsValue() rejected advertised model")
		}
		if configOptionAllowsValue(model, "model-c") {
			t.Fatal("configOptionAllowsValue() accepted unadvertised model")
		}
		if configOptionAllowsValue(options[0], "true") {
			t.Fatal("configOptionAllowsValue() accepted boolean option as select")
		}
	})
}

func TestLegacyModelStateAllows(t *testing.T) {
	t.Parallel()

	caps := Caps{SupportedModels: []string{"model-a", "model-b"}}

	t.Run("Should allow only advertised legacy models", func(t *testing.T) {
		t.Parallel()

		if !legacyModelStateAllows(caps, "model-b") {
			t.Fatal("legacyModelStateAllows() rejected advertised model")
		}
		if legacyModelStateAllows(caps, "model-c") {
			t.Fatal("legacyModelStateAllows() accepted unadvertised model")
		}
		if legacyModelStateAllows(Caps{}, "model-a") {
			t.Fatal("legacyModelStateAllows() accepted model without legacy state")
		}
	})

	t.Run("Should preserve legacy model lists when cloning caps", func(t *testing.T) {
		t.Parallel()

		if !slices.Equal(CloneCaps(caps).SupportedModels, caps.SupportedModels) {
			t.Fatalf("CloneCaps() did not preserve models")
		}
	})
}
