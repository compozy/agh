//go:build integration

package session

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/environment"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestManagerIntegrationFullLifecycle(t *testing.T) {
	h := newHarness(t)

	session := createSession(t, h)
	firstPrompt, err := h.manager.Prompt(testutil.Context(t), session.ID, "first")
	if err != nil {
		t.Fatalf("Prompt(first) error = %v", err)
	}
	firstEvents := collectEvents(t, firstPrompt)
	if len(firstEvents) != 2 {
		t.Fatalf("first prompt events = %d, want 2", len(firstEvents))
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	secondPrompt, err := h.manager.Prompt(testutil.Context(t), resumed.ID, "second")
	if err != nil {
		t.Fatalf("Prompt(second) error = %v", err)
	}
	secondEvents := collectEvents(t, secondPrompt)
	if len(secondEvents) != 2 {
		t.Fatalf("second prompt events = %d, want 2", len(secondEvents))
	}

	if err := h.manager.Stop(testutil.Context(t), resumed.ID); err != nil {
		t.Fatalf("final Stop() error = %v", err)
	}

	reopened, err := sessiondb.OpenSessionDB(testutil.Context(t), resumed.ID, resumed.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		if err := reopened.Close(testutil.Context(t)); err != nil {
			t.Fatalf("reopened.Close() error = %v", err)
		}
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopen) error = %v", err)
	}
	if len(events) != 8 {
		t.Fatalf("stored events = %d, want 8", len(events))
	}
	if !containsEventType(events, acp.EventTypeAgentMessage) || !containsEventType(events, acp.EventTypeDone) {
		t.Fatalf("stored events missing expected types: %#v", events)
	}
	if got := countEventType(events, EventTypeSessionStopped); got != 2 {
		t.Fatalf("stored %q events = %d, want 2", EventTypeSessionStopped, got)
	}

	meta := readMeta(t, resumed.MetaPath())
	if meta.State != string(StateStopped) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateStopped)
	}
}

func TestManagerIntegrationUsesRealSQLitePerSessionDB(t *testing.T) {
	h := newHarness(t)

	session := createSession(t, h)
	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "persist")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	_ = collectEvents(t, eventsCh)

	recorder, ok := session.recorderHandle().(*sessiondb.SessionDB)
	if !ok {
		t.Fatalf("recorder = %T, want *sessiondb.SessionDB", session.recorderHandle())
	}
	if got, want := recorder.Path(), session.DBPath(); got != want {
		t.Fatalf("SessionDB.Path() = %q, want %q", got, want)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	reopened, err := sessiondb.OpenSessionDB(testutil.Context(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		if err := reopened.Close(testutil.Context(t)); err != nil {
			t.Fatalf("reopened.Close() error = %v", err)
		}
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopen) error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("Query(reopen) returned 0 events, want persisted rows")
	}
}

