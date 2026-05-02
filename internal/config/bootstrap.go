package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const legacyDreamAgentName = "claude"

// LoadGlobalConfig loads only the user-global AGH config from the resolved home.
func LoadGlobalConfig(homePaths HomePaths) (Config, error) {
	return loadWithHome(homePaths, "", false, processEnvLookup)
}

// ResolveAgentName resolves an explicit session agent name or falls back to config defaults.
func ResolveAgentName(name string, defaults DefaultsConfig) (string, error) {
	if resolved := strings.TrimSpace(name); resolved != "" {
		return resolved, nil
	}
	if resolved := strings.TrimSpace(defaults.Agent); resolved != "" {
		return resolved, nil
	}
	return "", errors.New("agent name is required; run `agh install` or set defaults.agent")
}

// SaveBootstrapConfig writes the global bootstrap config managed by `agh install`.
func SaveBootstrapConfig(homePaths HomePaths, provider string, model string) (Config, error) {
	selectedProvider := CanonicalProviderName(provider)
	if selectedProvider == "" {
		return Config{}, errors.New("bootstrap provider is required")
	}
	selectedModel := strings.TrimSpace(model)
	if err := EnsureHomeLayout(homePaths); err != nil {
		return Config{}, err
	}

	current, err := LoadGlobalConfig(homePaths)
	if err != nil {
		return Config{}, err
	}
	resolvedProvider, err := current.ResolveProvider(selectedProvider)
	if err != nil {
		return Config{}, fmt.Errorf("resolve bootstrap provider %q: %w", selectedProvider, err)
	}
	if selectedModel == "" && resolvedProvider.RequiresRuntimeModel() {
		return Config{}, fmt.Errorf("bootstrap model is required for provider %q", selectedProvider)
	}

	target, err := ResolveConfigWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		return Config{}, err
	}

	dreamAgent := ""
	if strings.TrimSpace(current.Memory.Dream.Agent) == "" ||
		strings.TrimSpace(current.Memory.Dream.Agent) == legacyDreamAgentName {
		dreamAgent = DefaultAgentName
	}

	return EditConfigOverlay(homePaths, "", target, func(editor *OverlayEditor) error {
		if err := editor.SetValue([]string{"defaults", "agent"}, DefaultAgentName); err != nil {
			return err
		}
		if err := editor.SetValue([]string{"defaults", "provider"}, selectedProvider); err != nil {
			return err
		}
		if err := editor.SetValue([]string{"permissions", "mode"}, string(PermissionModeApproveAll)); err != nil {
			return err
		}
		if dreamAgent != "" {
			if err := editor.SetValue([]string{"memory", "dream", "agent"}, dreamAgent); err != nil {
				return err
			}
		}
		if selectedModel == "" {
			return nil
		}
		return editor.SetValue([]string{"providers", selectedProvider, "default_model"}, selectedModel)
	})
}

// EnsureBootstrapAgent creates the managed default agent definition if it does not already exist.
func EnsureBootstrapAgent(homePaths HomePaths) (string, bool, error) {
	if err := EnsureHomeLayout(homePaths); err != nil {
		return "", false, err
	}

	path := filepath.Join(homePaths.AgentsDir, DefaultAgentName, agentDefName)
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("stat bootstrap agent file %q: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, fmt.Errorf("create bootstrap agent directory %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(bootstrapAgentContents()), 0o600); err != nil {
		return "", false, fmt.Errorf("write bootstrap agent file %q: %w", path, err)
	}
	return path, true, nil
}

func bootstrapAgentContents() string {
	return strings.Join([]string{
		"---",
		"name: " + DefaultAgentName,
		"---",
		"",
		"You are AGH's default general-purpose agent.",
		"",
		"Operate autonomously, complete tasks end-to-end, and follow the active workspace instructions.",
		"Provider and model are resolved from the user's AGH configuration unless this agent is overridden.",
	}, "\n")
}
