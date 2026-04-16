package config

import (
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