func TestManagerIntegrationResumeWithChannelReinjectsBundledNetworkSkillBeforeACPStart(t *testing.T) {
	h := newHarness(t)
	networkSkill, err := bundled.LoadContent(networkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", networkSkillName, err)
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := strings.Count(h.driver.startCalls[0].SystemPrompt, networkSkill); got != 1 {
		t.Fatalf("create prompt network skill occurrences = %d, want 1", got)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := h.driver.startCalls[1].SystemPrompt; !strings.Contains(got, networkSkill) {
		t.Fatalf("resume system prompt = %q, want bundled network skill content", got)
	}
	if got := strings.Count(h.driver.startCalls[1].SystemPrompt, networkSkill); got != 1 {
		t.Fatalf("resume prompt network skill occurrences = %d, want 1", got)
	}
}

func TestManagerIntegrationFullLifecycleHooksFireInOrder(t *testing.T) {
	var (
		mu    sync.Mutex
		order []string
	)
	record := func(entry string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, entry)
	}

	dispatcher := &spyHookDispatcher{
		dispatchSessionPreCreateFn: func(_ context.Context, payload hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error) {
			record("session.pre_create")
			return payload, nil
		},
		dispatchPromptPostAssembleFn: func(_ context.Context, payload hookspkg.PromptPayload) (hookspkg.PromptPayload, error) {
			record("prompt.post_assemble")
			return payload, nil
		},
		dispatchAgentPreStartFn: func(_ context.Context, payload hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error) {
			record("agent.pre_start")
			return payload, nil
		},
		dispatchAgentSpawnedFn: func(_ context.Context, payload hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error) {
			record("agent.spawned")
			return payload, nil
		},
		dispatchSessionPostCreateFn: func(_ context.Context, payload hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error) {
			record("session.post_create")
			return payload, nil
		},
		dispatchInputPreSubmitFn: func(_ context.Context, payload hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error) {
			record("input.pre_submit")
			return payload, nil
		},
		dispatchTurnStartFn: func(_ context.Context, payload hookspkg.TurnStartPayload) (hookspkg.TurnStartPayload, error) {
			record("turn.start")
			return payload, nil
		},
		dispatchTurnEndFn: func(_ context.Context, payload hookspkg.TurnEndPayload) (hookspkg.TurnEndPayload, error) {
			record("turn.end")
			return payload, nil
		},
		dispatchMessageStartFn: func(_ context.Context, payload hookspkg.MessageStartPayload) (hookspkg.MessageStartPayload, error) {
			record("message.start")
			return payload, nil
		},
		dispatchMessageDeltaFn: func(_ context.Context, payload hookspkg.MessageDeltaPayload) (hookspkg.MessageDeltaPayload, error) {
			record("message.delta:" + payload.DeltaType)
			return payload, nil
		},
		dispatchMessageEndFn: func(_ context.Context, payload hookspkg.MessageEndPayload) (hookspkg.MessageEndPayload, error) {
			record("message.end")
			return payload, nil
		},
		dispatchEventPreRecordFn: func(_ context.Context, payload hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error) {
			record("event.pre_record:" + payload.RecordType)
			return payload, nil
		},
		dispatchEventPostRecordFn: func(_ context.Context, payload hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error) {
			record("event.post_record:" + payload.RecordType)
			return payload, nil
		},
		dispatchSessionPreStopFn: func(_ context.Context, payload hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error) {
			record("session.pre_stop")
			return payload, nil
		},
		dispatchAgentStoppedFn: func(_ context.Context, payload hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error) {
			record("agent.stopped")
			return payload, nil
		},
		dispatchSessionPostStopFn: func(_ context.Context, payload hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error) {
			record("session.post_stop")
			return payload, nil
		},
	}

	h := newHarness(t, WithHookSet(fullHookSet(dispatcher)))

	session := createSession(t, h)
	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	_ = collectEvents(t, eventsCh)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	want := []string{
		"session.pre_create",
		"prompt.post_assemble",
		"agent.pre_start",
		"agent.spawned",
		"session.post_create",
		"input.pre_submit",
		"turn.start",
		"event.pre_record:user_message",
		"event.post_record:user_message",
		"message.start",
		"message.delta:text",
		"event.pre_record:agent_message",
		"event.post_record:agent_message",
		"message.end",
		"event.pre_record:done",
		"event.post_record:done",
		"turn.end",
		"session.pre_stop",
		"event.pre_record:session_stopped",
		"event.post_record:session_stopped",
		"agent.stopped",
		"session.post_stop",
	}

	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()
	if !testutil.EqualStringSlices(got, want) {
		t.Fatalf("hook order = %#v, want %#v", got, want)
	}
}

func TestManagerIntegrationEnvironmentNativeHooksLifecycleOrder(t *testing.T) {
	var (
		mu        sync.Mutex
		order     []string
		afterTo   = make(chan struct{})
		ready     = make(chan struct{})
		afterFrom = make(chan struct{})
	)
	record := func(event string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, event)
	}
	waitFor := func(ctx context.Context, ch <-chan struct{}, label string) error {
		select {
		case <-ch:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return errors.New("timed out waiting for " + label)
		}
	}

	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{
			{
				Name:         "env-prepare",
				Event:        hookspkg.HookEnvironmentPrepare,
				Mode:         hookspkg.HookModeSync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			{
				Name:         "env-sync-before",
				Event:        hookspkg.HookEnvironmentSyncBefore,
				Mode:         hookspkg.HookModeSync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			{
				Name:         "env-sync-after",
				Event:        hookspkg.HookEnvironmentSyncAfter,
				Mode:         hookspkg.HookModeAsync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			{
				Name:         "env-ready",
				Event:        hookspkg.HookEnvironmentReady,
				Mode:         hookspkg.HookModeAsync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			{
				Name:         "env-stop",
				Event:        hookspkg.HookEnvironmentStop,
				Mode:         hookspkg.HookModeSync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
		},
		map[string]hookspkg.Executor{
			"env-prepare": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.EnvironmentPreparePayload) (hookspkg.EnvironmentPreparePatch, error) {
				if payload.EnvironmentID == "" || payload.WorkspaceID == "" {
					return hookspkg.EnvironmentPreparePatch{}, errors.New("environment.prepare missing identity fields")
				}
				record("environment.prepare")
				return hookspkg.EnvironmentPreparePatch{}, nil
			}),
			"env-sync-before": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.EnvironmentSyncBeforePayload) (hookspkg.EnvironmentSyncBeforePatch, error) {
				if payload.EnvironmentID == "" || payload.Direction == "" || payload.Reason == "" {
					return hookspkg.EnvironmentSyncBeforePatch{}, errors.New("environment.sync.before missing lifecycle fields")
				}
				record("environment.sync.before:" + payload.Direction)
				return hookspkg.EnvironmentSyncBeforePatch{}, nil
			}),
			"env-sync-after": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.EnvironmentSyncAfterPayload) (hookspkg.EnvironmentSyncAfterPatch, error) {
				if payload.EnvironmentID == "" || payload.Direction == "" || payload.DurationMS < 0 {
					return hookspkg.EnvironmentSyncAfterPatch{}, errors.New("environment.sync.after missing lifecycle fields")
				}
				record("environment.sync.after:" + payload.Direction)
				switch payload.Direction {
				case string(environment.SyncDirectionToRuntime):
					close(afterTo)
				case string(environment.SyncDirectionFromRuntime):
					close(afterFrom)
				default:
					return hookspkg.EnvironmentSyncAfterPatch{}, errors.New("unexpected sync direction " + payload.Direction)
				}
				return hookspkg.EnvironmentSyncAfterPatch{}, nil
			}),
			"env-ready": hookspkg.NewTypedNativeExecutor(func(ctx context.Context, _ hookspkg.RegisteredHook, payload hookspkg.EnvironmentReadyPayload) (hookspkg.EnvironmentReadyPatch, error) {
				if err := waitFor(ctx, afterTo, "environment.sync.after:to_runtime"); err != nil {
					return hookspkg.EnvironmentReadyPatch{}, err
				}
				if payload.EnvironmentID == "" || payload.RuntimeRootDir == "" {
					return hookspkg.EnvironmentReadyPatch{}, errors.New("environment.ready missing runtime fields")
				}
				record("environment.ready")
				close(ready)
				return hookspkg.EnvironmentReadyPatch{}, nil
			}),
			"env-stop": hookspkg.NewTypedNativeExecutor(func(ctx context.Context, _ hookspkg.RegisteredHook, payload hookspkg.EnvironmentStopPayload) (hookspkg.EnvironmentStopPatch, error) {
				if err := waitFor(ctx, afterFrom, "environment.sync.after:from_runtime"); err != nil {
					return hookspkg.EnvironmentStopPatch{}, err
				}
				if payload.EnvironmentID == "" || payload.StopReason == "" {
					return hookspkg.EnvironmentStopPatch{}, errors.New("environment.stop missing stop fields")
				}
				record("environment.stop")
				return hookspkg.EnvironmentStopPatch{}, nil
			}),
		},
	)

	h := newHarness(t, WithHookSet(HookSet{Environment: hooks}))
	session := createSession(t, h)
	if err := waitFor(testutil.Context(t), ready, "environment.ready"); err != nil {
		t.Fatalf("waiting for environment.ready: %v", err)
	}
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	want := []string{
		"environment.prepare",
		"environment.sync.before:to_runtime",
		"environment.sync.after:to_runtime",
		"environment.ready",
		"environment.sync.before:from_runtime",
		"environment.sync.after:from_runtime",
		"environment.stop",
	}
	mu.Lock()
	got := append([]string(nil), order...)
	mu.Unlock()
	if !testutil.EqualStringSlices(got, want) {
		t.Fatalf("environment hook order = %#v, want %#v", got, want)
	}
}

