package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHumanOutputProducesStyledTable(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listAgentsFn: func(_ context.Context, _ AgentQuery) ([]AgentRecord, error) {
			return []AgentRecord{{
				Name:        "coder",
				Provider:    "codex",
				Model:       "gpt-5.4",
				Tools:       []string{"shell", "edit"},
				Permissions: "approve-reads",
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "agent", "list", "-o", "human")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}
	if !strings.Contains(stdout, "Agents") || !strings.Contains(stdout, "Provider") ||
		!strings.Contains(stdout, "----") {
		t.Fatalf("human output = %q, want styled table", stdout)
	}
}

func TestJSONOutputProducesValidJSON(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listSessionsFn: func(_ context.Context, _ SessionListQuery) ([]SessionRecord, error) {
			return []SessionRecord{{
				ID:            "sess-1",
				Name:          "demo",
				AgentName:     "coder",
				WorkspaceID:   "ws-1",
				WorkspacePath: "/workspace/project",
				State:         "active",
				CreatedAt:     fixedTestNow.Add(-time.Minute),
				UpdatedAt:     fixedTestNow,
			}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "session", "list", "-o", "json")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}

	var decoded []SessionRecord
	if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(decoded) != 1 || decoded[0].ID != "sess-1" {
		t.Fatalf("decoded = %#v, want one session", decoded)
	}
}

func TestToonOutputProducesToonDocument(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		listAgentsFn: func(_ context.Context, _ AgentQuery) ([]AgentRecord, error) {
			return []AgentRecord{{Name: "coder", Provider: "codex", Tools: []string{"shell"}}}, nil
		},
	})

	stdout, _, err := executeRootCommand(t, deps, "agent", "list", "-o", "toon")
	if err != nil {
		t.Fatalf("executeRootCommand() error = %v", err)
	}
	if !strings.Contains(stdout, "agents[1]{name,provider,model,tool_count,permissions}:") {
		t.Fatalf("toon output = %q, want TOON header", stdout)
	}
}
