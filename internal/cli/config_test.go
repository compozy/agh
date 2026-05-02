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
	sandboxOut, _, err := executeRootCommand(t, deps, "config", "set", "defaults.sandbox", "local", "-o", "json")
	if err != nil {
		t.Fatalf("config set defaults.sandbox error = %v", err)
	}
	var sandboxSetRecord configSetRecord
	if err := json.Unmarshal([]byte(sandboxOut), &sandboxSetRecord); err != nil {
		t.Fatalf("json.Unmarshal(config set defaults.sandbox) error = %v", err)
	}
	if sandboxSetRecord.Path != "defaults.sandbox" || sandboxSetRecord.Value != "local" {
		t.Fatalf("config set sandbox record = %#v, want defaults.sandbox=local", sandboxSetRecord)
	}

	cfg, err := aghconfig.LoadGlobalConfig(homePaths)
	if err != nil {
		t.Fatalf("LoadGlobalConfig() error = %v", err)
	}
	if cfg.Defaults.Provider != "claude" {
		t.Fatalf("Defaults.Provider = %q, want claude", cfg.Defaults.Provider)
	}
	if cfg.Defaults.Sandbox != "local" {
		t.Fatalf("Defaults.Sandbox = %q, want local", cfg.Defaults.Sandbox)
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

func TestConfigSetRejectsLegacyEnvironmentMutationPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{name: "Should reject legacy defaults environment", path: "defaults.environment"},
		{name: "Should reject legacy environment profile", path: "environments.dev.backend"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(t, &stubClient{})
			_, _, err := executeRootCommand(t, deps, "config", "set", tt.path, "local")
			if err == nil {
				t.Fatalf("config set %s error = nil, want unsupported path", tt.path)
			}
			if !strings.Contains(err.Error(), "is not supported by config set") {
				t.Fatalf("config set %s error = %v, want unsupported path", tt.path, err)
			}
		})
	}
}

