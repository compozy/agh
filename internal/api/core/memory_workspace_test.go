package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestMemoryHandlersAndHelpers(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (handlerFixture, string, *stubDreamTrigger) {
		t.Helper()

		store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
		if err := store.EnsureDirs(); err != nil {
			t.Fatalf("EnsureDirs() error = %v", err)
		}

		workspace := filepath.Join(t.TempDir(), "workspace with space")
		if err := os.MkdirAll(workspace, 0o755); err != nil {
			t.Fatalf("MkdirAll(workspace) error = %v", err)
		}
		if err := store.Write(
			memory.ScopeGlobal,
			"global.md",
			[]byte(memoryDocument(t, "Global", memory.MemoryTypeUser, "hello")),
		); err != nil {
			t.Fatalf("Write(global) error = %v", err)
		}
		if err := store.ForWorkspace(workspace).
			Write(memory.ScopeWorkspace, "workspace.md", []byte(memoryDocument(t, "Workspace", memory.MemoryTypeProject, "world"))); err != nil {
			t.Fatalf("Write(workspace) error = %v", err)
		}

		trigger := &stubDreamTrigger{
			EnabledFn: true,
			Triggered: true,
			Reason:    "queued",
			Last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
		}
		manager := testutil.StubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) {
				info := testutil.NewSessionInfo("sess-a")
				info.Workspace = workspace
				return []*session.Info{info}, nil
			},
		}
		observer := testutil.StubObserver{
			HealthFn: func(context.Context) (observe.Health, error) {
				return observe.Health{Status: "ok", ActiveSessions: 1}, nil
			},
		}

		return newHandlerFixture(
			t,
			manager,
			observer,
			testutil.StubWorkspaceService{},
			store,
			trigger,
		), workspace, trigger
	}

	t.Run("Should list memory for a workspace", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _ := setup(t)
		query := url.Values{}
		query.Set("workspace", workspace)
		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory?"+query.Encode(), nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list memory status = %d, want %d", listResp.Code, http.StatusOK)
		}

		var headers []memory.Header
		testutil.DecodeJSONResponse(t, listResp, &headers)
		if len(headers) != 2 {
			t.Fatalf("memory headers len = %d, want 2", len(headers))
		}
		if headers[0].Filename == "" || headers[1].Filename == "" {
			t.Fatalf("memory headers = %#v", headers)
		}
	})

	t.Run("Should read global memory", func(t *testing.T) {
		t.Parallel()

		fixture, _, _ := setup(t)
		readResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/global.md?scope=global", nil)
		if readResp.Code != http.StatusOK {
			t.Fatalf("read memory status = %d, want %d", readResp.Code, http.StatusOK)
		}

		var payload contract.MemoryReadResponse
		testutil.DecodeJSONResponse(t, readResp, &payload)
		if payload.Content == "" {
			t.Fatalf("read payload = %#v, want non-empty content", payload)
		}
	})

	t.Run("Should write workspace memory", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _ := setup(t)
		writeBody, err := json.Marshal(contract.MemoryWriteRequest{
			Scope:     "workspace",
			Workspace: workspace,
			Content:   memoryDocument(t, "Project", memory.MemoryTypeProject, "updated"),
		})
		if err != nil {
			t.Fatalf("json.Marshal(write request) error = %v", err)
		}
		writeResp := performRequest(t, fixture.Engine, http.MethodPut, "/memory/new.md", writeBody)
		if writeResp.Code != http.StatusOK {
			t.Fatalf(
				"write memory status = %d, want %d; body=%s",
				writeResp.Code,
				http.StatusOK,
				writeResp.Body.String(),
			)
		}

		var payload contract.MemoryMutationResponse
		testutil.DecodeJSONResponse(t, writeResp, &payload)
		if !payload.OK {
			t.Fatalf("write payload = %#v, want ok=true", payload)
		}
	})

	t.Run("Should delete workspace memory", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _ := setup(t)
		writeBody, err := json.Marshal(contract.MemoryWriteRequest{
			Scope:     "workspace",
			Workspace: workspace,
			Content:   memoryDocument(t, "Project", memory.MemoryTypeProject, "updated"),
		})
		if err != nil {
			t.Fatalf("json.Marshal(write request) error = %v", err)
		}
		writeResp := performRequest(t, fixture.Engine, http.MethodPut, "/memory/new.md", writeBody)
		if writeResp.Code != http.StatusOK {
			t.Fatalf("write memory status = %d, want %d", writeResp.Code, http.StatusOK)
		}

		query := url.Values{}
		query.Set("scope", "workspace")
		query.Set("workspace", workspace)
		deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/memory/new.md?"+query.Encode(), nil)
		if deleteResp.Code != http.StatusOK {
			t.Fatalf("delete memory status = %d, want %d", deleteResp.Code, http.StatusOK)
		}

		var payload contract.MemoryMutationResponse
		testutil.DecodeJSONResponse(t, deleteResp, &payload)
		if !payload.OK {
			t.Fatalf("delete payload = %#v, want ok=true", payload)
		}
	})

	t.Run("Should trigger dream consolidation", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, trigger := setup(t)
		body, err := json.Marshal(contract.MemoryConsolidateRequest{Workspace: workspace})
		if err != nil {
			t.Fatalf("json.Marshal(consolidate request) error = %v", err)
		}
		consolidateResp := performRequest(t, fixture.Engine, http.MethodPost, "/memory/consolidate", body)
		if consolidateResp.Code != http.StatusOK {
			t.Fatalf("consolidate status=%d want=%d", consolidateResp.Code, http.StatusOK)
		}
		if trigger.Calls != 1 || trigger.Workspace != workspace {
			t.Fatalf("trigger calls=%d workspace=%q", trigger.Calls, trigger.Workspace)
		}

		var payload contract.MemoryConsolidateResponse
		testutil.DecodeJSONResponse(t, consolidateResp, &payload)
		if !payload.Triggered || payload.Reason != "queued" {
			t.Fatalf("consolidate payload = %#v", payload)
		}
	})

	t.Run("Should report observe health", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _ := setup(t)
		query := url.Values{}
		query.Set("workspace", workspace)
		healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/observe/health?"+query.Encode(), nil)
		if healthResp.Code != http.StatusOK {
			t.Fatalf("health status = %d, want %d", healthResp.Code, http.StatusOK)
		}

		var payload struct {
			Health observe.Health               `json:"health"`
			Memory contract.MemoryHealthPayload `json:"memory"`
		}
		testutil.DecodeJSONResponse(t, healthResp, &payload)
		if payload.Health.Status != "ok" || payload.Health.ActiveSessions != 1 {
			t.Fatalf("health payload = %#v", payload.Health)
		}
		if payload.Memory.WorkspaceFiles != 1 || !payload.Memory.DreamEnabled {
			t.Fatalf("memory payload = %#v", payload.Memory)
		}
	})

	t.Run("Should report memory health directly", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _ := setup(t)
		query := url.Values{}
		query.Set("workspace", workspace)
		healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/health?"+query.Encode(), nil)
		if healthResp.Code != http.StatusOK {
			t.Fatalf("memory health status = %d, want %d", healthResp.Code, http.StatusOK)
		}

		var payload contract.MemoryHealthPayload
		testutil.DecodeJSONResponse(t, healthResp, &payload)
		if payload.Status != "ok" || !payload.Configured || payload.WorkspaceCount != 1 || payload.WorkspaceFiles != 1 {
			t.Fatalf("memory health payload = %#v", payload)
		}
	})

	t.Run("Should report unavailable memory health when store is missing", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/health", nil)
		if healthResp.Code != http.StatusOK {
			t.Fatalf("memory health status = %d, want %d", healthResp.Code, http.StatusOK)
		}

		var payload contract.MemoryHealthPayload
		testutil.DecodeJSONResponse(t, healthResp, &payload)
		if payload.Status != "unavailable" || payload.Configured || payload.Reason == "" {
			t.Fatalf("memory health payload = %#v, want unavailable and unconfigured", payload)
		}
	})

	t.Run("Should report disabled memory health before missing store", func(t *testing.T) {
		t.Parallel()

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			nil,
			nil,
		)
		fixture.Handlers.Config.Memory.Enabled = false
		healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/health", nil)
		if healthResp.Code != http.StatusOK {
			t.Fatalf("memory health status = %d, want %d", healthResp.Code, http.StatusOK)
		}

		var payload contract.MemoryHealthPayload
		testutil.DecodeJSONResponse(t, healthResp, &payload)
		if payload.Status != "disabled" || payload.Reason == "" || payload.Enabled {
			t.Fatalf("memory health payload = %#v, want disabled", payload)
		}
	})

	t.Run("Should report degraded memory health for orphaned catalog rows", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		workspace := filepath.Join(baseDir, "workspace")
		store := memory.NewStore(
			filepath.Join(baseDir, "global"),
			memory.WithCatalogDatabasePath(filepath.Join(baseDir, "agh.db")),
		).ForWorkspace(workspace)
		if err := store.EnsureDirs(); err != nil {
			t.Fatalf("EnsureDirs() error = %v", err)
		}
		if err := store.Write(
			memory.ScopeWorkspace,
			"orphan.md",
			[]byte(memoryDocument(t, "Orphan", memory.MemoryTypeProject, "orphan signal")),
		); err != nil {
			t.Fatalf("Write(workspace) error = %v", err)
		}
		if _, err := store.Search(context.Background(), "orphan signal", memory.SearchOptions{
			Workspace: workspace,
			Limit:     5,
		}); err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if err := os.Remove(filepath.Join(workspace, aghconfig.DirName, "memory", "orphan.md")); err != nil {
			t.Fatalf("os.Remove(orphan memory) error = %v", err)
		}

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			store,
			nil,
		)
		query := url.Values{}
		query.Set("workspace", workspace)
		healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/health?"+query.Encode(), nil)
		if healthResp.Code != http.StatusOK {
			t.Fatalf("memory health status = %d, want %d", healthResp.Code, http.StatusOK)
		}

		var payload contract.MemoryHealthPayload
		testutil.DecodeJSONResponse(t, healthResp, &payload)
		if payload.Status != "degraded" || payload.OrphanedFiles != 1 || !strings.Contains(payload.Reason, "orphaned") {
			t.Fatalf("memory health payload = %#v, want degraded orphan report", payload)
		}
	})

	t.Run("Should list filtered redacted memory history", func(t *testing.T) {
		t.Parallel()

		baseDir := t.TempDir()
		workspace := filepath.Join(baseDir, "workspace")
		store := memory.NewStore(
			filepath.Join(baseDir, "global"),
			memory.WithCatalogDatabasePath(filepath.Join(baseDir, "agh.db")),
		).ForWorkspace(workspace)
		if err := store.EnsureDirs(); err != nil {
			t.Fatalf("EnsureDirs() error = %v", err)
		}
		if err := store.Write(
			memory.ScopeWorkspace,
			"project.md",
			[]byte(memoryDocument(t, "Project", memory.MemoryTypeProject, "common signal")),
		); err != nil {
			t.Fatalf("Write(workspace) error = %v", err)
		}
		since := time.Now().Add(-time.Second).UTC()
		if _, err := store.Search(context.Background(), "common token=super-secret", memory.SearchOptions{
			Workspace: workspace,
			Limit:     5,
		}); err != nil {
			t.Fatalf("Search() error = %v", err)
		}

		fixture := newHandlerFixture(
			t,
			testutil.StubSessionManager{},
			testutil.StubObserver{},
			testutil.StubWorkspaceService{},
			store,
			nil,
		)
		query := url.Values{}
		query.Set("workspace", workspace)
		query.Set("operation", "memory.search")
		query.Set("since", since.Format(time.RFC3339Nano))
		query.Set("limit", "2")
		historyResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/history?"+query.Encode(), nil)
		if historyResp.Code != http.StatusOK {
			t.Fatalf(
				"memory history status = %d, want %d; body=%s",
				historyResp.Code,
				http.StatusOK,
				historyResp.Body.String(),
			)
		}

		var payload contract.MemoryHistoryResponse
		testutil.DecodeJSONResponse(t, historyResp, &payload)
		if len(payload.Operations) != 1 {
			t.Fatalf("len(payload.Operations) = %d, want 1; payload=%#v", len(payload.Operations), payload)
		}
		got := payload.Operations[0]
		if got.Operation != "memory.search" || got.Workspace != workspace {
			t.Fatalf("operation payload = %#v, want workspace memory.search", got)
		}
		if strings.Contains(got.Summary, "super-secret") || !strings.Contains(got.Summary, "token=[REDACTED]") {
			t.Fatalf("operation summary = %q, want redacted token", got.Summary)
		}
	})

	t.Run("Should map validation errors to bad requests", func(t *testing.T) {
		t.Parallel()

		if status := core.StatusForMemoryError(
			core.NewMemoryValidationError(errors.New("bad")),
		); status != http.StatusBadRequest {
			t.Fatalf("StatusForMemoryError(validation) = %d, want %d", status, http.StatusBadRequest)
		}
	})
}

