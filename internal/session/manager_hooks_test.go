package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestCreateFailsWhenSessionPreCreateDenied(t *testing.T) {
	t.Parallel()

	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "deny-create",
			Event:        hookspkg.HookSessionPreCreate,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"deny-create": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.SessionPreCreatePayload) (hookspkg.SessionCreatePatch, error) {
				return hookspkg.SessionCreatePatch{
					ControlPatch: hookspkg.ControlPatch{
						Deny:       true,
						DenyReason: "blocked",
					},
				}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	_, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want pre-create denial")
	}
	if len(h.manager.List()) != 0 {
		t.Fatalf("List() = %d active sessions, want 0", len(h.manager.List()))
	}
	if got := h.notifier.createdCount(); got != 0 {
		t.Fatalf("created notifications = %d, want 0", got)
	}
}

func TestCreateUsesPatchedSessionPreCreatePayload(t *testing.T) {
	t.Parallel()

	const patchedName = "patched-session"
	sessionName := patchedName
	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "patch-create",
			Event:        hookspkg.HookSessionPreCreate,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"patch-create": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.SessionPreCreatePayload) (hookspkg.SessionCreatePatch, error) {
				sessionType := string(SessionTypeDream)
				return hookspkg.SessionCreatePatch{
					SessionName: &sessionName,
					SessionType: &sessionType,
				}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "original",
		Workspace: h.workspaceID,
		Type:      SessionTypeUser,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := session.Info().Name; got != patchedName {
		t.Fatalf("session name = %q, want %q", got, patchedName)
	}
	if got := session.Info().Type; got != SessionTypeDream {
		t.Fatalf("session type = %q, want %q", got, SessionTypeDream)
	}
	if got := h.driver.startCalls[0].Permissions; got != aghconfig.PermissionModeApproveAll {
		t.Fatalf("start permissions = %q, want %q", got, aghconfig.PermissionModeApproveAll)
	}
}

func TestPostCreateHookFiresAfterSessionActive(t *testing.T) {
	t.Parallel()

	payloadCh := make(chan hookspkg.SessionPostCreatePayload, 1)
	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "observe-post-create",
			Event:        hookspkg.HookSessionPostCreate,
			Mode:         hookspkg.HookModeAsync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"observe-post-create": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePatch, error) {
				payloadCh <- payload
				return hookspkg.SessionPostCreatePatch{}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	select {
	case payload := <-payloadCh:
		if payload.SessionID != session.ID {
			t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, session.ID)
		}
		if payload.State != string(StateActive) {
			t.Fatalf("payload.State = %q, want %q", payload.State, StateActive)
		}
		if payload.ACPSessionID == "" {
			t.Fatal("payload.ACPSessionID = empty, want active ACP session id")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for session.post_create hook")
	}
}

func TestResumeUsesPatchedPreResumePayloadAndFiresPostResume(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	const patchedName = "resumed-patched"
	postResumeCh := make(chan hookspkg.SessionPostResumePayload, 1)
	dispatcher := &spyHookDispatcher{
		dispatchSessionPreResumeFn: func(_ context.Context, payload hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error) {
			payload.SessionName = patchedName
			return payload, nil
		},
		dispatchSessionPostResumeFn: func(_ context.Context, payload hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error) {
			postResumeCh <- payload
			return payload, nil
		},
	}

	h.manager = newManagerWithHarness(t, h, WithHookDispatcher(dispatcher))
	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := resumed.Info().Name; got != patchedName {
		t.Fatalf("resumed name = %q, want %q", got, patchedName)
	}

	select {
	case payload := <-postResumeCh:
		if payload.SessionID != resumed.ID {
			t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, resumed.ID)
		}
		if payload.State != string(StateActive) {
			t.Fatalf("payload.State = %q, want %q", payload.State, StateActive)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for session.post_resume hook")
	}
}

func TestPromptUsesPatchedInputMessage(t *testing.T) {
	t.Parallel()

	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "patch-input",
			Event:        hookspkg.HookInputPreSubmit,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"patch-input": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPatch, error) {
				message := "patched message"
				return hookspkg.InputPreSubmitPatch{Message: &message}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "original message")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	_ = collectEvents(t, eventsCh)

	if got := h.driver.promptCalls[0].Message; got != "patched message" {
		t.Fatalf("prompt message = %q, want %q", got, "patched message")
	}

	stored, err := session.recorderHandle().Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(stored) == 0 || !strings.Contains(stored[0].Content, `"text":"patched message"`) {
		t.Fatalf("stored user message content = %q, want patched text", stored[0].Content)
	}
}

