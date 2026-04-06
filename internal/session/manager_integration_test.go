//go:build integration

package session

import (
	"github.com/pedronauck/agh/internal/testutil"
	"testing"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

func TestManagerIntegrationFullLifecycle(t *testing.T) {
	h := newHarness(t)

	session := createSession(t, h)
	firstPrompt, err := h.manager.Prompt(testutil.Context(t), session.ID, "first")
	if err != nil {
		t.Fatalf("Prompt(first) error = %v", err)
	}
	firstEvents := collectEvents(t, firstPrompt)
	if len(firstEvents) != 2 {
		t.Fatalf("first prompt events = %d, want 2", len(firstEvents))
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	secondPrompt, err := h.manager.Prompt(testutil.Context(t), resumed.ID, "second")
	if err != nil {
		t.Fatalf("Prompt(second) error = %v", err)
	}
	secondEvents := collectEvents(t, secondPrompt)
	if len(secondEvents) != 2 {
		t.Fatalf("second prompt events = %d, want 2", len(secondEvents))
	}

	if err := h.manager.Stop(testutil.Context(t), resumed.ID); err != nil {
		t.Fatalf("final Stop() error = %v", err)
	}

	reopened, err := store.OpenSessionDB(testutil.Context(t), resumed.ID, resumed.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		_ = reopened.Close(testutil.Context(t))
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopen) error = %v", err)
	}
	if len(events) != 8 {
		t.Fatalf("stored events = %d, want 8", len(events))
	}
	if !containsEventType(events, acp.EventTypeAgentMessage) || !containsEventType(events, acp.EventTypeDone) {
		t.Fatalf("stored events missing expected types: %#v", events)
	}
	if got := countEventType(events, EventTypeSessionStopped); got != 2 {
		t.Fatalf("stored %q events = %d, want 2", EventTypeSessionStopped, got)
	}

	meta := readMeta(t, resumed.MetaPath())
	if meta.State != string(StateStopped) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateStopped)
	}
}

func TestManagerIntegrationUsesRealSQLitePerSessionDB(t *testing.T) {
	h := newHarness(t)

	session := createSession(t, h)
	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "persist")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	_ = collectEvents(t, eventsCh)

	recorder, ok := session.recorderHandle().(*store.SessionDB)
	if !ok {
		t.Fatalf("recorder = %T, want *store.SessionDB", session.recorderHandle())
	}
	if got, want := recorder.Path(), session.DBPath(); got != want {
		t.Fatalf("SessionDB.Path() = %q, want %q", got, want)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	reopened, err := store.OpenSessionDB(testutil.Context(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		_ = reopened.Close(testutil.Context(t))
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopen) error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Query(reopen) returned 0 events, want persisted rows")
	}
}
