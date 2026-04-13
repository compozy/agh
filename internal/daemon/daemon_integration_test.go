//go:build integration

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	"github.com/pedronauck/agh/internal/acp"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const daemonSessionStopHelperEnvKey = "AGH_TEST_DAEMON_SESSION_STOP_HELPER"

func installExtensionForDaemonIntegration(t *testing.T, databasePath string, name string, opts daemonTestExtensionOptions, enabled bool) string {
	t.Helper()

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), databasePath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(%q) error = %v", databasePath, err)
	}
	defer func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	}()

	return installDaemonTestExtension(t, db, name, opts, enabled)
}

func (f *fakeSessionManager) promptCall(index int) struct {
	id  string
	msg string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index < 0 || index >= len(f.promptCalls) {
		return struct {
			id  string
			msg string
		}{}
	}
	return f.promptCalls[index]
}

func (f *fakeSessionManager) promptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.promptCalls)
}

func (f *fakeNetworkBindableSessionManager) setPrompting(sessionID string, prompting bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if prompting {
		f.prompting[sessionID] = true
		return
	}
	delete(f.prompting, sessionID)
}

func TestBootSequenceReady(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.sessions == nil || d.observer == nil || d.registry == nil {
		t.Fatalf("boot() did not wire runtime dependencies: sessions=%v observer=%v registry=%v", d.sessions, d.observer, d.registry)
	}
	if d.workspaceResolver == nil {
		t.Fatal("boot() did not wire the workspace resolver")
	}
	if _, err := os.Stat(homePaths.DatabaseFile); err != nil {
		t.Fatalf("stat global database error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); err != nil {
		t.Fatalf("stat daemon.json error = %v", err)
	}
	if _, err := AcquireLock(homePaths.DaemonLock, os.Getpid()); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("AcquireLock(second instance) error = %v, want ErrAlreadyRunning", err)
	}
}

func TestBootPublishesRunningAutomationBeforeServersStart(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Automation.Enabled = true

	var httpSawRunning bool
	var udsSawRunning bool

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(ctx context.Context, deps RuntimeDeps) (Server, error) {
		if deps.Automation == nil {
			t.Fatal("http factory received nil automation manager")
		}
		status, err := deps.Automation.Status(ctx)
		if err != nil {
			t.Fatalf("deps.Automation.Status(http) error = %v", err)
		}
		if !status.Running || !status.SchedulerRunning {
			t.Fatalf("http factory automation status = %#v, want running scheduler", status)
		}
		httpSawRunning = true
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(ctx context.Context, deps RuntimeDeps) (Server, error) {
		if deps.Automation == nil {
			t.Fatal("uds factory received nil automation manager")
		}
		status, err := deps.Automation.Status(ctx)
		if err != nil {
			t.Fatalf("deps.Automation.Status(uds) error = %v", err)
		}
		if !status.Running || !status.SchedulerRunning {
			t.Fatalf("uds factory automation status = %#v, want running scheduler", status)
		}
		udsSawRunning = true
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.automation == nil {
		t.Fatal("boot() did not publish the automation manager")
	}
	if !httpSawRunning || !udsSawRunning {
		t.Fatalf("server factories observed automation running: http=%v uds=%v, want both true", httpSawRunning, udsSawRunning)
	}
}

func TestBootPreservesAutomationEnabledOverlaysAcrossRestart(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Automation.Enabled = true
	cfg.Automation.Jobs = []aghconfig.AutomationJob{
		{
			Scope:     automationpkg.AutomationScopeGlobal,
			Name:      "restart-job",
			AgentName: "researcher",
			Prompt:    "Summarize the latest state.",
			Schedule: automationpkg.ScheduleSpec{
				Mode:     automationpkg.ScheduleModeEvery,
				Interval: "1h",
			},
			Enabled:   true,
			Retry:     automationpkg.DefaultRetryConfig(),
			FireLimit: automationpkg.DefaultFireLimitConfig(),
			Source:    automationpkg.JobSourceConfig,
		},
	}
	cfg.Automation.Triggers = []aghconfig.AutomationTrigger{
		{
			Scope:     automationpkg.AutomationScopeGlobal,
			Name:      "restart-trigger",
			AgentName: "reviewer",
			Prompt:    `Review session {{ index .Data "session_id" }}`,
			Event:     "session.stopped",
			Filter:    map[string]string{"data.agent_name": "reviewer"},
			Enabled:   true,
			Retry:     automationpkg.DefaultRetryConfig(),
			FireLimit: automationpkg.DefaultFireLimitConfig(),
			Source:    automationpkg.JobSourceConfig,
		},
	}

	newDaemon := func() *Daemon {
		d, err := New(
			WithHomePaths(homePaths),
			WithConfig(cfg),
			WithLogger(discardLogger()),
		)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}
		d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
			return &fakeSessionManager{}, nil
		}
		d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
			return &fakeObserver{}, nil
		}
		d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
			return &fakeServer{name: "http"}, nil
		}
		d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
			return &fakeServer{name: "uds"}, nil
		}
		return d
	}

	first := newDaemon()
	if err := first.boot(testutil.Context(t)); err != nil {
		t.Fatalf("first boot() error = %v", err)
	}

	jobs, err := first.automation.Jobs(testutil.Context(t))
	if err != nil {
		t.Fatalf("first automation.Jobs() error = %v", err)
	}
	job := findAutomationJobByName(jobs, "restart-job")
	if job == nil {
		t.Fatal("first boot missing restart-job")
	}
	triggers, err := first.automation.Triggers(testutil.Context(t))
	if err != nil {
		t.Fatalf("first automation.Triggers() error = %v", err)
	}
	trigger := findAutomationTriggerByName(triggers, "restart-trigger")
	if trigger == nil {
		t.Fatal("first boot missing restart-trigger")
	}

	if _, err := first.automation.SetJobEnabled(testutil.Context(t), job.ID, false); err != nil {
		t.Fatalf("SetJobEnabled() error = %v", err)
	}
	if _, err := first.automation.SetTriggerEnabled(testutil.Context(t), trigger.ID, false); err != nil {
		t.Fatalf("SetTriggerEnabled() error = %v", err)
	}
	if err := first.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("first Shutdown() error = %v", err)
	}

	second := newDaemon()
	if err := second.boot(testutil.Context(t)); err != nil {
		t.Fatalf("second boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("second Shutdown() error = %v", err)
		}
	})

	jobs, err = second.automation.Jobs(testutil.Context(t))
	if err != nil {
		t.Fatalf("second automation.Jobs() error = %v", err)
	}
	job = findAutomationJobByName(jobs, "restart-job")
	if job == nil || job.Enabled {
		t.Fatalf("restarted job = %#v, want disabled overlay", job)
	}

	triggers, err = second.automation.Triggers(testutil.Context(t))
	if err != nil {
		t.Fatalf("second automation.Triggers() error = %v", err)
	}
	trigger = findAutomationTriggerByName(triggers, "restart-trigger")
	if trigger == nil || trigger.Enabled {
		t.Fatalf("restarted trigger = %#v, want disabled overlay", trigger)
	}

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	defer func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	}()

	storedJob, err := db.GetJob(testutil.Context(t), job.ID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}
	if !storedJob.Enabled {
		t.Fatal("stored config job enabled default = false, want true")
	}
	jobOverlay, err := db.GetJobEnabledOverlay(testutil.Context(t), job.ID)
	if err != nil {
		t.Fatalf("GetJobEnabledOverlay() error = %v", err)
	}
	if jobOverlay.EnabledOverride {
		t.Fatal("job overlay enabled_override = true, want false")
	}

	storedTrigger, err := db.GetTrigger(testutil.Context(t), trigger.ID)
	if err != nil {
		t.Fatalf("GetTrigger() error = %v", err)
	}
	if !storedTrigger.Enabled {
		t.Fatal("stored config trigger enabled default = false, want true")
	}
	triggerOverlay, err := db.GetTriggerEnabledOverlay(testutil.Context(t), trigger.ID)
	if err != nil {
		t.Fatalf("GetTriggerEnabledOverlay() error = %v", err)
	}
	if triggerOverlay.EnabledOverride {
		t.Fatal("trigger overlay enabled_override = true, want false")
	}
}

