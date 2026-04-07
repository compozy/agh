package udsapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
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
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-123/stream", nil)
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
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
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
	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-123/stream", nil)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1; body=%s", len(records), recorder.Body.String())
	}
	if records[0].Event != session.EventTypeSessionStopped {
		t.Fatalf("records[0].Event = %q, want %q", records[0].Event, session.EventTypeSessionStopped)
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
				return []store.EventSummary{{ID: "sum-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: timestamp}}, nil
			case 2:
				close(done)
				return []store.EventSummary{{ID: "sum-2", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: timestamp.Add(time.Second)}}, nil
			default:
				return nil, nil
			}
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/observe/events/stream", nil)
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[0].ID == records[1].ID {
		t.Fatalf("expected distinct observe SSE ids, got %#v", records)
	}
}

func TestHelperBuildersCoverRemainingBranches(t *testing.T) {
	if acpCapsPayloadFromInfo(acp.ACPCaps{SupportsLoadSession: true}) == nil {
		t.Fatal("expected non-nil caps payload")
	}
	usage := int64(10)
	if tokenUsagePayloadFromUsage(&acp.TokenUsage{InputTokens: &usage}) == nil {
		t.Fatal("expected non-nil token usage payload")
	}
	if !observeEventAfterCursor(store.EventSummary{ID: "b", Sequence: 2, Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)}, observeCursor{Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC), Sequence: 1}) {
		t.Fatal("expected event to sort after cursor")
	}

	writer := &bufferFlusher{}
	if err := core.WriteSSE(writer, core.SSEMessage{ID: "1", Name: "done", Data: map[string]string{"ok": "true"}}); err != nil {
		t.Fatalf("writeSSE() error = %v", err)
	}
	if got := writer.String(); got == "" || !bytes.Contains([]byte(got), []byte("event: done")) {
		t.Fatalf("writeSSE output = %q", got)
	}
}

func TestNewHandlersAppliesDefaults(t *testing.T) {
	handlers := newHandlers(handlerConfig{})
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
