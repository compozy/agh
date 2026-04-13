package core_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

func TestBaseHandlersSessionEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	var createCalled atomic.Bool
	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{testutil.NewSessionInfo("sess-a")}, nil
		},
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			createCalled.Store(true)
			if opts.AgentName != "coder" || opts.Workspace != "alpha" || opts.Type != session.SessionTypeUser {
				t.Fatalf("Create opts = %#v", opts)
			}
			created := testutil.NewSession("sess-created")
			created.AgentName = opts.AgentName
			return created, nil
		},
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			if id == "missing" {
				return nil, session.ErrSessionNotFound
			}
			info := testutil.NewSessionInfo(id)
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
			resumed := testutil.NewSession(id)
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
		TranscriptFn: func(_ context.Context, _ string) ([]transcript.Message, error) {
			return []transcript.Message{{
				ID:        "msg-1",
				Role:      transcript.RoleUser,
				Content:   "hello",
				Timestamp: now,
			}}, nil
		},
	}

	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)

	t.Run("ShouldListSessions", func(t *testing.T) {
		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list status = %d, want %d", listResp.Code, http.StatusOK)
		}
	})

	t.Run("ShouldCreateSession", func(t *testing.T) {
		createResp := performRequest(t, fixture.Engine, http.MethodPost, "/sessions", []byte(`{"agent_name":"coder","workspace":"alpha"}`))
		if createResp.Code != http.StatusCreated || !createCalled.Load() {
			t.Fatalf("create status = %d, called=%v", createResp.Code, createCalled.Load())
		}
	})

	t.Run("ShouldGetSession", func(t *testing.T) {
		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a", nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get status = %d, want %d", getResp.Code, http.StatusOK)
		}
	})

	t.Run("ShouldReturnNotFoundForMissingSession", func(t *testing.T) {
		notFoundResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/missing", nil)
		if notFoundResp.Code != http.StatusNotFound {
			t.Fatalf("get missing status = %d, want %d", notFoundResp.Code, http.StatusNotFound)
		}
	})

	t.Run("ShouldStopSession", func(t *testing.T) {
		stopResp := performRequest(t, fixture.Engine, http.MethodDelete, "/sessions/sess-a", nil)
		if stopResp.Code != http.StatusNoContent {
			t.Fatalf("stop status = %d, want %d", stopResp.Code, http.StatusNoContent)
		}
		if got := stopResp.Body.String(); got != "" {
			t.Fatalf("stop body = %q, want empty", got)
		}
	})

	t.Run("ShouldResumeSession", func(t *testing.T) {
		resumeResp := performRequest(t, fixture.Engine, http.MethodPost, "/sessions/sess-a/resume", nil)
		if resumeResp.Code != http.StatusOK {
			t.Fatalf("resume status = %d, want %d", resumeResp.Code, http.StatusOK)
		}
	})

	t.Run("ShouldReturnSessionEvents", func(t *testing.T) {
		eventsResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/events?limit=10&after_sequence=5", nil)
		if eventsResp.Code != http.StatusOK {
			t.Fatalf("events status = %d, want %d", eventsResp.Code, http.StatusOK)
		}
	})

	t.Run("ShouldReturnSessionHistory", func(t *testing.T) {
		historyResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/history", nil)
		if historyResp.Code != http.StatusOK {
			t.Fatalf("history status = %d, want %d", historyResp.Code, http.StatusOK)
		}
	})

	t.Run("ShouldReturnSessionTranscript", func(t *testing.T) {
		transcriptResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/transcript", nil)
		if transcriptResp.Code != http.StatusOK {
			t.Fatalf("transcript status = %d, want %d", transcriptResp.Code, http.StatusOK)
		}
	})
}

func TestBaseHandlersStreamingAndObserveEndpoints(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	var sessionCalls atomic.Int32
	var observeCalls atomic.Int32
	manager := testutil.StubSessionManager{
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			return testutil.NewSessionInfo(id), nil
		},
		EventsFn: func(_ context.Context, id string, _ store.EventQuery) ([]store.SessionEvent, error) {
			switch sessionCalls.Add(1) {
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
			return []*session.SessionInfo{testutil.NewSessionInfo("sess-a")}, nil
		},
	}
	observer := testutil.StubObserver{
		QueryEventsFn: func(_ context.Context, _ store.EventSummaryQuery) ([]store.EventSummary, error) {
			call := observeCalls.Add(1)
			ts := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch call {
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

	fixture := newHandlerFixture(t, manager, observer, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.SetStreamDone(done)

	streamResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/stream", nil)
	if streamResp.Code != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", streamResp.Code, http.StatusOK)
	}
	if records := testutil.ParseSSE(t, streamResp.Body.String()); len(records) < 2 {
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

	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)
	testutil.WriteAgentDef(t, fixture.HomePaths, "coder")

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

func TestDaemonStatusIncludesNetworkDiagnosticsWithoutCredentials(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{{ID: "sess-1"}}, nil
		},
	}
	observer := testutil.StubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok", ActiveSessions: 1, Version: "dev"}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, observer, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.Config.Network.Enabled = true
	fixture.Handlers.Network = testutil.StubNetworkService{
		StatusFn: func(context.Context) (*network.NetworkStatus, error) {
			return &network.NetworkStatus{
				Enabled:      true,
				Status:       network.StatusRunning,
				ListenerHost: "127.0.0.1",
				ListenerPort: 4222,
				LocalPeers:   1,
				RemotePeers:  2,
				Channels:     3,
			}, nil
		},
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/daemon/status", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", resp.Code, http.StatusOK)
	}

	var payload struct {
		Daemon contract.DaemonStatusPayload `json:"daemon"`
	}
	testutil.DecodeJSONResponse(t, resp, &payload)
	if payload.Daemon.Network == nil {
		t.Fatal("daemon network payload = nil, want diagnostics")
	}
	if got, want := payload.Daemon.Network.ListenerPort, 4222; got != want {
		t.Fatalf("daemon network listener port = %d, want %d", got, want)
	}
	if got, want := payload.Daemon.Network.RemotePeers, 2; got != want {
		t.Fatalf("daemon network remote peers = %d, want %d", got, want)
	}
	if got, want := payload.Daemon.Network.Channels, 3; got != want {
		t.Fatalf("daemon network channels = %d, want %d", got, want)
	}
	if strings.Contains(strings.ToLower(resp.Body.String()), "token") {
		t.Fatalf("daemon status leaked credentials: %s", resp.Body.String())
	}
}
