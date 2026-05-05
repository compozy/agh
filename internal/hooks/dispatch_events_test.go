package hooks

import (
	"context"
	"testing"
	"time"
)

type recordingDispatchEventEmitter struct {
	calls int
}

func (e *recordingDispatchEventEmitter) EmitHookDispatchEvent(
	context.Context,
	any,
	RegisteredHook,
	DispatchPhase,
	HookRunOutcome,
	error,
	int,
	time.Time,
) {
	e.calls++
}

func TestDispatchEventEmitterContextRoundTrip(t *testing.T) {
	t.Parallel()

	emitter := &recordingDispatchEventEmitter{}

	ctx := WithDispatchEventEmitter(context.Background(), emitter)
	got := DispatchEventEmitterFromContext(ctx)
	if got != emitter {
		t.Fatalf("DispatchEventEmitterFromContext() = %#v, want %#v", got, emitter)
	}
	got.EmitHookDispatchEvent(
		ctx,
		PromptPayload{},
		RegisteredHook{Name: "hook"},
		DispatchPhaseStart,
		HookRunOutcomeApplied,
		nil,
		1,
		time.Now(),
	)
	if emitter.calls != 1 {
		t.Fatalf("emitter calls = %d, want 1", emitter.calls)
	}

	baseCtx := context.Background()
	if got := WithDispatchEventEmitter(baseCtx, nil); got != baseCtx {
		t.Fatalf("WithDispatchEventEmitter(ctx, nil) returned different context")
	}
	if got := DispatchEventEmitterFromContext(
		context.WithValue(context.Background(), dispatchEventEmitterContextKey{}, "bad"),
	); got != nil {
		t.Fatalf("DispatchEventEmitterFromContext(non-emitter) = %#v, want nil", got)
	}
}

func TestTurnIDFromPayloadTrimsSupportedPayloads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload any
	}{
		{name: "input", payload: InputPreSubmitPayload{TurnContext: TurnContext{TurnID: " turn-input "}}},
		{name: "prompt", payload: PromptPayload{TurnContext: TurnContext{TurnID: " turn-prompt "}}},
		{name: "event", payload: EventRecordPayload{TurnContext: TurnContext{TurnID: " turn-event "}}},
		{name: "turn", payload: TurnPayload{TurnContext: TurnContext{TurnID: " turn "}}},
		{name: "message", payload: MessagePayload{TurnContext: TurnContext{TurnID: " turn-message "}}},
		{name: "tool pre", payload: ToolPreCallPayload{TurnContext: TurnContext{TurnID: " turn-tool-pre "}}},
		{name: "tool post", payload: ToolPostCallPayload{TurnContext: TurnContext{TurnID: " turn-tool-post "}}},
		{name: "tool error", payload: ToolPostErrorPayload{TurnContext: TurnContext{TurnID: " turn-tool-error "}}},
		{
			name:    "permission request",
			payload: PermissionRequestPayload{TurnContext: TurnContext{TurnID: " turn-permission-request "}},
		},
		{
			name:    "permission resolution",
			payload: PermissionResolutionPayload{TurnContext: TurnContext{TurnID: " turn-permission-resolution "}},
		},
		{name: "context compact", payload: ContextCompactPayload{TurnContext: TurnContext{TurnID: " turn-compact "}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := TurnIDFromPayload(tc.payload)
			if got == "" || got[0] == ' ' || got[len(got)-1] == ' ' {
				t.Fatalf("TurnIDFromPayload() = %q, want non-empty trimmed turn ID", got)
			}
		})
	}

	if got := TurnIDFromPayload(SessionPreCreatePayload{}); got != "" {
		t.Fatalf("TurnIDFromPayload(unsupported) = %q, want empty", got)
	}
}

