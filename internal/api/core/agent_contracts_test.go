package core_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	"github.com/compozy/agh/internal/api/core"
	aghconfig "github.com/compozy/agh/internal/config"
)

func TestAgentPayloadDoesNotExposeMCPSecretBindings(t *testing.T) {
	t.Run("Should project MCP bindings as redacted presence without Vault refs", func(t *testing.T) {
		t.Parallel()

		payload := core.AgentPayloadFromDef(aghconfig.AgentDef{
			Name:     "reviewer",
			Provider: "codex",
			MCPServers: []aghconfig.MCPServer{{
				Name:      "github",
				Command:   "npx",
				SecretEnv: map[string]string{"GITHUB_TOKEN": "vault:mcp/global/github/env/GITHUB_TOKEN"},
				Auth: aghconfig.MCPAuthConfig{
					Type:            aghconfig.MCPAuthTypeOAuth2PKCE,
					ClientSecretRef: "vault:mcp/global/github/oauth/client-secret",
				},
			}},
		})

		if got, want := payload.MCPServers[0].SecretEnv["GITHUB_TOKEN"], aghconfig.RedactedValue(); got != want {
			t.Fatalf("MCP secret env projection = %q, want %q", got, want)
		}
		if !payload.MCPServers[0].Auth.ClientSecretConfigured {
			t.Fatal("MCP OAuth client secret configured = false, want presence metadata")
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("json.Marshal(agent payload) error = %v", err)
		}
		if strings.Contains(string(encoded), "vault:mcp/") {
			t.Fatalf("agent payload exposed MCP Vault binding: %s", encoded)
		}
	})
}

func TestCoordinatorConfigPayloadFromConfig(t *testing.T) {
	t.Parallel()

	baseConfig := aghconfig.CoordinatorConfig{
		Enabled:                       true,
		AgentName:                     " coordinator ",
		Provider:                      " codex ",
		Model:                         " gpt-5.4 ",
		DefaultTTL:                    90 * time.Minute,
		MaxChildren:                   5,
		MaxActiveSessionsPerWorkspace: 5,
	}

	tests := []struct {
		name        string
		cfg         aghconfig.CoordinatorConfig
		source      contract.CoordinatorConfigSource
		workspaceID string
		assert      func(*testing.T, contract.CoordinatorConfigPayload)
	}{
		{
			name:        "Should trim coordinator string fields",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: " ws-1 ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.AgentName != "coordinator" || payload.Provider != "codex" || payload.Model != "gpt-5.4" {
					t.Fatalf("trimmed fields = %q/%q/%q", payload.AgentName, payload.Provider, payload.Model)
				}
			},
		},
		{
			name:        "Should convert default TTL to seconds",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: "ws-1",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.DefaultTTLSeconds != 5400 {
					t.Fatalf("DefaultTTLSeconds = %d, want 5400", payload.DefaultTTLSeconds)
				}
			},
		},
		{
			name:        "Should map coordinator limits",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: "ws-1",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.MaxChildren != 5 || payload.MaxActiveSessionsPerWorkspace != 5 {
					t.Fatalf(
						"limits = %d/%d, want 5/5",
						payload.MaxChildren,
						payload.MaxActiveSessionsPerWorkspace,
					)
				}
			},
		},
		{
			name:        "Should preserve disabled configs",
			cfg:         aghconfig.CoordinatorConfig{Enabled: false},
			source:      contract.CoordinatorConfigSourceDefault,
			workspaceID: "",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.Enabled {
					t.Fatal("Enabled = true, want false")
				}
			},
		},
		{
			name:        "Should trim source workspace id",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceWorkspace,
			workspaceID: " ws-1 ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.Source != contract.CoordinatorConfigSourceWorkspace || payload.WorkspaceID != "ws-1" {
					t.Fatalf("source/workspace = %q/%q, want workspace/ws-1", payload.Source, payload.WorkspaceID)
				}
			},
		},
		{
			name:        "Should keep empty workspace id empty",
			cfg:         baseConfig,
			source:      contract.CoordinatorConfigSourceDefault,
			workspaceID: " ",
			assert: func(t *testing.T, payload contract.CoordinatorConfigPayload) {
				t.Helper()
				if payload.WorkspaceID != "" {
					t.Fatalf("WorkspaceID = %q, want empty", payload.WorkspaceID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			payload := core.CoordinatorConfigPayloadFromConfig(tt.cfg, tt.source, tt.workspaceID)
			tt.assert(t, payload)
		})
	}
}
