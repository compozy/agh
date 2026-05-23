package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
	"github.com/compozy/agh/internal/transcript"
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
	if got := messages[0].Role; got != transcript.UIRoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, transcript.UIRoleUser)
	}
	if got := messages[1].Role; got != transcript.UIRoleAssistant {
		t.Fatalf("messages[1].Role = %q, want %q", got, transcript.UIRoleAssistant)
	}
}

func TestManagerTranscriptIncludesSyntheticOriginMessages(t *testing.T) {
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
			TurnID:    "turn-user",
			Type:      acp.EventTypeUserMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"schema":"agh.session.event.v1","type":"user_message","text":"hello"}`,
			Timestamp: time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC),
		},
		{
			Sequence:  2,
			TurnID:    "turn-synth",
			Type:      acp.EventTypeSyntheticReentry,
			AgentName: session.Info().AgentName,
			Content:   `{"schema":"agh.session.event.v1","type":"synthetic_reentry","text":"daemon wake-up"}`,
			Timestamp: time.Date(2026, 4, 18, 13, 0, 1, 0, time.UTC),
		},
		{
			Sequence:  3,
			TurnID:    "turn-synth",
			Type:      acp.EventTypeAgentMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"schema":"agh.session.event.v1","type":"agent_message","text":"resuming work"}`,
			Timestamp: time.Date(2026, 4, 18, 13, 0, 2, 0, time.UTC),
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
	if len(messages) != 3 {
		t.Fatalf("Transcript() len = %d, want 3", len(messages))
	}
	if got := messages[0].Role; got != transcript.UIRoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, transcript.UIRoleUser)
	}
	if got := messages[1].Role; got != transcript.UIRoleSystem {
		t.Fatalf("messages[1].Role = %q, want %q", got, transcript.UIRoleSystem)
	}
	if got := transcript.UIMessageText(messages[1]); got != "daemon wake-up" {
		t.Fatalf("messages[1] text = %q, want %q", got, "daemon wake-up")
	}
	if got := messages[2].Role; got != transcript.UIRoleAssistant {
		t.Fatalf("messages[2].Role = %q, want %q", got, transcript.UIRoleAssistant)
	}
}

func TestManagerTranscriptReturnsStoredQueryErrors(t *testing.T) {
	t.Parallel()

	queryErr := errors.New("query failed")
	recorder := &queryRecorderStub{queryErr: queryErr}
	h := newHarness(t, WithStore(func(_ context.Context, _ string, _ string) (EventRecorder, error) {
		return recorder, nil
	}))
	writeStoppedSessionArtifacts(t, h, "stored-query-failure", true)

	_, err := h.manager.Transcript(testutil.Context(t), "stored-query-failure")
	if !errors.Is(err, queryErr) {
		t.Fatalf("Transcript() error = %v, want wrapped %v", err, queryErr)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder.closeCalls = %d, want 1", recorder.closeCalls)
	}
}

func TestManagerTranscriptLogsCleanupErrorsWithoutFailingSuccessfulRead(t *testing.T) {
	t.Parallel()

	recorder := &transcriptRecorderStub{
		queryRecorderStub: queryRecorderStub{
			events: []store.SessionEvent{{
				Sequence:  1,
				TurnID:    "turn-synth",
				Type:      acp.EventTypeSyntheticReentry,
				AgentName: "coder",
				Content:   `{"schema":"agh.session.event.v1","type":"synthetic_reentry","text":"daemon wake-up"}`,
				Timestamp: time.Date(2026, 4, 18, 13, 30, 0, 0, time.UTC),
			}},
		},
		closeErr: errors.New("close failed"),
	}
	h := newHarness(t, WithStore(func(_ context.Context, _ string, _ string) (EventRecorder, error) {
		return recorder, nil
	}))
	h.manager.logger = nil
	writeStoppedSessionArtifacts(t, h, "stored-cleanup-error", true)

	messages, err := h.manager.Transcript(testutil.Context(t), "stored-cleanup-error")
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Transcript() len = %d, want 1", len(messages))
	}
	if got := messages[0].Role; got != transcript.UIRoleSystem {
		t.Fatalf("messages[0].Role = %q, want %q", got, transcript.UIRoleSystem)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder.closeCalls = %d, want 1", recorder.closeCalls)
	}
}

type transcriptRecorderStub struct {
	queryRecorderStub
	closeErr error
}

func (s *transcriptRecorderStub) Close(context.Context) error {
	s.closeCalls++
	return s.closeErr
}
