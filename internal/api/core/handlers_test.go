package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/api/contract"
	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/testutil"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnostics"
	"github.com/compozy/agh/internal/events"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/observe"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/transcript"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestBaseHandlersSessionEndpoints(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	var createCalled atomic.Bool
	var repairSeen session.RepairOpts
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
		AttachSessionFn: func(_ context.Context, req store.SessionAttachRequest) (store.SessionAttach, error) {
			if req.SessionID != "sess-a" {
				t.Fatalf("AttachSession() session id = %q, want sess-a", req.SessionID)
			}
			return store.SessionAttach{
				SessionID:       req.SessionID,
				AttachedTo:      req.AttachedTo,
				AttachedAt:      req.Now,
				AttachExpiresAt: req.Now.Add(req.TTL),
			}, nil
		},
		RepairFn: func(_ context.Context, opts session.RepairOpts) (*session.RepairResult, error) {
			repairSeen = opts
			return &session.RepairResult{
				SessionID: opts.SessionID,
				Actions: []session.RepairAction{{
					Code:      session.RepairActionAppendTerminalError,
					TurnID:    "turn-1",
					Persisted: !opts.DryRun,
				}},
				Persisted: !opts.DryRun,
			}, nil
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

	t.Run("Should list sessions", func(t *testing.T) {
		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list status = %d, want %d", listResp.Code, http.StatusOK)
		}
	})

	t.Run("Should create sessions", func(t *testing.T) {
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

	t.Run("Should get session details", func(t *testing.T) {
		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces/ws-workspace/sessions/sess-a", nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get status = %d, want %d", getResp.Code, http.StatusOK)
		}
	})

	t.Run("Should return not found for missing sessions", func(t *testing.T) {
		notFoundResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/missing",
			nil,
		)
		if notFoundResp.Code != http.StatusNotFound {
			t.Fatalf("get missing status = %d, want %d", notFoundResp.Code, http.StatusNotFound)
		}
	})

	t.Run("Should delete sessions", func(t *testing.T) {
		deleteResp := performRequest(
			t,
			fixture.Engine,
			http.MethodDelete,
			"/workspaces/ws-workspace/sessions/sess-a",
			nil,
		)
		if deleteResp.Code != http.StatusNoContent {
			t.Fatalf("delete status = %d, want %d", deleteResp.Code, http.StatusNoContent)
		}
		if got := deleteResp.Body.String(); got != "" {
			t.Fatalf("delete body = %q, want empty", got)
		}
	})

	t.Run("Should stop sessions", func(t *testing.T) {
		stopResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/workspaces/ws-workspace/sessions/sess-a/stop",
			nil,
		)
		if stopResp.Code != http.StatusNoContent {
			t.Fatalf("stop status = %d, want %d", stopResp.Code, http.StatusNoContent)
		}
		if got := stopResp.Body.String(); got != "" {
			t.Fatalf("stop body = %q, want empty", got)
		}
	})

	t.Run("Should attach sessions", func(t *testing.T) {
		attachResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/workspaces/ws-workspace/sessions/sess-a/attach",
			nil,
		)
		if attachResp.Code != http.StatusOK {
			t.Fatalf("attach status = %d, want %d", attachResp.Code, http.StatusOK)
		}
		var payload contract.SessionAttachResponse
		if err := json.Unmarshal(attachResp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(attach response) error = %v", err)
		}
		if payload.Attach.SessionID != "sess-a" {
			t.Fatalf("attach session id = %q, want %q", payload.Attach.SessionID, "sess-a")
		}
		if payload.Attach.AttachedAt.IsZero() {
			t.Fatal("attach attached_at = zero, want populated timestamp")
		}
		if !payload.Attach.AttachExpiresAt.After(payload.Attach.AttachedAt) {
			t.Fatalf(
				"attach expires_at = %v, attached_at = %v; want expires after attached",
				payload.Attach.AttachExpiresAt,
				payload.Attach.AttachedAt,
			)
		}
		if payload.Session.ID != "sess-a" {
			t.Fatalf("session id = %q, want %q", payload.Session.ID, "sess-a")
		}
		if payload.Session.AttachedTo != payload.Attach.AttachedTo {
			t.Fatalf(
				"session attached_to = %q, attach attached_to = %q; want match",
				payload.Session.AttachedTo,
				payload.Attach.AttachedTo,
			)
		}
		if payload.Session.AttachExpiresAt == nil ||
			!payload.Session.AttachExpiresAt.Equal(payload.Attach.AttachExpiresAt) {
			t.Fatalf(
				"session attach_expires_at = %#v, attach attach_expires_at = %v; want match",
				payload.Session.AttachExpiresAt,
				payload.Attach.AttachExpiresAt,
			)
		}
	})

	t.Run("Should repair sessions", func(t *testing.T) {
		repairResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/workspaces/ws-workspace/sessions/sess-a/repair?dry_run=true&force=true",
			nil,
		)
		if repairResp.Code != http.StatusOK {
			t.Fatalf("repair status = %d, want %d", repairResp.Code, http.StatusOK)
		}
		if repairSeen.SessionID != "sess-a" || !repairSeen.DryRun || !repairSeen.Force {
			t.Fatalf("repair opts = %#v, want sess-a dry-run force", repairSeen)
		}
		var payload contract.SessionRepairResponse
		if err := json.Unmarshal(repairResp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(repair response) error = %v", err)
		}
		if payload.Repair.SessionID != "sess-a" {
			t.Fatalf("repair session id = %q, want sess-a", payload.Repair.SessionID)
		}
		if payload.Repair.Persisted {
			t.Fatalf("repair persisted = %v, want false for dry-run", payload.Repair.Persisted)
		}
		if got, want := len(payload.Repair.Actions), 1; got != want {
			t.Fatalf("repair actions len = %d, want %d", got, want)
		}
		action := payload.Repair.Actions[0]
		if got, want := action.Code, session.RepairActionAppendTerminalError; got != want {
			t.Fatalf("repair action code = %q, want %q", got, want)
		}
		if got, want := action.TurnID, "turn-1"; got != want {
			t.Fatalf("repair action turn id = %q, want %q", got, want)
		}
		if action.Persisted {
			t.Fatalf("repair action persisted = %v, want false for dry-run", action.Persisted)
		}
	})

	t.Run("Should reject conflicting repair query aliases", func(t *testing.T) {
		repairResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/workspaces/ws-workspace/sessions/sess-a/repair?dry_run=true&dry-run=false",
			nil,
		)
		if repairResp.Code != http.StatusBadRequest {
			t.Fatalf("repair conflicting alias status = %d, want %d", repairResp.Code, http.StatusBadRequest)
		}
		if !strings.Contains(repairResp.Body.String(), "conflicting boolean query values for dry_run, dry-run") {
			t.Fatalf("repair conflicting alias body = %q, want conflict message", repairResp.Body.String())
		}
	})

	t.Run("Should return session events", func(t *testing.T) {
		eventsResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/sess-a/events?limit=10&after_sequence=5",
			nil,
		)
		if eventsResp.Code != http.StatusOK {
			t.Fatalf("events status = %d, want %d", eventsResp.Code, http.StatusOK)
		}
	})

	t.Run("Should return session history", func(t *testing.T) {
		historyResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/sess-a/history",
			nil,
		)
		if historyResp.Code != http.StatusOK {
			t.Fatalf("history status = %d, want %d", historyResp.Code, http.StatusOK)
		}
	})

	t.Run("Should return session transcript", func(t *testing.T) {
		transcriptResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/sess-a/transcript",
			nil,
		)
		if transcriptResp.Code != http.StatusOK {
			t.Fatalf("transcript status = %d, want %d", transcriptResp.Code, http.StatusOK)
		}
	})
}

func TestCreateSessionProviderAuthFailureReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("Should return diagnostic for provider auth failure", func(t *testing.T) {
		t.Parallel()

		item := diagnostics.NewItem(
			"provider.codex.auth",
			contract.CodeProviderCLIMissing,
			contract.CategoryProvider,
			"Provider auth status",
			"Provider CLI is not installed or not available on PATH.",
			contract.SeverityError,
			contract.FreshnessLive,
		)
		manager := testutil.StubSessionManager{
			CreateFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
				return nil, acp.WrapFailure(
					store.FailureProviderAuth,
					"provider auth pre-start probe failed",
					diagnostics.NewStructuredError(item, errors.New("missing provider CLI")),
				)
			},
		}
		fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)

		response := performRequest(
			t,
			fixture.Engine,
			http.MethodPost,
			"/sessions",
			[]byte(`{"agent_name":"coder","provider":"codex","workspace":"alpha"}`),
		)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("create status = %d body = %s, want 422", response.Code, response.Body.String())
		}
		var payload contract.ErrorPayload
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(error payload) error = %v", err)
		}
		if payload.Diagnostic == nil {
			t.Fatal("payload.Diagnostic = nil, want provider diagnostic")
		}
		if got, want := payload.Diagnostic.Code, contract.CodeProviderCLIMissing; got != want {
			t.Fatalf("payload.Diagnostic.Code = %q, want %q", got, want)
		}
	})
}

