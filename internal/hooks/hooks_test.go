package hooks

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHooksCreatesEmptyStartedRegistry(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(t)
	if hooks.snapshot == nil {
		t.Fatal("snapshot = nil, want initialized map")
	}
	if len(hooks.snapshot) != 0 {
		t.Fatalf("len(snapshot) = %d, want 0", len(hooks.snapshot))
	}
	if hooks.pool == nil {
		t.Fatal("pool = nil, want initialized pool")
	}
	if !hooks.pool.started {
		t.Fatal("pool.started = false, want true")
	}
	if got := hooks.version.Load(); got != 0 {
		t.Fatalf("version = %d, want 0", got)
	}
}

func TestHooksRebuildPopulatesSnapshot(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "native-post-create",
				Event:        HookSessionPostCreate,
				Mode:         HookModeAsync,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithConfigDeclarations([]HookDecl{
			testSubprocessDecl("config-input", HookInputPreSubmit),
		}),
		WithAgentDeclarations([]HookDecl{
			testSubprocessDecl("agent-message", HookMessageEnd),
		}),
		WithSkillDeclarations([]HookDecl{
			func() HookDecl {
				decl := testSubprocessDecl("skill-tool", HookToolPostCall)
				decl.SkillSource = HookSkillSourceUser
				return decl
			}(),
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"native-post-create": NewNativeExecutor(func(context.Context, RegisteredHook, []byte) ([]byte, error) {
				return nil, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	if got := len(hooks.snapshot[HookSessionPostCreate]); got != 1 {
		t.Fatalf("len(snapshot[session.post_create]) = %d, want 1", got)
	}
	if got := len(hooks.snapshot[HookInputPreSubmit]); got != 1 {
		t.Fatalf("len(snapshot[input.pre_submit]) = %d, want 1", got)
	}
	if got := len(hooks.snapshot[HookMessageEnd]); got != 1 {
		t.Fatalf("len(snapshot[message.end]) = %d, want 1", got)
	}
	if got := len(hooks.snapshot[HookToolPostCall]); got != 1 {
		t.Fatalf("len(snapshot[tool.post_call]) = %d, want 1", got)
	}
	if got := hooks.version.Load(); got != 1 {
		t.Fatalf("version = %d, want 1", got)
	}

	if hooks.snapshot[HookSessionPostCreate][0].Source != HookSourceNative {
		t.Fatalf("native source = %q, want %q", hooks.snapshot[HookSessionPostCreate][0].Source, HookSourceNative)
	}
	if hooks.snapshot[HookToolPostCall][0].Decl.SkillSource != HookSkillSourceUser {
		t.Fatalf("skill source = %q, want %q", hooks.snapshot[HookToolPostCall][0].Decl.SkillSource, HookSkillSourceUser)
	}
}

func TestHooksRebuildInvalidDeclarationKeepsOldSnapshot(t *testing.T) {
	t.Parallel()

	configDecls := []HookDecl{testSubprocessDecl("valid-input", HookInputPreSubmit)}
	hooks := newTestHooks(
		t,
		WithConfigDeclarationProvider(func(context.Context) ([]HookDecl, error) {
			return cloneHookDecls(configDecls), nil
		}),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("initial Rebuild() error = %v, want nil", err)
	}

	beforeVersion := hooks.version.Load()
	beforeFingerprint := hooks.fingerprint
	beforeHook := hooks.snapshot[HookInputPreSubmit][0]

	configDecls = []HookDecl{
		{
			Name:    "invalid-sync-delta",
			Event:   HookMessageDelta,
			Mode:    HookModeSync,
			Command: "/bin/sh",
			Args:    []string{"-c", "printf '{}'"},
		},
	}

	err := hooks.Rebuild(t.Context())
	if err == nil {
		t.Fatal("Rebuild() error = nil, want non-nil")
	}
	if hooks.version.Load() != beforeVersion {
		t.Fatalf("version = %d, want %d", hooks.version.Load(), beforeVersion)
	}
	if hooks.fingerprint != beforeFingerprint {
		t.Fatal("fingerprint changed after failed rebuild")
	}
	if hooks.snapshot[HookInputPreSubmit][0] != beforeHook {
		t.Fatal("snapshot changed after failed rebuild")
	}
}

func TestHooksRebuildUnchangedSkipsSwap(t *testing.T) {
	t.Parallel()

	decls := []HookDecl{testSubprocessDecl("stable-input", HookInputPreSubmit)}
	hooks := newTestHooks(t, WithConfigDeclarations(decls))
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("first Rebuild() error = %v, want nil", err)
	}

	beforeVersion := hooks.version.Load()
	beforeHook := hooks.snapshot[HookInputPreSubmit][0]

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("second Rebuild() error = %v, want nil", err)
	}

	if hooks.version.Load() != beforeVersion {
		t.Fatalf("version = %d, want %d", hooks.version.Load(), beforeVersion)
	}
	if hooks.snapshot[HookInputPreSubmit][0] != beforeHook {
		t.Fatal("snapshot swapped for unchanged declarations")
	}
}

func TestHooksRebuildBumpsVersionOnSwap(t *testing.T) {
	t.Parallel()

	configDecls := []HookDecl{testSubprocessDecl("v1-input", HookInputPreSubmit)}
	hooks := newTestHooks(
		t,
		WithConfigDeclarationProvider(func(context.Context) ([]HookDecl, error) {
			return cloneHookDecls(configDecls), nil
		}),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("first Rebuild() error = %v, want nil", err)
	}

	configDecls = []HookDecl{testSubprocessDecl("v2-input", HookInputPreSubmit)}
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("second Rebuild() error = %v, want nil", err)
	}

	if got := hooks.version.Load(); got != 2 {
		t.Fatalf("version = %d, want 2", got)
	}
	if hooks.snapshot[HookInputPreSubmit][0].Name != "v2-input" {
		t.Fatalf("snapshot hook = %q, want %q", hooks.snapshot[HookInputPreSubmit][0].Name, "v2-input")
	}
}

func TestHooksConcurrentRebuildAndDispatch(t *testing.T) {
	declsA := []HookDecl{
		{
			Name:         "native-a",
			Event:        HookInputPreSubmit,
			Mode:         HookModeSync,
			ExecutorKind: HookExecutorNative,
		},
	}
	declsB := []HookDecl{
		{
			Name:         "native-a",
			Event:        HookInputPreSubmit,
			Mode:         HookModeSync,
			ExecutorKind: HookExecutorNative,
		},
		{
			Name:         "native-b",
			Event:        HookInputPreSubmit,
			Mode:         HookModeSync,
			Priority:     200,
			PrioritySet:  true,
			ExecutorKind: HookExecutorNative,
		},
	}

	var seq atomic.Int64
	hooks := newTestHooks(
		t,
		WithNativeDeclarationProvider(func(context.Context) ([]HookDecl, error) {
			if seq.Add(1)%2 == 0 {
				return cloneHookDecls(declsA), nil
			}
			return cloneHookDecls(declsB), nil
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"native-a": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				msg := payload.Message + "a"
				return InputPreSubmitPatch{Message: &msg}, nil
			}),
			"native-b": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				msg := payload.Message + "b"
				return InputPreSubmitPatch{Message: &msg}, nil
			}),
		})),
	)

	var wg sync.WaitGroup
	errCh := make(chan error, 16)

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if err := hooks.Rebuild(context.Background()); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if _, err := hooks.DispatchInputPreSubmit(context.Background(), InputPreSubmitPayload{
					PayloadBase: PayloadBase{Event: HookInputPreSubmit},
					Message:     "seed-",
				}); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent hooks operation error = %v, want nil", err)
		}
	}
}

