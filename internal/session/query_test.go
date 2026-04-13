package session

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerListAllRequiresContext(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	var nilCtx context.Context
	if _, err := h.manager.ListAll(nilCtx); err == nil {
		t.Fatal("ListAll(nil) error = nil, want non-nil")
	}
}

func TestManagerListAllReturnsActiveWhenSessionsDirMissing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if err := os.RemoveAll(h.homePaths.SessionsDir); err != nil {
		t.Fatalf("RemoveAll(sessions dir) error = %v", err)
	}

	infos, err := h.manager.ListAll(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("ListAll() = %d sessions, want 1", len(infos))
	}
	if got := infos[0].ID; got != session.ID {
		t.Fatalf("ListAll()[0].ID = %q, want %q", got, session.ID)
	}
	if got := infos[0].State; got != StateActive {
		t.Fatalf("ListAll()[0].State = %q, want %q", got, StateActive)
	}
}

func TestManagerListAllMergesActiveAndStoppedSessions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	active := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), active.ID)
	})

	stopped, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked-stopped",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create(networked stopped) error = %v", err)
	}
	if err := h.manager.Stop(testutil.Context(t), stopped.ID); err != nil {
		t.Fatalf("Stop(stopped) error = %v", err)
	}

	orphanDir := filepath.Join(h.homePaths.SessionsDir, "orphan")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(orphan) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(h.homePaths.SessionsDir, "notes.txt"), []byte("skip me"), 0o644); err != nil {
		t.Fatalf("WriteFile(notes) error = %v", err)
	}
	if err := os.WriteFile(active.MetaPath(), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(corrupt active meta) error = %v", err)
	}

	badStoppedDir := filepath.Join(h.homePaths.SessionsDir, "bad-stopped")
	if err := os.MkdirAll(badStoppedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bad-stopped) error = %v", err)
	}
	if err := os.WriteFile(store.SessionMetaFile(badStoppedDir), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(corrupt stopped meta) error = %v", err)
	}

	infos, err := h.manager.ListAll(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("ListAll() = %d sessions, want 2", len(infos))
	}

	if got := infos[0].ID; got != active.ID {
		t.Fatalf("ListAll()[0].ID = %q, want %q", got, active.ID)
	}
	if got := infos[0].State; got != StateActive {
		t.Fatalf("ListAll()[0].State = %q, want %q", got, StateActive)
	}
	if got := infos[0].Channel; got != "" {
		t.Fatalf("ListAll()[0].Channel = %q, want empty", got)
	}
	if got := infos[1].ID; got != stopped.ID {
		t.Fatalf("ListAll()[1].ID = %q, want %q", got, stopped.ID)
	}
	if got := infos[1].State; got != StateStopped {
		t.Fatalf("ListAll()[1].State = %q, want %q", got, StateStopped)
	}
	if got := infos[1].Channel; got != "builders" {
		t.Fatalf("ListAll()[1].Channel = %q, want %q", got, "builders")
	}
}

func TestManagerStatusReturnsActiveAndStoredSessions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	info, err := h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(active) error = %v", err)
	}
	if got := info.State; got != StateActive {
		t.Fatalf("Status(active).State = %q, want %q", got, StateActive)
	}
	if got := info.Channel; got != "" {
		t.Fatalf("Status(active).Channel = %q, want empty", got)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	info, err = h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(stopped) error = %v", err)
	}
	if got := info.State; got != StateStopped {
		t.Fatalf("Status(stopped).State = %q, want %q", got, StateStopped)
	}
	if got := info.Channel; got != "" {
		t.Fatalf("Status(stopped).Channel = %q, want empty", got)
	}

	var nilCtx context.Context
	if _, err := h.manager.Status(nilCtx, session.ID); err == nil {
		t.Fatal("Status(nil, id) error = nil, want non-nil")
	}
	if _, err := h.manager.Status(testutil.Context(t), "   "); err == nil {
		t.Fatal("Status(blank id) error = nil, want non-nil")
	}
	if _, err := h.manager.Status(testutil.Context(t), "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Status(missing) error = %v, want ErrSessionNotFound", err)
	}
}