func TestSessionContextFromPayloadCoversHookFamilies(t *testing.T) {
	t.Parallel()

	session := SessionContext{
		SessionID:   "session-1",
		AgentName:   "codex",
		WorkspaceID: "workspace-1",
		Workspace:   "/workspace",
	}
	cases := []struct {
		name     string
		payload  any
		expected SessionContext
	}{
		{name: "session pre create", payload: SessionPreCreatePayload{SessionContext: session}, expected: session},
		{name: "session lifecycle", payload: SessionLifecyclePayload{SessionContext: session}, expected: session},
		{name: "sandbox prepare", payload: SandboxPreparePayload{SessionContext: session}, expected: session},
		{name: "sandbox ready", payload: SandboxReadyPayload{SessionContext: session}, expected: session},
		{name: "sandbox sync before", payload: SandboxSyncBeforePayload{SessionContext: session}, expected: session},
		{name: "sandbox sync after", payload: SandboxSyncAfterPayload{SessionContext: session}, expected: session},
		{name: "sandbox stop", payload: SandboxStopPayload{SessionContext: session}, expected: session},
		{name: "input", payload: InputPreSubmitPayload{SessionContext: session}, expected: session},
		{name: "prompt", payload: PromptPayload{SessionContext: session}, expected: session},
		{name: "event", payload: EventRecordPayload{SessionContext: session}, expected: session},
		{name: "agent pre start", payload: AgentPreStartPayload{SessionContext: session}, expected: session},
		{name: "agent lifecycle", payload: AgentLifecyclePayload{SessionContext: session}, expected: session},
		{
			name:     "heartbeat wake before",
			payload:  AgentHeartbeatWakeBeforePayload{SessionContext: session},
			expected: session,
		},
		{
			name:     "heartbeat wake after",
			payload:  AgentHeartbeatWakeAfterPayload{SessionContext: session},
			expected: session,
		},
		{name: "session health", payload: SessionHealthUpdateAfterPayload{SessionContext: session}, expected: session},
		{name: "turn", payload: TurnPayload{SessionContext: session}, expected: session},
		{name: "message", payload: MessagePayload{SessionContext: session}, expected: session},
		{name: "tool pre", payload: ToolPreCallPayload{SessionContext: session}, expected: session},
		{name: "tool post", payload: ToolPostCallPayload{SessionContext: session}, expected: session},
		{name: "tool error", payload: ToolPostErrorPayload{SessionContext: session}, expected: session},
		{name: "permission request", payload: PermissionRequestPayload{SessionContext: session}, expected: session},
		{
			name:     "permission resolution",
			payload:  PermissionResolutionPayload{SessionContext: session},
			expected: session,
		},
		{name: "context compact", payload: ContextCompactPayload{SessionContext: session}, expected: session},
		{
			name: "coordinator pre spawn",
			payload: CoordinatorPreSpawnPayload{CoordinatorContext: CoordinatorContext{
				CoordinatorSessionID: "coord-session",
				AgentName:            "coord-agent",
				WorkspaceID:          "coord-workspace",
				Workspace:            "/coord",
			}},
			expected: SessionContext{
				SessionID:   "coord-session",
				AgentName:   "coord-agent",
				WorkspaceID: "coord-workspace",
				Workspace:   "/coord",
			},
		},
		{
			name: "coordinator lifecycle",
			payload: CoordinatorLifecyclePayload{CoordinatorContext: CoordinatorContext{
				CoordinatorSessionID: "coord-life-session",
				AgentName:            "coord-life-agent",
				WorkspaceID:          "coord-life-workspace",
				Workspace:            "/coord-life",
			}},
			expected: SessionContext{
				SessionID:   "coord-life-session",
				AgentName:   "coord-life-agent",
				WorkspaceID: "coord-life-workspace",
				Workspace:   "/coord-life",
			},
		},
		{
			name: "task enqueued",
			payload: TaskRunEnqueuedPayload{TaskRunContext: TaskRunContext{
				SessionID:      "task-session",
				AgentName:      "task-agent",
				WorkspaceID:    "task-workspace",
				SoulSnapshotID: "snap-task",
				SoulDigest:     "digest-task",
			}},
			expected: SessionContext{
				SessionID:          "task-session",
				AgentName:          "task-agent",
				WorkspaceID:        "task-workspace",
				SessionSoulContext: &SessionSoulContext{SoulSnapshotID: "snap-task", SoulDigest: "digest-task"},
			},
		},
		{
			name:     "task pre claim",
			payload:  TaskRunPreClaimPayload{TaskRunContext: TaskRunContext{SessionID: "task-pre"}},
			expected: SessionContext{SessionID: "task-pre"},
		},
		{
			name:     "task post claim",
			payload:  TaskRunPostClaimPayload{TaskRunContext: TaskRunContext{SessionID: "task-post"}},
			expected: SessionContext{SessionID: "task-post"},
		},
		{
			name:     "task lease",
			payload:  TaskRunLeasePayload{TaskRunContext: TaskRunContext{SessionID: "task-lease"}},
			expected: SessionContext{SessionID: "task-lease"},
		},
		{
			name: "spawn pre create",
			payload: SpawnPreCreatePayload{SpawnContext: SpawnContext{
				ChildSessionID: "child-session",
				AgentName:      "child-agent",
				WorkspaceID:    "spawn-workspace",
				Workspace:      "/spawn",
				SoulSnapshotID: "snap-spawn",
				SoulDigest:     "digest-spawn",
			}},
			expected: SessionContext{
				SessionID:          "child-session",
				AgentName:          "child-agent",
				WorkspaceID:        "spawn-workspace",
				Workspace:          "/spawn",
				SessionSoulContext: &SessionSoulContext{SoulSnapshotID: "snap-spawn", SoulDigest: "digest-spawn"},
			},
		},
		{
			name:     "spawn lifecycle",
			payload:  SpawnLifecyclePayload{SpawnContext: SpawnContext{ChildSessionID: "child-life"}},
			expected: SessionContext{SessionID: "child-life"},
		},
		{name: "unsupported", payload: NetworkPayload{SessionID: "network-session"}, expected: SessionContext{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SessionContextFromPayload(tc.payload)
			assertSessionContext(t, got, tc.expected)
		})
	}
}

