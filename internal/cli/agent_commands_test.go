package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
)

func TestAgentListAndInfoCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should list and inspect global agents", func(t *testing.T) {
		t.Parallel()

		agent := AgentRecord{
			Name:         "coder",
			Provider:     "fake",
			Command:      "codex",
			Model:        "gpt-5.4",
			Tools:        []string{"shell", "git"},
			Permissions:  "standard",
			CategoryPath: []string{"Marketing", "Sales"},
			Prompt:       "You are coder.",
			MCPServers: []AgentMCPServer{{
				Name:    "github",
				Command: "agh-github",
				Args:    []string{"serve"},
			}},
		}

		deps := newTestDeps(t, &stubClient{
			listAgentsFn: func(_ context.Context, query AgentQuery) ([]AgentRecord, error) {
				if query.Workspace != "" {
					t.Fatalf("ListAgents() workspace = %q, want empty", query.Workspace)
				}
				return []AgentRecord{agent}, nil
			},
			getAgentFn: func(_ context.Context, name string, query AgentQuery) (AgentRecord, error) {
				if name != agent.Name {
					t.Fatalf("GetAgent() name = %q, want %q", name, agent.Name)
				}
				if query.Workspace != "" {
					t.Fatalf("GetAgent() workspace = %q, want empty", query.Workspace)
				}
				return agent, nil
			},
		})

		stdout, _, err := executeRootCommand(t, deps, "agent", "list", "-o", "json")
		if err != nil {
			t.Fatalf("agent list error = %v", err)
		}

		var listed []AgentRecord
		if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
			t.Fatalf("json.Unmarshal(agent list) error = %v", err)
		}
		if len(listed) != 1 || listed[0].Name != agent.Name {
			t.Fatalf("listed agents = %#v, want one %q record", listed, agent.Name)
		}
		if got, want := strings.Join(listed[0].CategoryPath, ","), "Marketing,Sales"; got != want {
			t.Fatalf("listed agent category_path = %#v, want %q", listed[0].CategoryPath, want)
		}

		listHuman, _, err := executeRootCommand(t, deps, "agent", "list", "-o", "human")
		if err != nil {
			t.Fatalf("agent list human error = %v", err)
		}
		if !strings.Contains(listHuman, "Category") || !strings.Contains(listHuman, "Marketing / Sales") {
			t.Fatalf("agent list human output = %q, want category column", listHuman)
		}

		listToon, _, err := executeRootCommand(t, deps, "agent", "list", "-o", "toon")
		if err != nil {
			t.Fatalf("agent list toon error = %v", err)
		}
		if !strings.Contains(listToon, "agents[1]{name,provider,model,category,tool_count,permissions}:") ||
			!strings.Contains(listToon, "Marketing / Sales") {
			t.Fatalf("agent list toon output = %q, want category key", listToon)
		}

		human, _, err := executeRootCommand(t, deps, "agent", "info", agent.Name, "-o", "human")
		if err != nil {
			t.Fatalf("agent info human error = %v", err)
		}
		if !strings.Contains(human, "Agent") || !strings.Contains(human, agent.Name) ||
			!strings.Contains(human, "MCP Servers") ||
			!strings.Contains(human, "Marketing / Sales") {
			t.Fatalf("agent info human output = %q, want agent details", human)
		}

		toon, _, err := executeRootCommand(t, deps, "agent", "info", agent.Name, "-o", "toon")
		if err != nil {
			t.Fatalf("agent info toon error = %v", err)
		}
		if !strings.Contains(toon, "agent{name,provider,command,model,category,tools,permissions,prompt}:") ||
			!strings.Contains(toon, agent.Name) {
			t.Fatalf("agent info toon output = %q, want TOON agent object", toon)
		}
	})
}

