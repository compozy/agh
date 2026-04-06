package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

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

func TestPromptSessionHandlerReturnsAISDKSSEStream(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		promptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
			ch := make(chan acp.AgentEvent, 4)
			ch <- acp.AgentEvent{
				Type:      "agent_message",
				TurnID:    "turn-1",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				Text:      "hello",
			}
			ch <- acp.AgentEvent{
				Type:       "tool_call",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				Title:      "read_file",
				ToolCallID: "call-1",
			}
			ch <- acp.AgentEvent{
				Type:       "tool_result",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC),
				ToolCallID: "call-1",
			}
			ch <- acp.AgentEvent{
				Type:       "done",
				TurnID:     "turn-1",
				Timestamp:  time.Date(2026, 4, 3, 12, 0, 3, 0, time.UTC),
				StopReason: "end_turn",
			}
			close(ch)
			return ch, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-123/prompt", []byte(`{"messages":[{"role":"user","parts":[{"type":"text","text":"hello"}]}]}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
	if got := recorder.Header().Get("x-vercel-ai-ui-message-stream"); got != "v1" {
		t.Fatalf("x-vercel-ai-ui-message-stream = %q, want v1", got)
	}

	records := parseSSE(t, recorder.Body.String())
	if len(records) < 5 {
		t.Fatalf("len(records) = %d, want at least 5; body=%s", len(records), recorder.Body.String())
	}
	if string(records[len(records)-1].Data) != "[DONE]" {
		t.Fatalf("last data = %q, want [DONE]", string(records[len(records)-1].Data))
	}

	var foundAgentMessage bool
	var foundToolCall bool
	var foundDone bool
	for _, record := range records {
		if record.Event == "agent_message" {
			foundAgentMessage = true
		}
		if record.Event == "tool_call" {
			foundToolCall = true
		}
		if record.Event == "done" {
			foundDone = true
		}
	}
	if !foundAgentMessage || !foundToolCall || !foundDone {
		t.Fatalf("events missing native markers: agent_message=%v tool_call=%v done=%v body=%s", foundAgentMessage, foundToolCall, foundDone, recorder.Body.String())
	}

	var promptParts []map[string]any
	for _, record := range records[:len(records)-1] {
		if len(record.Data) == 0 || string(record.Data) == "[DONE]" {
			continue
		}
		var part map[string]any
		if err := json.Unmarshal(record.Data, &part); err != nil {
			t.Fatalf("json.Unmarshal(part) error = %v; data=%s", err, string(record.Data))
		}
		promptParts = append(promptParts, part)
	}

	types := make([]string, 0, len(promptParts))
	for _, part := range promptParts {
		if value, ok := part["type"].(string); ok {
			types = append(types, value)
		}
	}
	if !contains(types, "start") || !contains(types, "text-start") || !contains(types, "text-delta") || !contains(types, "tool-input-start") || !contains(types, "tool-output-available") || !contains(types, "finish") {
		t.Fatalf("part types = %#v", types)
	}
}

func TestSessionEventsAndHistoryHandlers(t *testing.T) {
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
		historyFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
			return []store.TurnHistory{{
				TurnID: "turn-1",
				Events: []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: "sess-123",
					Sequence:  7,
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

	eventsResp := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/events?type=agent_message&agent_name=coder&turn_id=turn-1&after_sequence=5&limit=10&since=2026-04-03T12:00:00Z", nil)
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", eventsResp.Code, http.StatusOK, eventsResp.Body.String())
	}
	if gotQuery.Type != "agent_message" || gotQuery.AgentName != "coder" || gotQuery.TurnID != "turn-1" || gotQuery.AfterSequence != 5 || gotQuery.Limit != 10 {
		t.Fatalf("query = %#v", gotQuery)
	}

	historyResp := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/history", nil)
	if historyResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", historyResp.Code, http.StatusOK, historyResp.Body.String())
	}
}

