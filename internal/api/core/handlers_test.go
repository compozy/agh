package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestBaseHandlersSessionEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	var createCalled atomic.Bool
	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{testutil.NewSessionInfo("sess-a")}, nil
		},
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			createCalled.Store(true)
			if opts.AgentName != "coder" ||
				opts.Provider != "fake" ||
				opts.Workspace != "alpha" ||
				opts.Type != session.SessionTypeUser {
				t.Fatalf("Create opts = %#v", opts)
			}
			created := testutil.NewSession("sess-created")
			created.AgentName = opts.AgentName
			created.Provider = opts.Provider
			return created, nil
		},
		StatusFn: func(_ context.Context, id string) (*session.Info, error) {
			if id == "missing" {
				return nil, session.ErrSessionNotFound
			}
			info := testutil.NewSessionInfo(id)
			info.CreatedAt = now
			info.UpdatedAt = now
			return info, nil
		},
		DeleteFn: func(_ context.Context, id string) error {
			if id != "sess-a" {
				t.Fatalf("Delete id = %q, want sess-a", id)
			}
			return nil
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
		TranscriptFn: func(_ context.Context, _ string) ([]transcript.UIMessage, error) {
			return []transcript.UIMessage{{
				ID:   "msg-1",
				Role: transcript.UIRoleUser,
				Parts: []transcript.UIMessagePart{{
					Type:  "text",
					Text:  "hello",
					State: "done",
				}},
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
		createResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/sessions",
			[]byte(`{"agent_name":"coder","provider":"fake","workspace":"alpha"}`),
		)
		if createResp.Code != http.StatusCreated || !createCalled.Load() {
			t.Fatalf("create status = %d, called=%v", createResp.Code, createCalled.Load())
		}
		var payload contract.SessionResponse
		if err := json.Unmarshal(createResp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(create response) error = %v", err)
		}
		if payload.Session.Provider != "fake" {
			t.Fatalf("created session provider = %q, want %q", payload.Session.Provider, "fake")
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

	t.Run("ShouldDeleteSession", func(t *testing.T) {
		deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/sessions/sess-a", nil)
		if deleteResp.Code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want %d", deleteResp.Code, http.StatusNoContent)
		}
		if got := deleteResp.Body.String(); got != "" {
			t.Fatalf("delete body = %q, want empty", got)
		}
	})

	t.Run("ShouldStopSession", func(t *testing.T) {
		stopResp := performRequest(t, fixture.Engine, http.MethodPost, "/sessions/sess-a/stop", nil)
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
		eventsResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/sessions/sess-a/events?limit=10&after_sequence=5",
			nil,
		)
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
		StatusFn: func(_ context.Context, id string) (*session.Info, error) {
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
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{testutil.NewSessionInfo("sess-a")}, nil
		},
	}
	observer := testutil.StubObserver{
		QueryEventsFn: func(_ context.Context, _ store.EventSummaryQuery) ([]store.EventSummary, error) {
			call := observeCalls.Add(1)
			ts := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch call {
			case 1:
				return []store.EventSummary{
					{ID: "sum-1", SessionID: "sess-a", Type: "agent_message", AgentName: "coder", Timestamp: ts},
				}, nil
			case 2:
				return []store.EventSummary{
					{
						ID:        "sum-2",
						SessionID: "sess-a",
						Type:      "done",
						AgentName: "coder",
						Timestamp: ts.Add(time.Second),
					},
				}, nil
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

	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
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

func TestBaseHandlersAgentCatalogEndpoints(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	fixture.Handlers.AgentCatalog = stubAgentCatalog{
		agents: []aghconfig.AgentDef{
			{Name: "zeta", Prompt: "Zeta prompt"},
			{Name: "alpha", Prompt: "Alpha prompt"},
		},
		get: map[string]aghconfig.AgentDef{
			"alpha": {Name: "alpha", Prompt: "Alpha prompt"},
		},
	}

	listResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list agent catalog status = %d, want %d", listResp.Code, http.StatusOK)
	}
	var listed contract.AgentsResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(list agents) error = %v", err)
	}
	if len(listed.Agents) != 2 || listed.Agents[0].Name != "alpha" || listed.Agents[1].Name != "zeta" {
		t.Fatalf("listed agents = %#v, want alpha then zeta", listed.Agents)
	}

	getResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/alpha", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get agent catalog status = %d, want %d", getResp.Code, http.StatusOK)
	}

	fixture.Handlers.AgentCatalog = stubAgentCatalog{getErr: os.ErrNotExist}
	missingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("get missing catalog agent status = %d, want %d", missingResp.Code, http.StatusNotFound)
	}

	fixture.Handlers.AgentCatalog = stubAgentCatalog{listErr: os.ErrNotExist}
	missingListResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if missingListResp.Code != http.StatusOK {
		t.Fatalf("list missing catalog status = %d, want %d", missingListResp.Code, http.StatusOK)
	}
	var missingList contract.AgentsResponse
	if err := json.Unmarshal(missingListResp.Body.Bytes(), &missingList); err != nil {
		t.Fatalf("json.Unmarshal(missing list agents) error = %v", err)
	}
	if len(missingList.Agents) != 0 {
		t.Fatalf("missing catalog agents = %#v, want empty list", missingList.Agents)
	}

	fixture.Handlers.AgentCatalog = stubAgentCatalog{listErr: errors.New("catalog unavailable")}
	errorResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if errorResp.Code != http.StatusInternalServerError {
		t.Fatalf("list catalog error status = %d, want %d", errorResp.Code, http.StatusInternalServerError)
	}
}

func TestBaseHandlersWorkspaceAgentEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("Should list and inspect workspace resolved agents", func(t *testing.T) {
		t.Parallel()

		const workspaceRef = "alpha"
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{
				ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
					if ref != workspaceRef {
						t.Fatalf("Resolve() ref = %q, want %q", ref, workspaceRef)
					}
					return workspacepkg.ResolvedWorkspace{
						Workspace: workspacepkg.Workspace{ID: "ws-1", Name: workspaceRef},
						Agents: []aghconfig.AgentDef{
							{Name: "founder", Provider: "codex", Prompt: "Lead the startup."},
							{Name: "qa", Provider: "codex", Prompt: "Stress test the release."},
						},
					}, nil
				},
			},
			nil,
			nil,
		)
		fixture.Handlers.AgentCatalog = stubAgentCatalog{
			agents: []aghconfig.AgentDef{
				{Name: "extension-agent", Provider: "codex", Prompt: "Projected by extension."},
			},
		}

		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents?workspace="+workspaceRef, nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf(
				"list workspace agents status = %d, want %d; body = %s",
				listResp.Code,
				http.StatusOK,
				listResp.Body.String(),
			)
		}
		var listed contract.AgentsResponse
		if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
			t.Fatalf("json.Unmarshal(list workspace agents) error = %v", err)
		}
		if got, want := len(listed.Agents), 3; got != want {
			t.Fatalf("len(workspace agents) = %d, want %d: %#v", got, want, listed.Agents)
		}
		if listed.Agents[0].Name != "extension-agent" ||
			listed.Agents[1].Name != "founder" ||
			listed.Agents[2].Name != "qa" {
			t.Fatalf("workspace agent order = %#v, want extension-agent, founder, qa", listed.Agents)
		}

		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/founder?workspace="+workspaceRef, nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf(
				"get workspace agent status = %d, want %d; body = %s",
				getResp.Code,
				http.StatusOK,
				getResp.Body.String(),
			)
		}
		var got contract.AgentResponse
		if err := json.Unmarshal(getResp.Body.Bytes(), &got); err != nil {
			t.Fatalf("json.Unmarshal(get workspace agent) error = %v", err)
		}
		if got.Agent.Name != "founder" || got.Agent.Provider != "codex" {
			t.Fatalf("get workspace agent = %#v, want founder/codex", got.Agent)
		}

		missingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing?workspace="+workspaceRef, nil)
		if missingResp.Code != http.StatusNotFound {
			t.Fatalf(
				"get missing workspace agent status = %d, want %d; body = %s",
				missingResp.Code,
				http.StatusNotFound,
				missingResp.Body.String(),
			)
		}
		if !strings.Contains(missingResp.Body.String(), "not available in workspace") {
			t.Fatalf("get missing workspace agent body = %s, want not available message", missingResp.Body.String())
		}
	})
}

type stubAgentCatalog struct {
	agents  []aghconfig.AgentDef
	get     map[string]aghconfig.AgentDef
	listErr error
	getErr  error
}

var _ core.AgentCatalog = stubAgentCatalog{}

func (s stubAgentCatalog) ListAgents(context.Context) ([]aghconfig.AgentDef, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]aghconfig.AgentDef(nil), s.agents...), nil
}

func (s stubAgentCatalog) GetAgent(_ context.Context, name string) (aghconfig.AgentDef, error) {
	if s.getErr != nil {
		return aghconfig.AgentDef{}, s.getErr
	}
	agent, ok := s.get[name]
	if !ok {
		return aghconfig.AgentDef{}, os.ErrNotExist
	}
	return agent, nil
}

func TestDaemonStatusIncludesNetworkDiagnosticsWithoutCredentials(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{{ID: "sess-1"}}, nil
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
		StatusFn: func(context.Context) (*network.Status, error) {
			return &network.Status{
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
