//go:build integration

package settings

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestProviderOverlayDeleteRevealsBuiltinFallbackMetadataCorrectly(t *testing.T) {
	ctx := context.Background()
	homePaths := testHomePaths(t)
	writeFile(t, homePaths.ConfigFile, `
[providers.codex]
command = "custom-codex"
`)

	service := testService(t, homePaths, Dependencies{})

	before, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionProviders})
	if err != nil {
		t.Fatalf("ListCollection(before providers) error = %v", err)
	}
	codex := findProviderItem(t, before.Providers, "codex")
	if got, want := codex.SourceMetadata.EffectiveSource.Kind, SourceKindGlobalConfig; got != want {
		t.Fatalf("before delete effective source = %q, want %q", got, want)
	}
	if codex.Fallback == nil {
		t.Fatal("before delete fallback = nil, want builtin fallback metadata")
	}

	if _, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		Name:              "codex",
	}); err != nil {
		t.Fatalf("DeleteCollectionItem(provider) error = %v", err)
	}

	after, err := service.ListCollection(ctx, CollectionRequest{Collection: CollectionProviders})
	if err != nil {
		t.Fatalf("ListCollection(after providers) error = %v", err)
	}
	codex = findProviderItem(t, after.Providers, "codex")
	if got, want := codex.SourceMetadata.EffectiveSource.Kind, SourceKindBuiltinProvider; got != want {
		t.Fatalf("after delete effective source = %q, want %q", got, want)
	}
	if codex.Fallback != nil {
		t.Fatalf("after delete fallback = %#v, want nil", codex.Fallback)
	}
}

func TestWorkspaceScopedMCPMutationResolvesWorkspaceRootAndPersistsToTarget(t *testing.T) {
	ctx := context.Background()
	homePaths := testHomePaths(t)
	workspaceRoot := filepath.Join(t.TempDir(), "workspace")

	service := testService(t, homePaths, Dependencies{
		WorkspaceResolver: fakeWorkspaceResolver{
			resolved: map[string]workspacepkg.ResolvedWorkspace{
				"ws-1": {
					Workspace: workspacepkg.Workspace{ID: "ws-1", RootDir: workspaceRoot},
				},
			},
		},
	})

	result, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{
			Collection:  CollectionMCPServers,
			Scope:       ScopeWorkspace,
			WorkspaceID: "ws-1",
		},
		Name: "workspace-alpha",
		MCPServer: &aghconfig.MCPServer{
			Command: "workspace-command",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(workspace mcp) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetWorkspaceMCPSidecar; got != want {
		t.Fatalf("workspace mcp write target = %q, want %q", got, want)
	}

	sidecarPath := filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.MCPJSONName)
	payload := readFile(t, sidecarPath)
	if !strings.Contains(payload, `"workspace-alpha"`) || !strings.Contains(payload, `"workspace-command"`) {
		t.Fatalf("workspace sidecar missing persisted MCP server:\n%s", payload)
	}
}

func TestMutationResultExposesSemanticWriteTarget(t *testing.T) {
	ctx := context.Background()
	homePaths := testHomePaths(t)
	service := testService(t, homePaths, Dependencies{})

	result, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
		CollectionRequest: CollectionRequest{Collection: CollectionProviders},
		Name:              "custom",
		Provider: &ProviderSettings{
			Command: "custom-provider",
		},
	})
	if err != nil {
		t.Fatalf("PutCollectionItem(provider) error = %v", err)
	}
	if got, want := result.WriteTarget, WriteTargetGlobalConfig; got != want {
		t.Fatalf("provider write target = %q, want %q", got, want)
	}
	if strings.Contains(string(result.WriteTarget), "/") {
		t.Fatalf("provider write target = %q, want semantic identifier not path", result.WriteTarget)
	}
}

func findProviderItem(t *testing.T, items []ProviderItem, name string) ProviderItem {
	t.Helper()
	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("Provider item %q not found in %#v", name, items)
	return ProviderItem{}
}
