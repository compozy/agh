package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	bootstrapDefaultsKey = "defaults"
)

const (
	toolSurfaceModelsKey = "models"
)

const (
	bootstrapDefaultKey     = "default"
	bootstrapPermissionsKey = "permissions"
	bootstrapProviderKey    = "provider"
)

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
	currentDreamAgent := strings.TrimSpace(current.Memory.Dream.Agent)
	if currentDreamAgent == "" ||
		currentDreamAgent == providerClaudeKey ||
		currentDreamAgent == DefaultAgentName {
		dreamAgent = DefaultMemoryDreamAgentName
	}

	return EditConfigOverlay(homePaths, "", target, func(editor *OverlayEditor) error {
		if err := editor.SetValue(
			[]string{bootstrapDefaultsKey, string(AgentResourceKind)},
			DefaultAgentName,
		); err != nil {
			return err
		}
		if err := editor.SetValue([]string{bootstrapDefaultsKey, bootstrapProviderKey}, selectedProvider); err != nil {
			return err
		}
		if err := editor.SetValue(
			[]string{bootstrapPermissionsKey, "mode"},
			string(PermissionModeApproveAll),
		); err != nil {
			return err
		}
		if dreamAgent != "" {
			if err := editor.SetValue(
				[]string{MemoryDirName, "dream", string(AgentResourceKind)},
				dreamAgent,
			); err != nil {
				return err
			}
		}
		if selectedModel == "" {
			return nil
		}
		return editor.SetValue(
			[]string{providersConfigKey, selectedProvider, toolSurfaceModelsKey, bootstrapDefaultKey},
			selectedModel,
		)
	})
}

// EnsureBootstrapAgent creates the managed default agent definition if it does not already exist.
func EnsureBootstrapAgent(homePaths HomePaths) (string, bool, error) {
	return ensureManagedAgent(homePaths, DefaultAgentName, bootstrapAgentContents())
}

// EnsureOnboardingAgent creates the managed first-run onboarding agent if it does not already exist.
// This agent interviews the operator during the web onboarding wizard and provisions channels and
// agents through its bounded coordination + workspace toolsets.
func EnsureOnboardingAgent(homePaths HomePaths) (string, bool, error) {
	contents, err := onboardingAgentContents()
	if err != nil {
		return "", false, err
	}
	return ensureManagedAgent(homePaths, OnboardingAgentName, contents)
}

func ensureManagedAgent(homePaths HomePaths, agentName string, contents string) (string, bool, error) {
	if err := EnsureHomeLayout(homePaths); err != nil {
		return "", false, err
	}

	path := filepath.Join(homePaths.AgentsDir, agentName, agentDefName)
	if _, err := os.Stat(path); err == nil {
		return path, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("stat managed agent file %q: %w", path, err)
	}

	agentDir := filepath.Dir(path)
	if err := os.MkdirAll(agentDir, privateDirMode); err != nil {
		return "", false, fmt.Errorf("create managed agent directory %q: %w", filepath.Dir(path), err)
	}
	if err := os.Chmod(agentDir, privateDirMode); err != nil {
		return "", false, fmt.Errorf("secure managed agent directory %q: %w", agentDir, err)
	}
	if err := os.WriteFile(path, []byte(contents), privateFileMode); err != nil {
		return "", false, fmt.Errorf("write managed agent file %q: %w", path, err)
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

func onboardingAgentContents() (string, error) {
	contents, _, err := RenderAgentDefinition(AgentDefinitionDraft{
		Name:        OnboardingAgentName,
		Tools:       onboardingAgentTools(),
		Permissions: string(PermissionModeApproveAll),
		Prompt:      onboardingAgentPrompt,
	})
	if err != nil {
		return "", fmt.Errorf("render onboarding agent: %w", err)
	}
	return string(contents), nil
}

const (
	// OnboardingAgentName is the managed first-run onboarding agent definition name.
	OnboardingAgentName = "onboarding"
)

func onboardingAgentTools() []string {
	return []string{
		"agh__workspace_list",
		"agh__workspace_describe",
		"agh__network_channels",
		"agh__network_channel_create",
		"agh__agent_create",
	}
}

const onboardingAgentPrompt = "You are AGH's onboarding agent. You run inside the first-run setup " +
	"wizard and help the operator finish configuring their workspace through a short, friendly " +
	"conversation. The default " +
	"model and workspaces are already configured before you start.\n\n" +
	"Start by calling `agh__workspace_list` and `agh__workspace_describe` so you know which workspace " +
	"you are configuring. Your job is to set up two things, one at a time:\n\n" +
	"1. Channels — places where the operator and their agents coordinate. Briefly explain what channels " +
	"are, suggest a few sensible defaults (for example general, engineering, design), and ask which ones " +
	"to create. Call `agh__network_channels` before creating channels so you do not duplicate an " +
	"existing one. For each channel the operator confirms, call `agh__network_channel_create` with " +
	"the workspace_id, a lowercase channel name (a-z, 0-9, dash, underscore), and a one-line purpose.\n\n" +
	"2. Agents — teammates that work in the channels. Ask which agents the operator wants. For each one " +
	"they confirm, call `agh__agent_create` with scope \"workspace\", the workspace, a name, a provider, " +
	"and a clear prompt describing the agent's role.\n\n" +
	"Rules:\n" +
	"- Do real work: actually call the tools to create what the operator confirms. Never claim something " +
	"was created without calling the tool.\n" +
	"- Confirm names and purposes before creating. Keep messages short.\n" +
	"- The operator can stop at any time and finish setup without creating anything; never pressure them.\n" +
	"- When you are done, briefly summarize what you created and tell them they can finish setup."
