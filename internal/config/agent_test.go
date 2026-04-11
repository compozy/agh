package config

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/frontmatter"
)

func TestParseAgentDefValidFrontmatterAndBody(t *testing.T) {
	agent, err := ParseAgentDef([]byte(`---
name: coder
provider: claude
model: claude-opus
tools: ["bash", "edit"]
permissions: approve-reads
mcp_servers:
  - name: github
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
---

You are a senior Go engineer.
`))
	if err != nil {
		t.Fatalf("ParseAgentDef() error = %v", err)
	}

	if agent.Name != "coder" || agent.Provider != "claude" || agent.Model != "claude-opus" {
		t.Fatalf("ParseAgentDef() = %#v", agent)
	}
	if len(agent.Tools) != 2 {
		t.Fatalf("ParseAgentDef() Tools = %#v", agent.Tools)
	}
	if !strings.Contains(agent.Prompt, "senior Go engineer") {
		t.Fatalf("ParseAgentDef() Prompt = %q", agent.Prompt)
	}
	if len(agent.MCPServers) != 1 || agent.MCPServers[0].Name != "github" {
		t.Fatalf("ParseAgentDef() MCPServers = %#v", agent.MCPServers)
	}
}

func TestParseAgentDefNormalizesCRLFAndPreservesConfigFrontmatterErrors(t *testing.T) {
	t.Parallel()

	agent, err := ParseAgentDef([]byte("---\r\nname: windows\r\nprovider: claude\r\n---\r\nPrompt on CRLF.\r\n"))
	if err != nil {
		t.Fatalf("ParseAgentDef() error = %v", err)
	}
	if got, want := agent.Prompt, "Prompt on CRLF."; got != want {
		t.Fatalf("ParseAgentDef() Prompt = %q, want %q", got, want)
	}

	if _, err := ParseAgentDef([]byte("plain markdown")); err == nil {
		t.Fatal("ParseAgentDef() missing frontmatter error = nil, want non-nil")
	} else if !errors.Is(err, ErrMissingAgentFrontmatter) || !errors.Is(err, frontmatter.ErrMissing) {
		t.Fatalf("ParseAgentDef() missing frontmatter error = %v, want mapped config + frontmatter sentinel", err)
	}

	if _, err := ParseAgentDef([]byte("---\nname: broken")); err == nil {
		t.Fatal("ParseAgentDef() unterminated frontmatter error = nil, want non-nil")
	} else if !errors.Is(err, ErrUnterminatedAgentFrontmatter) || !errors.Is(err, frontmatter.ErrUnterminated) {
		t.Fatalf("ParseAgentDef() unterminated frontmatter error = %v, want mapped config + frontmatter sentinel", err)
	}
}

func TestLoadAgentDefFromHomePath(t *testing.T) {
	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	writeFile(t, filepath.Join(homePaths.AgentsDir, "coder", agentDefName), `---
name: coder
provider: claude
---

You write reliable code.
`)

	agent, err := LoadAgentDef("coder", homePaths)
	if err != nil {
		t.Fatalf("LoadAgentDef() error = %v", err)
	}
	if agent.Name != "coder" || agent.Provider != "claude" {
		t.Fatalf("LoadAgentDef() = %#v", agent)
	}
}

func TestParseAgentDefMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing name",
			content: `---
provider: claude
---

prompt`,
		},
		{
			name: "missing prompt",
			content: `---
name: coder
provider: claude
---`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseAgentDef([]byte(tt.content)); err == nil {
				t.Fatal("ParseAgentDef() error = nil, want non-nil")
			}
		})
	}
}

func TestParseAgentDefAllowsMissingProvider(t *testing.T) {
	agent, err := ParseAgentDef([]byte(`---
name: general
---

You are the default agent.
`))
	if err != nil {
		t.Fatalf("ParseAgentDef() error = %v", err)
	}
	if agent.Provider != "" {
		t.Fatalf("ParseAgentDef() Provider = %q, want empty", agent.Provider)
	}
}

func TestParseAgentDefFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "missing frontmatter",
			content: "plain markdown",
		},
		{
			name: "unterminated frontmatter",
			content: `---
name: coder
provider: claude`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseAgentDef([]byte(tt.content)); err == nil {
				t.Fatal("ParseAgentDef() error = nil, want non-nil")
			}
		})
	}
}

func TestLoadAgentDefFileMissingReturnsError(t *testing.T) {
	if _, err := LoadAgentDefFile(filepath.Join(t.TempDir(), "missing", agentDefName)); err == nil {
		t.Fatal("LoadAgentDefFile() error = nil, want non-nil")
	}
}

func TestLoadAgentDefFileMergesMCPSidecar(t *testing.T) {
	t.Parallel()

	agentDir := filepath.Join(t.TempDir(), "coder")
	agentPath := filepath.Join(agentDir, agentDefName)
	writeFile(t, agentPath, `---
name: coder
provider: claude
mcp_servers:
  - name: inline-only
    command: inline-only-command
  - name: shared
    command: inline-shared
    args: ["--inline"]
---

Prompt.
`)
	writeFile(t, filepath.Join(agentDir, MCPJSONName), `{
  "mcpServers": {
    "shared": {
      "command": "sidecar-shared"
    },
    "sidecar-only": {
      "command": "sidecar-only-command"
    }
  }
}`)

	agent, err := LoadAgentDefFile(agentPath)
	if err != nil {
		t.Fatalf("LoadAgentDefFile() error = %v", err)
	}

	if got, want := len(agent.MCPServers), 3; got != want {
		t.Fatalf("LoadAgentDefFile() MCPServers len = %d, want %d (%#v)", got, want, agent.MCPServers)
	}
	if got, want := agent.MCPServers[1].Command, "sidecar-shared"; got != want {
		t.Fatalf("LoadAgentDefFile() shared.Command = %q, want %q", got, want)
	}
	if got := len(agent.MCPServers[1].Args); got != 0 {
		t.Fatalf("LoadAgentDefFile() shared.Args = %#v, want sidecar whole-object replacement", agent.MCPServers[1].Args)
	}
	if got, want := agent.MCPServers[2].Name, "sidecar-only"; got != want {
		t.Fatalf("LoadAgentDefFile() MCPServers[2].Name = %q, want %q", got, want)
	}
}

