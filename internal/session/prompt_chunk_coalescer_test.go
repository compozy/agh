package session

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
	"github.com/compozy/agh/internal/transcript"
)

func TestPromptChunkCoalescing(t *testing.T) {
	t.Parallel()

	t.Run("Should persist contiguous same-turn text chunks as one event row", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		h.driver.promptHook = func(proc *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent, 4)
			go func() {
				defer close(events)
				now := time.Now().UTC()
				for _, text := range []string{"hel", "lo ", "world"} {
					events <- acp.AgentEvent{
						Type:      acp.EventTypeAgentMessage,
						SessionID: proc.handle.SessionID,
						TurnID:    req.TurnID,
						Timestamp: now,
						Text:      text,
					}
				}
				events <- acp.AgentEvent{
					Type:      acp.EventTypeDone,
					SessionID: proc.handle.SessionID,
					TurnID:    req.TurnID,
					Timestamp: now.Add(time.Millisecond),
				}
			}()
			return events, nil
		}
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(context.Background(), session.ID); err != nil &&
				!errors.Is(err, ErrSessionNotFound) {
				t.Errorf("Stop(%q) error = %v", session.ID, err)
			}
		})

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		runtimeEvents := collectEvents(t, eventsCh)
		if got, want := countAgentEvents(runtimeEvents, acp.EventTypeAgentMessage), 3; got != want {
			t.Fatalf("agent_message runtime events = %d, want %d", got, want)
		}
		gotTexts := agentEventTexts(runtimeEvents, acp.EventTypeAgentMessage)
		wantTexts := []string{"hel", "lo ", "world"}
		if !slices.Equal(gotTexts, wantTexts) {
			t.Fatalf("agent_message runtime texts = %q, want %q", gotTexts, wantTexts)
		}

		stored := readStoredEvents(t, session)
		if got, want := countEventType(stored, acp.EventTypeAgentMessage), 1; got != want {
			t.Fatalf("agent_message stored rows = %d, want %d", got, want)
		}
		entries, err := h.manager.Transcript(testutil.Context(t), session.ID, store.EventQuery{})
		if err != nil {
			t.Fatalf("Transcript() error = %v", err)
		}
		if len(entries) == 0 || transcript.UIMessageText(entries[len(entries)-1].Message) != "hello world" {
			t.Fatalf("Transcript() = %#v, want coalesced assistant content", entries)
		}
	})

	t.Run("Should reprocess first chunk after a chunk run boundary", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		h.driver.promptHook = func(proc *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent, 5)
			go func() {
				defer close(events)
				now := time.Now().UTC()
				events <- acp.AgentEvent{
					Type:      acp.EventTypeAgentMessage,
					SessionID: proc.handle.SessionID,
					TurnID:    req.TurnID,
					Timestamp: now,
					Text:      "a",
				}
				for _, text := range []string{"b", "c"} {
					events <- acp.AgentEvent{
						Type:      acp.EventTypeThought,
						SessionID: proc.handle.SessionID,
						TurnID:    req.TurnID,
						Timestamp: now.Add(time.Millisecond),
						Text:      text,
					}
				}
				events <- acp.AgentEvent{
					Type:      acp.EventTypeDone,
					SessionID: proc.handle.SessionID,
					TurnID:    req.TurnID,
					Timestamp: now.Add(2 * time.Millisecond),
				}
			}()
			return events, nil
		}
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(context.Background(), session.ID); err != nil &&
				!errors.Is(err, ErrSessionNotFound) {
				t.Errorf("Stop(%q) error = %v", session.ID, err)
			}
		})

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		runtimeEvents := collectEvents(t, eventsCh)
		gotAgentMessages := agentEventTexts(runtimeEvents, acp.EventTypeAgentMessage)
		wantAgentMessages := []string{"a"}
		if !slices.Equal(gotAgentMessages, wantAgentMessages) {
			t.Fatalf("agent_message runtime texts = %q, want %q", gotAgentMessages, wantAgentMessages)
		}
		gotThoughts := agentEventTexts(runtimeEvents, acp.EventTypeThought)
		wantThoughts := []string{"b", "c"}
		if !slices.Equal(gotThoughts, wantThoughts) {
			t.Fatalf("thought runtime texts = %q, want %q", gotThoughts, wantThoughts)
		}

		stored := readStoredEvents(t, session)
		if got, want := countEventType(stored, acp.EventTypeAgentMessage), 1; got != want {
			t.Fatalf("agent_message stored rows = %d, want %d", got, want)
		}
		if got, want := countEventType(stored, acp.EventTypeThought), 1; got != want {
			t.Fatalf("thought stored rows = %d, want %d", got, want)
		}
		if got, want := storedTextByType(t, stored, acp.EventTypeThought), "bc"; got != want {
			t.Fatalf("thought stored text = %q, want %q", got, want)
		}
	})

	t.Run("Should publish chunk output when batch persistence fails", func(t *testing.T) {
		t.Parallel()

		recordErr := errors.New("batch persist failed")
		h := newHarness(t, WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return &failingBatchRecorder{batchErr: recordErr}, nil
		}))
		h.driver.promptHook = func(proc *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent, 2)
			go func() {
				defer close(events)
				now := time.Now().UTC()
				events <- acp.AgentEvent{
					Type:      acp.EventTypeAgentMessage,
					SessionID: proc.handle.SessionID,
					TurnID:    req.TurnID,
					Timestamp: now,
					Text:      "hidden ",
				}
				events <- acp.AgentEvent{
					Type:      acp.EventTypeAgentMessage,
					SessionID: proc.handle.SessionID,
					TurnID:    req.TurnID,
					Timestamp: now.Add(time.Millisecond),
					Text:      "chunk",
				}
			}()
			return events, nil
		}
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(context.Background(), session.ID); err != nil &&
				!errors.Is(err, ErrSessionNotFound) {
				t.Errorf("Stop(%q) error = %v", session.ID, err)
			}
		})

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		runtimeEvents := collectEvents(t, eventsCh)
		if got, want := countAgentEvents(runtimeEvents, acp.EventTypeAgentMessage), 2; got != want {
			t.Fatalf("agent_message runtime events = %d, want %d after batch persist failure", got, want)
		}
		gotTexts := agentEventTexts(runtimeEvents, acp.EventTypeAgentMessage)
		wantTexts := []string{"hidden ", "chunk"}
		if !slices.Equal(gotTexts, wantTexts) {
			t.Fatalf("agent_message runtime texts = %q, want %q", gotTexts, wantTexts)
		}
		notified := h.notifier.eventsForSession(session.ID)
		if got, want := countAgentEvents(notified, acp.EventTypeAgentMessage), 2; got != want {
			t.Fatalf("agent_message notifier events = %d, want %d after batch persist failure", got, want)
		}
	})
}

