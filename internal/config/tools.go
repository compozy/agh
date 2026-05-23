package config

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	toolspkg "github.com/compozy/agh/internal/tools"
)

const (
	// DefaultToolsMaxResultBytes is the TechSpec default result budget for descriptors without one.
	DefaultToolsMaxResultBytes int64 = 256 << 10
	// MaxToolsMaxResultBytes bounds config-level default result budgets.
	MaxToolsMaxResultBytes int64 = 16 << 20

	// DefaultToolsApprovalTimeoutSeconds is the TechSpec default approval wait.
	DefaultToolsApprovalTimeoutSeconds = 120
	// MinToolsApprovalTimeoutSeconds is the smallest supported approval wait.
	MinToolsApprovalTimeoutSeconds = 1
	// MaxToolsApprovalTimeoutSeconds is the largest supported approval wait.
	MaxToolsApprovalTimeoutSeconds = 600

	// DefaultHostedMCPBindNonceTTLSeconds is the TechSpec default hosted MCP bind window.
	DefaultHostedMCPBindNonceTTLSeconds = 30
	// MinHostedMCPBindNonceTTLSeconds is the smallest supported hosted MCP bind window.
	MinHostedMCPBindNonceTTLSeconds = 1
	// MaxHostedMCPBindNonceTTLSeconds is the largest supported hosted MCP bind window.
	MaxHostedMCPBindNonceTTLSeconds = 300
)

// ToolsExternalDefault controls default policy for external executable sources.
type ToolsExternalDefault string

const (
	// ToolsExternalDefaultDisabled keeps external tools operator-visible but not session-callable by default.
	ToolsExternalDefaultDisabled ToolsExternalDefault = "disabled"
	// ToolsExternalDefaultAsk requires approval for external tools that otherwise pass policy.
	ToolsExternalDefaultAsk ToolsExternalDefault = "ask"
	// ToolsExternalDefaultEnabled allows external tools that otherwise pass policy.
	ToolsExternalDefaultEnabled ToolsExternalDefault = "enabled"
)

// ToolsConfig controls registry, hosted MCP, and default tool-policy lifecycle settings.
type ToolsConfig struct {
	Enabled               bool                 `toml:"enabled"`
	HostedMCPEnabled      bool                 `toml:"hosted_mcp_enabled"`
	DefaultMaxResultBytes int64                `toml:"default_max_result_bytes"`
	HostedMCP             ToolsHostedMCPConfig `toml:"hosted_mcp"`
	Policy                ToolsPolicyConfig    `toml:"policy"`
}

// ToolsHostedMCPConfig controls AGH-hosted MCP launch binding values.
type ToolsHostedMCPConfig struct {
	BindNonceTTLSeconds int `toml:"bind_nonce_ttl_seconds"`
}

// ToolsPolicyConfig controls default registry policy values consumed by later policy evaluation.
type ToolsPolicyConfig struct {
	ExternalDefault        ToolsExternalDefault `toml:"external_default"`
	ApprovalTimeoutSeconds int                  `toml:"approval_timeout_seconds"`
	TrustedSources         []string             `toml:"trusted_sources,omitempty"`
}

// DefaultToolsConfig returns the TechSpec defaults for tool registry configuration.
func DefaultToolsConfig() ToolsConfig {
	return ToolsConfig{
		Enabled:               true,
		HostedMCPEnabled:      true,
		DefaultMaxResultBytes: DefaultToolsMaxResultBytes,
		HostedMCP: ToolsHostedMCPConfig{
			BindNonceTTLSeconds: DefaultHostedMCPBindNonceTTLSeconds,
		},
		Policy: ToolsPolicyConfig{
			ExternalDefault:        ToolsExternalDefaultDisabled,
			ApprovalTimeoutSeconds: DefaultToolsApprovalTimeoutSeconds,
			TrustedSources:         []string{},
		},
	}
}

// Validate ensures tools lifecycle configuration is deterministic and safe to consume.
func (c ToolsConfig) Validate(mcpServers []MCPServer, providers map[string]ProviderConfig) error {
	if c.DefaultMaxResultBytes < 0 || c.DefaultMaxResultBytes > MaxToolsMaxResultBytes {
		return fmt.Errorf(
			"tools.default_max_result_bytes must be between 0 and %d: %d",
			MaxToolsMaxResultBytes,
			c.DefaultMaxResultBytes,
		)
	}
	if err := c.HostedMCP.Validate(); err != nil {
		return err
	}
	return c.Policy.Validate(configuredMCPSourceOwners(mcpServers, providers))
}

// BindNonceTTL returns the configured hosted MCP nonce lifetime.
func (c ToolsHostedMCPConfig) BindNonceTTL() time.Duration {
	return time.Duration(c.BindNonceTTLSeconds) * time.Second
}

