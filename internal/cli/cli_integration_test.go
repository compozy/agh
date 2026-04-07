//go:build integration

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/udsapi"
	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestCLIRoundTripIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)

	startOut, _, err := executeRootCommand(t, h.deps, "daemon", "start", "-o", "json")
	if err != nil {
		t.Fatalf("daemon start error = %v", err)
	}
	var started DaemonStatus
	if err := json.Unmarshal([]byte(startOut), &started); err != nil {
		t.Fatalf("json.Unmarshal(start) error = %v", err)
	}
	if started.Status != "running" {
		t.Fatalf("start status = %q, want %q", started.Status, "running")
	}

	newOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(newOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created session id")
	}

	promptOut, _, err := executeRootCommand(t, h.deps, "session", "prompt", created.ID, "hello", "-o", "json")
	if err != nil {
		t.Fatalf("session prompt error = %v", err)
	}
	var promptEvents []AgentEventRecord
	if err := json.Unmarshal([]byte(promptOut), &promptEvents); err != nil {
		t.Fatalf("json.Unmarshal(prompt) error = %v", err)
	}
	if len(promptEvents) < 2 {
		t.Fatalf("prompt events = %d, want at least 2", len(promptEvents))
	}

	eventsOut, _, err := executeRootCommand(t, h.deps, "session", "events", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session events error = %v", err)
	}
	var events []SessionEventRecord
	if err := json.Unmarshal([]byte(eventsOut), &events); err != nil {
		t.Fatalf("json.Unmarshal(events) error = %v", err)
	}
	if len(events) < 2 {
		t.Fatalf("session events = %d, want at least 2", len(events))
	}

	stopOut, _, err := executeRootCommand(t, h.deps, "session", "stop", created.ID, "-o", "json")
	if err != nil {
		t.Fatalf("session stop error = %v", err)
	}
	var stopped SessionRecord
	if err := json.Unmarshal([]byte(stopOut), &stopped); err != nil {
		t.Fatalf("json.Unmarshal(stop) error = %v", err)
	}
	if stopped.State != string(session.StateStopped) {
		t.Fatalf("stopped.State = %q, want %q", stopped.State, session.StateStopped)
	}

	daemonStopOut, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
	if err != nil {
		t.Fatalf("daemon stop error = %v", err)
	}
	var daemonStopped DaemonStatus
	if err := json.Unmarshal([]byte(daemonStopOut), &daemonStopped); err != nil {
		t.Fatalf("json.Unmarshal(daemon stop) error = %v", err)
	}
	if daemonStopped.Status != "stopped" {
		t.Fatalf("daemon stop status = %q, want %q", daemonStopped.Status, "stopped")
	}

	if err := h.runner.waitForExit(); err != nil {
		t.Fatalf("waitForExit() error = %v", err)
	}
}

func TestSessionListOutputFormatsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}

	humanOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "human")
	if err != nil {
		t.Fatalf("session list human error = %v", err)
	}
	if !strings.Contains(humanOut, "Sessions") || !strings.Contains(humanOut, created.ID) {
		t.Fatalf("human output = %q, want session table", humanOut)
	}

	jsonOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("session list json error = %v", err)
	}
	var listed []SessionRecord
	if err := json.Unmarshal([]byte(jsonOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed = %#v, want one created session", listed)
	}

	toonOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--all", "-o", "toon")
	if err != nil {
		t.Fatalf("session list toon error = %v", err)
	}
	if !strings.Contains(toonOut, "sessions[1]{id,name,agent_name,state,workspace,updated_at}:") {
		t.Fatalf("toon output = %q, want TOON table", toonOut)
	}
}

func TestSessionEventsFollowIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--cwd", h.workspace, "-o", "json")
	if err != nil {
		t.Fatalf("session new error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}

	if _, _, err := executeRootCommand(t, h.deps, "session", "prompt", created.ID, "hello", "-o", "json"); err != nil {
		t.Fatalf("session prompt error = %v", err)
	}

	cmd := newRootCommand(h.deps)
	var stderr bytes.Buffer
	stdout := &lockedBuffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"session", "events", created.ID, "--follow", "-o", "json"})

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		done <- cmd.ExecuteContext(ctx)
	}()

	waitForCondition(t, 3*time.Second, func() bool {
		return strings.Contains(stdout.String(), `"type":"agent_message"`)
	})

	if _, _, err := executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json"); err != nil {
		t.Fatalf("daemon stop error = %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("follow command error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("follow output lines = %d, want at least 2", len(lines))
	}
	var sawAgentMessage bool
	for _, line := range lines {
		var event SessionEventRecord
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("json.Unmarshal(follow line) error = %v; line=%s", err, line)
		}
		if event.Type == "agent_message" {
			sawAgentMessage = true
		}
	}
	if !sawAgentMessage {
		t.Fatalf("follow output = %q, want streamed agent_message event", stdout.String())
	}
}

func TestWorkspaceCommandsIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	addOut, _, err := executeRootCommand(t, h.deps, "workspace", "add", h.workspace, "--name", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("workspace add error = %v", err)
	}
	var registered WorkspaceRecord
	if err := json.Unmarshal([]byte(addOut), &registered); err != nil {
		t.Fatalf("json.Unmarshal(workspace add) error = %v", err)
	}
	if registered.ID == "" {
		t.Fatal("expected registered workspace id")
	}

	infoOut, _, err := executeRootCommand(t, h.deps, "workspace", "info", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("workspace info error = %v", err)
	}
	var detail WorkspaceDetailRecord
	if err := json.Unmarshal([]byte(infoOut), &detail); err != nil {
		t.Fatalf("json.Unmarshal(workspace info) error = %v", err)
	}
	if detail.Workspace.ID != registered.ID {
		t.Fatalf("workspace info id = %q, want %q", detail.Workspace.ID, registered.ID)
	}

	sessionOut, _, err := executeRootCommand(t, h.deps, "session", "new", "--agent", "coder", "--name", "demo", "--workspace", "alpha", "-o", "json")
	if err != nil {
		t.Fatalf("session new with workspace error = %v", err)
	}
	var created SessionRecord
	if err := json.Unmarshal([]byte(sessionOut), &created); err != nil {
		t.Fatalf("json.Unmarshal(session new) error = %v", err)
	}
	if created.WorkspaceID != registered.ID {
		t.Fatalf("created.WorkspaceID = %q, want %q", created.WorkspaceID, registered.ID)
	}

	listOut, _, err := executeRootCommand(t, h.deps, "session", "list", "--workspace", "alpha", "--all", "-o", "json")
	if err != nil {
		t.Fatalf("session list --workspace error = %v", err)
	}
	var listed []SessionRecord
	if err := json.Unmarshal([]byte(listOut), &listed); err != nil {
		t.Fatalf("json.Unmarshal(session list) error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("listed = %#v, want one workspace-filtered session", listed)
	}
}

func TestMemoryWriteListIntegration(t *testing.T) {
	t.Parallel()

	h := newIntegrationHarness(t)
	mustExecuteRoot(t, h.deps, "daemon", "start", "-o", "json")
	defer func() {
		_, _, _ = executeRootCommand(t, h.deps, "daemon", "stop", "-o", "json")
		_ = h.runner.waitForExit()
	}()

	if _, _, err := executeRootCommand(t, h.deps, "memory", "write", "prefs.md", "--type", "user", "--description", "cli memory", "--content", "remember this", "-o", "json"); err != nil {
		t.Fatalf("memory write error = %v", err)
	}

	listOut, _, err := executeRootCommand(t, h.deps, "memory", "list", "--scope", "global", "-o", "json")
	if err != nil {
		t.Fatalf("memory list error = %v", err)
	}

	var memories []memoryListItem
	if err := json.Unmarshal([]byte(listOut), &memories); err != nil {
		t.Fatalf("json.Unmarshal(memory list) error = %v; out=%s", err, listOut)
	}
	if len(memories) != 1 || memories[0].Filename != "prefs.md" {
		t.Fatalf("memories = %#v, want prefs.md", memories)
	}
}

type integrationHarness struct {
	deps      commandDeps
	homePaths aghconfig.HomePaths
	workspace string
	runner    *integrationDaemon
}

type integrationDreamTrigger struct {
	enabled   bool
	triggered bool
	reason    string
	last      time.Time
}

func (t *integrationDreamTrigger) Trigger(context.Context, string) (bool, string, error) {
	return t.triggered, t.reason, nil
}

func (t *integrationDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return t.last, nil
}

func (t *integrationDreamTrigger) Enabled() bool {
	return t.enabled
}

type integrationDaemon struct {
	t         *testing.T
	homePaths aghconfig.HomePaths
	cfg       aghconfig.Config
	pid       int
	startedAt time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	done    chan error
}

type integrationDaemonProcess struct {
	pid  int
	done <-chan error
}