func TestLoadAgentDefRejectsBlankAndMismatchedNames(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	if _, err := LoadAgentDef("   ", homePaths); err == nil {
		t.Fatal("LoadAgentDef(blank) error = nil, want non-nil")
	}

	writeFile(t, filepath.Join(homePaths.AgentsDir, "coder", agentDefName), `---
name: reviewer
provider: claude
---

Mismatch
`)

	if _, err := LoadAgentDef("coder", homePaths); err == nil {
		t.Fatal("LoadAgentDef(mismatched name) error = nil, want non-nil")
	}
}

func TestWorkspaceDiscoveryRootsReturnsWorkspaceAdditionalGlobalOrder(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()

	roots := WorkspaceDiscoveryRoots(root, []string{additionalOne, additionalTwo}, homePaths)
	if got, want := len(roots), 4; got != want {
		t.Fatalf("len(WorkspaceDiscoveryRoots()) = %d, want %d", got, want)
	}

	if got, want := roots[0].Dir, root; got != want {
		t.Fatalf("roots[0].Dir = %q, want %q", got, want)
	}
	if got, want := roots[0].Source, WorkspaceDiscoverySourceWorkspace; got != want {
		t.Fatalf("roots[0].Source = %q, want %q", got, want)
	}
	if got, want := roots[1].Dir, additionalOne; got != want {
		t.Fatalf("roots[1].Dir = %q, want %q", got, want)
	}
	if got, want := roots[1].Source, WorkspaceDiscoverySourceAdditional; got != want {
		t.Fatalf("roots[1].Source = %q, want %q", got, want)
	}
	if got, want := roots[2].Dir, additionalTwo; got != want {
		t.Fatalf("roots[2].Dir = %q, want %q", got, want)
	}
	if got, want := roots[2].Source, WorkspaceDiscoverySourceAdditional; got != want {
		t.Fatalf("roots[2].Source = %q, want %q", got, want)
	}
	if got, want := roots[3].Dir, homePaths.HomeDir; got != want {
		t.Fatalf("roots[3].Dir = %q, want %q", got, want)
	}
	if got, want := roots[3].Source, WorkspaceDiscoverySourceGlobal; got != want {
		t.Fatalf("roots[3].Source = %q, want %q", got, want)
	}

	if got, want := roots[0].SkillsDir(), filepath.Join(root, DirName, SkillsDirName); got != want {
		t.Fatalf("roots[0].SkillsDir() = %q, want %q", got, want)
	}
	if got, want := roots[3].SkillsDir(), filepath.Join(homePaths.HomeDir, SkillsDirName); got != want {
		t.Fatalf("roots[3].SkillsDir() = %q, want %q", got, want)
	}
}

func TestLoadWorkspaceAgentDefsAppliesDocumentedPrecedence(t *testing.T) {
	t.Parallel()

	homePaths, err := ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	root := t.TempDir()
	additionalOne := t.TempDir()
	additionalTwo := t.TempDir()

	writeAgentDefinition(t, filepath.Join(homePaths.AgentsDir, "coder", agentDefName), "coder", "claude", "global")
	writeAgentDefinition(t, filepath.Join(homePaths.AgentsDir, "reviewer", agentDefName), "reviewer", "claude", "global-review")
	writeAgentDefinition(t, filepath.Join(additionalOne, DirName, AgentsDirName, "coder", agentDefName), "coder", "claude", "additional")
	writeAgentDefinition(t, filepath.Join(additionalOne, DirName, AgentsDirName, "pairer", agentDefName), "pairer", "claude", "additional-pair")
	writeAgentDefinition(t, filepath.Join(additionalTwo, DirName, AgentsDirName, "reviewer", agentDefName), "reviewer", "claude", "additional-review")
	writeAgentDefinition(t, filepath.Join(root, DirName, AgentsDirName, "coder", agentDefName), "coder", "claude", "workspace")

	agents, err := LoadWorkspaceAgentDefs(root, []string{additionalOne, additionalTwo}, homePaths)
	if err != nil {
		t.Fatalf("LoadWorkspaceAgentDefs() error = %v", err)
	}

	if got, want := agentModel(agents, "coder"), "workspace"; got != want {
		t.Fatalf("coder model = %q, want %q", got, want)
	}
	if got, want := agentModel(agents, "pairer"), "additional-pair"; got != want {
		t.Fatalf("pairer model = %q, want %q", got, want)
	}
	if got, want := agentModel(agents, "reviewer"), "additional-review"; got != want {
		t.Fatalf("reviewer model = %q, want %q", got, want)
	}
}

func writeAgentDefinition(t *testing.T, path string, name string, provider string, model string) {
	t.Helper()

	writeFile(t, path, `---
name: `+name+`
provider: `+provider+`
model: `+model+`
---

Prompt for `+name+`.
`)
}

func agentModel(agents []AgentDef, name string) string {
	for _, agent := range agents {
		if agent.Name == name {
			return agent.Model
		}
	}

	return ""
}