// Validate ensures hosted MCP lifecycle values are inside daemon bounds.
func (c ToolsHostedMCPConfig) Validate() error {
	if c.BindNonceTTLSeconds < MinHostedMCPBindNonceTTLSeconds ||
		c.BindNonceTTLSeconds > MaxHostedMCPBindNonceTTLSeconds {
		return fmt.Errorf(
			"tools.hosted_mcp.bind_nonce_ttl_seconds must be between %d and %d: %d",
			MinHostedMCPBindNonceTTLSeconds,
			MaxHostedMCPBindNonceTTLSeconds,
			c.BindNonceTTLSeconds,
		)
	}
	return nil
}

// ApprovalTimeout returns the configured tool approval wait.
func (c ToolsPolicyConfig) ApprovalTimeout() time.Duration {
	return time.Duration(c.ApprovalTimeoutSeconds) * time.Second
}

// Validate ensures policy defaults are inside the supported grammar.
func (c ToolsPolicyConfig) Validate(knownMCPSourceOwners map[string]struct{}) error {
	switch c.ExternalDefault {
	case ToolsExternalDefaultDisabled, ToolsExternalDefaultAsk, ToolsExternalDefaultEnabled:
	default:
		return fmt.Errorf(
			"tools.policy.external_default must be one of %q, %q, %q: %q",
			ToolsExternalDefaultDisabled,
			ToolsExternalDefaultAsk,
			ToolsExternalDefaultEnabled,
			c.ExternalDefault,
		)
	}
	if c.ApprovalTimeoutSeconds < MinToolsApprovalTimeoutSeconds ||
		c.ApprovalTimeoutSeconds > MaxToolsApprovalTimeoutSeconds {
		return fmt.Errorf(
			"tools.policy.approval_timeout_seconds must be between %d and %d: %d",
			MinToolsApprovalTimeoutSeconds,
			MaxToolsApprovalTimeoutSeconds,
			c.ApprovalTimeoutSeconds,
		)
	}

	seen := make(map[string]struct{}, len(c.TrustedSources))
	for idx, raw := range c.TrustedSources {
		field := fmt.Sprintf("tools.policy.trusted_sources[%d]", idx)
		ref, err := parseTrustedToolSourceRef(raw, field)
		if err != nil {
			return err
		}
		if _, ok := seen[ref.String()]; ok {
			return fmt.Errorf("%s duplicates trusted source %q", field, ref.String())
		}
		seen[ref.String()] = struct{}{}
		if ref.Kind == toolspkg.SourceMCP {
			if _, ok := knownMCPSourceOwners[ref.Owner]; !ok {
				return fmt.Errorf("%s references unknown MCP source %q", field, ref.Owner)
			}
		}
	}
	return nil
}

type trustedToolSourceRef struct {
	Kind  toolspkg.SourceKind
	Owner string
}

func (r trustedToolSourceRef) String() string {
	return string(r.Kind) + ":" + r.Owner
}

var trustedToolSourceOwnerPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func parseTrustedToolSourceRef(raw string, field string) (trustedToolSourceRef, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return trustedToolSourceRef{}, fmt.Errorf("%s is required", field)
	}
	if trimmed != raw {
		return trustedToolSourceRef{}, fmt.Errorf("%s must not include surrounding whitespace: %q", field, raw)
	}

	kindText, owner, ok := strings.Cut(trimmed, ":")
	if !ok || strings.Contains(owner, ":") {
		return trustedToolSourceRef{}, fmt.Errorf("%s must use kind:owner syntax: %q", field, raw)
	}
	kind := toolspkg.SourceKind(kindText)
	switch kind {
	case toolspkg.SourceMCP, toolspkg.SourceExtension:
	default:
		return trustedToolSourceRef{}, fmt.Errorf("%s kind must be %q or %q: %q", field, "mcp", "extension", kindText)
	}
	if !trustedToolSourceOwnerPattern.MatchString(owner) {
		return trustedToolSourceRef{}, fmt.Errorf(
			"%s owner must match %q: %q",
			field,
			trustedToolSourceOwnerPattern.String(),
			owner,
		)
	}

	return trustedToolSourceRef{Kind: kind, Owner: owner}, nil
}

func configuredMCPSourceOwners(mcpServers []MCPServer, providers map[string]ProviderConfig) map[string]struct{} {
	owners := make(map[string]struct{})
	add := func(server MCPServer) {
		name := strings.TrimSpace(server.Name)
		if name != "" {
			owners[name] = struct{}{}
		}
	}
	for _, server := range mcpServers {
		add(server)
	}
	for _, provider := range providers {
		for _, server := range provider.MCPServers {
			add(server)
		}
	}
	return owners
}