func TestShutdownCancelsActiveAutomationPrompt(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Automation.Enabled = true
	cfg.Automation.MaxConcurrentJobs = 1
	cfg.Automation.Jobs = []aghconfig.AutomationJob{
		{
			Scope:     automationpkg.AutomationScopeGlobal,
			Name:      "shutdown-job",
			AgentName: "researcher",
			Prompt:    "Summarize the latest state.",
			Schedule: automationpkg.ScheduleSpec{
				Mode:     automationpkg.ScheduleModeEvery,
				Interval: "10ms",
			},
			Enabled:   true,
			Retry:     automationpkg.DefaultRetryConfig(),
			FireLimit: automationpkg.DefaultFireLimitConfig(),
			Source:    automationpkg.JobSourceConfig,
		},
	}

	promptStarted := make(chan struct{}, 1)
	promptCancelled := make(chan struct{}, 1)
	sessions := &fakeSessionManager{
		promptStarted:      promptStarted,
		promptCtxCancelled: promptCancelled,
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return sessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}

	select {
	case <-promptStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("automation scheduler did not reach Prompt() in time")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Shutdown(testutil.Context(t))
	}()

	select {
	case <-promptCancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("automation prompt context was not cancelled during shutdown")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown() did not finish after automation prompt cancellation")
	}
}
func TestBootNetworkEnabledDeliversInboundAndShutsDownCleanly(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Network.Enabled = true

	bindableSessions := newFakeNetworkBindableSessionManager()
	promptStarted := make(chan string, 1)
	bindableSessions.promptNetworkFn = func(ctx context.Context, sessionID string, message string) (<-chan acp.AgentEvent, error) {
		bindableSessions.setPrompting(sessionID, true)
		select {
		case promptStarted <- message:
		default:
		}

		events := make(chan acp.AgentEvent)
		go func() {
			<-ctx.Done()
			bindableSessions.setPrompting(sessionID, false)
			close(events)
		}()
		return events, nil
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return bindableSessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}

	lifecycle := bindableSessions.currentNetworkPeerLifecycle()
	if lifecycle == nil {
		t.Fatal("network lifecycle binding = nil, want boot-time late binding")
	}
	if err := lifecycle.JoinChannel(testutil.Context(t), "sess-net", "coder.sess-net", "builders"); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	body, err := json.Marshal(map[string]any{"text": "hello from network"})
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}
	if _, err := d.network.Send(testutil.Context(t), network.SendRequest{
		SessionID: "sess-net",
		Channel:   "builders",
		Kind:      network.KindSay,
		Body:      body,
	}); err != nil {
		t.Fatalf("network.Send() error = %v", err)
	}

	select {
	case message := <-promptStarted:
		if !strings.Contains(message, "hello from network") {
			t.Fatalf("prompt message = %q, want network payload preview", message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for inbound network delivery")
	}

	status, err := d.network.Status(testutil.Context(t))
	if err != nil {
		t.Fatalf("network.Status() error = %v", err)
	}
	if status.LocalPeers != 1 || status.Channels != 1 {
		t.Fatalf("network.Status() = %#v, want 1 local peer and 1 channel", status)
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon info exists after shutdown: stat error = %v, want os.ErrNotExist", err)
	}
}

func TestBootNetworkShutdownTracksInterruptedInFlightDelivery(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Network.Enabled = true

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{Level: slog.LevelInfo}))
	bindableSessions := newFakeNetworkBindableSessionManager()
	promptStarted := make(chan string, 1)
	bindableSessions.promptNetworkFn = func(ctx context.Context, sessionID string, message string) (<-chan acp.AgentEvent, error) {
		bindableSessions.setPrompting(sessionID, true)
		select {
		case promptStarted <- message:
		default:
		}

		events := make(chan acp.AgentEvent)
		go func() {
			<-ctx.Done()
			bindableSessions.setPrompting(sessionID, false)
			close(events)
		}()
		return events, nil
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return bindableSessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}

	lifecycle := bindableSessions.currentNetworkPeerLifecycle()
	if lifecycle == nil {
		t.Fatal("network lifecycle binding = nil, want boot-time late binding")
	}
	if err := lifecycle.JoinChannel(testutil.Context(t), "sess-net", "coder.sess-net", "builders"); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	body, err := json.Marshal(map[string]any{"text": "shutdown during delivery"})
	if err != nil {
		t.Fatalf("json.Marshal(body) error = %v", err)
	}
	if _, err := d.network.Send(testutil.Context(t), network.SendRequest{
		SessionID: "sess-net",
		Channel:   "builders",
		Kind:      network.KindSay,
		Body:      body,
	}); err != nil {
		t.Fatalf("network.Send() error = %v", err)
	}

	select {
	case message := <-promptStarted:
		if !strings.Contains(message, "shutdown during delivery") {
			t.Fatalf("prompt message = %q, want network payload preview", message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for inbound network delivery")
	}

	status, err := d.network.Status(testutil.Context(t))
	if err != nil {
		t.Fatalf("network.Status() error = %v", err)
	}
	if status.MessagesDelivered != 0 || status.DeliveryWorkers != 1 {
		t.Fatalf("network.Status() before shutdown = %#v, want delivered=0 workers=1", status)
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	logOutput := logBuffer.String()
	for _, want := range []string{
		"network.message.delivery_interrupted",
		"pending_messages=1",
		"inflight_messages=1",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("log output missing %q:\n%s", want, logOutput)
		}
	}
	if strings.Contains(logOutput, "network.message.delivered") {
		t.Fatalf("log output unexpectedly reported delivered message:\n%s", logOutput)
	}
}
func TestBootLoadsExtensionsRebuildsHooksAndStopsOnShutdown(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	hookMarker := filepath.Join(t.TempDir(), "hook.json")
	shutdownMarker := filepath.Join(t.TempDir(), "shutdown.txt")
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, "ext-daemon", daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperEnv(shutdownMarker),
		hookCommand:    "/bin/sh",
		hookArgs: []string{
			"-c",
			`cat > "$1"; printf '{}'`,
			"agh-extension-hook",
			hookMarker,
		},
		hookEvent: hookspkg.HookSessionPostCreate,
	}, true)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	if d.extensions == nil {
		t.Fatal("boot() did not publish the extension runtime")
	}

	payload := hookspkg.SessionPostCreatePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPostCreate,
			Timestamp: time.Now().UTC(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID: "sess-ext",
			AgentName: "coder",
			State:     string(session.StateActive),
		},
	}
	if _, err := d.hooks.DispatchSessionPostCreate(testutil.Context(t), payload); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}

	waitForCondition(t, "extension hook marker", func() bool {
		_, err := os.Stat(hookMarker)
		return err == nil
	})
	hookPayload, err := os.ReadFile(hookMarker)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", hookMarker, err)
	}
	if !strings.Contains(string(hookPayload), "sess-ext") {
		t.Fatalf("hook payload = %q, want session id", string(hookPayload))
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if payload, err := os.ReadFile(shutdownMarker); err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", shutdownMarker, err)
	} else if strings.TrimSpace(string(payload)) != "shutdown" {
		t.Fatalf("shutdown marker = %q, want shutdown", string(payload))
	}
}

