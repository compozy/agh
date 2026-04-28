package cli

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestInstallCommandWritesBootstrapConfigAndAgent(t *testing.T) {
	t.Parallel()

	t.Run("Should use explicit provider and model flags for machine output", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		homePaths, err := deps.resolveHome()
		if err != nil {
			t.Fatalf("resolveHome() error = %v", err)
		}

		deps.runInstallWizard = func(context.Context, installWizardInput) (installWizardSelection, error) {
			t.Fatal("install wizard should not run when provider/model flags are explicit")
			return installWizardSelection{}, nil
		}

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"install",
			"--provider",
			"claude",
			"--model",
			"claude-sonnet-4-20250514",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("executeRootCommand(install) error = %v", err)
		}

		var decoded installRecord
		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(install) error = %v", err)
		}
		if decoded.AgentName != aghconfig.DefaultAgentName {
			t.Fatalf("decoded.AgentName = %q, want %q", decoded.AgentName, aghconfig.DefaultAgentName)
		}
		if decoded.Provider != "claude" {
			t.Fatalf("decoded.Provider = %q, want %q", decoded.Provider, "claude")
		}
		if decoded.Permissions != string(aghconfig.PermissionModeApproveAll) {
			t.Fatalf("decoded.Permissions = %q, want %q", decoded.Permissions, aghconfig.PermissionModeApproveAll)
		}

		cfg, err := aghconfig.LoadGlobalConfig(homePaths)
		if err != nil {
			t.Fatalf("LoadGlobalConfig() error = %v", err)
		}
		if cfg.Defaults.Agent != aghconfig.DefaultAgentName {
			t.Fatalf("cfg.Defaults.Agent = %q, want %q", cfg.Defaults.Agent, aghconfig.DefaultAgentName)
		}
		if cfg.Defaults.Provider != "claude" {
			t.Fatalf("cfg.Defaults.Provider = %q, want %q", cfg.Defaults.Provider, "claude")
		}

		agentContents, err := os.ReadFile(decoded.AgentFile)
		if err != nil {
			t.Fatalf("ReadFile(agent) error = %v", err)
		}
		if !strings.Contains(string(agentContents), "name: "+aghconfig.DefaultAgentName) {
			t.Fatalf("agent contents = %q, want bootstrap agent name", string(agentContents))
		}
	})
}

func TestInstallCommandMachineOutput(t *testing.T) {
	t.Parallel()

	t.Run("Should use effective defaults without opening the wizard", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		homePaths, err := deps.resolveHome()
		if err != nil {
			t.Fatalf("resolveHome() error = %v", err)
		}
		cfg, err := aghconfig.LoadGlobalConfig(homePaths)
		if err != nil {
			t.Fatalf("LoadGlobalConfig() error = %v", err)
		}
		input := buildInstallWizardInput(&cfg)
		if input.SelectedProvider == "" {
			t.Fatal("install input selected provider = empty, want default provider")
		}
		wantModel := strings.TrimSpace(input.SuggestedModels[input.SelectedProvider])
		if wantModel == "" {
			t.Fatalf("SuggestedModels[%q] = empty, want default model", input.SelectedProvider)
		}

		deps.runInstallWizard = func(context.Context, installWizardInput) (installWizardSelection, error) {
			t.Fatal("install wizard should not run for machine output")
			return installWizardSelection{}, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "install", "-o", "json")
		if err != nil {
			t.Fatalf("executeRootCommand(install -o json) error = %v", err)
		}

		var decoded installRecord
		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(install) error = %v", err)
		}
		if decoded.Provider != input.SelectedProvider {
			t.Fatalf("decoded.Provider = %q, want %q", decoded.Provider, input.SelectedProvider)
		}
		if decoded.Model != wantModel {
			t.Fatalf("decoded.Model = %q, want %q", decoded.Model, wantModel)
		}
	})
}

func TestInstallCommandHumanOutput(t *testing.T) {
	t.Parallel()

	t.Run("Should keep the interactive wizard for human output", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		deps.runInstallWizard = func(_ context.Context, input installWizardInput) (installWizardSelection, error) {
			if len(input.Providers) == 0 {
				t.Fatal("install wizard input providers = empty, want built-in providers")
			}
			return installWizardSelection{
				Provider: "claude",
				Model:    "claude-sonnet-4-20250514",
			}, nil
		}

		stdout, _, err := executeRootCommand(t, deps, "install")
		if err != nil {
			t.Fatalf("executeRootCommand(install) error = %v", err)
		}
		if !strings.Contains(stdout, "Provider:") || !strings.Contains(stdout, "claude") {
			t.Fatalf("install human stdout = %q, want selected provider", stdout)
		}
	})
}

