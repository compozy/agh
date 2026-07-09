package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	eventspkg "github.com/compozy/agh/internal/events"
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

	entries, err := h.manager.Transcript(testutil.Context(t), session.ID, store.EventQuery{})
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	messages := transcript.MessagesFromEntries(entries)
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

	entries, err := h.manager.Transcript(testutil.Context(t), session.ID, store.EventQuery{})
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	messages := transcript.MessagesFromEntries(entries)
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

	_, err := h.manager.Transcript(testutil.Context(t), "stored-query-failure", store.EventQuery{})
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

	entries, err := h.manager.Transcript(testutil.Context(t), "stored-cleanup-error", store.EventQuery{})
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	messages := transcript.MessagesFromEntries(entries)
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

func TestManagerTranscriptCache(t *testing.T) {
	t.Parallel()

	t.Run("Should fold only delta events after the cached cursor", func(t *testing.T) {
		t.Parallel()

		logs := newCaptureLogHandler()
		recorder := &filteringTranscriptRecorder{}
		h := newHarness(
			t,
			WithLogger(slog.New(logs)),
			WithStore(func(_ context.Context, _ string, _ string) (EventRecorder, error) {
				return recorder, nil
			}),
		)
		sessionID := "cached-transcript"
		writeStoppedSessionArtifacts(t, h, sessionID, true)
		recorder.Append(
			transcriptCacheEvent(t, sessionID, 1, "turn-cache", acp.EventTypeUserMessage, "hello"),
			transcriptCacheEvent(t, sessionID, 2, "turn-cache", acp.EventTypeAgentMessage, "hel"),
		)

		if _, err := h.manager.Transcript(testutil.Context(t), sessionID, store.EventQuery{}); err != nil {
			t.Fatalf("Transcript(initial) error = %v", err)
		}
		recorder.Append(
			transcriptCacheEvent(t, sessionID, 3, "turn-cache", acp.EventTypeAgentMessage, "lo"),
			transcriptCacheEvent(t, sessionID, 4, "turn-cache", acp.EventTypeDone, ""),
		)

		got, err := h.manager.Transcript(testutil.Context(t), sessionID, store.EventQuery{})
		if err != nil {
			t.Fatalf("Transcript(delta) error = %v", err)
		}
		want, err := transcript.ToUIEntries(recorder.Events())
		if err != nil {
			t.Fatalf("ToUIEntries() error = %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Transcript(delta) = %#v, want %#v", got, want)
		}

		calls := recorder.QueryCalls()
		if got, want := len(calls), 2; got != want {
			t.Fatalf("query calls = %d, want %d; calls=%#v", got, want, calls)
		}
		if calls[0].AfterSequence != 0 {
			t.Fatalf("initial query AfterSequence = %d, want 0", calls[0].AfterSequence)
		}
		if got, want := calls[1].AfterSequence, int64(2); got != want {
			t.Fatalf("delta query AfterSequence = %d, want %d", got, want)
		}

		record, ok := logs.FindByMessage(eventspkg.SessionTranscriptCacheRebuilt)
		if !ok {
			t.Fatalf("FindByMessage(%q) = false; records=%#v", eventspkg.SessionTranscriptCacheRebuilt, logs.Records())
		}
		if got, want := record.Attrs["workspace_id"], h.workspaceID; got != want {
			t.Fatalf("workspace_id log attr = %q, want %q", got, want)
		}
		if got := record.Attrs["session_id"]; got != sessionID {
			t.Fatalf("session_id log attr = %q, want %q", got, sessionID)
		}
	})

	t.Run("Should expose replay watermark after cache rebuild and below-window mutation", func(t *testing.T) {
		t.Parallel()

		recorder := &filteringTranscriptRecorder{}
		h := newHarness(
			t,
			WithStore(func(_ context.Context, _ string, _ string) (EventRecorder, error) {
				return recorder, nil
			}),
		)
		sessionID := "watermarked-transcript"
		writeStoppedSessionArtifacts(t, h, sessionID, true)
		recorder.Append(
			transcriptCacheEvent(t, sessionID, 1, "turn-watermark", acp.EventTypeUserMessage, "hello"),
			transcriptCacheEvent(t, sessionID, 2, "turn-watermark", acp.EventTypeAgentMessage, "hi"),
		)

		if _, err := h.manager.Transcript(testutil.Context(t), sessionID, store.EventQuery{}); err != nil {
			t.Fatalf("Transcript(initial) error = %v", err)
		}
		watermark := h.manager.TranscriptWatermark(testutil.Context(t), sessionID)
		if watermark.Sequence != 2 {
			t.Fatalf("TranscriptWatermark().Sequence = %d, want 2", watermark.Sequence)
		}
		if watermark.Reason != TranscriptWatermarkReasonCacheRebuild {
			t.Fatalf(
				"TranscriptWatermark().Reason = %q, want %q",
				watermark.Reason,
				TranscriptWatermarkReasonCacheRebuild,
			)
		}

		h.manager.markTranscriptBelowWindowMutation(sessionID, 1)
		watermark = h.manager.TranscriptWatermark(testutil.Context(t), sessionID)
		if watermark.Sequence != 2 {
			t.Fatalf("TranscriptWatermark().Sequence after mutation = %d, want 2", watermark.Sequence)
		}
		if watermark.Reason != TranscriptWatermarkReasonBelowWindowMutation {
			t.Fatalf(
				"TranscriptWatermark().Reason after mutation = %q, want %q",
				watermark.Reason,
				TranscriptWatermarkReasonBelowWindowMutation,
			)
		}
	})

	t.Run("Should keep cache entries scoped by session id", func(t *testing.T) {
		t.Parallel()

		recorders := map[string]*filteringTranscriptRecorder{
			"sess-cache-a": {},
			"sess-cache-b": {},
		}
		h := newHarness(
			t,
			WithStore(func(_ context.Context, sessionID string, _ string) (EventRecorder, error) {
				return recorders[sessionID], nil
			}),
		)
		for sessionID, recorder := range recorders {
			writeStoppedSessionArtifacts(t, h, sessionID, true)
			recorder.Append(transcriptCacheEvent(
				t,
				sessionID,
				1,
				"turn-"+sessionID,
				acp.EventTypeAgentMessage,
				"text-"+sessionID,
			))
		}

		entriesA, err := h.manager.Transcript(testutil.Context(t), "sess-cache-a", store.EventQuery{})
		if err != nil {
			t.Fatalf("Transcript(sess-cache-a) error = %v", err)
		}
		entriesB, err := h.manager.Transcript(testutil.Context(t), "sess-cache-b", store.EventQuery{})
		if err != nil {
			t.Fatalf("Transcript(sess-cache-b) error = %v", err)
		}

		textA := transcript.JoinUIMessageText(transcript.MessagesFromEntries(entriesA))
		textB := transcript.JoinUIMessageText(transcript.MessagesFromEntries(entriesB))
		if textA != "text-sess-cache-a" {
			t.Fatalf("session A transcript = %q, want text-sess-cache-a", textA)
		}
		if textB != "text-sess-cache-b" {
			t.Fatalf("session B transcript = %q, want text-sess-cache-b", textB)
		}
	})

	t.Run("Should not block another session while one transcript cache refresh is querying", func(t *testing.T) {
		t.Parallel()

		releaseSlowQuery := make(chan struct{})
		slowRecorder := &blockingTranscriptRecorder{
			filteringTranscriptRecorder: filteringTranscriptRecorder{},
			started:                     make(chan struct{}),
			release:                     releaseSlowQuery,
		}
		fastRecorder := &filteringTranscriptRecorder{}
		recorders := map[string]EventRecorder{
			"sess-cache-slow": slowRecorder,
			"sess-cache-fast": fastRecorder,
		}
		h := newHarness(
			t,
			WithStore(func(_ context.Context, sessionID string, _ string) (EventRecorder, error) {
				return recorders[sessionID], nil
			}),
		)
		writeStoppedSessionArtifacts(t, h, "sess-cache-slow", true)
		writeStoppedSessionArtifacts(t, h, "sess-cache-fast", true)
		slowRecorder.Append(transcriptCacheEvent(
			t,
			"sess-cache-slow",
			1,
			"turn-slow",
			acp.EventTypeAgentMessage,
			"slow",
		))
		fastRecorder.Append(transcriptCacheEvent(
			t,
			"sess-cache-fast",
			1,
			"turn-fast",
			acp.EventTypeAgentMessage,
			"fast",
		))

		slowDone := make(chan error, 1)
		go func() {
			_, err := h.manager.Transcript(testutil.Context(t), "sess-cache-slow", store.EventQuery{})
			slowDone <- err
		}()
		select {
		case <-slowRecorder.started:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("slow transcript query did not start")
		}

		fastDone := make(chan error, 1)
		go func() {
			entries, err := h.manager.Transcript(testutil.Context(t), "sess-cache-fast", store.EventQuery{})
			if err != nil {
				fastDone <- err
				return
			}
			if got := transcript.JoinUIMessageText(transcript.MessagesFromEntries(entries)); got != "fast" {
				fastDone <- fmt.Errorf("fast transcript = %q, want fast", got)
				return
			}
			fastDone <- nil
		}()
		select {
		case err := <-fastDone:
			if err != nil {
				close(releaseSlowQuery)
				t.Fatalf("Transcript(fast) error = %v", err)
			}
		case <-time.After(100 * time.Millisecond):
			close(releaseSlowQuery)
			t.Fatal("Transcript(fast) blocked behind unrelated transcript cache query")
		}

		close(releaseSlowQuery)
		if err := <-slowDone; err != nil {
			t.Fatalf("Transcript(slow) error = %v", err)
		}
	})

	t.Run("Should drop transcript cache when deleting a session", func(t *testing.T) {
		t.Parallel()

		recorder := &filteringTranscriptRecorder{}
		h := newHarness(
			t,
			WithStore(func(_ context.Context, _ string, _ string) (EventRecorder, error) {
				return recorder, nil
			}),
		)
		sessionID := "sess-cache-delete"
		writeStoppedSessionArtifacts(t, h, sessionID, true)
		recorder.Append(transcriptCacheEvent(
			t,
			sessionID,
			1,
			"turn-delete",
			acp.EventTypeAgentMessage,
			"cached",
		))

		if _, err := h.manager.Transcript(testutil.Context(t), sessionID, store.EventQuery{}); err != nil {
			t.Fatalf("Transcript(before delete) error = %v", err)
		}
		h.manager.transcriptCacheMu.Lock()
		_, cachedBefore := h.manager.transcriptCache[sessionID]
		h.manager.transcriptCacheMu.Unlock()
		if !cachedBefore {
			t.Fatal("transcript cache missing before delete")
		}

		if err := h.manager.Delete(testutil.Context(t), sessionID); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}
		h.manager.transcriptCacheMu.Lock()
		_, cachedAfter := h.manager.transcriptCache[sessionID]
		h.manager.transcriptCacheMu.Unlock()
		if cachedAfter {
			t.Fatal("transcript cache still present after delete")
		}
	})
}