func TestCreateUsesPatchedPrompt(t *testing.T) {
	t.Parallel()

	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "patch-prompt",
			Event:        hookspkg.HookPromptPostAssemble,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"patch-prompt": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.PromptPayload) (hookspkg.PromptPatch, error) {
				prompt := "patched system prompt"
				return hookspkg.PromptPatch{Prompt: &prompt}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].SystemPrompt; got != "patched system prompt" {
		t.Fatalf("start system prompt = %q, want %q", got, "patched system prompt")
	}
}

func TestAgentCrashedHookFiresOnProcessCrash(t *testing.T) {
	t.Parallel()

	payloadCh := make(chan hookspkg.AgentCrashedPayload, 1)
	hooks := newNativeHookDispatcher(t,
		[]hookspkg.HookDecl{{
			Name:         "observe-agent-crash",
			Event:        hookspkg.HookAgentCrashed,
			Mode:         hookspkg.HookModeAsync,
			ExecutorKind: hookspkg.HookExecutorNative,
		}},
		map[string]hookspkg.Executor{
			"observe-agent-crash": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, payload hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPatch, error) {
				payloadCh <- payload
				return hookspkg.AgentCrashedPatch{}, nil
			}),
		},
	)

	h := newHarness(t, WithHookDispatcher(hooks))
	session := createSession(t, h)

	h.driver.lastProcess().crash(errors.New("boom"), "stderr trace")
	waitForCondition(t, "session stop after crash", func() bool {
		_, ok := h.manager.Get(session.ID)
		return !ok
	})

	select {
	case payload := <-payloadCh:
		if payload.SessionID != session.ID {
			t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, session.ID)
		}
		if payload.Error != "boom" {
			t.Fatalf("payload.Error = %q, want %q", payload.Error, "boom")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for agent.crashed hook")
	}
}

func TestRecordEventDispatchesAroundPersistence(t *testing.T) {
	t.Parallel()

	order := make([]string, 0, 3)
	dispatcher := &spyHookDispatcher{
		dispatchEventPreRecordFn: func(_ context.Context, payload hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error) {
			order = append(order, "pre:"+payload.RecordType)
			return payload, nil
		},
		dispatchEventPostRecordFn: func(_ context.Context, payload hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error) {
			order = append(order, "post:"+payload.RecordType)
			return payload, nil
		},
	}
	h := newHarness(t, WithHookDispatcher(dispatcher))

	recorder := &orderedRecorder{
		onRecord: func(event store.SessionEvent) {
			order = append(order, "record:"+event.Type)
		},
	}
	now := h.manager.now()
	session := &Session{
		ID:          "sess-event",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		Workspace:   h.workspace,
		Type:        SessionTypeUser,
		State:       StateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		recorder:    recorder,
	}

	err := h.manager.recordEvent(testutil.Context(t), session, acp.AgentEvent{
		Type:      acp.EventTypeDone,
		TurnID:    "turn-1",
		Timestamp: now,
		Text:      "done",
	})
	if err != nil {
		t.Fatalf("recordEvent() error = %v", err)
	}

	want := []string{"pre:done", "record:done", "post:done"}
	if !testutil.EqualStringSlices(order, want) {
		t.Fatalf("dispatch order = %#v, want %#v", order, want)
	}
}

func newNativeHookDispatcher(t *testing.T, decls []hookspkg.HookDecl, executors map[string]hookspkg.Executor) *hookspkg.Hooks {
	t.Helper()

	hooks := hookspkg.NewHooks(
		hookspkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		hookspkg.WithNativeDeclarations(decls),
		hookspkg.WithExecutorResolver(func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
			executor := executors[strings.TrimSpace(decl.Name)]
			if executor == nil {
				return nil, errors.New("missing native executor")
			}
			return executor, nil
		}),
	)
	if err := hooks.Rebuild(testutil.Context(t)); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	t.Cleanup(hooks.Close)
	return hooks
}

