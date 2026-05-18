package settings

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestMCPServerTargetSelectorValidation(t *testing.T) {
	t.Parallel()

	t.Run("Should reject invalid put selectors without mutating MCP sources", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "alpha"
command = "config-before"
`)
		sidecarPath := filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName)
		writeFile(t, sidecarPath, `{
  "mcpServers": {
    "alpha": { "command": "sidecar-before" }
  }
}`)
		configBefore := readFile(t, homePaths.ConfigFile)
		sidecarBefore := readFile(t, sidecarPath)
		service := testService(t, homePaths, Dependencies{})

		_, err := service.PutCollectionItem(ctx, CollectionItemPutRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
			Name:              "alpha",
			Target:            TargetSelector("cfg"),
			MCPServer: &aghconfig.MCPServer{
				Command: "after",
			},
		})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("PutCollectionItem(invalid target) error = %v, want ErrValidation", err)
		}
		if !strings.Contains(err.Error(), "unsupported MCP target selector") {
			t.Fatalf("PutCollectionItem(invalid target) error = %v, want selector context", err)
		}
		if got := readFile(t, homePaths.ConfigFile); got != configBefore {
			t.Fatalf("config payload changed after invalid target:\n%s", got)
		}
		if got := readFile(t, sidecarPath); got != sidecarBefore {
			t.Fatalf("sidecar payload changed after invalid target:\n%s", got)
		}
	})

	t.Run("Should reject invalid delete selectors without mutating MCP sources", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		homePaths := testHomePaths(t)
		writeFile(t, homePaths.ConfigFile, `
[[mcp_servers]]
name = "alpha"
command = "config-before"
`)
		sidecarPath := filepath.Join(homePaths.HomeDir, aghconfig.MCPJSONName)
		writeFile(t, sidecarPath, `{
  "mcpServers": {
    "alpha": { "command": "sidecar-before" }
  }
}`)
		configBefore := readFile(t, homePaths.ConfigFile)
		sidecarBefore := readFile(t, sidecarPath)
		service := testService(t, homePaths, Dependencies{})

		_, err := service.DeleteCollectionItem(ctx, CollectionItemDeleteRequest{
			CollectionRequest: CollectionRequest{Collection: CollectionMCPServers},
			Name:              "alpha",
			Target:            TargetSelector("CONFIG"),
		})
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("DeleteCollectionItem(invalid target) error = %v, want ErrValidation", err)
		}
		if !strings.Contains(err.Error(), "unsupported MCP target selector") {
			t.Fatalf("DeleteCollectionItem(invalid target) error = %v, want selector context", err)
		}
		if got := readFile(t, homePaths.ConfigFile); got != configBefore {
			t.Fatalf("config payload changed after invalid target:\n%s", got)
		}
		if got := readFile(t, sidecarPath); got != sidecarBefore {
			t.Fatalf("sidecar payload changed after invalid target:\n%s", got)
		}
	})
}
