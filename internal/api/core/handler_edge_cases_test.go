package core_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type bufferFlusher struct {
	bytes.Buffer
}

func (bufferFlusher) Flush() {}

type failingFlusher struct {
	writes int
}

func (f *failingFlusher) Write(p []byte) (int, error) {
	f.writes++
	if f.writes > 1 {
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func (*failingFlusher) Flush() {}

type failNthWriteFlusher struct {
	writes int
	failAt int
	err    error
}

func (f *failNthWriteFlusher) Write(p []byte) (int, error) {
	f.writes++
	if f.writes == f.failAt {
		if f.err != nil {
			return 0, f.err
		}
		return 0, io.ErrClosedPipe
	}
	return len(p), nil
}

func (*failNthWriteFlusher) Flush() {}

func TestObserveAndSSEHelpers(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	event := store.EventSummary{
		ID:        "ev-1",
		SessionID: "sess-1",
		Sequence:  7,
		Type:      "agent_message",
		AgentName: "coder",
		Timestamp: timestamp,
	}

	if !core.ObserveEventAfterCursor(event, core.ObserveCursor{}) {
		t.Fatal("ObserveEventAfterCursor(empty cursor) = false, want true")
	}
	if core.ObserveEventAfterCursor(event, core.ObserveCursor{Timestamp: timestamp.Add(time.Second), ID: "older"}) {
		t.Fatal("ObserveEventAfterCursor(newer cursor) = true, want false")
	}
	if core.ObserveEventAfterCursor(event, core.ObserveCursor{Timestamp: timestamp, Sequence: 9}) {
		t.Fatal("ObserveEventAfterCursor(same timestamp higher sequence) = true, want false")
	}
	if got, want := core.ObserveEventID(event), "2026-04-03T12:00:00Z|00000000000000000007"; got != want {
		t.Fatalf("ObserveEventID() = %q, want %q", got, want)
	}

	writer := &bufferFlusher{}
	next := core.EmitObserveEvents(writer, []store.EventSummary{event}, core.ObserveCursor{})
	if next.Sequence != event.Sequence || next.Timestamp.IsZero() {
		t.Fatalf("EmitObserveEvents() cursor = %#v", next)
	}
	if writer.Len() == 0 {
		t.Fatal("expected SSE output to be written")
	}

	failingWriter := &failingFlusher{}
	prior := core.ObserveCursor{Timestamp: timestamp.Add(-time.Second), Sequence: 3, ID: "legacy"}
	if got := core.EmitObserveEvents(failingWriter, []store.EventSummary{event}, prior); got != prior {
		t.Fatalf("EmitObserveEvents(failing writer) cursor = %#v, want %#v", got, prior)
	}

	if err := core.WriteSSE(
		writer,
		core.SSEMessage{ID: "2", Name: "done", Data: map[string]string{"ok": "true"}},
	); err != nil {
		t.Fatalf("WriteSSE() error = %v", err)
	}
	if err := core.WriteSSERaw(writer, "3", `"raw"`, "raw"); err != nil {
		t.Fatalf("WriteSSERaw() error = %v", err)
	}
	if err := core.WriteSSE(nil, core.SSEMessage{}); err == nil {
		t.Fatal("WriteSSE(nil) error = nil, want non-nil")
	}
	if err := core.WriteSSERaw(nil, "", "null"); err == nil {
		t.Fatal("WriteSSERaw(nil) error = nil, want non-nil")
	}
}

func TestWriteSSERawWrapsStepFailuresWithContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		failAt int
		want   string
	}{
		{name: "ShouldWrapIDPrefixFailure", failAt: 1, want: "write sse id prefix"},
		{name: "ShouldWrapEventPrefixFailure", failAt: 4, want: "write sse event prefix"},
		{name: "ShouldWrapDataPrefixFailure", failAt: 7, want: "write sse data prefix"},
		{name: "ShouldWrapPayloadFailure", failAt: 8, want: "write sse data payload"},
		{name: "ShouldWrapTerminatorFailure", failAt: 9, want: "write sse message terminator"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writer := &failNthWriteFlusher{failAt: tt.failAt, err: io.ErrClosedPipe}
			err := core.WriteSSERaw(writer, "msg-1", `"raw"`, "done")
			if !errors.Is(err, io.ErrClosedPipe) {
				t.Fatalf("WriteSSERaw() error = %v, want %v", err, io.ErrClosedPipe)
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("WriteSSERaw() error = %v, want context %q", err, tt.want)
			}
		})
	}
}