func TestCorrelationFromPayloadCoversDispatchFamilies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		payload  any
		expected DispatchCorrelation
	}{
		{
			name: "coordinator pre spawn",
			payload: CoordinatorPreSpawnPayload{CoordinatorContext: CoordinatorContext{
				CoordinatorSessionID: " coord-session ",
				WorkflowID:           " workflow ",
			}},
			expected: DispatchCorrelation{
				CoordinatorSessionID: "coord-session",
				WorkflowID:           "workflow",
				ActorKind:            "agent_session",
				ActorID:              "coord-session",
			},
		},
		{
			name: "coordinator lifecycle",
			payload: CoordinatorLifecyclePayload{CoordinatorContext: CoordinatorContext{
				CoordinatorSessionID: " coord-life ",
				WorkflowID:           " workflow-life ",
			}},
			expected: DispatchCorrelation{
				CoordinatorSessionID: "coord-life",
				WorkflowID:           "workflow-life",
				ActorKind:            "agent_session",
				ActorID:              "coord-life",
			},
		},
		{
			name:    "task enqueued",
			payload: TaskRunEnqueuedPayload{TaskRunContext: taskRunCorrelationContext()},
			expected: DispatchCorrelation{
				TaskID:               "task-1",
				RunID:                "run-1",
				WorkflowID:           "workflow-1",
				ActorKind:            "agent_session",
				ActorID:              "session-actor",
				ReleaseReason:        "completed",
				CoordinatorSessionID: "coordinator-session",
			},
		},
		{
			name:     "task pre claim",
			payload:  TaskRunPreClaimPayload{TaskRunContext: taskRunCorrelationContext()},
			expected: taskRunDispatchCorrelation(),
		},
		{
			name:     "task post claim",
			payload:  TaskRunPostClaimPayload{TaskRunContext: taskRunCorrelationContext()},
			expected: taskRunDispatchCorrelation(),
		},
		{
			name:     "task lease",
			payload:  TaskRunLeasePayload{TaskRunContext: taskRunCorrelationContext()},
			expected: taskRunDispatchCorrelation(),
		},
		{
			name: "spawn pre create",
			payload: SpawnPreCreatePayload{SpawnContext: SpawnContext{
				TaskID:          " task-spawn ",
				RunID:           " run-spawn ",
				WorkflowID:      " workflow-spawn ",
				ChildSessionID:  " child ",
				ParentSessionID: " parent ",
			}},
			expected: DispatchCorrelation{
				TaskID:     "task-spawn",
				RunID:      "run-spawn",
				WorkflowID: "workflow-spawn",
				ActorKind:  "agent_session",
				ActorID:    "child",
			},
		},
		{
			name: "spawn lifecycle falls back to parent",
			payload: SpawnLifecyclePayload{SpawnContext: SpawnContext{
				TaskID:          " task-parent ",
				RunID:           " run-parent ",
				WorkflowID:      " workflow-parent ",
				ParentSessionID: " parent-only ",
			}},
			expected: DispatchCorrelation{
				TaskID:     "task-parent",
				RunID:      "run-parent",
				WorkflowID: "workflow-parent",
				ActorKind:  "agent_session",
				ActorID:    "parent-only",
			},
		},
		{
			name:    "network peer",
			payload: NetworkPayload{PeerFrom: " peer-a ", SessionID: " session-a "},
			expected: DispatchCorrelation{
				ActorKind: "network_peer",
				ActorID:   "peer-a",
			},
		},
		{
			name:    "network session fallback",
			payload: NetworkPayload{SessionID: " network-session "},
			expected: DispatchCorrelation{
				ActorKind: "network_peer",
				ActorID:   "network-session",
			},
		},
		{
			name:    "session fallback",
			payload: PromptPayload{SessionContext: SessionContext{SessionID: " prompt-session "}},
			expected: DispatchCorrelation{
				ActorKind: "agent_session",
				ActorID:   "prompt-session",
			},
		},
		{name: "empty fallback", payload: struct{}{}, expected: DispatchCorrelation{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CorrelationFromPayload(tc.payload)
			if got != tc.expected {
				t.Fatalf("CorrelationFromPayload() = %#v, want %#v", got, tc.expected)
			}
		})
	}
}

