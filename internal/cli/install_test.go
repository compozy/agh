package cli

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestInstallCommandWritesBootstrapConfigAndAgent(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}

	deps.runInstallWizard = func(_ context.Context, input installWizardInput) (installWizardSelection, error) {
		if len(input.Providers) == 0 {
			t.Fatal("install wizard input providers = empty, want built-in providers")
		}
		return installWizardSelection{
			Provider: "claude",
			Model:    "claude-sonnet-4-20250514",
		}, nil
	}

	stdout, _, err := executeRootCommand(t, deps, "install", "-o", "json")
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
}
