package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseMCPServersJSONAcceptsBothKeyStyles(t *testing.T) {
	t.Parallel()

	servers, err := ParseMCPServersJSON([]byte(`{
  "mcpServers": {
    "alpha": {
      "command": "alpha-inline",
      "args": ["--a"]
    }
  },
  "mcp_servers": {
    "alpha": {
      "command": "alpha-sidecar"
    },
    "beta": {
      "command": "beta-command",
      "env": {
        "TOKEN": "value"
      }
    }
  }
}`), "fixture")
	if err != nil {
		t.Fatalf("ParseMCPServersJSON() error = %v", err)
	}

	if got, want := len(servers), 2; got != want {
		t.Fatalf("len(ParseMCPServersJSON()) = %d, want %d", got, want)
	}
	if got, want := servers[0].Name, "alpha"; got != want {
		t.Fatalf("servers[0].Name = %q, want %q", got, want)
	}
	if got, want := servers[0].Command, "alpha-sidecar"; got != want {
		t.Fatalf("servers[0].Command = %q, want %q", got, want)
	}
	if got := len(servers[0].Args); got != 0 {
		t.Fatalf("servers[0].Args = %#v, want sidecar whole-object replacement", servers[0].Args)
	}
	if got, want := servers[1].Env["TOKEN"], "value"; got != want {
		t.Fatalf("servers[1].Env[TOKEN] = %q, want %q", got, want)
	}
}

func TestParseMCPServersJSONRejectsInvalidEntries(t *testing.T) {
	t.Parallel()

	if _, err := ParseMCPServersJSON(
		[]byte(`{"mcpServers":{"broken":{"args":["--missing-command"]}}}`),
		"broken.json",
	); err == nil {
		t.Fatal("ParseMCPServersJSON() error = nil, want missing command failure")
	} else if !strings.Contains(err.Error(), `mcp.json "broken.json"[0].command is required`) {
		t.Fatalf("ParseMCPServersJSON() error = %q, want missing command validation context", err.Error())
	}
}

func TestParseMCPServersJSONRejectsDuplicateNormalizedNames(t *testing.T) {
	t.Parallel()

	_, err := ParseMCPServersJSON([]byte(`{
  "mcpServers": {
    " foo ": { "command": "alpha" },
    "foo": { "command": "beta" }
  }
}`), "duplicates.json")
	if err == nil {
		t.Fatal("ParseMCPServersJSON() error = nil, want duplicate normalized-name failure")
	}
	if !strings.Contains(err.Error(), `duplicate MCP server name "foo" after normalization`) {
		t.Fatalf("ParseMCPServersJSON() error = %q, want duplicate normalized-name context", err.Error())
	}
}

func TestParseMCPServersJSONRejectsTrailingJSON(t *testing.T) {
	t.Parallel()

	_, err := ParseMCPServersJSON([]byte(`{"mcpServers":{"alpha":{"command":"npx"}}}{"extra":true}`), "trailing.json")
	if err == nil {
		t.Fatal("ParseMCPServersJSON() error = nil, want trailing JSON failure")
	}
	if !strings.Contains(err.Error(), "unexpected trailing JSON value") {
		t.Fatalf("ParseMCPServersJSON() error = %q, want trailing JSON context", err.Error())
	}
}

func TestLoadMCPServersJSONFileMissingIsOptional(t *testing.T) {
	t.Parallel()

	servers, err := LoadMCPServersJSONFile(filepath.Join(t.TempDir(), "missing", MCPJSONName))
	if err != nil {
		t.Fatalf("LoadMCPServersJSONFile() error = %v, want nil", err)
	}
	if servers != nil {
		t.Fatalf("LoadMCPServersJSONFile() = %#v, want nil for missing file", servers)
	}
}