func TestConversionAndStatusHelpers(t *testing.T) {
	t.Parallel()

	usageValue := int64(10)
	agentEvent := core.AgentEventPayloadFromEvent(acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "sess-1",
		TurnID:    "turn-1",
		Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Action:    "fs/read_text_file",
		Failure: &store.SessionFailure{
			Kind:    store.FailurePermission,
			Summary: "permission policy denied",
		},
		Usage: &acp.TokenUsage{
			InputTokens: &usageValue,
			Timestamp:   time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		},
		Raw: []byte(`{"ok":true}`),
	})
	if agentEvent.Type != acp.EventTypePermission || agentEvent.Usage == nil || agentEvent.Usage.InputTokens == nil {
		t.Fatalf("agent event payload = %#v", agentEvent)
	}
	if agentEvent.Failure == nil || agentEvent.Failure.Kind != store.FailurePermission {
		t.Fatalf("agent event failure = %#v", agentEvent.Failure)
	}
	if got := string(agentEvent.Raw); got != `{"ok":true}` {
		t.Fatalf("agent event raw payload = %s, want valid JSON passthrough", got)
	}
	plainRawEvent := core.AgentEventPayloadFromEvent(acp.AgentEvent{Raw: []byte("plain-text")})
	if got := string(plainRawEvent.Raw); got != `"plain-text"` {
		t.Fatalf("plain raw payload = %s, want quoted string", got)
	}
	if payload := core.PayloadJSON("plain-text"); string(payload) == "plain-text" {
		t.Fatalf("PayloadJSON() = %s, want quoted JSON", string(payload))
	}
	if status := core.StatusForWorkspaceError(workspacepkg.ErrWorkspacePathTaken); status != http.StatusConflict {
		t.Fatalf("StatusForWorkspaceError() = %d, want %d", status, http.StatusConflict)
	}
	if status := core.StatusForMemoryError(errors.New("boom")); status != http.StatusInternalServerError {
		t.Fatalf("StatusForMemoryError(default) = %d, want %d", status, http.StatusInternalServerError)
	}
	if status := core.StatusForMemoryError(nil); status != http.StatusOK {
		t.Fatalf("StatusForMemoryError(nil) = %d, want %d", status, http.StatusOK)
	}
	if got := core.NewMemoryValidationError(nil); got != nil {
		t.Fatalf("NewMemoryValidationError(nil) = %v, want nil", got)
	}

	sessions := core.SessionPayloadsForWorkspace([]*session.Info{
		{ID: "sess-1", WorkspaceID: "ws_alpha"},
		{ID: "sess-2", WorkspaceID: "ws_beta"},
	}, "ws_alpha")
	if len(sessions) != 1 || sessions[0].ID != "sess-1" {
		t.Fatalf("SessionPayloadsForWorkspace() = %#v", sessions)
	}
}

func TestBaseHandlersWorkspaceFilteringAndDefaults(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{
				{ID: "sess-1", WorkspaceID: "ws_alpha"},
				{ID: "sess-2", WorkspaceID: "ws_beta"},
			}, nil
		},
	}
	workspaces := testutil.StubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("Get workspace ref = %q, want alpha", ref)
			}
			return workspacepkg.Workspace{ID: "ws_alpha", RootDir: "/workspace"}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, workspaces, nil, nil)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions?workspace=alpha", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("filtered list status = %d, want %d", resp.Code, http.StatusOK)
	}

	fixture.Handlers.SetHTTPPort(4321)
	recorder := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("daemon status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var payload struct {
		Daemon contract.DaemonStatusPayload `json:"daemon"`
	}
	testutil.DecodeJSONResponse(t, recorder, &payload)
	if payload.Daemon.HTTPPort != 4321 {
		t.Fatalf("daemon http port = %d, want 4321", payload.Daemon.HTTPPort)
	}
	resolvedUserHomeDir, err := aghconfig.ResolvePath(os.Getenv("HOME"))
	if err != nil {
		t.Fatalf("ResolvePath(HOME) error = %v", err)
	}
	if payload.Daemon.UserHomeDir != resolvedUserHomeDir {
		t.Fatalf("daemon user home dir = %q, want %q", payload.Daemon.UserHomeDir, resolvedUserHomeDir)
	}

	handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{})
	if handlers.TransportName != "" {
		t.Fatalf("TransportName default = %q, want empty", handlers.TransportName)
	}
}

