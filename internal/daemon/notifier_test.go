package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestHooksNotifierDispatchesLifecycleAgentAndStreamEvents(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	var order []string
	runtime := &fakeHookRuntime{
		onDispatchCreate: func(_ context.Context, payload hookspkg.SessionPostCreatePayload) error {
			order = append(order, "create")
			if payload.Event != hookspkg.HookSessionPostCreate {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookSessionPostCreate)
			}
			if payload.Timestamp != fixedNow {
				t.Fatalf("payload.Timestamp = %s, want %s", payload.Timestamp, fixedNow)
			}
			if payload.SessionID != "sess-created" || payload.WorkspaceID != "ws-1" {
				t.Fatalf("payload = %#v, want session metadata", payload)
			}
			return nil
		},
		onDispatchStop: func(_ context.Context, payload hookspkg.SessionPostStopPayload) error {
			order = append(order, "stop")
			if payload.Event != hookspkg.HookSessionPostStop {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookSessionPostStop)
			}
			if payload.CreatedAt.IsZero() || payload.UpdatedAt.IsZero() {
				t.Fatalf("payload timestamps = %#v, want created/updated timestamps", payload)
			}
			return nil
		},
		onTurnStart: func(_ context.Context, payload hookspkg.TurnStartPayload) error {
			order = append(order, "turn-start")
			if payload.Event != hookspkg.HookTurnStart {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookTurnStart)
			}
			return nil
		},
		onTurnEnd: func(_ context.Context, payload hookspkg.TurnEndPayload) error {
			order = append(order, "turn-end")
			if payload.Event != hookspkg.HookTurnEnd {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookTurnEnd)
			}
			return nil
		},
		onMessageStart: func(_ context.Context, payload hookspkg.MessageStartPayload) error {
			order = append(order, "message-start")
			if payload.Event != hookspkg.HookMessageStart {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookMessageStart)
			}
			return nil
		},
		onMessageDelta: func(_ context.Context, payload hookspkg.MessageDeltaPayload) error {
			order = append(order, "message-delta")
			if payload.Event != hookspkg.HookMessageDelta {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookMessageDelta)
			}
			return nil
		},
		onMessageEnd: func(_ context.Context, payload hookspkg.MessageEndPayload) error {
			order = append(order, "message-end")
			if payload.Event != hookspkg.HookMessageEnd {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookMessageEnd)
			}
			return nil
		},
		onPreCompact: func(_ context.Context, payload hookspkg.ContextPreCompactPayload) error {
			order = append(order, "context-pre")
			if payload.Event != hookspkg.HookContextPreCompact {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookContextPreCompact)
			}
			return nil
		},
		onPostCompact: func(_ context.Context, payload hookspkg.ContextPostCompactPayload) error {
			order = append(order, "context-post")
			if payload.Event != hookspkg.HookContextPostCompact {
				t.Fatalf("payload.Event = %q, want %q", payload.Event, hookspkg.HookContextPostCompact)
			}
			return nil
		},
		onToolPreCall: func(_ context.Context, payload hookspkg.ToolPreCallPayload) error {
			order = append(order, "tool-pre")
			if payload.SessionID != "sess-created" || payload.WorkspaceID != "ws-1" {
				t.Fatalf("payload session context = %#v, want session metadata", payload.SessionContext)
			}
			if payload.ToolID != "Read" {
				t.Fatalf("payload.ToolID = %q, want %q", payload.ToolID, "Read")
			}
			return nil
		},
	}
	agentEvents := &recordingNotifier{}
	notifier := newHooksNotifier(discardLogger(), func() time.Time { return fixedNow })
	notifier.setRuntime(runtime, agentEvents)

	sess := &session.Session{
		ID:          "sess-created",
		Name:        "demo",
		AgentName:   "codex",
		WorkspaceID: "ws-1",
		Workspace:   "/tmp/ws-1",
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   fixedNow.Add(-time.Minute),
		UpdatedAt:   fixedNow,
	}

	if _, err := notifier.DispatchSessionPostCreate(
		testutil.Context(t),
		hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, fixedNow),
	); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := notifier.DispatchSessionPostStop(
		testutil.Context(t),
		hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, fixedNow),
	); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
	}
	if _, err := notifier.DispatchTurnStart(testutil.Context(t), hookspkg.TurnStartPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookTurnStart, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
	}); err != nil {
		t.Fatalf("DispatchTurnStart() error = %v", err)
	}
	if _, err := notifier.DispatchMessageStart(testutil.Context(t), hookspkg.MessageStartPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookMessageStart, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
		MessageID:      "msg-1",
	}); err != nil {
		t.Fatalf("DispatchMessageStart() error = %v", err)
	}
	if _, err := notifier.DispatchMessageDelta(testutil.Context(t), hookspkg.MessageDeltaPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookMessageDelta, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
		MessageID:      "msg-1",
	}); err != nil {
		t.Fatalf("DispatchMessageDelta() error = %v", err)
	}
	if _, err := notifier.DispatchMessageEnd(testutil.Context(t), hookspkg.MessageEndPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookMessageEnd, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
		MessageID:      "msg-1",
	}); err != nil {
		t.Fatalf("DispatchMessageEnd() error = %v", err)
	}
	if _, err := notifier.DispatchTurnEnd(testutil.Context(t), hookspkg.TurnEndPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookTurnEnd, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
	}); err != nil {
		t.Fatalf("DispatchTurnEnd() error = %v", err)
	}
	if _, err := notifier.DispatchContextPreCompact(testutil.Context(t), hookspkg.ContextPreCompactPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookContextPreCompact, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
	}); err != nil {
		t.Fatalf("DispatchContextPreCompact() error = %v", err)
	}
	if _, err := notifier.DispatchContextPostCompact(testutil.Context(t), hookspkg.ContextPostCompactPayload{
		PayloadBase:    hookspkg.PayloadBase{Event: hookspkg.HookContextPostCompact, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{SessionID: "sess-created"},
		TurnContext:    hookspkg.TurnContext{TurnID: "turn-1"},
	}); err != nil {
		t.Fatalf("DispatchContextPostCompact() error = %v", err)
	}
	rawToolEvent, err := json.Marshal(map[string]any{
		"sessionUpdate": "tool_call",
		"toolCallId":    "tool-1",
		"kind":          "read",
		"rawInput": map[string]any{
			"path": "/tmp/demo.txt",
		},
		"_meta": map[string]any{
			"claudeCode": map[string]any{
				"toolName": "Read",
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(tool event) error = %v", err)
	}
	notifier.OnAgentEventForSession(testutil.Context(t), sess, acp.AgentEvent{
		Type:       acp.EventTypeToolCall,
		SessionID:  "acp-session-1",
		TurnID:     "turn-1",
		ToolCallID: "tool-1",
		Raw:        rawToolEvent,
	})

	wantOrder := []string{
		"create",
		"stop",
		"turn-start",
		"message-start",
		"message-delta",
		"message-end",
		"turn-end",
		"context-pre",
		"context-post",
		"tool-pre",
	}
	if !testutil.EqualStringSlices(order, wantOrder) {
		t.Fatalf("dispatch order = %#v, want %#v", order, wantOrder)
	}
	if got, want := agentEvents.events, []string{"agent"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("agent event notifier events = %#v, want %#v", got, want)
	}
}

func TestHooksNotifierOnAgentEventUsesProvidedSessionIDAndLifecycleNoops(t *testing.T) {
	t.Parallel()

	var gotSessionID string
	runtime := &fakeHookRuntime{
		onToolPreCall: func(_ context.Context, payload hookspkg.ToolPreCallPayload) error {
			gotSessionID = payload.SessionID
			return nil
		},
	}
	notifier := newHooksNotifier(discardLogger(), func() time.Time {
		return time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	})
	notifier.setRuntime(runtime, &recordingNotifier{})

	notifier.OnSessionCreated(testutil.Context(t), nil)
	notifier.OnSessionStopped(testutil.Context(t), nil)
	notifier.OnAgentEvent(testutil.Context(t), "sess-direct", acp.AgentEvent{
		Type:       acp.EventTypeToolCall,
		SessionID:  "acp-session-1",
		TurnID:     "turn-1",
		ToolCallID: "tool-1",
		Raw: mustJSON(t, map[string]any{
			"sessionUpdate": "tool_call",
			"toolCallId":    "tool-1",
			"kind":          "read",
			"_meta": map[string]any{
				"claudeCode": map[string]any{
					"toolName": "Read",
				},
			},
		}),
	})

	if gotSessionID != "sess-direct" {
		t.Fatalf("tool hook SessionID = %q, want %q", gotSessionID, "sess-direct")
	}
}

func TestHooksNotifierLifecycleForwarding(t *testing.T) {
	t.Parallel()

	t.Run("Should forward full session metadata to downstream observers", func(t *testing.T) {
		t.Parallel()

		downstream := &recordingLifecycleForwardNotifier{}
		notifier := newHooksNotifier(discardLogger(), func() time.Time {
			return time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
		})
		notifier.setRuntime(&fakeHookRuntime{}, downstream)
		child := &session.Session{
			ID:          "sess-child",
			Name:        "spawned worker",
			AgentName:   "reviewer",
			Provider:    "codex",
			WorkspaceID: "ws-1",
			Channel:     "default",
			Type:        session.SessionTypeSpawned,
			Lineage: &store.SessionLineage{
				ParentSessionID: "sess-parent",
				RootSessionID:   "sess-parent",
				SpawnDepth:      1,
			},
			State:            session.StateActive,
			SoulSnapshotID:   "soul-child",
			SoulDigest:       "sha256:child",
			ParentSoulDigest: "sha256:parent",
		}

		notifier.OnSessionCreated(testutil.Context(t), child)
		notifier.OnSessionStopped(testutil.Context(t), child)

		if got := len(downstream.created); got != 1 {
			t.Fatalf("created calls = %d, want 1", got)
		}
		created := downstream.created[0].Info()
		if created.Lineage == nil ||
			created.Lineage.ParentSessionID != "sess-parent" ||
			created.ParentSoulDigest != "sha256:parent" ||
			created.SoulDigest != "sha256:child" {
			t.Fatalf("created session metadata = %#v", created)
		}
		if got := len(downstream.stopped); got != 1 {
			t.Fatalf("stopped calls = %d, want 1", got)
		}
	})
}

func TestHooksNotifierEmitsGlobalHookDispatchSummariesForAutonomyHooks(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 5, 4, 22, 0, 0, 0, time.UTC)
	decls := []hookspkg.HookDecl{
		{
			Name:         "coord-stop-observer",
			Event:        hookspkg.HookCoordinatorStopped,
			Mode:         hookspkg.HookModeSync,
			Source:       hookspkg.HookSourceConfig,
			Required:     true,
			Priority:     1000,
			PrioritySet:  true,
			ExecutorKind: hookspkg.HookExecutorNative,
		},
		{
			Name:         "task-release-observer",
			Event:        hookspkg.HookTaskRunReleased,
			Mode:         hookspkg.HookModeSync,
			Source:       hookspkg.HookSourceConfig,
			Required:     true,
			Priority:     1000,
			PrioritySet:  true,
			ExecutorKind: hookspkg.HookExecutorNative,
		},
	}
	executors := map[string]hookspkg.Executor{
		"coord-stop-observer": hookspkg.NewTypedNativeExecutor(
			func(
				context.Context,
				hookspkg.RegisteredHook,
				hookspkg.CoordinatorStoppedPayload,
			) (hookspkg.CoordinatorObservationPatch, error) {
				return hookspkg.CoordinatorObservationPatch{}, nil
			},
		),
		"task-release-observer": hookspkg.NewTypedNativeExecutor(
			func(
				context.Context,
				hookspkg.RegisteredHook,
				hookspkg.TaskRunReleasedPayload,
			) (hookspkg.TaskRunObservationPatch, error) {
				return hookspkg.TaskRunObservationPatch{}, nil
			},
		),
	}

	hooks := hookspkg.NewHooks(
		hookspkg.WithLogger(discardLogger()),
		hookspkg.WithNativeDeclarations(decls),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(executors)),
	)
	t.Cleanup(hooks.Close)
	if err := hooks.Rebuild(testutil.Context(t)); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	summaries := &recordingEventSummaryWriter{}
	notifier := newHooksNotifier(discardLogger(), func() time.Time { return fixedNow })
	notifier.setRuntime(hooks, nil, summaries)

	_, err := notifier.DispatchCoordinatorStopped(testutil.Context(t), hookspkg.CoordinatorStoppedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookCoordinatorStopped,
			Timestamp: fixedNow,
		},
		CoordinatorContext: hookspkg.CoordinatorContext{
			WorkspaceID:          "ws-1",
			AgentName:            "coordinator",
			CoordinatorSessionID: "sess-coordinator-1",
			TaskID:               "task-1",
			RunID:                "run-1",
			WorkflowID:           "wf-1",
		},
		StopReason: "completed",
	})
	if err != nil {
		t.Fatalf("DispatchCoordinatorStopped() error = %v", err)
	}

	_, err = notifier.DispatchTaskRunReleased(testutil.Context(t), hookspkg.TaskRunReleasedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunReleased,
			Timestamp: fixedNow,
		},
		TaskRunContext: hookspkg.TaskRunContext{
			TaskID:        "task-1",
			RunID:         "run-1",
			WorkspaceID:   "ws-1",
			WorkflowID:    "wf-1",
			SessionID:     "sess-worker-1",
			AgentName:     "worker",
			ActorKind:     "agent_session",
			ActorID:       "sess-worker-1",
			ReleaseReason: "manual_release",
		},
		PreviousRunStatus: "claimed",
		PreviousSessionID: "sess-worker-1",
		RecoveryReason:    "manual_release",
	})
	if err != nil {
		t.Fatalf("DispatchTaskRunReleased() error = %v", err)
	}

	records := summaries.snapshot()
	if got, want := len(records), 4; got != want {
		t.Fatalf("len(summary records) = %d, want %d", got, want)
	}

	coordinatorStart := findEventSummaryByHookName(t, records, "coord-stop-observer", "hook.dispatch.start")
	if got, want := coordinatorStart.CoordinatorSessionID, "sess-coordinator-1"; got != want {
		t.Fatalf("coordinator start CoordinatorSessionID = %q, want %q", got, want)
	}
	if got, want := coordinatorStart.WorkflowID, "wf-1"; got != want {
		t.Fatalf("coordinator start WorkflowID = %q, want %q", got, want)
	}

	releaseStart := findEventSummaryByHookName(t, records, "task-release-observer", "hook.dispatch.start")
	if got, want := releaseStart.ReleaseReason, "manual_release"; got != want {
		t.Fatalf("release start ReleaseReason = %q, want %q", got, want)
	}
	if got, want := releaseStart.ActorID, "sess-worker-1"; got != want {
		t.Fatalf("release start ActorID = %q, want %q", got, want)
	}
	if got, want := releaseStart.TaskID, "task-1"; got != want {
		t.Fatalf("release start TaskID = %q, want %q", got, want)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func findEventSummaryByHookName(
	t *testing.T,
	summaries []store.EventSummary,
	hookName string,
	eventType string,
) store.EventSummary {
	t.Helper()

	for _, summary := range summaries {
		if summary.HookName == hookName && summary.Type == eventType {
			return summary
		}
	}
	t.Fatalf("missing event summary for hook %q type %q in %#v", hookName, eventType, summaries)
	return store.EventSummary{}
}

type recordingLifecycleForwardNotifier struct {
	created []*session.Session
	stopped []*session.Session
}

func (n *recordingLifecycleForwardNotifier) OnSessionCreated(_ context.Context, sess *session.Session) {
	n.created = append(n.created, sess)
}

func (n *recordingLifecycleForwardNotifier) OnSessionStopped(_ context.Context, sess *session.Session) {
	n.stopped = append(n.stopped, sess)
}

func (n *recordingLifecycleForwardNotifier) OnAgentEvent(context.Context, string, any) {}

type recordingEventSummaryWriter struct {
	mu        sync.Mutex
	summaries []store.EventSummary
}

func (w *recordingEventSummaryWriter) WriteEventSummary(_ context.Context, summary store.EventSummary) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.summaries = append(w.summaries, summary)
	return nil
}