func TestListAgentsAndHealthHandlers(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")

	handlers := newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
		},
	}, stubObserver{
		healthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{
				Status:         "ok",
				UptimeSeconds:  5,
				ActiveSessions: 1,
				ActiveAgents:   1,
				Version:        "dev",
			}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	agentsResp := performRequest(t, engine, http.MethodGet, "/api/agents", nil)
	if agentsResp.Code != http.StatusOK {
		t.Fatalf("agents status = %d, want %d; body=%s", agentsResp.Code, http.StatusOK, agentsResp.Body.String())
	}
	var agents struct {
		Agents []agentPayload `json:"agents"`
	}
	decodeJSONResponse(t, agentsResp, &agents)
	if len(agents.Agents) != 1 || agents.Agents[0].Name != "coder" {
		t.Fatalf("agents = %#v", agents.Agents)
	}

	healthResp := performRequest(t, engine, http.MethodGet, "/api/observe/health", nil)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d; body=%s", healthResp.Code, http.StatusOK, healthResp.Body.String())
	}
	var health struct {
		Health observe.Health `json:"health"`
	}
	decodeJSONResponse(t, healthResp, &health)
	if health.Health.Status != "ok" || health.Health.ActiveSessions != 1 {
		t.Fatalf("health = %#v", health.Health)
	}
}

func TestObserveEventsAndApproveHandlers(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{
		queryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{{
				ID:        "sum-1",
				SessionID: "sess-1",
				Type:      "agent_message",
				AgentName: "coder",
				Summary:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	observeResp := performRequest(t, engine, http.MethodGet, "/api/observe/events?session_id=sess-1", nil)
	if observeResp.Code != http.StatusOK {
		t.Fatalf("observe status = %d, want %d; body=%s", observeResp.Code, http.StatusOK, observeResp.Body.String())
	}
	var observed struct {
		Events []observeEventPayload `json:"events"`
	}
	decodeJSONResponse(t, observeResp, &observed)
	if len(observed.Events) != 1 || observed.Events[0].ID != "sum-1" {
		t.Fatalf("events = %#v", observed.Events)
	}

	approveResp := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", nil)
	if approveResp.Code != http.StatusBadRequest {
		t.Fatalf("approve status = %d, want %d", approveResp.Code, http.StatusBadRequest)
	}
}

func TestApproveSessionHandlerValidatesAndRoutes(t *testing.T) {
	homePaths := newTestHomePaths(t)

	t.Run("missing decision", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("invalid decision", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"maybe"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("session not found", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			approveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrSessionNotFound
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/missing/approve", []byte(`{"turn_id":"turn-1","decision":"allow-once"}`))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
		}
	})

	t.Run("pending permission missing", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			approveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrPendingPermissionNotFound
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"reject-once"}`))
		if recorder.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
		}
	})

	t.Run("session not active", func(t *testing.T) {
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			approveFn: func(context.Context, string, acp.ApproveRequest) error {
				return session.ErrSessionNotActive
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"turn_id":"turn-1","decision":"reject-once"}`))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
		}
	})

	t.Run("valid request", func(t *testing.T) {
		var (
			gotID  string
			gotReq acp.ApproveRequest
		)
		engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
			approveFn: func(_ context.Context, id string, req acp.ApproveRequest) error {
				gotID = id
				gotReq = req
				return nil
			},
		}, stubObserver{}, homePaths))
		recorder := performRequest(t, engine, http.MethodPost, "/api/sessions/sess-1/approve", []byte(`{"request_id":"req-1","turn_id":"turn-1","decision":"allow-always"}`))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if gotID != "sess-1" {
			t.Fatalf("approve id = %q, want sess-1", gotID)
		}
		if gotReq.RequestID != "req-1" || gotReq.TurnID != "turn-1" || gotReq.Decision != "allow-always" {
			t.Fatalf("approve request = %#v", gotReq)
		}
	})
}

func TestErrorResponsesUseConsistentShape(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, context.DeadlineExceeded
		},
	}, stubObserver{}, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}

	var payload errorPayload
	decodeJSONResponse(t, recorder, &payload)
	if payload.Error == "" {
		t.Fatal("expected non-empty error payload")
	}
}

func TestStatusForSessionErrorIncludesApprovalCases(t *testing.T) {
	if status := statusForSessionError(session.ErrSessionNotActive); status != http.StatusBadRequest {
		t.Fatalf("statusForSessionError(ErrSessionNotActive) = %d, want %d", status, http.StatusBadRequest)
	}
	if status := statusForSessionError(session.ErrPendingPermissionNotFound); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrPendingPermissionNotFound) = %d, want %d", status, http.StatusConflict)
	}
	if status := statusForSessionError(session.ErrPendingPermissionConflict); status != http.StatusConflict {
		t.Fatalf("statusForSessionError(ErrPendingPermissionConflict) = %d, want %d", status, http.StatusConflict)
	}
	if status := statusForSessionError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("statusForSessionError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
}

func TestCORSHeadersPresentOnResponses(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{
		listAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{}, nil
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
