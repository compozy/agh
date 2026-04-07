package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentListAndInfoCommands(t *testing.T) {
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

	deps := newTestDeps(t, stubClient{
		listAgentsFn: func(context.Context) ([]AgentRecord, error) {
			return []AgentRecord{agent}, nil
		},
		getAgentFn: func(_ context.Context, name string) (AgentRecord, error) {
			if name != agent.Name {
				t.Fatalf("GetAgent() name = %q, want %q", name, agent.Name)
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
	if !strings.Contains(human, "Agent") || !strings.Contains(human, agent.Name) || !strings.Contains(human, "MCP Servers") {
		t.Fatalf("agent info human output = %q, want agent details", human)
	}

	toon, _, err := executeRootCommand(t, deps, "agent", "info", agent.Name, "-o", "toon")
	if err != nil {
		t.Fatalf("agent info toon error = %v", err)
	}
	if !strings.Contains(toon, "agent{name,provider,command,model,tools,permissions,prompt}:") || !strings.Contains(toon, agent.Name) {
		t.Fatalf("agent info toon output = %q, want TOON agent object", toon)
	}
}