func TestMemoryWrapperExports(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	if _, err := workspacepkg.EnsureIdentity(context.Background(), workspace); err != nil {
		t.Fatalf("EnsureIdentity(%q) error = %v", workspace, err)
	}
	req := contract.MemoryWriteRequest{
		Scope:     "workspace",
		Workspace: workspace,
		Content:   "---\nname: Project\ndescription: desc\ntype: project\n---\n\nbody",
	}
	scope, resolvedWorkspace, err := core.ResolveMemoryWriteScope(req)
	if err != nil {
		t.Fatalf("ResolveMemoryWriteScope() error = %v", err)
	}
	if scope != memcontract.ScopeWorkspace || resolvedWorkspace == "" {
		t.Fatalf("scope=%q workspace=%q", scope, resolvedWorkspace)
	}
	if _, err := core.ParseOptionalMemoryScope("bogus"); err == nil {
		t.Fatal("ParseOptionalMemoryScope(bogus) error = nil, want non-nil")
	}
	if _, err := core.ResolveMemoryWorkspace(""); err == nil {
		t.Fatal("ResolveMemoryWorkspace(\"\") error = nil, want non-nil")
	}
	if scope, resolved, err := core.ResolveMemoryWriteScope(contract.MemoryWriteRequest{
		Content: "---\nname: Global\ndescription: desc\ntype: user\n---\n\nbody",
	}); err != nil || scope != memcontract.ScopeGlobal || resolved != "" {
		t.Fatalf("ResolveMemoryWriteScope(user default) = %q %q %v", scope, resolved, err)
	}

	store := memory.NewStore(t.TempDir())
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	if err := store.ForWorkspace(workspace).
		Write(memcontract.ScopeWorkspace, "note.md", []byte("---\nname: note\ndescription: desc\ntype: project\n---\n\nbody")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			info := testutil.NewSessionInfo("sess-a")
			info.Workspace = workspace
			return []*session.Info{info}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, store, nil)
	if _, err := fixture.Handlers.ResolveMemoryLocation("note.md", "workspace", workspace); err != nil {
		t.Fatalf("ResolveMemoryLocation() error = %v", err)
	}
	workspacesOut, err := fixture.Handlers.MemoryHealthWorkspaces(context.Background(), "")
	if err != nil || len(workspacesOut) != 1 {
		t.Fatalf("MemoryHealthWorkspaces() = %#v, %v", workspacesOut, err)
	}
}

