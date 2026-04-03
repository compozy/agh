package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/observe"
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
		statusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
		eventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
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

func TestStreamObserveEventsPollsForNewEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	done := make(chan struct{})
	callCount := 0
	observer := stubObserver{
		queryEventsFn: func(context.Context, observe.EventQuery) ([]observe.Event, error) {
			callCount++
			timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch callCount {
			case 1:
				return []observe.Event{{ID: "sum-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: timestamp}}, nil
			case 2:
				close(done)
				return []observe.Event{{ID: "sum-2", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: timestamp.Add(time.Second)}}, nil
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
	if got := cloneStringMap(map[string]string{"A": "1"}); got["A"] != "1" {
		t.Fatalf("cloneStringMap() = %#v", got)
	}
	if acpCapsPayloadFromInfo(session.ACPCaps{SupportsLoadSession: true}) == nil {
		t.Fatal("expected non-nil caps payload")
	}
	usage := int64(10)
	payload := tokenUsagePayloadFromUsage(&session.TokenUsage{InputTokens: &usage, Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)})
	if payload == nil || payload.Timestamp == "" {
		t.Fatalf("tokenUsagePayloadFromUsage() = %#v", payload)
	}
	if !observeEventAfterCursor(observe.Event{ID: "b", Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)}, observeCursor{Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC), ID: "a"}) {
		t.Fatal("expected event to sort after cursor")
	}

	writer := &bufferFlusher{}
	if err := writeSSE(writer, sseMessage{ID: "1", Name: "done", Data: map[string]string{"ok": "true"}}); err != nil {
		t.Fatalf("writeSSE() error = %v", err)
	}
	if got := writer.String(); got == "" || !bytes.Contains([]byte(got), []byte("event: done")) {
		t.Fatalf("writeSSE output = %q", got)
	}
}

func TestNewHandlersAppliesDefaults(t *testing.T) {
	handlers := newHandlers(handlerConfig{})
	if handlers.logger == nil {
		t.Fatal("expected default logger")
	}
	if handlers.now == nil {
		t.Fatal("expected default clock")
	}
	if handlers.pollInterval != defaultPollInterval {
		t.Fatalf("pollInterval = %v, want %v", handlers.pollInterval, defaultPollInterval)
	}
	if handlers.agentLoader == nil {
		t.Fatal("expected default agent loader")
	}
	if handlers.startedAt.IsZero() {
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
	if status := statusForSessionError(os.ErrNotExist); status != http.StatusNotFound {
		t.Fatalf("statusForSessionError(os.ErrNotExist) = %d, want %d", status, http.StatusNotFound)
	}
	if status := statusForSessionError(session.ErrMaxSessionsReached); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrMaxSessionsReached) = %d, want %d", status, http.StatusConflict)
	}
	if status := statusForSessionError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("statusForSessionError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestExtractPromptMessageCoversContentFallbacks(t *testing.T) {
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
				{Type: "", Text: "part two"},
			},
		}},
	})
	if err != nil || message != "part one\npart two" {
		t.Fatalf("extractPromptMessage(parts) = %q, %v", message, err)
	}

	if _, err := extractPromptMessage(promptRequest{}); err == nil {
		t.Fatal("extractPromptMessage(empty) error = nil, want non-nil")
	}
}
