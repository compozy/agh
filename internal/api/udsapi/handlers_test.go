package udsapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type stubExtensionService struct {
	ListFn    func(context.Context) ([]contract.ExtensionPayload, error)
	InstallFn func(context.Context, contract.InstallExtensionRequest) (contract.ExtensionPayload, error)
	EnableFn  func(context.Context, string) (contract.ExtensionPayload, error)
	DisableFn func(context.Context, string) (contract.ExtensionPayload, error)
	StatusFn  func(context.Context, string) (contract.ExtensionPayload, error)
}

func (s stubExtensionService) List(ctx context.Context) ([]contract.ExtensionPayload, error) {
	if s.ListFn == nil {
		return nil, nil
	}
	return s.ListFn(ctx)
}

func (s stubExtensionService) Install(ctx context.Context, req contract.InstallExtensionRequest) (contract.ExtensionPayload, error) {
	if s.InstallFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.InstallFn(ctx, req)
}

func (s stubExtensionService) Enable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.EnableFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.EnableFn(ctx, name)
}

func (s stubExtensionService) Disable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.DisableFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.DisableFn(ctx, name)
}

func (s stubExtensionService) Status(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.StatusFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.StatusFn(ctx, name)
}

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
		"DELETE /api/automation/jobs/:id",
		"DELETE /api/automation/triggers/:id",
		"DELETE /api/memory/:filename",
		"DELETE /api/sessions/:id",
		"DELETE /api/tasks/:id/dependencies/:depends_on_id",
		"DELETE /api/workspaces/:id",
		"GET /api/agents",
		"GET /api/agents/:name",
		"GET /api/automation/jobs",
		"GET /api/automation/jobs/:id",
		"GET /api/automation/jobs/:id/runs",
		"GET /api/automation/runs",
		"GET /api/automation/runs/:id",
		"GET /api/automation/triggers",
		"GET /api/automation/triggers/:id",
		"GET /api/automation/triggers/:id/runs",
		"GET /api/bridges",
		"GET /api/bridges/:id",
		"GET /api/bridges/:id/routes",
		"GET /api/bridges/providers",
		"GET /api/daemon/status",
		"GET /api/extensions",
		"GET /api/extensions/:name",
		"GET /api/hooks/catalog",
		"GET /api/hooks/events",
		"GET /api/hooks/runs",
		"GET /api/memory",
		"GET /api/memory/:filename",
		"GET /api/network/inbox",
		"GET /api/network/peers",
		"GET /api/network/peers/:peer_id",
		"GET /api/network/channels",
		"GET /api/network/channels/:channel",
		"GET /api/network/channels/:channel/messages",
		"GET /api/network/status",
		"GET /api/observe/events",
		"GET /api/observe/events/stream",
		"GET /api/observe/health",
		"GET /api/sessions",
		"GET /api/sessions/:id",
		"GET /api/sessions/:id/events",
		"GET /api/sessions/:id/history",
		"GET /api/sessions/:id/transcript",
		"GET /api/sessions/:id/stream",
		"GET /api/skills",
		"GET /api/skills/:name",
		"GET /api/skills/:name/content",
		"GET /api/tasks",
		"GET /api/tasks/:id",
		"GET /api/tasks/:id/runs",
		"GET /api/workspaces",
		"GET /api/workspaces/:id",
		"PATCH /api/automation/jobs/:id",
		"PATCH /api/automation/triggers/:id",
		"PATCH /api/bridges/:id",
		"PATCH /api/tasks/:id",
		"PATCH /api/workspaces/:id",
		"POST /api/automation/jobs",
		"POST /api/automation/jobs/:id/trigger",
		"POST /api/automation/triggers",
		"POST /api/bridges",
		"POST /api/bridges/:id/disable",
		"POST /api/bridges/:id/enable",
		"POST /api/bridges/:id/restart",
		"POST /api/bridges/:id/test-delivery",
		"POST /api/extensions",
		"POST /api/extensions/:name/disable",
		"POST /api/extensions/:name/enable",
		"POST /api/memory/consolidate",
		"POST /api/network/channels",
		"POST /api/network/send",
		"POST /api/sessions",
		"POST /api/sessions/:id/approve",
		"POST /api/sessions/:id/prompt",
		"POST /api/sessions/:id/resume",
		"POST /api/skills/:name/disable",
		"POST /api/skills/:name/enable",
		"POST /api/task-runs/:id/attach-session",
		"POST /api/task-runs/:id/cancel",
		"POST /api/task-runs/:id/claim",
		"POST /api/task-runs/:id/complete",
		"POST /api/task-runs/:id/fail",
		"POST /api/task-runs/:id/start",
		"POST /api/tasks",
		"POST /api/tasks/:id/cancel",
		"POST /api/tasks/:id/children",
		"POST /api/tasks/:id/dependencies",
		"POST /api/tasks/:id/runs",
		"POST /api/workspaces",
		"POST /api/workspaces/resolve",
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
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "coder" || opts.Name != "demo" || opts.Workspace != "alpha" || opts.WorkspacePath != "" || opts.Channel != "builders" {
				t.Fatalf("Create() opts = %#v", opts)
			}
			sess := newSession("sess-123")
			sess.Channel = "builders"
			return sess, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder","name":"demo","workspace":"alpha","channel":"builders"}`))
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
	if response.Session.WorkspaceID != "ws-workspace" || response.Session.WorkspacePath != "/workspace" {
		t.Fatalf("session workspace = %#v", response.Session)
	}
	if response.Session.Channel != "builders" {
		t.Fatalf("session channel = %q, want %q", response.Session.Channel, "builders")
	}
}

func TestCreateSessionHandlerAllowsMissingAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "" {
				t.Fatalf("Create() AgentName = %q, want empty", opts.AgentName)
			}
			if opts.WorkspacePath == "" || opts.Workspace != "" {
				t.Fatalf("Create() workspace opts = %#v", opts)
			}
			return newSession("sess-default"), nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	recorder := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"name":"demo","workspace_path":"/workspace"}`))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
}

func TestListSessionsHandlerReturnsAllSessions(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
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

func TestListSessionsHandlerFiltersByWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	infoA := newSessionInfo("sess-a")
	infoB := newSessionInfo("sess-b")
	infoB.WorkspaceID = "ws-beta"
	infoB.Workspace = "/other"

	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{infoA, infoB}, nil
		},
	}
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("Get() ref = %q, want alpha", ref)
			}
			return workspacepkg.Workspace{ID: "ws-workspace", Name: "alpha"}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions?workspace=alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 1 || response.Sessions[0].ID != "sess-a" {
		t.Fatalf("sessions = %#v", response.Sessions)
	}
	if response.Sessions[0].WorkspaceID != "ws-workspace" {
		t.Fatalf("workspace_id = %q, want ws-workspace", response.Sessions[0].WorkspaceID)
	}
}

func TestHookEventsHandlerAvailableOnUDSRouter(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		QueryHookEventsFn: func(_ context.Context, filter hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error) {
			if filter.Family != hookspkg.HookEventFamilyTool {
				t.Fatalf("filter.Family = %q, want %q", filter.Family, hookspkg.HookEventFamilyTool)
			}
			if !filter.SyncOnly {
				t.Fatal("filter.SyncOnly = false, want true")
			}
			return []hookspkg.EventDescriptor{{
				Event:         hookspkg.HookToolPreCall,
				Family:        hookspkg.HookEventFamilyTool,
				SyncEligible:  true,
				PayloadSchema: "ToolPreCallPayload",
				PatchSchema:   "ToolCallPatch",
			}}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, observer, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/events?family=tool&sync_only=true", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Events []contract.HookEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if response.Events[0].Event != hookspkg.HookToolPreCall.String() {
		t.Fatalf("events[0].Event = %q, want %q", response.Events[0].Event, hookspkg.HookToolPreCall)
	}
}

func TestCreateWorkspaceHandlerRegistersWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	workspaces := stubWorkspaceService{
		RegisterFn: func(_ context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
			if opts.RootDir != rootDir || opts.Name != "alpha" || len(opts.AdditionalDirs) != 1 || opts.AdditionalDirs[0] != addDir || opts.DefaultAgent != "coder" {
				t.Fatalf("Register() opts = %#v", opts)
			}
			return workspacepkg.Workspace{
				ID:             "ws_alpha123",
				RootDir:        rootDir,
				AdditionalDirs: []string{addDir},
				Name:           "alpha",
				DefaultAgent:   "coder",
				CreatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	body := mustJSONBody(t, map[string]any{
		"root_dir":      rootDir,
		"name":          "alpha",
		"add_dirs":      []string{addDir},
		"default_agent": "coder",
	})
	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha123" {
		t.Fatalf("workspace.id = %q, want ws_alpha123", response.Workspace.ID)
	}
}

func TestListWorkspacesHandlerReturnsRows(t *testing.T) {
	homePaths := newTestHomePaths(t)
	workspaces := stubWorkspaceService{
		ListFn: func(context.Context) ([]workspacepkg.Workspace, error) {
			return []workspacepkg.Workspace{{
				ID:        "ws_alpha",
				RootDir:   "/workspace",
				Name:      "alpha",
				CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC),
			}}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspaces []workspacePayload `json:"workspaces"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Workspaces) != 1 || response.Workspaces[0].ID != "ws_alpha" {
		t.Fatalf("workspaces = %#v", response.Workspaces)
	}
}

