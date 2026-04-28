package core_test

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestBaseHandlersRejectInvalidRequestsAndMapErrors(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		CreateFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
			return nil, os.ErrNotExist
		},
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return nil, session.ErrSessionNotFound
		},
		ResumeFn: func(context.Context, string) (*session.Session, error) {
			return nil, session.ErrSessionNotFound
		},
		DeleteFn: func(context.Context, string) error {
			return session.ErrSessionNotFound
		},
		StopFn: func(context.Context, string) error {
			return session.ErrSessionNotFound
		},
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return nil, errors.New("list failed")
		},
	}
	observer := testutil.StubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return nil, errors.New("boom")
		},
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{}, errors.New("health failed")
		},
	}
	workspaces := testutil.StubWorkspaceService{
		RegisterFn: func(context.Context, workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{}, workspacepkg.ErrWorkspacePathTaken
		},
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
		},
		ResolveFn: func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
			return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceRootMissing
		},
		ResolveOrRegisterFn: func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
			return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceRootMissing
		},
	}

	fixture := newHandlerFixture(t, manager, observer, workspaces, nil, nil)

	requests := []struct {
		method string
		path   string
		body   []byte
		want   int
	}{
		{
			method: http.MethodPost,
			path:   "/sessions",
			body:   []byte(`{"agent_name":"coder"}`),
			want:   http.StatusBadRequest,
		},
		{
			method: http.MethodPost,
			path:   "/sessions",
			body:   []byte(`{"agent_name":"coder","workspace":"alpha"}`),
			want:   http.StatusNotFound,
		},
		{method: http.MethodGet, path: "/sessions/missing", want: http.StatusNotFound},
		{method: http.MethodPost, path: "/sessions/missing/resume", want: http.StatusNotFound},
		{method: http.MethodDelete, path: "/sessions/missing", want: http.StatusNotFound},
		{method: http.MethodGet, path: "/sessions/missing/events?since=bad", want: http.StatusBadRequest},
		{method: http.MethodGet, path: "/observe/events", want: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/observe/health", want: http.StatusInternalServerError},
		{method: http.MethodGet, path: "/daemon/status", want: http.StatusInternalServerError},
		{
			method: http.MethodPost,
			path:   "/workspaces",
			body:   []byte(`{"root_dir":"relative"}`),
			want:   http.StatusBadRequest,
		},
		{method: http.MethodGet, path: "/workspaces/ws-missing", want: http.StatusGone},
		{
			method: http.MethodPost,
			path:   "/workspaces/resolve",
			body:   []byte(`{"path":"/workspace"}`),
			want:   http.StatusGone,
		},
	}

	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, fixture.Engine, request.method, request.path, request.body)
			if resp.Code != request.want {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					request.method,
					request.path,
					resp.Code,
					request.want,
					resp.Body.String(),
				)
			}
		})
	}
}

func TestSessionHistoryEventsAndTranscriptErrorBranches(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return testutil.NewSessionInfo("sess-a"), nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			return nil, session.ErrSessionNotFound
		},
		HistoryFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
			return nil, session.ErrSessionNotFound
		},
		TranscriptFn: func(context.Context, string) ([]transcript.UIMessage, error) {
			return nil, session.ErrSessionNotFound
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)

	for _, path := range []string{
		"/sessions/sess-a/events",
		"/sessions/sess-a/history",
		"/sessions/sess-a/transcript",
	} {
		resp := performRequest(t, fixture.Engine, http.MethodGet, path, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d", path, resp.Code, http.StatusNotFound)
		}
	}
}

func TestStreamSessionAndObserveErrorBranches(t *testing.T) {
	t.Parallel()

	manager := testutil.StubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			info := testutil.NewSessionInfo("sess-a")
			info.State = session.StateStopped
			info.UpdatedAt = time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC)
			return info, nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			return nil, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, testutil.StubWorkspaceService{}, nil, nil)

	badStream := performRequest(t, fixture.Engine, http.MethodGet, "/sessions/sess-a/stream", nil)
	if badStream.Code != http.StatusOK {
		t.Fatalf("stream stopped status = %d, want %d", badStream.Code, http.StatusOK)
	}

	badHeader := testutil.PerformRequestWithHeaders(
		t,
		fixture.Engine,
		http.MethodGet,
		"/sessions/sess-a/stream",
		nil,
		map[string]string{"Last-Event-ID": "bad"},
	)
	if badHeader.Code != http.StatusBadRequest {
		t.Fatalf("stream bad header status = %d, want %d", badHeader.Code, http.StatusBadRequest)
	}

	observeFixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	observeBadHeader := testutil.PerformRequestWithHeaders(
		t,
		observeFixture.Engine,
		http.MethodGet,
		"/observe/events/stream",
		nil,
		map[string]string{"Last-Event-ID": "bad"},
	)
	if observeBadHeader.Code != http.StatusBadRequest {
		t.Fatalf("observe bad header status = %d, want %d", observeBadHeader.Code, http.StatusBadRequest)
	}
}