func TestBootContinuesAfterCorruptExtensionAndKeepsHealthyExtensions(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	hookMarker := filepath.Join(t.TempDir(), "hook.json")
	shutdownMarker := filepath.Join(t.TempDir(), "shutdown.txt")
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, "ext-good", daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperEnv(shutdownMarker),
		hookCommand:    "/bin/sh",
		hookArgs: []string{
			"-c",
			`cat > "$1"; printf '{}'`,
			"agh-extension-hook",
			hookMarker,
		},
		hookEvent: hookspkg.HookSessionPostCreate,
	}, true)
	badDir := installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, "ext-bad", daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperEnv(""),
	}, true)
	writeDaemonFile(t, filepath.Join(badDir, "extension.toml"), "not = [valid")

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v, want boot to continue after corrupt extension", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if !strings.Contains(logBuffer.String(), "extension manager start failed") {
		t.Fatalf("log output = %q, want extension start failure entry", logBuffer.String())
	}

	payload := hookspkg.SessionPostCreatePayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPostCreate,
			Timestamp: time.Now().UTC(),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID: "sess-good",
			AgentName: "coder",
			State:     string(session.StateActive),
		},
	}
	if _, err := d.hooks.DispatchSessionPostCreate(testutil.Context(t), payload); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}

	waitForCondition(t, "healthy extension hook marker", func() bool {
		_, err := os.Stat(hookMarker)
		return err == nil
	})
	hookPayload, err := os.ReadFile(hookMarker)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", hookMarker, err)
	}
	if !strings.Contains(string(hookPayload), "sess-good") {
		t.Fatalf("hook payload = %q, want healthy extension session id", string(hookPayload))
	}
}

