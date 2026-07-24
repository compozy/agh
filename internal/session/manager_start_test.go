package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
	"github.com/compozy/agh/internal/transcript"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestCreateAcceptedPersistsStartingBeforeProviderStartupCompletes(t *testing.T) {
	t.Parallel()
	t.Run(
		"Should persist starting and expose an empty transcript before provider activation",
		testCreateAcceptedPersistsStarting,
	)
	t.Run(
		"Should remain externally starting until the committed startup run completes",
		testCreateAcceptedRemainsStartingUntilCommitted,
	)
}

func testCreateAcceptedPersistsStarting(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	startEntered := make(chan struct{})
	releaseStart := make(chan struct{})
	h.driver.startHook = func(opts acp.StartOpts, _ int) (*fakeProcess, error) {
		close(startEntered)
		<-releaseStart
		return newFakeProcess(opts.AgentName, opts.Command, opts.Cwd, "acp-accepted"), nil
	}

	startedAt := time.Now()
	accepted, err := h.manager.CreateAccepted(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	acceptanceLatency := time.Since(startedAt)
	if err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	if acceptanceLatency >= 250*time.Millisecond {
		t.Fatalf("CreateAccepted() latency = %s, want < 250ms while provider startup is blocked", acceptanceLatency)
	}
	if accepted.State != StateStarting {
		t.Fatalf("CreateAccepted() State = %q, want %q", accepted.State, StateStarting)
	}
	live, ok := h.manager.Get(accepted.ID)
	if !ok || live.Info().State != StateStarting {
		t.Fatalf("Get(%q) = %#v/%t, want durable starting session", accepted.ID, live, ok)
	}
	if meta := readMeta(t, live.MetaPath()); meta.State != string(StateStarting) {
		t.Fatalf("accepted meta State = %q, want %q", meta.State, StateStarting)
	}
	page, err := h.manager.TranscriptPage(testutil.Context(t), accepted.ID, transcript.PageQuery{})
	if err != nil {
		t.Fatalf("TranscriptPage(starting) error = %v", err)
	}
	if len(page.Entries) != 0 {
		t.Fatalf("TranscriptPage(starting) entries = %#v, want empty", page.Entries)
	}
	<-startEntered
	close(releaseStart)
	waitForCondition(t, "accepted session activation", func() bool {
		live, ok := h.manager.Get(accepted.ID)
		return ok && live.Info().State == StateActive
	})
	if err := h.manager.Stop(testutil.Context(t), accepted.ID); err != nil {
		t.Fatalf("Stop() cleanup error = %v", err)
	}
}

func testCreateAcceptedRemainsStartingUntilCommitted(t *testing.T) {
	t.Helper()
	created := make(chan struct{})
	release := make(chan struct{})
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	h := newHarness(t, WithNotifier(&blockingCreatedNotifier{created: created, release: release}))

	accepted, err := h.manager.CreateAccepted(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	<-created

	status, err := h.manager.Status(testutil.Context(t), accepted.ID)
	if err != nil {
		t.Fatalf("Status(starting commit) error = %v", err)
	}
	if status.State != StateStarting {
		t.Fatalf("Status(starting commit) State = %q, want %q", status.State, StateStarting)
	}
	listed := h.manager.List()
	if len(listed) != 1 || listed[0].ID != accepted.ID || listed[0].State != StateStarting {
		t.Fatalf("List(starting commit) = %#v, want one starting session", listed)
	}
	if _, err := h.manager.SendPrompt(testutil.Context(t), accepted.ID, SendPromptOpts{
		Message: "too early",
	}); !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("SendPrompt(starting commit) error = %v, want %v", err, ErrSessionNotActive)
	}

	close(release)
	released = true
	waitForCondition(t, "committed session activation", func() bool {
		current, statusErr := h.manager.Status(testutil.Context(t), accepted.ID)
		return statusErr == nil && current.State == StateActive
	})
	if err := h.manager.Stop(testutil.Context(t), accepted.ID); err != nil {
		t.Fatalf("Stop() cleanup error = %v", err)
	}
}

type blockingCreatedNotifier struct {
	created chan struct{}
	release chan struct{}
}

func (n *blockingCreatedNotifier) OnSessionCreated(context.Context, *Session) {
	close(n.created)
	<-n.release
}

func (*blockingCreatedNotifier) OnSessionStopped(context.Context, *Session) {}

func (*blockingCreatedNotifier) OnAgentEvent(context.Context, string, any) {}

func TestCreateAcceptedPersistsBackgroundStartupFailure(t *testing.T) {
	t.Parallel()
	t.Run("Should persist background startup failure after durable acceptance", testCreateAcceptedPersistsFailure)
}

func testCreateAcceptedPersistsFailure(t *testing.T) {
	t.Helper()
	h := newHarness(t)
	startErr := errors.New("provider unavailable")
	h.driver.startHook = func(acp.StartOpts, int) (*fakeProcess, error) {
		return nil, startErr
	}

	accepted, err := h.manager.CreateAccepted(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	if accepted.State != StateStarting {
		t.Fatalf("CreateAccepted() State = %q, want %q", accepted.State, StateStarting)
	}
	waitForCondition(t, "durable startup failure", func() bool {
		info, statusErr := h.manager.Status(testutil.Context(t), accepted.ID)
		return statusErr == nil && info.State == StateStopped && info.Failure != nil
	})
	info, err := h.manager.Status(testutil.Context(t), accepted.ID)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if info.Failure.Kind != store.FailureStartup || !strings.Contains(info.Failure.Summary, startErr.Error()) {
		t.Fatalf("startup failure = %#v, want startup kind with provider cause", info.Failure)
	}
}

func TestStopJoinsAcceptedStartupAndPreventsLateActivation(t *testing.T) {
	t.Parallel()
	t.Run("Should join accepted startup and prevent late activation", testStopJoinsAcceptedStartup)
}

func testStopJoinsAcceptedStartup(t *testing.T) {
	t.Helper()
	entered := make(chan struct{})
	h := newHarness(t, WithPromptAssembler(promptAssemblerFunc(func(
		ctx context.Context,
		_ aghconfig.AgentDef,
		_ *workspacepkg.ResolvedWorkspace,
	) (string, error) {
		close(entered)
		<-ctx.Done()
		return "", ctx.Err()
	})))
	accepted, err := h.manager.CreateAccepted(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	<-entered
	if err := h.manager.Stop(testutil.Context(t), accepted.ID); err != nil {
		t.Fatalf("Stop(starting) error = %v", err)
	}
	if _, ok := h.manager.Get(accepted.ID); ok {
		t.Fatalf("Get(%q) found session after joined startup stop", accepted.ID)
	}
	info, err := h.manager.Status(testutil.Context(t), accepted.ID)
	if err != nil {
		t.Fatalf("Status(stopped start) error = %v", err)
	}
	if info.State != StateStopped {
		t.Fatalf("Status(stopped start).State = %q, want %q", info.State, StateStopped)
	}
	if got := len(h.driver.startCalls); got != 0 {
		t.Fatalf("driver start calls = %d, want 0 after startup cancellation", got)
	}
}

func TestDeleteJoinsAcceptedStartupAndRemovesDurableSession(t *testing.T) {
	t.Parallel()
	t.Run("Should join accepted startup and remove its durable session", testDeleteJoinsAcceptedStartup)
}

func testDeleteJoinsAcceptedStartup(t *testing.T) {
	t.Helper()
	entered := make(chan struct{})
	h := newHarness(t, WithPromptAssembler(promptAssemblerFunc(func(
		ctx context.Context,
		_ aghconfig.AgentDef,
		_ *workspacepkg.ResolvedWorkspace,
	) (string, error) {
		close(entered)
		<-ctx.Done()
		return "", ctx.Err()
	})))
	accepted, err := h.manager.CreateAccepted(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("CreateAccepted() error = %v", err)
	}
	<-entered
	if err := h.manager.Delete(testutil.Context(t), accepted.ID); err != nil {
		t.Fatalf("Delete(starting) error = %v", err)
	}
	if _, ok := h.manager.Get(accepted.ID); ok {
		t.Fatalf("Get(%q) found session after delete joined startup", accepted.ID)
	}
	if _, err := h.manager.Status(testutil.Context(t), accepted.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Status(deleted start) error = %v, want ErrSessionNotFound", err)
	}
	if got := len(h.driver.startCalls); got != 0 {
		t.Fatalf("driver start calls = %d, want 0 after startup deletion", got)
	}
}

func TestSessionStartEnvFiltersDaemonSecrets(t *testing.T) {
	t.Parallel()

	t.Run("Should remove credential shaped daemon variables and keep AGH session context", func(t *testing.T) {
		t.Parallel()

		env := sessionStartEnv(
			[]string{
				"PATH=/usr/bin",
				"OPENAI_API_KEY=sk-secret",
				"GITHUB_TOKEN=ghp-secret",
				"PROVIDER_HOME=/tmp/provider",
			},
			&Session{
				ID:                   "sess-1",
				AgentName:            "coder",
				NetworkParticipation: testLiveParticipation("ws-test", "ops"),
			},
		)

		if got := envValue(env, "OPENAI_API_KEY"); got != "" {
			t.Fatalf("OPENAI_API_KEY = %q, want filtered", got)
		}
		if got := envValue(env, "GITHUB_TOKEN"); got != "" {
			t.Fatalf("GITHUB_TOKEN = %q, want filtered", got)
		}
		if got := envValue(env, "PROVIDER_HOME"); got != "/tmp/provider" {
			t.Fatalf("PROVIDER_HOME = %q, want %q", got, "/tmp/provider")
		}
		if got := envValue(env, "AGH_SESSION_ID"); got != "sess-1" {
			t.Fatalf("AGH_SESSION_ID = %q, want %q", got, "sess-1")
		}
		if got := envValue(env, "AGH_PEER_ID"); got == "" {
			t.Fatal("AGH_PEER_ID = empty, want network peer id")
		}
	})
}

func TestSessionStartEnvForProviderSupportsIsolatedPolicy(t *testing.T) {
	t.Parallel()

	t.Run("Should keep only operational env before adding session context", func(t *testing.T) {
		t.Parallel()

		env := sessionStartEnvForProvider(
			[]string{
				"PATH=/usr/bin",
				"HOME=/Users/operator",
				"OPENAI_API_KEY=sk-secret",
				"FEATURE_FLAG=enabled",
				"PROVIDER_HOME=/tmp/provider",
			},
			&Session{
				ID:                   "sess-1",
				AgentName:            "coder",
				NetworkParticipation: testLiveParticipation("ws-test", "ops"),
			},
			aghconfig.ProviderEnvPolicyIsolated,
		)

		if got := envValue(env, "OPENAI_API_KEY"); got != "" {
			t.Fatalf("OPENAI_API_KEY = %q, want isolated env to drop secrets", got)
		}
		if got := envValue(env, "FEATURE_FLAG"); got != "" {
			t.Fatalf("FEATURE_FLAG = %q, want isolated env to drop non-allowlisted variables", got)
		}
		if got := envValue(env, "PATH"); got != "/usr/bin" {
			t.Fatalf("PATH = %q, want %q", got, "/usr/bin")
		}
		if got := envValue(env, "AGH_SESSION_ID"); got != "sess-1" {
			t.Fatalf("AGH_SESSION_ID = %q, want %q", got, "sess-1")
		}
	})
}
