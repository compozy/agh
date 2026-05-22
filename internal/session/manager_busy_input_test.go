package session

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestManagerBusyInputQueue(t *testing.T) {
	t.Run("Should queue busy user input and dispatch it after the active turn ends", func(t *testing.T) {
		t.Parallel()

		queueStore := openManagerInputQueueStore(t)
		h := newHarness(
			t,
			WithSessionInputQueueStore(queueStore),
			WithSessionBusyInputConfig(aghconfig.SessionBusyInputConfig{
				DefaultMode:  string(BusyInputModeQueue),
				QueueCap:     3,
				MaxTextBytes: 4096,
			}),
		)
		registerManagerInputQueueWorkspace(t, queueStore, h)
		sess := createSession(t, h)
		registerManagerInputQueueSession(t, queueStore, h, sess)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), sess.ID); err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		})

		firstPromptEntered := make(chan struct{})
		releaseFirstPrompt := make(chan struct{})
		secondPromptEntered := make(chan struct{})
		var releaseOnce sync.Once
		t.Cleanup(func() {
			releaseOnce.Do(func() {
				close(releaseFirstPrompt)
			})
		})
		h.driver.promptHook = func(_ *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent)
			go func() {
				defer close(events)
				switch req.Message {
				case "first prompt":
					close(firstPromptEntered)
					<-releaseFirstPrompt
				case "queued prompt":
					close(secondPromptEntered)
				}
				emitDonePromptEvents(events, sess.ID, req.TurnID)
			}()
			return events, nil
		}

		firstEvents, err := h.manager.SendPrompt(testutil.Context(t), sess.ID, SendPromptOpts{
			Message: "first prompt",
		})
		if err != nil {
			t.Fatalf("SendPrompt(first) error = %v", err)
		}
		if firstEvents.Events == nil {
			t.Fatal("SendPrompt(first).Events = nil, want accepted stream")
		}
		<-firstPromptEntered

		queued, err := h.manager.SendPrompt(testutil.Context(t), sess.ID, SendPromptOpts{
			Message: "queued prompt",
			Mode:    BusyInputModeQueue,
		})
		if err != nil {
			t.Fatalf("SendPrompt(queue) error = %v", err)
		}
		if !queued.Queued || queued.Status != "queued" || queued.QueueEntryID == "" || queued.QueueGeneration != 0 {
			t.Fatalf("queued result = %#v, want queued generation 0", queued)
		}
		if got := len(managerPromptCalls(h)); got != 1 {
			t.Fatalf("len(promptCalls) while first prompt active = %d, want 1", got)
		}

		releaseOnce.Do(func() {
			close(releaseFirstPrompt)
		})
		_ = collectEvents(t, firstEvents.Events)
		waitForCondition(t, "queued prompt dispatch", func() bool {
			return len(managerPromptCalls(h)) == 2
		})
		<-secondPromptEntered
		promptCalls := managerPromptCalls(h)
		if got := promptCalls[1].Message; got != "queued prompt" {
			t.Fatalf("queued dispatch message = %q, want queued prompt", got)
		}
		if got := promptCalls[1].Meta.TurnSource; got != acp.PromptTurnSourceUser {
			t.Fatalf("queued dispatch turn source = %q, want user", got)
		}
	})
}

