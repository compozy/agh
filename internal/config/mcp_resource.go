package config

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pedronauck/agh/internal/resources"
)

const (
	// MCPServerResourceKind is the canonical desired-state resource kind for MCP server records.
	MCPServerResourceKind     resources.ResourceKind = "mcp_server"
	mcpServerResourceMaxBytes                        = 256 << 10
)

// NewMCPServerResourceCodec builds the canonical MCP server resource codec.
func NewMCPServerResourceCodec() (resources.KindCodec[MCPServer], error) {
	return resources.NewJSONCodec(MCPServerResourceKind, mcpServerResourceMaxBytes, validateMCPServerSpec)
}

func validateMCPServerSpec(
	_ context.Context,
	scope resources.ResourceScope,
	spec MCPServer,
) (MCPServer, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return MCPServer{}, fmt.Errorf("config: validate mcp resource scope: %w", err)
	}

	normalized := cloneMCPServer(spec)
	normalized.Name = strings.TrimSpace(spec.Name)
	normalized.Command = strings.TrimSpace(spec.Command)
	for idx, arg := range normalized.Args {
		normalized.Args[idx] = strings.TrimSpace(arg)
	}
	if len(normalized.Env) > 0 {
		keys := make([]string, 0, len(normalized.Env))
		for key := range normalized.Env {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		canonicalEnv := make(map[string]string, len(keys))
		for _, key := range keys {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey == "" {
				continue
			}
			canonicalEnv[trimmedKey] = strings.TrimSpace(normalized.Env[key])
		}
		if len(canonicalEnv) == 0 {
			normalized.Env = nil
		} else {
			normalized.Env = canonicalEnv
		}
	}

	if err := normalized.Validate("mcp_server"); err != nil {
		return MCPServer{}, fmt.Errorf("config: validate mcp resource spec: %w", err)
	}
	return normalized, nil
}
