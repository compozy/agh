package daemon

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestHooksNotifierDispatchesLifecycleAgentAndStreamEvents(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	var order []string
	runtime := &fakeHookRuntime{
		onRebuild: func(context.Context) error {
			order = append(order, "rebuild")
			return nil
		},
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
		onAgentEvent: func(context.Context, string, any) {
			order = append(order, "hook-agent")
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

	if _, err := notifier.DispatchSessionPostCreate(testutil.Context(t), hookspkg.SessionPostCreatePayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, fixedNow))); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := notifier.DispatchSessionPostStop(testutil.Context(t), hookspkg.SessionPostStopPayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, fixedNow))); err != nil {
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
	notifier.OnAgentEvent(testutil.Context(t), "sess-created", struct{ Type string }{Type: "done"})

	wantOrder := []string{
		"rebuild",
		"create",
		"rebuild",
		"stop",
		"turn-start",
		"message-start",
		"message-delta",
		"message-end",
		"turn-end",
		"context-pre",
		"context-post",
		"hook-agent",
	}
	if !testutil.EqualStringSlices(order, wantOrder) {
		t.Fatalf("dispatch order = %#v, want %#v", order, wantOrder)
	}
	if got, want := agentEvents.events, []string{"agent"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("agent event notifier events = %#v, want %#v", got, want)
	}
}

func TestDaemonNativeHooksDriveObserverAndDreamCallbacks(t *testing.T) {
	t.Parallel()

	observer := &spyLifecycleObserver{}
	dream := &spyDreamRuntime{}
	decls, executors := daemonNativeHooks(observer, dream)
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

	if _, err := hooks.DispatchSessionPostCreate(testutil.Context(t), hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, fixedNow)); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := hooks.DispatchSessionPostStop(testutil.Context(t), hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, fixedNow)); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
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

	resolved := workspaceResolvedForTest("ws-1", "/tmp/ws-1")
	scoped := scopeWorkspaceHookDecls([]hookspkg.HookDecl{original}, resolved)
	if len(scoped) != 1 {
		t.Fatalf("len(scoped) = %d, want 1", len(scoped))
	}
	if scoped[0].Matcher.WorkspaceID != resolved.ID {
		t.Fatalf("scoped WorkspaceID = %q, want %q", scoped[0].Matcher.WorkspaceID, resolved.ID)
	}
	if scoped[0].Matcher.WorkspaceRoot != resolved.RootDir {
		t.Fatalf("scoped WorkspaceRoot = %q, want %q", scoped[0].Matcher.WorkspaceRoot, resolved.RootDir)
	}
	if original.Matcher.WorkspaceID != "" || original.Matcher.WorkspaceRoot != "" {
		t.Fatalf("original matcher workspace fields were mutated: %#v", original.Matcher)
	}

	if got := cloneStringMap(nil); got != nil {
		t.Fatalf("cloneStringMap(nil) = %#v, want nil", got)
	}
}

func TestDispatchRuntimeAndExecutorResolvers(t *testing.T) {
	t.Parallel()

	notifier := newHooksNotifier(discardLogger(), func() time.Time { return time.Date(2026, 4, 9, 16, 0, 0, 0, time.UTC) })
	payload, err := dispatchRuntime(notifier, nil, hookspkg.HookSessionPostCreate, "seed", false, func(_ hookRuntime, _ context.Context, item string) (string, error) {
		return item + "-unused", nil
	})
	if err != nil {
		t.Fatalf("dispatchRuntime(nil runtime) error = %v, want nil", err)
	}
	if payload != "seed" {
		t.Fatalf("dispatchRuntime(nil runtime) payload = %q, want %q", payload, "seed")
	}

	var rebuildCalls int
	runtime := &fakeHookRuntime{
		onRebuild: func(context.Context) error {
			rebuildCalls++
			return errors.New("rebuild failed")
		},
	}
	notifier.setRuntime(runtime, nil)

	result, err := dispatchRuntime(notifier, context.Background(), hookspkg.HookEventPreRecord, "seed", false, func(_ hookRuntime, _ context.Context, item string) (string, error) {
		return item + "-ok", nil
	})
	if err != nil {
		t.Fatalf("dispatchRuntime(rebuild false) error = %v, want nil", err)
	}
	if result != "seed-ok" {
		t.Fatalf("dispatchRuntime(rebuild false) result = %q, want %q", result, "seed-ok")
	}
	if rebuildCalls != 0 {
		t.Fatalf("rebuildCalls = %d, want 0 when rebuild=false", rebuildCalls)
	}

	result, err = dispatchRuntime(notifier, context.Background(), hookspkg.HookSessionPostCreate, "seed", true, func(_ hookRuntime, _ context.Context, item string) (string, error) {
		return item + "-after-rebuild", nil
	})
	if err != nil {
		t.Fatalf("dispatchRuntime(rebuild true) error = %v, want nil", err)
	}
	if result != "seed-after-rebuild" {
		t.Fatalf("dispatchRuntime(rebuild true) result = %q, want %q", result, "seed-after-rebuild")
	}
	if rebuildCalls != 1 {
		t.Fatalf("rebuildCalls = %d, want 1", rebuildCalls)
	}

	_, err = dispatchRuntime(notifier, nil, hookspkg.HookSessionPostCreate, "seed", false, func(_ hookRuntime, _ context.Context, item string) (string, error) {
		return item, nil
	})
	if err == nil || !strings.Contains(err.Error(), "requires a non-nil context") {
		t.Fatalf("dispatchRuntime(nil context) error = %v, want non-nil context detail", err)
	}

	subprocessExecutor, err := defaultDaemonExecutorResolver(hookspkg.HookDecl{
		Name:         "subprocess",
		ExecutorKind: hookspkg.HookExecutorSubprocess,
		Command:      "/bin/sh",
	})
	if err != nil {
		t.Fatalf("defaultDaemonExecutorResolver(subprocess) error = %v, want nil", err)
	}
	if subprocessExecutor.Kind() != hookspkg.HookExecutorSubprocess {
		t.Fatalf("subprocess executor kind = %q, want %q", subprocessExecutor.Kind(), hookspkg.HookExecutorSubprocess)
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
		"bound": hookspkg.NewTypedNativeExecutor(func(context.Context, hookspkg.RegisteredHook, hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePatch, error) {
			return hookspkg.SessionPostCreatePatch{}, nil
		}),
	})
	nativeExecutor, err := resolver(hookspkg.HookDecl{Name: "bound", ExecutorKind: hookspkg.HookExecutorNative})
	if err != nil {
		t.Fatalf("daemonExecutorResolver(bound native) error = %v, want nil", err)
	}
	if nativeExecutor.Kind() != hookspkg.HookExecutorNative {
		t.Fatalf("native executor kind = %q, want %q", nativeExecutor.Kind(), hookspkg.HookExecutorNative)
	}

	if _, err := resolver(hookspkg.HookDecl{Name: "missing", ExecutorKind: hookspkg.HookExecutorNative}); err == nil || !strings.Contains(err.Error(), "missing native hook executor") {
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