func TestManagerBusyInputInterrupt(t *testing.T) {
	t.Run("Should advance generation cancel stale queue and send replacement prompt", func(t *testing.T) {
		t.Parallel()

		queueStore := openManagerInputQueueStore(t)
		h := newHarness(
			t,
			WithSessionInputQueueStore(queueStore),
			WithSessionBusyInputConfig(aghconfig.SessionBusyInputConfig{
				DefaultMode:  string(BusyInputModeQueue),
				QueueCap:     3,
				MaxTextBytes: 4096,
			}),
		)
		registerManagerInputQueueWorkspace(t, queueStore, h)
		sess := createSession(t, h)
		registerManagerInputQueueSession(t, queueStore, h, sess)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), sess.ID); err != nil {
				t.Errorf("Stop() error = %v", err)
			}
		})

		firstPromptEntered := make(chan struct{})
		releaseFirstPrompt := make(chan struct{})
		var releaseOnce sync.Once
		t.Cleanup(func() {
			releaseOnce.Do(func() {
				close(releaseFirstPrompt)
			})
		})
		h.driver.cancelHook = func(*fakeProcess) error {
			releaseOnce.Do(func() {
				close(releaseFirstPrompt)
			})
			return nil
		}
		h.driver.promptHook = func(_ *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
			events := make(chan acp.AgentEvent)
			go func() {
				defer close(events)
				if req.Message == "first prompt" {
					close(firstPromptEntered)
					<-releaseFirstPrompt
				}
				emitDonePromptEvents(events, sess.ID, req.TurnID)
			}()
			return events, nil
		}

		firstEvents, err := h.manager.SendPrompt(testutil.Context(t), sess.ID, SendPromptOpts{
			Message: "first prompt",
		})
		if err != nil {
			t.Fatalf("SendPrompt(first) error = %v", err)
		}
		<-firstPromptEntered
		queued, err := h.manager.SendPrompt(testutil.Context(t), sess.ID, SendPromptOpts{
			Message: "stale queued prompt",
			Mode:    BusyInputModeQueue,
		})
		if err != nil {
			t.Fatalf("SendPrompt(stale queue) error = %v", err)
		}
		if !queued.Queued {
			t.Fatalf("queued result = %#v, want queued", queued)
		}

		interrupted, err := h.manager.SendPrompt(testutil.Context(t), sess.ID, SendPromptOpts{
			Message: "replacement prompt",
			Mode:    BusyInputModeInterrupt,
		})
		if err != nil {
			t.Fatalf("SendPrompt(interrupt) error = %v", err)
		}
		if !interrupted.Interrupted || interrupted.QueueGeneration != 1 || interrupted.CanceledQueuedEntries != 1 {
			t.Fatalf("interrupted result = %#v, want generation 1 with one canceled queue entry", interrupted)
		}
		if interrupted.Events == nil {
			t.Fatal("SendPrompt(interrupt).Events = nil, want replacement stream")
		}
		_ = collectEvents(t, firstEvents.Events)
		_ = collectEvents(t, interrupted.Events)
		promptCalls := managerPromptCalls(h)
		messages := make([]string, 0, len(promptCalls))
		for _, call := range promptCalls {
			messages = append(messages, call.Message)
		}
		if !slices.Equal(messages, []string{"first prompt", "replacement prompt"}) {
			t.Fatalf("prompt messages = %#v, want first then replacement without stale queue", messages)
		}
	})
}

func managerPromptCalls(h *harness) []acp.PromptRequest {
	h.driver.mu.Lock()
	defer h.driver.mu.Unlock()
	return append([]acp.PromptRequest(nil), h.driver.promptCalls...)
}

func emitDonePromptEvents(events chan<- acp.AgentEvent, sessionID string, turnID string) {
	ts := time.Now().UTC()
	events <- acp.AgentEvent{
		Type:      acp.EventTypeAgentMessage,
		SessionID: sessionID,
		TurnID:    turnID,
		Timestamp: ts,
		Text:      "reply",
	}
	events <- acp.AgentEvent{
		Type:       acp.EventTypeDone,
		SessionID:  sessionID,
		TurnID:     turnID,
		Timestamp:  ts,
		StopReason: "end_turn",
	}
}

func openManagerInputQueueStore(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	ctx := testutil.Context(t)
	queueStore, err := globaldb.OpenGlobalDB(ctx, filepath.Join(t.TempDir(), store.GlobalDatabaseName))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := queueStore.Close(ctx); err != nil {
			t.Errorf("Close(globalDB) error = %v", err)
		}
	})
	return queueStore
}

func registerManagerInputQueueWorkspace(t *testing.T, queueStore *globaldb.GlobalDB, h *harness) {
	t.Helper()

	if err := queueStore.InsertWorkspace(testutil.Context(t), workspacepkg.Workspace{
		ID:        h.workspaceID,
		RootDir:   h.workspace,
		Name:      h.workspaceName,
		CreatedAt: time.Date(2026, 5, 21, 11, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 5, 21, 11, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
}

func registerManagerInputQueueSession(
	t *testing.T,
	queueStore *globaldb.GlobalDB,
	h *harness,
	sess *Session,
) {
	t.Helper()

	if err := queueStore.RegisterSession(testutil.Context(t), store.SessionInfo{
		ID:          sess.ID,
		Name:        "Input Queue",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: h.workspaceID,
		SessionType: string(SessionTypeUser),
		State:       string(StateActive),
		CreatedAt:   time.Date(2026, 5, 21, 11, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 5, 21, 11, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}
}