func TestRunGracefulShutdownViaContextCancellation(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(runCtx)
	}()

	<-d.readyCh
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after shutdown: stat error = %v, want os.ErrNotExist", err)
	}

	lock, err := AcquireLock(homePaths.DaemonLock, os.Getpid())
	if err != nil {
		t.Fatalf("AcquireLock(after shutdown) error = %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("lock.Release() error = %v", err)
	}
}

func TestRunGracefulShutdownViaSignal(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	signalCh := make(chan os.Signal, 1)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
		WithSignalBridge(signalCh),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(context.Background())
	}()

	<-d.readyCh
	signalCh <- syscall.SIGINT

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after signal shutdown: stat error = %v, want os.ErrNotExist", err)
	}
}

func TestShutdownPersistsShutdownStopReason(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	command := daemonSessionStopHelperCommand(t)
	cfg.Providers["claude"] = aghconfig.ProviderConfig{Command: command}
	writeDaemonIntegrationAgentDef(t, homePaths, "coder", command)

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", workspaceRoot, err)
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	shutdown := false
	t.Cleanup(func() {
		if shutdown {
			return
		}
		_ = d.Shutdown(testutil.Context(t))
	})

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}

	sess, err := d.sessions.Create(testutil.Context(t), session.CreateOpts{
		AgentName:     "coder",
		WorkspacePath: workspaceRoot,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	shutdown = true

	meta, err := store.ReadSessionMeta(sess.MetaPath())
	if err != nil {
		t.Fatalf("ReadSessionMeta(%q) error = %v", sess.MetaPath(), err)
	}
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopShutdown {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopShutdown)
	}
}

func TestBootInitializesMemoryStoreAndAssemblerIntegration(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.GlobalDir = filepath.Join(homePaths.HomeDir, "external-memory")

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.memoryStore == nil {
		t.Fatal("boot() did not initialize the memory store")
	}
	if capturedDeps.PromptAssembler == nil {
		t.Fatal("boot() did not inject the prompt assembler")
	}
	if capturedDeps.SkillRegistry == nil {
		t.Fatal("boot() did not inject the skills registry")
	}
	if capturedDeps.MCPResolver == nil {
		t.Fatal("boot() did not inject the MCP resolver")
	}
	if capturedDeps.WorkspaceResolver == nil {
		t.Fatal("boot() did not inject the workspace resolver")
	}
	if _, err := os.Stat(cfg.Memory.GlobalDir); err != nil {
		t.Fatalf("stat external memory directory error = %v", err)
	}
}

func TestBootLoadsBundledSkillsIntoPromptAssemblerInSkillsOnlyMode(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if capturedDeps.PromptAssembler == nil {
		t.Fatal("boot() did not inject the prompt assembler")
	}
	if capturedDeps.WorkspaceResolver == nil {
		t.Fatal("boot() did not inject the workspace resolver")
	}
	if d.skillsRegistry == nil {
		t.Fatal("boot() did not initialize the skills registry")
	}
	if _, ok := d.skillsRegistry.Get("agh-session-guide"); !ok {
		t.Fatal("skills registry does not contain bundled skill agh-session-guide")
	}

	prompt, err := capturedDeps.PromptAssembler.Assemble(context.Background(), testPromptAgent("Base prompt."), workspacepkg.ResolvedWorkspace{})
	if err != nil {
		t.Fatalf("PromptAssembler.Assemble() error = %v", err)
	}

	assertPromptContainsInOrder(t, prompt, "Base prompt.", "<available-skills>", "agh-session-guide")
	assertPromptExcludes(t, prompt, "# Persistent Memory")
}

func TestBootLeavesSkillDependenciesNilWhenSkillsDisabled(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Skills.Enabled = false

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if capturedDeps.SkillRegistry != nil {
		t.Fatalf("boot() SkillRegistry = %#v, want nil when skills are disabled", capturedDeps.SkillRegistry)
	}
	if capturedDeps.MCPResolver != nil {
		t.Fatalf("boot() MCPResolver = %#v, want nil when skills are disabled", capturedDeps.MCPResolver)
	}
}

func TestBootBuildsHooksFromWorkspaceConfigAgentAndSkills(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(filepath.Join(workspaceRoot, aghconfig.DirName), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Join(workspaceRoot, aghconfig.DirName), err)
	}

	scriptPath := writeDaemonHookScript(t, t.TempDir(), "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")
	configOutput := filepath.Join(t.TempDir(), "config-create.json")
	agentOutput := filepath.Join(t.TempDir(), "agent-stop.json")
	skillOutput := filepath.Join(t.TempDir(), "skill-create.json")

	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "config.toml"), `
[[hooks.declarations]]
name = "config-create"
event = "session.post_create"
mode = "sync"
command = "`+scriptPath+`"
args = ["`+configOutput+`"]
`)
	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "agents", "coder", "AGENT.md"), `---
name: coder
provider: claude
hooks:
  - name: agent-stop
    event: session.post_stop
    mode: sync
    command: `+scriptPath+`
    args: ["`+agentOutput+`"]
---

Prompt.
`)
	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "skills", "local-hook", "SKILL.md"), `---
name: local-hook
description: workspace lifecycle hook
metadata:
  agh:
    hooks:
      - event: session.post_create
        mode: sync
        command: `+scriptPath+`
        args:
          - `+skillOutput+`
---

body
`)

	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspaceRoot)

	var capturedDeps SessionManagerDeps
	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.hooks == nil {
		t.Fatal("boot() did not initialize hooks runtime")
	}
	if capturedDeps.Notifier == nil {
		t.Fatal("boot() did not inject the hooks notifier")
	}
	if capturedDeps.Hooks.Session == nil {
		t.Fatal("boot() did not inject the hooks dispatcher")
	}

	sess := &session.Session{
		ID:          "sess-1",
		Name:        "demo",
		AgentName:   "coder",
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        session.SessionTypeUser,
		State:       session.StateStopped,
		CreatedAt:   time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
	}

	if _, err := capturedDeps.Hooks.Session.DispatchSessionPostCreate(testutil.Context(t), hookspkg.SessionPostCreatePayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := capturedDeps.Hooks.Session.DispatchSessionPostStop(testutil.Context(t), hookspkg.SessionPostStopPayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
	}

	assertLifecycleHookPayload(t, configOutput, hookspkg.HookSessionPostCreate, resolvedWorkspace)
	assertLifecycleHookPayload(t, skillOutput, hookspkg.HookSessionPostCreate, resolvedWorkspace)
	assertLifecycleHookPayload(t, agentOutput, hookspkg.HookSessionPostStop, resolvedWorkspace)
}

