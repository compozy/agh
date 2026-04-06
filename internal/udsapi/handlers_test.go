package udsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func TestRegisterRoutesCoversTechSpecEndpoints(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	routes := engine.Routes()
	got := make([]string, 0, len(routes))
	for _, route := range routes {
		got = append(got, route.Method+" "+route.Path)
	}
	sort.Strings(got)

	want := []string{
		"DELETE /api/memory/:filename",
		"DELETE /api/sessions/:id",
		"GET /api/agents",
		"GET /api/agents/:name",
		"GET /api/daemon/status",
		"GET /api/memory",
		"GET /api/memory/:filename",
		"GET /api/observe/events",
		"GET /api/observe/events/stream",
		"GET /api/observe/health",
		"GET /api/sessions",
		"GET /api/sessions/:id",
		"GET /api/sessions/:id/events",
		"GET /api/sessions/:id/history",
		"GET /api/sessions/:id/stream",
		"POST /api/memory/consolidate",
		"POST /api/sessions",
		"POST /api/sessions/:id/approve",
		"POST /api/sessions/:id/prompt",
		"POST /api/sessions/:id/resume",
		"PUT /api/memory/:filename",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("len(routes) = %d, want %d\nroutes=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("route[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCreateSessionHandlerReturnsSessionID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		createFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "coder" || opts.Name != "demo" || opts.Workspace != "/workspace" {
				t.Fatalf("Create() opts = %#v", opts)
			}
			return newSession("sess-123"), nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder","name":"demo","workspace":"/workspace"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Session sessionPayload `json:"session"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Session.ID != "sess-123" {
		t.Fatalf("session.id = %q, want %q", response.Session.ID, "sess-123")
	}
}

func TestCreateSessionHandlerAllowsMissingAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		createFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "" {
				t.Fatalf("Create() AgentName = %q, want empty", opts.AgentName)
			}
			return newSession("sess-default"), nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"name":"demo"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
}

func TestListSessionsHandlerReturnsAllSessions(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-a"), newSessionInfo("sess-b")}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(response.Sessions))
	}
}

func TestStopSessionHandlerReturnsStopped(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		stopFn: func(_ context.Context, id string) error {
			if id != "sess-123" {
				t.Fatalf("Stop() id = %q, want sess-123", id)
			}
			return nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodDelete, "/api/sessions/sess-123", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestResumeSessionHandlerReturnsSession(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		resumeFn: func(_ context.Context, id string) (*session.Session, error) {
			if id != "sess-123" {
				t.Fatalf("Resume() id = %q, want sess-123", id)
			}
			return newSession("sess-123"), nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/resume", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestPromptSessionHandlerReturnsSSEStream(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		promptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
			ch := make(chan acp.AgentEvent, 2)
			ch <- acp.AgentEvent{
				Type:      "agent_message",
				TurnID:    "turn-1",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				Text:      "hello",
			}
			ch <- acp.AgentEvent{
				Type:       "done",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				StopReason: "end_turn",
			}
			close(ch)
			return ch, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"message":"hello"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
	}
	if records[0].Event != "agent_message" || records[1].Event != "done" {
		t.Fatalf("events = [%s %s], want [agent_message done]", records[0].Event, records[1].Event)
	}
}

func TestPromptSessionHandlerRejectsEmptyMessage(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"message":""}`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestSessionEventsHandlerReturnsFilteredEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		eventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
			gotQuery = query
			return []store.SessionEvent{{
				ID:        "ev-1",
				SessionID: "sess-123",
				Sequence:  7,
				TurnID:    "turn-1",
				Type:      "agent_message",
				AgentName: "coder",
				Content:   `{"text":"hello"}`,
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/events?type=agent_message&agent_name=coder&turn_id=turn-1&after_sequence=5&limit=10&since=2026-04-03T12:00:00Z", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotQuery.Type != "agent_message" || gotQuery.AgentName != "coder" || gotQuery.TurnID != "turn-1" || gotQuery.AfterSequence != 5 || gotQuery.Limit != 10 {
		t.Fatalf("query = %#v", gotQuery)
	}

	var response struct {
		Events []sessionEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Events) != 1 || response.Events[0].Sequence != 7 {
		t.Fatalf("events = %#v", response.Events)
	}
}

func TestSessionHistoryHandlerReturnsTurns(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		historyFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
			return []store.TurnHistory{{
				TurnID: "turn-1",
				Events: []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: "sess-123",
					Sequence:  1,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				}},
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/history", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		History []turnHistoryPayload `json:"history"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.History) != 1 || response.History[0].TurnID != "turn-1" {
		t.Fatalf("history = %#v", response.History)
	}
}

