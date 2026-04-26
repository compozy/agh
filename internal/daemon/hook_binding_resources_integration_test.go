//go:build integration

package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/testutil"
)

type hookBindingIntegrationHarness struct {
	store    resources.Store[hookspkg.HookDecl]
	driver   resources.ReconcileDriver
	hooks    *hookspkg.Hooks
	notifier *hooksNotifier
	actor    resources.MutationActor
}

func TestHookBindingResourceReconcileFiresToolHookThroughSessionNotifier(t *testing.T) {
	toolPayloads := make(chan hookspkg.ToolPreCallPayload, 1)
	h := newHookBindingIntegrationHarness(t, map[string]hookspkg.Executor{
		"tool-hook": hookspkg.NewTypedNativeExecutor(
			func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.ToolPreCallPayload) (hookspkg.ToolCallPatch, error) {
				select {
				case toolPayloads <- payload:
				default:
				}
				return hookspkg.ToolCallPatch{}, nil
			},
		),
	})

	record := h.putBinding(t, "tool-hook", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "tool-hook",
		Event:        hookspkg.HookToolPreCall,
		Source:       hookspkg.HookSourceNative,
		Mode:         hookspkg.HookModeSync,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	if record.Version <= 0 {
		t.Fatalf("record.Version = %d, want positive", record.Version)
	}
	if err := h.driver.RunBoot(testutil.Context(t)); err != nil {
		t.Fatalf("driver.RunBoot() error = %v", err)
	}

	h.notifier.OnAgentEventForSession(testutil.Context(t), integrationSession(), acp.AgentEvent{
		Type:       acp.EventTypeToolCall,
		SessionID:  "acp-session-1",
		TurnID:     "turn-1",
		ToolCallID: "tool-1",
		Raw:        mustMarshalJSON(t, toolEventRaw("tool_call", "", nil)),
	})

	select {
	case payload := <-toolPayloads:
		if payload.SessionID != "sess-1" || payload.WorkspaceID != "ws-1" {
			t.Fatalf("payload.SessionContext = %#v, want session metadata", payload.SessionContext)
		}
		if payload.ToolName != "Read" {
			t.Fatalf("payload.ToolName = %q, want %q", payload.ToolName, "Read")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for resource-backed tool.pre_call hook")
	}
}

func TestHookBindingResourceReconcileFiresPermissionHooksThroughSessionNotifier(t *testing.T) {
	requests := make(chan hookspkg.PermissionRequestPayload, 1)
	resolved := make(chan hookspkg.PermissionResolvedPayload, 1)
	denied := make(chan hookspkg.PermissionDeniedPayload, 1)

	h := newHookBindingIntegrationHarness(t, map[string]hookspkg.Executor{
		"perm-request": hookspkg.NewTypedNativeExecutor(
			func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.PermissionRequestPayload) (hookspkg.PermissionRequestPatch, error) {
				requests <- payload
				return hookspkg.PermissionRequestPatch{}, nil
			},
		),
		"perm-resolved": hookspkg.NewTypedNativeExecutor(
			func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.PermissionResolvedPayload) (hookspkg.PermissionResolvedPatch, error) {
				resolved <- payload
				return hookspkg.PermissionResolvedPatch{}, nil
			},
		),
		"perm-denied": hookspkg.NewTypedNativeExecutor(
			func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.PermissionDeniedPayload) (hookspkg.PermissionDeniedPatch, error) {
				denied <- payload
				return hookspkg.PermissionDeniedPatch{}, nil
			},
		),
	})

	h.putBinding(t, "perm-request", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "perm-request",
		Event:        hookspkg.HookPermissionRequest,
		Source:       hookspkg.HookSourceNative,
		Mode:         hookspkg.HookModeSync,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	h.putBinding(t, "perm-resolved", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "perm-resolved",
		Event:        hookspkg.HookPermissionResolved,
		Source:       hookspkg.HookSourceNative,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	h.putBinding(t, "perm-denied", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "perm-denied",
		Event:        hookspkg.HookPermissionDenied,
		Source:       hookspkg.HookSourceNative,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	if err := h.driver.RunBoot(testutil.Context(t)); err != nil {
		t.Fatalf("driver.RunBoot() error = %v", err)
	}

	sessionValue := integrationSession()
	h.notifier.OnAgentEventForSession(testutil.Context(t), sessionValue, acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "acp-session-1",
		TurnID:    "turn-1",
		RequestID: "perm-1",
		Action:    "session/request_permission",
		Resource:  "/tmp/secret.txt",
		Raw:       mustMarshalJSON(t, permissionEventRaw("perm-1", "", "Read")),
	})
	h.notifier.OnAgentEventForSession(testutil.Context(t), sessionValue, acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "acp-session-1",
		TurnID:    "turn-1",
		RequestID: "perm-1",
		Action:    "session/request_permission",
		Resource:  "/tmp/secret.txt",
		Decision:  "allow",
		Raw:       mustMarshalJSON(t, permissionEventRaw("perm-1", "allow", "Read")),
	})
	h.notifier.OnAgentEventForSession(testutil.Context(t), sessionValue, acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "acp-session-1",
		TurnID:    "turn-1",
		RequestID: "perm-1",
		Action:    "session/request_permission",
		Resource:  "/tmp/secret.txt",
		Decision:  "deny",
		Raw:       mustMarshalJSON(t, permissionEventRaw("perm-1", "deny", "Read")),
	})

	select {
	case payload := <-requests:
		if payload.SessionID != "sess-1" || payload.ToolCall.Kind != "Read" {
			t.Fatalf("permission.request payload = %#v, want sess-1 and Read tool", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for permission.request hook")
	}

	select {
	case payload := <-resolved:
		if payload.Decision != "allow" || payload.DecisionClass != "resolved" {
			t.Fatalf("permission.resolved payload = %#v, want allow/resolved", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for permission.resolved hook")
	}

	select {
	case payload := <-denied:
		if payload.Decision != "deny" || payload.DecisionClass != "denied" {
			t.Fatalf("permission.denied payload = %#v, want deny/denied", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for permission.denied hook")
	}
}

func TestHookBindingResourceReconcileFiresTaskRunHookThroughDaemonBridge(t *testing.T) {
	taskRunPayloads := make(chan hookspkg.TaskRunEnqueuedPayload, 1)
	h := newHookBindingIntegrationHarness(t, map[string]hookspkg.Executor{
		"task-run-hook": hookspkg.NewTypedNativeExecutor(
			func(
				_ context.Context,
				_ hookspkg.RegisteredHook,
				payload hookspkg.TaskRunEnqueuedPayload,
			) (hookspkg.TaskRunObservationPatch, error) {
				select {
				case taskRunPayloads <- payload:
				default:
				}
				return hookspkg.TaskRunObservationPatch{}, nil
			},
		),
	})

	h.putBinding(t, "task-run-hook", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "task-run-hook",
		Event:        hookspkg.HookTaskRunEnqueued,
		Source:       hookspkg.HookSourceNative,
		Mode:         hookspkg.HookModeSync,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			WorkspaceID: "ws-1",
			Autonomy: &hookspkg.AutonomyMatcher{
				TaskID:                "task-1",
				CoordinationChannelID: "coord-ch-1",
			},
		},
	})
	if err := h.driver.RunBoot(testutil.Context(t)); err != nil {
		t.Fatalf("driver.RunBoot() error = %v", err)
	}

	if _, err := h.notifier.DispatchTaskRunEnqueued(testutil.Context(t), hookspkg.TaskRunEnqueuedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunEnqueued,
			Timestamp: time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC),
		},
		TaskRunContext: hookspkg.TaskRunContext{
			TaskID:                "task-1",
			RunID:                 "run-1",
			WorkspaceID:           "ws-1",
			CoordinationChannelID: "coord-ch-1",
		},
	}); err != nil {
		t.Fatalf("DispatchTaskRunEnqueued() error = %v", err)
	}

	select {
	case payload := <-taskRunPayloads:
		if payload.RunID != "run-1" || payload.CoordinationChannelID != "coord-ch-1" {
			t.Fatalf("task-run payload = %#v, want run and coordination channel metadata", payload.TaskRunContext)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for resource-backed task.run.enqueued hook")
	}
}

func TestHookBindingResourceReconcileFailurePreservesAppliedRuntimeState(t *testing.T) {
	toolPayloads := make(chan hookspkg.ToolPreCallPayload, 2)

	h := newHookBindingIntegrationHarness(t, map[string]hookspkg.Executor{
		"tool-stable": hookspkg.NewTypedNativeExecutor(
			func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.ToolPreCallPayload) (hookspkg.ToolCallPatch, error) {
				toolPayloads <- payload
				return hookspkg.ToolCallPatch{}, nil
			},
		),
	})

	record := h.putBinding(t, "tool-hook", 0, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "tool-stable",
		Event:        hookspkg.HookToolPreCall,
		Source:       hookspkg.HookSourceNative,
		Mode:         hookspkg.HookModeSync,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	if err := h.driver.RunBoot(testutil.Context(t)); err != nil {
		t.Fatalf("initial driver.RunBoot() error = %v", err)
	}
	h.notifier.OnAgentEventForSession(testutil.Context(t), integrationSession(), acp.AgentEvent{
		Type:       acp.EventTypeToolCall,
		SessionID:  "acp-session-1",
		TurnID:     "turn-1",
		ToolCallID: "tool-1",
		Raw:        mustMarshalJSON(t, toolEventRaw("tool_call", "", nil)),
	})
	select {
	case <-toolPayloads:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial stable hook dispatch")
	}

	_ = h.putBinding(t, "tool-hook", record.Version, resources.ResourceScope{
		Kind: resources.ResourceScopeKindWorkspace,
		ID:   "ws-1",
	}, hookspkg.HookDecl{
		Name:         "tool-missing",
		Event:        hookspkg.HookToolPreCall,
		Source:       hookspkg.HookSourceNative,
		Mode:         hookspkg.HookModeSync,
		ExecutorKind: hookspkg.HookExecutorNative,
		Matcher: hookspkg.HookMatcher{
			AgentName: "codex",
			ToolName:  "Read",
		},
	})
	if err := h.driver.RunBoot(testutil.Context(t)); err == nil {
		t.Fatal("driver.RunBoot() error = nil, want missing executor failure")
	}

	h.notifier.OnAgentEventForSession(testutil.Context(t), integrationSession(), acp.AgentEvent{
		Type:       acp.EventTypeToolCall,
		SessionID:  "acp-session-1",
		TurnID:     "turn-1",
		ToolCallID: "tool-1",
		Raw:        mustMarshalJSON(t, toolEventRaw("tool_call", "", nil)),
	})
	select {
	case payload := <-toolPayloads:
		if payload.ToolName != "Read" || payload.SessionID != "sess-1" {
			t.Fatalf("post-failure payload = %#v, want stable hook payload", payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for preserved stable hook after projector failure")
	}
}

func newHookBindingIntegrationHarness(
	t *testing.T,
	nativeExecutors map[string]hookspkg.Executor,
) *hookBindingIntegrationHarness {
	t.Helper()

	db := openDaemonTestGlobalDB(t)
	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}
	codec, err := newHookBindingCodec()
	if err != nil {
		t.Fatalf("newHookBindingCodec() error = %v", err)
	}
	store, err := newHookBindingStore(kernel, codec)
	if err != nil {
		t.Fatalf("newHookBindingStore() error = %v", err)
	}
	hooks := hookspkg.NewHooks(
		hookspkg.WithLogger(discardLogger()),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(nativeExecutors)),
	)
	t.Cleanup(hooks.Close)

	registration, err := resources.NewTypedProjectorRegistration(codec, newHookBindingProjector(hooks))
	if err != nil {
		t.Fatalf("NewTypedProjectorRegistration() error = %v", err)
	}
	driver, err := resources.NewReconcileDriver(
		kernel,
		resources.MutationActor{
			Kind:     resources.MutationActorKindDaemon,
			ID:       "integration-control",
			Source:   resources.ResourceSource{Kind: resources.ResourceSourceKind("daemon"), ID: "integration"},
			MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		},
		[]resources.ProjectorRegistration{registration},
		resources.WithReconcileLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("resources.NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		if err := driver.Close(testutil.Context(t)); err != nil {
			t.Fatalf("driver.Close() error = %v", err)
		}
	})

	notifier := newHooksNotifier(discardLogger(), func() time.Time {
		return time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	})
	notifier.setRuntime(hooks, nil)

	return &hookBindingIntegrationHarness{
		store:    store,
		driver:   driver,
		hooks:    hooks,
		notifier: notifier,
		actor: resources.MutationActor{
			Kind:     resources.MutationActorKindDaemon,
			ID:       "integration-writer",
			Source:   resources.ResourceSource{Kind: resources.ResourceSourceKind("daemon"), ID: "integration"},
			MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		},
	}
}

func (h *hookBindingIntegrationHarness) putBinding(
	t *testing.T,
	id string,
	expectedVersion int64,
	scope resources.ResourceScope,
	decl hookspkg.HookDecl,
) resources.Record[hookspkg.HookDecl] {
	t.Helper()

	spec, err := validateHookBindingSpec(testutil.Context(t), scope, decl)
	if err != nil {
		t.Fatalf("validateHookBindingSpec() error = %v", err)
	}
	record, err := h.store.Put(testutil.Context(t), h.actor, resources.Draft[hookspkg.HookDecl]{
		ID:              id,
		Scope:           scope,
		ExpectedVersion: expectedVersion,
		Spec:            spec,
	})
	if err != nil {
		t.Fatalf("store.Put(%q) error = %v", id, err)
	}
	return record
}

func integrationSession() *session.Session {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	return &session.Session{
		ID:          "sess-1",
		Name:        "demo",
		AgentName:   "codex",
		WorkspaceID: "ws-1",
		Workspace:   "/tmp/ws-1",
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   now.Add(-time.Minute),
		UpdatedAt:   now,
	}
}