func TestBootSkillsWatcherRebuildsHooksBeforeNextDispatch(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true
	cfg.Skills.PollInterval = 10 * time.Millisecond

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspaceRoot)
	outputPath := filepath.Join(t.TempDir(), "watched-create.json")
	scriptPath := writeDaemonHookScript(t, t.TempDir(), "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")

	var capturedDeps SessionManagerDeps
	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	if capturedDeps.Hooks.Session == nil {
		t.Fatal("boot() did not inject the hooks dispatcher")
	}

	initialVersion := d.hooks.Version()
	writeDaemonFile(t, filepath.Join(homePaths.SkillsDir, "watched-hook", "SKILL.md"), `---
name: watched-hook
description: reloaded hook
metadata:
  agh:
    hooks:
      - event: session.post_create
        mode: sync
        command: `+scriptPath+`
        args:
          - `+outputPath+`
---

body
`)

	waitForCondition(t, "hooks rebuild after watcher refresh", func() bool {
		if _, ok := d.skillsRegistry.Get("watched-hook"); !ok {
			return false
		}
		return d.hooks.Version() > initialVersion
	})

	sess := &session.Session{
		ID:          "sess-watch",
		AgentName:   "general",
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
	}

	if _, err := capturedDeps.Hooks.Session.DispatchSessionPostCreate(testutil.Context(t), hookspkg.SessionPostCreatePayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	assertLifecycleHookPayload(t, outputPath, hookspkg.HookSessionPostCreate, resolvedWorkspace)
}

