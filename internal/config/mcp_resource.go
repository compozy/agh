package config

import (
	"context"
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
		return MCPServer{}, err
	}

	normalized := cloneMCPServer(spec)
	normalized.Name = strings.TrimSpace(spec.Name)
	normalized.Command = strings.TrimSpace(spec.Command)
	for idx, arg := range normalized.Args {
		normalized.Args[idx] = strings.TrimSpace(arg)
	}
	if len(normalized.Env) > 0 {
		for key, value := range normalized.Env {
			trimmedKey := strings.TrimSpace(key)
			delete(normalized.Env, key)
			if trimmedKey == "" {
				continue
			}
			normalized.Env[trimmedKey] = strings.TrimSpace(value)
		}
		if len(normalized.Env) == 0 {
			normalized.Env = nil
		}
	}

	if err := normalized.Validate("mcp_server"); err != nil {
		return MCPServer{}, err
	}
	return normalized, nil
}