func TestWorkspaceHandlersDelegateToService(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) (handlerFixture, workspacepkg.Workspace, workspacepkg.ResolvedWorkspace, *bool, *bool, *bool, string, string) {
		t.Helper()

		rootDir := filepath.Join(t.TempDir(), "root dir")
		addDir := filepath.Join(t.TempDir(), "add dir")
		if err := os.MkdirAll(rootDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(rootDir) error = %v", err)
		}
		if err := os.MkdirAll(addDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(addDir) error = %v", err)
		}
		workspace := workspacepkg.Workspace{
			ID:             "ws_alpha",
			RootDir:        rootDir,
			AdditionalDirs: []string{addDir},
			Name:           "alpha",
			DefaultAgent:   "coder",
			EnvironmentRef: "daytona-dev",
			CreatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 4, 3, 12, 1, 0, 0, time.UTC),
		}
		resolved := workspacepkg.ResolvedWorkspace{
			Workspace: workspace,
			Config: aghconfig.Config{
				Providers: map[string]aghconfig.ProviderConfig{
					"alpha": {Command: "alpha --acp"},
				},
			},
			Agents: []aghconfig.AgentDef{{
				Name:     "coder",
				Provider: "fake",
				Prompt:   "hello",
			}},
			Skills: []workspacepkg.SkillPath{{
				Dir:    filepath.Join(rootDir, ".skills", "build"),
				Source: "workspace",
			}},
		}
		updateCalled := false
		deleteCalled := false
		resolveCalled := false
		workspaces := testutil.StubWorkspaceService{
			RegisterFn: func(_ context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
				if opts.RootDir != rootDir || len(opts.AdditionalDirs) != 1 ||
					opts.DefaultAgent != "coder" ||
					opts.EnvironmentRef != "daytona-dev" {
					t.Fatalf("Register opts = %#v", opts)
				}
				return workspace, nil
			},
			ListFn: func(context.Context) ([]workspacepkg.Workspace, error) {
				return []workspacepkg.Workspace{workspace}, nil
			},
			GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
				return workspace, nil
			},
			ResolveFn: func(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
				return resolved, nil
			},
			UpdateFn: func(_ context.Context, id string, opts workspacepkg.UpdateOptions) error {
				updateCalled = true
				if id != workspace.ID || opts.Name == nil || *opts.Name != "beta" {
					t.Fatalf("Update call = %q %#v", id, opts)
				}
				if opts.EnvironmentRef != nil && *opts.EnvironmentRef != "local-dev" {
					t.Fatalf("Update environment ref = %#v, want local-dev", opts.EnvironmentRef)
				}
				return nil
			},
			UnregisterFn: func(_ context.Context, id string) error {
				deleteCalled = true
				if id != workspace.ID {
					t.Fatalf("Unregister id = %q, want %q", id, workspace.ID)
				}
				return nil
			},
			ResolveOrRegisterFn: func(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
				resolveCalled = true
				if path != rootDir {
					t.Fatalf("ResolveOrRegister path = %q, want %q", path, rootDir)
				}
				return resolved, nil
			},
		}
		manager := testutil.StubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) {
				info := testutil.NewSessionInfo("sess-a")
				info.WorkspaceID = workspace.ID
				return []*session.Info{info}, nil
			},
		}

		return newHandlerFixture(
			t,
			manager,
			testutil.StubObserver{},
			workspaces,
			nil,
			nil,
		), workspace, resolved, &updateCalled, &deleteCalled, &resolveCalled, rootDir, addDir
	}

	t.Run("Should create a workspace", func(t *testing.T) {
		t.Parallel()

		fixture, _, _, _, _, _, rootDir, addDir := setup(t)
		createBody, err := json.Marshal(contract.CreateWorkspaceRequest{
			RootDir:        rootDir,
			AddDirs:        []string{addDir},
			Name:           "alpha",
			DefaultAgent:   "coder",
			EnvironmentRef: "daytona-dev",
		})
		if err != nil {
			t.Fatalf("json.Marshal(create workspace request) error = %v", err)
		}
		createResp := performRequest(t, fixture.Engine, http.MethodPost, "/workspaces", createBody)
		if createResp.Code != http.StatusCreated {
			t.Fatalf("create workspace status = %d, want %d", createResp.Code, http.StatusCreated)
		}

		var payload struct {
			Workspace contract.WorkspacePayload `json:"workspace"`
		}
		testutil.DecodeJSONResponse(t, createResp, &payload)
		if payload.Workspace.RootDir != rootDir || len(payload.Workspace.AddDirs) != 1 ||
			payload.Workspace.AddDirs[0] != addDir {
			t.Fatalf("create workspace payload = %#v", payload.Workspace)
		}
	})

	t.Run("Should list workspaces", func(t *testing.T) {
		t.Parallel()

		fixture, _, _, _, _, _, _, _ := setup(t)
		listResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list workspaces status = %d, want %d", listResp.Code, http.StatusOK)
		}

		var payload struct {
			Workspaces []contract.WorkspacePayload `json:"workspaces"`
		}
		testutil.DecodeJSONResponse(t, listResp, &payload)
		if len(payload.Workspaces) != 1 || payload.Workspaces[0].ID != "ws_alpha" {
			t.Fatalf("list workspaces payload = %#v", payload.Workspaces)
		}
	})

	t.Run("Should get a workspace with sessions", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, resolved, _, _, _, _, _ := setup(t)
		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces/"+workspace.ID, nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get workspace status = %d, want %d", getResp.Code, http.StatusOK)
		}

		var getPayload contract.WorkspaceDetailPayload
		testutil.DecodeJSONResponse(t, getResp, &getPayload)
		if len(getPayload.Sessions) != 1 || getPayload.Sessions[0].WorkspaceID != workspace.ID {
			t.Fatalf("sessions payload = %#v", getPayload.Sessions)
		}
		expectedProviders := core.SessionProviderOptionPayloadsFromConfig(&resolved.Config)
		if got, want := len(getPayload.Providers), len(expectedProviders); got != want {
			t.Fatalf("len(providers) = %d, want %d (%#v)", got, want, getPayload.Providers)
		}
		for i, want := range expectedProviders {
			if got := getPayload.Providers[i]; got != want {
				t.Fatalf("providers[%d] = %#v, want %#v", i, got, want)
			}
		}
	})

	t.Run("Should merge projected catalog agents into workspace detail", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _, _, _, _, _, _ := setup(t)
		fixture.Handlers.AgentCatalog = stubAgentCatalog{
			agents: []aghconfig.AgentDef{
				{
					Name:     "coder",
					Provider: "catalog-should-not-win",
					Prompt:   "global duplicate",
				},
				{
					Name:     "qa-extension-agent",
					Provider: "codex",
					Prompt:   "extension agent",
				},
			},
		}

		getResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces/"+workspace.ID, nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get workspace status = %d, want %d", getResp.Code, http.StatusOK)
		}

		var getPayload contract.WorkspaceDetailPayload
		testutil.DecodeJSONResponse(t, getResp, &getPayload)
		if got, want := len(getPayload.Agents), 2; got != want {
			t.Fatalf("len(agents) = %d, want %d: %#v", got, want, getPayload.Agents)
		}
		if got, want := getPayload.Agents[0].Name, "coder"; got != want {
			t.Fatalf("agents[0].name = %q, want %q", got, want)
		}
		if got, want := getPayload.Agents[0].Provider, "fake"; got != want {
			t.Fatalf("agents[0].provider = %q, want workspace-scoped provider %q", got, want)
		}
		if got, want := getPayload.Agents[1].Name, "qa-extension-agent"; got != want {
			t.Fatalf("agents[1].name = %q, want %q", got, want)
		}
	})

	t.Run("Should update a workspace via the service", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _, updateCalled, _, _, _, _ := setup(t)
		updateResp := performRequest(
			t,
			fixture.Engine,
			http.MethodPatch,
			"/workspaces/"+workspace.ID,
			[]byte(`{"name":"beta"}`),
		)
		if updateResp.Code != http.StatusOK || !*updateCalled {
			t.Fatalf("update status=%d called=%v", updateResp.Code, *updateCalled)
		}

		var payload struct {
			Workspace contract.WorkspacePayload `json:"workspace"`
		}
		testutil.DecodeJSONResponse(t, updateResp, &payload)
		if payload.Workspace.Name != "alpha" {
			t.Fatalf("update workspace payload = %#v", payload.Workspace)
		}
	})

	t.Run("Should delete a workspace via the service", func(t *testing.T) {
		t.Parallel()

		fixture, workspace, _, _, deleteCalled, _, _, _ := setup(t)
		deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/workspaces/"+workspace.ID, nil)
		if deleteResp.Code != http.StatusNoContent || !*deleteCalled {
			t.Fatalf("delete status=%d called=%v", deleteResp.Code, *deleteCalled)
		}
	})

	t.Run("Should resolve a workspace path via the service", func(t *testing.T) {
		t.Parallel()

		fixture, _, _, _, _, resolveCalled, rootDir, _ := setup(t)
		resolveBody, err := json.Marshal(contract.ResolveWorkspaceRequest{Path: rootDir})
		if err != nil {
			t.Fatalf("json.Marshal(resolve workspace request) error = %v", err)
		}
		resolveResp := performRequest(t, fixture.Engine, http.MethodPost, "/workspaces/resolve", resolveBody)
		if resolveResp.Code != http.StatusOK || !*resolveCalled {
			t.Fatalf("resolve status=%d called=%v", resolveResp.Code, *resolveCalled)
		}

		var payload struct {
			Workspace contract.WorkspacePayload `json:"workspace"`
		}
		testutil.DecodeJSONResponse(t, resolveResp, &payload)
		if payload.Workspace.RootDir != rootDir {
			t.Fatalf("resolve workspace payload = %#v", payload.Workspace)
		}
	})
}

