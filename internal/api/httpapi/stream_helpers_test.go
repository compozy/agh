package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type bufferFlusher struct {
	bytes.Buffer
}

func (bufferFlusher) Flush() {}

func TestStreamSessionHandlerPollsForNewEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	done := make(chan struct{})
	callCount := 0
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return newSessionInfo("sess-123"), nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			callCount++
			switch callCount {
			case 1:
				return []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: "sess-123",
					Sequence:  1,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				}}, nil
			case 2:
				close(done)
				return []store.SessionEvent{{
					ID:        "ev-2",
					SessionID: "sess-123",
					Sequence:  2,
					TurnID:    "turn-1",
					Type:      "done",
					AgentName: "coder",
					Content:   `{"stop_reason":"end_turn"}`,
					Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				}}, nil
			default:
				return nil, nil
			}
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/stream",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[0].ID != "1" || records[1].ID != "2" {
		t.Fatalf("records = %#v", records)
	}
}

func TestStreamSessionHandlerStopsWhenSessionIsAlreadyStopped(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			info := newSessionInfo("sess-123")
			info.State = session.StateStopped
			info.UpdatedAt = time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC)
			return info, nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			return nil, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/stream",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1; body=%s", len(records), recorder.Body.String())
	}
	if records[0].Event != session.EventTypeSessionStopped {
		t.Fatalf("records[0].Event = %q, want %q", records[0].Event, session.EventTypeSessionStopped)
	}
	var payload sessionEventPayload
	if err := json.Unmarshal(records[0].Data, &payload); err != nil {
		t.Fatalf("json.Unmarshal(stop payload) error = %v; data=%s", err, string(records[0].Data))
	}
	if payload.WorkspaceID != "ws-workspace" || payload.WorkspacePath != "/workspace" {
		t.Fatalf("stop payload = %#v", payload)
	}
}

func TestStreamObserveEventsPollsForNewEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	done := make(chan struct{})
	callCount := 0
	observer := stubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			callCount++
			timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch callCount {
			case 1:
				return []store.EventSummary{
					{ID: "sum-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: timestamp},
				}, nil
			case 2:
				close(done)
				return []store.EventSummary{
					{
						ID:        "sum-2",
						SessionID: "sess-1",
						Type:      "done",
						AgentName: "coder",
						Timestamp: timestamp.Add(time.Second),
					},
				}, nil
			default:
				return nil, nil
			}
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/observe/events/stream",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[0].ID == records[1].ID {
		t.Fatalf("expected distinct observe SSE ids, got %#v", records)
	}
}