func TestSessionRecapIncludesRedactedObserverMarkers(t *testing.T) {
	t.Parallel()

	t.Run("Should include redacted transcript markers from observer summaries", func(t *testing.T) {
		t.Parallel()

		occurredAt := time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC)
		marker, err := transcript.NewMarker(
			transcript.MarkerPromptInterrupted,
			"Prompt interrupted",
			occurredAt,
			map[string]any{"reason": "operator"},
		)
		if err != nil {
			t.Fatalf("transcript.NewMarker() error = %v", err)
		}
		rawMarker, err := json.Marshal(marker)
		if err != nil {
			t.Fatalf("json.Marshal(marker) error = %v", err)
		}
		openMarker, err := transcript.NewMarker(
			transcript.MarkerSessionUnhealthy,
			"Runtime health check failed.",
			occurredAt.Add(-500*time.Millisecond),
			map[string]any{"stall_reason": store.SessionStallReasonProcessUnhealthy},
		)
		if err != nil {
			t.Fatalf("transcript.NewMarker(open) error = %v", err)
		}
		recoveredMarker, err := transcript.NewMarker(
			transcript.MarkerSessionRecovered,
			"Runtime activity recovered.",
			occurredAt.Add(-250*time.Millisecond),
			map[string]any{"stall_reason": store.SessionStallReasonProcessUnhealthy},
		)
		if err != nil {
			t.Fatalf("transcript.NewMarker(recovered) error = %v", err)
		}
		providerMarker, err := transcript.NewMarker(
			transcript.MarkerProviderFailure,
			"Provider failed.",
			occurredAt.Add(250*time.Millisecond),
			map[string]any{"failure_kind": string(store.FailurePrompt)},
		)
		if err != nil {
			t.Fatalf("transcript.NewMarker(provider) error = %v", err)
		}
		preRecoveryProviderMarker, err := transcript.NewMarker(
			transcript.MarkerProviderFailure,
			"Provider auth failed before runtime recovery.",
			occurredAt.Add(-750*time.Millisecond),
			map[string]any{"failure_kind": string(store.FailureProviderAuth)},
		)
		if err != nil {
			t.Fatalf("transcript.NewMarker(preRecoveryProvider) error = %v", err)
		}
		markerEventContent := func(marker transcript.Marker, turnID string) string {
			t.Helper()
			agentEvent, err := marker.AgentEvent("sess-a", turnID)
			if err != nil {
				t.Fatalf("marker.AgentEvent(%s) error = %v", marker.Kind, err)
			}
			content, err := transcript.MarshalAgentEvent(agentEvent)
			if err != nil {
				t.Fatalf("transcript.MarshalAgentEvent(%s) error = %v", marker.Kind, err)
			}
			return content
		}

		manager := testutil.StubSessionManager{
			StatusFn: func(context.Context, string) (*session.Info, error) {
				return testutil.NewSessionInfo("sess-a"), nil
			},
			EventsFn: func(_ context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
				if id != "sess-a" {
					t.Fatalf("Events() id = %q, want sess-a", id)
				}
				switch query.Type {
				case "":
					if query.Limit != 500 {
						t.Fatalf("Events() query = %#v, want recap query limit 500", query)
					}
					return []store.SessionEvent{{
						ID:        "ev-1",
						SessionID: id,
						Sequence:  1,
						TurnID:    "turn-1",
						Type:      "agent_message",
						AgentName: "coder",
						Content:   `{"text":"hello"}`,
						Timestamp: occurredAt.Add(-time.Second),
					}}, nil
				case events.TranscriptMarkerCreated:
					if query.Limit != 500 {
						t.Fatalf("Events() created marker limit = %d, want %d", query.Limit, 500)
					}
					return []store.SessionEvent{
						{
							ID:        "ev-marker-provider-before-recovery",
							SessionID: id,
							Sequence:  1,
							TurnID:    "turn-provider-before",
							Type:      events.TranscriptMarkerCreated,
							AgentName: "coder",
							Content:   markerEventContent(preRecoveryProviderMarker, "turn-provider-before"),
							Timestamp: occurredAt.Add(-750 * time.Millisecond),
						},
						{
							ID:        "ev-marker-open",
							SessionID: id,
							Sequence:  2,
							TurnID:    "turn-open",
							Type:      events.TranscriptMarkerCreated,
							AgentName: "coder",
							Content:   markerEventContent(openMarker, "turn-open"),
							Timestamp: occurredAt.Add(-500 * time.Millisecond),
						},
						{
							ID:        "ev-marker-recovered",
							SessionID: id,
							Sequence:  3,
							TurnID:    "turn-open",
							Type:      events.TranscriptMarkerCreated,
							AgentName: "coder",
							Content:   markerEventContent(recoveredMarker, "turn-open"),
							Timestamp: occurredAt.Add(-250 * time.Millisecond),
						},
						{
							ID:        "ev-marker-provider",
							SessionID: id,
							Sequence:  4,
							TurnID:    "turn-provider",
							Type:      events.TranscriptMarkerCreated,
							AgentName: "coder",
							Content:   markerEventContent(providerMarker, "turn-provider"),
							Timestamp: occurredAt.Add(250 * time.Millisecond),
						},
					}, nil
				case events.TranscriptMarkerRedacted:
					if query.Limit != 500 {
						t.Fatalf("Events() redacted marker limit = %d, want %d", query.Limit, 500)
					}
					return nil, nil
				default:
					t.Fatalf("unexpected Events() query = %#v", query)
					return nil, nil
				}
			},
			InputQueueFn: func(_ context.Context, id string) (session.InputQueueSummary, error) {
				if id != "sess-a" {
					t.Fatalf("InputQueueSummary() id = %q, want sess-a", id)
				}
				return session.InputQueueSummary{QueueGeneration: 7, PendingInputs: 2}, nil
			},
		}
		observer := testutil.StubObserver{
			QueryEventsFn: func(_ context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
				switch query.Type {
				case events.TranscriptMarkerCreated:
					return nil, nil
				case events.TranscriptMarkerRedacted:
					return []store.EventSummary{{
						ID:        "sum-redacted",
						SessionID: "sess-a",
						Type:      events.TranscriptMarkerRedacted,
						Content:   rawMarker,
						Timestamp: occurredAt,
					}}, nil
				default:
					t.Fatalf("unexpected marker summary query = %#v", query)
					return nil, nil
				}
			},
		}

		fixture := newHandlerFixture(t, manager, observer, testutil.StubWorkspaceService{}, nil, nil)
		response := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/sess-a/recap?limit=5",
			nil,
		)
		if response.Code != http.StatusOK {
			t.Fatalf("recap status = %d body=%s, want %d", response.Code, response.Body.String(), http.StatusOK)
		}

		var payload contract.SessionRecapResponse
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(recap response) error = %v", err)
		}
		if got, want := len(payload.Recap.RecentMarkers), 1; got != want {
			t.Fatalf("len(recent_markers) = %d, want %d", got, want)
		}
		if got, want := payload.Recap.RecentMarkers[0].Kind, transcript.MarkerPromptInterrupted; got != want {
			t.Fatalf("recent_markers[0].Kind = %q, want %q", got, want)
		}
		if got, want := payload.Recap.PendingInputs, 2; got != want {
			t.Fatalf("pending_inputs = %d, want %d", got, want)
		}
		if got, want := payload.Recap.PendingMarkers, 2; got != want {
			t.Fatalf("pending_markers = %d, want %d", got, want)
		}
		if got, want := payload.Recap.Snapshot.QueueGeneration, int64(7); got != want {
			t.Fatalf("snapshot.queue_generation = %d, want %d", got, want)
		}
		if got, want := payload.Recap.Snapshot.Consistency, "persisted_reads"; got != want {
			t.Fatalf("snapshot.consistency = %q, want %q", got, want)
		}
	})
}