func (w *recordingEventSummaryWriter) snapshot() []store.EventSummary {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]store.EventSummary(nil), w.summaries...)
}

func TestDaemonNativeHooksDriveObserverAndDreamCallbacks(t *testing.T) {
	t.Parallel()

	observer := &spyLifecycleObserver{}
	dream := &spyDreamRuntime{}
	extractor := newSpyMessagePersistedObserver()
	decls, executors := daemonNativeHooks(observer, dream, extractor)
	hooks := hookspkg.NewHooks(
		hookspkg.WithLogger(discardLogger()),
		hookspkg.WithNativeDeclarations(decls),
		hookspkg.WithExecutorResolver(daemonExecutorResolver(executors)),
	)
	t.Cleanup(hooks.Close)

	if err := hooks.Rebuild(testutil.Context(t)); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	fixedNow := time.Date(2026, 4, 9, 15, 0, 0, 0, time.UTC)
	sess := &session.Session{
		ID:          "sess-user",
		Name:        "demo",
		AgentName:   "codex",
		WorkspaceID: "ws-1",
		Workspace:   "/tmp/ws-1",
		Type:        session.SessionTypeUser,
		State:       session.StateStopped,
		CreatedAt:   fixedNow.Add(-time.Hour),
		UpdatedAt:   fixedNow,
	}

	if _, err := hooks.DispatchSessionPostCreate(
		testutil.Context(t),
		hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, fixedNow),
	); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := hooks.DispatchSessionPostStop(
		testutil.Context(t),
		hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, fixedNow),
	); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
	}
	messagePayload := hookspkg.SessionMessagePersistedPayload{
		PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookSessionMessagePersisted, Timestamp: fixedNow},
		SessionContext: hookspkg.SessionContext{
			SessionID:   sess.ID,
			AgentName:   sess.AgentName,
			WorkspaceID: sess.WorkspaceID,
			Workspace:   sess.Workspace,
		},
		MessageID:  "msg-1",
		MessageSeq: 1,
		Role:       "assistant",
		Text:       "done",
	}
	if _, err := hooks.DispatchSessionMessagePersisted(testutil.Context(t), messagePayload); err != nil {
		t.Fatalf("DispatchSessionMessagePersisted() error = %v", err)
	}

	if got := len(observer.created); got != 1 {
		t.Fatalf("len(observer.created) = %d, want 1", got)
	}
	if got := len(observer.stopped); got != 1 {
		t.Fatalf("len(observer.stopped) = %d, want 1", got)
	}
	if observer.created[0].Info().CreatedAt != sess.CreatedAt {
		t.Fatalf("observer created CreatedAt = %s, want %s", observer.created[0].Info().CreatedAt, sess.CreatedAt)
	}
	if got, want := dream.calls, []string{"session_stop:ws-1"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("dream calls = %#v, want %#v", got, want)
	}
	gotMessage := extractor.wait(t)
	if gotMessage.SessionID != sess.ID || gotMessage.MessageID != "msg-1" {
		t.Fatalf("extractor payload = %#v, want session/message ids", gotMessage)
	}
}