func TestManagerStatusRepairsIncompleteStartMetadata(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	meta, err := store.ReadSessionMeta(session.MetaPath())
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	meta.State = string(StateStarting)
	meta.StopReason = nil
	meta.StopDetail = ""
	meta.ACPSessionID = stringPointer(originalACP)
	if err := store.WriteSessionMeta(session.MetaPath(), meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	info, err := h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(repaired) error = %v", err)
	}
	if got := info.State; got != StateStopped {
		t.Fatalf("Status(repaired).State = %q, want %q", got, StateStopped)
	}
	if got := info.StopReason; got != store.StopError {
		t.Fatalf("Status(repaired).StopReason = %q, want %q", got, store.StopError)
	}
	if got := info.StopDetail; got != "start did not complete" {
		t.Fatalf("Status(repaired).StopDetail = %q, want %q", got, "start did not complete")
	}
	if got := info.ACPSessionID; got != "" {
		t.Fatalf("Status(repaired).ACPSessionID = %q, want empty", got)
	}

	repairedMeta, err := store.ReadSessionMeta(session.MetaPath())
	if err != nil {
		t.Fatalf("ReadSessionMeta(repaired) error = %v", err)
	}
	if got := repairedMeta.State; got != string(StateStopped) {
		t.Fatalf("repaired meta state = %q, want %q", got, StateStopped)
	}
	if repairedMeta.StopReason == nil || *repairedMeta.StopReason != store.StopError {
		t.Fatalf("repaired meta stop reason = %#v, want %q", repairedMeta.StopReason, store.StopError)
	}
	if got := repairedMeta.StopDetail; got != "start did not complete" {
		t.Fatalf("repaired meta stop detail = %q, want %q", got, "start did not complete")
	}
	if repairedMeta.ACPSessionID != nil {
		t.Fatalf("repaired meta ACPSessionID = %#v, want nil", repairedMeta.ACPSessionID)
	}
}