func TestHealthHandlerReturnsRetentionAndPersistencePayload(t *testing.T) {
	t.Parallel()

	lastSweepAt := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	lastCutoffAt := lastSweepAt.AddDate(0, 0, -14)
	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{
			HealthFn: func(context.Context) (observe.Health, error) {
				return observe.Health{
					Status:             "degraded",
					ActiveSessions:     2,
					GlobalDBSizeBytes:  4096,
					SessionDBSizeBytes: 2048,
					Persistence: observe.PersistenceHealth{
						Status:             "degraded",
						GlobalDBSizeBytes:  4096,
						SessionDBSizeBytes: 2048,
					},
					Retention: observe.RetentionHealth{
						Enabled:                  true,
						RetentionDays:            14,
						SweepIntervalSeconds:     int64((24 * time.Hour).Seconds()),
						LastSweepStatus:          "error",
						LastSweepAt:              &lastSweepAt,
						LastCutoffAt:             &lastCutoffAt,
						LastSweepError:           "disk full",
						DeletedEventSummaries:    3,
						DeletedTokenStats:        2,
						DeletedPermissionLogRows: 1,
					},
					Failures: observe.FailureHealth{
						Status: "degraded",
						Total:  1,
						ByKind: map[store.FailureKind]int{store.FailureProcess: 1},
						Recent: []observe.SessionFailureHealth{{
							SessionID:       "sess-crash",
							AgentName:       "coder",
							Provider:        "claude",
							WorkspaceID:     "ws-1",
							State:           "stopped",
							FailureKind:     store.FailureProcess,
							Summary:         "provider crashed",
							CrashBundlePath: "/tmp/crash.json",
							UpdatedAt:       lastSweepAt,
						}},
					},
					AgentProbes: []acp.ProbeResult{{
						AgentName:  "coder",
						Provider:   "claude",
						Command:    "missing-agent",
						Status:     acp.ProbeStatusMissing,
						Error:      "not found",
						CheckedAt:  lastSweepAt,
						DurationMS: 7,
					}},
					Version: "dev",
				}, nil
			},
		},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload contract.StatusPayload
	decodeJSON(t, resp.Body.Bytes(), &payload)
	if payload.Health.Persistence.Status != "degraded" ||
		payload.Health.Persistence.GlobalDBSizeBytes != 4096 ||
		payload.Health.Persistence.SessionDBSizeBytes != 2048 {
		t.Fatalf("health.persistence = %#v, want degraded persistence payload", payload.Health.Persistence)
	}
	if !payload.Health.Retention.Enabled ||
		payload.Health.Retention.RetentionDays != 14 ||
		payload.Health.Retention.LastSweepStatus != "error" ||
		payload.Health.Retention.LastSweepError != "disk full" ||
		payload.Health.Retention.DeletedEventSummaries != 3 ||
		payload.Health.Retention.DeletedTokenStats != 2 ||
		payload.Health.Retention.DeletedPermissionLogRows != 1 {
		t.Fatalf("health.retention = %#v, want typed retention payload", payload.Health.Retention)
	}
	if payload.Health.Retention.LastSweepAt == nil || !payload.Health.Retention.LastSweepAt.Equal(lastSweepAt) {
		t.Fatalf("health.retention.last_sweep_at = %#v, want %s", payload.Health.Retention.LastSweepAt, lastSweepAt)
	}
	if payload.Health.Retention.LastCutoffAt == nil || !payload.Health.Retention.LastCutoffAt.Equal(lastCutoffAt) {
		t.Fatalf("health.retention.last_cutoff_at = %#v, want %s", payload.Health.Retention.LastCutoffAt, lastCutoffAt)
	}
	if payload.Health.Failures.Status != "degraded" ||
		payload.Health.Failures.Total != 1 ||
		payload.Health.Failures.ByKind[store.FailureProcess] != 1 ||
		len(payload.Health.Failures.Recent) != 1 {
		t.Fatalf("health.failures = %#v, want lifecycle failure payload", payload.Health.Failures)
	}
	if payload.Health.AgentProbes == nil ||
		len(payload.Health.AgentProbes) != 1 ||
		payload.Health.AgentProbes[0].Status != acp.ProbeStatusMissing {
		t.Fatalf("health.agent_probes = %#v, want missing probe payload", payload.Health.AgentProbes)
	}
}