func TestMarketplaceHookAllowedHonorsConsentKeys(t *testing.T) {
	t.Parallel()

	marketplaceSkill := marketplaceSkillForTest("registry.example", "@registry/hook-a", "hash-123")
	if marketplaceHookAllowed(marketplaceSkill, nil) {
		t.Fatal("marketplaceHookAllowed() = true, want false without consent")
	}

	allowed := marketplaceHookAllowlist([]string{"@registry/hook-a"})
	if !marketplaceHookAllowed(marketplaceSkill, allowed) {
		t.Fatal("marketplaceHookAllowed() = false, want true for allowed slug")
	}

	allowed = marketplaceHookAllowlist([]string{"registry.example:@registry/hook-a"})
	if !marketplaceHookAllowed(marketplaceSkill, allowed) {
		t.Fatal("marketplaceHookAllowed() = false, want true for allowed registry slug")
	}

	allowed = marketplaceHookAllowlist([]string{"hash-123"})
	if !marketplaceHookAllowed(marketplaceSkill, allowed) {
		t.Fatal("marketplaceHookAllowed() = false, want true for allowed hash")
	}
}

func TestHooksBridgeHelperCloningAndTimestamp(t *testing.T) {
	t.Parallel()

	notifier := newHooksNotifier(discardLogger(), nil)
	before := time.Now().UTC().Add(-time.Second)
	got := notifier.timestamp()
	after := time.Now().UTC().Add(time.Second)
	if got.Before(before) || got.After(after) {
		t.Fatalf("timestamp() = %s, want current time between %s and %s", got, before, after)
	}

	readOnly := true
	original := hookspkg.HookDecl{
		Name:     "config-hook",
		Source:   hookspkg.HookSourceConfig,
		Args:     []string{"one"},
		Env:      map[string]string{"KEY": "value"},
		Metadata: map[string]string{"note": "keep"},
		Matcher: hookspkg.HookMatcher{
			ToolReadOnly: &readOnly,
		},
	}

	filtered := filterHookDeclsBySource([]hookspkg.HookDecl{original}, hookspkg.HookSourceConfig)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	filtered[0].Args[0] = "changed"
	filtered[0].Env["KEY"] = "changed"
	filtered[0].Metadata["note"] = "changed"
	*filtered[0].Matcher.ToolReadOnly = false

	if original.Args[0] != "one" {
		t.Fatalf("original.Args = %#v, want unchanged", original.Args)
	}
	if original.Env["KEY"] != "value" {
		t.Fatalf("original.Env = %#v, want unchanged", original.Env)
	}
	if original.Metadata["note"] != "keep" {
		t.Fatalf("original.Metadata = %#v, want unchanged", original.Metadata)
	}
	if !*original.Matcher.ToolReadOnly {
		t.Fatal("original matcher ToolReadOnly was mutated")
	}

	if got := cloneStringMap(nil); got != nil {
		t.Fatalf("cloneStringMap(nil) = %#v, want nil", got)
	}
}