func TestManagerIntegrationContextCompactionUsesPatchedParams(t *testing.T) {
	reason := "patched-reason"
	strategy := "patched-strategy"
	postSeen := make(chan hookspkg.ContextPostCompactPayload, 1)

	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{
			{
				Name:         "context-pre",
				Event:        hookspkg.HookContextPreCompact,
				Mode:         hookspkg.HookModeSync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
			{
				Name:         "context-post",
				Event:        hookspkg.HookContextPostCompact,
				Mode:         hookspkg.HookModeAsync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
		},
		map[string]hookspkg.Executor{
			"context-pre": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.ContextPreCompactPayload) (hookspkg.ContextPreCompactPatch, error) {
				return hookspkg.ContextPreCompactPatch{
					Reason:   &reason,
					Strategy: &strategy,
				}, nil
			}),
			"context-post": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.ContextPostCompactPayload) (hookspkg.ContextPostCompactPatch, error) {
				postSeen <- payload
				return hookspkg.ContextPostCompactPatch{}, nil
			}),
		},
	)

	h := newHarness(t, WithHookSet(fullHookSet(hooks)))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	var seen hookspkg.ContextPreCompactPayload
	result, err := h.manager.runContextCompaction(
		testutil.Context(t),
		session,
		"turn-context",
		"manual",
		"noop",
		"",
		nil,
		func(_ context.Context, payload hookspkg.ContextPreCompactPayload) (hookspkg.ContextPostCompactPayload, error) {
			seen = payload
			return hookspkg.ContextPostCompactPayload{
				Summary: "after",
			}, nil
		},
	)
	if err != nil {
		t.Fatalf("runContextCompaction() error = %v", err)
	}
	if seen.Reason != reason || seen.Strategy != strategy {
		t.Fatalf("compactor saw reason/strategy = %q/%q, want %q/%q", seen.Reason, seen.Strategy, reason, strategy)
	}
	if result.Reason != reason || result.Strategy != strategy {
		t.Fatalf("result reason/strategy = %q/%q, want %q/%q", result.Reason, result.Strategy, reason, strategy)
	}
	select {
	case payload := <-postSeen:
		if payload.Reason != reason || payload.Strategy != strategy {
			t.Fatalf("post hook saw reason/strategy = %q/%q, want %q/%q", payload.Reason, payload.Strategy, reason, strategy)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for context.post_compact hook")
	}
}

func TestManagerIntegrationPreStopRequiredHookErrorPreventsCleanStop(t *testing.T) {
	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "required-pre-stop",
			Event:        hookspkg.HookSessionPreStop,
			Mode:         hookspkg.HookModeSync,
			Required:     true,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"required-pre-stop": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPatch, error) {
				return hookspkg.SessionPreStopPatch{}, errors.New("required hook failed")
			}),
		},
	)

	h := newHarness(t, WithHookSet(fullHookSet(hooks)))
	session := createSession(t, h)

	err := h.manager.Stop(testutil.Context(t), session.ID)
	if err == nil {
		t.Fatal("Stop() error = nil, want required pre-stop hook failure")
	}
	if got := session.Info().State; got != StateActive {
		t.Fatalf("session state after failed Stop() = %q, want %q", got, StateActive)
	}
	if _, ok := h.manager.Get(session.ID); !ok {
		t.Fatalf("Get(%q) = missing, want active session after failed stop", session.ID)
	}

	h.manager.hooks = HookSet{}
	if cleanupErr := h.manager.Stop(testutil.Context(t), session.ID); cleanupErr != nil {
		t.Fatalf("cleanup Stop() error = %v", cleanupErr)
	}
}