func TestBuildInstallWizardInputAndBundleFormats(t *testing.T) {
	t.Parallel()

	t.Run("Should build wizard input and bundle formats", func(t *testing.T) {
		t.Parallel()

		cfg := aghconfig.DefaultWithHome(aghconfig.HomePaths{})
		cfg.Defaults.Provider = "codex"
		cfg.Providers["custom"] = aghconfig.ProviderConfig{DefaultModel: "custom-model"}

		input := buildInstallWizardInput(&cfg)
		if len(input.Providers) == 0 {
			t.Fatal("buildInstallWizardInput() providers = empty, want builtin/custom providers")
		}
		if input.SelectedProvider != "codex" {
			t.Fatalf("SelectedProvider = %q, want %q", input.SelectedProvider, "codex")
		}
		if input.SuggestedModels["custom"] != "custom-model" {
			t.Fatalf("SuggestedModels[custom] = %q, want %q", input.SuggestedModels["custom"], "custom-model")
		}

		record := installRecord{
			AgentName:    aghconfig.DefaultAgentName,
			Provider:     "codex",
			Model:        "gpt-5.4",
			Permissions:  string(aghconfig.PermissionModeApproveAll),
			ConfigFile:   "/tmp/config.toml",
			AgentFile:    "/tmp/AGENT.md",
			CreatedAgent: true,
		}

		human, err := installBundle(record).human()
		if err != nil {
			t.Fatalf("installBundle().human() error = %v", err)
		}
		if !strings.Contains(human, "Install") || !strings.Contains(human, "created bootstrap agent file") {
			t.Fatalf("install human output = %q, want install summary", human)
		}

		toon, err := installBundle(record).toon()
		if err != nil {
			t.Fatalf("installBundle().toon() error = %v", err)
		}
		if !strings.Contains(
			toon,
			"install{agent_name,provider,model,permissions,config_file,agent_file,created_agent,managed,manager}:",
		) {
			t.Fatalf("install toon output = %q, want TOON header", toon)
		}
	})
}

func TestInstallWizardModelTransitions(t *testing.T) {
	t.Parallel()

	t.Run("Should transition through provider model and confirmation steps", func(t *testing.T) {
		t.Parallel()

		model := newInstallWizardModel(installWizardInput{
			Providers:        []string{"claude", "codex"},
			SelectedProvider: "claude",
			SuggestedModels: map[string]string{
				"claude": "claude-sonnet",
				"codex":  "gpt-5.4",
			},
		})

		if model.Init() == nil {
			t.Fatal("Init() = nil, want blink command")
		}
		if !strings.Contains(model.View(), "Select the default provider") {
			t.Fatalf("provider view = %q, want provider prompt", model.View())
		}

		if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown}); cmd != nil {
			t.Fatalf("provider navigation cmd = %v, want nil", cmd)
		}
		if model.selected != 1 {
			t.Fatalf("selected = %d, want 1", model.selected)
		}

		if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter}); cmd == nil {
			t.Fatal("provider enter cmd = nil, want blink command")
		}
		if model.step != installWizardStepModel || model.provider != "codex" {
			t.Fatalf("provider step transition = %#v, want model step for codex", model)
		}
		if model.modelInput.Value() != "gpt-5.4" {
			t.Fatalf("modelInput.Value() = %q, want %q", model.modelInput.Value(), "gpt-5.4")
		}
		if !strings.Contains(model.View(), "Selected provider: codex") {
			t.Fatalf("model view = %q, want selected provider", model.View())
		}

		model.modelInput.SetValue("")
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if model.errText != "model is required" {
			t.Fatalf("errText = %q, want model is required", model.errText)
		}

		model.modelInput.SetValue("gpt-5.4")
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if model.step != installWizardStepConfirm {
			t.Fatalf("step = %v, want confirm", model.step)
		}
		if !strings.Contains(model.View(), "Review the bootstrap configuration.") {
			t.Fatalf("confirm view = %q, want review prompt", model.View())
		}

		if _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
			t.Fatal("confirm esc cmd = nil, want blink command")
		}
		if model.step != installWizardStepModel {
			t.Fatalf("step after esc = %v, want model", model.step)
		}

		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		model.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if !model.done {
			t.Fatal("done = false, want true after confirm enter")
		}
	})
}
