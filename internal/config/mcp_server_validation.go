package config

import (
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/vault"
)

// Validate ensures the MCP server entry is usable.
func (s MCPServer) Validate(path string) error {
	transport := s.EffectiveTransport()
	if err := transport.Validate(path + ".transport"); err != nil {
		return err
	}
	if err := s.Auth.Validate(path + ".auth"); err != nil {
		return err
	}
	switch {
	case strings.TrimSpace(s.Name) == "":
		return fmt.Errorf("%s.name is required", path)
	case transport == MCPServerTransportStdio && strings.TrimSpace(s.Command) == "":
		return fmt.Errorf("%s.command is required", path)
	case transport == MCPServerTransportStdio && strings.TrimSpace(s.URL) != "":
		return fmt.Errorf("%s.url requires remote transport", path)
	case transport != MCPServerTransportStdio && strings.TrimSpace(s.URL) == "":
		return fmt.Errorf("%s.url is required for %s transport", path, transport)
	case transport != MCPServerTransportStdio && strings.TrimSpace(s.Command) != "":
		return fmt.Errorf("%s.command is only valid for stdio transport", path)
	case transport != MCPServerTransportStdio && len(s.Args) > 0:
		return fmt.Errorf("%s.args is only valid for stdio transport", path)
	case transport != MCPServerTransportStdio && len(s.Env) > 0:
		return fmt.Errorf("%s.env is only valid for stdio transport", path)
	case transport == MCPServerTransportStdio && !s.Auth.IsZero():
		return fmt.Errorf("%s.auth is only valid for remote MCP servers", path)
	case transport != MCPServerTransportStdio && len(s.SecretEnv) > 0:
		return fmt.Errorf("%s.secret_env is only valid for stdio transport", path)
	default:
		return validateStdioMCPEnv(path, transport, s.Env, s.SecretEnv)
	}
}

func validateStdioMCPEnv(
	path string,
	transport MCPServerTransport,
	env map[string]string,
	secretEnv map[string]string,
) error {
	if transport != MCPServerTransportStdio {
		return nil
	}
	for key := range env {
		if forbiddenStdioMCPEnvKey(key) {
			return fmt.Errorf("%s.env.%s is forbidden for stdio MCP servers", path, strings.TrimSpace(key))
		}
		if vault.SecretLikeEnvName(key) {
			return fmt.Errorf("%s.env.%s must move secret-like values to secret_env", path, strings.TrimSpace(key))
		}
	}
	for key := range secretEnv {
		if forbiddenStdioMCPEnvKey(key) {
			return fmt.Errorf(
				"%s.secret_env.%s is forbidden for stdio MCP servers",
				path,
				strings.TrimSpace(key),
			)
		}
	}
	return vault.ValidateSecretEnvMap(path, "mcp", secretEnv)
}

func forbiddenStdioMCPEnvKey(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	switch normalized {
	case providerNodeOptionsValue, "PYTHONPATH", "PYTHONHOME", "LD_PRELOAD":
		return true
	default:
		return strings.HasPrefix(normalized, "DYLD_")
	}
}
