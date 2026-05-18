package session

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestPromptCallerCancellationClawpatch(t *testing.T) {
	t.Run("Should keep accepted prompt execution after caller context cancellation", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		driver := &promptContextCapturingDriver{fakeDriver: h.driver}
		h.manager = newManagerWithHarness(t, h, WithDriver(driver))
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil &&
				!errors.Is(err, ErrSessionNotFound) {
				t.Errorf("Stop(%q) cleanup error = %v", session.ID, err)
			}
		})

		source := make(chan acp.AgentEvent, 1)
		var turnID string
		h.driver.promptHook = func(_ *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			turnID = req.TurnID
			return source, nil
		}

		callerCtx, cancelCaller := context.WithCancel(testutil.Context(t))
		eventsCh, err := h.manager.Prompt(callerCtx, session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		providerCtx := driver.lastPromptContext(t)
		waitForCondition(t, "session prompting", func() bool {
			return session.IsPrompting()
		})

		cancelCaller()
		select {
		case <-callerCtx.Done():
		default:
			t.Fatal("caller context is still active after cancel")
		}
		select {
		case <-providerCtx.Done():
			t.Fatalf("provider context canceled with caller context: %v", providerCtx.Err())
		default:
		}
		if !session.IsPrompting() {
			t.Fatal("session prompting = false after caller cancellation, want active prompt execution")
		}

		source <- acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			SessionID: session.Info().ACPSessionID,
			TurnID:    turnID,
			Timestamp: time.Date(2026, 5, 17, 16, 0, 0, 0, time.UTC),
			Text:      "still running",
		}
		waitForCondition(t, "agent message persistence after caller cancellation", func() bool {
			events, queryErr := session.recorderHandle().Query(testutil.Context(t), store.EventQuery{})
			return queryErr == nil && countEventType(events, acp.EventTypeAgentMessage) == 1
		})

		if err := h.manager.CancelPrompt(testutil.Context(t), session.ID); err != nil {
			t.Fatalf("CancelPrompt() error = %v", err)
		}
		select {
		case <-providerCtx.Done():
		default:
			t.Fatal("provider context is still active after CancelPrompt()")
		}
		close(source)
		if events := collectEvents(t, eventsCh); len(events) != 0 {
			t.Fatalf("delivered events after caller cancellation = %d, want 0", len(events))
		}
		waitForCondition(t, "prompt state cleared after explicit cancellation", func() bool {
			return !session.IsPrompting()
		})
	})
}

type promptContextCapturingDriver struct {
	*fakeDriver
	mu       sync.Mutex
	contexts []context.Context
}

func (d *promptContextCapturingDriver) Prompt(
	ctx context.Context,
	proc *AgentProcess,
	req acp.PromptRequest,
) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.contexts = append(d.contexts, ctx)
	d.mu.Unlock()
	return d.fakeDriver.Prompt(ctx, proc, req)
}

func (d *promptContextCapturingDriver) lastPromptContext(t *testing.T) context.Context {
	t.Helper()

	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.contexts) == 0 {
		t.Fatal("driver prompt contexts = 0, want at least 1")
	}
	return d.contexts[len(d.contexts)-1]
}
