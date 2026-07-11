package config

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/reasoning"
)

// Validate reports whether the provider model block is usable.
func (m ProviderModelsConfig) Validate(path string) error {
	if strings.TrimSpace(m.Default) == "" && m.Default != "" {
		return fmt.Errorf("%s.default must not be whitespace-only", path)
	}
	seen := make(map[string]struct{}, len(m.Curated))
	for idx, model := range m.Curated {
		modelPath := fmt.Sprintf("%s.curated[%d]", path, idx)
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return fmt.Errorf("%s.id is required", modelPath)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%s.id duplicates %q", modelPath, id)
		}
		seen[id] = struct{}{}
		efforts := make(map[string]struct{}, len(model.ReasoningEfforts))
		for effortIdx, effort := range model.ReasoningEfforts {
			effortPath := fmt.Sprintf("%s.reasoning_efforts[%d]", modelPath, effortIdx)
			trimmed := strings.TrimSpace(effort)
			if effort != trimmed {
				return &reasoning.InvalidEffortError{Path: effortPath, Value: effort}
			}
			if !reasoning.IsValid(effort) {
				return &reasoning.InvalidEffortError{Path: effortPath, Value: effort}
			}
			if _, ok := efforts[effort]; ok {
				return fmt.Errorf("%s duplicates %q", effortPath, effort)
			}
			efforts[effort] = struct{}{}
		}
		defaultEffort := model.DefaultReasoningEffort
		if defaultEffort != "" {
			defaultPath := modelPath + ".default_reasoning_effort"
			trimmedDefault := strings.TrimSpace(defaultEffort)
			if defaultEffort != trimmedDefault {
				return &reasoning.InvalidEffortError{Path: defaultPath, Value: defaultEffort}
			}
			if !reasoning.IsValid(defaultEffort) {
				return &reasoning.InvalidEffortError{Path: defaultPath, Value: defaultEffort}
			}
			if len(efforts) > 0 {
				if _, ok := efforts[defaultEffort]; !ok {
					return fmt.Errorf("%s must be listed in reasoning_efforts", defaultPath)
				}
			}
		}
		if err := validateProviderModelReleaseDate(modelPath, model.ReleaseDate); err != nil {
			return err
		}
	}
	if err := m.Discovery.Validate(path + ".discovery"); err != nil {
		return err
	}
	return m.Reasoning.Validate(path + ".reasoning")
}