func TestScopeWorkspaceHookDeclsOnlyInjectsSupportedMatcherFields(t *testing.T) {
	t.Parallel()

	newDecls := func() []hookspkg.HookDecl {
		return []hookspkg.HookDecl{
			{
				Name:  "session",
				Event: hookspkg.HookSessionPostCreate,
			},
			{
				Name:  "task-run",
				Event: hookspkg.HookTaskRunEnqueued,
			},
			{
				Name:  "message",
				Event: hookspkg.HookMessageDelta,
			},
		}
	}
	resolved := workspaceResolvedForTest("ws-1", "/tmp/ws-1")

	testCases := []struct {
		name   string
		assert func(t *testing.T, decls []hookspkg.HookDecl, scoped []hookspkg.HookDecl)
	}{
		{
			name: "Should inject workspace id and root for session hooks",
			assert: func(t *testing.T, _ []hookspkg.HookDecl, scoped []hookspkg.HookDecl) {
				t.Helper()

				if scoped[0].Matcher.WorkspaceID != resolved.ID {
					t.Fatalf("session WorkspaceID = %q, want %q", scoped[0].Matcher.WorkspaceID, resolved.ID)
				}
				if scoped[0].Matcher.WorkspaceRoot != resolved.RootDir {
					t.Fatalf("session WorkspaceRoot = %q, want %q", scoped[0].Matcher.WorkspaceRoot, resolved.RootDir)
				}
				if err := hookspkg.ValidateMatcherForEvent(scoped[0].Event, scoped[0].Matcher); err != nil {
					t.Fatalf("session matcher validation error = %v", err)
				}
			},
		},
		{
			name: "Should inject only workspace id for task-run hooks",
			assert: func(t *testing.T, _ []hookspkg.HookDecl, scoped []hookspkg.HookDecl) {
				t.Helper()

				if scoped[1].Matcher.WorkspaceID != resolved.ID {
					t.Fatalf("task-run WorkspaceID = %q, want %q", scoped[1].Matcher.WorkspaceID, resolved.ID)
				}
				if scoped[1].Matcher.WorkspaceRoot != "" {
					t.Fatalf(
						"task-run WorkspaceRoot = %q, want empty because task-run hooks do not support it",
						scoped[1].Matcher.WorkspaceRoot,
					)
				}
				if err := hookspkg.ValidateMatcherForEvent(scoped[1].Event, scoped[1].Matcher); err != nil {
					t.Fatalf("task-run matcher validation error = %v", err)
				}
			},
		},
		{
			name: "Should not inject workspace fields for message hooks",
			assert: func(t *testing.T, _ []hookspkg.HookDecl, scoped []hookspkg.HookDecl) {
				t.Helper()

				if scoped[2].Matcher.WorkspaceID != "" || scoped[2].Matcher.WorkspaceRoot != "" {
					t.Fatalf(
						"message matcher workspace fields = %#v, want no unsupported workspace scoping",
						scoped[2].Matcher,
					)
				}
				if err := hookspkg.ValidateMatcherForEvent(scoped[2].Event, scoped[2].Matcher); err != nil {
					t.Fatalf("message matcher validation error = %v", err)
				}
			},
		},
		{
			name: "Should not mutate original declarations",
			assert: func(t *testing.T, decls []hookspkg.HookDecl, _ []hookspkg.HookDecl) {
				t.Helper()

				for idx, decl := range decls {
					if decl.Matcher.WorkspaceID != "" || decl.Matcher.WorkspaceRoot != "" {
						t.Fatalf("decls[%d] matcher workspace fields were mutated: %#v", idx, decl.Matcher)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decls := newDecls()
			scoped := scopeWorkspaceHookDecls(decls, &resolved)
			if len(scoped) != len(decls) {
				t.Fatalf("len(scoped) = %d, want %d", len(scoped), len(decls))
			}
			tc.assert(t, decls, scoped)
		})
	}
}

func TestScopeWorkspaceHookDeclsInjectsWorkspaceWorkingDirWhenUnset(t *testing.T) {
	t.Parallel()

	t.Run("Should inherit the workspace root as working directory for relative hooks", func(t *testing.T) {
		t.Parallel()

		resolved := workspaceResolvedForTest("ws-1", "/tmp/ws-1")
		decls := []hookspkg.HookDecl{
			{
				Name:  "task-run",
				Event: hookspkg.HookTaskRunEnqueued,
			},
			{
				Name:  "message",
				Event: hookspkg.HookMessageDelta,
			},
		}

		scoped := scopeWorkspaceHookDecls(decls, &resolved)
		if got, want := scoped[0].WorkingDir, resolved.RootDir; got != want {
			t.Fatalf("task-run WorkingDir = %q, want %q", got, want)
		}
		if got, want := scoped[1].WorkingDir, resolved.RootDir; got != want {
			t.Fatalf("message WorkingDir = %q, want %q", got, want)
		}
	})

	t.Run("Should preserve an explicit working directory", func(t *testing.T) {
		t.Parallel()

		resolved := workspaceResolvedForTest("ws-1", "/tmp/ws-1")
		decls := []hookspkg.HookDecl{{
			Name:       "explicit",
			Event:      hookspkg.HookTaskRunEnqueued,
			WorkingDir: "/tmp/keep-me",
		}}

		scoped := scopeWorkspaceHookDecls(decls, &resolved)
		if got, want := scoped[0].WorkingDir, "/tmp/keep-me"; got != want {
			t.Fatalf("explicit WorkingDir = %q, want %q", got, want)
		}
	})
}

func TestDispatchRuntimeAndExecutorResolvers(t *testing.T) {
	t.Parallel()

	notifier := newHooksNotifier(
		discardLogger(),
		func() time.Time { return time.Date(2026, 4, 9, 16, 0, 0, 0, time.UTC) },
	)
	var nilCtx context.Context
	payload, err := dispatchRuntime(
		nilCtx,
		notifier,
		hookspkg.HookSessionPostCreate,
		"seed",
		func(_ hookRuntime, _ context.Context, item string) (string, error) {
			return item + "-unused", nil
		},
	)
	if err != nil {
		t.Fatalf("dispatchRuntime(nil runtime) error = %v, want nil", err)
	}
	if payload != "seed" {
		t.Fatalf("dispatchRuntime(nil runtime) payload = %q, want %q", payload, "seed")
	}

	runtime := &fakeHookRuntime{}
	notifier.setRuntime(runtime, nil)

	result, err := dispatchRuntime(
		context.Background(),
		notifier,
		hookspkg.HookEventPreRecord,
		"seed",
		func(_ hookRuntime, _ context.Context, item string) (string, error) {
			return item + "-ok", nil
		},
	)
	if err != nil {
		t.Fatalf("dispatchRuntime(rebuild false) error = %v, want nil", err)
	}
	if result != "seed-ok" {
		t.Fatalf("dispatchRuntime(rebuild false) result = %q, want %q", result, "seed-ok")
	}

	result, err = dispatchRuntime(
		context.Background(),
		notifier,
		hookspkg.HookSessionPostCreate,
		"seed",
		func(_ hookRuntime, _ context.Context, item string) (string, error) {
			return item + "-dispatched", nil
		},
	)
	if err != nil {
		t.Fatalf("dispatchRuntime() error = %v, want nil", err)
	}
	if result != "seed-dispatched" {
		t.Fatalf("dispatchRuntime() result = %q, want %q", result, "seed-dispatched")
	}

	_, err = dispatchRuntime(
		nilCtx,
		notifier,
		hookspkg.HookSessionPostCreate,
		"seed",
		func(_ hookRuntime, _ context.Context, item string) (string, error) {
			return item, nil
		},
	)
	if err == nil || !strings.Contains(err.Error(), "requires a non-nil context") {
		t.Fatalf("dispatchRuntime(nil context) error = %v, want non-nil context detail", err)
	}

	workspaceRoot := t.TempDir()
	subprocessExecutor, err := defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "subprocess",
		ExecutorKind: hookspkg.HookExecutorSubprocess,
		Command:      "/bin/sh",
		Args:         []string{"-c", "printf '%s|' \"$HOOK_SCOPE_ENV\"; pwd"},
		Env:          map[string]string{"HOOK_SCOPE_ENV": "kept"},
		Matcher:      hookspkg.HookMatcher{WorkspaceRoot: workspaceRoot},
	})
	if err != nil {
		t.Fatalf("defaultDaemonExecutorResolver(subprocess) error = %v, want nil", err)
	}
	if subprocessExecutor.Kind() != hookspkg.HookExecutorSubprocess {
		t.Fatalf("subprocess executor kind = %q, want %q", subprocessExecutor.Kind(), hookspkg.HookExecutorSubprocess)
	}
	output, err := subprocessExecutor.Execute(t.Context(), hookspkg.RegisteredHook{Name: "subprocess"}, nil)
	if err != nil {
		t.Fatalf("subprocess executor.Execute() error = %v, want nil", err)
	}
	resolvedWorkspaceRoot, err := filepath.EvalSymlinks(workspaceRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks(workspaceRoot) error = %v, want nil", err)
	}
	if got := strings.TrimSpace(string(output)); got != "kept|"+resolvedWorkspaceRoot {
		t.Fatalf("subprocess executor output = %q, want %q", got, "kept|"+resolvedWorkspaceRoot)
	}

	explicitDir := t.TempDir()
	subprocessExecutor, err = defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "subprocess-working-dir",
		ExecutorKind: hookspkg.HookExecutorSubprocess,
		Command:      "/bin/sh",
		Args:         []string{"-c", "pwd"},
		WorkingDir:   explicitDir,
		Matcher:      hookspkg.HookMatcher{WorkspaceRoot: workspaceRoot},
	})
	if err != nil {
		t.Fatalf("defaultDaemonExecutorResolver(subprocess working dir) error = %v, want nil", err)
	}
	output, err = subprocessExecutor.Execute(t.Context(), hookspkg.RegisteredHook{Name: "subprocess-working-dir"}, nil)
	if err != nil {
		t.Fatalf("subprocess executor.Execute(working dir) error = %v, want nil", err)
	}
	resolvedExplicitDir, err := filepath.EvalSymlinks(explicitDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(explicitDir) error = %v, want nil", err)
	}
	if got := strings.TrimSpace(string(output)); got != resolvedExplicitDir {
		t.Fatalf("subprocess executor working dir output = %q, want %q", got, resolvedExplicitDir)
	}

	wasmExecutor, err := defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "wasm",
		ExecutorKind: hookspkg.HookExecutorWASM,
	})
	if err != nil {
		t.Fatalf("defaultDaemonExecutorResolver(wasm) error = %v, want nil", err)
	}
	if wasmExecutor.Kind() != hookspkg.HookExecutorWASM {
		t.Fatalf("wasm executor kind = %q, want %q", wasmExecutor.Kind(), hookspkg.HookExecutorWASM)
	}

	if _, err := defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "native",
		ExecutorKind: hookspkg.HookExecutorNative,
	}); err == nil || !strings.Contains(err.Error(), "requires an explicit binding") {
		t.Fatalf("defaultDaemonExecutorResolver(native) error = %v, want explicit binding error", err)
	}

	if _, err := defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "unknown",
		ExecutorKind: hookspkg.HookExecutorKind("mystery"),
	}); err == nil || !strings.Contains(err.Error(), "unsupported executor kind") {
		t.Fatalf("defaultDaemonExecutorResolver(unknown) error = %v, want unsupported kind error", err)
	}

	resolver := daemonExecutorResolver(map[string]hookspkg.Executor{
		"bound": hookspkg.NewTypedNativeExecutor(
			func(context.Context, hookspkg.RegisteredHook, hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePatch, error) {
				return hookspkg.SessionPostCreatePatch{}, nil
			},
		),
	})
	nativeExecutor, err := resolver(hookspkg.HookDecl{Name: "bound", ExecutorKind: hookspkg.HookExecutorNative})
	if err != nil {
		t.Fatalf("daemonExecutorResolver(bound native) error = %v, want nil", err)
	}
	if nativeExecutor.Kind() != hookspkg.HookExecutorNative {
		t.Fatalf("native executor kind = %q, want %q", nativeExecutor.Kind(), hookspkg.HookExecutorNative)
	}

	if _, err := resolver(
		hookspkg.HookDecl{Name: "missing", ExecutorKind: hookspkg.HookExecutorNative},
	); err == nil ||
		!strings.Contains(err.Error(), "missing native hook executor") {
		t.Fatalf("daemonExecutorResolver(missing native) error = %v, want missing native executor error", err)
	}
}

