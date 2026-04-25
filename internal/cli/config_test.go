package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestConfigCommandsMutateValidateAndInspectTempHome(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}

	setOut, _, err := executeRootCommand(t, deps, "config", "set", "defaults.provider", "claude", "-o", "json")
	if err != nil {
		t.Fatalf("config set defaults.provider error = %v", err)
	}
	var setRecord configSetRecord
	if err := json.Unmarshal([]byte(setOut), &setRecord); err != nil {
		t.Fatalf("json.Unmarshal(config set) error = %v", err)
	}
	if setRecord.Path != "defaults.provider" || setRecord.Value != "claude" {
		t.Fatalf("config set record = %#v, want defaults.provider=claude", setRecord)
	}

	cfg, err := aghconfig.LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if cfg.Defaults.Provider != "claude" {
		t.Fatalf("Defaults.Provider = %q, want claude", cfg.Defaults.Provider)
	}

	getOut, _, err := executeRootCommand(t, deps, "config", "get", "defaults.provider", "-o", "json")
	if err != nil {
		t.Fatalf("config get defaults.provider error = %v", err)
	}
	var valueRecord configValueRecord
	if err := json.Unmarshal([]byte(getOut), &valueRecord); err != nil {
		t.Fatalf("json.Unmarshal(config get) error = %v", err)
	}
	if valueRecord.Value != "claude" || valueRecord.Redacted {
		t.Fatalf("config get record = %#v, want unredacted claude", valueRecord)
	}

	validateOut, _, err := executeRootCommand(t, deps, "config", "validate", "-o", "json")
	if err != nil {
		t.Fatalf("config validate error = %v", err)
	}
	var validateRecord configValidateRecord
	if err := json.Unmarshal([]byte(validateOut), &validateRecord); err != nil {
		t.Fatalf("json.Unmarshal(config validate) error = %v", err)
	}
	if validateRecord.Status != "valid" || validateRecord.ConfigFile != homePaths.ConfigFile {
		t.Fatalf("config validate record = %#v, want valid config file", validateRecord)
	}

	pathOut, _, err := executeRootCommand(t, deps, "config", "path", "-o", "json")
	if err != nil {
		t.Fatalf("config path error = %v", err)
	}
	var pathRecord configPathRecord
	if err := json.Unmarshal([]byte(pathOut), &pathRecord); err != nil {
		t.Fatalf("json.Unmarshal(config path) error = %v", err)
	}
	if pathRecord.GlobalConfig != homePaths.ConfigFile ||
		pathRecord.GlobalMCPJSON != filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName) {
		t.Fatalf("config path record = %#v, want resolved home paths", pathRecord)
	}
}

func TestConfigCommandsUseWorkspaceScopeAndValidateBeforeWriting(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	workspaceRoot := t.TempDir()
	deps.getwd = func() (string, error) {
		return workspaceRoot, nil
	}

	if _, _, err := executeRootCommand(
		t,
		deps,
		"config",
		"set",
		"network.default_channel",
		"builders",
		"--scope",
		"workspace",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("workspace config set error = %v", err)
	}

	workspaceConfig := filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.ConfigName)
	contents, err := os.ReadFile(workspaceConfig)
	if err != nil {
		t.Fatalf("ReadFile(workspace config) error = %v", err)
	}
	if !strings.Contains(string(contents), `default_channel = "builders"`) {
		t.Fatalf("workspace config = %s, want default_channel builders", string(contents))
	}

	before := string(contents)
	_, _, err = executeRootCommand(
		t,
		deps,
		"config",
		"set",
		"http.port",
		"70000",
		"--scope",
		"workspace",
	)
	if err == nil {
		t.Fatal("config set invalid http.port error = nil, want validation failure")
	}
	after, readErr := os.ReadFile(workspaceConfig)
	if readErr != nil {
		t.Fatalf("ReadFile(workspace config after invalid set) error = %v", readErr)
	}
	if string(after) != before {
		t.Fatalf("workspace config changed after invalid set\nbefore:\n%s\nafter:\n%s", before, string(after))
	}
}

func TestConfigOutputRedactsMCPAndEnvironmentSecrets(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := deps.resolveHome()
	if err != nil {
		t.Fatalf("resolveHome() error = %v", err)
	}
	writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "remote"
command = "remote-mcp"
env = { MCP_TOKEN = "raw-mcp-secret" }

[environments.dev]
backend = "local"

[environments.dev.env]
API_TOKEN = "raw-env-secret"
`)

	listOut, _, err := executeRootCommand(t, deps, "config", "list", "-o", "json")
	if err != nil {
		t.Fatalf("config list error = %v", err)
	}
	if strings.Contains(listOut, "raw-mcp-secret") || strings.Contains(listOut, "raw-env-secret") {
		t.Fatalf("config list leaked secret values:\n%s", listOut)
	}
	if !strings.Contains(listOut, aghconfig.RedactedValue()) {
		t.Fatalf("config list = %s, want redacted placeholder", listOut)
	}

	getOut, _, err := executeRootCommand(t, deps, "config", "get", "mcp_servers[0].env.MCP_TOKEN", "-o", "json")
	if err != nil {
		t.Fatalf("config get redacted MCP env error = %v", err)
	}
	var valueRecord configValueRecord
	if err := json.Unmarshal([]byte(getOut), &valueRecord); err != nil {
		t.Fatalf("json.Unmarshal(config get redacted) error = %v", err)
	}
	if valueRecord.Value != aghconfig.RedactedValue() || !valueRecord.Redacted {
		t.Fatalf("redacted config value = %#v, want placeholder/redacted", valueRecord)
	}
}

func TestConfigSetRedactsSensitiveMutationOutputAndManagedModeBlocksMutation(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	if _, _, err := executeRootCommand(t, deps, "config", "set", "environments.dev.backend", "local"); err != nil {
		t.Fatalf("config set environment backend error = %v", err)
	}
	out, _, err := executeRootCommand(
		t,
		deps,
		"config",
		"set",
		"environments.dev.env.API_TOKEN",
		"raw-secret",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("config set environment env error = %v", err)
	}
	if strings.Contains(out, "raw-secret") {
		t.Fatalf("config set leaked secret value:\n%s", out)
	}
	var setRecord configSetRecord
	if err := json.Unmarshal([]byte(out), &setRecord); err != nil {
		t.Fatalf("json.Unmarshal(secret config set) error = %v", err)
	}
	if setRecord.Value != aghconfig.RedactedValue() || !setRecord.Redacted {
		t.Fatalf("secret config set record = %#v, want redacted placeholder", setRecord)
	}

	managedDeps := newTestDeps(t, &stubClient{})
	managedDeps.getenv = func(key string) string {
		if key == managedEnvName {
			return "homebrew"
		}
		return ""
	}
	_, _, err = executeRootCommand(t, managedDeps, "config", "set", "defaults.provider", "claude")
	if err == nil {
		t.Fatal("managed config set error = nil, want managed mutation refusal")
	}
	if !strings.Contains(err.Error(), "managed by homebrew") {
		t.Fatalf("managed config set error = %v, want homebrew detail", err)
	}
}

func TestCompletionCommandEmitsShellCompletion(t *testing.T) {
	t.Parallel()

	out, _, err := executeRootCommand(t, newTestDeps(t, &stubClient{}), "completion", "bash")
	if err != nil {
		t.Fatalf("completion bash error = %v", err)
	}
	for _, want := range []string{"bash completion V2 for agh", "__start_agh"} {
		if !strings.Contains(out, want) {
			t.Fatalf("completion output missing %q\n%s", want, out[:min(len(out), 400)])
		}
	}
}