func TestLogsEndpointsRequireObserver(t *testing.T) {
	t.Parallel()

	t.Run("Should return service unavailable when observer is missing", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Observer = nil

		for _, path := range []string{
			"/logs?workspace_id=ws-workspace",
			"/logs/stream?workspace_id=ws-workspace",
		} {
			response := performRequest(t, fixture.Engine, http.MethodGet, path, nil)
			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf(
					"%s status = %d, want %d; body=%s",
					path,
					response.Code,
					http.StatusServiceUnavailable,
					response.Body.String(),
				)
			}
			if !strings.Contains(response.Body.String(), "observer is required") {
				t.Fatalf("%s body = %q, want observer error", path, response.Body.String())
			}
		}
	})
}

func TestBaseHandlersStreamingAndObserveEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("Should stream session events and expose observe endpoints", func(t *testing.T) {
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

		streamResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/workspaces/ws-workspace/sessions/sess-a/stream",
			nil,
		)
		if streamResp.Code != http.StatusOK {
			t.Fatalf("stream status = %d, want %d", streamResp.Code, http.StatusOK)
		}
		if records := testutil.ParseSSE(t, streamResp.Body.String()); len(records) < 2 {
			t.Fatalf("stream records = %d, want at least 2", len(records))
		}

		observeResp := performRequest(t, fixture.Engine, http.MethodGet, "/logs?workspace_id=ws-workspace", nil)
		if observeResp.Code != http.StatusOK {
			t.Fatalf("observe status = %d, want %d", observeResp.Code, http.StatusOK)
		}

		statusResp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if statusResp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", statusResp.Code, http.StatusOK)
		}

		doctorResp := performRequest(t, fixture.Engine, http.MethodGet, "/doctor", nil)
		if doctorResp.Code != http.StatusOK {
			t.Fatalf("doctor = %d, want %d", doctorResp.Code, http.StatusOK)
		}
	})
}

func TestBaseHandlersAgentEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("Should serve agent read endpoints and surface loader failures", func(t *testing.T) {
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
		if _, _, err := aghconfig.EnsureOnboardingAgent(fixture.HomePaths); err != nil {
			t.Fatalf("EnsureOnboardingAgent() error = %v", err)
		}

		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/coder", nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get agent status = %d, want %d", getResp.Code, http.StatusOK)
		}

		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list agents status = %d, want %d", listResp.Code, http.StatusOK)
		}
		var listed contract.AgentsResponse
		if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
			t.Fatalf("json.Unmarshal(list agents) error = %v", err)
		}
		if len(listed.Agents) != 1 || listed.Agents[0].Name != "coder" {
			t.Fatalf("listed agents = %#v, want only coder", listed.Agents)
		}
		onboardingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/onboarding", nil)
		if onboardingResp.Code != http.StatusNotFound {
			t.Fatalf("get onboarding status = %d, want %d", onboardingResp.Code, http.StatusNotFound)
		}
		var onboardingPayload contract.ErrorPayload
		if err := json.Unmarshal(onboardingResp.Body.Bytes(), &onboardingPayload); err != nil {
			t.Fatalf("json.Unmarshal(onboarding error) error = %v; body=%s", err, onboardingResp.Body.String())
		}
		if !strings.Contains(onboardingPayload.Error, "not available") {
			t.Fatalf("onboarding error = %q, want not-available message", onboardingPayload.Error)
		}

		fixture.Handlers.AgentLoader = func(string, aghconfig.HomePaths) (aghconfig.AgentDef, error) {
			return aghconfig.AgentDef{}, errors.New("boom")
		}
		missingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
		if missingResp.Code != http.StatusInternalServerError {
			t.Fatalf("missing agent status = %d, want %d", missingResp.Code, http.StatusInternalServerError)
		}
	})
}

