package daemon

import (
	"context"
	"testing"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestHooksNotifierDispatchesLifecycleAndAgentEvents(t *testing.T) {
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

	notifier.OnSessionCreated(testutil.Context(t), sess)
	notifier.OnSessionStopped(testutil.Context(t), sess)
	notifier.OnAgentEvent(testutil.Context(t), "sess-created", struct{ Type string }{Type: "done"})

	wantOrder := []string{"rebuild", "create", "rebuild", "stop", "hook-agent"}
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