func TestGetWorkspaceHandlerReturnsDetail(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	sharedSkillDir := filepath.Join(rootDir, ".agh", "skills", "review")
	resolved := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:        "ws_alpha",
			RootDir:   rootDir,
			Name:      "alpha",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		Agents: []aghconfig.AgentDef{{
			Name:     "coder",
			Provider: "fake",
			Prompt:   "hello",
		}},
		Skills: []workspacepkg.SkillPath{{
			Dir:    sharedSkillDir,
			Source: "workspace",
		}},
	}
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			info := newSessionInfo("sess-a")
			info.WorkspaceID = "ws_alpha"
			return []*session.SessionInfo{info}, nil
		},
	}
	workspaces := stubWorkspaceService{
		ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
			if ref != "ws_alpha" {
				t.Fatalf("Resolve() ref = %q, want ws_alpha", ref)
			}
			return resolved, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload        `json:"workspace"`
		Sessions  []sessionPayload        `json:"sessions"`
		Agents    []agentPayload          `json:"agents"`
		Skills    []workspaceSkillPayload `json:"skills"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" || len(response.Sessions) != 1 || len(response.Agents) != 1 || len(response.Skills) != 1 {
		t.Fatalf("workspace detail = %#v", response)
	}
	if response.Skills[0].Name != "review" {
		t.Fatalf("skill name = %q, want review", response.Skills[0].Name)
	}
}

func TestUpdateWorkspaceHandlerUpdatesWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	var updated bool
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if !updated {
				return workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "alpha"}, nil
			}
			return workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "beta", AdditionalDirs: []string{addDir}, DefaultAgent: "reviewer"}, nil
		},
		UpdateFn: func(_ context.Context, id string, opts workspacepkg.UpdateOptions) error {
			if id != "ws_alpha" || opts.Name == nil || *opts.Name != "beta" || opts.AdditionalDirs == nil || len(*opts.AdditionalDirs) != 1 || (*opts.AdditionalDirs)[0] != addDir || opts.DefaultAgent == nil || *opts.DefaultAgent != "reviewer" {
				t.Fatalf("Update() id=%q opts=%#v", id, opts)
			}
			updated = true
			return nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	body := mustJSONBody(t, map[string]any{
		"name":          "beta",
		"add_dirs":      []string{addDir},
		"default_agent": "reviewer",
	})
	recorder := performRequest(t, engine, http.MethodPatch, "/api/workspaces/ws_alpha", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.Name != "beta" || len(response.Workspace.AddDirs) != 1 {
		t.Fatalf("workspace = %#v", response.Workspace)
	}
}

func TestDeleteWorkspaceHandlerReturnsNoContent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	workspaces := stubWorkspaceService{
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{ID: "ws_alpha", Name: "alpha"}, nil
		},
		UnregisterFn: func(_ context.Context, id string) error {
			if id != "ws_alpha" {
				t.Fatalf("Unregister() id = %q, want ws_alpha", id)
			}
			return nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodDelete, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
}

func TestResolveWorkspaceHandlerReturnsWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	workspaces := stubWorkspaceService{
		ResolveOrRegisterFn: func(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
			if path != rootDir {
				t.Fatalf("ResolveOrRegister() path = %q, want %q", path, rootDir)
			}
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:        "ws_alpha",
					RootDir:   rootDir,
					Name:      "alpha",
					CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/workspaces/resolve",
		mustJSONBody(t, map[string]string{"path": rootDir}),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" {
		t.Fatalf("workspace.id = %q, want ws_alpha", response.Workspace.ID)
	}
}

func TestStopSessionHandlerReturnsStopped(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StopFn: func(_ context.Context, id string) error {
			if id != "sess-123" {
				t.Fatalf("Stop() id = %q, want sess-123", id)
			}
			return nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodDelete, "/api/sessions/sess-123", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}

func TestResumeSessionHandlerReturnsSession(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		ResumeFn: func(_ context.Context, id string) (*session.Session, error) {
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
		PromptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
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
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
		EventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
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
	if response.Events[0].WorkspaceID != "ws-workspace" || response.Events[0].WorkspacePath != "/workspace" {
		t.Fatalf("event workspace = %#v", response.Events[0])
	}
}

func TestSessionHistoryHandlerReturnsTurns(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
		HistoryFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
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
	if got := response.History[0].Events[0]; got.WorkspaceID != "ws-workspace" || got.WorkspacePath != "/workspace" {
		t.Fatalf("history event workspace = %#v", got)
	}
}

func TestSessionTranscriptHandlerReturnsMessages(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		TranscriptFn: func(context.Context, string) ([]transcript.Message, error) {
			return []transcript.Message{{
				ID:        "msg-1",
				Role:      transcript.RoleAssistant,
				Content:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions/sess-123/transcript", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Messages []transcript.Message `json:"messages"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(response.Messages))
	}
	if got := response.Messages[0].Content; got != "hello" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "hello")
	}
}