func TestBaseHandlersCreateAgentEndpoint(t *testing.T) {
	t.Parallel()

	t.Run("Should create a global AGENT.md definition", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		body := mustJSON(t, contract.CreateAgentRequest{
			Scope: contract.AgentCreateScopeGlobal,
			Agent: contract.CreateAgentPayload{
				Name:         "pricing_strategist",
				Provider:     "claude",
				Model:        "claude-sonnet-4-6",
				Tools:        []string{"builtin__shell"},
				Permissions:  contract.SettingsPermissionModeApproveReads,
				CategoryPath: []string{"Strategy"},
				Skills:       &contract.CreateAgentSkillsConfig{Disabled: []string{"legacy-skill"}},
				Prompt:       "You own pricing strategy.",
			},
		})

		resp := performRequest(t, fixture.Engine, http.MethodPost, "/agents", body)
		if resp.Code != http.StatusCreated {
			t.Fatalf(
				"create global agent status = %d, want %d; body=%s",
				resp.Code,
				http.StatusCreated,
				resp.Body.String(),
			)
		}
		var payload contract.AgentResponse
		decodeJSON(t, resp.Body.Bytes(), &payload)
		if payload.Agent.Name != "pricing_strategist" || payload.Agent.Provider != "claude" {
			t.Fatalf("created agent payload = %#v, want pricing strategist", payload.Agent)
		}
		path := filepath.Join(
			fixture.HomePaths.AgentsDir,
			"pricing_strategist",
			aghconfig.AgentDefinitionFileName,
		)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("os.Stat(created AGENT.md) error = %v", err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("created AGENT.md mode = %v, want 0600", info.Mode().Perm())
		}
		loaded, err := aghconfig.LoadAgentDefFile(path)
		if err != nil {
			t.Fatalf("LoadAgentDefFile(created AGENT.md) error = %v", err)
		}
		if loaded.Permissions != string(contract.SettingsPermissionModeApproveReads) ||
			len(loaded.Skills.Disabled) != 1 || loaded.Skills.Disabled[0] != "legacy-skill" {
			t.Fatalf("loaded agent = %#v, want permissions and disabled skill", loaded)
		}
	})

	t.Run("Should create a workspace AGENT.md definition", func(t *testing.T) {
		t.Parallel()

		workspaceRoot := t.TempDir()
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{
				ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
					if ref != "alpha" {
						t.Fatalf("Resolve() ref = %q, want alpha", ref)
					}
					return workspacepkg.ResolvedWorkspace{
						Workspace: workspacepkg.Workspace{
							ID:      "ws-alpha",
							Name:    "alpha",
							RootDir: workspaceRoot,
						},
						WorkspaceID: "ws-alpha",
					}, nil
				},
			},
			nil,
			nil,
		)
		body := mustJSON(t, contract.CreateAgentRequest{
			Scope:     contract.AgentCreateScopeWorkspace,
			Workspace: "alpha",
			Agent: contract.CreateAgentPayload{
				Name:     "qa_operator",
				Provider: "codex",
				Prompt:   "Stress test the workspace.",
			},
		})

		resp := performRequest(t, fixture.Engine, http.MethodPost, "/agents", body)
		if resp.Code != http.StatusCreated {
			t.Fatalf(
				"create workspace agent status = %d, want %d; body=%s",
				resp.Code,
				http.StatusCreated,
				resp.Body.String(),
			)
		}
		path := filepath.Join(
			workspaceRoot,
			aghconfig.DirName,
			aghconfig.AgentsDirName,
			"qa_operator",
			aghconfig.AgentDefinitionFileName,
		)
		loaded, err := aghconfig.LoadAgentDefFile(path)
		if err != nil {
			t.Fatalf("LoadAgentDefFile(workspace AGENT.md) error = %v", err)
		}
		if loaded.Name != "qa_operator" || loaded.Provider != "codex" {
			t.Fatalf("loaded workspace agent = %#v, want qa_operator", loaded)
		}
	})

	t.Run("Should reject duplicate AGENT.md definitions", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		body := mustJSON(t, contract.CreateAgentRequest{
			Scope: contract.AgentCreateScopeGlobal,
			Agent: contract.CreateAgentPayload{
				Name:     "duplicate",
				Provider: "codex",
				Prompt:   "First definition.",
			},
		})
		first := performRequest(t, fixture.Engine, http.MethodPost, "/agents", body)
		if first.Code != http.StatusCreated {
			t.Fatalf(
				"initial create status = %d, want %d; body=%s",
				first.Code,
				http.StatusCreated,
				first.Body.String(),
			)
		}
		second := performRequest(t, fixture.Engine, http.MethodPost, "/agents", body)
		if second.Code != http.StatusConflict {
			t.Fatalf(
				"duplicate create status = %d, want %d; body=%s",
				second.Code,
				http.StatusConflict,
				second.Body.String(),
			)
		}
	})

	t.Run("Should reject invalid simple AGENT.md fields", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			req  contract.CreateAgentRequest
		}{
			{
				name: "missing provider",
				req: contract.CreateAgentRequest{
					Scope: contract.AgentCreateScopeGlobal,
					Agent: contract.CreateAgentPayload{Name: "missing_provider", Prompt: "Prompt."},
				},
			},
			{
				name: "reserved internal name",
				req: contract.CreateAgentRequest{
					Scope: contract.AgentCreateScopeGlobal,
					Agent: contract.CreateAgentPayload{
						Name:     aghconfig.OnboardingAgentName,
						Provider: "codex",
						Prompt:   "Reserved.",
					},
				},
			},
			{
				name: "invalid permission",
				req: contract.CreateAgentRequest{
					Scope: contract.AgentCreateScopeGlobal,
					Agent: contract.CreateAgentPayload{
						Name:        "bad_permission",
						Provider:    "codex",
						Permissions: "maybe",
						Prompt:      "Prompt.",
					},
				},
			},
			{
				name: "invalid tool",
				req: contract.CreateAgentRequest{
					Scope: contract.AgentCreateScopeGlobal,
					Agent: contract.CreateAgentPayload{
						Name:     "bad_tool",
						Provider: "codex",
						Tools:    []string{"shell"},
						Prompt:   "Prompt.",
					},
				},
			},
			{
				name: "invalid category",
				req: contract.CreateAgentRequest{
					Scope: contract.AgentCreateScopeGlobal,
					Agent: contract.CreateAgentPayload{
						Name:         "bad_category",
						Provider:     "codex",
						CategoryPath: []string{"Engineering/Platform"},
						Prompt:       "Prompt.",
					},
				},
			},
		}
		for _, tc := range tests {
			t.Run("Should reject "+tc.name, func(t *testing.T) {
				t.Parallel()
				fixture := newHandlerFixture(
					t,
					testutil.StubSessionManager{},
					testutil.StubObserver{},
					testutil.StubWorkspaceService{},
					nil,
					nil,
				)
				resp := performRequest(t, fixture.Engine, http.MethodPost, "/agents", mustJSON(t, tc.req))
				if resp.Code != http.StatusBadRequest {
					t.Fatalf(
						"invalid create status = %d, want %d; body=%s",
						resp.Code,
						http.StatusBadRequest,
						resp.Body.String(),
					)
				}
			})
		}
	})

	t.Run("Should map unresolved workspaces through workspace errors", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{
				ResolveFn: func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
					return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
				},
			},
			nil,
			nil,
		)
		body := mustJSON(t, contract.CreateAgentRequest{
			Scope:     contract.AgentCreateScopeWorkspace,
			Workspace: "missing",
			Agent: contract.CreateAgentPayload{
				Name:     "operator",
				Provider: "codex",
				Prompt:   "Operate.",
			},
		})
		resp := performRequest(t, fixture.Engine, http.MethodPost, "/agents", body)
		if resp.Code != http.StatusNotFound {
			t.Fatalf(
				"missing workspace status = %d, want %d; body=%s",
				resp.Code,
				http.StatusNotFound,
				resp.Body.String(),
			)
		}
	})
}

