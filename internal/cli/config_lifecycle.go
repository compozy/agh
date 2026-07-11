package cli

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/api/contract"
	settingspkg "github.com/compozy/agh/internal/settings"
)

type configMutationLifecycle struct {
	Lifecycle       string
	Applied         bool
	RestartRequired bool
	RestartScope    string
	NextAction      string
}

func classifyConfigSetLifecycle(path []string) (configMutationLifecycle, error) {
	field := strings.Join(path, ".")
	section := settingsSectionForConfigMutation(path)
	if section == "" {
		return restartRequiredConfigLifecycle(), nil
	}
	classification, err := settingspkg.ClassifyMutation(settingspkg.MutationDescriptor{
		Section:       section,
		ChangedFields: []string{field},
	})
	if err != nil {
		return configMutationLifecycle{}, fmt.Errorf(
			"cli: classify lifecycle for config path %q: %w",
			field,
			err,
		)
	}
	return configLifecycleFromSettings(classification), nil
}

func settingsSectionForConfigMutation(path []string) settingspkg.SectionName {
	if len(path) == 0 {
		return ""
	}
	switch path[0] {
	case configDaemonKey, "defaults", "http", "limits", configPermissionsKey, sessionSessionKey:
		return settingspkg.SectionGeneral
	case configMemoryKey:
		return settingspkg.SectionMemory
	case configSkillsKey:
		return settingspkg.SectionSkills
	case "automation":
		return settingspkg.SectionAutomation
	case configNetworkKey:
		return settingspkg.SectionNetwork
	case "log", "observability":
		return settingspkg.SectionObservability
	case "extensions", "hooks":
		return settingspkg.SectionHooksExtensions
	case configProvidersKey:
		return settingspkg.SectionName(settingspkg.CollectionProviders)
	case "mcp-servers":
		return settingspkg.SectionName(settingspkg.CollectionMCPServers)
	case configPathSandboxes:
		return settingspkg.SectionName(settingspkg.CollectionSandboxes)
	default:
		return ""
	}
}

func configLifecycleFromSettings(classification settingspkg.MutationClassification) configMutationLifecycle {
	nextAction := contract.SettingsApplyNextActionNone
	if classification.RestartRequired {
		nextAction = contract.SettingsApplyNextActionRestartDaemon
	}
	return configMutationLifecycle{
		Lifecycle:       string(classification.Lifecycle),
		Applied:         classification.Applied,
		RestartRequired: classification.RestartRequired,
		RestartScope:    classification.RestartScope,
		NextAction:      string(nextAction),
	}
}

func restartRequiredConfigLifecycle() configMutationLifecycle {
	return configMutationLifecycle{
		Lifecycle:       string(contract.SettingsApplyLifecycleRestartRequired),
		RestartRequired: true,
		RestartScope:    configDaemonKey,
		NextAction:      string(contract.SettingsApplyNextActionRestartDaemon),
	}
}