func TestStreamSessionHandlerUsesLastEventID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return newSessionInfo("sess-123"), nil
		},
		EventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
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
	var payload sessionEventPayload
	decodeSSEData(t, records[0], &payload)
	if payload.WorkspaceID != "ws-workspace" || payload.WorkspacePath != "/workspace" {
		t.Fatalf("stream payload workspace = %#v", payload)
	}
}

func TestStreamSessionHandlerSyntheticStoppedEventIncludesWorkspaceContext(t *testing.T) {
	homePaths := newTestHomePaths(t)
	stoppedAt := time.Date(2026, 4, 3, 12, 0, 5, 0, time.UTC)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
			info := newSessionInfo("sess-123")
			info.State = session.StateStopped
			info.UpdatedAt = stoppedAt
			return info, nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			return nil, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/sess-123/stream", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 || records[0].Event != session.EventTypeSessionStopped {
		t.Fatalf("records = %#v", records)
	}
	var payload sessionEventPayload
	decodeSSEData(t, records[0], &payload)
	if payload.WorkspaceID != "ws-workspace" || payload.WorkspacePath != "/workspace" {
		t.Fatalf("stopped payload workspace = %#v", payload)
	}
	if payload.Timestamp != stoppedAt {
		t.Fatalf("stopped payload timestamp = %v, want %v", payload.Timestamp, stoppedAt)
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
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
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

func TestExtensionStatusHandlerTrimsName(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{
		StatusFn: func(_ context.Context, name string) (contract.ExtensionPayload, error) {
			if name != "ext-a" {
				t.Fatalf("Status() name = %q, want %q", name, "ext-a")
			}
			return contract.ExtensionPayload{Name: name, Enabled: true, State: "active"}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/extensions/%20ext-a%20", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Extension contract.ExtensionPayload `json:"extension"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Extension.Name != "ext-a" {
		t.Fatalf("extension.name = %q, want %q", response.Extension.Name, "ext-a")
	}
}

func TestExtensionStatusHandlerRejectsBlankName(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/extensions/%20%20%20", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestHealthHandlerReturnsMetrics(t *testing.T) {
	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
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
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return []*session.SessionInfo{newSessionInfo("sess-1")}, nil
		},
	}
	observer := stubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
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
		StatusFn: func(context.Context, string) (*session.SessionInfo, error) {
			return nil, session.ErrSessionNotFound
		},
		CreateFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
			return nil, session.ErrMaxSessionsReached
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	getResp := performRequest(t, engine, http.MethodGet, "/api/sessions/missing", nil)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("GET /api/sessions/:id status = %d, want 404", getResp.Code)
	}

	postResp := performRequest(t, engine, http.MethodPost, "/api/sessions", []byte(`{"agent_name":"coder","workspace":"alpha"}`))
	if postResp.Code != http.StatusConflict {
		t.Fatalf("POST /api/sessions status = %d, want 409", postResp.Code)
	}
}

func TestObserveEventStreamUsesLastEventIDCursor(t *testing.T) {
	homePaths := newTestHomePaths(t)
	timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	observer := stubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{
				{ID: "sum-a", SessionID: "sess-1", Sequence: 1, Type: "agent_message", AgentName: "coder", Timestamp: timestamp},
				{ID: "sum-b", SessionID: "sess-1", Sequence: 2, Type: "done", AgentName: "coder", Timestamp: timestamp},
			}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	done := make(chan struct{})
	close(done)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequest(http.MethodGet, "/api/observe/events/stream", nil)
	req.Header.Set("Last-Event-ID", timestamp.Format(time.RFC3339Nano)+"|00000000000000000001")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) == 0 {
		t.Fatalf("expected at least one SSE record, got body=%s", recorder.Body.String())
	}
	if records[0].ID != timestamp.Format(time.RFC3339Nano)+"|00000000000000000002" {
		t.Fatalf("record id = %q, want %q", records[0].ID, timestamp.Format(time.RFC3339Nano)+"|00000000000000000002")
	}
}