func TestConfigValidateRepairEnvRepairsWorkspaceDotEnvWithoutLeakingValues(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	workspaceRoot := t.TempDir()
	dotenvPath := filepath.Join(workspaceRoot, ".env")
	before := "AGH_TASK09_API_KEY=very-secret\u200b-token OTHER=value\n"
	if err := os.WriteFile(dotenvPath, []byte(before), 0o600); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}

	stdout, _, err := executeRootCommand(
		t,
		deps,
		"config",
		"validate",
		"--workspace",
		workspaceRoot,
		"--repair-env",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("config validate --repair-env error = %v", err)
	}
	if strings.Contains(stdout, "very-secret") {
		t.Fatalf("config validate --repair-env leaked .env value:\n%s", stdout)
	}

	var record configValidateRecord
	if err := json.Unmarshal([]byte(stdout), &record); err != nil {
		t.Fatalf("json.Unmarshal(config validate --repair-env) error = %v", err)
	}
	if record.DotEnv == nil {
		t.Fatal("config validate --repair-env DotEnv = nil, want repair report")
	}
	if record.DotEnv.Status != aghconfig.DotEnvStatusRepaired || !record.DotEnv.Repaired {
		t.Fatalf("DotEnv report = %#v, want repaired", record.DotEnv)
	}
	if len(record.DotEnv.Diagnostics) != 2 {
		t.Fatalf("DotEnv diagnostics = %#v, want multi-key and sanitizer diagnostics", record.DotEnv.Diagnostics)
	}

	after, readErr := os.ReadFile(dotenvPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(.env after repair) error = %v", readErr)
	}
	for _, want := range []string{
		"AGH_TASK09_API_KEY=very-secret-token",
		"OTHER=value",
	} {
		if !strings.Contains(string(after), want) {
			t.Fatalf("repaired .env missing %q:\n%s", want, string(after))
		}
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

func TestConfigOutputRedactsMCPAndSandboxSecrets(t *testing.T) {
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
secret_env = { MCP_TOKEN = "env:MCP_TOKEN" }

[sandboxes.dev]
backend = "local"

	[sandboxes.dev.secret_env]
	API_TOKEN = "vault:sandbox/dev/api-token"
`)

	listOut, _, err := executeRootCommand(t, deps, "config", "list", "-o", "json")
	if err != nil {
		t.Fatalf("config list error = %v", err)
	}
	if strings.Contains(listOut, "env:MCP_TOKEN") ||
		strings.Contains(listOut, "vault:sandbox/dev/api-token") {
		t.Fatalf("config list leaked secret values:\n%s", listOut)
	}
	if !strings.Contains(listOut, aghconfig.RedactedValue()) {
		t.Fatalf("config list = %s, want redacted placeholder", listOut)
	}

	getOut, _, err := executeRootCommand(t, deps, "config", "get", "mcp_servers[0].secret_env.MCP_TOKEN", "-o", "json")
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

func TestConfigRenderingAndMutationHelpers(t *testing.T) {
	t.Parallel()

	entries := []configEntry{
		{Path: "defaults.provider", Value: "claude"},
		{Path: "http.port", Value: int64(4141)},
		{Path: "telemetry.enabled", Value: true},
		{Path: "mcp_servers[0].env.API_TOKEN", Value: aghconfig.RedactedValue(), Redacted: true},
		{Path: "providers.claude.models", Value: []string{"sonnet", "opus"}},
	}

	t.Run("Should render show bundle outputs", func(t *testing.T) {
		t.Parallel()

		showBundle := configShowBundle(configShowRecord{
			Scope:    "global",
			Redacted: true,
			Config:   map[string]any{"defaults": map[string]any{"provider": "claude"}},
		}, entries)
		showHuman, err := showBundle.human()
		if err != nil {
			t.Fatalf("configShowBundle.human() error = %v", err)
		}
		if !strings.Contains(showHuman, "defaults.provider") || !strings.Contains(showHuman, "true") {
			t.Fatalf("config show human = %q, want rows", showHuman)
		}
		showToon, err := showBundle.toon()
		if err != nil {
			t.Fatalf("configShowBundle.toon() error = %v", err)
		}
		if !strings.Contains(showToon, "config[5]") {
			t.Fatalf("config show toon = %q, want toon rows", showToon)
		}
	})

	t.Run("Should render list bundle outputs", func(t *testing.T) {
		t.Parallel()

		listBundle := configListBundle(configListRecord{Scope: "global", Entries: entries})
		listHuman, err := listBundle.human()
		if err != nil {
			t.Fatalf("configListBundle.human() error = %v", err)
		}
		if !strings.Contains(listHuman, "mcp_servers[0].env.API_TOKEN") {
			t.Fatalf("config list human = %q, want redacted entry", listHuman)
		}
		listToon, err := listBundle.toon()
		if err != nil {
			t.Fatalf("configListBundle.toon() error = %v", err)
		}
		if !strings.Contains(listToon, "config[5]") {
			t.Fatalf("config list toon = %q, want toon rows", listToon)
		}
	})

	t.Run("Should render path bundle outputs", func(t *testing.T) {
		t.Parallel()

		pathBundle := configPathBundle(configPathRecord{
			HomeDir:              "/home/agh",
			GlobalConfig:         "/home/agh/config.toml",
			GlobalMCPJSON:        "/home/agh/mcp.json",
			Scope:                "workspace",
			WorkspaceRoot:        "/workspace/project",
			WorkspaceConfig:      "/workspace/project/.agh/config.toml",
			WorkspaceMCPJSON:     "/workspace/project/.agh/mcp.json",
			Managed:              true,
			Manager:              "homebrew",
			SelectedConfigTarget: "/workspace/project/.agh/config.toml",
		})
		pathHuman, err := pathBundle.human()
		if err != nil {
			t.Fatalf("configPathBundle.human() error = %v", err)
		}
		if !strings.Contains(pathHuman, "Workspace Config") || !strings.Contains(pathHuman, "homebrew") {
			t.Fatalf("config path human = %q, want workspace details", pathHuman)
		}
		pathToon, err := pathBundle.toon()
		if err != nil {
			t.Fatalf("configPathBundle.toon() error = %v", err)
		}
		if !strings.Contains(pathToon, "config_paths") || !strings.Contains(pathToon, "/workspace/project") {
			t.Fatalf("config path toon = %q, want workspace fields", pathToon)
		}
	})

	t.Run("Should classify mutation paths", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name        string
			path        string
			wantKind    configSetValueKind
			wantRedact  bool
			wantAllowed bool
		}{
			{
				name:        "Should allow provider command",
				path:        "providers.claude.command",
				wantKind:    configSetString,
				wantAllowed: true,
			},
			{
				name:        "Should redact sandbox env values",
				path:        "sandboxes.dev.env.API_TOKEN",
				wantKind:    configSetString,
				wantRedact:  true,
				wantAllowed: true,
			},
			{
				name:        "Should allow sandbox public ingress",
				path:        "sandboxes.dev.network.allow_public_ingress",
				wantKind:    configSetBool,
				wantAllowed: true,
			},
			{
				name:        "Should allow sandbox network allow list",
				path:        "sandboxes.dev.network.allow_list",
				wantKind:    configSetStringSlice,
				wantAllowed: true,
			},
			{
				name:        "Should allow sandbox Daytona image",
				path:        "sandboxes.dev.daytona.image",
				wantKind:    configSetString,
				wantAllowed: true,
			},
			{name: "Should reject unknown sandbox values", path: "sandboxes.dev.unknown.value", wantAllowed: false},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				_, kind, redacted, err := configMutationPath(tc.path)
				allowed := err == nil
				if allowed != tc.wantAllowed || (allowed && (kind != tc.wantKind || redacted != tc.wantRedact)) {
					t.Fatalf(
						"classifyConfigMutationPath(%q) = (%v, %v, %v), want (%v, %v, %v)",
						tc.path,
						kind,
						redacted,
						allowed,
						tc.wantKind,
						tc.wantRedact,
						tc.wantAllowed,
					)
				}
			})
		}
	})

	t.Run("Should parse string slice values", func(t *testing.T) {
		t.Parallel()

		values, err := parseStringSliceValue(`["alpha","beta"]`)
		if err != nil {
			t.Fatalf("parseStringSliceValue(json) error = %v", err)
		}
		if strings.Join(values, ",") != "alpha,beta" {
			t.Fatalf("parseStringSliceValue(json) = %#v, want alpha,beta", values)
		}
		values, err = parseStringSliceValue("alpha, beta, ,gamma")
		if err != nil {
			t.Fatalf("parseStringSliceValue(csv) error = %v", err)
		}
		if strings.Join(values, ",") != "alpha,beta,gamma" {
			t.Fatalf("parseStringSliceValue(csv) = %#v, want alpha,beta,gamma", values)
		}
		values, err = parseStringSliceValue(" ")
		if err != nil {
			t.Fatalf("parseStringSliceValue(empty) error = %v", err)
		}
		if len(values) != 0 {
			t.Fatalf("parseStringSliceValue(empty) = %#v, want empty", values)
		}
		if _, err := parseStringSliceValue(`["ok",1]`); err == nil {
			t.Fatal("parseStringSliceValue(invalid json) error = nil, want error")
		} else if !strings.Contains(err.Error(), "string") {
			t.Fatalf("parseStringSliceValue(invalid json) error = %v, want element type detail", err)
		}
	})
}

func TestConfigSetRedactsSensitiveMutationOutputAndManagedModeBlocksMutation(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	if _, _, err := executeRootCommand(t, deps, "config", "set", "sandboxes.dev.backend", "local"); err != nil {
		t.Fatalf("config set sandbox backend error = %v", err)
	}
	out, _, err := executeRootCommand(
		t,
		deps,
		"config",
		"set",
		"sandboxes.dev.secret_env.API_TOKEN",
		"vault:sandbox/dev/api-token",
		"-o",
		"json",
	)
	if err != nil {
		t.Fatalf("config set sandbox env error = %v", err)
	}
	if strings.Contains(out, "vault:sandbox/dev/api-token") {
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