func countAgentEvents(events []acp.AgentEvent, want string) int {
	count := 0
	for _, event := range events {
		if event.Type == want {
			count++
		}
	}
	return count
}

func agentEventTexts(events []acp.AgentEvent, want string) []string {
	texts := make([]string, 0)
	for _, event := range events {
		if event.Type == want {
			texts = append(texts, event.Text)
		}
	}
	return texts
}

func storedTextByType(t *testing.T, events []store.SessionEvent, want string) string {
	t.Helper()

	for _, event := range events {
		if event.Type != want {
			continue
		}
		agentEvent, err := transcript.UnmarshalAgentEvent(event.Content)
		if err != nil {
			t.Fatalf("UnmarshalAgentEvent(%s) error = %v", event.Type, err)
		}
		return agentEvent.Text
	}
	t.Fatalf("stored event type %q not found", want)
	return ""
}

type failingBatchRecorder struct {
	batchErr error
}

func (r *failingBatchRecorder) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (r *failingBatchRecorder) RecordPersistedBatch(
	context.Context,
	[]store.SessionEvent,
) ([]store.SessionEvent, error) {
	return nil, r.batchErr
}

func (r *failingBatchRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (r *failingBatchRecorder) Query(context.Context, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (r *failingBatchRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (r *failingBatchRecorder) Close(context.Context) error {
	return nil
}
