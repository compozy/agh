package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentListAndInfoCommands(t *testing.T) {
	t.Parallel()

	t.Run("Should list and inspect global agents", func(t *testing.T) {
		t.Parallel()

		agent := AgentRecord{
			Name:        "coder",
			Provider:    "fake",
			Command:     "codex",
			Model:       "gpt-5.4",
			Tools:       []string{"shell", "git"},
			Permissions: "standard",
			Prompt:      "You are coder.",
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

		human, _, err := executeRootCommand(t, deps, "agent", "info", agent.Name, "-o", "human")
		if err != nil {
			t.Fatalf("agent info human error = %v", err)
		}
		if !strings.Contains(human, "Agent") || !strings.Contains(human, agent.Name) ||
			!strings.Contains(human, "MCP Servers") {
			t.Fatalf("agent info human output = %q, want agent details", human)
		}

		toon, _, err := executeRootCommand(t, deps, "agent", "info", agent.Name, "-o", "toon")
		if err != nil {
			t.Fatalf("agent info toon error = %v", err)
		}
		if !strings.Contains(toon, "agent{name,provider,command,model,tools,permissions,prompt}:") ||
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
