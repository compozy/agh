package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const legacyDreamAgentName = "claude"

// LoadGlobalConfig loads only the user-global AGH config from the resolved home.
func LoadGlobalConfig(homePaths HomePaths) (Config, error) {
	return loadWithHome(homePaths, "", false)
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
	selectedProvider := strings.TrimSpace(provider)
	if selectedProvider == "" {
		return Config{}, errors.New("bootstrap provider is required")
	}
	selectedModel := strings.TrimSpace(model)
	if selectedModel == "" {
		return Config{}, errors.New("bootstrap model is required")
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		return Config{}, err
	}

	current, err := LoadGlobalConfig(homePaths)
	if err != nil {
		return Config{}, err
	}

	overlay, err := loadConfigOverlayFile(homePaths.ConfigFile)
	if err != nil {
		return Config{}, fmt.Errorf("load global config overlay: %w", err)
	}

	overlay.Defaults.Agent = stringPtr(DefaultAgentName)
	overlay.Defaults.Provider = stringPtr(selectedProvider)
	overlay.Permissions.Mode = permissionModePtr(PermissionModeApproveAll)
	if strings.TrimSpace(current.Memory.Dream.Agent) == "" ||
		strings.TrimSpace(current.Memory.Dream.Agent) == legacyDreamAgentName {
		overlay.Memory.Dream.Agent = stringPtr(DefaultAgentName)
	}
	if overlay.Providers == nil {
		overlay.Providers = make(map[string]providerOverlay)
	}
	providerOverlay := overlay.Providers[selectedProvider]
	providerOverlay.DefaultModel = stringPtr(selectedModel)
	overlay.Providers[selectedProvider] = providerOverlay

	finalCfg := DefaultWithHome(homePaths)
	if err := overlay.Apply(&finalCfg); err != nil {
		return Config{}, fmt.Errorf("apply bootstrap config overlay: %w", err)
	}
	if err := applyConfigMCPSidecarFile(globalMCPJSONFile(homePaths), &finalCfg); err != nil {
		return Config{}, fmt.Errorf("load global MCP JSON: %w", err)
	}
	if err := normalizeConfigPaths(&finalCfg); err != nil {
		return Config{}, err
	}
	if err := finalCfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate bootstrap config: %w", err)
	}

	if err := writeConfigOverlayFile(homePaths.ConfigFile, &overlay); err != nil {
		return Config{}, err
	}
	return finalCfg, nil
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

func writeConfigOverlayFile(path string, overlay *configOverlay) error {
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(overlay); err != nil {
		return fmt.Errorf("encode config file %q: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory %q: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write config file %q: %w", path, err)
	}
	return nil
}

func stringPtr(value string) *string {
	return &value
}

func permissionModePtr(value PermissionMode) *PermissionMode {
	return &value
}