func TestWorkspaceUpdateSupportsAddDirsAndDefaultAgent(t *testing.T) {
	t.Parallel()

	t.Run("Should update add_dirs and default_agent", func(t *testing.T) {
		t.Parallel()

		rootDir := t.TempDir()
		addDir := t.TempDir()
		workspace := workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "alpha"}
		var captured workspacepkg.UpdateOptions
		workspaces := testutil.StubWorkspaceService{
			GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
				return workspace, nil
			},
			UpdateFn: func(_ context.Context, _ string, opts workspacepkg.UpdateOptions) error {
				captured = opts
				return nil
			},
		}
		fixture := newHandlerFixture(t, testutil.StubSessionManager{}, testutil.StubObserver{}, workspaces, nil, nil)

		resp := performRequest(
			t,
			fixture.Engine,
			http.MethodPatch,
			"/workspaces/ws_alpha",
			[]byte(`{"add_dirs":["`+addDir+`"],"default_agent":"coder","environment_ref":"local-dev"}`),
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("update add_dirs/default_agent status = %d, want %d", resp.Code, http.StatusOK)
		}
		if captured.AdditionalDirs == nil || len(*captured.AdditionalDirs) != 1 ||
			(*captured.AdditionalDirs)[0] != addDir {
			t.Fatalf("captured add dirs = %#v", captured.AdditionalDirs)
		}
		if captured.DefaultAgent == nil || *captured.DefaultAgent != "coder" {
			t.Fatalf("captured default agent = %#v", captured.DefaultAgent)
		}
		if captured.EnvironmentRef == nil || *captured.EnvironmentRef != "local-dev" {
			t.Fatalf("captured environment ref = %#v", captured.EnvironmentRef)
		}
	})
}

func memoryDocument(t *testing.T, name string, typ memory.Type, body string) string {
	t.Helper()

	header := memory.Header{
		Name:        name,
		Description: "desc",
		Type:        typ,
	}
	metadata, err := yaml.Marshal(header)
	if err != nil {
		t.Fatalf("yaml.Marshal() error = %v", err)
	}
	return "---\n" + string(metadata) + "---\n\n" + body
}

func escapeJSON(value string) string {
	quoted := strconv.Quote(value)
	return quoted[1 : len(quoted)-1]
}
