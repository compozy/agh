package session

import (
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/transcript"
)

func TestManagerTranscriptDelegatesToTranscriptAssembler(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
			t.Logf("h.manager.Stop failed for session %s: %v", session.ID, err)
		}
	})

	recorder := session.recorderHandle()
	events := []store.SessionEvent{
		{
			Sequence:  1,
			TurnID:    "turn-1",
			Type:      acp.EventTypeUserMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"schema":"agh.session.event.v1","type":"user_message","text":"hello"}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		{
			Sequence:  2,
			TurnID:    "turn-1",
			Type:      acp.EventTypeAgentMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"schema":"agh.session.event.v1","type":"agent_message","text":"hi"}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		},
	}
	for _, event := range events {
		if err := recorder.Record(testutil.Context(t), event); err != nil {
			t.Fatalf("Record(%s) error = %v", event.Type, err)
		}
	}

	messages, err := h.manager.Transcript(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("Transcript() len = %d, want 2", len(messages))
	}
	if got := messages[0].Role; got != transcript.RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, transcript.RoleUser)
	}
	if got := messages[1].Role; got != transcript.RoleAssistant {
		t.Fatalf("messages[1].Role = %q, want %q", got, transcript.RoleAssistant)
	}
}
