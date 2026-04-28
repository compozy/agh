package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/skills"
	skillbundled "github.com/pedronauck/agh/internal/skills/bundled"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
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
}

func TestDaemonBootToolRegistry(t *testing.T) {
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
}

func newDaemonNativeRegistry(
	t *testing.T,
	deps daemonNativeToolsDeps,
	policyInputs toolspkg.PolicyInputs,
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
	toolsets, err := toolspkg.BuiltinToolsetCatalog()
	if err != nil {
		t.Fatalf("BuiltinToolsetCatalog() error = %v", err)
	}
	registry, err = toolspkg.NewRegistry(
		toolspkg.WithProviders(provider),
		toolspkg.WithPolicyInputs(policyInputs, toolsets),
		toolspkg.WithDefaultMaxResultBytes(aghconfig.DefaultToolsMaxResultBytes),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	return registry
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

type nativeNetworkStub struct {
	sendErr      error
	sendCalls    int
	lastSend     network.SendRequest
	peers        []network.PeerInfo
	peersCalls   int
	peersChannel string
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
	return n.sendCalls + n.peersCalls
}

func (n *nativeNetworkStub) ListChannels(context.Context) ([]network.ChannelInfo, error) {
	return nil, nil
}

func (n *nativeNetworkStub) Status(context.Context) (*network.Status, error) {
	return &network.Status{Enabled: true, Status: network.StatusRunning}, nil
}

func (n *nativeNetworkStub) Inbox(context.Context, string) ([]network.Envelope, error) {
	return nil, nil
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
