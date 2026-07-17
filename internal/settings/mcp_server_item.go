package settings

import (
	"strings"

	aghconfig "github.com/compozy/agh/internal/config"
	mcpauth "github.com/compozy/agh/internal/mcp/auth"
)

// MCPServerItem is one MCP server collection row.
type MCPServerItem struct {
	Name           string
	Transport      aghconfig.MCPServerTransport
	Command        string
	Args           []string
	Env            map[string]string
	SecretEnv      map[string]string
	URL            string
	Auth           aghconfig.MCPAuthConfig
	AuthStatus     *mcpauth.Status
	RuntimeStatus  *MCPServerRuntimeStatus
	Scope          ScopeKind
	WorkspaceID    string
	CatalogEntry   string
	CatalogVersion string
	SourceMetadata SourceMetadata
}

func baseMCPServerItem(
	effective mcpSourceEntry,
	entries []mcpSourceEntry,
	scope ScopeKind,
	workspaceID string,
) MCPServerItem {
	shadowed := make([]SourceRef, 0, len(entries)-1)
	for idx := len(entries) - 2; idx >= 0; idx-- {
		shadowed = append(shadowed, entries[idx].Source)
	}
	return MCPServerItem{
		Name:           effective.Server.Name,
		Transport:      effective.Server.EffectiveTransport(),
		Command:        effective.Server.Command,
		Args:           append([]string(nil), effective.Server.Args...),
		Env:            aghconfig.RedactStringMap(effective.Server.Env),
		SecretEnv:      cloneStringMap(effective.Server.SecretEnv),
		URL:            strings.TrimSpace(effective.Server.URL),
		Auth:           effective.Server.Auth,
		Scope:          scope,
		WorkspaceID:    workspaceID,
		CatalogEntry:   strings.TrimSpace(effective.Server.CatalogEntry),
		CatalogVersion: strings.TrimSpace(effective.Server.CatalogVersion),
		SourceMetadata: SourceMetadata{
			EffectiveSource:  effective.Source,
			ShadowedSources:  shadowed,
			AvailableTargets: availableTargetsForScope(scope),
		},
	}
}

func committedMCPServerItem(
	server aghconfig.MCPServer,
	scope ScopeKind,
	workspaceID string,
	target WriteTargetKind,
) MCPServerItem {
	entry := mcpSourceEntry{
		Source: sourceRefForWriteTarget(target, workspaceID),
		Target: target,
		Server: server,
	}
	return baseMCPServerItem(entry, []mcpSourceEntry{entry}, scope, workspaceID)
}