func TestAgentCommandsPassWorkspaceQuery(t *testing.T) {
	t.Parallel()

	t.Run("Should pass workspace query to list and info calls", func(t *testing.T) {
		t.Parallel()

		const workspace = "ws-test"
		agent := AgentRecord{Name: "founder", Provider: "codex", Prompt: "lead"}
		deps := newTestDeps(t, &stubClient{
			listAgentsFn: func(_ context.Context, query AgentQuery) ([]AgentRecord, error) {
				if query.Workspace != workspace {
					t.Fatalf("ListAgents() workspace = %q, want %q", query.Workspace, workspace)
				}
				return []AgentRecord{agent}, nil
			},
			getAgentFn: func(_ context.Context, name string, query AgentQuery) (AgentRecord, error) {
				if name != agent.Name {
					t.Fatalf("GetAgent() name = %q, want %q", name, agent.Name)
				}
				if query.Workspace != workspace {
					t.Fatalf("GetAgent() workspace = %q, want %q", query.Workspace, workspace)
				}
				return agent, nil
			},
		})

		stdout, _, err := executeRootCommand(t, deps, "agent", "list", "--workspace", workspace, "-o", "json")
		if err != nil {
			t.Fatalf("agent list --workspace error = %v", err)
		}
		if !strings.Contains(stdout, agent.Name) {
			t.Fatalf("agent list --workspace output = %q, want %q", stdout, agent.Name)
		}

		stdout, _, err = executeRootCommand(
			t,
			deps,
			"agent",
			"info",
			agent.Name,
			"--workspace",
			workspace,
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("agent info --workspace error = %v", err)
		}
		if !strings.Contains(stdout, agent.Name) {
			t.Fatalf("agent info --workspace output = %q, want %q", stdout, agent.Name)
		}
	})
}

func TestAgentCreateCommand(t *testing.T) {
	t.Parallel()

	t.Run("Should create a workspace-local agent definition", func(t *testing.T) {
		t.Parallel()

		workspaceRoot := t.TempDir()
		const workspaceRef = "ws-alpha"
		deps := newTestDeps(t, &stubClient{
			getWorkspaceFn: func(_ context.Context, ref string) (WorkspaceDetailRecord, error) {
				if ref != workspaceRef {
					t.Fatalf("GetWorkspace() ref = %q, want %q", ref, workspaceRef)
				}
				return WorkspaceDetailRecord{
					Workspace: WorkspaceRecord{
						ID:      workspaceRef,
						RootDir: workspaceRoot,
						Name:    "Alpha",
					},
				}, nil
			},
		})

		stdout, _, err := executeRootCommand(
			t,
			deps,
			"agent",
			"create",
			"pricing_strategist",
			"--workspace",
			workspaceRef,
			"--provider",
			"claude",
			"--model",
			"claude-sonnet-4-6",
			"--prompt",
			"You own Ad8 pricing strategy.",
			"--tool",
			"builtin__shell",
			"--category",
			"Strategy",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("agent create error = %v", err)
		}
		var created AgentRecord
		if err := json.Unmarshal([]byte(stdout), &created); err != nil {
			t.Fatalf("json.Unmarshal(agent create) error = %v", err)
		}
		if created.Name != "pricing_strategist" || created.Provider != "claude" ||
			created.Model != "claude-sonnet-4-6" || created.Prompt != "You own Ad8 pricing strategy." {
			t.Fatalf("created agent = %#v, want pricing strategist metadata", created)
		}

		path := filepath.Join(
			workspaceRoot,
			aghconfig.DirName,
			aghconfig.AgentsDirName,
			"pricing_strategist",
			"AGENT.md",
		)
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat(created AGENT.md) error = %v", err)
		}
		if fileInfo.Mode().Perm() != 0o600 {
			t.Fatalf("created AGENT.md mode = %v, want 0600", fileInfo.Mode().Perm())
		}
		loaded, err := aghconfig.LoadAgentDefFile(path)
		if err != nil {
			t.Fatalf("LoadAgentDefFile(created AGENT.md) error = %v", err)
		}
		if len(loaded.Tools) != 1 || loaded.Name != created.Name || loaded.Provider != created.Provider ||
			loaded.Tools[0] != "builtin__shell" {
			t.Fatalf("loaded agent = %#v, want created agent definition", loaded)
		}
	})
}

func TestAgentWorkspaceFlagRejectsEmptyExplicitValue(t *testing.T) {
	t.Parallel()

	t.Run("Should reject an explicitly blank workspace flag", func(t *testing.T) {
		t.Parallel()

		deps := newTestDeps(t, &stubClient{})
		_, _, err := executeRootCommand(t, deps, "agent", "list", "--workspace", " ")
		if err == nil {
			t.Fatal("agent list --workspace blank error = nil, want error")
		}
		if !strings.Contains(err.Error(), "workspace flag cannot be empty") {
			t.Fatalf("agent list --workspace blank error = %v, want workspace flag message", err)
		}
	})
}