func TestDispatchInputPreSubmitRejectsNilHooksAndContext(t *testing.T) {
	t.Parallel()

	payload := InputPreSubmitPayload{
		PayloadBase: PayloadBase{Event: HookInputPreSubmit},
		Message:     "seed",
	}

	var nilHooks *Hooks
	if _, err := nilHooks.DispatchInputPreSubmit(context.Background(), payload); err == nil || !strings.Contains(err.Error(), "dispatcher is nil") {
		t.Fatalf("DispatchInputPreSubmit(nil hooks) error = %v, want nil dispatcher detail", err)
	}

	hooks := newTestHooks(t, WithConfigDeclarations([]HookDecl{
		testSubprocessDecl("input-hook", HookInputPreSubmit),
	}))
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	if _, err := hooks.DispatchInputPreSubmit(nilTestContext(), payload); err == nil || !strings.Contains(err.Error(), "dispatch context is nil") {
		t.Fatalf("DispatchInputPreSubmit(nil context) error = %v, want nil context detail", err)
	}
}

func nilTestContext() context.Context {
	var ctx context.Context
	return ctx
}

func TestDispatchInputPreSubmitReturnsOriginalPayloadWhenNoHooksMatch(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(t)
	payload := InputPreSubmitPayload{
		PayloadBase: PayloadBase{Event: HookInputPreSubmit},
		Message:     "unchanged",
	}

	got, err := hooks.DispatchInputPreSubmit(t.Context(), payload)
	if err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, payload) {
		t.Fatalf("DispatchInputPreSubmit() = %#v, want %#v", got, payload)
	}
}