func TestPutMCPSidecarServerPreservesUnknownTopLevelKeysAndUntouchedEntries(t *testing.T) {
	t.Parallel()

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
  "custom": { "enabled": true },
  "mcpServers": {
    "alpha": { "command": "alpha-command" },
    "beta": { "command": "beta-command", "env": { "TOKEN": "value" } }
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
		t.Fatalf("root keys = %v, want preserved version key", mapsKeys(root))
	}
	if _, ok := root["custom"]; !ok {
		t.Fatalf("root keys = %v, want preserved custom key", mapsKeys(root))
	}

	var servers map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcpServers"], &servers); err != nil {
		t.Fatalf("json.Unmarshal(mcpServers) error = %v", err)
	}
	if got, want := servers["alpha"].Command, "updated-alpha"; got != want {
		t.Fatalf("servers[alpha].Command = %q, want %q", got, want)
	}
	if got := len(servers["alpha"].Args); got != 1 || servers["alpha"].Args[0] != "--flag" {
		t.Fatalf("servers[alpha].Args = %#v, want [--flag]", servers["alpha"].Args)
	}
	if got, want := servers["beta"].Command, "beta-command"; got != want {
		t.Fatalf("servers[beta].Command = %q, want %q", got, want)
	}
	if got, want := servers["beta"].Env["TOKEN"], "value"; got != want {
		t.Fatalf("servers[beta].Env[TOKEN] = %q, want %q", got, want)
	}
}

func TestDeleteMCPSidecarServerPrefersSnakeCaseCollectionWhenBothExist(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	target, err := ResolveMCPSidecarWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveMCPSidecarWriteTarget() error = %v", err)
	}

	writeFile(t, target.path, `{
  "mcpServers": {
    "alpha": { "command": "camel" }
  },
  "mcp_servers": {
    "alpha": { "command": "snake" },
    "beta": { "command": "keep" }
  }
}`)

	cfg, deleted, err := DeleteMCPSidecarServer(homePaths, "", target, "alpha")
	if err != nil {
		t.Fatalf("DeleteMCPSidecarServer() error = %v", err)
	}
	if !deleted {
		t.Fatal("DeleteMCPSidecarServer() deleted = false, want true")
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

	var camel map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcpServers"], &camel); err != nil {
		t.Fatalf("json.Unmarshal(mcpServers) error = %v", err)
	}
	if got, want := camel["alpha"].Command, "camel"; got != want {
		t.Fatalf("camel[alpha].Command = %q, want %q", got, want)
	}

	var snake map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcp_servers"], &snake); err != nil {
		t.Fatalf("json.Unmarshal(mcp_servers) error = %v", err)
	}
	if _, ok := snake["alpha"]; ok {
		t.Fatalf("snake collection still contains alpha: %#v", snake)
	}
	if got, want := snake["beta"].Command, "keep"; got != want {
		t.Fatalf("snake[beta].Command = %q, want %q", got, want)
	}
}

func TestPutMCPSidecarServerPreservesExistingCamelCaseCollection(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	target, err := ResolveMCPSidecarWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveMCPSidecarWriteTarget() error = %v", err)
	}

	writeFile(t, target.path, `{
  "mcpServers": {
    "alpha": { "command": "camel" }
  },
  "mcp_servers": {
    "beta": { "command": "snake" }
  }
}`)

	_, err = PutMCPSidecarServer(homePaths, "", target, MCPServer{
		Name:    "alpha",
		Command: "updated-camel",
	})
	if err != nil {
		t.Fatalf("PutMCPSidecarServer() error = %v", err)
	}

	payload, err := os.ReadFile(target.path)
	if err != nil {
		t.Fatalf("ReadFile(mcp.json) error = %v", err)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(payload, &root); err != nil {
		t.Fatalf("json.Unmarshal(root) error = %v", err)
	}

	var camel map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcpServers"], &camel); err != nil {
		t.Fatalf("json.Unmarshal(mcpServers) error = %v", err)
	}
	if got, want := camel["alpha"].Command, "updated-camel"; got != want {
		t.Fatalf("camel[alpha].Command = %q, want %q", got, want)
	}

	var snake map[string]mcpJSONServer
	if err := json.Unmarshal(root["mcp_servers"], &snake); err != nil {
		t.Fatalf("json.Unmarshal(mcp_servers) error = %v", err)
	}
	if got, want := snake["beta"].Command, "snake"; got != want {
		t.Fatalf("snake[beta].Command = %q, want %q", got, want)
	}
	if _, ok := snake["alpha"]; ok {
		t.Fatalf("snake collection unexpectedly contains alpha: %#v", snake)
	}
}