func TestRunDreamTickerAndSpawnerIntegration(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Dream.CheckInterval = 10 * time.Millisecond

	workspace := filepath.Join(t.TempDir(), "workspace")
	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspace)
	dream := &fakeDreamService{
		shouldRun: true,
		runHook: func(ctx context.Context, spawn memory.SessionSpawner, workspace string) error {
			return spawn(ctx, "memory-consolidation", "integration prompt", workspace)
		},
	}
	sessions := &fakeSessionManager{
		infos: []*session.SessionInfo{
			{
				ID:          "sess-user",
				WorkspaceID: resolvedWorkspace.ID,
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return sessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.newDreamService = func(opts ...memory.Option) consolidation.Service {
		return dream
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	runCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(runCtx)
	}()

	<-d.readyCh
	waitForCondition(t, "integration dream run", func() bool {
		return sessions.createCount() > 0
	})

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := sessions.createCall(0).Type; got != session.SessionTypeDream {
		t.Fatalf("Create() session type = %q, want %q", got, session.SessionTypeDream)
	}
	if got := sessions.createCall(0).Workspace; got != resolvedWorkspace.ID {
		t.Fatalf("Create() workspace = %q, want %q", got, resolvedWorkspace.ID)
	}
	if got := sessions.createCall(0).WorkspacePath; got != "" {
		t.Fatalf("Create() workspace_path = %q, want empty", got)
	}
	if got := sessions.promptCount(); got == 0 || sessions.promptCall(0).msg != "integration prompt" {
		t.Fatalf("Prompt() calls = %d, want integration prompt", got)
	}
}

func TestBootStartsBridgeExtensionWithBoundRuntime(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	markerPath := filepath.Join(t.TempDir(), "bridge-init.jsonl")
	extensionName := "ext-bridge-daemon"
	instanceID := "brg-daemon-init"
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, extensionName, daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperScenarioEnv("record_initialize", markerPath),
		capabilities:   []string{extensionprotocol.CapabilityProvideBridgeAdapter},
		actions: []string{
			string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
			string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
			string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		},
		security: []string{"bridge.read", "bridge.write"},
	}, true)

	registry := openDaemonIntegrationGlobalDB(t, homePaths.DatabaseFile)
	bridgeRegistry := bridgepkg.NewRegistry(registry)
	instance, err := bridgeRegistry.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
		ID:            instanceID,
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "slack",
		ExtensionName: extensionName,
		DisplayName:   "Daemon Bridge",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if err := registry.PutBridgeSecretBinding(testutil.Context(t), bridgepkg.BridgeSecretBinding{
		BridgeInstanceID: instance.ID,
		BindingName:      "bot_token",
		VaultRef:         "vault://bridges/ext-bridge-daemon/bot-token",
		Kind:             "bot_token",
		CreatedAt:        time.Date(2026, 4, 11, 13, 30, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2026, 4, 11, 13, 30, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("PutBridgeSecretBinding() error = %v", err)
	}

	resolver := &recordingBridgeSecretResolver{
		values: map[string]string{
			"bot_token": "token-daemon",
		},
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
		WithBridgeSecretResolver(resolver),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.bridges == nil {
		t.Fatal("boot() did not publish the bridge runtime")
	}

	waitForCondition(t, "bridge initialize marker", func() bool {
		return markerLineCount(markerPath) >= 1
	})

	markers := readDaemonInitializeMarkers(t, markerPath)
	if len(markers) == 0 {
		t.Fatal("initialize markers = empty, want bridge launch handshake")
	}
	request := markers[0].Request
	if len(request.Methods.ExtensionServices) != 1 || request.Methods.ExtensionServices[0] != "bridges/deliver" {
		t.Fatalf("initialize extension services = %#v, want [bridges/deliver]", request.Methods.ExtensionServices)
	}
	if request.Runtime.Bridge == nil {
		t.Fatal("initialize runtime bridge = nil, want bound launch payload")
	}
	if got, want := request.Runtime.Bridge.Instance.ID, instanceID; got != want {
		t.Fatalf("initialize runtime bridge instance id = %q, want %q", got, want)
	}
	if got := request.Runtime.Bridge.BoundSecrets; len(got) != 1 || got[0].BindingName != "bot_token" || got[0].Value != "token-daemon" {
		t.Fatalf("initialize runtime bridge bound secrets = %#v, want resolved bot_token binding", got)
	}
	if len(resolver.calls) != 1 || resolver.calls[0].BridgeInstanceID != instanceID {
		t.Fatalf("ResolveBridgeSecret() calls = %#v, want one call for %q", resolver.calls, instanceID)
	}
}

func TestCreateEnabledBridgeAfterBootReloadsErroredExtension(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	markerPath := filepath.Join(t.TempDir(), "bridge-create.jsonl")
	extensionName := "ext-bridge-create"
	instanceID := "brg-daemon-create"
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, extensionName, daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperScenarioEnv("record_initialize", markerPath),
		capabilities:   []string{extensionprotocol.CapabilityProvideBridgeAdapter},
		actions: []string{
			string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
			string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
			string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		},
		security: []string{"bridge.read", "bridge.write"},
	}, true)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.bridges == nil {
		t.Fatal("boot() did not publish the bridge runtime")
	}

	waitForCondition(t, "bridge extension stays registered until an instance exists", func() bool {
		ext, err := d.extensions.Get(extensionName)
		return err == nil && ext != nil && ext.Status.Registered && !ext.Status.Active && ext.Status.LastError == ""
	})
	if got := markerLineCount(markerPath); got != 0 {
		t.Fatalf("initialize marker count before create = %d, want 0", got)
	}

	created, err := d.bridges.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
		ID:            instanceID,
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "slack",
		ExtensionName: extensionName,
		DisplayName:   "Create Bridge",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusStarting,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if created == nil {
		t.Fatal("CreateInstance() = nil, want non-nil")
	}

	waitForCondition(t, "bridge initialize marker after create", func() bool {
		return markerLineCount(markerPath) >= 1
	})
	markers := readDaemonInitializeMarkers(t, markerPath)
	if len(markers) == 0 {
		t.Fatal("initialize markers after create = empty, want launch handshake")
	}
	if got, want := markers[len(markers)-1].Request.Runtime.Bridge.Instance.ID, instanceID; got != want {
		t.Fatalf("initialize runtime bridge instance id after create = %q, want %q", got, want)
	}

	waitForCondition(t, "bridge extension recovers after create", func() bool {
		ext, err := d.extensions.Get(extensionName)
		return err == nil && ext != nil && ext.Status.Active
	})
}

func TestBridgeRuntimeRestartPreservesRouteContinuity(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	markerPath := filepath.Join(t.TempDir(), "bridge-restart.jsonl")
	extensionName := "ext-bridge-restart"
	instanceID := "brg-daemon-restart"
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, extensionName, daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperScenarioEnv("exit_once_record_deliveries", markerPath),
		capabilities:   []string{extensionprotocol.CapabilityProvideBridgeAdapter},
		actions: []string{
			string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
			string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
			string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		},
		security: []string{"bridge.read", "bridge.write"},
	}, true)

	registry := openDaemonIntegrationGlobalDB(t, homePaths.DatabaseFile)
	bridgeRegistry := bridgepkg.NewRegistry(registry)
	if _, err := bridgeRegistry.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
		ID:            instanceID,
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "slack",
		ExtensionName: extensionName,
		DisplayName:   "Restart Bridge",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}); err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	if d.bridges == nil {
		t.Fatal("boot() did not publish the bridge runtime")
	}

	route, err := d.bridges.UpsertRoute(testutil.Context(t), bridgepkg.BridgeRoute{
		Scope:            bridgepkg.ScopeGlobal,
		BridgeInstanceID: instanceID,
		PeerID:           "peer-restart",
		SessionID:        "sess-restart",
		AgentName:        "coder",
		LastActivityAt:   time.Date(2026, 4, 11, 13, 45, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("UpsertRoute() error = %v", err)
	}

	target := bridgepkg.DeliveryTarget{
		BridgeInstanceID: instanceID,
		PeerID:           "peer-restart",
		Mode:             bridgepkg.DeliveryModeDirectSend,
	}
	if _, err := d.bridges.Broker().RegisterPromptDelivery(testutil.Context(t), bridgepkg.PromptDeliveryRegistration{
		SessionID:      "sess-restart",
		TurnID:         "turn-restart",
		ExtensionName:  extensionName,
		DeliveryID:     "del-restart",
		RoutingKey:     route.RoutingKey(),
		DeliveryTarget: target,
	}); err != nil {
		t.Fatalf("RegisterPromptDelivery() error = %v", err)
	}
	if err := d.bridges.Broker().Deliver(testutil.Context(t), bridgepkg.DeliveryEvent{
		DeliveryID:       "del-restart",
		BridgeInstanceID: instanceID,
		RoutingKey:       route.RoutingKey(),
		DeliveryTarget:   target,
		Seq:              1,
		EventType:        bridgepkg.DeliveryEventTypeStart,
		Content:          bridgepkg.MessageContent{Text: "hello"},
	}); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := d.bridges.Broker().Deliver(testutil.Context(t), bridgepkg.DeliveryEvent{
		DeliveryID:       "del-restart",
		BridgeInstanceID: instanceID,
		RoutingKey:       route.RoutingKey(),
		DeliveryTarget:   target,
		Seq:              2,
		EventType:        bridgepkg.DeliveryEventTypeFinal,
		Content:          bridgepkg.MessageContent{Text: "hello"},
		Final:            true,
	}); err != nil {
		t.Fatalf("Deliver(final) error = %v", err)
	}

	waitForCondition(t, "bridge delivery resume marker", func() bool {
		payload, err := os.ReadFile(markerPath)
		return err == nil && strings.Contains(string(payload), `"event_type":"resume"`)
	})

	markers := readDaemonDeliveryMarkers(t, markerPath)
	if len(markers) < 2 {
		t.Fatalf("delivery markers = %d, want at least start + resume", len(markers))
	}
	if got := markers[0].Request.Event.EventType; got != bridgepkg.DeliveryEventTypeStart {
		t.Fatalf("first delivery event = %q, want start", got)
	}

	resumeIndex := -1
	for idx, marker := range markers {
		if marker.Request.Event.EventType == bridgepkg.DeliveryEventTypeResume {
			resumeIndex = idx
			break
		}
	}
	if resumeIndex < 0 {
		t.Fatalf("delivery markers = %#v, want resume event", markers)
	}
	if markers[resumeIndex].PID == markers[0].PID {
		t.Fatalf("resume marker pid = %d, want restart to use a different process than %d", markers[resumeIndex].PID, markers[0].PID)
	}
	if markers[resumeIndex].Request.Snapshot == nil {
		t.Fatal("resume marker snapshot = nil, want resumable state")
	}
	if got, want := markers[resumeIndex].Request.Snapshot.DeliveryID, "del-restart"; got != want {
		t.Fatalf("resume snapshot delivery id = %q, want %q", got, want)
	}

	resolved, err := d.bridges.ResolveRoute(testutil.Context(t), route.RoutingKey())
	if err != nil {
		t.Fatalf("ResolveRoute(after restart) error = %v", err)
	}
	if got, want := resolved.RoutingKeyHash, route.RoutingKeyHash; got != want {
		t.Fatalf("ResolveRoute(after restart).RoutingKeyHash = %q, want %q", got, want)
	}
}

func TestDaemonShutdownClosesBridgeRuntimeCleanly(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	markerPath := filepath.Join(t.TempDir(), "bridge-shutdown.txt")
	extensionName := "ext-bridge-shutdown"
	instanceID := "brg-daemon-shutdown"
	installExtensionForDaemonIntegration(t, homePaths.DatabaseFile, extensionName, daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperScenarioEnv("slow_record_deliveries", markerPath),
		capabilities:   []string{extensionprotocol.CapabilityProvideBridgeAdapter},
		actions: []string{
			string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
			string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
			string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		},
		security: []string{"bridge.read", "bridge.write"},
	}, true)

	registry := openDaemonIntegrationGlobalDB(t, homePaths.DatabaseFile)
	bridgeRegistry := bridgepkg.NewRegistry(registry)
	if _, err := bridgeRegistry.CreateInstance(testutil.Context(t), bridgepkg.CreateInstanceRequest{
		ID:            instanceID,
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "slack",
		ExtensionName: extensionName,
		DisplayName:   "Shutdown Bridge",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}); err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	if d.bridges == nil {
		t.Fatal("boot() did not publish the bridge runtime")
	}

	route, err := d.bridges.UpsertRoute(testutil.Context(t), bridgepkg.BridgeRoute{
		Scope:            bridgepkg.ScopeGlobal,
		BridgeInstanceID: instanceID,
		PeerID:           "peer-shutdown",
		SessionID:        "sess-shutdown",
		AgentName:        "coder",
		LastActivityAt:   time.Date(2026, 4, 11, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("UpsertRoute() error = %v", err)
	}

	target := bridgepkg.DeliveryTarget{
		BridgeInstanceID: instanceID,
		PeerID:           "peer-shutdown",
		Mode:             bridgepkg.DeliveryModeDirectSend,
	}
	if _, err := d.bridges.Broker().RegisterPromptDelivery(testutil.Context(t), bridgepkg.PromptDeliveryRegistration{
		SessionID:      "sess-shutdown",
		TurnID:         "turn-shutdown",
		ExtensionName:  extensionName,
		DeliveryID:     "del-shutdown",
		RoutingKey:     route.RoutingKey(),
		DeliveryTarget: target,
	}); err != nil {
		t.Fatalf("RegisterPromptDelivery() error = %v", err)
	}
	if err := d.bridges.Broker().Deliver(testutil.Context(t), bridgepkg.DeliveryEvent{
		DeliveryID:       "del-shutdown",
		BridgeInstanceID: instanceID,
		RoutingKey:       route.RoutingKey(),
		DeliveryTarget:   target,
		Seq:              1,
		EventType:        bridgepkg.DeliveryEventTypeStart,
		Content:          bridgepkg.MessageContent{Text: "hello"},
	}); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}

	waitForCondition(t, "bridge delivery started before shutdown", func() bool {
		return markerLineCount(markerPath) >= 1
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := d.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	payload, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", markerPath, err)
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if got, want := lines[len(lines)-1], "shutdown"; got != want {
		t.Fatalf("shutdown marker final line = %q, want %q", got, want)
	}
}

func integrationHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("AGH_HOME", homeDir)
	t.Setenv("HOME", homeDir)

	homePaths, err := aghconfig.ResolveHomePathsFrom(homeDir)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	homePaths.DaemonSocket = shortSocketPath(t)
	return homePaths
}

func TestDaemonSessionStopACPHelperProcess(t *testing.T) {
	if os.Getenv(daemonSessionStopHelperEnvKey) != "1" {
		return
	}

	conn := acpsdk.NewAgentSideConnection(daemonSessionStopACPAgent{}, os.Stdout, os.Stdin)
	<-conn.Done()
	os.Exit(0)
}

func seedDaemonWorkspace(t *testing.T, homePaths aghconfig.HomePaths, root string) workspacepkg.ResolvedWorkspace {
	t.Helper()

	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}

	registry, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	defer func() {
		if err := registry.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	resolved, err := resolver.ResolveOrRegister(testutil.Context(t), root)
	if err != nil {
		t.Fatalf("ResolveOrRegister(%q) error = %v", root, err)
	}
	return resolved
}

func findAutomationJobByName(jobs []automationpkg.Job, name string) *automationpkg.Job {
	for idx := range jobs {
		if jobs[idx].Name == name {
			return &jobs[idx]
		}
	}
	return nil
}

func findAutomationTriggerByName(triggers []automationpkg.Trigger, name string) *automationpkg.Trigger {
	for idx := range triggers {
		if triggers[idx].Name == name {
			return &triggers[idx]
		}
	}
	return nil
}

func writeDaemonHookScript(t *testing.T, dir string, name string, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

func daemonSessionStopHelperCommand(t *testing.T) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	return shellquote.Join(
		"env",
		daemonSessionStopHelperEnvKey+"=1",
		bin,
		"-test.run=TestDaemonSessionStopACPHelperProcess",
	)
}

func writeDaemonIntegrationAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string, command string) {
	t.Helper()

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	content := strings.Join([]string{
		"---",
		"name: " + name,
		"provider: claude",
		"command: " + command,
		"---",
		"You are a coding assistant.",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func openDaemonIntegrationGlobalDB(t *testing.T, databasePath string) *globaldb.GlobalDB {
	t.Helper()

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), databasePath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(%q) error = %v", databasePath, err)
	}
	t.Cleanup(func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	})
	return db
}

func readDaemonInitializeMarkers(t *testing.T, path string) []daemonInitializeMarker {
	t.Helper()

	lines, err := readDaemonMarkerLines(path)
	if err != nil {
		t.Fatalf("readDaemonMarkerLines(%q) error = %v", path, err)
	}

	markers := make([]daemonInitializeMarker, 0, len(lines))
	for _, line := range lines {
		var marker daemonInitializeMarker
		if err := json.Unmarshal([]byte(line), &marker); err != nil {
			t.Fatalf("json.Unmarshal(initialize marker) error = %v; line=%q", err, line)
		}
		markers = append(markers, marker)
	}
	return markers
}

func readDaemonDeliveryMarkers(t *testing.T, path string) []daemonDeliveryMarker {
	t.Helper()

	lines, err := readDaemonMarkerLines(path)
	if err != nil {
		t.Fatalf("readDaemonMarkerLines(%q) error = %v", path, err)
	}

	markers := make([]daemonDeliveryMarker, 0, len(lines))
	for _, line := range lines {
		var marker daemonDeliveryMarker
		if err := json.Unmarshal([]byte(line), &marker); err != nil {
			t.Fatalf("json.Unmarshal(delivery marker) error = %v; line=%q", err, line)
		}
		markers = append(markers, marker)
	}
	return markers
}

func readDaemonMarkerLines(path string) ([]string, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	return filtered, nil
}

type daemonSessionStopACPAgent struct{}

func (daemonSessionStopACPAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (daemonSessionStopACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (daemonSessionStopACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (daemonSessionStopACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{SessionId: "daemon-stop-helper"}, nil
}

func (daemonSessionStopACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (daemonSessionStopACPAgent) Prompt(context.Context, acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (daemonSessionStopACPAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func assertLifecycleHookPayload(t *testing.T, path string, wantEvent hookspkg.HookEvent, wantWorkspace workspacepkg.ResolvedWorkspace) {
	t.Helper()

	var (
		payloadBytes []byte
		payload      hookspkg.SessionLifecyclePayload
		readOK       bool
		unmarshalOK  bool
	)

	t.Run("read file", func(t *testing.T) {
		var err error
		payloadBytes, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}
		readOK = true
	})

	t.Run("unmarshal", func(t *testing.T) {
		if !readOK {
			t.Skip("payload unavailable after read failure")
		}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
		}
		unmarshalOK = true
	})

	t.Run("event", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.Event != wantEvent {
			t.Fatalf("payload.Event = %q, want %q", payload.Event, wantEvent)
		}
	})

	t.Run("workspace id", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.WorkspaceID != wantWorkspace.ID {
			t.Fatalf("payload.WorkspaceID = %q, want %q", payload.WorkspaceID, wantWorkspace.ID)
		}
	})

	t.Run("workspace path", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.Workspace != wantWorkspace.RootDir {
			t.Fatalf("payload.Workspace = %q, want %q", payload.Workspace, wantWorkspace.RootDir)
		}
	})
}