func TestHookTypeValidationBranches(t *testing.T) {
	t.Parallel()

	var source HookSource
	if err := source.UnmarshalText([]byte(" config ")); err != nil {
		t.Fatalf("UnmarshalText(config) error = %v, want nil", err)
	}
	if source != HookSourceConfig {
		t.Fatalf("source = %v, want HookSourceConfig", source)
	}
	if _, err := HookSource(200).MarshalText(); err == nil {
		t.Fatal("MarshalText(invalid) error = nil, want error")
	}
	if err := source.UnmarshalText([]byte("unknown")); err == nil {
		t.Fatal("UnmarshalText(unknown) error = nil, want error")
	}

	if _, err := PriorityFromInt(42); err != nil {
		t.Fatalf("PriorityFromInt(42) error = %v, want nil", err)
	}

	hook := RegisteredHook{
		Name:    "network-observer",
		Event:   HookNetworkMessagePersisted,
		Source:  HookSourceConfig,
		Mode:    HookModeAsync,
		Timeout: time.Second,
	}
	if err := hook.Validate(); err != nil {
		t.Fatalf("RegisteredHook.Validate() error = %v, want nil", err)
	}
	hook.Required = true
	if err := hook.Validate(); err == nil {
		t.Fatal("RegisteredHook.Validate(required async) error = nil, want error")
	}

	resolved := &ResolvedHook{RegisteredHook: RegisteredHook{
		Name:   "resolved",
		Event:  HookPromptPostAssemble,
		Source: HookSourceConfig,
		Mode:   HookModeAsync,
		Executor: NewTypedNativeExecutor(
			func(context.Context, RegisteredHook, PromptPayload) (PromptPatch, error) {
				return PromptPatch{}, nil
			},
		),
	}, Decl: HookDecl{
		Name:         "resolved",
		ExecutorKind: HookExecutorNative,
		SkillSource:  HookSkillSourceWorkspace,
	}}
	if err := resolved.Validate(); err != nil {
		t.Fatalf("ResolvedHook.Validate() error = %v, want nil", err)
	}
	resolved.Decl.Name = "other"
	if err := resolved.Validate(); err == nil {
		t.Fatal("ResolvedHook.Validate(name mismatch) error = nil, want error")
	}
	if err := (*ResolvedHook)(nil).Validate(); err == nil {
		t.Fatal("ResolvedHook.Validate(nil) error = nil, want error")
	}
}

func taskRunCorrelationContext() TaskRunContext {
	return TaskRunContext{
		TaskID:        " task-1 ",
		RunID:         " run-1 ",
		WorkflowID:    " workflow-1 ",
		ActorKind:     " agent_session ",
		ActorID:       " session-actor ",
		ReleaseReason: " completed ",
		SessionID:     " coordinator-session ",
	}
}

func taskRunDispatchCorrelation() DispatchCorrelation {
	return DispatchCorrelation{
		TaskID:               "task-1",
		RunID:                "run-1",
		WorkflowID:           "workflow-1",
		ActorKind:            "agent_session",
		ActorID:              "session-actor",
		ReleaseReason:        "completed",
		CoordinatorSessionID: "coordinator-session",
	}
}

func assertSessionContext(t *testing.T, got SessionContext, want SessionContext) {
	t.Helper()

	if got.SessionID != want.SessionID ||
		got.AgentName != want.AgentName ||
		got.WorkspaceID != want.WorkspaceID ||
		got.Workspace != want.Workspace {
		t.Fatalf("SessionContextFromPayload() = %#v, want %#v", got, want)
	}
	if want.SessionSoulContext == nil {
		if got.SessionSoulContext != nil {
			t.Fatalf("SessionSoulContext = %#v, want nil", got.SessionSoulContext)
		}
		return
	}
	if got.SessionSoulContext == nil {
		t.Fatalf("SessionSoulContext = nil, want %#v", want.SessionSoulContext)
	}
	if got.SoulSnapshotID != want.SoulSnapshotID || got.SoulDigest != want.SoulDigest {
		t.Fatalf("SessionSoulContext = %#v, want %#v", got.SessionSoulContext, want.SessionSoulContext)
	}
}
