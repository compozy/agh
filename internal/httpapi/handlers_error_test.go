package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
)

func TestCreateGetResumeAndStopHandlersReturnExpectedErrors(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		createFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
			return nil, os.ErrNotExist
		},
		statusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return nil, session.ErrSessionNotFound
		},
		resumeFn: func(context.Context, string) (*session.Session, error) {
			return nil, session.ErrSessionNotFound
		},
		stopFn: func(context.Context, string) error {
			return session.ErrSessionNotFound
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	createResp := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder"}`))
	if createResp.Code != http.StatusNotFound {
		t.Fatalf("create status = %d, want %d", createResp.Code, http.StatusNotFound)
	}

	getResp := performRequest(t, engine, http.MethodGet, "/api/sessions/missing", nil)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("GET status = %d, want %d", getResp.Code, http.StatusNotFound)
	}

	resumeResp := performRequest(t, engine, http.MethodPost, "/api/sessions/missing/resume", nil)
	if resumeResp.Code != http.StatusNotFound {
		t.Fatalf("resume status = %d, want %d", resumeResp.Code, http.StatusNotFound)
	}

	stopResp := performRequest(t, engine, http.MethodDelete, "/api/sessions/missing", nil)
	if stopResp.Code != http.StatusNotFound {
		t.Fatalf("stop status = %d, want %d", stopResp.Code, http.StatusNotFound)
	}
}

func TestHandlersRejectBadPromptAndQueryValues(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		statusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	badPrompt := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"message":""}`))
	if badPrompt.Code != http.StatusBadRequest {
		t.Fatalf("bad prompt status = %d, want %d", badPrompt.Code, http.StatusBadRequest)
	}

	eventsResp := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/events?since=bad", nil)
	if eventsResp.Code != http.StatusBadRequest {
		t.Fatalf("events bad query status = %d, want %d", eventsResp.Code, http.StatusBadRequest)
	}

	streamResp := performRequestWithHeaders(t, engine, http.MethodGet, "/api/sessions/sess-123/stream", nil, map[string]string{"Last-Event-ID": "bad"})
	if streamResp.Code != http.StatusBadRequest {
		t.Fatalf("stream bad header status = %d, want %d", streamResp.Code, http.StatusBadRequest)
	}
}

func TestPromptSessionHandlerCoversThoughtPermissionAndErrorBranches(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		promptFn: func(context.Context, string, string) (<-chan session.AgentEvent, error) {
			ch := make(chan session.AgentEvent, 3)
			ch <- session.AgentEvent{
				Type:      "thought",
				TurnID:    "turn-err",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				Text:      "thinking",
			}
			ch <- session.AgentEvent{
				Type:      "permission",
				TurnID:    "turn-err",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				Action:    "fs/read_text_file",
				Decision:  "allow",
			}
			ch <- session.AgentEvent{
				Type:      "error",
				TurnID:    "turn-err",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC),
				Error:     "boom",
			}
			close(ch)
			return ch, nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	resp := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"message":"hello"}`))
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	records := parseSSE(t, resp.Body.String())
	if len(records) < 5 {
		t.Fatalf("records = %d, want at least 5; body=%s", len(records), resp.Body.String())
	}
	if string(records[len(records)-1].Data) != "[DONE]" {
		t.Fatalf("last record data = %q, want [DONE]", string(records[len(records)-1].Data))
	}

	var partTypes []string
	for _, record := range records[:len(records)-1] {
		if len(record.Data) == 0 {
			continue
		}
		var part map[string]any
		if err := json.Unmarshal(record.Data, &part); err != nil {
			t.Fatalf("json.Unmarshal(part) error = %v; data=%s", err, string(record.Data))
		}
		if value, ok := part["type"].(string); ok {
			partTypes = append(partTypes, value)
		}
	}
	if !contains(partTypes, "reasoning-start") || !contains(partTypes, "reasoning-delta") || !contains(partTypes, "reasoning-end") || !contains(partTypes, "data-agh-permission") || !contains(partTypes, "error") || !contains(partTypes, "finish") {
		t.Fatalf("part types = %#v", partTypes)
	}
}

func TestAgentObserveHealthAndDaemonStatusErrorPaths(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{
		queryEventsFn: func(context.Context, observe.EventQuery) ([]observe.Event, error) {
			return nil, errors.New("boom")
		},
		healthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{}, errors.New("health failed")
		},
	}, homePaths)
	handlers.agentLoader = func(_ string, _ aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{}, os.ErrNotExist
	}
	engine := newTestRouter(t, handlers)

	agentResp := performRequest(t, engine, http.MethodGet, "/api/agents/missing", nil)
	if agentResp.Code != http.StatusNotFound {
		t.Fatalf("agent status = %d, want %d", agentResp.Code, http.StatusNotFound)
	}

	observeResp := performRequest(t, engine, http.MethodGet, "/api/observe/events", nil)
	if observeResp.Code != http.StatusInternalServerError {
		t.Fatalf("observe status = %d, want %d", observeResp.Code, http.StatusInternalServerError)
	}

	healthResp := performRequest(t, engine, http.MethodGet, "/api/observe/health", nil)
	if healthResp.Code != http.StatusInternalServerError {
		t.Fatalf("health status = %d, want %d", healthResp.Code, http.StatusInternalServerError)
	}

	statusHandlers := newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, errors.New("list failed")
		},
	}, stubObserver{
		healthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok"}, nil
		},
	}, homePaths)
	statusEngine := newTestRouter(t, statusHandlers)
	statusResp := performRequest(t, statusEngine, http.MethodGet, "/api/daemon/status", nil)
	if statusResp.Code != http.StatusInternalServerError {
		t.Fatalf("daemon status = %d, want %d", statusResp.Code, http.StatusInternalServerError)
	}
}

func TestCORSMiddlewareRejectsDisallowedOrigins(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/sessions", nil)
	req.Header.Set("Origin", "https://evil.example")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSMiddlewareAllowsLoopbackOrigins(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, nil
		},
	}, stubObserver{}, homePaths))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/sessions", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}

func TestRespondErrorSanitizesInternalFailures(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, errors.New("secret internal path")
		},
	}, stubObserver{}, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var payload errorPayload
	decodeJSONResponse(t, recorder, &payload)
	if payload.Error != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("error payload = %q, want %q", payload.Error, http.StatusText(http.StatusInternalServerError))
	}
}

func TestObserveStreamBadHeaderAndMissingAgentsDir(t *testing.T) {
	homePaths := newTestHomePaths(t)
	if err := os.RemoveAll(homePaths.AgentsDir); err != nil {
		t.Fatalf("os.RemoveAll(AgentsDir) error = %v", err)
	}
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	agentsResp := performRequest(t, engine, http.MethodGet, "/api/agents", nil)
	if agentsResp.Code != http.StatusOK {
		t.Fatalf("agents status = %d, want %d", agentsResp.Code, http.StatusOK)
	}

	observeResp := performRequestWithHeaders(t, engine, http.MethodGet, "/api/observe/events/stream", nil, map[string]string{"Last-Event-ID": "bad"})
	if observeResp.Code != http.StatusBadRequest {
		t.Fatalf("observe stream status = %d, want %d", observeResp.Code, http.StatusBadRequest)
	}
}
