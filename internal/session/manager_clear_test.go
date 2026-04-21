package session

import (
	"errors"
	"strings"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestClearConversationRestartsSameSessionWithFreshContext(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	firstEvents, err := h.manager.Prompt(testutil.Context(t), session.ID, "before clear")
	if err != nil {
		t.Fatalf("Prompt(before clear) error = %v", err)
	}
	_ = collectEvents(t, firstEvents)

	originalACP := session.Info().ACPSessionID

	cleared, err := h.manager.ClearConversation(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("ClearConversation() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), cleared.ID)
	})

	if got, want := cleared.ID, session.ID; got != want {
		t.Fatalf("cleared.ID = %q, want %q", got, want)
	}
	if got := cleared.Info().State; got != StateActive {
		t.Fatalf("cleared state = %q, want %q", got, StateActive)
	}
	if got := cleared.Info().ACPSessionID; got == "" || got == originalACP {
		t.Fatalf("cleared ACP session id = %q, want fresh non-empty id distinct from %q", got, originalACP)
	}
	if got := len(h.driver.startCalls); got != 2 {
		t.Fatalf("len(startCalls) = %d, want 2", got)
	}
	if got := h.driver.startCalls[1].ResumeSessionID; got != "" {
		t.Fatalf("clear restart ResumeSessionID = %q, want empty for fresh provider context", got)
	}

	messages, err := h.manager.Transcript(testutil.Context(t), cleared.ID)
	if err != nil {
		t.Fatalf("Transcript(after clear) error = %v", err)
	}
	if got := len(messages); got != 0 {
		t.Fatalf("Transcript(after clear) len = %d, want 0", got)
	}

	stored := readStoredEvents(t, cleared)
	if got := len(stored); got != 0 {
		t.Fatalf("stored events after clear = %d, want 0", got)
	}

	secondEvents, err := h.manager.Prompt(testutil.Context(t), cleared.ID, "after clear")
	if err != nil {
		t.Fatalf("Prompt(after clear) error = %v", err)
	}
	_ = collectEvents(t, secondEvents)

	stored = readStoredEvents(t, cleared)
	if got := len(stored); got == 0 {
		t.Fatal("stored events after second prompt = 0, want persisted prompt data")
	}
	for _, event := range stored {
		if strings.Contains(event.Content, "before clear") {
			t.Fatalf("stored event content still contains cleared prompt: %s", event.Content)
		}
	}
}

func TestClearConversationRejectsPromptInProgress(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	releasePrompt := make(chan struct{})
	h.driver.promptHook = func(_ *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		events := make(chan acp.AgentEvent)
		go func() {
			defer close(events)
			<-releasePrompt
			events <- acp.AgentEvent{
				Type:      acp.EventTypeDone,
				SessionID: session.Info().ACPSessionID,
				TurnID:    req.TurnID,
			}
		}()
		return events, nil
	}

	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	waitForCondition(t, "prompt setup", func() bool {
		return session.IsPrompting()
	})

	_, err = h.manager.ClearConversation(testutil.Context(t), session.ID)
	if !errors.Is(err, ErrPromptInProgress) {
		t.Fatalf("ClearConversation() error = %v, want %v", err, ErrPromptInProgress)
	}

	close(releasePrompt)
	_ = collectEvents(t, eventsCh)
	if stopErr := h.manager.Stop(testutil.Context(t), session.ID); stopErr != nil {
		t.Fatalf("cleanup Stop() error = %v", stopErr)
	}
}