func TestListAgentsHandlesMissingDirectory(t *testing.T) {
	t.Parallel()

	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	if err := os.RemoveAll(fixture.HomePaths.AgentsDir); err != nil {
		t.Fatalf("RemoveAll(AgentsDir) error = %v", err)
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list agents missing dir status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestListAgentsWorkspaceResolverUnavailable(t *testing.T) {
	t.Parallel()

	t.Run("Should return service unavailable when workspace resolver is missing", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Workspaces = nil

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/agents?workspace=alpha", nil)
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf(
				"workspace agents status = %d, want %d; body=%s",
				resp.Code,
				http.StatusServiceUnavailable,
				resp.Body.String(),
			)
		}
		var payload contract.ErrorPayload
		testutil.DecodeJSONResponse(t, resp, &payload)
		if !strings.Contains(payload.Error, "workspace resolver unavailable") {
			t.Fatalf("workspace agents error = %#v, want resolver unavailable detail", payload)
		}
	})
}

func TestListAgentsSkipsUnreadableDefinitions(t *testing.T) {
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
	testutil.WriteAgentDef(t, fixture.HomePaths, "broken")
	fixture.Handlers.AgentLoader = func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		if name == "broken" {
			return aghconfig.AgentDef{}, errors.New("bad agent")
		}
		return aghconfig.LoadAgentDef(name, homePaths)
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/agents", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("list agents skip unreadable status = %d, want %d", resp.Code, http.StatusOK)
	}
}

func TestMemoryHelpersAndMissingStoreBranches(t *testing.T) {
	t.Parallel()

	store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	workspace := t.TempDir()
	globalDoc := []byte(memoryDocument(t, "Shared", memory.MemoryTypeUser, "global"))
	workspaceDoc := []byte(memoryDocument(t, "Shared", memory.MemoryTypeProject, "workspace"))
	if err := store.Write(memory.ScopeGlobal, "shared.md", globalDoc); err != nil {
		t.Fatalf("Write(global) error = %v", err)
	}
	if err := store.ForWorkspace(workspace).Write(memory.ScopeWorkspace, "shared.md", workspaceDoc); err != nil {
		t.Fatalf("Write(workspace) error = %v", err)
	}
	if err := store.ForWorkspace(workspace).
		Write(memory.ScopeWorkspace, "workspace-only.md", workspaceDoc); err != nil {
		t.Fatalf("Write(workspace-only) error = %v", err)
	}

	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		store,
		nil,
	)
	if _, err := fixture.Handlers.ResolveMemoryLocation("workspace-only.md", "", workspace); err != nil {
		t.Fatalf("ResolveMemoryLocation(workspace-only) error = %v", err)
	}
	if _, err := fixture.Handlers.ResolveMemoryLocation(
		"shared.md",
		"",
		workspace,
	); !errors.Is(
		err,
		memory.ErrValidation,
	) {
		t.Fatalf("ResolveMemoryLocation(shared) error = %v, want validation", err)
	}
	if _, _, err := core.ResolveMemoryWriteScope(contract.MemoryWriteRequest{}); !errors.Is(err, memory.ErrValidation) {
		t.Fatalf("ResolveMemoryWriteScope(empty) error = %v, want validation", err)
	}

	noStoreFixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		nil,
		nil,
	)
	requests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/memory"},
		{method: http.MethodGet, path: "/memory/valid.md?scope=global"},
		{
			method: http.MethodPut,
			path:   "/memory/valid.md",
			body: []byte(
				`{"scope":"global","content":"` + escapeJSON(
					memoryDocument(t, "Valid", memory.MemoryTypeUser, "hello"),
				) + `"}`,
			),
		},
		{method: http.MethodDelete, path: "/memory/valid.md?scope=global"},
	}
	for _, request := range requests {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			resp := performRequest(t, noStoreFixture.Engine, request.method, request.path, request.body)
			if resp.Code != http.StatusInternalServerError {
				t.Fatalf(
					"%s %s status = %d, want %d",
					request.method,
					request.path,
					resp.Code,
					http.StatusInternalServerError,
				)
			}
		})
	}
}

