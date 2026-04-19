package config

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/resources"
)

func TestAgentResourceCodecRejectsInvalidSpecs(t *testing.T) {
	t.Parallel()

	codec, err := NewAgentResourceCodec()
	if err != nil {
		t.Fatalf("NewAgentResourceCodec() error = %v", err)
	}
	scope := resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal}

	tests := []struct {
		name    string
		spec    AgentDef
		wantErr string
	}{
		{
			name: "ShouldRejectMissingName",
			spec: AgentDef{
				Prompt: "You are helpful.",
			},
			wantErr: "agent name is required",
		},
		{
			name: "ShouldRejectMissingPrompt",
			spec: AgentDef{
				Name: "coder",
			},
			wantErr: "agent prompt is required",
		},
		{
			name: "ShouldRejectInvalidPermissions",
			spec: AgentDef{
				Name:        "coder",
				Prompt:      "You are helpful.",
				Permissions: "invalid",
			},
			wantErr: "agent.permissions",
		},
		{
			name: "ShouldRejectInvalidMCPServer",
			spec: AgentDef{
				Name:   "coder",
				Prompt: "You are helpful.",
				MCPServers: []MCPServer{{
					Name: "github",
				}},
			},
			wantErr: "agent.mcp_servers[0]",
		},
		{
			name: "ShouldRejectInvalidCapabilityCatalog",
			spec: AgentDef{
				Name:   "coder",
				Prompt: "You are helpful.",
				Capabilities: &CapabilityCatalog{
					Capabilities: []CapabilityDef{{
						ID: "build-site",
					}},
				},
			},
			wantErr: "agent.capabilities",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw, err := codec.Encode(tt.spec)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}
			_, err = codec.DecodeAndValidate(context.Background(), scope, raw)
			if err == nil {
				t.Fatal("DecodeAndValidate() error = nil, want validation error")
			}
			if !errors.Is(err, resources.ErrValidation) {
				t.Fatalf("DecodeAndValidate() error = %v, want resources.ErrValidation", err)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("DecodeAndValidate() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestAgentResourceCodecCanonicalizesTypedRecordSpec(t *testing.T) {
	t.Parallel()

	codec, err := NewAgentResourceCodec()
	if err != nil {
		t.Fatalf("NewAgentResourceCodec() error = %v", err)
	}
	raw, err := codec.Encode(AgentDef{
		Name:   " coder ",
		Prompt: " Build things. ",
		Tools:  []string{" github.search ", "", "github.search", " * "},
		Capabilities: &CapabilityCatalog{
			Capabilities: []CapabilityDef{{
				ID:                " build-site ",
				Summary:           " Build the landing page. ",
				Outcome:           " A finished landing page. ",
				ContextNeeded:     []string{" repo ", "", " brand brief "},
				ExecutionOutline:  []string{" inspect ", " build "},
				ArtifactsExpected: []string{" final page "},
			}},
		},
		MCPServers: []MCPServer{{
			Name:    " github ",
			Command: " npx ",
			Args:    []string{" -y "},
		}},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got, err := codec.DecodeAndValidate(
		context.Background(),
		resources.ResourceScope{Kind: resources.ResourceScopeKindWorkspace, ID: "ws_1"},
		raw,
	)
	if err != nil {
		t.Fatalf("DecodeAndValidate() error = %v", err)
	}
	if got.Name != "coder" || got.Prompt != "Build things." {
		t.Fatalf("decoded agent = %#v, want trimmed name and prompt", got)
	}
	if want := []string{"github.search", "*"}; strings.Join(got.Tools, ",") != strings.Join(want, ",") {
		t.Fatalf("Tools = %#v, want %#v", got.Tools, want)
	}
	if gotCount, wantCount := len(got.MCPServers), 1; gotCount != wantCount {
		t.Fatalf("len(MCPServers) = %d, want %d", gotCount, wantCount)
	}
	if got.MCPServers[0].Name != "github" || got.MCPServers[0].Command != "npx" {
		t.Fatalf("MCPServers = %#v, want trimmed name/command", got.MCPServers)
	}
	if got.Capabilities == nil || len(got.Capabilities.Capabilities) != 1 {
		t.Fatalf("Capabilities = %#v, want one normalized capability", got.Capabilities)
	}
	if got.Capabilities.Capabilities[0].ID != "build-site" {
		t.Fatalf("Capabilities[0].ID = %q, want build-site", got.Capabilities.Capabilities[0].ID)
	}
	if want := []string{
		"repo",
		"brand brief",
	}; strings.Join(
		got.Capabilities.Capabilities[0].ContextNeeded,
		",",
	) != strings.Join(
		want,
		",",
	) {
		t.Fatalf("ContextNeeded = %#v, want %#v", got.Capabilities.Capabilities[0].ContextNeeded, want)
	}
}
