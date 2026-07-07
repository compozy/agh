package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/acp"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/session"
)

func TestLoopRuntimeSessionChannelShouldMatchNetworkGrammar(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		workspaceID string
		suffix      string
		want        string
	}{
		{
			name:        "Should preserve generated workspace ids",
			workspaceID: "ws_524b04478f9c1062",
			suffix:      "main",
			want:        "loop_ws_524b04478f9c1062_main",
		},
		{
			name:        "Should normalize invalid separators",
			workspaceID: "WS:Alpha/Primary",
			suffix:      "Review Gate:Approval",
			want:        "loop_ws_alpha_primary_review_gate_approval",
		},
		{
			name:        "Should fall back for empty fragments",
			workspaceID: " ",
			suffix:      " ",
			want:        "loop_workspace_main",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := loopRuntimeSessionChannel(looppkg.WorkspaceID(tc.workspaceID), tc.suffix)
			if got != tc.want {
				t.Fatalf("loopRuntimeSessionChannel() = %q, want %q", got, tc.want)
			}
			if err := network.ValidateChannel(got); err != nil {
				t.Fatalf("ValidateChannel(%q) error = %v", got, err)
			}
		})
	}
}

func TestLoopRuntimeSessionChannelShouldBoundLongHandles(t *testing.T) {
	t.Parallel()

	t.Run("Should bound long handles and keep workspace scoped prefix", func(t *testing.T) {
		t.Parallel()

		got := loopRuntimeSessionChannel(
			looppkg.WorkspaceID("ws_524b04478f9c1062"),
			strings.Repeat("long-node-handle-", 8),
		)
		if len(got) > 64 {
			t.Fatalf("loopRuntimeSessionChannel() length = %d, want <= 64: %q", len(got), got)
		}
		if err := network.ValidateChannel(got); err != nil {
			t.Fatalf("ValidateChannel(%q) error = %v", got, err)
		}
		if !strings.HasPrefix(got, "loop_ws_524b04478f9c1062_") {
			t.Fatalf("loopRuntimeSessionChannel() = %q, want workspace-scoped prefix", got)
		}
	})
}

func TestCollectLoopPromptResultShouldNotTreatProtocolRawAsStructuredOutput(t *testing.T) {
	t.Parallel()

	t.Run("Should ignore ACP protocol raw and preserve agent text", func(t *testing.T) {
		t.Parallel()

		manager := loopPromptResultSessionManager{
			events: []acp.AgentEvent{{
				Text: "```json\n{\"summary\":\"done\",\"message\":\"loop channel result\"}\n```",
				Raw:  json.RawMessage(`{"session_update":"agent_message_chunk"}`),
			}},
		}
		result, err := collectLoopPromptResult(
			context.Background(),
			manager,
			"sess-loop",
			looppkg.ActionPromptRequest{Message: "loop event probe"},
		)
		if err != nil {
			t.Fatalf("collectLoopPromptResult() error = %v", err)
		}
		if strings.TrimSpace(result.Text) == "" || !strings.Contains(result.Text, `"summary":"done"`) {
			t.Fatalf("collectLoopPromptResult() text = %q, want agent text", result.Text)
		}
		if len(result.Structured) != 0 {
			t.Fatalf("collectLoopPromptResult() structured = %s, want empty protocol raw ignored", result.Structured)
		}
	})
}

type loopPromptResultSessionManager struct {
	events []acp.AgentEvent
}

func (m loopPromptResultSessionManager) Create(context.Context, session.CreateOpts) (*session.Session, error) {
	return nil, errors.New("unexpected Create call")
}

func (m loopPromptResultSessionManager) Prompt(
	ctx context.Context,
	_ string,
	_ string,
) (<-chan acp.AgentEvent, error) {
	events := make(chan acp.AgentEvent, len(m.events))
	for _, event := range m.events {
		select {
		case events <- event:
		case <-ctx.Done():
			close(events)
			return nil, ctx.Err()
		}
	}
	close(events)
	return events, nil
}

func (m loopPromptResultSessionManager) CancelPrompt(context.Context, string) error {
	return errors.New("unexpected CancelPrompt call")
}
