//go:build integration

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEditConfigOverlayGlobalWritePreservesStructureOnDisk(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	target, err := ResolveConfigWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveConfigWriteTarget() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
# global structure
[defaults]
agent = "legacy"

[network]
enabled = true
`)

	cfg, err := EditConfigOverlay(homePaths, "", target, func(editor *OverlayEditor) error {
		return editor.SetValue([]string{"defaults", "agent"}, "general")
	})
	if err != nil {
		t.Fatalf("EditConfigOverlay() error = %v", err)
	}
	if got, want := cfg.Defaults.Agent, "general"; got != want {
		t.Fatalf("Defaults.Agent = %q, want %q", got, want)
	}

	payload, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	text := string(payload)
	for _, want := range []string{
		"# global structure",
		"[network]",
		"enabled = true",
		`agent = "general"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config contents missing %q\n%s", want, text)
		}
	}
}

func TestEditConfigOverlayWorkspaceWriteLeavesGlobalConfigUntouched(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	target, err := ResolveConfigWriteTarget(homePaths, workspaceRoot, WriteScopeWorkspace)
	if err != nil {
		t.Fatalf("ResolveConfigWriteTarget() error = %v", err)
	}

	writeFile(t, homePaths.ConfigFile, `
[defaults]
agent = "global"
`)

	cfg, err := EditConfigOverlay(homePaths, workspaceRoot, target, func(editor *OverlayEditor) error {
		return editor.SetValue([]string{"defaults", "agent"}, "workspace")
	})
	if err != nil {
		t.Fatalf("EditConfigOverlay() error = %v", err)
	}
	if got, want := cfg.Defaults.Agent, "workspace"; got != want {
		t.Fatalf("Load merged Defaults.Agent = %q, want %q", got, want)
	}

	globalPayload, err := os.ReadFile(homePaths.ConfigFile)
	if err != nil {
		t.Fatalf("ReadFile(global config) error = %v", err)
	}
	if !strings.Contains(string(globalPayload), `agent = "global"`) {
		t.Fatalf("global config was modified unexpectedly\n%s", globalPayload)
	}

	workspacePayload, err := os.ReadFile(filepath.Join(workspaceRoot, DirName, ConfigName))
	if err != nil {
		t.Fatalf("ReadFile(workspace config) error = %v", err)
	}
	if !strings.Contains(string(workspacePayload), `agent = "workspace"`) {
		t.Fatalf("workspace config missing updated agent\n%s", workspacePayload)
	}

	globalOnly, err := LoadForHome(homePaths)
	if err != nil {
		t.Fatalf("LoadForHome(global) error = %v", err)
	}
	if got, want := globalOnly.Defaults.Agent, "global"; got != want {
		t.Fatalf("global-only Defaults.Agent = %q, want %q", got, want)
	}
}

func TestPutMCPSidecarServerWritesAndPreservesUnaffectedEntries(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	target, err := ResolveMCPSidecarWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveMCPSidecarWriteTarget() error = %v", err)
	}

	writeFile(t, target.path, `{
  "version": 1,
  "mcpServers": {
    "alpha": { "command": "alpha" },
    "beta": { "command": "beta" }
  }
}`)

	cfg, err := PutMCPSidecarServer(homePaths, "", target, MCPServer{
		Name:    "alpha",
		Command: "updated-alpha",
		Args:    []string{"--flag"},
	})
	if err != nil {
		t.Fatalf("PutMCPSidecarServer() error = %v", err)
	}
	if got, want := len(cfg.MCPServers), 2; got != want {
		t.Fatalf("len(Config.MCPServers) = %d, want %d", got, want)
	}

	payload, err := os.ReadFile(target.path)
	if err != nil {
		t.Fatalf("ReadFile(mcp.json) error = %v", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatalf("json.Unmarshal(root) error = %v", err)
	}
	if _, ok := root["version"]; !ok {
		t.Fatalf("root keys = %v, want preserved version key", root)
	}

	var servers map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcpServers"], &servers); err != nil {
		t.Fatalf("json.Unmarshal(mcpServers) error = %v", err)
	}
	if got, want := servers["alpha"].Command, "updated-alpha"; got != want {
		t.Fatalf("servers[alpha].Command = %q, want %q", got, want)
	}
	if got, want := servers["beta"].Command, "beta"; got != want {
		t.Fatalf("servers[beta].Command = %q, want %q", got, want)
	}
}