func TestStreamObserveEventsCarriesHarnessLifecyclePayloads(t *testing.T) {
	homePaths := newTestHomePaths(t)
	done := make(chan struct{})
	var doneOnce sync.Once
	var capturedQuery store.EventSummaryQuery
	observer := stubObserver{
		QueryEventsFn: func(_ context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
			capturedQuery = query
			doneOnce.Do(func() { close(done) })
			return []store.EventSummary{{
				ID:        "sum-harness",
				SessionID: "sess-harness",
				Type:      "harness.context_resolved",
				AgentName: "coder",
				Summary:   "surface=startup sections=memory|skills|network",
				Timestamp: time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/observe/events/stream?session_id=sess-harness",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if got, want := len(records), 1; got != want {
		t.Fatalf("len(records) = %d, want %d; body=%s", got, want, recorder.Body.String())
	}

	var payload observeEventPayload
	if err := json.Unmarshal(records[0].Data, &payload); err != nil {
		t.Fatalf("json.Unmarshal(observe payload) error = %v", err)
	}
	if got, want := payload.Type, "harness.context_resolved"; got != want {
		t.Fatalf("payload.Type = %q, want %q", got, want)
	}
	if got, want := payload.SessionID, "sess-harness"; got != want {
		t.Fatalf("payload.SessionID = %q, want %q", got, want)
	}
	if got, want := capturedQuery.SessionID, "sess-harness"; got != want {
		t.Fatalf("observer query session_id = %q, want %q", got, want)
	}
	if !bytes.Contains(records[0].Data, []byte("sections=memory|skills|network")) {
		t.Fatalf("payload = %s, want harness summary content", string(records[0].Data))
	}
}

func TestStreamBridgeHealthPollsForChangedSnapshots(t *testing.T) {
	homePaths := newTestHomePaths(t)
	done := make(chan struct{})
	callCount := 0
	observer := stubObserver{
		QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
			callCount++
			switch callCount {
			case 1, 2:
				return []observe.BridgeInstanceHealth{{
					BridgeInstanceID:  "brg-123",
					Status:            bridgepkg.BridgeStatusAuthRequired,
					AuthFailuresTotal: 1,
				}}, nil
			case 3:
				close(done)
				return []observe.BridgeInstanceHealth{{
					BridgeInstanceID:      "brg-123",
					Status:                bridgepkg.BridgeStatusReady,
					RouteCount:            2,
					DeliveryFailuresTotal: 1,
				}}, nil
			default:
				return nil, nil
			}
		},
	}
	handlers := newTestHandlersWithBridges(
		t,
		stubSessionManager{},
		observer,
		stubBridgeService{},
		stubWorkspaceService{},
		homePaths,
	)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/bridges/health/stream",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[0].Event != "snapshot" || records[1].Event != "snapshot" {
		t.Fatalf("events = %#v, want snapshot events", records)
	}
	if records[0].ID == records[1].ID {
		t.Fatalf("expected distinct snapshot ids, got %#v", records)
	}

	var first contract.BridgeHealthStreamPayload
	if err := json.Unmarshal(records[0].Data, &first); err != nil {
		t.Fatalf("json.Unmarshal(first snapshot) error = %v", err)
	}
	if got, want := first.BridgeHealth["brg-123"].Status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("first status = %q, want %q", got, want)
	}

	var second contract.BridgeHealthStreamPayload
	if err := json.Unmarshal(records[1].Data, &second); err != nil {
		t.Fatalf("json.Unmarshal(second snapshot) error = %v", err)
	}
	if got, want := second.BridgeHealth["brg-123"].Status, bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("second status = %q, want %q", got, want)
	}
	if got, want := second.BridgeHealth["brg-123"].RouteCount, 2; got != want {
		t.Fatalf("second route_count = %d, want %d", got, want)
	}
}

func TestStreamBridgeHealthEmitsErrorEventWhenPollingFails(t *testing.T) {
	homePaths := newTestHomePaths(t)
	callCount := 0
	observer := stubObserver{
		QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
			callCount++
			if callCount == 1 {
				return []observe.BridgeInstanceHealth{
					{BridgeInstanceID: "brg-123", Status: bridgepkg.BridgeStatusStarting},
				}, nil
			}
			return nil, errors.New("bridge observer unavailable")
		},
	}
	handlers := newTestHandlersWithBridges(
		t,
		stubSessionManager{},
		observer,
		stubBridgeService{},
		stubWorkspaceService{},
		homePaths,
	)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/bridges/health/stream",
		http.NoBody,
	)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[1].Event != "error" {
		t.Fatalf("records[1].Event = %q, want error", records[1].Event)
	}

	var payload contract.ErrorPayload
	if err := json.Unmarshal(records[1].Data, &payload); err != nil {
		t.Fatalf("json.Unmarshal(error payload) error = %v", err)
	}
	if got, want := payload.Error, "bridge observer unavailable"; got != want {
		t.Fatalf("payload.Error = %q, want %q", got, want)
	}
}

func TestHelperBuildersCoverRemainingBranches(t *testing.T) {
	if acpCapsPayloadFromInfo(acp.Caps{SupportsLoadSession: true}) == nil {
		t.Fatal("expected non-nil caps payload")
	}
	usage := int64(10)
	payload := core.TokenUsagePayloadFromUsage(
		&acp.TokenUsage{InputTokens: &usage, Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)},
	)
	if payload == nil || payload.Timestamp.IsZero() {
		t.Fatalf("TokenUsagePayloadFromUsage() = %#v", payload)
	}
	if !observeEventAfterCursor(
		store.EventSummary{ID: "b", Sequence: 2, Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)},
		observeCursor{Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC), Sequence: 1},
	) {
		t.Fatal("expected event to sort after cursor")
	}

	writer := &bufferFlusher{}
	if err := core.WriteSSE(
		writer,
		core.SSEMessage{ID: "1", Name: "done", Data: map[string]string{"ok": "true"}},
	); err != nil {
		t.Fatalf("writeSSE() error = %v", err)
	}
	if got := writer.String(); got == "" || !bytes.Contains([]byte(got), []byte("event: done")) {
		t.Fatalf("writeSSE output = %q", got)
	}
}

