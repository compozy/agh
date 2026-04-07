package skills

import (
	"log/slog"
	"sort"
	"strings"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

// MCPResolver collects and resolves MCP server declarations from active skills.
type MCPResolver struct {
	allowedMarketplace []string
	logger             *slog.Logger
}

// NewMCPResolver constructs an MCPResolver from skills config and logger settings.
func NewMCPResolver(cfg aghconfig.SkillsConfig, logger *slog.Logger) *MCPResolver {
	if logger == nil {
		logger = slog.Default()
	}

	return &MCPResolver{
		allowedMarketplace: cloneStrings(cfg.AllowedMarketplaceMCP),
		logger:             logger,
	}
}

// Resolve returns MCP servers from active skills after trust-tier filtering.
func (mr *MCPResolver) Resolve(skills []*Skill) []aghconfig.MCPServer {
	if len(skills) == 0 {
		return nil
	}

	ordered := orderSkillsBySource(skills)
	allowedMarketplace := make(map[string]struct{}, len(mr.allowedMarketplace))
	for _, name := range mr.allowedMarketplace {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		allowedMarketplace[trimmed] = struct{}{}
	}

	resolved := make([]aghconfig.MCPServer, 0)
	index := make(map[string]int)
	origins := make([]mcpOrigin, 0)

	for _, skill := range ordered {
		if skill == nil || len(skill.MCPServers) == 0 {
			continue
		}
		for _, server := range skill.MCPServers {
			if !mcpServerAllowed(skill, allowedMarketplace) {
				mr.logger.Warn(
					"blocked MCP server",
					"skill_name", skill.Meta.Name,
					"mcp_server", server.Name,
					"source", skillSourceName(skill.Source),
				)
				continue
			}

			resolvedServer := toConfigMCPServer(server)
			name := strings.TrimSpace(resolvedServer.Name)
			if idx, ok := index[name]; ok && name != "" {
				resolved[idx] = resolvedServer
				origins[idx] = mcpOrigin{
					skillName: skill.Meta.Name,
					source:    skill.Source,
				}
				continue
			}

			resolved = append(resolved, resolvedServer)
			origins = append(origins, mcpOrigin{
				skillName: skill.Meta.Name,
				source:    skill.Source,
			})
			if name != "" {
				index[name] = len(resolved) - 1
			}
		}
	}

	for i, server := range resolved {
		mr.logger.Info(
			"resolved MCP server",
			"skill_name", origins[i].skillName,
			"mcp_server", server.Name,
			"source", skillSourceName(origins[i].source),
		)
	}

	if len(resolved) == 0 {
		return nil
	}

	return resolved
}

type mcpOrigin struct {
	skillName string
	source    SkillSource
}

func orderSkillsBySource(skills []*Skill) []*Skill {
	ordered := append([]*Skill(nil), skills...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left == nil || right == nil {
			return left != nil
		}
		return left.Source < right.Source
	})
	return ordered
}

func mcpServerAllowed(skill *Skill, allowedMarketplace map[string]struct{}) bool {
	if skill == nil {
		return false
	}

	switch skill.Source {
	case SourceBundled, SourceUser, SourceAdditional, SourceWorkspace:
		return true
	case SourceMarketplace:
		_, ok := allowedMarketplace[strings.TrimSpace(skill.Meta.Name)]
		return ok
	default:
		return false
	}
}

func toConfigMCPServer(decl MCPServerDecl) aghconfig.MCPServer {
	return aghconfig.MCPServer{
		Name:    decl.Name,
		Command: decl.Command,
		Args:    append([]string(nil), decl.Args...),
		Env:     cloneStringMap(decl.Env),
	}
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}

	return append([]string(nil), values...)
}