func TestDeleteMCPSidecarServerNoOpPreservesDocument(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	target, err := ResolveMCPSidecarWriteTarget(homePaths, "", WriteScopeGlobal)
	if err != nil {
		t.Fatalf("ResolveMCPSidecarWriteTarget() error = %v", err)
	}

	writeFile(t, target.path, `{
  "custom": { "enabled": true },
  "mcpServers": {
    "alpha": { "command": "camel" }
  }
}`)

	before, err := os.ReadFile(target.path)
	if err != nil {
		t.Fatalf("ReadFile(before) error = %v", err)
	}

	cfg, deleted, err := DeleteMCPSidecarServer(homePaths, "", target, "missing")
	if err != nil {
		t.Fatalf("DeleteMCPSidecarServer() error = %v", err)
	}
	if deleted {
		t.Fatal("DeleteMCPSidecarServer() deleted = true, want false")
	}
	if got, want := len(cfg.MCPServers), 1; got != want {
		t.Fatalf("len(Config.MCPServers) = %d, want %d", got, want)
	}

	after, err := os.ReadFile(target.path)
	if err != nil {
		t.Fatalf("ReadFile(after) error = %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("mcp.json changed on no-op delete\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestEditableMCPJSONDocumentCollectionForPut(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		document    editableMCPJSONDocument
		serverName  string
		wantKey     string
		wantPresent bool
	}{
		{
			name:       "existing snake entry wins",
			serverName: "alpha",
			document: editableMCPJSONDocument{
				snake: mcpJSONCollection{
					key:       "mcp_servers",
					nameIndex: map[string]string{"alpha": "Alpha"},
				},
				camel: newMCPJSONCollection("mcpServers"),
			},
			wantKey:     "mcp_servers",
			wantPresent: false,
		},
		{
			name:       "existing camel entry wins",
			serverName: "alpha",
			document: editableMCPJSONDocument{
				camel: mcpJSONCollection{
					key:       "mcpServers",
					nameIndex: map[string]string{"alpha": "alpha"},
				},
				snake: newMCPJSONCollection("mcp_servers"),
			},
			wantKey:     "mcpServers",
			wantPresent: false,
		},
		{
			name:       "present snake collection wins for new entry",
			serverName: "beta",
			document: editableMCPJSONDocument{
				snake: mcpJSONCollection{
					key:     "mcp_servers",
					present: true,
				},
				camel: newMCPJSONCollection("mcpServers"),
			},
			wantKey:     "mcp_servers",
			wantPresent: true,
		},
		{
			name:       "present camel collection wins for new entry",
			serverName: "beta",
			document: editableMCPJSONDocument{
				camel: mcpJSONCollection{
					key:     "mcpServers",
					present: true,
				},
				snake: newMCPJSONCollection("mcp_servers"),
			},
			wantKey:     "mcpServers",
			wantPresent: true,
		},
		{
			name:       "both present prefer snake for new entry",
			serverName: "beta",
			document: editableMCPJSONDocument{
				camel: mcpJSONCollection{
					key:     "mcpServers",
					present: true,
				},
				snake: mcpJSONCollection{
					key:     "mcp_servers",
					present: true,
				},
			},
			wantKey:     "mcp_servers",
			wantPresent: true,
		},
		{
			name:       "missing collections default to camel",
			serverName: "beta",
			document: editableMCPJSONDocument{
				camel: newMCPJSONCollection("mcpServers"),
				snake: newMCPJSONCollection("mcp_servers"),
			},
			wantKey:     "mcpServers",
			wantPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			collection := tt.document.collectionForPut(tt.serverName)
			if got, want := collection.key, tt.wantKey; got != want {
				t.Fatalf("collectionForPut().key = %q, want %q", got, want)
			}
			if got, want := collection.present, tt.wantPresent; got != want {
				t.Fatalf("collectionForPut().present = %v, want %v", got, want)
			}
		})
	}
}

func TestEditableMCPJSONDocumentPutUsesExistingNameAndDelete(t *testing.T) {
	t.Parallel()

	document, err := loadEditableMCPJSONDocument([]byte(`{
  "mcpServers": {
    " Alpha ": { "command": "old" }
  }
}`), "fixture")
	if err != nil {
		t.Fatalf("loadEditableMCPJSONDocument() error = %v", err)
	}

	if err := document.Put(MCPServer{Name: "alpha", Command: "updated"}); err != nil {
		t.Fatalf("document.Put() error = %v", err)
	}
	if _, ok := document.camel.entries[" Alpha "]; !ok {
		t.Fatalf("document.Put() entries = %#v, want existing name preserved", document.camel.entries)
	}

	if !document.Delete("alpha") {
		t.Fatal("document.Delete(alpha) = false, want true")
	}
	if document.Delete("alpha") {
		t.Fatal("document.Delete(alpha) second call = true, want false")
	}
}

func mapsKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
