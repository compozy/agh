package skills

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestNewMCPResolverClonesAllowedMarketplaceConfig(t *testing.T) {
	t.Parallel()

	cfg := aghconfig.SkillsConfig{
		AllowedMarketplaceMCP: []string{"marketplace-skill"},
	}

	resolver := NewMCPResolver(cfg, nil)
	cfg.AllowedMarketplaceMCP[0] = "changed"

	if len(resolver.allowedMarketplace) != 1 || resolver.allowedMarketplace[0] != "marketplace-skill" {
		t.Fatalf("allowedMarketplace = %#v, want cloned allowlist", resolver.allowedMarketplace)
	}
	if resolver.logger == nil {
		t.Fatal("logger = nil, want default logger")
	}
}

func TestMCPResolverResolveAutoApprovesTrustedSources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source SkillSource
	}{
		{name: "bundled", source: SourceBundled},
		{name: "user", source: SourceUser},
		{name: "additional", source: SourceAdditional},
		{name: "workspace", source: SourceWorkspace},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver, logs := newResolverForTest(nil)
			skill := newSkillWithServer(tt.name+"-skill", tt.source, MCPServerDecl{
				Name:    "filesystem",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem"},
				Env: map[string]string{
					"ROOT": "/workspace",
				},
			})

			got := resolver.Resolve([]*Skill{skill})
			if len(got) != 1 {
				t.Fatalf("Resolve() len = %d, want 1", len(got))
			}
			if got[0].Name != "filesystem" || got[0].Command != "npx" {
				t.Fatalf("Resolve() server = %#v, want converted MCP server", got[0])
			}
			if len(got[0].Args) != 2 || got[0].Args[1] != "@modelcontextprotocol/server-filesystem" {
				t.Fatalf("Resolve() Args = %#v, want copied args", got[0].Args)
			}
			if got[0].Env["ROOT"] != "/workspace" {
				t.Fatalf("Resolve() Env = %#v, want copied env", got[0].Env)
			}

			output := logs.String()
			if !strings.Contains(output, "level=INFO") {
				t.Fatalf("logs = %q, want info log", output)
			}
			if !strings.Contains(output, "skill_name="+tt.name+"-skill") {
				t.Fatalf("logs = %q, want skill name", output)
			}
			if !strings.Contains(output, "mcp_server=filesystem") {
				t.Fatalf("logs = %q, want MCP server name", output)
			}
			if !strings.Contains(output, "source="+skillSourceName(tt.source)) {
				t.Fatalf("logs = %q, want source %q", output, skillSourceName(tt.source))
			}
		})
	}
}

func TestMCPResolverResolveBlocksMarketplaceServerWithoutConsent(t *testing.T) {
	t.Parallel()

	resolver, logs := newResolverForTest(nil)

	got := resolver.Resolve([]*Skill{
		newSkillWithServer("marketplace-skill", SourceMarketplace, MCPServerDecl{
			Name:    "github",
			Command: "npx",
		}),
	})
	if got != nil {
		t.Fatalf("Resolve() = %#v, want nil", got)
	}

	output := logs.String()
	if !strings.Contains(output, "level=WARN") {
		t.Fatalf("logs = %q, want warn log", output)
	}
	if !strings.Contains(output, "skill_name=marketplace-skill") {
		t.Fatalf("logs = %q, want skill name", output)
	}
	if !strings.Contains(output, "mcp_server=github") {
		t.Fatalf("logs = %q, want MCP server name", output)
	}
	if !strings.Contains(output, "source=marketplace") {
		t.Fatalf("logs = %q, want marketplace source", output)
	}
}

func TestMCPResolverResolveAllowsMarketplaceServerWithConsent(t *testing.T) {
	t.Parallel()

	resolver, logs := newResolverForTest([]string{"marketplace-skill"})

	got := resolver.Resolve([]*Skill{
		newSkillWithServer("marketplace-skill", SourceMarketplace, MCPServerDecl{
			Name:    "github",
			Command: "npx",
		}),
	})
	if len(got) != 1 {
		t.Fatalf("Resolve() len = %d, want 1", len(got))
	}
	if got[0].Name != "github" || got[0].Command != "npx" {
		t.Fatalf("Resolve() server = %#v, want approved marketplace MCP server", got[0])
	}

	output := logs.String()
	if strings.Contains(output, "level=WARN") {
		t.Fatalf("logs = %q, want no warning log", output)
	}
	if !strings.Contains(output, "level=INFO") {
		t.Fatalf("logs = %q, want info log", output)
	}
}

func TestMCPResolverResolveSkipsSkillsWithoutMCPServers(t *testing.T) {
	t.Parallel()

	resolver, logs := newResolverForTest(nil)

	got := resolver.Resolve([]*Skill{
		{Meta: SkillMeta{Name: "empty"}, Source: SourceUser},
		nil,
	})
	if got != nil {
		t.Fatalf("Resolve() = %#v, want nil", got)
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func TestMCPResolverResolveDeduplicatesByServerNameUsingHigherPrecedenceSkill(t *testing.T) {
	t.Parallel()

	resolver, logs := newResolverForTest(nil)

	got := resolver.Resolve([]*Skill{
		newSkillWithServer("workspace-skill", SourceWorkspace, MCPServerDecl{
			Name:    "shared",
			Command: "workspace-cmd",
		}),
		newSkillWithServer("bundled-skill", SourceBundled, MCPServerDecl{
			Name:    "shared",
			Command: "bundled-cmd",
		}),
	})
	if len(got) != 1 {
		t.Fatalf("Resolve() len = %d, want 1", len(got))
	}
	if got[0].Command != "workspace-cmd" {
		t.Fatalf("Resolve() Command = %q, want workspace override", got[0].Command)
	}

	output := logs.String()
	if !strings.Contains(output, "skill_name=workspace-skill") {
		t.Fatalf("logs = %q, want winning skill logged", output)
	}
	if strings.Contains(output, "skill_name=bundled-skill") {
		t.Fatalf("logs = %q, want only final resolved skill logged", output)
	}
}

func TestMCPResolverResolveReturnsNilForEmptySkillList(t *testing.T) {
	t.Parallel()

	resolver, logs := newResolverForTest(nil)

	if got := resolver.Resolve(nil); got != nil {
		t.Fatalf("Resolve(nil) = %#v, want nil", got)
	}
	if got := resolver.Resolve([]*Skill{}); got != nil {
		t.Fatalf("Resolve(empty) = %#v, want nil", got)
	}
	if logs.Len() != 0 {
		t.Fatalf("logs = %q, want empty logs", logs.String())
	}
}

func newResolverForTest(allowed []string) (*MCPResolver, *bytes.Buffer) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	resolver := NewMCPResolver(aghconfig.SkillsConfig{
		AllowedMarketplaceMCP: allowed,
	}, logger)
	return resolver, &logs
}

func newSkillWithServer(name string, source SkillSource, server MCPServerDecl) *Skill {
	return &Skill{
		Meta: SkillMeta{
			Name:        name,
			Description: "test skill",
		},
		Source:     source,
		MCPServers: []MCPServerDecl{server},
	}
}