func TestDispatchMethodsSmokeNoHooks(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(t)
	ctx := t.Context()

	if _, err := hooks.DispatchSessionPreCreate(ctx, SessionPreCreatePayload{PayloadBase: PayloadBase{Event: HookSessionPreCreate}}); err != nil {
		t.Fatalf("DispatchSessionPreCreate() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchSessionPostCreate(ctx, SessionPostCreatePayload{PayloadBase: PayloadBase{Event: HookSessionPostCreate}}); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchSessionPreResume(ctx, SessionPreResumePayload{PayloadBase: PayloadBase{Event: HookSessionPreResume}}); err != nil {
		t.Fatalf("DispatchSessionPreResume() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchSessionPostResume(ctx, SessionPostResumePayload{PayloadBase: PayloadBase{Event: HookSessionPostResume}}); err != nil {
		t.Fatalf("DispatchSessionPostResume() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchSessionPreStop(ctx, SessionPreStopPayload{PayloadBase: PayloadBase{Event: HookSessionPreStop}}); err != nil {
		t.Fatalf("DispatchSessionPreStop() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchSessionPostStop(ctx, SessionPostStopPayload{PayloadBase: PayloadBase{Event: HookSessionPostStop}}); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchInputPreSubmit(ctx, InputPreSubmitPayload{PayloadBase: PayloadBase{Event: HookInputPreSubmit}}); err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchPromptPostAssemble(ctx, PromptPayload{PayloadBase: PayloadBase{Event: HookPromptPostAssemble}}); err != nil {
		t.Fatalf("DispatchPromptPostAssemble() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchEventPreRecord(ctx, EventPreRecordPayload{PayloadBase: PayloadBase{Event: HookEventPreRecord}}); err != nil {
		t.Fatalf("DispatchEventPreRecord() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchEventPostRecord(ctx, EventPostRecordPayload{PayloadBase: PayloadBase{Event: HookEventPostRecord}}); err != nil {
		t.Fatalf("DispatchEventPostRecord() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchAgentPreStart(ctx, AgentPreStartPayload{PayloadBase: PayloadBase{Event: HookAgentPreStart}}); err != nil {
		t.Fatalf("DispatchAgentPreStart() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchAgentSpawned(ctx, AgentSpawnedPayload{PayloadBase: PayloadBase{Event: HookAgentSpawned}}); err != nil {
		t.Fatalf("DispatchAgentSpawned() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchAgentCrashed(ctx, AgentCrashedPayload{PayloadBase: PayloadBase{Event: HookAgentCrashed}}); err != nil {
		t.Fatalf("DispatchAgentCrashed() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchAgentStopped(ctx, AgentStoppedPayload{PayloadBase: PayloadBase{Event: HookAgentStopped}}); err != nil {
		t.Fatalf("DispatchAgentStopped() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchTurnStart(ctx, TurnStartPayload{PayloadBase: PayloadBase{Event: HookTurnStart}}); err != nil {
		t.Fatalf("DispatchTurnStart() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchTurnEnd(ctx, TurnEndPayload{PayloadBase: PayloadBase{Event: HookTurnEnd}}); err != nil {
		t.Fatalf("DispatchTurnEnd() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchMessageStart(ctx, MessageStartPayload{PayloadBase: PayloadBase{Event: HookMessageStart}}); err != nil {
		t.Fatalf("DispatchMessageStart() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchMessageDelta(ctx, MessageDeltaPayload{PayloadBase: PayloadBase{Event: HookMessageDelta}}); err != nil {
		t.Fatalf("DispatchMessageDelta() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchMessageEnd(ctx, MessageEndPayload{PayloadBase: PayloadBase{Event: HookMessageEnd}}); err != nil {
		t.Fatalf("DispatchMessageEnd() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchToolPreCall(ctx, ToolPreCallPayload{PayloadBase: PayloadBase{Event: HookToolPreCall}}); err != nil {
		t.Fatalf("DispatchToolPreCall() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchToolPostCall(ctx, ToolPostCallPayload{PayloadBase: PayloadBase{Event: HookToolPostCall}}); err != nil {
		t.Fatalf("DispatchToolPostCall() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchToolPostError(ctx, ToolPostErrorPayload{PayloadBase: PayloadBase{Event: HookToolPostError}}); err != nil {
		t.Fatalf("DispatchToolPostError() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchPermissionRequest(ctx, PermissionRequestPayload{PayloadBase: PayloadBase{Event: HookPermissionRequest}}); err != nil {
		t.Fatalf("DispatchPermissionRequest() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchPermissionResolved(ctx, PermissionResolvedPayload{PayloadBase: PayloadBase{Event: HookPermissionResolved}}); err != nil {
		t.Fatalf("DispatchPermissionResolved() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchPermissionDenied(ctx, PermissionDeniedPayload{PayloadBase: PayloadBase{Event: HookPermissionDenied}}); err != nil {
		t.Fatalf("DispatchPermissionDenied() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchContextPreCompact(ctx, ContextPreCompactPayload{PayloadBase: PayloadBase{Event: HookContextPreCompact}}); err != nil {
		t.Fatalf("DispatchContextPreCompact() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchContextPostCompact(ctx, ContextPostCompactPayload{PayloadBase: PayloadBase{Event: HookContextPostCompact}}); err != nil {
		t.Fatalf("DispatchContextPostCompact() error = %v, want nil", err)
	}
}

func TestDispatchInputPreSubmitAppliesMatchingHooksInOrder(t *testing.T) {
	t.Parallel()

	var seen []string
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "append-a",
				Event:        HookInputPreSubmit,
				Mode:         HookModeSync,
				Priority:     200,
				PrioritySet:  true,
				ExecutorKind: HookExecutorNative,
			},
			{
				Name:         "append-b",
				Event:        HookInputPreSubmit,
				Mode:         HookModeSync,
				Priority:     100,
				PrioritySet:  true,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"append-a": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				seen = append(seen, payload.Message)
				msg := payload.Message + "a"
				return InputPreSubmitPatch{Message: &msg}, nil
			}),
			"append-b": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				seen = append(seen, payload.Message)
				msg := payload.Message + "b"
				return InputPreSubmitPatch{Message: &msg}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	result, err := hooks.DispatchInputPreSubmit(t.Context(), InputPreSubmitPayload{
		PayloadBase: PayloadBase{Event: HookInputPreSubmit},
		Message:     "seed-",
	})
	if err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v, want nil", err)
	}
	if result.Message != "seed-ab" {
		t.Fatalf("result.Message = %q, want %q", result.Message, "seed-ab")
	}
	if got, want := len(seen), 2; got != want {
		t.Fatalf("len(seen) = %d, want %d", got, want)
	}
	if seen[0] != "seed-" || seen[1] != "seed-a" {
		t.Fatalf("seen = %#v, want [seed- seed-a]", seen)
	}
}

func TestDispatchSessionPreCreateAppliesPatch(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "session-precreate",
				Event:        HookSessionPreCreate,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{AgentName: "codex"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"session-precreate": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ SessionPreCreatePayload) (SessionCreatePatch, error) {
				name := "renamed"
				workspace := "/tmp/next"
				return SessionCreatePatch{
					SessionName: &name,
					Workspace:   &workspace,
				}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	result, err := hooks.DispatchSessionPreCreate(t.Context(), SessionPreCreatePayload{
		PayloadBase: PayloadBase{Event: HookSessionPreCreate},
		SessionContext: SessionContext{
			AgentName:   "codex",
			SessionName: "old",
			Workspace:   "/tmp/old",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPreCreate() error = %v, want nil", err)
	}
	if result.SessionName != "renamed" {
		t.Fatalf("result.SessionName = %q, want %q", result.SessionName, "renamed")
	}
	if result.Workspace != "/tmp/next" {
		t.Fatalf("result.Workspace = %q, want %q", result.Workspace, "/tmp/next")
	}
}

func TestDispatchPromptPostAssembleAppliesPatch(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "prompt-post-assemble",
				Event:        HookPromptPostAssemble,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{InputClass: "chat"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"prompt-post-assemble": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ PromptPayload) (PromptPatch, error) {
				prompt := "patched"
				return PromptPatch{
					Prompt:        &prompt,
					ContextBlocks: []ContextBlock{{Kind: "note", Text: "ctx", Metadata: map[string]string{"k": "v"}}},
				}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	result, err := hooks.DispatchPromptPostAssemble(t.Context(), PromptPayload{
		PayloadBase: PayloadBase{Event: HookPromptPostAssemble},
		InputClass:  "chat",
		Prompt:      "original",
	})
	if err != nil {
		t.Fatalf("DispatchPromptPostAssemble() error = %v, want nil", err)
	}
	if result.Prompt != "patched" {
		t.Fatalf("result.Prompt = %q, want %q", result.Prompt, "patched")
	}
	if got := len(result.ContextBlocks); got != 1 {
		t.Fatalf("len(result.ContextBlocks) = %d, want 1", got)
	}
	if result.ContextBlocks[0].Metadata["k"] != "v" {
		t.Fatalf("metadata = %#v, want key k", result.ContextBlocks[0].Metadata)
	}
}

func TestDispatchEventPreRecordRunsAsyncHook(t *testing.T) {
	t.Parallel()

	called := make(chan string, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "event-observer",
				Event:        HookEventPreRecord,
				Mode:         HookModeAsync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{ACPEventType: "agent_message"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"event-observer": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload EventPreRecordPayload) (EventPreRecordPatch, error) {
				called <- payload.RecordType
				return EventPreRecordPatch{}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	if _, err := hooks.DispatchEventPreRecord(t.Context(), EventPreRecordPayload{
		PayloadBase: PayloadBase{Event: HookEventPreRecord},
		RecordType:  "agent_message",
	}); err != nil {
		t.Fatalf("DispatchEventPreRecord() error = %v, want nil", err)
	}

	select {
	case got := <-called:
		if got != "agent_message" {
			t.Fatalf("async event payload = %q, want %q", got, "agent_message")
		}
	case <-time.After(time.Second):
		t.Fatal("async event hook was not called")
	}
}

func TestDispatchAgentHooksApplyPatches(t *testing.T) {
	t.Parallel()

	spawned := make(chan struct{}, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "agent-prestart",
				Event:        HookAgentPreStart,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{AgentName: "codex"},
			},
			{
				Name:         "agent-spawned",
				Event:        HookAgentSpawned,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{AgentName: "codex"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"agent-prestart": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ AgentPreStartPayload) (AgentStartPatch, error) {
				command := "/bin/next"
				cwd := "/tmp/next"
				return AgentStartPatch{
					Command: &command,
					Args:    []string{"--next"},
					Cwd:     &cwd,
				}, nil
			}),
			"agent-spawned": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ AgentSpawnedPayload) (AgentSpawnedPatch, error) {
				spawned <- struct{}{}
				return AgentSpawnedPatch{}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	preStart, err := hooks.DispatchAgentPreStart(t.Context(), AgentPreStartPayload{
		PayloadBase:    PayloadBase{Event: HookAgentPreStart},
		SessionContext: SessionContext{AgentName: "codex"},
		Command:        "/bin/original",
		Cwd:            "/tmp/original",
	})
	if err != nil {
		t.Fatalf("DispatchAgentPreStart() error = %v, want nil", err)
	}
	if preStart.Command != "/bin/next" {
		t.Fatalf("preStart.Command = %q, want %q", preStart.Command, "/bin/next")
	}
	if preStart.Cwd != "/tmp/next" {
		t.Fatalf("preStart.Cwd = %q, want %q", preStart.Cwd, "/tmp/next")
	}
	if got := len(preStart.Args); got != 1 {
		t.Fatalf("len(preStart.Args) = %d, want 1", got)
	}

	if _, err := hooks.DispatchAgentSpawned(t.Context(), AgentSpawnedPayload{
		PayloadBase:    PayloadBase{Event: HookAgentSpawned},
		SessionContext: SessionContext{AgentName: "codex"},
	}); err != nil {
		t.Fatalf("DispatchAgentSpawned() error = %v, want nil", err)
	}

	select {
	case <-spawned:
	case <-time.After(time.Second):
		t.Fatal("spawned hook was not called")
	}
}

func TestDispatchTurnAndMessageHooksApplyPatches(t *testing.T) {
	t.Parallel()

	turnCalled := make(chan struct{}, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "turn-start",
				Event:        HookTurnStart,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{InputClass: "chat"},
			},
			{
				Name:         "message-start",
				Event:        HookMessageStart,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{MessageRole: "assistant"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"turn-start": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ TurnStartPayload) (TurnStartPatch, error) {
				turnCalled <- struct{}{}
				return TurnStartPatch{}, nil
			}),
			"message-start": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ MessageStartPayload) (MessageStartPatch, error) {
				role := "tool"
				delta := "replacement"
				text := "patched"
				return MessageStartPatch{
					Role:      &role,
					DeltaType: &delta,
					Text:      &text,
				}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	if _, err := hooks.DispatchTurnStart(t.Context(), TurnStartPayload{
		PayloadBase: PayloadBase{Event: HookTurnStart},
		InputClass:  "chat",
	}); err != nil {
		t.Fatalf("DispatchTurnStart() error = %v, want nil", err)
	}
	select {
	case <-turnCalled:
	case <-time.After(time.Second):
		t.Fatal("turn-start hook was not called")
	}

	result, err := hooks.DispatchMessageStart(t.Context(), MessageStartPayload{
		PayloadBase: PayloadBase{Event: HookMessageStart},
		Role:        "assistant",
	})
	if err != nil {
		t.Fatalf("DispatchMessageStart() error = %v, want nil", err)
	}
	if result.Role != "tool" || result.DeltaType != "replacement" || result.Text != "patched" {
		t.Fatalf("message result = %#v, want patched role/delta/text", result)
	}
}

func TestDispatchToolHooksApplyPatches(t *testing.T) {
	t.Parallel()

	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "tool-pre",
				Event:        HookToolPreCall,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{ToolName: "read"},
			},
			{
				Name:         "tool-post",
				Event:        HookToolPostCall,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{ToolName: "read"},
			},
			{
				Name:         "tool-error",
				Event:        HookToolPostError,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{ToolName: "read"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"tool-pre": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ ToolPreCallPayload) (ToolCallPatch, error) {
				name := "write"
				namespace := "fs"
				readOnly := false
				return ToolCallPatch{
					ToolName:      &name,
					ToolNamespace: &namespace,
					ReadOnly:      &readOnly,
					ToolInput:     []byte(`{"patched":true}`),
				}, nil
			}),
			"tool-post": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ ToolPostCallPayload) (ToolResultPatch, error) {
				title := "patched-result"
				return ToolResultPatch{
					Title:      &title,
					ToolResult: []byte(`{"ok":true}`),
				}, nil
			}),
			"tool-error": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ ToolPostErrorPayload) (ToolPostErrorPatch, error) {
				title := "patched-error"
				errText := "patched failure"
				return ToolPostErrorPatch{
					Title: &title,
					Error: &errText,
				}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	pre, err := hooks.DispatchToolPreCall(t.Context(), ToolPreCallPayload{
		PayloadBase: PayloadBase{Event: HookToolPreCall},
		ToolCallRef: ToolCallRef{ToolName: "read"},
	})
	if err != nil {
		t.Fatalf("DispatchToolPreCall() error = %v, want nil", err)
	}
	if pre.ToolName != "write" || pre.ToolNamespace != "fs" || pre.ReadOnly {
		t.Fatalf("pre = %#v, want patched tool identity", pre)
	}
	if string(pre.ToolInput) != `{"patched":true}` {
		t.Fatalf("pre.ToolInput = %s, want patched json", pre.ToolInput)
	}

	post, err := hooks.DispatchToolPostCall(t.Context(), ToolPostCallPayload{
		PayloadBase: PayloadBase{Event: HookToolPostCall},
		ToolCallRef: ToolCallRef{ToolName: "read"},
	})
	if err != nil {
		t.Fatalf("DispatchToolPostCall() error = %v, want nil", err)
	}
	if post.Title != "patched-result" || string(post.ToolResult) != `{"ok":true}` {
		t.Fatalf("post = %#v, want patched title/result", post)
	}

	postErr, err := hooks.DispatchToolPostError(t.Context(), ToolPostErrorPayload{
		PayloadBase: PayloadBase{Event: HookToolPostError},
		ToolCallRef: ToolCallRef{ToolName: "read"},
	})
	if err != nil {
		t.Fatalf("DispatchToolPostError() error = %v, want nil", err)
	}
	if postErr.Title != "patched-error" || postErr.Error != "patched failure" {
		t.Fatalf("postErr = %#v, want patched title/error", postErr)
	}
}

func TestDispatchPermissionAndContextHooksApplyPatches(t *testing.T) {
	t.Parallel()

	resolvedCalled := make(chan struct{}, 1)
	deniedCalled := make(chan struct{}, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "permission-request",
				Event:        HookPermissionRequest,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{DecisionClass: "tool"},
			},
			{
				Name:         "permission-resolved",
				Event:        HookPermissionResolved,
				Mode:         HookModeAsync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{DecisionClass: "tool"},
			},
			{
				Name:         "permission-denied",
				Event:        HookPermissionDenied,
				Mode:         HookModeAsync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{DecisionClass: "tool"},
			},
			{
				Name:         "context-pre",
				Event:        HookContextPreCompact,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
				Matcher:      HookMatcher{CompactionReason: "token_limit"},
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"permission-request": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ PermissionRequestPayload) (PermissionRequestPatch, error) {
				decision := "allow-once"
				decisionClass := "tool-patched"
				return PermissionRequestPatch{
					Decision:      &decision,
					DecisionClass: &decisionClass,
				}, nil
			}),
			"permission-resolved": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ PermissionResolvedPayload) (PermissionResolvedPatch, error) {
				resolvedCalled <- struct{}{}
				return PermissionResolvedPatch{}, nil
			}),
			"permission-denied": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ PermissionDeniedPayload) (PermissionDeniedPatch, error) {
				deniedCalled <- struct{}{}
				return PermissionDeniedPatch{}, nil
			}),
			"context-pre": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ ContextPreCompactPayload) (ContextPreCompactPatch, error) {
				reason := "manual"
				strategy := "summarize"
				return ContextPreCompactPatch{
					Reason:        &reason,
					Strategy:      &strategy,
					ContextBlocks: []ContextBlock{{Kind: "summary", Text: "patched"}},
				}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	permission, err := hooks.DispatchPermissionRequest(t.Context(), PermissionRequestPayload{
		PayloadBase:    PayloadBase{Event: HookPermissionRequest},
		DecisionClass:  "tool",
		Decision:       "allow",
		SessionContext: SessionContext{},
	})
	if err != nil {
		t.Fatalf("DispatchPermissionRequest() error = %v, want nil", err)
	}
	if permission.Decision != "allow-once" || permission.DecisionClass != "tool-patched" {
		t.Fatalf("permission = %#v, want patched decision fields", permission)
	}

	if _, err := hooks.DispatchPermissionResolved(t.Context(), PermissionResolvedPayload{
		PayloadBase:   PayloadBase{Event: HookPermissionResolved},
		DecisionClass: "tool",
	}); err != nil {
		t.Fatalf("DispatchPermissionResolved() error = %v, want nil", err)
	}
	if _, err := hooks.DispatchPermissionDenied(t.Context(), PermissionDeniedPayload{
		PayloadBase:   PayloadBase{Event: HookPermissionDenied},
		DecisionClass: "tool",
	}); err != nil {
		t.Fatalf("DispatchPermissionDenied() error = %v, want nil", err)
	}

	select {
	case <-resolvedCalled:
	case <-time.After(time.Second):
		t.Fatal("permission-resolved async hook was not called")
	}
	select {
	case <-deniedCalled:
	case <-time.After(time.Second):
		t.Fatal("permission-denied async hook was not called")
	}

	contextPayload, err := hooks.DispatchContextPreCompact(t.Context(), ContextPreCompactPayload{
		PayloadBase: PayloadBase{Event: HookContextPreCompact},
		Reason:      "token_limit",
	})
	if err != nil {
		t.Fatalf("DispatchContextPreCompact() error = %v, want nil", err)
	}
	if contextPayload.Reason != "manual" || contextPayload.Strategy != "summarize" {
		t.Fatalf("contextPayload = %#v, want patched reason/strategy", contextPayload)
	}
	if got := len(contextPayload.ContextBlocks); got != 1 {
		t.Fatalf("len(contextPayload.ContextBlocks) = %d, want 1", got)
	}
}

func TestHooksDispatchSessionPostCreate(t *testing.T) {
	t.Parallel()

	called := make(chan SessionLifecyclePayload, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "observe-create",
				Event:        HookSessionPostCreate,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"observe-create": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload SessionLifecyclePayload) (SessionPostCreatePatch, error) {
				called <- payload
				return SessionPostCreatePatch{}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	_, err := hooks.DispatchSessionPostCreate(t.Context(), SessionPostCreatePayload{
		PayloadBase: PayloadBase{
			Event:     HookSessionPostCreate,
			Timestamp: time.Unix(123, 0).UTC(),
		},
		SessionContext: SessionContext{
			SessionID:   "sess-created",
			SessionName: "demo",
			AgentName:   "codex",
			WorkspaceID: "ws-1",
			Workspace:   "/tmp/ws",
			SessionType: "user",
			State:       "active",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v, want nil", err)
	}

	select {
	case payload := <-called:
		if payload.Event != HookSessionPostCreate {
			t.Fatalf("payload.Event = %q, want %q", payload.Event, HookSessionPostCreate)
		}
		if payload.SessionID != "sess-created" {
			t.Fatalf("payload.SessionID = %q, want %q", payload.SessionID, "sess-created")
		}
		if payload.Timestamp != time.Unix(123, 0).UTC() {
			t.Fatalf("payload.Timestamp = %s, want fixed clock", payload.Timestamp)
		}
	case <-time.After(time.Second):
		t.Fatal("post-create hook was not called")
	}
}

func TestHooksDispatchSessionPostStop(t *testing.T) {
	t.Parallel()

	called := make(chan SessionLifecyclePayload, 1)
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "observe-stop",
				Event:        HookSessionPostStop,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"observe-stop": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload SessionLifecyclePayload) (SessionPostStopPatch, error) {
				called <- payload
				return SessionPostStopPatch{}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	_, err := hooks.DispatchSessionPostStop(t.Context(), SessionPostStopPayload{
		PayloadBase: PayloadBase{Event: HookSessionPostStop},
		SessionContext: SessionContext{
			SessionID:   "sess-stopped",
			SessionName: "demo",
			AgentName:   "codex",
			SessionType: "system",
			State:       "stopped",
		},
	})
	if err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v, want nil", err)
	}

	select {
	case payload := <-called:
		if payload.Event != HookSessionPostStop {
			t.Fatalf("payload.Event = %q, want %q", payload.Event, HookSessionPostStop)
		}
		if payload.State != "stopped" {
			t.Fatalf("payload.State = %q, want %q", payload.State, "stopped")
		}
	case <-time.After(time.Second):
		t.Fatal("post-stop hook was not called")
	}
}

func TestHooksCloseDrainsAsyncPool(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "async-input",
				Event:        HookInputPreSubmit,
				Mode:         HookModeAsync,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"async-input": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, _ InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				close(started)
				<-release
				close(done)
				return InputPreSubmitPatch{}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	if _, err := hooks.DispatchInputPreSubmit(t.Context(), InputPreSubmitPayload{
		PayloadBase: PayloadBase{Event: HookInputPreSubmit},
		Message:     "seed",
	}); err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v, want nil", err)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("async hook did not start")
	}

	closed := make(chan struct{})
	go func() {
		hooks.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close() returned before async hook completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("async hook did not finish")
	}

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close() did not return after async hook completion")
	}
}

func TestNewHooksAppliesOptionsAndDefaultResolver(t *testing.T) {
	t.Parallel()

	logger := discardPoolLogger()
	hooks := newTestHooks(
		t,
		WithLogger(logger),
		WithAsyncWorkerCount(2),
		WithAsyncQueueCapacity(3),
		WithAsyncDrainTimeout(4*time.Second),
	)

	if hooks.logger != logger {
		t.Fatal("logger option was not applied")
	}
	if hooks.pool.workerCount != 2 {
		t.Fatalf("workerCount = %d, want 2", hooks.pool.workerCount)
	}
	if hooks.pool.queueCapacity != 3 {
		t.Fatalf("queueCapacity = %d, want 3", hooks.pool.queueCapacity)
	}
	if hooks.pool.drainTimeout != 4*time.Second {
		t.Fatalf("drainTimeout = %s, want 4s", hooks.pool.drainTimeout)
	}
	if hooks.Version() != 0 {
		t.Fatalf("Version() = %d, want 0", hooks.Version())
	}

	executor, err := defaultExecutorResolver(HookDecl{
		Name:         "wasm-stub",
		ExecutorKind: HookExecutorWASM,
	})
	if err != nil {
		t.Fatalf("defaultExecutorResolver(wasm) error = %v, want nil", err)
	}
	if executor.Kind() != HookExecutorWASM {
		t.Fatalf("executor.Kind() = %q, want %q", executor.Kind(), HookExecutorWASM)
	}
	if _, err := defaultExecutorResolver(HookDecl{
		Name:         "native-missing",
		ExecutorKind: HookExecutorNative,
	}); err == nil {
		t.Fatal("defaultExecutorResolver(native) error = nil, want non-nil")
	}

	hooks.OnAgentEvent(t.Context(), "session-id", struct{ Type string }{Type: "done"})
}

func newTestHooks(t *testing.T, opts ...Option) *Hooks {
	t.Helper()

	hooks := NewHooks(opts...)
	t.Cleanup(hooks.Close)
	return hooks
}

func testExecutorResolver(native map[string]Executor) ExecutorResolver {
	return func(decl HookDecl) (Executor, error) {
		if decl.ExecutorKind == HookExecutorNative {
			executor := native[decl.Name]
			if executor == nil {
				return nil, errors.New("missing native executor")
			}
			return executor, nil
		}
		return defaultExecutorResolver(decl)
	}
}

func testSubprocessDecl(name string, event HookEvent) HookDecl {
	return HookDecl{
		Name:    name,
		Event:   event,
		Mode:    HookModeAsync,
		Command: "/bin/sh",
		Args:    []string{"-c", "printf '{}'"},
	}
}