type transcriptRecorderStub struct {
	queryRecorderStub
	closeErr error
}

func (s *transcriptRecorderStub) Close(context.Context) error {
	s.closeCalls++
	return s.closeErr
}

type filteringTranscriptRecorder struct {
	mu         sync.Mutex
	events     []store.SessionEvent
	queryCalls []store.EventQuery
}

type blockingTranscriptRecorder struct {
	filteringTranscriptRecorder
	started chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (r *filteringTranscriptRecorder) Append(events ...store.SessionEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, events...)
}

func (r *filteringTranscriptRecorder) Events() []store.SessionEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]store.SessionEvent(nil), r.events...)
}

func (r *filteringTranscriptRecorder) QueryCalls() []store.EventQuery {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]store.EventQuery(nil), r.queryCalls...)
}

func (r *filteringTranscriptRecorder) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (r *filteringTranscriptRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (r *filteringTranscriptRecorder) Query(_ context.Context, query store.EventQuery) ([]store.SessionEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.queryCalls = append(r.queryCalls, query)

	events := make([]store.SessionEvent, 0, len(r.events))
	for _, event := range r.events {
		if query.AfterSequence > 0 && event.Sequence <= query.AfterSequence {
			continue
		}
		if query.BeforeSequence > 0 && event.Sequence >= query.BeforeSequence {
			continue
		}
		events = append(events, event)
	}
	if query.Limit > 0 && len(events) > query.Limit {
		events = events[len(events)-query.Limit:]
	}
	return append([]store.SessionEvent(nil), events...), nil
}

func (r *blockingTranscriptRecorder) Query(
	ctx context.Context,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	shouldBlock := false
	r.once.Do(func() {
		shouldBlock = true
		close(r.started)
	})
	if shouldBlock {
		select {
		case <-r.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return r.filteringTranscriptRecorder.Query(ctx, query)
}

func (r *filteringTranscriptRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (r *filteringTranscriptRecorder) Close(context.Context) error {
	return nil
}

func transcriptCacheEvent(
	t *testing.T,
	sessionID string,
	sequence int64,
	turnID string,
	eventType string,
	text string,
) store.SessionEvent {
	t.Helper()
	timestamp := time.Date(2026, 7, 7, 12, 0, 0, int(sequence), time.UTC)
	payload, err := json.Marshal(map[string]string{
		"schema":     "agh.session.event.v1",
		"type":       eventType,
		"session_id": sessionID,
		"turn_id":    turnID,
		"timestamp":  timestamp.Format(time.RFC3339Nano),
		"text":       text,
	})
	if err != nil {
		t.Fatalf("json.Marshal(%s) error = %v", eventType, err)
	}
	return store.SessionEvent{
		ID:        sessionID + "-" + eventType,
		SessionID: sessionID,
		Sequence:  sequence,
		TurnID:    turnID,
		Type:      eventType,
		AgentName: "coder",
		Content:   string(payload),
		Timestamp: timestamp,
	}
}