func TestWorkspaceUpdateValidationAndDeleteErrors(t *testing.T) {
	t.Parallel()

	workspace := workspacepkg.Workspace{ID: "ws_alpha", RootDir: t.TempDir(), Name: "alpha"}
	workspaces := testutil.StubWorkspaceService{
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspace, nil
		},
		UnregisterFn: func(context.Context, string) error {
			return workspacepkg.ErrWorkspaceHasSessions
		},
		UpdateFn: func(context.Context, string, workspacepkg.UpdateOptions) error {
			return nil
		},
	}
	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, workspaces, nil, nil)

	badUpdate := performRequest(t, fixture.Engine, http.MethodPatch, "/workspaces/ws_alpha", []byte(`{"name":""}`))
	if badUpdate.Code != http.StatusBadRequest {
		t.Fatalf("bad update status = %d, want %d", badUpdate.Code, http.StatusBadRequest)
	}

	deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/workspaces/ws_alpha", nil)
	if deleteResp.Code != http.StatusConflict {
		t.Fatalf("delete conflict status = %d, want %d", deleteResp.Code, http.StatusConflict)
	}
}

func TestWorkspaceValidationBranches(t *testing.T) {
	t.Parallel()

	workspace := workspacepkg.Workspace{ID: "ws_alpha", RootDir: t.TempDir(), Name: "alpha"}
	workspaces := testutil.StubWorkspaceService{
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspace, nil
		},
	}
	fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, workspaces, nil, nil)

	createResp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/workspaces",
		[]byte(`{"root_dir":"`+workspace.RootDir+`","add_dirs":["relative"]}`),
	)
	if createResp.Code != http.StatusBadRequest {
		t.Fatalf("create invalid add_dirs status = %d, want %d", createResp.Code, http.StatusBadRequest)
	}

	updateResp := performRequest(
		t,
		fixture.Engine,
		http.MethodPatch,
		"/workspaces/ws_alpha",
		[]byte(`{"add_dirs":["relative"]}`),
	)
	if updateResp.Code != http.StatusBadRequest {
		t.Fatalf("update invalid add_dirs status = %d, want %d", updateResp.Code, http.StatusBadRequest)
	}

	resolveResp := performRequest(
		t,
		fixture.Engine,
		http.MethodPost,
		"/workspaces/resolve",
		[]byte(`{"path":"relative"}`),
	)
	if resolveResp.Code != http.StatusBadRequest {
		t.Fatalf("resolve invalid path status = %d, want %d", resolveResp.Code, http.StatusBadRequest)
	}
}

func TestMemoryErrorAndDisabledBranches(t *testing.T) {
	t.Parallel()

	store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	fixture := newHandlerFixture(
		t,
		testutil.StubSessionManager{},
		testutil.StubObserver{},
		testutil.StubWorkspaceService{},
		store,
		nil,
	)

	readMissing := performRequest(t, fixture.Engine, http.MethodGet, "/memory/missing.md?scope=global", nil)
	if readMissing.Code != http.StatusNotFound {
		t.Fatalf("read missing status = %d, want %d", readMissing.Code, http.StatusNotFound)
	}

	deleteMissing := performRequest(t, fixture.Engine, http.MethodDelete, "/memory/missing.md?scope=global", nil)
	if deleteMissing.Code != http.StatusNotFound {
		t.Fatalf("delete missing status = %d, want %d", deleteMissing.Code, http.StatusNotFound)
	}

	badWrite := performRequest(
		t,
		fixture.Engine,
		http.MethodPut,
		"/memory/bad.md",
		[]byte(`{"scope":"global","content":"not frontmatter"}`),
	)
	if badWrite.Code != http.StatusBadRequest {
		t.Fatalf("bad write status = %d, want %d", badWrite.Code, http.StatusBadRequest)
	}

	badConsolidate := performRequest(t, fixture.Engine, http.MethodPost, "/memory/consolidate", []byte(`{`))
	if badConsolidate.Code != http.StatusBadRequest {
		t.Fatalf("bad consolidate status = %d, want %d", badConsolidate.Code, http.StatusBadRequest)
	}

	disabledConsolidate := performRequest(t, fixture.Engine, http.MethodPost, "/memory/consolidate", nil)
	if disabledConsolidate.Code != http.StatusOK {
		t.Fatalf("disabled consolidate status = %d, want %d", disabledConsolidate.Code, http.StatusOK)
	}
}