type spyLifecycleObserver struct {
	created []*session.Session
	stopped []*session.Session
}

func (s *spyLifecycleObserver) OnSessionCreated(_ context.Context, sess *session.Session) {
	s.created = append(s.created, sess)
}

func (s *spyLifecycleObserver) OnSessionStopped(_ context.Context, sess *session.Session) {
	s.stopped = append(s.stopped, sess)
}

type spyDreamRuntime struct {
	calls []string
}

func (s *spyDreamRuntime) EnqueueCheck(reason string, workspaceRef string) {
	s.calls = append(s.calls, reason+":"+workspaceRef)
}

type spyMessagePersistedObserver struct {
	ch chan hookspkg.SessionMessagePersistedPayload
}

func newSpyMessagePersistedObserver() *spyMessagePersistedObserver {
	return &spyMessagePersistedObserver{ch: make(chan hookspkg.SessionMessagePersistedPayload, 1)}
}

func (s *spyMessagePersistedObserver) HandleSessionMessagePersisted(
	_ context.Context,
	payload hookspkg.SessionMessagePersistedPayload,
) error {
	s.ch <- payload
	return nil
}

func (s *spyMessagePersistedObserver) wait(t *testing.T) hookspkg.SessionMessagePersistedPayload {
	t.Helper()
	select {
	case payload := <-s.ch:
		return payload
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for memory extractor hook")
		return hookspkg.SessionMessagePersistedPayload{}
	}
}

func marketplaceSkillForTest(registry string, slug string, hash string) *skills.Skill {
	return &skills.Skill{
		Source: skills.SourceMarketplace,
		Meta:   skills.SkillMeta{Name: "marketplace-hook"},
		Provenance: &skills.Provenance{
			Registry: registry,
			Slug:     slug,
			Hash:     hash,
		},
	}
}

func workspaceResolvedForTest(id string, root string) workspacepkg.ResolvedWorkspace {
	return workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      id,
			RootDir: root,
		},
	}
}
