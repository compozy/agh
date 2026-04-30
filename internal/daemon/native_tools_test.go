package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"testing"
	"time"

	apitest "github.com/pedronauck/agh/internal/api/testutil"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	memorypkg "github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	builtintools "github.com/pedronauck/agh/internal/tools/builtin"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestDaemonNativeTools(t *testing.T) {
	t.Parallel()

	t.Run("Should dispatch skill catalog tools through the real skill registry", func(t *testing.T) {
		t.Parallel()

		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Skills: newLoadedNativeSkillRegistry(t),
		}, nativeApproveAllPolicyInputs())

		listResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDSkillList},
		)
		if err != nil {
			t.Fatalf("Registry.Call(skill_list) error = %v", err)
		}
		requireNativeStructuredContains(t, listResult, []byte(`"agh-memory-guide"`))

		searchResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDSkillSearch,
				Input:  json.RawMessage(`{"query":"network"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(skill_search) error = %v", err)
		}
		requireNativeStructuredContains(t, searchResult, []byte(`"agh-network"`))

		viewResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDSkillView,
				Input:  json.RawMessage(`{"name":"agh-memory-guide"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(skill_view) error = %v", err)
		}
		requireNativeStructuredContains(t, viewResult, []byte(`# AGH Memory Guide`))
		if len(viewResult.Content) != 1 ||
			!bytes.Contains([]byte(viewResult.Content[0].Text), []byte(`# AGH Memory Guide`)) {
			t.Fatalf("skill_view content = %#v, want real skill body", viewResult.Content)
		}
	})

	t.Run("Should expose bootstrap diagnostics and exclude non-MVP lifecycle tools", func(t *testing.T) {
		t.Parallel()

		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Skills:  newLoadedNativeSkillRegistry(t),
			Network: &nativeNetworkStub{},
			Tasks:   &nativeTaskManager{},
		}, nativeApproveAllPolicyInputs())

		listResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{Operator: true},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDToolList},
		)
		if err != nil {
			t.Fatalf("Registry.Call(tool_list) error = %v", err)
		}
		requireNativeStructuredContains(t, listResult, []byte(`"agh__task_child_create"`))
		requireNativeStructuredExcludes(t, listResult, []byte(`"agh__task_claim"`))
		requireNativeStructuredExcludes(t, listResult, []byte(`"agh__skill_install"`))

		searchResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{Operator: true},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDToolSearch,
				Input:  json.RawMessage(`{"query":"child"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(tool_search) error = %v", err)
		}
		requireNativeStructuredContains(t, searchResult, []byte(`"agh__task_child_create"`))

		infoResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{Operator: true},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDToolInfo,
				Input:  json.RawMessage(`{"tool_id":"agh__network_send"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(tool_info) error = %v", err)
		}
		requireNativeStructuredContains(t, infoResult, []byte(`"open_world"`))

		_, err = registry.Get(t.Context(), toolspkg.Scope{Operator: true}, "agh__task_claim")
		if !errors.Is(err, toolspkg.ErrToolNotFound) {
			t.Fatalf("Registry.Get(excluded task claim) error = %v, want ErrToolNotFound", err)
		}
	})

	t.Run("Should keep unavailable read surfaces operator-only with deterministic reasons", func(t *testing.T) {
		t.Parallel()

		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{}, nativeApproveAllPolicyInputs())

		operatorViews, err := registry.List(t.Context(), toolspkg.Scope{Operator: true})
		if err != nil {
			t.Fatalf("Registry.List(operator) error = %v", err)
		}
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDNetworkStatus)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDSessionList)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDWorkspaceDescribe)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDMemoryList)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDObserveEvents)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDBridgesList)

		sessionViews, err := registry.List(t.Context(), toolspkg.Scope{SessionID: "sess-1"})
		if err != nil {
			t.Fatalf("Registry.List(session) error = %v", err)
		}
		for _, id := range []toolspkg.ToolID{
			toolspkg.ToolIDNetworkStatus,
			toolspkg.ToolIDSessionList,
			toolspkg.ToolIDWorkspaceDescribe,
			toolspkg.ToolIDMemoryList,
			toolspkg.ToolIDObserveEvents,
			toolspkg.ToolIDBridgesList,
		} {
			if nativeToolViewByID(sessionViews, id) != nil {
				t.Fatalf("session projection leaked unavailable tool %s", id)
			}
		}
	})

	t.Run("Should mark workspace describe unavailable without hiding lighter workspace reads", func(t *testing.T) {
		t.Parallel()

		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Workspaces: apitest.StubWorkspaceService{},
		}, nativeApproveAllPolicyInputs())

		operatorViews, err := registry.List(t.Context(), toolspkg.Scope{Operator: true})
		if err != nil {
			t.Fatalf("Registry.List(operator) error = %v", err)
		}
		requireNativeToolAvailable(t, operatorViews, toolspkg.ToolIDWorkspaceList)
		requireNativeToolAvailable(t, operatorViews, toolspkg.ToolIDWorkspaceInfo)
		requireNativeToolUnavailableReason(t, operatorViews, toolspkg.ToolIDWorkspaceDescribe)

		sessionViews, err := registry.List(t.Context(), toolspkg.Scope{SessionID: "sess-1"})
		if err != nil {
			t.Fatalf("Registry.List(session) error = %v", err)
		}
		requireNativeViewContains(t, sessionViews, toolspkg.ToolIDWorkspaceList)
		requireNativeViewContains(t, sessionViews, toolspkg.ToolIDWorkspaceInfo)
		requireNativeViewExcludes(t, sessionViews, toolspkg.ToolIDWorkspaceDescribe)
	})

	t.Run("Should reject schema-invalid task input before service calls", func(t *testing.T) {
		t.Parallel()

		tasks := &nativeTaskManager{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Tasks: tasks,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskCreate,
				Input:  json.RawMessage(`{"scope":"global","title":"root","parent_task_id":"not-allowed"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolInvalidInput) {
			t.Fatalf("Registry.Call(task_create invalid input) error = %v, want ErrToolInvalidInput", err)
		}
		if tasks.createCalls != 0 {
			t.Fatalf("CreateTask calls = %d, want 0", tasks.createCalls)
		}
	})

	t.Run("Should reject schema-invalid input for every native built-in before service calls", func(t *testing.T) {
		t.Parallel()

		tasks := &nativeTaskManager{}
		networkService := &nativeNetworkStub{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Skills:  newLoadedNativeSkillRegistry(t),
			Network: networkService,
			Tasks:   tasks,
		}, nativeApproveAllPolicyInputs())

		cases := []struct {
			id    toolspkg.ToolID
			input json.RawMessage
		}{
			{toolspkg.ToolIDToolList, json.RawMessage(`{"limit":"bad"}`)},
			{toolspkg.ToolIDToolSearch, json.RawMessage(`{"query":7}`)},
			{toolspkg.ToolIDToolInfo, json.RawMessage(`{"tool_id":7}`)},
			{toolspkg.ToolIDSkillList, json.RawMessage(`{"limit":"bad"}`)},
			{toolspkg.ToolIDSkillSearch, json.RawMessage(`{"query":7}`)},
			{toolspkg.ToolIDSkillView, json.RawMessage(`{"name":7}`)},
			{toolspkg.ToolIDNetworkPeers, json.RawMessage(`{"channel":7}`)},
			{toolspkg.ToolIDNetworkSend, json.RawMessage(`{"channel":"default","kind":"say","body":"bad"}`)},
			{toolspkg.ToolIDTaskList, json.RawMessage(`{"limit":"bad"}`)},
			{toolspkg.ToolIDTaskRead, json.RawMessage(`{"task_id":7}`)},
			{toolspkg.ToolIDTaskCreate, json.RawMessage(`{"scope":"global","title":"root","parent_task_id":"nope"}`)},
			{toolspkg.ToolIDTaskChildCreate, json.RawMessage(`{"parent_task_id":"parent","scope":"global","title":7}`)},
			{toolspkg.ToolIDTaskUpdate, json.RawMessage(`{"task_id":"task","clear_owner":"no"}`)},
			{toolspkg.ToolIDTaskCancel, json.RawMessage(`{"task_id":7}`)},
			{toolspkg.ToolIDTaskRunList, json.RawMessage(`{"task_id":"task","limit":"bad"}`)},
		}

		for _, tc := range cases {
			t.Run(tc.id.String(), func(t *testing.T) {
				_, err := registry.Call(
					t.Context(),
					toolspkg.Scope{Operator: true},
					toolspkg.CallRequest{ToolID: tc.id, Input: tc.input},
				)
				if !errors.Is(err, toolspkg.ErrToolInvalidInput) {
					t.Fatalf("Registry.Call(%s) error = %v, want ErrToolInvalidInput", tc.id, err)
				}
			})
		}
		if got := tasks.totalCalls(); got != 0 {
			t.Fatalf("task manager calls = %d, want 0", got)
		}
		if got := networkService.totalCalls(); got != 0 {
			t.Fatalf("network calls = %d, want 0", got)
		}
	})

	t.Run("Should require approval for mutating tools under approve-reads policy", func(t *testing.T) {
		t.Parallel()

		tasks := &nativeTaskManager{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Tasks: tasks,
		}, toolspkg.PolicyInputs{
			SystemPermissionMode: toolspkg.PermissionModeApproveReads,
			ApprovalAvailable:    false,
		})

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskCreate,
				Input:  json.RawMessage(`{"scope":"global","title":"root"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolApprovalRequired) {
			t.Fatalf("Registry.Call(task_create approve-reads) error = %v, want ErrToolApprovalRequired", err)
		}
		if tasks.createCalls != 0 {
			t.Fatalf("CreateTask calls = %d, want 0", tasks.createCalls)
		}
	})

	t.Run("Should mutate allowed config paths and reject guarded config paths", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			HomePaths: homePaths,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDConfigSet,
				Input:  json.RawMessage(`{"path":"defaults.agent","value":"planner"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(config_set allowed) error = %v", err)
		}
		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		if cfg.Defaults.Agent != "planner" {
			t.Fatalf("Defaults.Agent = %q, want planner", cfg.Defaults.Agent)
		}

		result, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDConfigGet,
				Input:  json.RawMessage(`{"path":"defaults.agent"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(config_get) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte(`"planner"`))

		cases := []struct {
			path   string
			reason toolspkg.ReasonCode
		}{
			{path: "daemon.socket", reason: toolspkg.ReasonConfigTrustRootForbidden},
			{path: "http.port", reason: toolspkg.ReasonConfigTrustRootForbidden},
			{path: "providers.claude.api_key_env", reason: toolspkg.ReasonConfigSecretPathForbidden},
			{path: "mcp_servers[0].env.TOKEN", reason: toolspkg.ReasonConfigSecretPathForbidden},
			{path: "sandboxes.default.runtime_root", reason: toolspkg.ReasonConfigTrustRootForbidden},
		}
		for _, tc := range cases {
			t.Run(tc.path, func(t *testing.T) {
				_, err := registry.Call(
					t.Context(),
					toolspkg.Scope{},
					toolspkg.CallRequest{
						ToolID: toolspkg.ToolIDConfigSet,
						Input: json.RawMessage(
							fmt.Sprintf(`{"path":%q,"value":"blocked"}`, tc.path),
						),
					},
				)
				requireToolReason(t, err, toolspkg.ErrToolDenied, tc.reason)
			})
		}
	})

	t.Run("Should require approval before config writes reach persistence", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			HomePaths: homePaths,
		}, toolspkg.PolicyInputs{
			SystemPermissionMode: toolspkg.PermissionModeApproveReads,
			ApprovalAvailable:    false,
		})

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDConfigSet,
				Input:  json.RawMessage(`{"path":"defaults.agent","value":"planner"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolApprovalRequired) {
			t.Fatalf("Registry.Call(config_set approve-reads) error = %v, want ErrToolApprovalRequired", err)
		}
		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		if cfg.Defaults.Agent == "planner" {
			t.Fatal("Defaults.Agent was mutated before approval")
		}
	})

	t.Run("Should manage config backed hooks through normalized binding sync", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		observer := &nativeObserverStub{}
		bindings := &nativeHookBindingsStub{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			HomePaths:    homePaths,
			Observer:     observer,
			HookBindings: bindings,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksCreate,
				Input: json.RawMessage(
					`{"name":"tool-audit","event":"tool.pre_call","command":"/bin/echo","args":["audit"]}`,
				),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(hooks_create) error = %v", err)
		}
		if bindings.syncCalls != 1 {
			t.Fatalf("HookBindings.Sync calls = %d, want 1", bindings.syncCalls)
		}
		target, err := aghconfig.ResolveConfigWriteTarget(homePaths, "", aghconfig.WriteScopeGlobal)
		if err != nil {
			t.Fatalf("ResolveConfigWriteTarget() error = %v", err)
		}
		decls, err := aghconfig.OverlayHookDeclarations(target)
		if err != nil {
			t.Fatalf("OverlayHookDeclarations() error = %v", err)
		}
		if len(decls) != 1 || decls[0].Name != "tool-audit" || decls[0].Command != "/bin/echo" {
			t.Fatalf("OverlayHookDeclarations() = %#v, want created hook", decls)
		}

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksDisable,
				Input:  json.RawMessage(`{"name":"tool-audit"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(hooks_disable) error = %v", err)
		}
		cfg, err := aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome() error = %v", err)
		}
		active, err := aghconfig.HookDeclarations(cfg.Hooks, nil)
		if err != nil {
			t.Fatalf("HookDeclarations(disabled) error = %v", err)
		}
		if len(active) != 0 {
			t.Fatalf("HookDeclarations(disabled) len = %d, want 0", len(active))
		}

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksEnable,
				Input:  json.RawMessage(`{"name":"tool-audit"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(hooks_enable) error = %v", err)
		}
		cfg, err = aghconfig.LoadForHome(homePaths)
		if err != nil {
			t.Fatalf("LoadForHome(enabled) error = %v", err)
		}
		active, err = aghconfig.HookDeclarations(cfg.Hooks, nil)
		if err != nil {
			t.Fatalf("HookDeclarations(enabled) error = %v", err)
		}
		if len(active) != 1 || active[0].Name != "tool-audit" {
			t.Fatalf("HookDeclarations(enabled) = %#v, want active tool-audit", active)
		}

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksUpdate,
				Input:  json.RawMessage(`{"name":"tool-audit","command":"/usr/bin/env","args":["true"]}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(hooks_update) error = %v", err)
		}
		decls, err = aghconfig.OverlayHookDeclarations(target)
		if err != nil {
			t.Fatalf("OverlayHookDeclarations(updated) error = %v", err)
		}
		if len(decls) != 1 || decls[0].Command != "/usr/bin/env" || len(decls[0].Args) != 1 {
			t.Fatalf("OverlayHookDeclarations(updated) = %#v, want updated command", decls)
		}

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksDelete,
				Input:  json.RawMessage(`{"name":"tool-audit"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(hooks_delete) error = %v", err)
		}
		decls, err = aghconfig.OverlayHookDeclarations(target)
		if err != nil {
			t.Fatalf("OverlayHookDeclarations(deleted) error = %v", err)
		}
		if len(decls) != 0 {
			t.Fatalf("OverlayHookDeclarations(deleted) = %#v, want empty", decls)
		}
		if bindings.syncCalls != 5 {
			t.Fatalf("HookBindings.Sync calls = %d, want 5", bindings.syncCalls)
		}
	})

	t.Run("Should reject immutable hook sources and secret hook executor inputs", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		bindings := &nativeHookBindingsStub{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			HomePaths: homePaths,
			Observer: &nativeObserverStub{
				catalog: []hookspkg.CatalogEntry{{
					Name:   "native-session",
					Event:  hookspkg.HookSessionPostCreate,
					Source: hookspkg.HookSourceNative,
					Mode:   hookspkg.HookModeAsync,
				}},
			},
			HookBindings: bindings,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksUpdate,
				Input:  json.RawMessage(`{"name":"native-session","command":"/bin/echo"}`),
			},
		)
		requireToolReason(t, err, toolspkg.ErrToolDenied, toolspkg.ReasonHookSourceImmutable)

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksCreate,
				Input: json.RawMessage(
					`{"name":"secret-hook","event":"tool.pre_call","command":"/bin/echo","env":{"API_KEY":"secret"}}`,
				),
			},
		)
		requireToolReason(t, err, toolspkg.ErrToolDenied, toolspkg.ReasonHookSecretInputForbidden)
		if bindings.syncCalls != 0 {
			t.Fatalf("HookBindings.Sync calls = %d, want 0 after denied hooks", bindings.syncCalls)
		}
	})

	t.Run("Should require approval before hook mutations reach config writer", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		bindings := &nativeHookBindingsStub{}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			HomePaths:    homePaths,
			Observer:     &nativeObserverStub{},
			HookBindings: bindings,
		}, toolspkg.PolicyInputs{
			SystemPermissionMode: toolspkg.PermissionModeApproveReads,
			ApprovalAvailable:    false,
		})

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDHooksCreate,
				Input:  json.RawMessage(`{"name":"blocked","event":"tool.pre_call","command":"/bin/echo"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolApprovalRequired) {
			t.Fatalf("Registry.Call(hooks_create approve-reads) error = %v, want ErrToolApprovalRequired", err)
		}
		if bindings.syncCalls != 0 {
			t.Fatalf("HookBindings.Sync calls = %d, want 0 before approval", bindings.syncCalls)
		}
	})

	t.Run("Should route bounded task tools through task service boundaries", func(t *testing.T) {
		t.Parallel()

		tasks := &nativeTaskManager{
			listSummaries: []taskpkg.Summary{{
				ID:     "task-listed",
				Title:  "Listed task",
				Status: taskpkg.TaskStatusPending,
				Scope:  taskpkg.ScopeWorkspace,
			}},
			runs: []taskpkg.Run{{
				ID:     "run-listed",
				TaskID: "task-run",
				Status: taskpkg.TaskRunStatusQueued,
			}},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Tasks: tasks,
		}, nativeApproveAllPolicyInputs())
		scope := toolspkg.Scope{SessionID: "sess-actor", WorkspaceID: "ws-1"}

		_, err := registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskList,
				Input:  json.RawMessage(`{"scope":"workspace","status":"pending","limit":3}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_list) error = %v", err)
		}
		if tasks.listCalls != 1 ||
			tasks.lastQuery.WorkspaceID != "ws-1" ||
			tasks.lastQuery.Status != taskpkg.TaskStatusPending {
			t.Fatalf("ListTasks calls/query = %d/%#v, want workspace pending query", tasks.listCalls, tasks.lastQuery)
		}

		_, err = registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskRead,
				Input:  json.RawMessage(`{"task_id":"task-read"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_read) error = %v", err)
		}
		if tasks.getCalls != 1 || tasks.lastGetID != "task-read" {
			t.Fatalf("GetTask calls/id = %d/%q, want task-read", tasks.getCalls, tasks.lastGetID)
		}

		_, err = registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskCreate,
				Input:  json.RawMessage(`{"scope":"global","title":"root task"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_create) error = %v", err)
		}
		if tasks.createCalls != 1 || tasks.lastCreateSpec.WorkspaceID != "" {
			t.Fatalf(
				"CreateTask calls/spec = %d/%#v, want global task without caller workspace fallback",
				tasks.createCalls,
				tasks.lastCreateSpec,
			)
		}

		_, err = registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskUpdate,
				Input:  json.RawMessage(`{"task_id":"task-update","title":"Updated task","clear_owner":true}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_update) error = %v", err)
		}
		if tasks.updateCalls != 1 ||
			tasks.lastUpdateID != "task-update" ||
			tasks.lastPatch.Title == nil ||
			*tasks.lastPatch.Title != "Updated task" ||
			!tasks.lastPatch.ClearOwner {
			t.Fatalf(
				"UpdateTask calls/patch = %d/%q/%#v, want title patch",
				tasks.updateCalls,
				tasks.lastUpdateID,
				tasks.lastPatch,
			)
		}

		_, err = registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskCancel,
				Input:  json.RawMessage(`{"task_id":"task-cancel","reason":"operator canceled"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_cancel) error = %v", err)
		}
		if tasks.cancelCalls != 1 ||
			tasks.lastCancelID != "task-cancel" ||
			tasks.lastCancel.Reason != "operator canceled" {
			t.Fatalf(
				"CancelTask calls/request = %d/%q/%#v, want cancellation request",
				tasks.cancelCalls,
				tasks.lastCancelID,
				tasks.lastCancel,
			)
		}

		_, err = registry.Call(
			t.Context(),
			scope,
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskRunList,
				Input:  json.RawMessage(`{"task_id":"task-run","status":"queued","limit":2}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(task_run_list) error = %v", err)
		}
		if tasks.runListCalls != 1 ||
			tasks.lastRunListTaskID != "task-run" ||
			tasks.lastRunQuery.Status != taskpkg.TaskRunStatusQueued {
			t.Fatalf(
				"ListTaskRuns calls/query = %d/%q/%#v, want queued run query",
				tasks.runListCalls,
				tasks.lastRunListTaskID,
				tasks.lastRunQuery,
			)
		}
	})

	t.Run("Should route child creation through task child-lineage service boundary", func(t *testing.T) {
		t.Parallel()

		tasks := &nativeTaskManager{
			childErr: fmt.Errorf("%w: child parent task id is required", taskpkg.ErrValidation),
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Tasks: tasks,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{WorkspaceID: "ws-1"},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDTaskChildCreate,
				Input: json.RawMessage(
					`{"parent_task_id":"parent-1","scope":"workspace","title":"child"}`,
				),
			},
		)
		if !errors.Is(err, taskpkg.ErrValidation) || !errors.Is(err, toolspkg.ErrToolBackendFailed) {
			t.Fatalf("Registry.Call(task_child_create) error = %v, want wrapped task validation", err)
		}
		if tasks.createCalls != 0 {
			t.Fatalf("CreateTask calls = %d, want 0", tasks.createCalls)
		}
		if tasks.childCreateCalls != 1 {
			t.Fatalf("CreateChildTask calls = %d, want 1", tasks.childCreateCalls)
		}
		if tasks.childParentID != "parent-1" {
			t.Fatalf("child parent id = %q, want parent-1", tasks.childParentID)
		}
		if tasks.childSpec.WorkspaceID != "ws-1" {
			t.Fatalf("child workspace_id = %q, want caller workspace fallback", tasks.childSpec.WorkspaceID)
		}
	})

	t.Run("Should list network peers through the existing network service boundary", func(t *testing.T) {
		t.Parallel()

		networkService := &nativeNetworkStub{
			peers: []network.PeerInfo{{PeerID: "peer-1"}},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Network: networkService,
		}, nativeApproveAllPolicyInputs())

		result, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDNetworkPeers,
				Input:  json.RawMessage(`{"channel":"default"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(network_peers) error = %v", err)
		}
		requireNativeStructuredContains(t, result, []byte(`"peer-1"`))
		if networkService.peersCalls != 1 || networkService.peersChannel != "default" {
			t.Fatalf(
				"ListPeers calls/channel = %d/%q, want default channel",
				networkService.peersCalls,
				networkService.peersChannel,
			)
		}
	})

	t.Run("Should read network inspection tools through the existing network service boundary", func(t *testing.T) {
		t.Parallel()

		networkService := &nativeNetworkStub{
			status:   &network.Status{Enabled: true, Status: network.StatusRunning, LocalPeers: 2},
			channels: []network.ChannelInfo{{Channel: "builders", PeerCount: 2}},
			inbox: []network.Envelope{{
				ID:      "msg-1",
				Kind:    network.KindSay,
				Channel: "builders",
				From:    "peer-1",
				Body:    json.RawMessage(`{"text":"hello"}`),
			}},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Network: networkService,
		}, nativeApproveAllPolicyInputs())

		statusResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDNetworkStatus},
		)
		if err != nil {
			t.Fatalf("Registry.Call(network_status) error = %v", err)
		}
		requireNativeStructuredContains(t, statusResult, []byte(`"local_peers":2`))

		channelsResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDNetworkChannels},
		)
		if err != nil {
			t.Fatalf("Registry.Call(network_channels) error = %v", err)
		}
		requireNativeStructuredContains(t, channelsResult, []byte(`"channel":"builders"`))

		inboxResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-1"},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDNetworkInbox},
		)
		if err != nil {
			t.Fatalf("Registry.Call(network_inbox) error = %v", err)
		}
		requireNativeStructuredContains(t, inboxResult, []byte(`"msg-1"`))
		if networkService.statusCalls != 1 ||
			networkService.channelsCalls != 1 ||
			networkService.inboxCalls != 1 ||
			networkService.inboxSessionID != "sess-1" {
			t.Fatalf("network inspection calls = %#v", networkService)
		}
	})

	t.Run("Should send network messages through the existing network service boundary", func(t *testing.T) {
		t.Parallel()

		networkService := &nativeNetworkStub{
			sendErr: fmt.Errorf("%w: session=sess-missing", network.ErrLocalPeerNotFound),
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Network: networkService,
		}, nativeApproveAllPolicyInputs())

		_, err := registry.Call(
			t.Context(),
			toolspkg.Scope{SessionID: "sess-scope"},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDNetworkSend,
				Input: json.RawMessage(
					`{"session_id":"sess-missing","channel":"default","kind":"say","body":{"text":"hello"}}`,
				),
			},
		)
		if !errors.Is(err, network.ErrLocalPeerNotFound) || !errors.Is(err, toolspkg.ErrToolBackendFailed) {
			t.Fatalf("Registry.Call(network_send) error = %v, want wrapped network error", err)
		}
		if networkService.sendCalls != 1 {
			t.Fatalf("Network.Send calls = %d, want 1", networkService.sendCalls)
		}
		if networkService.lastSend.SessionID != "sess-missing" {
			t.Fatalf("SendRequest.SessionID = %q, want input session", networkService.lastSend.SessionID)
		}
		if got, want := string(networkService.lastSend.Body), `{"text":"hello"}`; got != want {
			t.Fatalf("SendRequest.Body = %s, want %s", got, want)
		}
	})
	t.Run("Should read session tools through the existing session manager boundary", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
		info := &session.Info{
			ID:          "sess-1",
			AgentName:   "coder",
			WorkspaceID: "ws-1",
			State:       session.StateActive,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		manager := apitest.StubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) {
				return []*session.Info{info}, nil
			},
			StatusFn: func(_ context.Context, id string) (*session.Info, error) {
				if id != "sess-1" {
					return nil, session.ErrSessionNotFound
				}
				return info, nil
			},
			EventsFn: func(_ context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
				if id != "sess-1" || query.Limit != 1 {
					t.Fatalf("Events query = %q %#v", id, query)
				}
				return []store.SessionEvent{{
					ID:        "event-1",
					SessionID: id,
					Sequence:  1,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: now,
				}}, nil
			},
			HistoryFn: func(_ context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error) {
				if id != "sess-1" || query.Limit != 1 {
					t.Fatalf("History query = %q %#v", id, query)
				}
				return []store.TurnHistory{{
					TurnID: "turn-1",
					Events: []store.SessionEvent{{
						ID:        "event-1",
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
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Sessions: manager,
		}, nativeApproveAllPolicyInputs())

		for _, tc := range []struct {
			id    toolspkg.ToolID
			input json.RawMessage
			want  []byte
		}{
			{toolspkg.ToolIDSessionList, nil, []byte(`"sess-1"`)},
			{toolspkg.ToolIDSessionStatus, json.RawMessage(`{"session_id":"sess-1"}`), []byte(`"session"`)},
			{toolspkg.ToolIDSessionEvents, json.RawMessage(`{"session_id":"sess-1","limit":1}`), []byte(`"event-1"`)},
			{toolspkg.ToolIDSessionHistory, json.RawMessage(`{"session_id":"sess-1","limit":1}`), []byte(`"turn-1"`)},
			{toolspkg.ToolIDSessionDescribe, json.RawMessage(`{"session_id":"sess-1","limit":1}`), []byte(`"history"`)},
		} {
			t.Run(tc.id.String(), func(t *testing.T) {
				result, err := registry.Call(
					t.Context(),
					toolspkg.Scope{},
					toolspkg.CallRequest{ToolID: tc.id, Input: tc.input},
				)
				if err != nil {
					t.Fatalf("Registry.Call(%s) error = %v", tc.id, err)
				}
				requireNativeStructuredContains(t, result, tc.want)
			})
		}
	})

	t.Run("Should read workspace tools through the existing workspace service boundary", func(t *testing.T) {
		t.Parallel()

		now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
		workspace := workspacepkg.Workspace{
			ID:           "ws-1",
			RootDir:      "/workspace/agh",
			Name:         "agh",
			DefaultAgent: "coder",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		manager := apitest.StubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) {
				return []*session.Info{{ID: "sess-1", AgentName: "coder", WorkspaceID: "ws-1"}}, nil
			},
		}
		workspaces := apitest.StubWorkspaceService{
			ListFn: func(context.Context) ([]workspacepkg.Workspace, error) {
				return []workspacepkg.Workspace{workspace}, nil
			},
			GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
				if ref != "ws-1" {
					return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
				}
				return workspace, nil
			},
			ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
				if ref != "ws-1" {
					return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
				}
				return workspacepkg.ResolvedWorkspace{
					Workspace: workspace,
					Agents:    []aghconfig.AgentDef{{Name: "coder", Provider: "codex"}},
					Skills:    []workspacepkg.SkillPath{{Dir: "/workspace/agh/skills/review", Source: "workspace"}},
				}, nil
			},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Sessions:   manager,
			Workspaces: workspaces,
		}, nativeApproveAllPolicyInputs())

		for _, tc := range []struct {
			id    toolspkg.ToolID
			input json.RawMessage
			want  []byte
		}{
			{toolspkg.ToolIDWorkspaceList, nil, []byte(`"workspaces"`)},
			{toolspkg.ToolIDWorkspaceInfo, json.RawMessage(`{"workspace":"ws-1"}`), []byte(`"workspace"`)},
			{toolspkg.ToolIDWorkspaceDescribe, json.RawMessage(`{"workspace":"ws-1"}`), []byte(`"skills"`)},
		} {
			t.Run(tc.id.String(), func(t *testing.T) {
				result, err := registry.Call(
					t.Context(),
					toolspkg.Scope{},
					toolspkg.CallRequest{ToolID: tc.id, Input: tc.input},
				)
				if err != nil {
					t.Fatalf("Registry.Call(%s) error = %v", tc.id, err)
				}
				requireNativeStructuredContains(t, result, tc.want)
			})
		}
	})

	t.Run("Should read memory tools through the current memory store with redaction", func(t *testing.T) {
		t.Parallel()

		rawClaim := "agh_claim_secret123"
		globalDir := filepath.Join(t.TempDir(), "global-memory")
		catalogPath := filepath.Join(t.TempDir(), "memory.db")
		memoryStore := memorypkg.NewStore(globalDir, memorypkg.WithCatalogDatabasePath(catalogPath))
		workspaceRoot := filepath.Join(t.TempDir(), "workspace")
		if err := memoryStore.Write(
			memorypkg.ScopeGlobal,
			"global.md",
			nativeMemoryDocument(
				"Global "+rawClaim,
				"Global description "+rawClaim,
				memorypkg.MemoryTypeUser,
				"global memory body "+rawClaim,
			),
		); err != nil {
			t.Fatalf("Write(global memory) error = %v", err)
		}
		if err := memoryStore.ForWorkspace(workspaceRoot).Write(
			memorypkg.ScopeWorkspace,
			"workspace.md",
			nativeMemoryDocument(
				"Workspace "+rawClaim,
				"Workspace description "+rawClaim,
				memorypkg.MemoryTypeProject,
				"workspace memory body "+rawClaim,
			),
		); err != nil {
			t.Fatalf("Write(workspace memory) error = %v", err)
		}
		workspaces := apitest.StubWorkspaceService{
			GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
				if ref != "ws-1" {
					return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
				}
				return workspacepkg.Workspace{ID: "ws-1", RootDir: workspaceRoot}, nil
			},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			MemoryStore: memoryStore,
			Workspaces:  workspaces,
		}, nativeApproveAllPolicyInputs())

		listResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryList,
				Input:  json.RawMessage(`{"scope":"workspace","workspace":"ws-1"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_list) error = %v", err)
		}
		requireNativeStructuredContains(t, listResult, []byte(`"workspace.md"`))
		requireNativeStructuredExcludes(t, listResult, []byte(rawClaim))

		globalListResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryList,
				Input:  json.RawMessage(`{"scope":"global"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_list global) error = %v", err)
		}
		requireNativeStructuredContains(t, globalListResult, []byte(`"global.md"`))
		requireNativeStructuredExcludes(t, globalListResult, []byte(`"workspace.md"`))
		requireNativeStructuredExcludes(t, globalListResult, []byte(rawClaim))

		combinedListResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryList,
				Input:  json.RawMessage(`{"workspace":"ws-1"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_list combined) error = %v", err)
		}
		requireNativeStructuredContains(t, combinedListResult, []byte(`"global.md"`))
		requireNativeStructuredContains(t, combinedListResult, []byte(`"workspace.md"`))
		requireNativeStructuredExcludes(t, combinedListResult, []byte(rawClaim))

		readResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryRead,
				Input:  json.RawMessage(`{"filename":"global.md","scope":"global"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_read) error = %v", err)
		}
		requireNativeStructuredContains(t, readResult, []byte(`agh_claim_[REDACTED]`))
		requireNativeStructuredExcludes(t, readResult, []byte(rawClaim))

		workspaceReadResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryRead,
				Input:  json.RawMessage(`{"filename":"workspace.md","scope":"workspace","workspace":"ws-1"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_read workspace) error = %v", err)
		}
		requireNativeStructuredContains(t, workspaceReadResult, []byte(`"workspace.md"`))
		requireNativeStructuredExcludes(t, workspaceReadResult, []byte(rawClaim))

		searchResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemorySearch,
				Input:  json.RawMessage(`{"query":"memory","workspace":"ws-1"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_search) error = %v", err)
		}
		requireNativeStructuredContains(t, searchResult, []byte(`"workspace.md"`))
		requireNativeStructuredExcludes(t, searchResult, []byte(rawClaim))

		historyResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryHistory,
				Input:  json.RawMessage(`{"workspace":"ws-1","limit":10}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(memory_history) error = %v", err)
		}
		requireNativeStructuredContains(t, historyResult, []byte(`"memory.search"`))
		requireNativeStructuredExcludes(t, historyResult, []byte(rawClaim))

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryRead,
				Input:  json.RawMessage(`{"filename":"missing.md","scope":"global"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolNotFound) {
			t.Fatalf("Registry.Call(memory_read missing) error = %v, want ErrToolNotFound", err)
		}
		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDMemoryHistory,
				Input:  json.RawMessage(`{"since":"not-a-date"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolInvalidInput) {
			t.Fatalf("Registry.Call(memory_history invalid since) error = %v, want ErrToolInvalidInput", err)
		}
	})

	t.Run("Should read observe tools through the observer without leaking event secrets", func(t *testing.T) {
		t.Parallel()

		rawClaim := "agh_claim_observe123"
		now := time.Date(2026, 4, 29, 15, 0, 0, 0, time.UTC)
		observer := &nativeObserverStub{
			eventSummaries: []store.EventSummary{
				{
					ID:        "evt-1",
					SessionID: "sess-1",
					Type:      "agent_message",
					AgentName: "coder",
					Summary:   "deploy completed " + rawClaim,
					Timestamp: now,
				},
				{
					ID:        "evt-2",
					SessionID: "sess-2",
					Type:      "agent_message",
					AgentName: "reviewer",
					Summary:   "review completed",
					Timestamp: now.Add(time.Second),
				},
			},
			health: observe.Health{
				Status:         "ok",
				ActiveSessions: 1,
				ActiveAgents:   1,
				Retention: observe.RetentionHealth{
					LastSweepError: "sweep failed " + rawClaim,
				},
				Version: "test",
			},
		}
		registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
			Observer: observer,
		}, nativeApproveAllPolicyInputs())

		eventsResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDObserveEvents,
				Input:  json.RawMessage(`{"limit":1}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(observe_events) error = %v", err)
		}
		requireNativeStructuredContains(t, eventsResult, []byte(`"evt-1"`))
		requireNativeStructuredExcludes(t, eventsResult, []byte(rawClaim))

		filteredEventsResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDObserveEvents,
				Input:  json.RawMessage(`{"session_id":"sess-2","since":"2026-04-29T15:00:00Z"}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(observe_events filtered) error = %v", err)
		}
		requireNativeStructuredContains(t, filteredEventsResult, []byte(`"evt-2"`))
		requireNativeStructuredExcludes(t, filteredEventsResult, []byte(`"evt-1"`))

		searchResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDObserveSearch,
				Input:  json.RawMessage(`{"query":"deploy","limit":10}`),
			},
		)
		if err != nil {
			t.Fatalf("Registry.Call(observe_search) error = %v", err)
		}
		requireNativeStructuredContains(t, searchResult, []byte(`"evt-1"`))
		requireNativeStructuredExcludes(t, searchResult, []byte(`"evt-2"`))
		requireNativeStructuredExcludes(t, searchResult, []byte(rawClaim))

		metricsResult, err := registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{ToolID: toolspkg.ToolIDObserveMetrics},
		)
		if err != nil {
			t.Fatalf("Registry.Call(observe_metrics) error = %v", err)
		}
		requireNativeStructuredContains(t, metricsResult, []byte(`"active_sessions":1`))
		requireNativeStructuredContains(t, metricsResult, []byte(`agh_claim_[REDACTED]`))
		requireNativeStructuredExcludes(t, metricsResult, []byte(rawClaim))

		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDObserveEvents,
				Input:  json.RawMessage(`{"since":"not-a-date"}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolInvalidInput) {
			t.Fatalf("Registry.Call(observe_events invalid since) error = %v, want ErrToolInvalidInput", err)
		}
		_, err = registry.Call(
			t.Context(),
			toolspkg.Scope{},
			toolspkg.CallRequest{
				ToolID: toolspkg.ToolIDObserveSearch,
				Input:  json.RawMessage(`{"query":""}`),
			},
		)
		if !errors.Is(err, toolspkg.ErrToolInvalidInput) {
			t.Fatalf("Registry.Call(observe_search empty query) error = %v, want ErrToolInvalidInput", err)
		}
	})

	t.Run(
		"Should read bridge tools through existing projections without leaking credential material",
		func(t *testing.T) {
			t.Parallel()

			rawClaim := "agh_claim_bridge123"
			now := time.Date(2026, 4, 29, 16, 0, 0, 0, time.UTC)
			degradation := &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: "refresh failed " + rawClaim,
			}
			instance := bridgepkg.BridgeInstance{
				ID:             "bridge-1",
				Scope:          bridgepkg.ScopeGlobal,
				Platform:       "slack",
				ExtensionName:  "slack-ext",
				DisplayName:    "Slack",
				Source:         bridgepkg.BridgeInstanceSourceDynamic,
				Enabled:        true,
				Status:         bridgepkg.BridgeStatusReady,
				DMPolicy:       bridgepkg.BridgeDMPolicyOpen,
				RoutingPolicy:  bridgepkg.RoutingPolicy{IncludePeer: true},
				ProviderConfig: json.RawMessage(`{"bot_token":"secret-value"}`),
				Degradation:    degradation,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			observer := &nativeObserverStub{
				bridgeHealth: []observe.BridgeInstanceHealth{{
					BridgeInstanceID: "bridge-1",
					Status:           bridgepkg.BridgeStatusReady,
					RouteCount:       2,
					LastError:        "provider returned " + rawClaim,
				}},
			}
			bridges := apitest.StubBridgeService{
				ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return []bridgepkg.BridgeInstance{instance}, nil
				},
				GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
					if id != "bridge-1" {
						return nil, bridgepkg.ErrBridgeInstanceNotFound
					}
					next := instance
					return &next, nil
				},
			}
			registry := newDaemonNativeRegistry(t, daemonNativeToolsDeps{
				Bridges:  bridges,
				Observer: observer,
			}, nativeApproveAllPolicyInputs())

			listResult, err := registry.Call(
				t.Context(),
				toolspkg.Scope{},
				toolspkg.CallRequest{ToolID: toolspkg.ToolIDBridgesList},
			)
			if err != nil {
				t.Fatalf("Registry.Call(bridges_list) error = %v", err)
			}
			requireNativeStructuredContains(t, listResult, []byte(`"bridge-1"`))
			requireNativeStructuredContains(t, listResult, []byte(`"route_count":2`))
			requireNativeStructuredContains(t, listResult, []byte(`agh_claim_[REDACTED]`))
			requireNativeStructuredExcludes(t, listResult, []byte(`bot_token`))
			requireNativeStructuredExcludes(t, listResult, []byte(`secret-value`))
			requireNativeStructuredExcludes(t, listResult, []byte(rawClaim))

			statusResult, err := registry.Call(
				t.Context(),
				toolspkg.Scope{},
				toolspkg.CallRequest{
					ToolID: toolspkg.ToolIDBridgesStatus,
					Input:  json.RawMessage(`{"bridge_id":"bridge-1"}`),
				},
			)
			if err != nil {
				t.Fatalf("Registry.Call(bridges_status) error = %v", err)
			}
			requireNativeStructuredContains(t, statusResult, []byte(`"bridge-1"`))
			requireNativeStructuredExcludes(t, statusResult, []byte(`bot_token`))
			requireNativeStructuredExcludes(t, statusResult, []byte(`secret-value`))
			requireNativeStructuredExcludes(t, statusResult, []byte(rawClaim))

			aggregateStatusResult, err := registry.Call(
				t.Context(),
				toolspkg.Scope{},
				toolspkg.CallRequest{ToolID: toolspkg.ToolIDBridgesStatus},
			)
			if err != nil {
				t.Fatalf("Registry.Call(bridges_status aggregate) error = %v", err)
			}
			requireNativeStructuredContains(t, aggregateStatusResult, []byte(`"status_counts":{"ready":1}`))
			requireNativeStructuredExcludes(t, aggregateStatusResult, []byte(`bot_token`))
			requireNativeStructuredExcludes(t, aggregateStatusResult, []byte(rawClaim))

			_, err = registry.Call(
				t.Context(),
				toolspkg.Scope{},
				toolspkg.CallRequest{
					ToolID: toolspkg.ToolIDBridgesStatus,
					Input:  json.RawMessage(`{"bridge_id":"missing"}`),
				},
			)
			if !errors.Is(err, bridgepkg.ErrBridgeInstanceNotFound) {
				t.Fatalf("Registry.Call(bridges_status missing) error = %v, want ErrBridgeInstanceNotFound", err)
			}
		},
	)
}

func TestDaemonBootToolRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Should wire the native registry during daemon boot", func(t *testing.T) {
		t.Parallel()

		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		skillsRegistry := newLoadedNativeSkillRegistry(t)
		state := &bootState{
			cfg:            cfg,
			skillsRegistry: skillsRegistry,
			deps: RuntimeDeps{
				SkillsRegistry: skillsRegistry,
				Network:        &nativeNetworkStub{},
				Tasks:          &nativeTaskManager{},
			},
		}
		daemon := &Daemon{}

		if err := daemon.bootToolRegistry(t.Context(), state); err != nil {
			t.Fatalf("bootToolRegistry() error = %v", err)
		}
		if state.toolRegistry == nil || state.deps.ToolRegistry == nil {
			t.Fatalf(
				"tool registry wiring = state:%#v deps:%#v, want both populated",
				state.toolRegistry,
				state.deps.ToolRegistry,
			)
		}
		view, err := state.deps.ToolRegistry.Get(t.Context(), toolspkg.Scope{Operator: true}, toolspkg.ToolIDTaskCancel)
		if err != nil {
			t.Fatalf("ToolRegistry.Get(task_cancel) error = %v", err)
		}
		if !view.Descriptor.Destructive || view.Descriptor.ReadOnly {
			t.Fatalf("task_cancel risk flags = %#v, want destructive mutating tool", view.Descriptor)
		}
		_, err = state.deps.ToolRegistry.Get(t.Context(), toolspkg.Scope{Operator: true}, "agh__skill_remove")
		if !errors.Is(err, toolspkg.ErrToolNotFound) {
			t.Fatalf("ToolRegistry.Get(skill_remove) error = %v, want ErrToolNotFound", err)
		}
	})
}

func TestDaemonNativeRuntimePolicyResolver(t *testing.T) {
	t.Parallel()

	t.Run("Should resolve default discovery and scoped runtime policy inputs", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		homePaths := testHomePaths(t)
		cfg := testConfig(t, homePaths)
		sessions := &nativeToolPolicySessionStub{
			info: &session.Info{
				ID:        "sess-1",
				AgentName: "coder",
				State:     session.StateActive,
			},
		}
		agents := &nativeToolPolicyAgentResolverStub{
			agent: aghconfig.AgentDef{
				Name:     "coder",
				Provider: "opencode",
				Prompt:   "Use the available tools to help the user.",
			},
		}
		resolver, err := newNativeToolPolicyResolver(nativeToolPolicyResolverDeps{
			Config:            &cfg,
			Sessions:          sessions,
			AgentResolver:     agents,
			ApprovalAvailable: true,
			DefaultToolsets: []toolspkg.ToolsetID{
				toolspkg.ToolsetIDBootstrap,
				toolspkg.ToolsetIDCatalog,
			},
		})
		if err != nil {
			t.Fatalf("newNativeToolPolicyResolver() error = %v", err)
		}
		registry := newDaemonNativeRegistryWithPolicyResolver(t, daemonNativeToolsDeps{
			Skills: newLoadedNativeSkillRegistry(t),
			Tasks:  &nativeTaskManager{},
		}, resolver)
		scope := toolspkg.Scope{SessionID: "sess-1"}

		views, err := registry.SessionProjection(ctx, scope)
		if err != nil {
			t.Fatalf("SessionProjection(default discovery) error = %v", err)
		}
		requireNativeViewContains(t, views, toolspkg.ToolIDToolList)
		requireNativeViewContains(t, views, toolspkg.ToolIDToolInfo)
		requireNativeViewContains(t, views, toolspkg.ToolIDSkillView)
		requireNativeViewExcludes(t, views, toolspkg.ToolIDTaskRead)

		_, err = registry.Call(ctx, scope, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDToolInfo,
			Input:  json.RawMessage(`{"tool_id":"agh__tool_list"}`),
		})
		if err != nil {
			t.Fatalf("Registry.Call(tool_info default discovery) error = %v", err)
		}

		agents.agent.Toolsets = []string{toolspkg.ToolsetIDTasks.String()}
		views, err = registry.SessionProjection(ctx, scope)
		if err != nil {
			t.Fatalf("SessionProjection(agent narrowed to tasks) error = %v", err)
		}
		requireNativeViewContains(t, views, toolspkg.ToolIDTaskRead)
		requireNativeViewExcludes(t, views, toolspkg.ToolIDToolInfo)
		_, err = registry.Call(ctx, scope, toolspkg.CallRequest{
			ToolID: toolspkg.ToolIDToolInfo,
			Input:  json.RawMessage(`{"tool_id":"agh__tool_list"}`),
		})
		requireToolReason(t, err, toolspkg.ErrToolDenied, toolspkg.ReasonPolicyDenied)

		agents.agent.Toolsets = nil
		sessions.info.Lineage = &store.SessionLineage{
			ParentSessionID: "parent-1",
			RootSessionID:   "root-1",
			SpawnDepth:      1,
			PermissionPolicy: store.SessionPermissionPolicy{
				Tools: []string{toolspkg.ToolIDToolInfo.String()},
			},
		}
		views, err = registry.SessionProjection(ctx, scope)
		if err != nil {
			t.Fatalf("SessionProjection(session lineage) error = %v", err)
		}
		requireNativeViewContains(t, views, toolspkg.ToolIDToolInfo)
		requireNativeViewExcludes(t, views, toolspkg.ToolIDToolList)
		_, err = registry.Call(ctx, scope, toolspkg.CallRequest{ToolID: toolspkg.ToolIDToolList})
		requireToolReason(t, err, toolspkg.ErrToolDenied, toolspkg.ReasonSessionDenied)
	})
}

func newDaemonNativeRegistry(
	t *testing.T,
	deps daemonNativeToolsDeps,
	policyInputs toolspkg.PolicyInputs,
) *toolspkg.RuntimeRegistry {
	t.Helper()

	return newDaemonNativeRegistryWithPolicyResolver(
		t,
		deps,
		toolspkg.NewStaticPolicyInputResolver(policyInputs),
	)
}

func newDaemonNativeRegistryWithPolicyResolver(
	t *testing.T,
	deps daemonNativeToolsDeps,
	resolver toolspkg.PolicyInputResolver,
) *toolspkg.RuntimeRegistry {
	t.Helper()

	var registry *toolspkg.RuntimeRegistry
	deps.Registry = func() toolspkg.Registry {
		return registry
	}
	provider, err := newDaemonNativeProvider(deps)
	if err != nil {
		t.Fatalf("newDaemonNativeProvider() error = %v", err)
	}
	toolsets, err := builtintools.ToolsetCatalog()
	if err != nil {
		t.Fatalf("builtin.ToolsetCatalog() error = %v", err)
	}
	registry, err = toolspkg.NewRegistry(
		toolspkg.WithProviders(provider),
		toolspkg.WithPolicyInputResolver(resolver, toolsets),
		toolspkg.WithDefaultMaxResultBytes(aghconfig.DefaultToolsMaxResultBytes),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
}

type nativeToolPolicySessionStub struct {
	info *session.Info
	err  error
}

func (s *nativeToolPolicySessionStub) Status(context.Context, string) (*session.Info, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.info, nil
}

type nativeToolPolicyAgentResolverStub struct {
	agent aghconfig.AgentDef
	err   error
}

func (r *nativeToolPolicyAgentResolverStub) ResolveAgent(
	name string,
	_ *workspacepkg.ResolvedWorkspace,
) (aghconfig.AgentDef, error) {
	if r.err != nil {
		return aghconfig.AgentDef{}, r.err
	}
	if name != r.agent.Name {
		return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, name)
	}
	return r.agent, nil
}

func nativeApproveAllPolicyInputs() toolspkg.PolicyInputs {
	return toolspkg.PolicyInputs{
		SystemPermissionMode: toolspkg.PermissionModeApproveAll,
		ApprovalAvailable:    true,
	}
}

func newLoadedNativeSkillRegistry(t *testing.T) *skills.Registry {
	t.Helper()

	registry := skills.NewRegistry(skills.RegistryConfig{BundledFS: skillbundled.FS()})
	if err := registry.LoadAll(t.Context()); err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	return registry
}

func requireNativeStructuredContains(t *testing.T, result toolspkg.ToolResult, needle []byte) {
	t.Helper()

	if !bytes.Contains(result.Structured, needle) {
		t.Fatalf("structured result = %s, want to contain %s", result.Structured, needle)
	}
}

func requireNativeStructuredExcludes(t *testing.T, result toolspkg.ToolResult, needle []byte) {
	t.Helper()

	if bytes.Contains(result.Structured, needle) {
		t.Fatalf("structured result = %s, want to exclude %s", result.Structured, needle)
	}
}

func nativeToolViewByID(views []toolspkg.ToolView, id toolspkg.ToolID) *toolspkg.ToolView {
	for i := range views {
		if views[i].Descriptor.ID == id {
			return &views[i]
		}
	}
	return nil
}

func requireNativeToolAvailable(t *testing.T, views []toolspkg.ToolView, id toolspkg.ToolID) {
	t.Helper()

	view := nativeToolViewByID(views, id)
	if view == nil {
		t.Fatalf("projection does not contain %q: %#v", id, views)
	}
	if !view.Availability.Available {
		t.Fatalf("%s availability = %#v, want available", id, view.Availability)
	}
}

func requireNativeToolUnavailableReason(t *testing.T, views []toolspkg.ToolView, id toolspkg.ToolID) {
	t.Helper()

	view := nativeToolViewByID(views, id)
	if view == nil {
		t.Fatalf("projection does not contain %q: %#v", id, views)
	}
	if view.Availability.Available {
		t.Fatalf("%s availability = %#v, want unavailable", id, view.Availability)
	}
	if len(view.Availability.ReasonCodes) == 0 ||
		!slices.Contains(view.Availability.ReasonCodes, toolspkg.ReasonDependencyMissing) {
		t.Fatalf(
			"%s availability reasons = %#v, want dependency_missing",
			id,
			view.Availability.ReasonCodes,
		)
	}
}

func requireNativeViewContains(t *testing.T, views []toolspkg.ToolView, id toolspkg.ToolID) {
	t.Helper()

	for i := range views {
		if views[i].Descriptor.ID == id {
			return
		}
	}
	t.Fatalf("projection does not contain %q: %#v", id, views)
}

func requireNativeViewExcludes(t *testing.T, views []toolspkg.ToolView, id toolspkg.ToolID) {
	t.Helper()

	for i := range views {
		if views[i].Descriptor.ID == id {
			t.Fatalf("projection contains %q: %#v", id, views)
		}
	}
}

func nativeMemoryDocument(name string, description string, typ memorypkg.Type, body string) []byte {
	return fmt.Appendf(nil,
		"---\nname: %s\ndescription: %s\ntype: %s\n---\n\n%s",
		name,
		description,
		typ,
		body,
	)
}

func requireToolReason(t *testing.T, err error, target error, reason toolspkg.ReasonCode) {
	t.Helper()

	if !errors.Is(err, target) {
		t.Fatalf("error = %v, want %v", err, target)
	}
	got, ok := toolspkg.ReasonOf(err)
	if !ok || got != reason {
		t.Fatalf("ReasonOf(error) = %q/%v, want %q", got, ok, reason)
	}
}

type nativeHookBindingsStub struct {
	syncCalls int
	err       error
}

func (b *nativeHookBindingsStub) Sync(context.Context) error {
	b.syncCalls++
	return b.err
}

type nativeObserverStub struct {
	catalog        []hookspkg.CatalogEntry
	catalogCall    int
	runs           []hookspkg.HookRunRecord
	events         []hookspkg.EventDescriptor
	eventSummaries []store.EventSummary
	bridgeHealth   []observe.BridgeInstanceHealth
	health         observe.Health
}

func (o *nativeObserverStub) QueryEvents(
	_ context.Context,
	query store.EventSummaryQuery,
) ([]store.EventSummary, error) {
	results := make([]store.EventSummary, 0, len(o.eventSummaries))
	for _, event := range o.eventSummaries {
		if query.SessionID != "" && event.SessionID != query.SessionID {
			continue
		}
		if query.AgentName != "" && event.AgentName != query.AgentName {
			continue
		}
		if query.Type != "" && event.Type != query.Type {
			continue
		}
		if !query.Since.IsZero() && event.Timestamp.Before(query.Since) {
			continue
		}
		results = append(results, event)
	}
	if query.Limit > 0 && query.Limit < len(results) {
		return results[:query.Limit], nil
	}
	return results, nil
}

func (o *nativeObserverStub) QueryHookCatalog(
	_ context.Context,
	_ hookspkg.CatalogFilter,
) ([]hookspkg.CatalogEntry, error) {
	o.catalogCall++
	return append([]hookspkg.CatalogEntry(nil), o.catalog...), nil
}

func (o *nativeObserverStub) QueryHookRuns(
	context.Context,
	store.HookRunQuery,
) ([]hookspkg.HookRunRecord, error) {
	return append([]hookspkg.HookRunRecord(nil), o.runs...), nil
}

func (o *nativeObserverStub) QueryHookEvents(
	context.Context,
	hookspkg.EventFilter,
) ([]hookspkg.EventDescriptor, error) {
	if len(o.events) > 0 {
		return append([]hookspkg.EventDescriptor(nil), o.events...), nil
	}
	return hookspkg.AllEventDescriptors(), nil
}

func (o *nativeObserverStub) QueryBridgeHealth(context.Context) ([]observe.BridgeInstanceHealth, error) {
	return append([]observe.BridgeInstanceHealth(nil), o.bridgeHealth...), nil
}

func (o *nativeObserverStub) Health(context.Context) (observe.Health, error) {
	return o.health, nil
}

func (o *nativeObserverStub) QueryTaskDashboard(
	context.Context,
	observe.TaskDashboardQuery,
) (observe.TaskDashboardView, error) {
	return observe.TaskDashboardView{}, nil
}

func (o *nativeObserverStub) QueryTaskInbox(
	context.Context,
	observe.TaskInboxQuery,
	taskpkg.ActorIdentity,
) (observe.TaskInboxView, error) {
	return observe.TaskInboxView{}, nil
}

type nativeNetworkStub struct {
	sendErr        error
	sendCalls      int
	lastSend       network.SendRequest
	peers          []network.PeerInfo
	peersCalls     int
	peersChannel   string
	status         *network.Status
	statusCalls    int
	channels       []network.ChannelInfo
	channelsCalls  int
	inbox          []network.Envelope
	inboxCalls     int
	inboxSessionID string
}

func (n *nativeNetworkStub) Send(_ context.Context, req network.SendRequest) (string, error) {
	n.sendCalls++
	n.lastSend = req
	if n.sendErr != nil {
		return "", n.sendErr
	}
	return "msg-1", nil
}

func (n *nativeNetworkStub) ListPeers(_ context.Context, channel string) ([]network.PeerInfo, error) {
	n.peersCalls++
	n.peersChannel = channel
	return append([]network.PeerInfo(nil), n.peers...), nil
}

func (n *nativeNetworkStub) totalCalls() int {
	return n.sendCalls + n.peersCalls + n.statusCalls + n.channelsCalls + n.inboxCalls
}

func (n *nativeNetworkStub) ListChannels(context.Context) ([]network.ChannelInfo, error) {
	n.channelsCalls++
	return append([]network.ChannelInfo(nil), n.channels...), nil
}

func (n *nativeNetworkStub) Status(context.Context) (*network.Status, error) {
	n.statusCalls++
	if n.status != nil {
		status := *n.status
		return &status, nil
	}
	return &network.Status{Enabled: true, Status: network.StatusRunning}, nil
}

func (n *nativeNetworkStub) Inbox(_ context.Context, sessionID string) ([]network.Envelope, error) {
	n.inboxCalls++
	n.inboxSessionID = sessionID
	return append([]network.Envelope(nil), n.inbox...), nil
}

func (n *nativeNetworkStub) WaitInbox(context.Context, string, string) ([]network.Envelope, error) {
	return nil, nil
}

var errUnexpectedNativeTaskCall = errors.New("unexpected native task manager call")

type nativeTaskManager struct {
	unsupportedNativeTaskManager
	createCalls       int
	lastCreateSpec    taskpkg.CreateTask
	childCreateCalls  int
	childParentID     string
	childSpec         taskpkg.CreateTask
	childErr          error
	listCalls         int
	lastQuery         taskpkg.Query
	listSummaries     []taskpkg.Summary
	getCalls          int
	lastGetID         string
	getView           *taskpkg.View
	updateCalls       int
	lastUpdateID      string
	lastPatch         taskpkg.Patch
	updateTask        *taskpkg.Task
	cancelCalls       int
	lastCancelID      string
	lastCancel        taskpkg.CancelTask
	cancelTask        *taskpkg.Task
	runListCalls      int
	lastRunListTaskID string
	lastRunQuery      taskpkg.RunQuery
	runs              []taskpkg.Run
}

func (m *nativeTaskManager) CreateTask(
	_ context.Context,
	spec taskpkg.CreateTask,
	_ taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	m.createCalls++
	m.lastCreateSpec = spec
	return &taskpkg.Task{
		ID:          firstNonEmpty(spec.ID, "task-created"),
		Scope:       spec.Scope,
		WorkspaceID: spec.WorkspaceID,
		Title:       spec.Title,
		Status:      taskpkg.TaskStatusPending,
	}, nil
}

func (m *nativeTaskManager) UpdateTask(
	_ context.Context,
	id string,
	patch taskpkg.Patch,
	_ taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	m.updateCalls++
	m.lastUpdateID = id
	m.lastPatch = patch
	if m.updateTask != nil {
		return m.updateTask, nil
	}
	return &taskpkg.Task{ID: id, Title: stringValue(patch.Title), Status: taskpkg.TaskStatusPending}, nil
}

func (m *nativeTaskManager) CancelTask(
	_ context.Context,
	id string,
	req taskpkg.CancelTask,
	_ taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	m.cancelCalls++
	m.lastCancelID = id
	m.lastCancel = req
	if m.cancelTask != nil {
		return m.cancelTask, nil
	}
	return &taskpkg.Task{ID: id, Title: "Canceled task", Status: taskpkg.TaskStatusCanceled}, nil
}

func (m *nativeTaskManager) GetTask(
	_ context.Context,
	id string,
	_ taskpkg.ActorContext,
) (*taskpkg.View, error) {
	m.getCalls++
	m.lastGetID = id
	if m.getView != nil {
		return m.getView, nil
	}
	return &taskpkg.View{
		Summary: taskpkg.Summary{ID: id, Title: "Read task", Status: taskpkg.TaskStatusPending},
		Task:    taskpkg.Task{ID: id, Title: "Read task", Status: taskpkg.TaskStatusPending},
	}, nil
}

func (m *nativeTaskManager) ListTaskRuns(
	_ context.Context,
	taskID string,
	query taskpkg.RunQuery,
	_ taskpkg.ActorContext,
) ([]taskpkg.Run, error) {
	m.runListCalls++
	m.lastRunListTaskID = taskID
	m.lastRunQuery = query
	return append([]taskpkg.Run(nil), m.runs...), nil
}

func (m *nativeTaskManager) ListTasks(
	_ context.Context,
	query taskpkg.Query,
	_ taskpkg.ActorContext,
) ([]taskpkg.Summary, error) {
	m.listCalls++
	m.lastQuery = query
	return append([]taskpkg.Summary(nil), m.listSummaries...), nil
}

func (m *nativeTaskManager) totalCalls() int {
	return m.createCalls +
		m.childCreateCalls +
		m.listCalls +
		m.getCalls +
		m.updateCalls +
		m.cancelCalls +
		m.runListCalls
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (m *nativeTaskManager) CreateChildTask(
	_ context.Context,
	parentTaskID string,
	spec taskpkg.CreateTask,
	_ taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	m.childCreateCalls++
	m.childParentID = parentTaskID
	m.childSpec = spec
	if m.childErr != nil {
		return nil, m.childErr
	}
	return &taskpkg.Task{
		ID:           firstNonEmpty(spec.ID, "task-child-created"),
		Scope:        spec.Scope,
		WorkspaceID:  spec.WorkspaceID,
		ParentTaskID: parentTaskID,
		Title:        spec.Title,
		Status:       taskpkg.TaskStatusPending,
	}, nil
}

type unsupportedNativeTaskManager struct{}

func (unsupportedNativeTaskManager) CreateTask(
	context.Context,
	taskpkg.CreateTask,
	taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) CreateChildTask(
	context.Context,
	string,
	taskpkg.CreateTask,
	taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) DeleteTask(context.Context, string, taskpkg.ActorContext) error {
	return errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) UpdateTask(
	context.Context,
	string,
	taskpkg.Patch,
	taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) PublishTask(
	context.Context,
	string,
	taskpkg.ExecutionRequest,
	taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) StartTask(
	context.Context,
	string,
	taskpkg.ExecutionRequest,
	taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ApproveTask(
	context.Context,
	string,
	taskpkg.ExecutionRequest,
	taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) RejectTask(
	context.Context,
	string,
	taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) CancelTask(
	context.Context,
	string,
	taskpkg.CancelTask,
	taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) MarkTaskRead(
	context.Context,
	string,
	taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	return taskpkg.TriageState{}, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ArchiveTask(
	context.Context,
	string,
	taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	return taskpkg.TriageState{}, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) DismissTask(
	context.Context,
	string,
	taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	return taskpkg.TriageState{}, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) AddDependency(
	context.Context,
	taskpkg.AddDependency,
	taskpkg.ActorContext,
) error {
	return errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) RemoveDependency(
	context.Context,
	string,
	string,
	taskpkg.ActorContext,
) error {
	return errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) EnqueueRun(
	context.Context,
	taskpkg.EnqueueRun,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ClaimNextRun(
	context.Context,
	taskpkg.ClaimCriteria,
	taskpkg.ActorContext,
) (*taskpkg.ClaimResult, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ClaimRun(
	context.Context,
	string,
	taskpkg.ClaimRun,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) StartRun(
	context.Context,
	string,
	taskpkg.StartRun,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) AttachRunSession(
	context.Context,
	string,
	string,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) HeartbeatRunLease(
	context.Context,
	taskpkg.LeaseHeartbeat,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ReleaseRunLease(
	context.Context,
	taskpkg.LeaseRelease,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) CompleteRunLease(
	context.Context,
	taskpkg.LeaseCompletion,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) FailRunLease(
	context.Context,
	taskpkg.LeaseFailure,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) CompleteRun(
	context.Context,
	string,
	taskpkg.RunResult,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) FailRun(
	context.Context,
	string,
	taskpkg.RunFailure,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) CancelRun(
	context.Context,
	string,
	taskpkg.CancelRun,
	taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) RecoverExpiredRunLeases(
	context.Context,
	taskpkg.ExpiredLeaseRecovery,
	taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) GetTask(
	context.Context,
	string,
	taskpkg.ActorContext,
) (*taskpkg.View, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ListTaskRuns(
	context.Context,
	string,
	taskpkg.RunQuery,
	taskpkg.ActorContext,
) ([]taskpkg.Run, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) ListTasks(
	context.Context,
	taskpkg.Query,
	taskpkg.ActorContext,
) ([]taskpkg.Summary, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) Timeline(
	context.Context,
	string,
	taskpkg.TimelineQuery,
	taskpkg.ActorContext,
) ([]taskpkg.TimelineItem, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) Stream(
	context.Context,
	string,
	taskpkg.StreamQuery,
	taskpkg.ActorContext,
) (<-chan taskpkg.StreamEvent, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) Tree(
	context.Context,
	string,
	taskpkg.ActorContext,
) (*taskpkg.TreeView, error) {
	return nil, errUnexpectedNativeTaskCall
}

func (unsupportedNativeTaskManager) RunDetail(
	context.Context,
	string,
	taskpkg.ActorContext,
) (*taskpkg.RunDetailView, error) {
	return nil, errUnexpectedNativeTaskCall
}