func TestStreamSessionHandlerUsesLastEventID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		statusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
		eventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
			gotQuery = query
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
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	done := make(chan struct{})
	close(done)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-123/stream", nil)
	req.Header.Set("Last-Event-ID", "1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if gotQuery.AfterSequence != 1 {
		t.Fatalf("AfterSequence = %d, want 1", gotQuery.AfterSequence)
	}
	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 || records[0].ID != "2" || records[0].Event != "done" {
		t.Fatalf("records = %#v", records)
	}
}

func TestListAgentsHandlerReturnsAvailableAgents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")
	writeAgentDef(t, homePaths, "researcher")
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/agents", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Agents []agentPayload `json:"agents"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Agents) != 2 {
		t.Fatalf("len(agents) = %d, want 2", len(response.Agents))
	}
	if response.Agents[0].Name != "coder" {
		t.Fatalf("first agent = %q, want coder", response.Agents[0].Name)
	}
}

func TestGetAgentHandlerReturnsAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/agents/coder", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Agent agentPayload `json:"agent"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Agent.Name != "coder" || response.Agent.Provider != "fake" {
		t.Fatalf("agent = %#v", response.Agent)
	}
}

func TestObserveEventsHandlerReturnsEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		queryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{{
				ID:        "sum-1",
				SessionID: "sess-123",
				Type:      "agent_message",
				AgentName: "coder",
				Summary:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/observe/events", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Events []observeEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Events) != 1 || response.Events[0].ID != "sum-1" {
		t.Fatalf("events = %#v", response.Events)
	}
}

func TestHealthHandlerReturnsMetrics(t *testing.T) {
	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		healthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{
				Status:         "ok",
				ActiveSessions: 3,
			}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/observe/health", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Health observe.Health `json:"health"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Health.ActiveSessions != 3 {
		t.Fatalf("health.active_sessions = %d, want 3", response.Health.ActiveSessions)
	}
}

func TestDaemonStatusHandlerReturnsRunningState(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
		},
	}
	observer := stubObserver{
		healthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok", ActiveSessions: 1, Version: "dev"}, nil
		},
	}
	handlers := newTestHandlers(t, manager, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/daemon/status", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Daemon daemonStatusPayload `json:"daemon"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Daemon.Status != "running" {
		t.Fatalf("daemon.status = %q, want running", response.Daemon.Status)
	}
	if response.Daemon.TotalSessions != 1 {
		t.Fatalf("daemon.total_sessions = %d, want 1", response.Daemon.TotalSessions)
	}
}

func TestHelperParsersAndPayloads(t *testing.T) {
	if _, err := parseOptionalTime("bad-time"); err == nil {
		t.Fatal("parseOptionalTime() error = nil, want non-nil")
	}
	if _, err := parseOptionalInt("bad"); err == nil {
		t.Fatal("parseOptionalInt() error = nil, want non-nil")
	}
	if _, err := parseOptionalInt64("bad"); err == nil {
		t.Fatal("parseOptionalInt64() error = nil, want non-nil")
	}
	if _, err := parseObserveCursor("bad"); err == nil {
		t.Fatal("parseObserveCursor() error = nil, want non-nil")
	}
	if got := string(payloadJSON("not-json")); got != `"not-json"` {
		t.Fatalf("payloadJSON(non-json) = %s, want %q", got, `"not-json"`)
	}
	if tokenUsagePayloadFromUsage(nil) != nil {
		t.Fatal("tokenUsagePayloadFromUsage(nil) = non-nil, want nil")
	}
}

func TestSessionErrorMappingUsesNotFoundAndConflict(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		statusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return nil, session.ErrSessionNotFound
		},
		createFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
			return nil, session.ErrMaxSessionsReached
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	getResp := performRequest(t, engine, http.MethodGet, "/api/sessions/missing", nil)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("GET /api/sessions/:id status = %d, want 404", getResp.Code)
	}

	postResp := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder"}`))
	if postResp.Code != http.StatusConflict {
		t.Fatalf("POST /api/sessions status = %d, want 409", postResp.Code)
	}
}

func TestObserveEventStreamUsesLastEventIDCursor(t *testing.T) {
	homePaths := newTestHomePaths(t)
	timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	observer := stubObserver{
		queryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{
				{ID: "sum-a", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: timestamp},
				{ID: "sum-b", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: timestamp},
			}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	done := make(chan struct{})
	close(done)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequest(http.MethodGet, "/api/observe/events/stream", nil)
	req.Header.Set("Last-Event-ID", timestamp.Format(time.RFC3339Nano)+"|sum-a")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) == 0 {
		t.Fatalf("expected at least one SSE record, got body=%s", recorder.Body.String())
	}
	if records[0].ID != timestamp.Format(time.RFC3339Nano)+"|sum-b" {
		t.Fatalf("record id = %q, want %q", records[0].ID, timestamp.Format(time.RFC3339Nano)+"|sum-b")
	}
}
