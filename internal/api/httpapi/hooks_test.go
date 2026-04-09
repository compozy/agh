package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestHookCatalogHandlerReturnsResolvedHooksAndWorkspaceFilter(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		QueryHookCatalogFn: func(_ context.Context, filter hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error) {
			if filter.WorkspaceID != "ws-alpha" {
				t.Fatalf("filter.WorkspaceID = %q, want ws-alpha", filter.WorkspaceID)
			}
			if filter.WorkspaceRoot != "/workspace/alpha" {
				t.Fatalf("filter.WorkspaceRoot = %q, want /workspace/alpha", filter.WorkspaceRoot)
			}
			if filter.AgentName != "coder" {
				t.Fatalf("filter.AgentName = %q, want coder", filter.AgentName)
			}
			return []hookspkg.CatalogEntry{
				{
					Order:    1,
					Name:     "native-first",
					Event:    hookspkg.HookSessionPostCreate,
					Source:   hookspkg.HookSourceNative,
					Mode:     hookspkg.HookModeSync,
					Priority: 1000,
				},
				{
					Order:    2,
					Name:     "config-second",
					Event:    hookspkg.HookSessionPostCreate,
					Source:   hookspkg.HookSourceConfig,
					Mode:     hookspkg.HookModeSync,
					Priority: 0,
				},
			}, nil
		},
	}
	workspaces := stubWorkspaceService{
		ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
			if ref != "alpha" {
				t.Fatalf("Resolve() ref = %q, want alpha", ref)
			}
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:      "ws-alpha",
					RootDir: "/workspace/alpha",
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, stubSessionManager{}, observer, workspaces, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/catalog?workspace=alpha&agent=coder", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Hooks []contract.HookCatalogPayload `json:"hooks"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Hooks), 2; got != want {
		t.Fatalf("len(hooks) = %d, want %d", got, want)
	}
	if response.Hooks[0].Name != "native-first" || response.Hooks[0].Order != 1 || response.Hooks[0].Source != "native" {
		t.Fatalf("hooks[0] = %#v", response.Hooks[0])
	}
	if response.Hooks[1].Name != "config-second" || response.Hooks[1].Order != 2 || response.Hooks[1].Source != "config" {
		t.Fatalf("hooks[1] = %#v", response.Hooks[1])
	}
}

func TestHookRunsHandlerReturnsExecutionHistoryWithPatchDiffs(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			if id != "sess-hook" {
				t.Fatalf("Status() id = %q, want sess-hook", id)
			}
			return newSessionInfo(id), nil
		},
	}
	observer := stubObserver{
		QueryHookRunsFn: func(_ context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
			if query.SessionID != "sess-hook" {
				t.Fatalf("query.SessionID = %q, want sess-hook", query.SessionID)
			}
			if query.Event != hookspkg.HookPermissionRequest.String() {
				t.Fatalf("query.Event = %q, want %q", query.Event, hookspkg.HookPermissionRequest)
			}
			return []hookspkg.HookRunRecord{
				{
					HookName:      "permission-audit",
					Event:         hookspkg.HookPermissionRequest,
					Source:        hookspkg.HookSourceConfig,
					Mode:          hookspkg.HookModeSync,
					Duration:      25 * time.Millisecond,
					Outcome:       hookspkg.HookRunOutcomeDenied,
					DispatchDepth: 2,
					PatchApplied:  []byte(`{"decision":"deny","reason":"policy"}`),
					Error:         "denied by policy",
					Required:      true,
					RecordedAt:    time.Date(2026, 4, 9, 18, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlers(t, manager, observer, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/runs?session=sess-hook&event=permission.request", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Runs []contract.HookRunPayload `json:"runs"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Runs), 1; got != want {
		t.Fatalf("len(runs) = %d, want %d", got, want)
	}
	if response.Runs[0].HookName != "permission-audit" {
		t.Fatalf("runs[0].HookName = %q, want permission-audit", response.Runs[0].HookName)
	}
	if string(response.Runs[0].PatchApplied) != `{"decision":"deny","reason":"policy"}` {
		t.Fatalf("runs[0].PatchApplied = %s, want deny patch", response.Runs[0].PatchApplied)
	}
	if response.Runs[0].DispatchDepth != 2 || response.Runs[0].Outcome != string(hookspkg.HookRunOutcomeDenied) {
		t.Fatalf("runs[0] = %#v", response.Runs[0])
	}
}

func TestHookRunsHandlerRejectsMissingSession(t *testing.T) {
	t.Parallel()

	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, newTestHomePaths(t)))
	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/runs", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestHookRunsHandlerRejectsInvalidEvent(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(_ context.Context, id string) (*session.SessionInfo, error) {
			return newSessionInfo(id), nil
		},
	}

	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/runs?session=sess-hook&event=not-a-hook", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestHookEventsHandlerReturnsPayloads(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		QueryHookEventsFn: func(_ context.Context) ([]hookspkg.EventDescriptor, error) {
			return []hookspkg.EventDescriptor{
				{
					Event:         hookspkg.HookMessageDelta,
					Family:        hookspkg.HookEventFamilyMessage,
					SyncEligible:  false,
					PayloadSchema: "MessageDeltaPayload",
					PatchSchema:   "MessageDeltaPatch",
				},
				{
					Event:         hookspkg.HookPermissionRequest,
					Family:        hookspkg.HookEventFamilyPermission,
					SyncEligible:  true,
					PayloadSchema: "PermissionRequestPayload",
					PatchSchema:   "PermissionRequestPatch",
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, observer, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/events", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Events []contract.HookEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Events), 2; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if response.Events[0].Event != hookspkg.HookMessageDelta.String() || response.Events[0].SyncEligible {
		t.Fatalf("events[0] = %#v", response.Events[0])
	}
	if response.Events[1].Event != hookspkg.HookPermissionRequest.String() || !response.Events[1].SyncEligible {
		t.Fatalf("events[1] = %#v", response.Events[1])
	}
}