type spyHookDispatcher struct {
	dispatchSessionPreCreateFn   func(context.Context, hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error)
	dispatchSessionPostCreateFn  func(context.Context, hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error)
	dispatchSessionPreResumeFn   func(context.Context, hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error)
	dispatchSessionPostResumeFn  func(context.Context, hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error)
	dispatchSessionPreStopFn     func(context.Context, hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error)
	dispatchSessionPostStopFn    func(context.Context, hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error)
	dispatchInputPreSubmitFn     func(context.Context, hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error)
	dispatchPromptPostAssembleFn func(context.Context, hookspkg.PromptPayload) (hookspkg.PromptPayload, error)
	dispatchEventPreRecordFn     func(context.Context, hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error)
	dispatchEventPostRecordFn    func(context.Context, hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error)
	dispatchAgentPreStartFn      func(context.Context, hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error)
	dispatchAgentSpawnedFn       func(context.Context, hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error)
	dispatchAgentCrashedFn       func(context.Context, hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error)
	dispatchAgentStoppedFn       func(context.Context, hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error)
}

func (s *spyHookDispatcher) DispatchSessionPreCreate(ctx context.Context, payload hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error) {
	if s.dispatchSessionPreCreateFn != nil {
		return s.dispatchSessionPreCreateFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchSessionPostCreate(ctx context.Context, payload hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error) {
	if s.dispatchSessionPostCreateFn != nil {
		return s.dispatchSessionPostCreateFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchSessionPreResume(ctx context.Context, payload hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error) {
	if s.dispatchSessionPreResumeFn != nil {
		return s.dispatchSessionPreResumeFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchSessionPostResume(ctx context.Context, payload hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error) {
	if s.dispatchSessionPostResumeFn != nil {
		return s.dispatchSessionPostResumeFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchSessionPreStop(ctx context.Context, payload hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error) {
	if s.dispatchSessionPreStopFn != nil {
		return s.dispatchSessionPreStopFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchSessionPostStop(ctx context.Context, payload hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error) {
	if s.dispatchSessionPostStopFn != nil {
		return s.dispatchSessionPostStopFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchInputPreSubmit(ctx context.Context, payload hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error) {
	if s.dispatchInputPreSubmitFn != nil {
		return s.dispatchInputPreSubmitFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchPromptPostAssemble(ctx context.Context, payload hookspkg.PromptPayload) (hookspkg.PromptPayload, error) {
	if s.dispatchPromptPostAssembleFn != nil {
		return s.dispatchPromptPostAssembleFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchEventPreRecord(ctx context.Context, payload hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error) {
	if s.dispatchEventPreRecordFn != nil {
		return s.dispatchEventPreRecordFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchEventPostRecord(ctx context.Context, payload hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error) {
	if s.dispatchEventPostRecordFn != nil {
		return s.dispatchEventPostRecordFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchAgentPreStart(ctx context.Context, payload hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error) {
	if s.dispatchAgentPreStartFn != nil {
		return s.dispatchAgentPreStartFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchAgentSpawned(ctx context.Context, payload hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error) {
	if s.dispatchAgentSpawnedFn != nil {
		return s.dispatchAgentSpawnedFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchAgentCrashed(ctx context.Context, payload hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error) {
	if s.dispatchAgentCrashedFn != nil {
		return s.dispatchAgentCrashedFn(ctx, payload)
	}
	return payload, nil
}

func (s *spyHookDispatcher) DispatchAgentStopped(ctx context.Context, payload hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error) {
	if s.dispatchAgentStoppedFn != nil {
		return s.dispatchAgentStoppedFn(ctx, payload)
	}
	return payload, nil
}

type orderedRecorder struct {
	onRecord func(store.SessionEvent)
	events   []store.SessionEvent
}

func (r *orderedRecorder) Record(_ context.Context, event store.SessionEvent) error {
	r.events = append(r.events, event)
	if r.onRecord != nil {
		r.onRecord(event)
	}
	return nil
}

func (r *orderedRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (r *orderedRecorder) Query(context.Context, store.EventQuery) ([]store.SessionEvent, error) {
	return append([]store.SessionEvent(nil), r.events...), nil
}

func (r *orderedRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (r *orderedRecorder) Close(context.Context) error {
	return nil
}
