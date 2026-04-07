package core_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
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

	store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
	if err := store.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs() error = %v", err)
	}
	workspace := t.TempDir()
	if err := store.Write(memory.ScopeGlobal, "global.md", []byte(memoryDocument(t, "Global", memory.MemoryTypeUser, "hello"))); err != nil {
		t.Fatalf("Write(global) error = %v", err)
	}
	if err := store.ForWorkspace(workspace).Write(memory.ScopeWorkspace, "workspace.md", []byte(memoryDocument(t, "Workspace", memory.MemoryTypeProject, "world"))); err != nil {
		t.Fatalf("Write(workspace) error = %v", err)
	}

	trigger := &stubDreamTrigger{
		EnabledFn: true,
		Triggered: true,
		Reason:    "queued",
		Last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}
	manager := testutil.StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			info := testutil.NewSessionInfo("sess-a")
			info.Workspace = workspace
			return []*session.SessionInfo{info}, nil
		},
	}
	observer := testutil.StubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok", ActiveSessions: 1}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, observer, testutil.StubWorkspaceService{}, store, trigger)

	listResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory?workspace="+workspace, nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list memory status = %d, want %d", listResp.Code, http.StatusOK)
	}

	readResp := performRequest(t, fixture.Engine, http.MethodGet, "/memory/global.md?scope=global", nil)
	if readResp.Code != http.StatusOK {
		t.Fatalf("read memory status = %d, want %d", readResp.Code, http.StatusOK)
	}

	writeBody := []byte(`{"scope":"workspace","workspace":"` + workspace + `","content":"` + escapeJSON(memoryDocument(t, "Project", memory.MemoryTypeProject, "updated")) + `"}`)
	writeResp := performRequest(t, fixture.Engine, http.MethodPut, "/memory/new.md", writeBody)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("write memory status = %d, want %d; body=%s", writeResp.Code, http.StatusOK, writeResp.Body.String())
	}

	deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/memory/new.md?scope=workspace&workspace="+workspace, nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete memory status = %d, want %d", deleteResp.Code, http.StatusOK)
	}

	consolidateResp := performRequest(t, fixture.Engine, http.MethodPost, "/memory/consolidate", []byte(`{"workspace":"`+workspace+`"}`))
	if consolidateResp.Code != http.StatusOK || trigger.Calls != 1 || trigger.Workspace != workspace {
		t.Fatalf("consolidate status=%d calls=%d workspace=%q", consolidateResp.Code, trigger.Calls, trigger.Workspace)
	}

	healthResp := performRequest(t, fixture.Engine, http.MethodGet, "/observe/health?workspace="+workspace, nil)
	if healthResp.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", healthResp.Code, http.StatusOK)
	}

	if status := core.StatusForMemoryError(core.NewMemoryValidationError(errors.New("bad"))); status != http.StatusBadRequest {
		t.Fatalf("StatusForMemoryError(validation) = %d, want %d", status, http.StatusBadRequest)
	}
}

func TestWorkspaceHandlersDelegateToService(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	addDir := t.TempDir()
	workspace := workspacepkg.Workspace{
		ID:             "ws_alpha",
		RootDir:        rootDir,
		AdditionalDirs: []string{addDir},
		Name:           "alpha",
		DefaultAgent:   "coder",
		CreatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 3, 12, 1, 0, 0, time.UTC),
	}
	resolved := workspacepkg.ResolvedWorkspace{
		Workspace: workspace,
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
			if opts.RootDir != rootDir || len(opts.AdditionalDirs) != 1 || opts.DefaultAgent != "coder" {
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
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			info := testutil.NewSessionInfo("sess-a")
			info.WorkspaceID = workspace.ID
			return []*session.SessionInfo{info}, nil
		},
	}
	fixture := newHandlerFixture(t, manager, testutil.StubObserver{}, workspaces, nil, nil)

	createBody := []byte(`{"root_dir":"` + rootDir + `","add_dirs":["` + addDir + `"],"name":"alpha","default_agent":"coder"}`)
	createResp := performRequest(t, fixture.Engine, http.MethodPost, "/workspaces", createBody)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create workspace status = %d, want %d", createResp.Code, http.StatusCreated)
	}

	listResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list workspaces status = %d, want %d", listResp.Code, http.StatusOK)
	}

	getResp := performRequest(t, fixture.Engine, http.MethodGet, "/workspaces/"+workspace.ID, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get workspace status = %d, want %d", getResp.Code, http.StatusOK)
	}
	var getPayload struct {
		Sessions []contract.SessionPayload `json:"sessions"`
	}
	testutil.DecodeJSONResponse(t, getResp, &getPayload)
	if len(getPayload.Sessions) != 1 || getPayload.Sessions[0].WorkspaceID != workspace.ID {
		t.Fatalf("sessions payload = %#v", getPayload.Sessions)
	}

	updateResp := performRequest(t, fixture.Engine, http.MethodPatch, "/workspaces/"+workspace.ID, []byte(`{"name":"beta"}`))
	if updateResp.Code != http.StatusOK || !updateCalled {
		t.Fatalf("update status=%d called=%v", updateResp.Code, updateCalled)
	}

	deleteResp := performRequest(t, fixture.Engine, http.MethodDelete, "/workspaces/"+workspace.ID, nil)
	if deleteResp.Code != http.StatusNoContent || !deleteCalled {
		t.Fatalf("delete status=%d called=%v", deleteResp.Code, deleteCalled)
	}

	resolveResp := performRequest(t, fixture.Engine, http.MethodPost, "/workspaces/resolve", []byte(`{"path":"`+rootDir+`"}`))
	if resolveResp.Code != http.StatusOK || !resolveCalled {
		t.Fatalf("resolve status=%d called=%v", resolveResp.Code, resolveCalled)
	}
}

func TestWorkspaceUpdateSupportsAddDirsAndDefaultAgent(t *testing.T) {
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

	resp := performRequest(t, fixture.Engine, http.MethodPatch, "/workspaces/ws_alpha", []byte(`{"add_dirs":["`+addDir+`"],"default_agent":"coder"}`))
	if resp.Code != http.StatusOK {
		t.Fatalf("update add_dirs/default_agent status = %d, want %d", resp.Code, http.StatusOK)
	}
	if captured.AdditionalDirs == nil || len(*captured.AdditionalDirs) != 1 || (*captured.AdditionalDirs)[0] != addDir {
		t.Fatalf("captured add dirs = %#v", captured.AdditionalDirs)
	}
	if captured.DefaultAgent == nil || *captured.DefaultAgent != "coder" {
		t.Fatalf("captured default agent = %#v", captured.DefaultAgent)
	}
}

func memoryDocument(t *testing.T, name string, typ memory.MemoryType, body string) string {
	t.Helper()

	header := memory.MemoryHeader{
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
	payload, _ := json.Marshal(value)
	return string(payload[1 : len(payload)-1])
}