func TestManagerEventsAndHistoryUseStoredEvents(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	runtimeEvents := collectEvents(t, eventsCh)
	if len(runtimeEvents) != 2 {
		t.Fatalf("Prompt() events = %d, want 2", len(runtimeEvents))
	}

	activeEvents, err := h.manager.Events(testutil.Context(t), session.ID, store.EventQuery{})
	if err != nil {
		t.Fatalf("Events(active) error = %v", err)
	}
	if len(activeEvents) != 3 {
		t.Fatalf("Events(active) = %d events, want 3", len(activeEvents))
	}
	activeHistory, err := h.manager.History(testutil.Context(t), session.ID, store.EventQuery{})
	if err != nil {
		t.Fatalf("History(active) error = %v", err)
	}
	if len(activeHistory) != 1 {
		t.Fatalf("History(active) = %d turns, want 1", len(activeHistory))
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	stopOnly, err := h.manager.Events(testutil.Context(t), session.ID, store.EventQuery{
		Type:  EventTypeSessionStopped,
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("Events(stopOnly) error = %v", err)
	}
	if len(stopOnly) != 1 {
		t.Fatalf("Events(stopOnly) = %d events, want 1", len(stopOnly))
	}
	if got := stopOnly[0].Type; got != EventTypeSessionStopped {
		t.Fatalf("Events(stopOnly)[0].Type = %q, want %q", got, EventTypeSessionStopped)
	}
	if got := countEventType(stopOnly, EventTypeSessionStopped); got != 1 {
		t.Fatalf("Events(stopOnly) %q count = %d, want 1", EventTypeSessionStopped, got)
	}

	afterPrompt, err := h.manager.Events(testutil.Context(t), session.ID, store.EventQuery{
		AfterSequence: activeEvents[len(activeEvents)-1].Sequence,
	})
	if err != nil {
		t.Fatalf("Events(after prompt) error = %v", err)
	}
	if len(afterPrompt) != 1 {
		t.Fatalf("Events(after prompt) = %d events, want 1", len(afterPrompt))
	}
	if got := afterPrompt[0].Type; got != EventTypeSessionStopped {
		t.Fatalf("Events(after prompt)[0].Type = %q, want %q", got, EventTypeSessionStopped)
	}

	stoppedHistory, err := h.manager.History(testutil.Context(t), session.ID, store.EventQuery{})
	if err != nil {
		t.Fatalf("History(stopped) error = %v", err)
	}
	if len(stoppedHistory) != 2 {
		t.Fatalf("History(stopped) = %d turns, want 2", len(stoppedHistory))
	}
	if got := stoppedHistory[1].Events[0].Type; got != EventTypeSessionStopped {
		t.Fatalf("History(stopped)[1] first event type = %q, want %q", got, EventTypeSessionStopped)
	}
}

func TestManagerOpenQueryRecorderValidationAndCleanup(t *testing.T) {
	t.Parallel()

	t.Run("requires context and session id", func(t *testing.T) {
		h := newHarness(t)
		var nilCtx context.Context
		if _, _, err := h.manager.openQueryRecorder(nilCtx, "sess-1"); err == nil {
			t.Fatal("openQueryRecorder(nil, id) error = nil, want non-nil")
		}
		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), "   "); err == nil {
			t.Fatal("openQueryRecorder(ctx, blank) error = nil, want non-nil")
		}
	})

	t.Run("active session requires recorder", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)
		t.Cleanup(func() {
			_ = h.manager.Stop(testutil.Context(t), session.ID)
		})

		session.setRecorder(nil)
		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), session.ID); err == nil {
			t.Fatal("openQueryRecorder(active with nil recorder) error = nil, want non-nil")
		}
	})

	t.Run("missing session metadata", func(t *testing.T) {
		h := newHarness(t)
		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), "missing"); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("openQueryRecorder(missing) error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("missing database file", func(t *testing.T) {
		h := newHarness(t)
		writeStoppedSessionArtifacts(t, h, "stored-no-db", false)

		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), "stored-no-db"); !errors.Is(err, ErrSessionNotFound) {
			t.Fatalf("openQueryRecorder(no db) error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("store open failure", func(t *testing.T) {
		openErr := errors.New("boom")
		h := newHarness(t, WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return nil, openErr
		}))
		writeStoppedSessionArtifacts(t, h, "stored-open-failure", true)

		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), "stored-open-failure"); !errors.Is(err, openErr) {
			t.Fatalf("openQueryRecorder(open failure) error = %v, want wrapped %v", err, openErr)
		}
	})

	t.Run("cleanup closes reopened recorder", func(t *testing.T) {
		recorder := &queryRecorderStub{}
		h := newHarness(t, WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return recorder, nil
		}))
		writeStoppedSessionArtifacts(t, h, "stored-cleanup", true)

		got, cleanup, err := h.manager.openQueryRecorder(testutil.Context(t), "stored-cleanup")
		if err != nil {
			t.Fatalf("openQueryRecorder(cleanup) error = %v", err)
		}
		if got != recorder {
			t.Fatalf("openQueryRecorder(cleanup) recorder = %T, want queryRecorderStub", got)
		}
		if cleanup == nil {
			t.Fatal("openQueryRecorder(cleanup) cleanup = nil, want non-nil")
		}
		if err := cleanup(); err != nil {
			t.Fatalf("cleanup() error = %v", err)
		}
		if recorder.closeCalls != 1 {
			t.Fatalf("cleanup() close calls = %d, want 1", recorder.closeCalls)
		}
	})
}