func TestNewHandlersAppliesDefaults(t *testing.T) {
	handlers := newHandlers(&handlerConfig{})
	if handlers.Logger == nil {
		t.Fatal("expected default logger")
	}
	if handlers.Now == nil {
		t.Fatal("expected default clock")
	}
	if handlers.PollInterval != defaultPollInterval {
		t.Fatalf("pollInterval = %v, want %v", handlers.PollInterval, defaultPollInterval)
	}
	if handlers.AgentLoader == nil {
		t.Fatal("expected default agent loader")
	}
	if handlers.StartedAt.IsZero() {
		t.Fatal("expected non-zero startedAt")
	}
}

func TestPayloadAndStatusHelpersCoverRemainingBranches(t *testing.T) {
	if got := string(payloadJSON("")); got != "null" {
		t.Fatalf("payloadJSON(blank) = %q, want null", got)
	}
	if got := string(payloadJSON("plain-text")); got == "" || got == "plain-text" {
		t.Fatalf("payloadJSON(plain-text) = %q, want quoted JSON", got)
	}
	if status := core.StatusForSessionError(os.ErrNotExist); status != http.StatusNotFound {
		t.Fatalf("statusForSessionError(os.ErrNotExist) = %d, want %d", status, http.StatusNotFound)
	}
	if status := core.StatusForSessionError(session.ErrPromptInProgress); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrPromptInProgress) = %d, want %d", status, http.StatusConflict)
	}
	if status := core.StatusForSessionError(workspacepkg.ErrWorkspaceNotFound); status != http.StatusNotFound {
		t.Fatalf("statusForSessionError(ErrWorkspaceNotFound) = %d, want %d", status, http.StatusNotFound)
	}
	if status := core.StatusForSessionError(workspacepkg.ErrWorkspaceRootMissing); status != http.StatusGone {
		t.Fatalf("statusForSessionError(ErrWorkspaceRootMissing) = %d, want %d", status, http.StatusGone)
	}
	if status := statusForWorkspaceError(workspacepkg.ErrWorkspacePathTaken); status != http.StatusConflict {
		t.Fatalf("statusForWorkspaceError(ErrWorkspacePathTaken) = %d, want %d", status, http.StatusConflict)
	}
	if status := statusForWorkspaceError(workspacepkg.ErrWorkspaceHasSessions); status != http.StatusConflict {
		t.Fatalf("statusForWorkspaceError(ErrWorkspaceHasSessions) = %d, want %d", status, http.StatusConflict)
	}
	if status := core.StatusForSessionError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("statusForSessionError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestExtractPromptMessageCoversContentFallbacks(t *testing.T) {
	t.Parallel()

	message, err := extractPromptMessage(promptRequest{
		Messages: []uiMessageEnvelope{{
			Role:    "user",
			Content: "content path",
		}},
	})
	if err != nil || message != "content path" {
		t.Fatalf("extractPromptMessage(content) = %q, %v", message, err)
	}

	message, err = extractPromptMessage(promptRequest{
		Messages: []uiMessageEnvelope{{
			Role: "assistant",
		}, {
			Role: "user",
			Parts: []uiMessageTextPart{
				{Type: "tool-call", Text: "ignored"},
				{Type: "text", Text: "part one"},
				{Type: " text ", Text: "part one point five"},
				{Type: "", Text: "part two"},
			},
		}},
	})
	if err != nil || message != "part one\npart one point five\npart two" {
		t.Fatalf("extractPromptMessage(parts) = %q, %v", message, err)
	}

	if _, err := extractPromptMessage(promptRequest{}); err == nil {
		t.Fatal("extractPromptMessage(empty) error = nil, want non-nil")
	}
}

func TestDrainPromptEventsAsyncIsTracked(t *testing.T) {
	t.Parallel()

	events := make(chan acp.AgentEvent)
	handlers := &Handlers{
		BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{}),
	}

	canceled := make(chan struct{})
	handlers.drainPromptEventsAsync(context.Background(), events, func() {
		close(canceled)
	})
	events <- acp.AgentEvent{Type: acp.EventTypeRuntimeProgress}
	close(events)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := handlers.waitForPromptDrains(ctx); err != nil {
		t.Fatalf("waitForPromptDrains() error = %v", err)
	}
	select {
	case <-canceled:
	default:
		t.Fatal("detached prompt cancel was not called after drain")
	}
}
