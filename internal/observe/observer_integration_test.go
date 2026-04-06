//go:build integration

package observe

import (
	"github.com/pedronauck/agh/internal/testutil"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func TestObserverIntegrationFullFlow(t *testing.T) {
	h := newHarness(t)
	sess := newSession("sess-integration", session.StateActive, h.workspace, h.now)

	h.observer.OnSessionCreated(testutil.Context(t), sess)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "agent_message",
		TurnID:    "turn-int-1",
		Timestamp: h.now.Add(time.Minute),
		Text:      "assistant reply",
	})

	totalTokens := int64(9)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "done",
		TurnID:    "turn-int-1",
		Timestamp: h.now.Add(2 * time.Minute),
		Usage: &acp.TokenUsage{
			TurnID:      "turn-int-1",
			TotalTokens: &totalTokens,
			Timestamp:   h.now.Add(2 * time.Minute),
		},
	})

	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "permission",
		TurnID:    "turn-int-2",
		Timestamp: h.now.Add(3 * time.Minute),
		Action:    "session/request_permission",
		Resource:  h.workspace,
		Decision:  "allow",
	})

	sess.State = session.StateStopped
	sess.UpdatedAt = h.now.Add(4 * time.Minute)
	h.observer.OnSessionStopped(testutil.Context(t), sess)

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 3; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}

	stats, err := h.observer.QueryTokenStats(testutil.Context(t), store.TokenStatsQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryTokenStats() error = %v", err)
	}
	if got, want := len(stats), 1; got != want {
		t.Fatalf("len(stats) = %d, want %d", got, want)
	}
	if stats[0].TotalTokens == nil || *stats[0].TotalTokens != 9 {
		t.Fatalf("stats[0].TotalTokens = %#v, want 9", stats[0].TotalTokens)
	}

	permissions, err := h.observer.QueryPermissionLog(testutil.Context(t), store.PermissionLogQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryPermissionLog() error = %v", err)
	}
	if got, want := len(permissions), 1; got != want {
		t.Fatalf("len(permissions) = %d, want %d", got, want)
	}
	if permissions[0].Decision != "allow" {
		t.Fatalf("permissions[0].Decision = %q, want allow", permissions[0].Decision)
	}
}