type integrationNotifierFanout struct {
	notifiers []session.Notifier
}

type integrationDriver struct {
	mu       sync.Mutex
	nextPID  int
	nextSess int
	states   map[*session.AgentProcess]chan struct{}
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func newIntegrationHarness(t *testing.T) integrationHarness {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	socketPath := shortSocketPath(t)
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	writeAgentDef(t, homePaths, "coder")

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Daemon.Socket = socketPath
	cfg.Providers = map[string]aghconfig.ProviderConfig{
		"fake": {Command: "fake-agent"},
	}

	runner := &integrationDaemon{
		t:         t,
		homePaths: homePaths,
		cfg:       cfg,
		pid:       4242,
		startedAt: time.Now().UTC(),
	}

	deps := commandDeps{
		loadConfig: func() (aghconfig.Config, error) {
			return cfg, nil
		},
		resolveHome: func() (aghconfig.HomePaths, error) {
			return homePaths, nil
		},
		ensureHome: aghconfig.EnsureHomeLayout,
		newClient:  NewClient,
		newDaemon: func() (daemonRunner, error) {
			return runner, nil
		},
		readDaemonInfo: aghdaemon.ReadInfo,
		signalProcess:  runner.signalProcess,
		processAlive:   runner.processAlive,
		getwd: func() (string, error) {
			return t.TempDir(), nil
		},
		getenv: func(string) string { return "" },
		now: func() time.Time {
			return time.Now().UTC()
		},
		pollInterval: 10 * time.Millisecond,
		startTimeout: 5 * time.Second,
		stopTimeout:  5 * time.Second,
		spawnDetached: func(aghconfig.HomePaths) (daemonProcess, error) {
			return runner.spawnDetached()
		},
	}

	return integrationHarness{
		deps:      deps,
		homePaths: homePaths,
		workspace: t.TempDir(),
		runner:    runner,
	}
}

func (p *integrationDaemonProcess) PID() int {
	return p.pid
}

func (p *integrationDaemonProcess) Wait() error {
	return <-p.done
}

func (d *integrationDaemon) spawnDetached() (daemonProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.running {
		return nil, errors.New("integration daemon already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	d.running = true
	d.cancel = cancel
	d.done = done

	go func() {
		err := d.Run(ctx)
		done <- err
		close(done)
		d.mu.Lock()
		d.running = false
		d.cancel = nil
		d.done = nil
		d.mu.Unlock()
	}()

	return &integrationDaemonProcess{pid: d.pid, done: done}, nil
}

func (d *integrationDaemon) Run(ctx context.Context) error {
	registry, err := store.OpenGlobalDB(context.Background(), d.homePaths.DatabaseFile)
	if err != nil {
		return fmt.Errorf("open global db: %w", err)
	}
	defer func() {
		_ = registry.Close(context.Background())
	}()

	fanout := &integrationNotifierFanout{}
	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(d.homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(string) (aghconfig.Config, error) { return d.cfg, nil }),
	)
	if err != nil {
		return fmt.Errorf("new workspace resolver: %w", err)
	}
	manager, err := session.NewManager(
		session.WithHomePaths(d.homePaths),
		session.WithWorkspaceResolver(resolver),
		session.WithLogger(discardLogger()),
		session.WithDriver(newIntegrationDriver()),
		session.WithNotifier(fanout),
	)
	if err != nil {
		return fmt.Errorf("new session manager: %w", err)
	}

	observer, err := observe.New(
		context.Background(),
		observe.WithHomePaths(d.homePaths),
		observe.WithRegistry(registry),
		observe.WithSessionSource(manager),
		observe.WithLogger(discardLogger()),
		observe.WithStartTime(d.startedAt),
	)
	if err != nil {
		return fmt.Errorf("new observer: %w", err)
	}
	defer func() {
		_ = observer.Close(context.Background())
	}()
	fanout.notifiers = append(fanout.notifiers, observer)

	memoryStore := memory.NewStore(d.homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		return fmt.Errorf("ensure memory dirs: %w", err)
	}
	dreamTrigger := &integrationDreamTrigger{
		enabled:   true,
		triggered: true,
		last:      time.Date(2026, 4, 4, 3, 30, 0, 0, time.UTC),
	}

	server, err := udsapi.New(
		udsapi.WithHomePaths(d.homePaths),
		udsapi.WithConfig(d.cfg),
		udsapi.WithSocketPath(d.cfg.Daemon.Socket),
		udsapi.WithLogger(discardLogger()),
		udsapi.WithStartedAt(d.startedAt),
		udsapi.WithPollInterval(10*time.Millisecond),
		udsapi.WithSessionManager(manager),
		udsapi.WithObserver(observer),
		udsapi.WithWorkspaceResolver(resolver),
		udsapi.WithMemoryStore(memoryStore),
		udsapi.WithDreamTrigger(dreamTrigger),
	)
	if err != nil {
		return fmt.Errorf("new uds server: %w", err)
	}

	if err := server.Start(context.Background()); err != nil {
		return fmt.Errorf("start uds server: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, info := range manager.List() {
			if info == nil || info.State == session.StateStopped {
				continue
			}
			_ = manager.Stop(shutdownCtx, info.ID)
		}
		_ = server.Shutdown(shutdownCtx)
		_ = aghdaemon.RemoveInfo(d.homePaths.DaemonInfo)
	}()

	if err := aghdaemon.WriteInfo(d.homePaths.DaemonInfo, aghdaemon.Info{
		PID:       d.pid,
		Port:      d.cfg.HTTP.Port,
		StartedAt: d.startedAt,
	}); err != nil {
		return fmt.Errorf("write daemon info: %w", err)
	}

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

func (d *integrationDaemon) signalProcess(pid int, sig syscall.Signal) error {
	d.mu.Lock()
	cancel := d.cancel
	running := d.running
	d.mu.Unlock()

	if !running || pid != d.pid {
		return fmt.Errorf("integration daemon pid %d is not running", pid)
	}
	if sig != syscall.SIGTERM {
		return fmt.Errorf("unsupported signal %v", sig)
	}
	cancel()
	return nil
}

func (d *integrationDaemon) processAlive(pid int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running && pid == d.pid
}

func (d *integrationDaemon) waitForExit() error {
	d.mu.Lock()
	done := d.done
	d.mu.Unlock()
	if done == nil {
		return nil
	}
	return <-done
}

func (f *integrationNotifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *integrationNotifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	for _, notifier := range f.notifiers {
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}

func newIntegrationDriver() *integrationDriver {
	return &integrationDriver{
		nextPID:  2000,
		nextSess: 1,
		states:   make(map[*session.AgentProcess]chan struct{}),
	}
}

func (d *integrationDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.nextPID++
	d.nextSess++
	done := make(chan struct{})
	sessionID := strings.TrimSpace(opts.ResumeSessionID)
	if sessionID == "" {
		sessionID = fmt.Sprintf("acp-session-%d", d.nextSess)
	}

	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       d.nextPID,
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: sessionID,
		Caps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModels:     []string{"fake-model"},
		},
		StartedAt: time.Now().UTC(),
		Done:      done,
		Wait: func() error {
			<-done
			return nil
		},
	})
	d.states[proc] = done
	return proc, nil
}

func (d *integrationDriver) Prompt(_ context.Context, proc *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	ch := make(chan acp.AgentEvent, 2)
	ch <- acp.AgentEvent{
		Type:      "agent_message",
		SessionID: proc.SessionID,
		TurnID:    req.TurnID,
		Timestamp: time.Now().UTC(),
		Text:      req.Message,
	}
	ch <- acp.AgentEvent{
		Type:       "done",
		SessionID:  proc.SessionID,
		TurnID:     req.TurnID,
		Timestamp:  time.Now().UTC(),
		StopReason: "end_turn",
	}
	close(ch)
	return ch, nil
}

func (d *integrationDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *integrationDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	done, ok := d.states[proc]
	if !ok {
		return nil
	}
	select {
	case <-done:
	default:
		close(done)
	}
	delete(d.states, proc)
	return nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	root := filepath.Join(os.TempDir(), fmt.Sprintf("agh-cli-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	return filepath.Join(root, "daemon.sock")
}

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()

	agentDir := filepath.Join(homePaths.AgentsDir, name)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", agentDir, err)
	}
	content := strings.Join([]string{
		"---",
		"name: " + name,
		"provider: fake",
		"model: fake-model",
		"---",
		"",
		"You are the integration test agent.",
	}, "\n")
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(AGENT.md) error = %v", err)
	}
}

func mustExecuteRoot(t *testing.T, deps commandDeps, args ...string) string {
	t.Helper()

	stdout, stderr, err := executeRootCommand(t, deps, args...)
	if err != nil {
		t.Fatalf("executeRootCommand(%v) error = %v; stderr=%s", args, err, stderr)
	}
	return stdout
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for condition")
}
