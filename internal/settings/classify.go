package settings

import (
	"errors"
	"fmt"
	"strings"
)

// ClassifyMutation maps one section or collection mutation onto the v1 runtime-apply matrix.
func ClassifyMutation(descriptor MutationDescriptor) (MutationClassification, error) {
	section := descriptor.Section
	if section == "" {
		return MutationClassification{}, errors.New("settings: mutation section is required")
	}

	action := strings.TrimSpace(descriptor.Action)
	if action != "" {
		return classifyAction(section, action)
	}
	if len(descriptor.ChangedFields) == 0 {
		return MutationClassification{}, errors.New("settings: mutation fields or action are required")
	}

	var (
		classification MutationClassification
		seen           bool
	)
	for _, field := range descriptor.ChangedFields {
		next, err := classifyField(section, field)
		if err != nil {
			return MutationClassification{}, err
		}
		if !seen {
			classification = next
			seen = true
			continue
		}
		if classification.Behavior != next.Behavior {
			return MutationClassification{}, fmt.Errorf(
				"settings: section %q mixes %q and %q changes in one mutation",
				section,
				classification.Behavior,
				next.Behavior,
			)
		}
	}

	return classification, nil
}

func classifyAction(section SectionName, action string) (MutationClassification, error) {
	switch section {
	case SectionGeneral:
		if action == "restart" {
			return MutationClassification{
				Behavior: MutationBehaviorActionTrigger,
				Applied:  true,
			}, nil
		}
	case SectionMemory:
		if action == "consolidate" {
			return MutationClassification{
				Behavior: MutationBehaviorActionTrigger,
				Applied:  true,
			}, nil
		}
	case SectionHooksExtensions:
		switch action {
		case "extension-install", "extension-enable", "extension-disable":
			return MutationClassification{
				Behavior: MutationBehaviorActionTrigger,
				Applied:  true,
			}, nil
		}
	}

	return MutationClassification{}, fmt.Errorf("settings: unsupported action %q for section %q", action, section)
}

func classifyField(section SectionName, field string) (MutationClassification, error) {
	trimmed := strings.TrimSpace(field)
	if trimmed == "" {
		return MutationClassification{}, errors.New("settings: mutation field is required")
	}

	switch section {
	case SectionGeneral, SectionMemory, SectionAutomation, SectionNetwork, SectionObservability:
		return restartRequiredClassification(), nil
	case SectionSkills:
		if strings.HasPrefix(trimmed, "skills.disabled_skills") {
			return MutationClassification{
				Behavior: MutationBehaviorAppliedNow,
				Applied:  true,
			}, nil
		}
		if strings.HasPrefix(trimmed, "skills.") {
			return restartRequiredClassification(), nil
		}
	case SectionHooksExtensions:
		if strings.HasPrefix(trimmed, "extensions.") || strings.HasPrefix(trimmed, "hooks.") {
			return restartRequiredClassification(), nil
		}
	}

	switch {
	case strings.HasPrefix(trimmed, "providers."):
		return restartRequiredClassification(), nil
	case strings.HasPrefix(trimmed, "mcp-servers."):
		return restartRequiredClassification(), nil
	case strings.HasPrefix(trimmed, "sandboxes."):
		return restartRequiredClassification(), nil
	case strings.HasPrefix(trimmed, "hooks."):
		return restartRequiredClassification(), nil
	default:
		return MutationClassification{}, fmt.Errorf(
			"settings: unsupported mutation field %q for section %q",
			trimmed,
			section,
		)
	}
}

func restartRequiredClassification() MutationClassification {
	return MutationClassification{
		Behavior:        MutationBehaviorRestartRequired,
		RestartRequired: true,
		RestartScope:    "daemon",
	}
}
