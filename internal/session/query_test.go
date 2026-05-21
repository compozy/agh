package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/events"
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

func TestManagerStatusRejectsTraversalSessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	escapedID := createEscapedStoredSession(t, h)

	info, err := h.manager.Status(testutil.Context(t), "../"+escapedID)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Status(traversal) error = %v, want ErrSessionNotFound", err)
	}
	if info != nil {
		t.Fatalf("Status(traversal) info = %#v, want nil", info)
	}
}

func TestNormalizeStoredSessionIDRejectsWindowsDriveRelativePath(t *testing.T) {
	t.Parallel()

	if _, err := normalizeStoredSessionID("C:escape"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("normalizeStoredSessionID(windows drive-relative) error = %v, want ErrSessionNotFound", err)
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
	if got := info.StopDetail; got != resumeStopDetailStartIncomplete {
		t.Fatalf("Status(repaired).StopDetail = %q, want %q", got, resumeStopDetailStartIncomplete)
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
	if got := repairedMeta.StopDetail; got != resumeStopDetailStartIncomplete {
		t.Fatalf("repaired meta stop detail = %q, want %q", got, resumeStopDetailStartIncomplete)
	}
	if repairedMeta.ACPSessionID != nil {
		t.Fatalf("repaired meta ACPSessionID = %#v, want nil", repairedMeta.ACPSessionID)
	}
}

func TestManagerStatusRepairsInterruptedSessionAsStalledWhenLiveSubprocessIsStale(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	meta, err := store.ReadSessionMeta(session.MetaPath())
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	lastUpdate := time.Now().UTC().Add(-DefaultLivenessStallAfter - time.Minute)
	startedAt := time.Now().UTC().Add(-10 * time.Minute)
	meta.State = string(StateActive)
	meta.StopReason = nil
	meta.StopDetail = ""
	meta.Liveness = &store.SessionLivenessMeta{
		SubprocessPID:       os.Getpid(),
		SubprocessStartedAt: &startedAt,
		LastUpdateAt:        &lastUpdate,
	}
	if err := store.WriteSessionMeta(session.MetaPath(), meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	info, err := h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(stalled) error = %v", err)
	}
	if got, want := info.State, StateStopped; got != want {
		t.Fatalf("Status(stalled).State = %q, want %q", got, want)
	}
	if got, want := info.StopReason, store.StopAgentCrashed; got != want {
		t.Fatalf("Status(stalled).StopReason = %q, want %q", got, want)
	}
	if got, want := info.StopDetail, resumeStopDetailAgentStalled; got != want {
		t.Fatalf("Status(stalled).StopDetail = %q, want %q", got, want)
	}
	if info.Liveness == nil {
		t.Fatal("Status(stalled).Liveness = nil, want liveness metadata")
	}
	if got, want := info.Liveness.StallState, store.SessionStallStateDetected; got != want {
		t.Fatalf("Status(stalled).Liveness.StallState = %q, want %q", got, want)
	}
	if got, want := info.Liveness.StallReason, store.SessionStallReasonActivityTimeout; got != want {
		t.Fatalf("Status(stalled).Liveness.StallReason = %q, want %q", got, want)
	}
}

func TestManagerStatusDoesNotRepairPendingStartMetadata(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionID := "sess-pending"
	sessionDir := filepath.Join(h.homePaths.SessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}

	acpSessionID := "acp-pending"
	meta := store.SessionMeta{
		ID:           sessionID,
		Name:         "pending",
		AgentName:    "coder",
		WorkspaceID:  h.workspaceID,
		State:        string(StateStarting),
		ACPSessionID: stringPointer(acpSessionID),
		CreatedAt:    time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 20, 12, 0, 1, 0, time.UTC),
	}
	metaPath := store.SessionMetaFile(sessionDir)
	if err := store.WriteSessionMeta(metaPath, meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	h.manager.mu.Lock()
	h.manager.pending[sessionID] = struct{}{}
	h.manager.mu.Unlock()

	info, err := h.manager.Status(testutil.Context(t), sessionID)
	if err != nil {
		t.Fatalf("Status(pending) error = %v", err)
	}
	if got := info.State; got != StateStarting {
		t.Fatalf("Status(pending).State = %q, want %q", got, StateStarting)
	}
	if got := info.ACPSessionID; got != acpSessionID {
		t.Fatalf("Status(pending).ACPSessionID = %q, want %q", got, acpSessionID)
	}
	if got := info.StopDetail; got != "" {
		t.Fatalf("Status(pending).StopDetail = %q, want empty", got)
	}

	storedMeta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if got := storedMeta.State; got != string(StateStarting) {
		t.Fatalf("stored meta state = %q, want %q", got, StateStarting)
	}
	if storedMeta.StopReason != nil {
		t.Fatalf("stored meta stop reason = %#v, want nil", storedMeta.StopReason)
	}
	if got := stringValue(storedMeta.ACPSessionID); got != acpSessionID {
		t.Fatalf("stored meta ACPSessionID = %q, want %q", got, acpSessionID)
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
	if len(afterPrompt) != 2 {
		t.Fatalf("Events(after prompt) = %d events, want 2", len(afterPrompt))
	}
	if got := afterPrompt[0].Type; got != EventTypeSessionStopped {
		t.Fatalf("Events(after prompt)[0].Type = %q, want %q", got, EventTypeSessionStopped)
	}
	if got := afterPrompt[1].Type; got != events.TranscriptMarkerCreated {
		t.Fatalf("Events(after prompt)[1].Type = %q, want %q", got, events.TranscriptMarkerCreated)
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

func TestManagerEventsRejectTraversalSessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	escapedID := createEscapedStoredSession(t, h)

	events, err := h.manager.Events(testutil.Context(t), "../"+escapedID, store.EventQuery{})
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Events(traversal) error = %v, want ErrSessionNotFound", err)
	}
	if events != nil {
		t.Fatalf("Events(traversal) = %#v, want nil", events)
	}
}

func TestManagerOpenQueryRecorderValidationAndCleanup(t *testing.T) {
	t.Parallel()

	t.Run("Should requires context and session id", func(t *testing.T) {
		h := newHarness(t)
		var nilCtx context.Context
		if _, _, err := h.manager.openQueryRecorder(nilCtx, "sess-1"); err == nil {
			t.Fatal("openQueryRecorder(nil, id) error = nil, want non-nil")
		}
		if _, _, err := h.manager.openQueryRecorder(testutil.Context(t), "   "); err == nil {
			t.Fatal("openQueryRecorder(ctx, blank) error = nil, want non-nil")
		}
	})

	t.Run("Should active session requires recorder", func(t *testing.T) {
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

	t.Run("Should wait for finalization before returning active recorder", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)
		done := make(chan struct{})
		h.manager.mu.Lock()
		h.manager.finalizing[session.ID] = done
		h.manager.mu.Unlock()
		t.Cleanup(func() {
			h.manager.finishFinalization(session.ID)
			_ = h.manager.Stop(testutil.Context(t), session.ID)
		})

		ctx, cancel := context.WithCancel(testutil.Context(t))
		cancel()
		if _, _, err := h.manager.openQueryRecorder(ctx, session.ID); !errors.Is(err, context.Canceled) {
			t.Fatalf("openQueryRecorder(finalizing canceled ctx) error = %v, want context.Canceled", err)
		}
	})

	t.Run("Should finalizing active session reopens stored events after recorder closes", func(t *testing.T) {
		h := newHarness(t)
		session := createSession(t, h)

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		runtimeEvents := collectEvents(t, eventsCh)
		if len(runtimeEvents) == 0 {
			t.Fatal("Prompt() returned no runtime events, want recorded prompt events")
		}

		activeEvents, err := h.manager.Events(testutil.Context(t), session.ID, store.EventQuery{})
		if err != nil {
			t.Fatalf("Events(active) error = %v", err)
		}
		if len(activeEvents) == 0 {
			t.Fatal("Events(active) returned no events, want persisted session events")
		}

		recorder := session.recorderHandle()
		if recorder == nil {
			t.Fatal("recorderHandle() = nil, want active recorder")
		}
		if err := recorder.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(active recorder) error = %v", err)
		}

		done := make(chan struct{})
		h.manager.mu.Lock()
		h.manager.finalizing[session.ID] = done
		h.manager.mu.Unlock()

		type queryResult struct {
			events []store.SessionEvent
			err    error
		}
		started := make(chan struct{})
		resultCh := make(chan queryResult, 1)
		ctx, cancel := context.WithTimeout(testutil.Context(t), 5*time.Second)
		defer cancel()
		go func() {
			close(started)
			got, cleanup, err := h.manager.openQueryRecorder(ctx, session.ID)
			if err != nil {
				resultCh <- queryResult{err: err}
				return
			}
			events, err := got.Query(ctx, store.EventQuery{})
			if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
				err = cleanupErr
			}
			resultCh <- queryResult{events: events, err: err}
		}()

		now := time.Now().UTC()
		if err := session.beginStopping(now); err != nil {
			t.Fatalf("beginStopping() error = %v", err)
		}
		if err := session.markStopped(now); err != nil {
			t.Fatalf("markStopped() error = %v", err)
		}
		if err := h.manager.writeMeta(session); err != nil {
			t.Fatalf("writeMeta(stopped) error = %v", err)
		}
		h.manager.removeActive(session.ID)
		h.manager.finishFinalization(session.ID)

		result := <-resultCh
		if result.err != nil {
			t.Fatalf("openQueryRecorder(finalizing active) error = %v", result.err)
		}
		if len(result.events) != len(activeEvents) {
			t.Fatalf("openQueryRecorder(finalizing active) events = %d, want %d", len(result.events), len(activeEvents))
		}
	})

	t.Run("Should wait for finalization before reading a closed recorder handle", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		session := createSession(t, h)

		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			t.Fatalf("Prompt() error = %v", err)
		}
		runtimeEvents := collectEvents(t, eventsCh)
		if len(runtimeEvents) == 0 {
			t.Fatal("Prompt() returned no runtime events, want recorded prompt events")
		}

		activeEvents, err := h.manager.Events(testutil.Context(t), session.ID, store.EventQuery{})
		if err != nil {
			t.Fatalf("Events(active) error = %v", err)
		}
		if len(activeEvents) == 0 {
			t.Fatal("Events(active) returned no events, want persisted session events")
		}

		recorder := session.recorderHandle()
		if recorder == nil {
			t.Fatal("recorderHandle() = nil, want active recorder")
		}
		if err := recorder.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(active recorder) error = %v", err)
		}

		done := make(chan struct{})
		h.manager.mu.Lock()
		h.manager.finalizing[session.ID] = done
		h.manager.mu.Unlock()

		type queryResult struct {
			events []store.SessionEvent
			err    error
		}
		started := make(chan struct{})
		resultCh := make(chan queryResult, 1)
		ctx, cancel := context.WithTimeout(testutil.Context(t), 5*time.Second)
		defer cancel()
		go func() {
			close(started)
			got, cleanup, err := h.manager.openQueryRecorder(ctx, session.ID)
			if err != nil {
				resultCh <- queryResult{err: err}
				return
			}
			events, err := got.Query(ctx, store.EventQuery{})
			if cleanupErr := cleanup(); cleanupErr != nil && err == nil {
				err = cleanupErr
			}
			resultCh <- queryResult{events: events, err: err}
		}()

		<-started
		select {
		case result := <-resultCh:
			t.Fatalf(
				"openQueryRecorder returned before finalization finished: events=%d err=%v",
				len(result.events),
				result.err,
			)
		case <-time.After(50 * time.Millisecond):
		}

		now := time.Now().UTC()
		if err := session.beginStopping(now); err != nil {
			t.Fatalf("beginStopping() error = %v", err)
		}
		if err := session.markStopped(now); err != nil {
			t.Fatalf("markStopped() error = %v", err)
		}
		if err := h.manager.writeMeta(session); err != nil {
			t.Fatalf("writeMeta(stopped) error = %v", err)
		}
		h.manager.removeActive(session.ID)
		h.manager.finishFinalization(session.ID)

		result := <-resultCh
		if result.err != nil {
			t.Fatalf("openQueryRecorder(finalized active) error = %v", result.err)
		}
		if err := compareQueriedSessionEvents(activeEvents, result.events); err != nil {
			t.Fatalf("openQueryRecorder(finalized active) events mismatch: %v", err)
		}
	})

	t.Run("Should missing session metadata", func(t *testing.T) {
		h := newHarness(t)
		if _, _, err := h.manager.openQueryRecorder(
			testutil.Context(t),
			"missing",
		); !errors.Is(
			err,
			ErrSessionNotFound,
		) {
			t.Fatalf("openQueryRecorder(missing) error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("Should missing database file", func(t *testing.T) {
		h := newHarness(t)
		writeStoppedSessionArtifacts(t, h, "stored-no-db", false)

		if _, _, err := h.manager.openQueryRecorder(
			testutil.Context(t),
			"stored-no-db",
		); !errors.Is(
			err,
			ErrSessionNotFound,
		) {
			t.Fatalf("openQueryRecorder(no db) error = %v, want ErrSessionNotFound", err)
		}
	})

	t.Run("Should store open failure", func(t *testing.T) {
		openErr := errors.New("boom")
		h := newHarness(t, WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return nil, openErr
		}))
		writeStoppedSessionArtifacts(t, h, "stored-open-failure", true)

		if _, _, err := h.manager.openQueryRecorder(
			testutil.Context(t),
			"stored-open-failure",
		); !errors.Is(
			err,
			openErr,
		) {
			t.Fatalf("openQueryRecorder(open failure) error = %v, want wrapped %v", err, openErr)
		}
	})

	t.Run("Should cleanup closes reopened recorder", func(t *testing.T) {
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
	if _, err := h.manager.readMetaWithContext(testutil.Context(t), "   "); err == nil {
		t.Fatal("readMeta(blank) error = nil, want non-nil")
	}
	if _, err := h.manager.readMetaWithContext(testutil.Context(t), "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("readMetaWithContext(missing) error = %v, want ErrSessionNotFound", err)
	}

	invalidDir := filepath.Join(h.homePaths.SessionsDir, "invalid")
	if err := os.MkdirAll(invalidDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(invalid) error = %v", err)
	}
	if err := os.WriteFile(store.SessionMetaFile(invalidDir), []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(invalid meta) error = %v", err)
	}
	if _, err := h.manager.readMetaWithContext(testutil.Context(t), "invalid"); err == nil {
		t.Fatal("readMetaWithContext(invalid) error = nil, want non-nil")
	}

	acpID := "  acp-123  "
	stopReason := store.StopTimeout
	createdAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	info := sessionInfoFromMeta(store.SessionMeta{
		ID:              "sess-1",
		Name:            "stored",
		AgentName:       "coder",
		Provider:        "codex",
		Model:           "  gpt-4o  ",
		ReasoningEffort: "  high  ",
		WorkspaceID:     "ws-1",
		State:           string(StateStopped),
		StopReason:      &stopReason,
		StopDetail:      "deadline exceeded",
		ACPSessionID:    &acpID,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	})
	if got := info.ACPSessionID; got != "acp-123" {
		t.Fatalf("sessionInfoFromMeta().ACPSessionID = %q, want %q", got, "acp-123")
	}
	if got := info.Provider; got != "codex" {
		t.Fatalf("sessionInfoFromMeta().Provider = %q, want %q", got, "codex")
	}
	if got := info.Model; got != "gpt-4o" {
		t.Fatalf("sessionInfoFromMeta().Model = %q, want %q", got, "gpt-4o")
	}
	if got := info.ReasoningEffort; got != "high" {
		t.Fatalf("sessionInfoFromMeta().ReasoningEffort = %q, want %q", got, "high")
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
	sorted := sortSessionInfos([]*Info{
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

func compareQueriedSessionEvents(want []store.SessionEvent, got []store.SessionEvent) error {
	if len(got) != len(want) {
		return fmt.Errorf("count = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index].ID != want[index].ID {
			return fmt.Errorf("event[%d].id = %q, want %q", index, got[index].ID, want[index].ID)
		}
		if got[index].Sequence != want[index].Sequence {
			return fmt.Errorf(
				"event[%d].sequence = %d, want %d",
				index,
				got[index].Sequence,
				want[index].Sequence,
			)
		}
		if got[index].SessionID != want[index].SessionID {
			return fmt.Errorf("event[%d].session_id = %q, want %q", index, got[index].SessionID, want[index].SessionID)
		}
		if got[index].TurnID != want[index].TurnID {
			return fmt.Errorf("event[%d].turn_id = %q, want %q", index, got[index].TurnID, want[index].TurnID)
		}
		if got[index].Type != want[index].Type {
			return fmt.Errorf("event[%d].type = %q, want %q", index, got[index].Type, want[index].Type)
		}
		if got[index].AgentName != want[index].AgentName {
			return fmt.Errorf(
				"event[%d].agent_name = %q, want %q",
				index,
				got[index].AgentName,
				want[index].AgentName,
			)
		}
		if got[index].Content != want[index].Content {
			return fmt.Errorf("event[%d].content = %q, want %q", index, got[index].Content, want[index].Content)
		}
		if !got[index].Timestamp.Equal(want[index].Timestamp) {
			return fmt.Errorf(
				"event[%d].timestamp = %s, want %s",
				index,
				got[index].Timestamp.UTC().Format(time.RFC3339Nano),
				want[index].Timestamp.UTC().Format(time.RFC3339Nano),
			)
		}
	}
	return nil
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

func createEscapedStoredSession(t *testing.T, h *harness) string {
	t.Helper()

	escapedID := "escaped-session"
	escapedDir := filepath.Join(filepath.Dir(h.homePaths.SessionsDir), escapedID)
	if err := os.MkdirAll(escapedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", escapedDir, err)
	}

	now := time.Now().UTC()
	if err := store.WriteSessionMeta(store.SessionMetaFile(escapedDir), store.SessionMeta{
		ID:          escapedID,
		Name:        "escaped",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		SessionType: string(SessionTypeUser),
		State:       string(StateStopped),
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta(%q) error = %v", escapedDir, err)
	}

	recorder, err := h.manager.openStore(testutil.Context(t), escapedID, store.SessionDBFile(escapedDir))
	if err != nil {
		t.Fatalf("openStore(%q) error = %v", escapedID, err)
	}
	t.Cleanup(func() {
		_ = recorder.Close(testutil.Context(t))
	})

	if err := recorder.Record(testutil.Context(t), store.SessionEvent{
		TurnID:    "turn-1",
		Type:      acp.EventTypeAgentMessage,
		AgentName: "coder",
		Content:   `{"type":"agent_message","text":"escaped"}`,
		Timestamp: now,
	}); err != nil {
		t.Fatalf("Record() error = %v", err)
	}

	return escapedID
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