func TestReadMetaAndQueryHelpers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if _, err := h.manager.readMeta("   "); err == nil {
		t.Fatal("readMeta(blank) error = nil, want non-nil")
	}
	if _, err := h.manager.readMeta("missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("readMeta(missing) error = %v, want ErrSessionNotFound", err)
	}

	invalidDir := filepath.Join(h.homePaths.SessionsDir, "invalid")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(invalid) error = %v", err)
	}
	if err := os.WriteFile(store.SessionMetaFile(invalidDir), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(invalid meta) error = %v", err)
	}
	if _, err := h.manager.readMeta("invalid"); err == nil {
		t.Fatal("readMeta(invalid) error = nil, want non-nil")
	}

	acpID := "  acp-123  "
	stopReason := store.StopTimeout
	createdAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	info := sessionInfoFromMeta(store.SessionMeta{
		ID:           "sess-1",
		Name:         "stored",
		AgentName:    "coder",
		WorkspaceID:  "ws-1",
		State:        string(StateStopped),
		StopReason:   &stopReason,
		StopDetail:   "deadline exceeded",
		ACPSessionID: &acpID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	})
	if got := info.ACPSessionID; got != "acp-123" {
		t.Fatalf("sessionInfoFromMeta().ACPSessionID = %q, want %q", got, "acp-123")
	}
	if got := info.State; got != StateStopped {
		t.Fatalf("sessionInfoFromMeta().State = %q, want %q", got, StateStopped)
	}
	if got := info.Type; got != SessionTypeUser {
		t.Fatalf("sessionInfoFromMeta().Type = %q, want %q", got, SessionTypeUser)
	}
	if got := info.StopReason; got != store.StopTimeout {
		t.Fatalf("sessionInfoFromMeta().StopReason = %q, want %q", got, store.StopTimeout)
	}
	if got := info.StopDetail; got != "deadline exceeded" {
		t.Fatalf("sessionInfoFromMeta().StopDetail = %q, want %q", got, "deadline exceeded")
	}

	t.Run("Should keep stop fields empty for legacy metadata", func(t *testing.T) {
		legacyInfo := sessionInfoFromMeta(store.SessionMeta{
			ID:          "sess-legacy",
			AgentName:   "coder",
			WorkspaceID: "ws-1",
			State:       string(StateStopped),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
		if got := legacyInfo.StopReason; got != "" {
			t.Fatalf("sessionInfoFromMeta(legacy).StopReason = %q, want empty", got)
		}
		if got := legacyInfo.StopDetail; got != "" {
			t.Fatalf("sessionInfoFromMeta(legacy).StopDetail = %q, want empty", got)
		}
	})

	sameTime := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	sorted := sortSessionInfos([]*SessionInfo{
		{ID: "b", CreatedAt: sameTime},
		nil,
		{ID: "a", CreatedAt: sameTime},
		{ID: "c", CreatedAt: sameTime.Add(time.Minute)},
	})
	if got := []string{sorted[0].ID, sorted[1].ID, sorted[2].ID}; strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("sortSessionInfos() ids = %v, want [a b c]", got)
	}

	if got := stringValue(nil); got != "" {
		t.Fatalf("stringValue(nil) = %q, want empty", got)
	}
	if got := stringValue(&acpID); got != "acp-123" {
		t.Fatalf("stringValue(channeld) = %q, want %q", got, "acp-123")
	}
}

func TestNewAgentProcessDefaultsAndNotifierNoop(t *testing.T) {
	t.Parallel()

	args := []string{"--json"}
	proc := NewAgentProcess(AgentProcessOptions{
		PID:       42,
		AgentName: "coder",
		Args:      args,
	})
	args[0] = "--changed"

	select {
	case <-proc.Done():
	default:
		t.Fatal("Done() channel is not closed by default")
	}
	if err := proc.Wait(); err != nil {
		t.Fatalf("Wait() error = %v, want nil", err)
	}
	if got := proc.Stderr(); got != "" {
		t.Fatalf("Stderr() = %q, want empty", got)
	}
	if got := proc.Args[0]; got != "--json" {
		t.Fatalf("NewAgentProcess() copied args = %q, want %q", got, "--json")
	}
}

func writeStoppedSessionArtifacts(t *testing.T, h *harness, id string, withDB bool) string {
	t.Helper()

	sessionDir := filepath.Join(h.homePaths.SessionsDir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sessionDir, err)
	}

	now := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	if err := store.WriteSessionMeta(store.SessionMetaFile(sessionDir), store.SessionMeta{
		ID:          id,
		Name:        "stored",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		State:       string(StateStopped),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta(%q) error = %v", id, err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	if withDB {
		if err := os.WriteFile(dbPath, nil, 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", dbPath, err)
		}
	}

	return dbPath
}

type queryRecorderStub struct {
	events      []store.SessionEvent
	history     []store.TurnHistory
	queryErr    error
	historyErr  error
	closeCalls  int
	queryCalls  []store.EventQuery
	historyCall []store.EventQuery
}

func (s *queryRecorderStub) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (s *queryRecorderStub) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (s *queryRecorderStub) Query(_ context.Context, query store.EventQuery) ([]store.SessionEvent, error) {
	s.queryCalls = append(s.queryCalls, query)
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	return append([]store.SessionEvent(nil), s.events...), nil
}

func (s *queryRecorderStub) History(_ context.Context, query store.EventQuery) ([]store.TurnHistory, error) {
	s.historyCall = append(s.historyCall, query)
	if s.historyErr != nil {
		return nil, s.historyErr
	}
	return append([]store.TurnHistory(nil), s.history...), nil
}

func (s *queryRecorderStub) Close(context.Context) error {
	s.closeCalls++
	return nil
}