func TestBaseHandlersHealthAndDaemonStatusErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("Should health observer failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{}, errors.New("boom")
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf(
				"health status = %d, want %d; body=%s",
				resp.Code,
				http.StatusInternalServerError,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should health memory failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			&stubDreamTrigger{EnabledFn: true, LastErr: errors.New("dream status failed")},
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf(
				"health status = %d, want %d; body=%s",
				resp.Code,
				http.StatusOK,
				resp.Body.String(),
			)
		}
		var payload contract.StatusPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if payload.Memory.Status != "unavailable" || payload.Memory.Reason == "" {
			t.Fatalf("health memory = %#v, want structured unavailable memory payload", payload.Memory)
		}
	})

	t.Run("Should health automation failure", func(t *testing.T) {
		fixture := newHandlerFixtureWithAutomation(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubAutomationManager{
				StatusFn: func(context.Context) (automationpkg.ManagerStatus, error) {
					return automationpkg.ManagerStatus{}, errors.New("automation status failed")
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf(
				"health status = %d, want %d; body=%s",
				resp.Code,
				http.StatusInternalServerError,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should daemon status session list failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return nil, errors.New("list failed")
				},
			},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf(
				"daemon status = %d, want %d; body=%s",
				resp.Code,
				http.StatusInternalServerError,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should daemon status observer failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{}, errors.New("observer failed")
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf(
				"daemon status = %d, want %d; body=%s",
				resp.Code,
				http.StatusInternalServerError,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should daemon status network enabled without service", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return []*session.Info{}, nil
				},
			},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = nil

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf(
				"daemon status = %d, want %d; body=%s",
				resp.Code,
				http.StatusOK,
				resp.Body.String(),
			)
		}
		var payload contract.StatusPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if payload.Daemon.Network == nil || payload.Daemon.Network.Status != "unavailable" {
			t.Fatalf("daemon network = %#v, want unavailable status", payload.Daemon.Network)
		}
	})

	t.Run("Should daemon status network missing payload", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return []*session.Info{}, nil
				},
			},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			StatusFn: func(context.Context) (*network.Status, error) {
				return nil, nil
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf(
				"daemon status = %d, want %d; body=%s",
				resp.Code,
				http.StatusOK,
				resp.Body.String(),
			)
		}
		var payload contract.StatusPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if payload.Daemon.Network == nil || payload.Daemon.Network.Status != "unavailable" {
			t.Fatalf("daemon network = %#v, want unavailable status", payload.Daemon.Network)
		}
	})

	t.Run("Should daemon status network failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return []*session.Info{}, nil
				},
			},
			testutil.StubObserver{
				HealthFn: func(context.Context) (observe.Health, error) {
					return observe.Health{Status: "ok", Version: "dev"}, nil
				},
			},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Config.Network.Enabled = true
		fixture.Handlers.Network = testutil.StubNetworkService{
			StatusFn: func(context.Context) (*network.Status, error) {
				return nil, errors.New("network failed")
			},
		}

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/status", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf(
				"daemon status = %d, want %d; body=%s",
				resp.Code,
				http.StatusOK,
				resp.Body.String(),
			)
		}
		var payload contract.StatusPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if payload.Daemon.Network == nil || payload.Daemon.Network.Status != "unavailable" {
			t.Fatalf("daemon network = %#v, want unavailable status", payload.Daemon.Network)
		}
	})
}

func TestBaseHandlersListSessionsErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("Should list all failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return nil, errors.New("list failed")
				},
			},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions", nil)
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf(
				"list sessions status = %d, want %d; body=%s",
				resp.Code,
				http.StatusInternalServerError,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should workspace lookup failure", func(t *testing.T) {
		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{
				ListAllFn: func(context.Context) ([]*session.Info, error) {
					return []*session.Info{{ID: "sess-1", WorkspaceID: "ws_alpha"}}, nil
				},
			},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{
				GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
					return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
				},
			},
			nil,
			nil,
		)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/sessions?workspace=alpha", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("list sessions status = %d, want %d; body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
		}
	})
}

func TestObserveStreamAndParseObserveQuery(t *testing.T) {
	t.Parallel()

	done := make(chan struct{})
	callCount := 0
	observer := testutil.StubObserver{
		QueryEventsFn: func(_ context.Context, _ store.EventSummaryQuery) ([]store.EventSummary, error) {
			callCount++
			ts := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
			switch callCount {
			case 1:
				return []store.EventSummary{
					{ID: "sum-1", SessionID: "sess-1", Type: "agent_message", AgentName: "coder", Timestamp: ts},
				}, nil
			case 2:
				close(done)
				return []store.EventSummary{
					{
						ID:        "sum-2",
						SessionID: "sess-1",
						Type:      "done",
						AgentName: "coder",
						Timestamp: ts.Add(time.Second),
					},
				}, nil
			default:
				return nil, nil
			}
		},
	}
	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, observer, testutil.StubWorkspaceService{}, nil, nil)
	fixture.Handlers.SetStreamDone(done)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/workspaces/ws-workspace/observe/events/stream?agent_name=coder",
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("observe stream status = %d, want %d", resp.Code, http.StatusOK)
	}
	if records := testutil.ParseSSE(t, resp.Body.String()); len(records) < 2 {
		t.Fatalf("observe stream records = %d, want at least 2", len(records))
	}
}

func TestBaseHandlersGetAgentNotFound(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	fixture.Handlers.AgentLoader = func(string, aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{}, os.ErrNotExist
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/agents/missing", nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("get missing agent status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}
