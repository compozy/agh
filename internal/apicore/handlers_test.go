package apicore_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/apitest"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

func TestBaseHandlersSessionEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	createCalled := false
	manager := apitest.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{apitest.NewSessionInfo("sess-a")}, nil
		},
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			createCalled = true
			if opts.AgentName != "coder" || opts.Workspace != "alpha" {
				t.Fatalf("Create opts = %#v", opts)
			}
			created := apitest.NewSession("sess-created")
			created.AgentName = opts.AgentName
			return created, nil
		},
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			if id == "missing" {
				return nil, session.ErrSessionNotFound
			}
			info := apitest.NewSessionInfo(id)
			info.CreatedAt = now
			info.UpdatedAt = now
			return info, nil
		},
		StopFn: func(_ context.Context, id string) error {
			if id != "sess-a" {
				t.Fatalf("Stop id = %q, want sess-a", id)
			}
			return nil
		},
		ResumeFn: func(_ context.Context, id string) (*session.Session, error) {
			resumed := apitest.NewSession(id)
			resumed.State = session.StateActive
			return resumed, nil
		},
		EventsFn: func(_ context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
			if id != "sess-a" || query.Limit != 10 || query.AfterSequence != 5 {
				t.Fatalf("Events call = %q %#v", id, query)
			}
			return []store.SessionEvent{{
				ID:        "ev-1",
				SessionID: id,
				Sequence:  6,
				TurnID:    "turn-1",
				Type:      "agent_message",
				AgentName: "coder",
				Content:   `{"text":"hello"}`,
				Timestamp: now,
			}}, nil
		},
		HistoryFn: func(_ context.Context, id string, _ store.EventQuery) ([]store.TurnHistory, error) {
			return []store.TurnHistory{{
				TurnID: "turn-1",
				Events: []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: id,
					Sequence:  1,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: now,
				}},
			}}, nil
		},
		TranscriptFn: func(_ context.Context, _ string) ([]session.TranscriptMessage, error) {
			return []session.TranscriptMessage{{
				ID:        "msg-1",
				Role:      session.TranscriptRoleUser,
				Content:   "hello",
				Timestamp: now,
			}}, nil
		},
	}

	fixture := newHandlerFixture(t, manager, apitest.StubObserver{}, apitest.StubWorkspaceService{}, nil, nil)

	listResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResp.Code, http.StatusOK)
	}

	createResp := performRequest(t, fixture.Engine, http.MethodPost, "/sessions", []byte(`{"agent_name":"coder","workspace":"alpha"}`))
	if createResp.Code != http.StatusCreated || !createCalled {
		t.Fatalf("create status = %d, called=%v", createResp.Code, createCalled)
	}

	getResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResp.Code, http.StatusOK)
	}

	notFoundResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/missing", nil)
	if notFoundResp.Code != http.StatusNotFound {
		t.Fatalf("get missing status = %d, want %d", notFoundResp.Code, http.StatusNotFound)
	}

	stopResp := performRequest(t, fixture.Engine, http.MethodDelete, "/sessions/sess-a", nil)
	if stopResp.Code != http.StatusOK {
		t.Fatalf("stop status = %d, want %d", stopResp.Code, http.StatusOK)
	}

	resumeResp := performRequest(t, fixture.Engine, http.MethodPost, "/sessions/sess-a/resume", nil)
	if resumeResp.Code != http.StatusOK {
		t.Fatalf("resume status = %d, want %d", resumeResp.Code, http.StatusOK)
	}

	eventsResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/events?limit=10&after_sequence=5", nil)
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("events status = %d, want %d", eventsResp.Code, http.StatusOK)
	}

	historyResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/history", nil)
	if historyResp.Code != http.StatusOK {
		t.Fatalf("history status = %d, want %d", historyResp.Code, http.StatusOK)
	}

	transcriptResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/transcript", nil)
	if transcriptResp.Code != http.StatusOK {
		t.Fatalf("transcript status = %d, want %d", transcriptResp.Code, http.StatusOK)
	}
}

func TestBaseHandlersStreamingAndObserveEndpoints(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	sessionCalls := 0
	observeCalls := 0
	manager := apitest.StubSessionManager{
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			return apitest.NewSessionInfo(id), nil
		},
		EventsFn: func(_ context.Context, id string, _ store.EventQuery) ([]store.SessionEvent, error) {
			sessionCalls++
			switch sessionCalls {
			case 1:
				return []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: id,
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
					SessionID: id,
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
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{apitest.NewSessionInfo("sess-a")}, nil
		},
	}
	observer := apitest.StubObserver{
		QueryEventsFn: func(_ context.Context, _ store.EventSummaryQuery) ([]store.EventSummary, error) {
			observeCalls++
			ts := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch observeCalls {
			case 1:
				return []store.EventSummary{{ID: "sum-1", SessionID: "sess-a", Type: "agent_message", AgentName: "coder", Timestamp: ts}}, nil
			case 2:
				return []store.EventSummary{{ID: "sum-2", SessionID: "sess-a", Type: "done", AgentName: "coder", Timestamp: ts.Add(time.Second)}}, nil
			default:
				return nil, nil
			}
		},
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok", ActiveSessions: 1, Version: "dev"}, nil
		},
	}

	fixture := newHandlerFixture(t, manager, observer, apitest.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.SetStreamDone(done)

	streamResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/stream", nil)
	if streamResp.Code != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", streamResp.Code, http.StatusOK)
	}
	if records := apitest.ParseSSE(t, streamResp.Body.String()); len(records) < 2 {
		t.Fatalf("stream records = %d, want at least 2", len(records))
	}

	observeResp := performRequest(t, fixture.Engine, http.MethodGet, "/observe/events", nil)
	if observeResp.Code != http.StatusOK {
		t.Fatalf("observe status = %d, want %d", observeResp.Code, http.StatusOK)
	}

	healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/observe/health", nil)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthResp.Code, http.StatusOK)
	}

	statusResp := performRequest(t, fixture.Engine, http.MethodGet, "/daemon/status", nil)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", statusResp.Code, http.StatusOK)
	}
}

func TestBaseHandlersAgentEndpoints(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(t, apitest.StubSessionManager{}, apitest.StubObserver{}, apitest.StubWorkspaceService{}, nil, nil)
	apitest.WriteAgentDef(t, fixture.HomePaths, "coder")

	getResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/coder", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get agent status = %d, want %d", getResp.Code, http.StatusOK)
	}

	listResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list agents status = %d, want %d", listResp.Code, http.StatusOK)
	}

	fixture.Handlers.AgentLoader = func(string, aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{}, errors.New("boom")
	}
	missingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
	if missingResp.Code != http.StatusInternalServerError {
		t.Fatalf("missing agent status = %d, want %d", missingResp.Code, http.StatusInternalServerError)
	}
}

func TestBaseHandlersApprovePermissionGapResolvedInStub(t *testing.T) {
	t.Parallel()

	manager := apitest.StubSessionManager{
		ApproveFn: func(_ context.Context, id string, req acp.ApproveRequest) error {
			if id != "sess-a" || req.TurnID != "turn-1" {
				t.Fatalf("ApprovePermission call = %q %#v", id, req)
			}
			return nil
		},
	}

	if err := manager.ApprovePermission(context.Background(), "sess-a", acp.ApproveRequest{TurnID: "turn-1"}); err != nil {
		t.Fatalf("ApprovePermission() error = %v", err)
	}
}