func TestBaseHandlersAgentCatalogEndpoints(t *testing.T) {
	t.Parallel()

	t.Run("Should expose agent catalog lists, lookups, and error mappings", func(t *testing.T) {
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
				{Name: aghconfig.OnboardingAgentName, Prompt: "Onboarding prompt"},
			},
			get: map[string]aghconfig.AgentDef{
				"alpha":                       {Name: "alpha", Prompt: "Alpha prompt"},
				aghconfig.OnboardingAgentName: {Name: aghconfig.OnboardingAgentName, Prompt: "Onboarding prompt"},
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
		onboardingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/onboarding", nil)
		if onboardingResp.Code != http.StatusNotFound {
			t.Fatalf("get onboarding catalog status = %d, want %d", onboardingResp.Code, http.StatusNotFound)
		}
		var onboardingPayload contract.ErrorPayload
		if err := json.Unmarshal(onboardingResp.Body.Bytes(), &onboardingPayload); err != nil {
			t.Fatalf("json.Unmarshal(onboarding catalog error) error = %v; body=%s", err, onboardingResp.Body.String())
		}
		if !strings.Contains(onboardingPayload.Error, "not available") {
			t.Fatalf("onboarding catalog error = %q, want not-available message", onboardingPayload.Error)
		}

		fixture.Handlers.AgentCatalog = stubAgentCatalog{getErr: os.ErrNotExist}
		missingResp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
		if missingResp.Code != http.StatusNotFound {
			t.Fatalf("get missing catalog agent status = %d, want %d", missingResp.Code, http.StatusNotFound)
		}
		var missingPayload contract.ErrorPayload
		if err := json.Unmarshal(missingResp.Body.Bytes(), &missingPayload); err != nil {
			t.Fatalf("json.Unmarshal(missing catalog error) error = %v; body=%s", err, missingResp.Body.String())
		}
		if !strings.Contains(missingPayload.Error, "file does not exist") {
			t.Fatalf("missing catalog error = %q, want file-missing message", missingPayload.Error)
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
	})
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
							{Name: aghconfig.OnboardingAgentName, Provider: "codex", Prompt: "Internal onboarding."},
							{Name: "qa", Provider: "codex", Prompt: "Stress test the release."},
						},
						AgentDiagnostics: []workspacepkg.AgentDiagnostic{{
							Name:      "broken",
							Path:      "/workspace/.agh/agents/broken/AGENT.md",
							ErrorKind: "frontmatter.missing",
							Message:   "config: missing YAML frontmatter",
						}, {
							Name:      aghconfig.OnboardingAgentName,
							Path:      "/workspace/.agh/agents/onboarding/AGENT.md",
							ErrorKind: "frontmatter.missing",
							Message:   "config: missing YAML frontmatter",
						}},
					}, nil
				},
			},
			nil,
			nil,
		)
		fixture.Handlers.AgentCatalog = stubAgentCatalog{
			agents: []aghconfig.AgentDef{
				{Name: "extension-agent", Provider: "codex", Prompt: "Projected by extension."},
				{Name: aghconfig.OnboardingAgentName, Provider: "codex", Prompt: "Projected onboarding."},
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
		if got, want := len(listed.Agents), 4; got != want {
			t.Fatalf("len(workspace agents) = %d, want %d: %#v", got, want, listed.Agents)
		}
		if listed.Agents[0].Name != "broken" ||
			listed.Agents[1].Name != "extension-agent" ||
			listed.Agents[2].Name != "founder" ||
			listed.Agents[3].Name != "qa" {
			t.Fatalf("workspace agent order = %#v, want broken, extension-agent, founder, qa", listed.Agents)
		}
		if len(listed.Agents[0].Diagnostics) != 1 ||
			listed.Agents[0].Diagnostics[0].ErrorKind != "frontmatter.missing" {
			t.Fatalf("workspace malformed agent diagnostics = %#v, want frontmatter.missing", listed.Agents[0])
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
		onboardingResp := performRequest(
			t,
			fixture.Engine,
			http.MethodGet,
			"/agents/onboarding?workspace="+workspaceRef,
			nil,
		)
		if onboardingResp.Code != http.StatusNotFound {
			t.Fatalf(
				"get workspace onboarding status = %d, want %d; body = %s",
				onboardingResp.Code,
				http.StatusNotFound,
				onboardingResp.Body.String(),
			)
		}
		var onboardingPayload contract.ErrorPayload
		if err := json.Unmarshal(onboardingResp.Body.Bytes(), &onboardingPayload); err != nil {
			t.Fatalf(
				"json.Unmarshal(workspace onboarding error) error = %v; body=%s",
				err,
				onboardingResp.Body.String(),
			)
		}
		if !strings.Contains(onboardingPayload.Error, "not available") {
			t.Fatalf("workspace onboarding error = %q, want not-available message", onboardingPayload.Error)
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

	t.Run("Should include network diagnostics in daemon status without leaking credentials", func(t *testing.T) {
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

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
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
		bodyLower := strings.ToLower(resp.Body.String())
		for _, forbidden := range []string{
			"token=",
			"claim_token",
			"agh_claim_",
			"authorization: bearer",
			"pkce_verifier",
			"oauth_code",
			"access_token",
			"refresh_token",
			"secret_binding",
		} {
			if strings.Contains(bodyLower, forbidden) {
				t.Fatalf("daemon status leaked credentials (%s): %s", forbidden, resp.Body.String())
			}
		}
	})

	t.Run("Should report enabled network as unavailable when status cannot be collected", func(t *testing.T) {
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
		fixture.Handlers.Network = nil

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/doctor", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("doctor status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
		var payload contract.DoctorPayload
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(doctor) error = %v", err)
		}
		for _, item := range payload.Items {
			if item.ID != "doctor.network.status" {
				continue
			}
			if item.Code != contract.CodeNetworkUnavailable || item.Severity != contract.SeverityWarn {
				t.Fatalf("network diagnostic = %#v, want unavailable warning", item)
			}
			return
		}
		t.Fatalf("doctor items = %#v, want network diagnostic", payload.Items)
	})

	t.Run("Should keep doctor status ok for informational diagnostics", func(t *testing.T) {
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
		fixture.Handlers.Config.Network.Enabled = false

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/doctor", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("doctor status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
		var payload contract.DoctorPayload
		if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
			t.Fatalf("json.Unmarshal(doctor) error = %v", err)
		}
		for _, item := range payload.Items {
			if item.ID == "doctor.network.status" &&
				item.Code == contract.CodeNetworkDisabled &&
				item.Severity == contract.SeverityInfo {
				return
			}
		}
		t.Fatalf("doctor items = %#v, want disabled network info diagnostic", payload.Items)
	})
}

func TestLogsEndpointsRejectConflictingAliases(t *testing.T) {
	t.Parallel()

	t.Run("Should reject conflicting logs query aliases", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		for _, path := range []string{
			"/logs?after_seq=1&after_sequence=2",
			"/logs?workspace_id=ws-one&workspace=ws-two",
			"/logs?run=run-one&run_id=run-two",
		} {
			resp := performRequest(t, fixture.Engine, http.MethodGet, path, nil)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("%s status = %d, want %d; body=%s", path, resp.Code, http.StatusBadRequest, resp.Body.String())
			}
			if !strings.Contains(resp.Body.String(), "conflicting query values") {
				t.Fatalf("%s body = %q, want conflicting query values message", path, resp.Body.String())
			}
		}
	})
}
