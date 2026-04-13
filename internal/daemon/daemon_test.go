package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestAcquireLockSucceedsWithoutExistingLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "daemon.lock")

	lock, err := AcquireLock(lockPath, os.Getpid())
	if err != nil {
		t.Fatalf("AcquireLock() error = %v", err)
	}
	t.Cleanup(func() {
		if err := lock.Release(); err != nil {
			t.Fatalf("lock.Release() error = %v", err)
		}
	})

	if got := lock.StalePID(); got != 0 {
		t.Fatalf("lock.StalePID() = %d, want 0", got)
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("os.ReadFile(lock) error = %v", err)
	}
	if got, want := strings.TrimSpace(string(data)), strconvString(os.Getpid()); got != want {
		t.Fatalf("lock file contents = %q, want %q", got, want)
	}
}

func TestAcquireLockFailsWhenAnotherDaemonHoldsTheLock(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "daemon.lock")

	first, err := AcquireLock(lockPath, os.Getpid())
	if err != nil {
		t.Fatalf("AcquireLock(first) error = %v", err)
	}
	t.Cleanup(func() {
		if err := first.Release(); err != nil {
			t.Fatalf("first.Release() error = %v", err)
		}
	})

	second, err := AcquireLock(lockPath, os.Getpid())
	if second != nil {
		t.Fatalf("AcquireLock(second) lock = %#v, want nil", second)
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("AcquireLock(second) error = %v, want ErrAlreadyRunning", err)
	}
}

func TestAcquireLockReclaimsStalePID(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "daemon.lock")
	if err := os.WriteFile(lockPath, []byte("999999\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(lock) error = %v", err)
	}

	lock, err := acquireLock(lockPath, 1234, lockDeps{
		newFlock:     func(path string) *flock.Flock { return flock.New(path) },
		processAlive: func(pid int) bool { return false },
	})
	if err != nil {
		t.Fatalf("acquireLock() error = %v", err)
	}
	t.Cleanup(func() {
		if err := lock.Release(); err != nil {
			t.Fatalf("lock.Release() error = %v", err)
		}
	})

	if got, want := lock.StalePID(), 999999; got != want {
		t.Fatalf("lock.StalePID() = %d, want %d", got, want)
	}
}

func TestInfoWriteReadAndRemoveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	now := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	info := Info{
		PID:       4242,
		Port:      2123,
		StartedAt: now,
		Network: &NetworkInfo{
			Enabled:      true,
			Status:       network.StatusRunning,
			ListenerHost: "127.0.0.1",
			ListenerPort: 4222,
		},
	}

	if err := WriteInfo(path, info); err != nil {
		t.Fatalf("WriteInfo() error = %v", err)
	}

	got, err := ReadInfo(path)
	if err != nil {
		t.Fatalf("ReadInfo() error = %v", err)
	}
	if got.PID != info.PID || got.Port != info.Port || !got.StartedAt.Equal(info.StartedAt) {
		t.Fatalf("ReadInfo() = %#v, want %#v", got, info)
	}

	if err := RemoveInfo(path); err != nil {
		t.Fatalf("RemoveInfo() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json exists after RemoveInfo(): stat error = %v, want os.ErrNotExist", err)
	}
}

func TestBootWithNetworkDisabledKeepsDaemonOperational(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Network.Enabled = false

	d := newTestDaemon(t, homePaths, cfg)
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
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
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.network != nil {
		t.Fatal("boot() network runtime = non-nil, want nil when disabled")
	}
	if d.info.Network == nil {
		t.Fatal("boot() daemon info network = nil, want disabled diagnostics")
	}
	if d.info.Network.Enabled {
		t.Fatalf("boot() daemon info network enabled = %v, want false", d.info.Network.Enabled)
	}
	if got, want := d.info.Network.Status, network.StatusDisabled; got != want {
		t.Fatalf("boot() daemon info network status = %q, want %q", got, want)
	}
}

func TestBootEnabledNetworkLateBindsSessionCallbacksAndPersistsSafeDiagnostics(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Network.Enabled = true

	bindableSessions := newFakeNetworkBindableSessionManager()
	d := newTestDaemon(t, homePaths, cfg)
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
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
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.network == nil {
		t.Fatal("boot() network runtime = nil, want initialized manager")
	}
	if bindableSessions.currentNetworkPeerLifecycle() == nil {
		t.Fatal("boot() did not late-bind network peer lifecycle")
	}
	if bindableSessions.currentTurnEndNotifier() == nil {
		t.Fatal("boot() did not late-bind turn-end notifier")
	}

	info, err := ReadInfo(homePaths.DaemonInfo)
	if err != nil {
		t.Fatalf("ReadInfo(daemon.json) error = %v", err)
	}
	if info.Network == nil {
		t.Fatal("daemon info network diagnostics = nil, want populated diagnostics")
	}
	if !info.Network.Enabled {
		t.Fatal("daemon info network enabled = false, want true")
	}
	if got, want := info.Network.Status, network.StatusRunning; got != want {
		t.Fatalf("daemon info network status = %q, want %q", got, want)
	}
	if info.Network.ListenerPort <= 0 {
		t.Fatalf("daemon info network listener port = %d, want positive", info.Network.ListenerPort)
	}

	rawInfo, err := os.ReadFile(homePaths.DaemonInfo)
	if err != nil {
		t.Fatalf("os.ReadFile(daemon.json) error = %v", err)
	}
	if strings.Contains(strings.ToLower(string(rawInfo)), "token") {
		t.Fatalf("daemon info leaked credentials: %s", string(rawInfo))
	}
}

func TestBootEnabledNetworkRejectsSessionManagersMissingBindingSurface(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Network.Enabled = true

	d := newTestDaemon(t, homePaths, cfg)
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}

	err := d.boot(testutil.Context(t))
	if !errors.Is(err, errMissingNetworkBindingSurface) {
		t.Fatalf("boot() error = %v, want errMissingNetworkBindingSurface", err)
	}
}

func TestBootRemovesStaleSocketAndCleansOrphans(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	staleSocket := cfg.Daemon.Socket
	if err := os.WriteFile(staleSocket, []byte("stale"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(socket) error = %v", err)
	}

	d := newTestDaemon(t, homePaths, cfg)
	d.pid = func() int { return 777 }
	d.acquireLock = func(path string, pid int) (*Lock, error) {
		return &Lock{path: path, stalePID: 444}, nil
	}

	registry := &recordingRegistry{path: homePaths.DatabaseFile}
	observer := &fakeObserver{result: store.ReconcileResult{Indexed: []string{"sess-a"}}}
	sessionManager := &fakeSessionManager{}
	var signals []string
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return registry, nil
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return sessionManager, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return observer, nil
	}
	d.listProcesses = func(context.Context) ([]processInfo, error) {
		return []processInfo{{PID: 1001, PPID: 444}, {PID: 2002, PPID: 111}}, nil
	}
	d.orphanGraceWait = 2 * time.Millisecond
	d.orphanPollWait = time.Millisecond
	d.signalProcess = func(pid int, sig syscall.Signal) error {
		signals = append(signals, sig.String()+":"+strconvString(pid))
		return nil
	}
	d.processAlive = func(pid int) bool { return pid == 1001 }

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	socketInfo, err := os.Lstat(staleSocket)
	if err != nil {
		t.Fatalf("os.Lstat(socket) error = %v", err)
	}
	if socketInfo.Mode()&os.ModeSocket == 0 {
		t.Fatalf("socket mode = %v, want unix socket", socketInfo.Mode())
	}
	if !observer.reconciled {
		t.Fatal("boot() did not call observer.Reconcile")
	}
	if got, want := signals, []string{"terminated:1001", "killed:1001"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("cleanup orphan signals = %#v, want %#v", got, want)
	}

	info, err := ReadInfo(homePaths.DaemonInfo)
	if err != nil {
		t.Fatalf("ReadInfo(daemon.json) error = %v", err)
	}
	if got, want := info.PID, 777; got != want {
		t.Fatalf("daemon info pid = %d, want %d", got, want)
	}
}

func TestCleanupOrphansAllowsGracefulExitBeforeSIGKILL(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	d := newTestDaemon(t, homePaths, cfg)

	var (
		signals   []string
		aliveCall int
	)
	d.listProcesses = func(context.Context) ([]processInfo, error) {
		return []processInfo{{PID: 1001, PPID: 444}}, nil
	}
	d.orphanGraceWait = 10 * time.Millisecond
	d.orphanPollWait = time.Millisecond
	d.signalProcess = func(pid int, sig syscall.Signal) error {
		signals = append(signals, sig.String()+":"+strconvString(pid))
		return nil
	}
	d.processAlive = func(pid int) bool {
		aliveCall++
		return aliveCall == 1
	}

	if err := d.cleanupOrphans(testutil.Context(t), 444); err != nil {
		t.Fatalf("cleanupOrphans() error = %v", err)
	}
	if got, want := signals, []string{"terminated:1001"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("cleanup orphan signals = %#v, want %#v", got, want)
	}
}

func TestBootRejectsConcurrentCallWhileFirstBootIsInProgress(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	d := newTestDaemon(t, homePaths, cfg)

	loadStarted := make(chan struct{})
	releaseLoad := make(chan struct{})
	d.loadConfig = func() (aghconfig.Config, error) {
		close(loadStarted)
		<-releaseLoad
		return cfg, nil
	}
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
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

	firstBoot := make(chan error, 1)
	go func() {
		firstBoot <- d.boot(testutil.Context(t))
	}()

	<-loadStarted
	if err := d.boot(testutil.Context(t)); err == nil || !strings.Contains(err.Error(), "already booted") {
		t.Fatalf("concurrent boot error = %v, want already booted", err)
	}

	close(releaseLoad)
	if err := <-firstBoot; err != nil {
		t.Fatalf("first boot error = %v", err)
	}
	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestShutdownTearsDownInRequiredOrder(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	d := newTestDaemon(t, homePaths, cfg)

	var events []string
	d.extensions = &fakeExtensionRuntime{
		onStop: func() {
			events = append(events, "extensions")
		},
	}
	d.automation = &fakeAutomationManager{
		onShutdown: func() {
			events = append(events, "automation")
		},
	}
	d.sessions = &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}, {ID: "sess-b"}},
		onStop: func(id string) {
			events = append(events, "session:"+id)
		},
	}
	d.network = &fakeNetworkRuntime{
		onShutdown: func() {
			events = append(events, "network")
		},
	}
	d.httpServer = &fakeServer{name: "http", onShutdown: func() { events = append(events, "http") }}
	d.udsServer = &fakeServer{name: "uds", onShutdown: func() { events = append(events, "uds") }}
	d.registry = &recordingRegistry{
		path: homePaths.DatabaseFile,
		onClose: func() {
			events = append(events, "db")
		},
	}
	d.hooks = &fakeHookRuntime{
		onClose: func() {
			events = append(events, "hooks")
		},
	}
	d.lock = &Lock{
		path: homePaths.DaemonLock,
		releaseFn: func() error {
			events = append(events, "lock")
			return nil
		},
	}
	d.closeLogger = func() error {
		events = append(events, "logger")
		return nil
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	want := []string{"extensions", "automation", "session:sess-a", "session:sess-b", "http", "uds", "network", "hooks", "db", "lock", "logger"}
	if !testutil.EqualStringSlices(events, want) {
		t.Fatalf("Shutdown() order = %#v, want %#v", events, want)
	}
}

func TestBootExtensionsBuildsManagerWhenNoExtensionsInstalled(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	homePaths := testHomePaths(t)
	d := newTestDaemon(t, homePaths, testConfig(t, homePaths))

	runtime := &fakeExtensionRuntime{}
	var managerBuilt bool
	d.newExtensionManager = func(extensionManagerDeps) extensionRuntime {
		managerBuilt = true
		return runtime
	}

	rebuilds := 0
	state := &bootState{
		logger:   discardLogger(),
		registry: db,
		sessions: &fakeSessionManager{},
		observer: &fakeObserver{},
		hooks: &fakeHookRuntime{
			onRebuild: func(context.Context) error {
				rebuilds++
				return nil
			},
		},
	}
	cleanup := &bootCleanup{}

	if err := d.bootExtensions(testutil.Context(t), state, cleanup); err != nil {
		t.Fatalf("bootExtensions() error = %v", err)
	}

	if !managerBuilt {
		t.Fatal("bootExtensions() did not build an extension manager")
	}
	if runtime.startCount != 1 {
		t.Fatalf("extension runtime start count = %d, want 1", runtime.startCount)
	}
	if rebuilds != 1 {
		t.Fatalf("hook rebuild count = %d, want 1", rebuilds)
	}
	if state.extensions != runtime {
		t.Fatalf("state.extensions = %#v, want runtime", state.extensions)
	}
	if state.deps.Extensions == nil {
		t.Fatal("state.deps.Extensions = nil, want extension service")
	}
	if len(cleanup.fns) != 1 {
		t.Fatalf("cleanup fns = %d, want 1", len(cleanup.fns))
	}
}

func TestBootExtensionsBuildsManagerDepsAndRebuildsHooks(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	installDaemonTestExtension(t, db, "ext-present", daemonTestExtensionOptions{}, true)

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	memStore := memory.NewStore(t.TempDir())
	skillsRegistry := skills.NewRegistry(skills.RegistryConfig{})
	sessions := &fakeSessionManager{}
	observer := &fakeObserver{}
	logger := discardLogger()
	runtime := &fakeExtensionRuntime{}

	var captured extensionManagerDeps
	d := newTestDaemon(t, homePaths, cfg)
	d.newExtensionManager = func(deps extensionManagerDeps) extensionRuntime {
		captured = deps
		return runtime
	}

	rebuilds := 0
	state := &bootState{
		logger:         logger,
		registry:       db,
		memoryStore:    memStore,
		skillsRegistry: skillsRegistry,
		sessions:       sessions,
		observer:       observer,
		hooks: &fakeHookRuntime{
			onRebuild: func(context.Context) error {
				rebuilds++
				return nil
			},
		},
	}
	cleanup := &bootCleanup{}

	if err := d.bootExtensions(testutil.Context(t), state, cleanup); err != nil {
		t.Fatalf("bootExtensions() error = %v", err)
	}

	if runtime.startCount != 1 {
		t.Fatalf("extension runtime start count = %d, want 1", runtime.startCount)
	}
	if rebuilds != 1 {
		t.Fatalf("hook rebuild count = %d, want 1", rebuilds)
	}
	if captured.Registry == nil {
		t.Fatal("captured extension registry = nil")
	}
	if captured.Sessions != sessions {
		t.Fatal("captured sessions dependency mismatch")
	}
	if captured.MemoryStore != memStore {
		t.Fatal("captured memory store dependency mismatch")
	}
	if captured.Observer != observer {
		t.Fatal("captured observer dependency mismatch")
	}
	if captured.SkillsRegistry != skillsRegistry {
		t.Fatal("captured skills registry dependency mismatch")
	}
	if captured.Logger != logger {
		t.Fatal("captured logger dependency mismatch")
	}
	if state.extensions != runtime {
		t.Fatalf("state.extensions = %#v, want runtime", state.extensions)
	}
	if len(cleanup.fns) != 1 {
		t.Fatalf("cleanup fns = %d, want 1", len(cleanup.fns))
	}
}

func TestBootExtensionsLogsStartFailureAndContinues(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	installDaemonTestExtension(t, db, "ext-broken", daemonTestExtensionOptions{}, true)

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	runtime := &fakeExtensionRuntime{startErr: errors.New("boom")}
	homePaths := testHomePaths(t)
	d := newTestDaemon(t, homePaths, testConfig(t, homePaths))
	d.newExtensionManager = func(extensionManagerDeps) extensionRuntime {
		return runtime
	}

	rebuilds := 0
	state := &bootState{
		logger:   logger,
		registry: db,
		sessions: &fakeSessionManager{},
		observer: &fakeObserver{},
		bridges:  &bridgeRuntime{broker: bridgepkg.NewBroker(nil)},
		hooks: &fakeHookRuntime{
			onRebuild: func(context.Context) error {
				rebuilds++
				return nil
			},
		},
	}
	cleanup := &bootCleanup{}

	if err := d.bootExtensions(testutil.Context(t), state, cleanup); err != nil {
		t.Fatalf("bootExtensions() error = %v, want nil", err)
	}

	if runtime.startCount != 1 {
		t.Fatalf("extension runtime start count = %d, want 1", runtime.startCount)
	}
	if rebuilds != 0 {
		t.Fatalf("hook rebuild count = %d, want 0 after failed start", rebuilds)
	}
	if len(cleanup.fns) != 1 {
		t.Fatalf("cleanup fns = %d, want 1", len(cleanup.fns))
	}
	if state.currentExtensionRuntime() != nil {
		t.Fatalf("state.extensions = %#v, want nil after failed start", state.currentExtensionRuntime())
	}
	if state.deps.Extensions != nil {
		t.Fatalf("state.deps.Extensions = %#v, want nil after failed start", state.deps.Extensions)
	}
	if state.bridges.extensions != nil {
		t.Fatalf("state.bridges.extensions = %#v, want nil after failed start", state.bridges.extensions)
	}
	if !strings.Contains(logBuffer.String(), "extension manager start failed") {
		t.Fatalf("log output = %q, want extension start failure message", logBuffer.String())
	}
}

func TestBootAutomationBuildsManagerDepsAndAttachesHookBoundary(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Automation.Enabled = true

	resolver, err := workspacepkg.NewResolver(
		db,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	baseLifecycle := &recordingNotifier{}
	managerLifecycle := &recordingNotifier{}
	baseTelemetry := &recordingHookTelemetrySink{}
	managerTelemetry := &recordingHookTelemetrySink{}
	manager := &fakeAutomationManager{
		sessionObserver:   managerLifecycle,
		hookTelemetrySink: managerTelemetry,
		status:            automationpkg.ManagerStatus{Running: true, SchedulerRunning: true},
	}

	var captured automationManagerDeps
	d := newTestDaemon(t, homePaths, cfg)
	d.newAutomationManager = func(deps automationManagerDeps) (automationRuntime, error) {
		captured = deps
		return manager, nil
	}

	state := &bootState{
		cfg:                cfg,
		logger:             discardLogger(),
		registry:           db,
		sessions:           &fakeSessionManager{},
		workspaceResolver:  resolver,
		lifecycleObservers: newSessionLifecycleFanout(baseLifecycle),
		hookTelemetrySinks: newHookTelemetryFanout(baseTelemetry),
	}
	cleanup := &bootCleanup{}

	if err := d.bootAutomation(testutil.Context(t), state, cleanup); err != nil {
		t.Fatalf("bootAutomation() error = %v", err)
	}

	if captured.Store == nil {
		t.Fatal("captured.Store = nil")
	}
	if captured.Sessions != state.sessions {
		t.Fatal("captured.Sessions dependency mismatch")
	}
	if captured.WorkspaceResolver != resolver {
		t.Fatal("captured.WorkspaceResolver dependency mismatch")
	}
	if got, want := captured.Config.Enabled, cfg.Automation.Enabled; got != want {
		t.Fatalf("captured.Config.Enabled = %v, want %v", got, want)
	}
	if got, want := captured.Config.Timezone, cfg.Automation.Timezone; got != want {
		t.Fatalf("captured.Config.Timezone = %q, want %q", got, want)
	}
	if got, want := captured.Config.MaxConcurrentJobs, cfg.Automation.MaxConcurrentJobs; got != want {
		t.Fatalf("captured.Config.MaxConcurrentJobs = %d, want %d", got, want)
	}
	if got, want := captured.Config.DefaultFireLimit, cfg.Automation.DefaultFireLimit; got != want {
		t.Fatalf("captured.Config.DefaultFireLimit = %#v, want %#v", got, want)
	}
	if got, want := captured.GlobalWorkspacePath, homePaths.HomeDir; got != want {
		t.Fatalf("captured.GlobalWorkspacePath = %q, want %q", got, want)
	}
	if manager.startCount != 1 {
		t.Fatalf("manager start count = %d, want 1", manager.startCount)
	}
	if state.automation != manager {
		t.Fatalf("state.automation = %#v, want manager", state.automation)
	}
	if state.deps.Automation != manager {
		t.Fatalf("state.deps.Automation = %#v, want manager", state.deps.Automation)
	}
	if len(cleanup.fns) != 1 {
		t.Fatalf("cleanup fns = %d, want 1", len(cleanup.fns))
	}

	state.lifecycleObservers.OnSessionCreated(testutil.Context(t), &session.Session{ID: "sess-automation"})
	if got, want := managerLifecycle.events, []string{"created"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("manager lifecycle events = %#v, want %#v", got, want)
	}
	if got, want := baseLifecycle.events, []string{"created"}; !testutil.EqualStringSlices(got, want) {
		t.Fatalf("base lifecycle events = %#v, want %#v", got, want)
	}

	if err := state.hookTelemetrySinks.WriteHookRecord(testutil.Context(t), "sess-automation", hookspkg.HookRunRecord{HookName: "post-stop"}); err != nil {
		t.Fatalf("WriteHookRecord() error = %v", err)
	}
	if got, want := baseTelemetry.count(), 1; got != want {
		t.Fatalf("base telemetry count = %d, want %d", got, want)
	}
	if got, want := managerTelemetry.count(), 1; got != want {
		t.Fatalf("manager telemetry count = %d, want %d", got, want)
	}
}

func TestHooksNotifierNoopDispatchesWithoutRuntime(t *testing.T) {
	t.Parallel()

	notifier := newHooksNotifier(discardLogger(), func() time.Time {
		return time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
	})

	notifier.OnSessionCreated(testutil.Context(t), &session.Session{ID: "sess-created"})
	notifier.OnSessionStopped(testutil.Context(t), &session.Session{ID: "sess-stopped"})

	if _, err := notifier.DispatchSessionPreCreate(testutil.Context(t), hookspkg.SessionPreCreatePayload{}); err != nil {
		t.Fatalf("DispatchSessionPreCreate() error = %v", err)
	}
	if _, err := notifier.DispatchSessionPreResume(testutil.Context(t), hookspkg.SessionPreResumePayload{}); err != nil {
		t.Fatalf("DispatchSessionPreResume() error = %v", err)
	}
	if _, err := notifier.DispatchSessionPostResume(testutil.Context(t), hookspkg.SessionPostResumePayload{}); err != nil {
		t.Fatalf("DispatchSessionPostResume() error = %v", err)
	}
	if _, err := notifier.DispatchSessionPreStop(testutil.Context(t), hookspkg.SessionPreStopPayload{}); err != nil {
		t.Fatalf("DispatchSessionPreStop() error = %v", err)
	}
	if _, err := notifier.DispatchInputPreSubmit(testutil.Context(t), hookspkg.InputPreSubmitPayload{}); err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v", err)
	}
	if _, err := notifier.DispatchPromptPostAssemble(testutil.Context(t), hookspkg.PromptPayload{}); err != nil {
		t.Fatalf("DispatchPromptPostAssemble() error = %v", err)
	}
	if _, err := notifier.DispatchEventPreRecord(testutil.Context(t), hookspkg.EventPreRecordPayload{}); err != nil {
		t.Fatalf("DispatchEventPreRecord() error = %v", err)
	}
	if _, err := notifier.DispatchEventPostRecord(testutil.Context(t), hookspkg.EventPostRecordPayload{}); err != nil {
		t.Fatalf("DispatchEventPostRecord() error = %v", err)
	}
	if _, err := notifier.DispatchAgentPreStart(testutil.Context(t), hookspkg.AgentPreStartPayload{}); err != nil {
		t.Fatalf("DispatchAgentPreStart() error = %v", err)
	}
	if _, err := notifier.DispatchAgentSpawned(testutil.Context(t), hookspkg.AgentSpawnedPayload{}); err != nil {
		t.Fatalf("DispatchAgentSpawned() error = %v", err)
	}
	if _, err := notifier.DispatchAgentCrashed(testutil.Context(t), hookspkg.AgentCrashedPayload{}); err != nil {
		t.Fatalf("DispatchAgentCrashed() error = %v", err)
	}
	if _, err := notifier.DispatchAgentStopped(testutil.Context(t), hookspkg.AgentStoppedPayload{}); err != nil {
		t.Fatalf("DispatchAgentStopped() error = %v", err)
	}
}

func TestDaemonExtensionServiceInstallStatusAndDisable(t *testing.T) {
	t.Parallel()

	db := openDaemonTestGlobalDB(t)
	registry := extensionpkg.NewRegistry(db.DB())
	manager := extensionpkg.NewManager(registry, extensionpkg.WithLogger(discardLogger()))
	if err := manager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("manager.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Stop(testutil.Context(t)); err != nil {
			t.Fatalf("manager.Stop() error = %v", err)
		}
	})

	rebuilds := 0
	fixedNow := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	service := newDaemonExtensionService(
		registry,
		manager,
		&fakeHookRuntime{
			onRebuild: func(context.Context) error {
				rebuilds++
				return nil
			},
		},
		discardLogger(),
		func() time.Time { return fixedNow },
	)

	fixtureDir := filepath.Join(t.TempDir(), "service-ext")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", fixtureDir, err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "extension.toml"), []byte(daemonTestExtensionManifest("service-ext", daemonTestExtensionOptions{
		runtimeCommand: daemonExtensionHelperCommand(t),
		runtimeArgs:    daemonExtensionHelperArgs(),
		runtimeEnv:     daemonExtensionHelperEnv(""),
	})), 0o644); err != nil {
		t.Fatalf("os.WriteFile(extension.toml) error = %v", err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(fixtureDir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum() error = %v", err)
	}

	installed, err := service.Install(testutil.Context(t), contract.InstallExtensionRequest{
		Path:     fixtureDir,
		Checksum: checksum,
	})
	if err != nil {
		t.Fatalf("service.Install() error = %v", err)
	}
	if installed.Name != "service-ext" || installed.State != "active" || !installed.DaemonRunning {
		t.Fatalf("installed extension = %#v, want active daemon-backed extension", installed)
	}

	status, err := service.Status(testutil.Context(t), "service-ext")
	if err != nil {
		t.Fatalf("service.Status() error = %v", err)
	}
	if status.Name != "service-ext" || status.State != "active" {
		t.Fatalf("status = %#v, want active extension", status)
	}

	disabled, err := service.Disable(testutil.Context(t), "service-ext")
	if err != nil {
		t.Fatalf("service.Disable() error = %v", err)
	}
	if disabled.State != "disabled" || disabled.Enabled {
		t.Fatalf("disabled extension = %#v, want disabled extension", disabled)
	}

	listed, err := service.List(testutil.Context(t))
	if err != nil {
		t.Fatalf("service.List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].State != "disabled" {
		t.Fatalf("listed extensions = %#v, want one disabled extension", listed)
	}
	if rebuilds != 2 {
		t.Fatalf("hook rebuild count = %d, want 2", rebuilds)
	}
}

func TestExtensionDeclarationProviderReturnsRuntimeDeclarations(t *testing.T) {
	t.Parallel()

	want := []hookspkg.HookDecl{
		{
			Name:         "ext-turn-start",
			Event:        hookspkg.HookTurnStart,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorSubprocess,
			Command:      "/bin/sh",
			Args:         []string{"-c", "printf '{}'"},
		},
	}
	runtime := &fakeExtensionRuntime{hookDecls: want}

	got, err := extensionDeclarationProvider(func() extensionRuntime { return runtime })(testutil.Context(t))
	if err != nil {
		t.Fatalf("extensionDeclarationProvider() error = %v", err)
	}
	if !testutil.EqualStringSlices([]string{got[0].Name}, []string{want[0].Name}) {
		t.Fatalf("extensionDeclarationProvider() = %#v, want %#v", got, want)
	}
}

func TestChainDeclarationProvidersWrapsProviderErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("provider boom")
	provider := chainDeclarationProviders(
		func(context.Context) ([]hookspkg.HookDecl, error) {
			return nil, wantErr
		},
	)

	_, err := provider(testutil.Context(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("provider error = %v, want wrapped %v", err, wantErr)
	}
	if err == nil || !strings.Contains(err.Error(), "provider 1") {
		t.Fatalf("provider error = %v, want provider context", err)
	}
}

func TestExtensionDeclarationProviderWrapsRuntimeErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("runtime boom")
	runtime := &fakeExtensionRuntime{hookErr: wantErr}

	_, err := extensionDeclarationProvider(func() extensionRuntime { return runtime })(testutil.Context(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("extensionDeclarationProvider() error = %v, want wrapped %v", err, wantErr)
	}
	if err == nil || !strings.Contains(err.Error(), "extension runtime") {
		t.Fatalf("extensionDeclarationProvider() error = %v, want runtime context", err)
	}
}

func TestBootStateExtensionRuntimeAccessIsSynchronized(t *testing.T) {
	t.Parallel()

	state := &bootState{}
	runtime := &fakeExtensionRuntime{
		hookDecls: []hookspkg.HookDecl{{
			Name:         "ext-turn-start",
			Event:        hookspkg.HookTurnStart,
			Mode:         hookspkg.HookModeSync,
			ExecutorKind: hookspkg.HookExecutorSubprocess,
			Command:      "/bin/sh",
			Args:         []string{"-c", "printf '{}'"},
		}},
	}
	provider := extensionDeclarationProvider(state.currentExtensionRuntime)

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(2)

		go func(iteration int) {
			defer wg.Done()
			<-start
			for j := 0; j < 128; j++ {
				if (iteration+j)%2 == 0 {
					state.setExtensionRuntime(runtime)
				} else {
					state.setExtensionRuntime(nil)
				}
			}
		}(i)

		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 128; j++ {
				_, _ = provider(context.Background())
			}
		}()
	}

	close(start)
	wg.Wait()
}

func TestShutdownDrainsHooksBeforeClosingDatabase(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	d := newTestDaemon(t, homePaths, cfg)

	asyncStarted := make(chan struct{}, 1)
	asyncRelease := make(chan struct{})
	dbClosed := make(chan struct{}, 1)

	hooks := hookspkg.NewHooks(
		hookspkg.WithLogger(discardLogger()),
		hookspkg.WithNativeDeclarations([]hookspkg.HookDecl{
			{
				Name:         "async-stop",
				Event:        hookspkg.HookSessionPostStop,
				Mode:         hookspkg.HookModeAsync,
				ExecutorKind: hookspkg.HookExecutorNative,
			},
		}),
		hookspkg.WithExecutorResolver(testHookExecutorResolver(map[string]hookspkg.Executor{
			"async-stop": hookspkg.NewTypedNativeExecutor(func(_ context.Context, _ hookspkg.RegisteredHook, _ hookspkg.SessionLifecyclePayload) (hookspkg.SessionPostStopPatch, error) {
				asyncStarted <- struct{}{}
				<-asyncRelease
				return hookspkg.SessionPostStopPatch{}, nil
			}),
		})),
	)
	t.Cleanup(hooks.Close)
	if err := hooks.Rebuild(testutil.Context(t)); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	notifier := newHooksNotifier(discardLogger(), func() time.Time { return time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC) })
	notifier.setRuntime(hooks, nil)

	d.sessions = &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}},
		onStop: func(string) {
			if _, err := notifier.DispatchSessionPostStop(context.Background(), hookspkg.SessionPostStopPayload(hookSessionLifecyclePayload(&session.Session{
				ID:          "sess-a",
				AgentName:   "codex",
				WorkspaceID: "ws-1",
				Workspace:   "/tmp/ws-1",
				Type:        session.SessionTypeUser,
				State:       session.StateStopped,
				CreatedAt:   time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
			}, hookspkg.HookSessionPostStop, time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)))); err != nil {
				t.Fatalf("DispatchSessionPostStop() error = %v", err)
			}
		},
	}
	d.hooks = hooks
	d.registry = &recordingRegistry{
		path: homePaths.DatabaseFile,
		onClose: func() {
			dbClosed <- struct{}{}
		},
	}
	d.closeLogger = func() error { return nil }

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Shutdown(testutil.Context(t))
	}()

	select {
	case <-asyncStarted:
	case <-time.After(time.Second):
		t.Fatal("async hook did not start before shutdown blocked")
	}

	select {
	case <-dbClosed:
		t.Fatal("database closed before hooks drained")
	default:
	}

	close(asyncRelease)
	if err := <-errCh; err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case <-dbClosed:
	case <-time.After(time.Second):
		t.Fatal("database was not closed after hooks drained")
	}
}

func TestBootFailureCleansUpStartedResourcesInReverseOrder(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	d := newTestDaemon(t, homePaths, cfg)

	var events []string
	d.closeLogger = func() error {
		events = append(events, "logger")
		return nil
	}
	d.acquireLock = func(path string, pid int) (*Lock, error) {
		return &Lock{
			path: path,
			releaseFn: func() error {
				events = append(events, "lock")
				return nil
			},
		}, nil
	}
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{
			path: homePaths.DatabaseFile,
			onClose: func() {
				events = append(events, "db")
			},
		}, nil
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{
			name: "http",
			onShutdown: func() {
				events = append(events, "http")
			},
		}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return nil, errors.New("uds boom")
	}

	if err := d.boot(testutil.Context(t)); err == nil || !strings.Contains(err.Error(), "uds boom") {
		t.Fatalf("boot() error = %v, want uds boom", err)
	}

	want := []string{"http", "db", "lock", "logger"}
	if !testutil.EqualStringSlices(events, want) {
		t.Fatalf("boot() cleanup order = %#v, want %#v", events, want)
	}
}

func TestBootFailureWhenWritingDaemonInfoCleansUpAllServers(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	infoDir := filepath.Join(homePaths.HomeDir, "daemon-info-dir")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(infoDir) error = %v", err)
	}

	d := newTestDaemon(t, homePaths, cfg)
	d.homePaths.DaemonInfo = infoDir

	var events []string
	d.closeLogger = func() error {
		events = append(events, "logger")
		return nil
	}
	d.acquireLock = func(path string, pid int) (*Lock, error) {
		return &Lock{
			path: path,
			releaseFn: func() error {
				events = append(events, "lock")
				return nil
			},
		}, nil
	}
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{
			path: homePaths.DatabaseFile,
			onClose: func() {
				events = append(events, "db")
			},
		}, nil
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http", onShutdown: func() { events = append(events, "http") }}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds", onShutdown: func() { events = append(events, "uds") }}, nil
	}

	if err := d.boot(testutil.Context(t)); err == nil || !strings.Contains(err.Error(), "daemon info") {
		t.Fatalf("boot() error = %v, want daemon info failure", err)
	}

	want := []string{"uds", "http", "db", "lock", "logger"}
	if !testutil.EqualStringSlices(events, want) {
		t.Fatalf("boot() cleanup order = %#v, want %#v", events, want)
	}
}

func TestVerifyImportBoundariesReportsViolations(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}

	sourceDir := filepath.Join(root, "internal", "worker")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(sourceDir) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sourceDir, "worker.go"),
		[]byte("package worker\n\nimport _ \"github.com/pedronauck/agh/internal/daemon\"\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(worker.go) error = %v", err)
	}

	violations, err := verifyImportBoundaries(root)
	if err != nil {
		t.Fatalf("verifyImportBoundaries() error = %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("verifyImportBoundaries() violations = %d, want 1", len(violations))
	}
}

func TestVerifyImportBoundariesAllowsDaemonSubpackages(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}

	sourceDir := filepath.Join(root, "internal", "daemon", "subsystem")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(sourceDir) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sourceDir, "subsystem.go"),
		[]byte("package subsystem\n\nimport _ \"github.com/pedronauck/agh/internal/cli\"\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(subsystem.go) error = %v", err)
	}

	violations, err := verifyImportBoundaries(root)
	if err != nil {
		t.Fatalf("verifyImportBoundaries() error = %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("verifyImportBoundaries() violations = %d, want 0", len(violations))
	}
}

func TestVerifyImportBoundariesDoesNotExemptHTTPPackages(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}

	sourceDir := filepath.Join(root, "internal", "api", "httpapi")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(sourceDir) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(sourceDir, "handler.go"),
		[]byte("package httpapi\n\nimport _ \"github.com/pedronauck/agh/internal/cli\"\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(handler.go) error = %v", err)
	}

	violations, err := verifyImportBoundaries(root)
	if err != nil {
		t.Fatalf("verifyImportBoundaries() error = %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("verifyImportBoundaries() violations = %d, want 1", len(violations))
	}
}

func TestStopSessionsIgnoresNotFoundAndHandlesNilManager(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.stopSessions(testutil.Context(t), nil); err != nil {
		t.Fatalf("stopSessions(nil) error = %v", err)
	}

	manager := &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}},
		stopErr: func(id string) error {
			return fmt.Errorf("%w: %s", session.ErrSessionNotFound, id)
		},
	}
	if err := d.stopSessions(testutil.Context(t), manager); err != nil {
		t.Fatalf("stopSessions(not found) error = %v", err)
	}
}

func TestStopSessionsUsesShutdownCauseWhenSupported(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	manager := &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}},
	}
	if err := d.stopSessions(testutil.Context(t), manager); err != nil {
		t.Fatalf("stopSessions() error = %v", err)
	}

	if got := len(manager.stopWithCauseCalls); got != 1 {
		t.Fatalf("StopWithCause() calls = %d, want 1", got)
	}
	call := manager.stopWithCauseCalls[0]
	if call.id != "sess-a" {
		t.Fatalf("StopWithCause() id = %q, want %q", call.id, "sess-a")
	}
	if call.cause != session.CauseShutdown {
		t.Fatalf("StopWithCause() cause = %v, want %v", call.cause, session.CauseShutdown)
	}
	if call.detail != "daemon shutdown" {
		t.Fatalf("StopWithCause() detail = %q, want %q", call.detail, "daemon shutdown")
	}
	if got := len(manager.stopCalls); got != 0 {
		t.Fatalf("Stop() calls = %d, want 0 when StopWithCause is available", got)
	}
}

func TestCleanupOrphansHandlesListAndSignalErrors(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	d.listProcesses = func(context.Context) ([]processInfo, error) {
		return nil, errors.New("ps failed")
	}
	if err := d.cleanupOrphans(testutil.Context(t), 1); err == nil || !strings.Contains(err.Error(), "ps failed") {
		t.Fatalf("cleanupOrphans(list failure) error = %v, want ps failed", err)
	}

	d.listProcesses = func(context.Context) ([]processInfo, error) {
		return []processInfo{{PID: 10, PPID: 5}}, nil
	}
	d.signalProcess = func(int, syscall.Signal) error {
		return errors.New("signal failed")
	}
	if err := d.cleanupOrphans(testutil.Context(t), 5); err == nil || !strings.Contains(err.Error(), "signal failed") {
		t.Fatalf("cleanupOrphans(signal failure) error = %v, want signal failed", err)
	}
	if err := d.cleanupOrphans(testutil.Context(t), 0); err != nil {
		t.Fatalf("cleanupOrphans(no stale pid) error = %v", err)
	}
}

func TestOptionsConfigureDaemon(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	signalCh := make(chan os.Signal, 1)
	httpFactory := func(context.Context, RuntimeDeps) (Server, error) { return &fakeServer{name: "http"}, nil }
	udsFactory := func(context.Context, RuntimeDeps) (Server, error) { return &fakeServer{name: "uds"}, nil }
	now := time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfigLoader(func() (aghconfig.Config, error) { return cfg, nil }),
		WithLogger(discardLogger()),
		WithNow(func() time.Time { return now }),
		WithHTTPServerFactory(httpFactory),
		WithUDSServerFactory(udsFactory),
		WithSignalBridge(signalCh),
		WithBoundaryVerification(true),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got, err := d.loadConfig(); err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	} else if got.HTTP.Port != cfg.HTTP.Port {
		t.Fatalf("loadConfig() port = %d, want %d", got.HTTP.Port, cfg.HTTP.Port)
	}
	if got := d.now(); !got.Equal(now) {
		t.Fatalf("now() = %v, want %v", got, now)
	}
	if d.signalCh != signalCh {
		t.Fatal("WithSignalBridge() did not apply")
	}
	if !d.verifyBoundaries {
		t.Fatal("WithBoundaryVerification(true) did not apply")
	}
}

func TestRunShutsDownOnInjectedSignal(t *testing.T) {
	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	signalCh := make(chan os.Signal, 1)

	d := newTestDaemon(t, homePaths, cfg)
	d.signalCh = signalCh
	d.acquireLock = func(path string, pid int) (*Lock, error) {
		return &Lock{path: path}, nil
	}
	d.openRegistry = func(context.Context, string) (Registry, error) {
		return &recordingRegistry{path: homePaths.DatabaseFile}, nil
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(context.Background())
	}()

	<-d.readyCh
	signalCh <- syscall.SIGTERM

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestBoundariesUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(internal) error = %v", err)
	}

	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.boundaryRoot = root

	if err := d.Boundaries(testutil.Context(t)); err != nil {
		t.Fatalf("Boundaries() error = %v", err)
	}
}

func TestBoundariesReturnsViolations(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	violatingDir := filepath.Join(root, "internal", "worker")
	if err := os.MkdirAll(violatingDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(violatingDir) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(violatingDir, "worker.go"),
		[]byte("package worker\n\nimport _ \"github.com/pedronauck/agh/internal/cli\"\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(worker.go) error = %v", err)
	}

	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.boundaryRoot = root

	if err := d.Boundaries(testutil.Context(t)); err == nil {
		t.Fatal("Boundaries() error = nil, want violation")
	}
}

func TestBoundariesUsesWorkingDirectoryWhenRootUnset(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/pedronauck/agh\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(go.mod) error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(internal) error = %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(root) error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.Boundaries(testutil.Context(t)); err != nil {
		t.Fatalf("Boundaries() error = %v", err)
	}
}

func TestLoadConfigFromHomeAppliesOverlayAndNormalizesSocket(t *testing.T) {
	homePaths := testHomePaths(t)
	if err := os.WriteFile(
		homePaths.ConfigFile,
		[]byte("[daemon]\nsocket = \"~/agh-test.sock\"\n[http]\nport = 4242\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(config) error = %v", err)
	}

	cfg, err := loadConfigFromHome(homePaths)
	if err != nil {
		t.Fatalf("loadConfigFromHome() error = %v", err)
	}
	if got, want := cfg.HTTP.Port, 4242; got != want {
		t.Fatalf("cfg.HTTP.Port = %d, want %d", got, want)
	}
	if !strings.Contains(cfg.Daemon.Socket, "agh-test.sock") || !filepath.IsAbs(cfg.Daemon.Socket) {
		t.Fatalf("cfg.Daemon.Socket = %q, want expanded absolute path", cfg.Daemon.Socket)
	}
}

func TestLoadConfigFromHomeValidationError(t *testing.T) {
	homePaths := testHomePaths(t)
	if err := os.WriteFile(
		homePaths.ConfigFile,
		[]byte("[http]\nport = 70000\n"),
		0o644,
	); err != nil {
		t.Fatalf("os.WriteFile(config) error = %v", err)
	}

	if _, err := loadConfigFromHome(homePaths); err == nil {
		t.Fatal("loadConfigFromHome(invalid config) error = nil, want non-nil")
	}
}

func TestShouldVerifyBoundariesFromEnv(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.getenv = func(string) string { return "yes" }
	if !d.shouldVerifyBoundaries() {
		t.Fatal("shouldVerifyBoundaries() = false, want true")
	}
	d.verifyBoundaries = false
	d.getenv = func(string) string { return "" }
	if d.shouldVerifyBoundaries() {
		t.Fatal("shouldVerifyBoundaries() = true, want false")
	}
	d.getenv = nil
	if d.shouldVerifyBoundaries() {
		t.Fatal("shouldVerifyBoundaries() with nil getenv = true, want false")
	}
	d.verifyBoundaries = true
	if !d.shouldVerifyBoundaries() {
		t.Fatal("shouldVerifyBoundaries() with explicit option = false, want true")
	}
}

func TestSignalSourceDefaultsToOSSignalRegistration(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ch, stop := d.signalSource()
	if ch == nil {
		t.Fatal("signalSource() bridge = nil")
	}
	stop()
}

func TestBootInjectsComposedAssemblerForFeatureFlagCombinations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		memoryEnabled bool
		skillsEnabled bool
		wantMemory    bool
		wantSkills    bool
	}{
		{
			name:          "memory on and skills on",
			memoryEnabled: true,
			skillsEnabled: true,
			wantMemory:    true,
			wantSkills:    true,
		},
		{
			name:          "memory on and skills off",
			memoryEnabled: true,
			skillsEnabled: false,
			wantMemory:    true,
			wantSkills:    false,
		},
		{
			name:          "memory off and skills on",
			memoryEnabled: false,
			skillsEnabled: true,
			wantMemory:    false,
			wantSkills:    true,
		},
		{
			name:          "memory off and skills off",
			memoryEnabled: false,
			skillsEnabled: false,
			wantMemory:    false,
			wantSkills:    false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			homePaths := testHomePaths(t)
			cfg := testConfig(t, homePaths)
			cfg.Memory.Enabled = tc.memoryEnabled
			cfg.Skills.Enabled = tc.skillsEnabled
			cfg.Memory.GlobalDir = filepath.Join(homePaths.HomeDir, "custom-memory")

			d := newTestDaemon(t, homePaths, cfg)

			var capturedDeps SessionManagerDeps
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
				t.Fatal("boot() did not inject the composed prompt assembler")
			}
			if capturedDeps.WorkspaceResolver == nil {
				t.Fatal("boot() did not inject the workspace resolver")
			}
			if d.workspaceResolver == nil {
				t.Fatal("boot() did not retain the workspace resolver")
			}
			if got := d.memoryStore != nil; got != tc.wantMemory {
				t.Fatalf("memory store initialized = %t, want %t", got, tc.wantMemory)
			}
			if got := d.skillsRegistry != nil; got != tc.wantSkills {
				t.Fatalf("skills registry initialized = %t, want %t", got, tc.wantSkills)
			}

			workspace := filepath.Join(t.TempDir(), "workspace")
			writeDaemonMemoryIndex(t, cfg.Memory.GlobalDir, workspace)

			prompt, err := capturedDeps.PromptAssembler.Assemble(context.Background(), testPromptAgent("Base prompt."), workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{RootDir: workspace},
			})
			if err != nil {
				t.Fatalf("PromptAssembler.Assemble() error = %v", err)
			}

			assertPromptContainsInOrder(t, prompt, orderedFragments(tc.wantMemory, tc.wantSkills)...)
			assertPromptExcludes(t, prompt, excludedFragments(tc.wantMemory, tc.wantSkills)...)

			if tc.wantMemory {
				if info, err := os.Stat(cfg.Memory.GlobalDir); err != nil {
					t.Fatalf("stat memory.global_dir error = %v", err)
				} else if !info.IsDir() {
					t.Fatalf("memory.global_dir mode = %v, want directory", info.Mode())
				}
			}
			if tc.wantSkills {
				if skills := d.skillsRegistry.List(); len(skills) == 0 {
					t.Fatal("skills registry list = empty, want bundled skills")
				}
			}
		})
	}
}

func TestBootCreatesWorkspaceResolverAndInjectsSessionManager(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)

	var capturedDeps SessionManagerDeps
	var capturedUDSDeps RuntimeDeps
	d := newTestDaemon(t, homePaths, cfg)
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
	d.udsFactory = func(_ context.Context, deps RuntimeDeps) (Server, error) {
		capturedUDSDeps = deps
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

	if d.workspaceResolver == nil {
		t.Fatal("boot() did not create the daemon workspace resolver")
	}
	if capturedDeps.WorkspaceResolver == nil {
		t.Fatal("boot() did not inject the session manager workspace resolver")
	}
	if capturedUDSDeps.WorkspaceService == nil {
		t.Fatal("boot() did not inject the uds workspace service")
	}
	if capturedUDSDeps.WorkspaceService != d.workspaceResolver {
		t.Fatal("boot() injected a different workspace service into uds")
	}

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	resolved := resolveDaemonWorkspace(t, capturedDeps.WorkspaceResolver, workspaceRoot)
	if got, want := resolved.RootDir, canonicalDaemonRoot(t, workspaceRoot); got != want {
		t.Fatalf("resolved workspace root = %q, want %q", got, want)
	}
}

func TestBootSkillsWatcherRefreshesOnGlobalChangesAndStopsOnShutdown(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true
	cfg.Skills.PollInterval = 10 * time.Millisecond

	d := newTestDaemon(t, homePaths, cfg)
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

	registry := d.skillsRegistry
	if registry == nil {
		t.Fatal("boot() did not initialize the skills registry")
	}

	writeDaemonSkill(t, filepath.Join(homePaths.HomeDir, ".agents", "skills"), "watched-skill", "Global watched skill")
	waitForCondition(t, "watcher refresh after boot", func() bool {
		_, ok := registry.Get("watched-skill")
		return ok
	})
	versionAfterRefresh := registry.GlobalVersion()

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	writeDaemonSkill(t, filepath.Join(homePaths.HomeDir, ".agents", "skills"), "after-shutdown", "Should not be observed")
	time.Sleep(4 * cfg.Skills.PollInterval)

	if got := registry.GlobalVersion(); got != versionAfterRefresh {
		t.Fatalf("registry version after shutdown = %d, want %d", got, versionAfterRefresh)
	}
	if _, ok := registry.Get("after-shutdown"); ok {
		t.Fatal("skills watcher continued refreshing after shutdown")
	}
}

func TestShutdownStopsSkillsWatcherBeforeSessions(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true
	cfg.Skills.PollInterval = 10 * time.Millisecond

	var skillsDone <-chan struct{}
	sessions := &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}},
		onStop: func(string) {
			select {
			case <-skillsDone:
			default:
				t.Error("skills watcher was still running when session shutdown started")
			}
		},
	}

	d := newTestDaemon(t, homePaths, cfg)
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
	skillsDone = d.skillsDone
	if skillsDone == nil {
		t.Fatal("boot() did not start the skills watcher")
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestSkillsRegistryConfigUsesDaemonHomeAndDisabledSkills(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Skills.DisabledSkills = []string{"alpha", "beta"}

	d := newTestDaemon(t, homePaths, cfg)

	registryCfg, err := d.skillsRegistryConfig(cfg)
	if err != nil {
		t.Fatalf("skillsRegistryConfig() error = %v", err)
	}

	if registryCfg.BundledFS == nil {
		t.Fatal("skillsRegistryConfig() BundledFS = nil")
	}
	if got, want := registryCfg.UserSkillsDir, homePaths.SkillsDir; got != want {
		t.Fatalf("skillsRegistryConfig() UserSkillsDir = %q, want %q", got, want)
	}
	if got, want := registryCfg.UserAgentsDir, filepath.Join(homePaths.HomeDir, ".agents", "skills"); got != want {
		t.Fatalf("skillsRegistryConfig() UserAgentsDir = %q, want %q", got, want)
	}
	if got := registryCfg.DisabledSkills; len(got) != 2 || got[0] != "alpha" || got[1] != "beta" {
		t.Fatalf("skillsRegistryConfig() DisabledSkills = %#v, want [alpha beta]", got)
	}
}

func TestRunSkipsDreamLoopWhenMemoryOrDreamDisabled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		patch func(*aghconfig.Config)
	}{
		{
			name: "memory disabled",
			patch: func(cfg *aghconfig.Config) {
				cfg.Memory.Enabled = false
			},
		},
		{
			name: "dream disabled",
			patch: func(cfg *aghconfig.Config) {
				cfg.Memory.Dream.Enabled = false
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			homePaths := testHomePaths(t)
			cfg := testConfig(t, homePaths)
			tc.patch(&cfg)

			d := newTestDaemon(t, homePaths, cfg)
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

			runCtx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			go func() {
				errCh <- d.Run(runCtx)
			}()

			<-d.readyCh
			waitForCondition(t, "dream loop skipped", func() bool {
				d.mu.Lock()
				defer d.mu.Unlock()
				return d.dreamRuntime == nil
			})

			cancel()
			if err := <-errCh; err != nil {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestDreamTickerRunsAndStopsOnCancellation(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Dream.CheckInterval = 10 * time.Millisecond

	dream := &fakeDreamService{shouldRun: true}
	d := newTestDaemon(t, homePaths, cfg)
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return &fakeSessionManager{}, nil
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
	waitForCondition(t, "dream loop started", func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return d.dreamRuntime != nil
	})
	waitForCondition(t, "dream ticker run", func() bool {
		return dream.runCount() > 0
	})

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	runCount := dream.runCount()
	time.Sleep(30 * time.Millisecond)
	if got := dream.runCount(); got != runCount {
		t.Fatalf("dream run count after shutdown = %d, want %d", got, runCount)
	}
}

func TestSessionStopNotifierQueuesDreamCheck(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Dream.CheckInterval = time.Hour

	workspace := filepath.Join(t.TempDir(), "workspace")
	sessions := &fakeSessionManager{}
	dream := &fakeDreamService{
		shouldRun: true,
		runHook: func(ctx context.Context, spawn memory.SessionSpawner, workspace string) error {
			return spawn(ctx, "memory-consolidation", "session-stop prompt", workspace)
		},
	}
	var dispatcher session.HookSet

	d := newTestDaemon(t, homePaths, cfg)
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		dispatcher = deps.Hooks
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
	waitForCondition(t, "dream loop started", func() bool {
		d.mu.Lock()
		defer d.mu.Unlock()
		return d.dreamRuntime != nil
	})
	if dispatcher.Session == nil {
		t.Fatal("session manager hook set = nil")
	}

	resolved := resolveDaemonWorkspace(t, d.workspaceResolver, workspace)
	if _, err := dispatcher.Session.DispatchSessionPostStop(context.Background(), hookspkg.SessionPostStopPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPostStop,
			Timestamp: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:   "sess-user",
			WorkspaceID: resolved.ID,
			SessionType: string(session.SessionTypeUser),
			State:       string(session.StateStopped),
		},
	}); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
	}
	waitForCondition(t, "dream run from session stop", func() bool {
		return dream.runCount() == 1
	})
	waitForCondition(t, "dream session workspace propagated", func() bool {
		return sessions.createCount() == 1
	})
	if got := sessions.createCall(0).Workspace; got != resolved.ID {
		t.Fatalf("Create() workspace = %q, want %q", got, resolved.ID)
	}
	if got := sessions.createCall(0).WorkspacePath; got != "" {
		t.Fatalf("Create() workspace_path = %q, want empty", got)
	}

	if _, err := dispatcher.Session.DispatchSessionPostStop(context.Background(), hookspkg.SessionPostStopPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookSessionPostStop,
			Timestamp: time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		},
		SessionContext: hookspkg.SessionContext{
			SessionID:   "sess-dream",
			SessionType: string(session.SessionTypeDream),
			State:       string(session.StateStopped),
		},
	}); err != nil {
		t.Fatalf("DispatchSessionPostStop(dream) error = %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if got := dream.runCount(); got != 1 {
		t.Fatalf("dream run count after dream-session stop = %d, want 1", got)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRemoveStaleSocketBehaviors(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "daemon.sock")
	if err := removeStaleSocket(socketPath); err != nil {
		t.Fatalf("removeStaleSocket(missing) error = %v", err)
	}

	if err := os.WriteFile(socketPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(socket) error = %v", err)
	}
	if err := removeStaleSocket(socketPath); err != nil {
		t.Fatalf("removeStaleSocket(file) error = %v", err)
	}

	dirPath := filepath.Join(t.TempDir(), "dir")
	if err := os.MkdirAll(filepath.Join(dirPath, "child"), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(dirPath) error = %v", err)
	}
	if err := removeStaleSocket(dirPath); err == nil {
		t.Fatal("removeStaleSocket(dir) error = nil, want non-nil")
	}
}

func TestResolveDaemonPortUsesReporterWhenAvailable(t *testing.T) {
	if got, want := resolveDaemonPort(2123, portReportingServer{port: 9090}), 9090; got != want {
		t.Fatalf("resolveDaemonPort() = %d, want %d", got, want)
	}
	if got, want := resolveDaemonPort(2123, &fakeServer{name: "default"}), 2123; got != want {
		t.Fatalf("resolveDaemonPort(default) = %d, want %d", got, want)
	}
}

func TestListProcessesAndSignalProcess(t *testing.T) {
	processes, err := listProcesses(testutil.Context(t))
	if err != nil {
		t.Fatalf("listProcesses() error = %v", err)
	}
	if len(processes) == 0 {
		t.Fatal("listProcesses() returned no processes")
	}

	if err := procutil.Signal(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Fatalf("procutil.Signal(self, 0) error = %v", err)
	}
	if err := procutil.Signal(0, syscall.SIGTERM); err == nil {
		t.Fatal("procutil.Signal(invalid pid) error = nil, want non-nil")
	}
}

func TestProcessAliveAndRuntimeLoggerHelpers(t *testing.T) {
	if procutil.Alive(0) {
		t.Fatal("procutil.Alive(0) = true, want false")
	}
	if !procutil.Alive(os.Getpid()) {
		t.Fatal("procutil.Alive(self) = false, want true")
	}
	if procutil.Alive(999999) && runtime.GOOS != "windows" {
		t.Fatal("procutil.Alive(999999) = true, want false")
	}

	d, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got := d.runtimeLogger(); got == nil {
		t.Fatal("runtimeLogger() = nil")
	}
}

func TestInfoValidationAndReadFailures(t *testing.T) {
	if err := (Info{}).Validate(); err == nil {
		t.Fatal("Info.Validate() error = nil, want non-nil")
	}
	if err := (Info{PID: 1, Port: -1, StartedAt: time.Now().UTC()}).Validate(); err == nil {
		t.Fatal("Info.Validate(invalid port) error = nil, want non-nil")
	}
	if err := (Info{PID: 1, Port: 1, StartedAt: time.Now().UTC()}).Validate(); err != nil {
		t.Fatalf("Info.Validate(valid) error = %v", err)
	}
	if err := WriteInfo("", Info{}); err == nil {
		t.Fatal("WriteInfo(blank path) error = nil, want non-nil")
	}
	if err := RemoveInfo(""); err != nil {
		t.Fatalf("RemoveInfo(blank path) error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{bad"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(bad.json) error = %v", err)
	}
	if _, err := ReadInfo(path); err == nil {
		t.Fatal("ReadInfo(invalid JSON) error = nil, want non-nil")
	}
	if _, err := ReadInfo(""); err == nil {
		t.Fatal("ReadInfo(blank path) error = nil, want non-nil")
	}

	validPath := filepath.Join(t.TempDir(), "nested", "daemon.json")
	validInfo := Info{PID: 10, Port: 20, StartedAt: time.Now().UTC()}
	if err := WriteInfo(validPath, validInfo); err != nil {
		t.Fatalf("WriteInfo(valid path) error = %v", err)
	}
	if err := syncDir(filepath.Dir(validPath)); err != nil {
		t.Fatalf("syncDir(valid dir) error = %v", err)
	}
	if err := RemoveInfo(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatalf("RemoveInfo(missing file) error = %v", err)
	}
}

func TestDaemonNetworkInfoHelpersValidateAndRedactRuntimeStatus(t *testing.T) {
	ctx := testutil.Context(t)

	if err := (NetworkInfo{}).Validate(); err == nil {
		t.Fatal("NetworkInfo.Validate() error = nil, want non-nil")
	}
	if err := (NetworkInfo{Status: network.StatusRunning, ListenerPort: 65536}).Validate(); err == nil {
		t.Fatal("NetworkInfo.Validate(invalid port) error = nil, want non-nil")
	}
	if err := (NetworkInfo{Status: network.StatusRunning, ListenerPort: 4222}).Validate(); err != nil {
		t.Fatalf("NetworkInfo.Validate(valid) error = %v", err)
	}

	disabledInfo, err := daemonNetworkInfo(ctx, aghconfig.NetworkConfig{}, nil)
	if err != nil {
		t.Fatalf("daemonNetworkInfo(disabled) error = %v", err)
	}
	if disabledInfo == nil || disabledInfo.Enabled || disabledInfo.Status != network.StatusDisabled {
		t.Fatalf("daemonNetworkInfo(disabled) = %#v, want disabled snapshot", disabledInfo)
	}

	if _, err := daemonNetworkInfo(ctx, aghconfig.NetworkConfig{Enabled: true}, nil); err == nil {
		t.Fatal("daemonNetworkInfo(enabled nil service) error = nil, want non-nil")
	}
	if _, err := daemonNetworkInfo(ctx, aghconfig.NetworkConfig{Enabled: true}, &fakeNetworkRuntime{}); err == nil {
		t.Fatal("daemonNetworkInfo(nil status) error = nil, want non-nil")
	}

	info, err := daemonNetworkInfo(ctx, aghconfig.NetworkConfig{Enabled: true}, &fakeNetworkRuntime{
		status: &network.NetworkStatus{
			Enabled:      true,
			Status:       " running ",
			ListenerHost: " 127.0.0.1 ",
			ListenerPort: 4222,
		},
	})
	if err != nil {
		t.Fatalf("daemonNetworkInfo(runtime status) error = %v", err)
	}
	if info == nil {
		t.Fatal("daemonNetworkInfo(runtime status) = nil, want populated diagnostics")
	}
	if !info.Enabled || info.Status != network.StatusRunning || info.ListenerHost != "127.0.0.1" || info.ListenerPort != 4222 {
		t.Fatalf("daemonNetworkInfo(runtime status) = %#v, want trimmed listener diagnostics", info)
	}
}

func TestLockHelpersAndErrors(t *testing.T) {
	lock := &Lock{path: "/tmp/daemon.lock"}
	if got := lock.Path(); got != "/tmp/daemon.lock" {
		t.Fatalf("lock.Path() = %q, want %q", got, "/tmp/daemon.lock")
	}

	err := errAlreadyRunning{pid: 42}
	if !strings.Contains(err.Error(), "42") {
		t.Fatalf("errAlreadyRunning.Error() = %q, want pid in message", err.Error())
	}
	if got := (errAlreadyRunning{}).Error(); got != ErrAlreadyRunning.Error() {
		t.Fatalf("errAlreadyRunning{}.Error() = %q, want %q", got, ErrAlreadyRunning.Error())
	}
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("errors.Is(errAlreadyRunning, ErrAlreadyRunning) = false, want true")
	}

	if _, err := AcquireLock("", 1); err == nil {
		t.Fatal("AcquireLock(blank path) error = nil, want non-nil")
	}
	if _, err := AcquireLock(filepath.Join(t.TempDir(), "daemon.lock"), 0); err == nil {
		t.Fatal("AcquireLock(invalid pid) error = nil, want non-nil")
	}

	released := false
	if err := (&Lock{releaseFn: func() error { released = true; return nil }}).Release(); err != nil {
		t.Fatalf("Lock.Release(releaseFn) error = %v", err)
	}
	if !released {
		t.Fatal("Lock.Release() did not use injected release function")
	}
	if got := ((*Lock)(nil)).Path(); got != "" {
		t.Fatalf("nil lock Path() = %q, want empty", got)
	}
	if got := ((*Lock)(nil)).StalePID(); got != 0 {
		t.Fatalf("nil lock StalePID() = %d, want 0", got)
	}
	if err := ((*Lock)(nil)).Release(); err != nil {
		t.Fatalf("nil lock Release() error = %v, want nil", err)
	}

	path := filepath.Join(t.TempDir(), "pid.lock")
	if err := os.WriteFile(path, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(pid.lock) error = %v", err)
	}
	if got, err := readLockPID(path); err != nil {
		t.Fatalf("readLockPID(invalid contents) error = %v", err)
	} else if got != 0 {
		t.Fatalf("readLockPID(invalid contents) = %d, want 0", got)
	}
	if err := writeLockPID(path, 0); err != nil {
		t.Fatalf("writeLockPID(0) error = %v", err)
	}
	if data, err := os.ReadFile(path); err != nil {
		t.Fatalf("os.ReadFile(pid.lock) error = %v", err)
	} else if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("writeLockPID(0) contents = %q, want empty", string(data))
	}
}

func waitForCondition(t *testing.T, label string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", label)
}

func testHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	homePaths.DaemonSocket = shortSocketPath(t)
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func testConfig(t *testing.T, homePaths aghconfig.HomePaths) aghconfig.Config {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)
	cfg.Daemon.Socket = homePaths.DaemonSocket
	cfg.Automation.Enabled = false
	return cfg
}

func writeDaemonMemoryIndex(t *testing.T, globalDir string, workspace string) {
	t.Helper()

	writeDaemonFile(t, filepath.Join(globalDir, "MEMORY.md"), "- [Global](global.md) - global note")
	writeDaemonFile(t, filepath.Join(workspace, aghconfig.DirName, "memory", "MEMORY.md"), "- [Workspace](workspace.md) - workspace note")
}

func writeDaemonSkill(t *testing.T, root string, name string, description string) {
	t.Helper()

	content := fmt.Sprintf(`---
name: %s
description: %s
---

# %s
`, name, description, name)
	writeDaemonFile(t, filepath.Join(root, name, "SKILL.md"), content)
}

func writeDaemonFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func resolveDaemonWorkspace(t *testing.T, resolver workspacepkg.WorkspaceResolver, root string) workspacepkg.ResolvedWorkspace {
	t.Helper()

	if resolver == nil {
		t.Fatal("workspace resolver = nil")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}

	resolved, err := resolver.ResolveOrRegister(testutil.Context(t), root)
	if err != nil {
		t.Fatalf("ResolveOrRegister(%q) error = %v", root, err)
	}
	return resolved
}

func canonicalDaemonRoot(t *testing.T, root string) string {
	t.Helper()

	evaluated, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("filepath.EvalSymlinks(%q) error = %v", root, err)
	}
	canonical, err := filepath.Abs(evaluated)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", evaluated, err)
	}
	return canonical
}

func orderedFragments(wantMemory bool, wantSkills bool) []string {
	fragments := make([]string, 0, 3)
	if wantMemory {
		fragments = append(fragments, "# Persistent Memory")
	}
	fragments = append(fragments, "Base prompt.")
	if wantSkills {
		fragments = append(fragments, "<available-skills>", "agh-session-guide")
	}
	return fragments
}

func excludedFragments(wantMemory bool, wantSkills bool) []string {
	fragments := make([]string, 0, 2)
	if !wantMemory {
		fragments = append(fragments, "# Persistent Memory")
	}
	if !wantSkills {
		fragments = append(fragments, "<available-skills>")
	}
	return fragments
}

func assertPromptContainsInOrder(t *testing.T, prompt string, fragments ...string) {
	t.Helper()

	searchFrom := 0
	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}

		offset := strings.Index(prompt[searchFrom:], fragment)
		if offset < 0 {
			t.Fatalf("prompt %q does not contain %q", prompt, fragment)
		}
		searchFrom += offset + len(fragment)
	}
}

func assertPromptExcludes(t *testing.T, prompt string, fragments ...string) {
	t.Helper()

	for _, fragment := range fragments {
		if fragment == "" {
			continue
		}
		if strings.Contains(prompt, fragment) {
			t.Fatalf("prompt %q contains unexpected fragment %q", prompt, fragment)
		}
	}
}

func newTestDaemon(t *testing.T, homePaths aghconfig.HomePaths, cfg aghconfig.Config) *Daemon {
	t.Helper()

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.getenv = func(key string) string {
		if key == "HOME" {
			return homePaths.HomeDir
		}
		return os.Getenv(key)
	}
	return d
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strconvString(v int) string {
	return fmt.Sprintf("%d", v)
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(os.TempDir(), fmt.Sprintf("agh-%d.sock", time.Now().UTC().UnixNano()))
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	return path
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(:0) error = %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", ln.Addr())
	}
	return tcpAddr.Port
}

type fakeSessionManager struct {
	mu               sync.Mutex
	infos            []*session.SessionInfo
	onStop           func(string)
	stopErr          func(string) error
	stopWithCauseErr func(string, session.StopCause, string) error
	createCalls      []session.CreateOpts
	promptCalls      []struct {
		id  string
		msg string
	}
	promptStarted      chan struct{}
	promptRelease      <-chan struct{}
	promptCtxCancelled chan struct{}
	stopCalls          []string
	stopWithCauseCalls []fakeStopWithCauseCall
}

type fakeStopWithCauseCall struct {
	id     string
	cause  session.StopCause
	detail string
}

func (f *fakeSessionManager) Create(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls = append(f.createCalls, opts)
	sessionID := fmt.Sprintf("dream-%d", len(f.createCalls))
	workspaceID := strings.TrimSpace(opts.Workspace)
	workspace := strings.TrimSpace(opts.WorkspacePath)
	if workspace == "" {
		workspace = workspaceID
	}
	return &session.Session{
		ID:          sessionID,
		AgentName:   opts.AgentName,
		WorkspaceID: workspaceID,
		Workspace:   workspace,
		Type:        opts.Type,
		State:       session.StateActive,
	}, nil
}

func (f *fakeSessionManager) List() []*session.SessionInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*session.SessionInfo(nil), f.infos...)
}

func (f *fakeSessionManager) ListAll(context.Context) ([]*session.SessionInfo, error) {
	return f.List(), nil
}

func (f *fakeSessionManager) Status(_ context.Context, id string) (*session.SessionInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, info := range f.infos {
		if info != nil && info.ID == id {
			return info, nil
		}
	}
	return nil, session.ErrSessionNotFound
}

func (f *fakeSessionManager) Events(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (f *fakeSessionManager) History(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (f *fakeSessionManager) Transcript(context.Context, string) ([]transcript.Message, error) {
	return nil, nil
}

func (f *fakeSessionManager) Stop(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls = append(f.stopCalls, id)
	if f.onStop != nil && len(f.infos) > 0 {
		f.onStop(f.infos[0].ID)
		f.infos = f.infos[1:]
	}
	if f.stopErr != nil {
		return f.stopErr(id)
	}
	return nil
}

func (f *fakeSessionManager) StopWithCause(_ context.Context, id string, cause session.StopCause, detail string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopWithCauseCalls = append(f.stopWithCauseCalls, fakeStopWithCauseCall{
		id:     id,
		cause:  cause,
		detail: detail,
	})
	if f.onStop != nil && len(f.infos) > 0 {
		f.onStop(f.infos[0].ID)
		f.infos = f.infos[1:]
	}
	if f.stopWithCauseErr != nil {
		return f.stopWithCauseErr(id, cause, detail)
	}
	if f.stopErr != nil {
		return f.stopErr(id)
	}
	return nil
}

func (f *fakeSessionManager) Resume(context.Context, string) (*session.Session, error) {
	return nil, nil
}

func (f *fakeSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	f.mu.Lock()
	f.promptCalls = append(f.promptCalls, struct {
		id  string
		msg string
	}{id: id, msg: msg})
	promptStarted := f.promptStarted
	promptRelease := f.promptRelease
	promptCtxCancelled := f.promptCtxCancelled
	f.mu.Unlock()

	if promptStarted != nil {
		select {
		case promptStarted <- struct{}{}:
		default:
		}
	}

	if promptRelease != nil || promptCtxCancelled != nil {
		ch := make(chan acp.AgentEvent)
		go func() {
			defer close(ch)
			if promptRelease == nil {
				if ctx != nil {
					<-ctx.Done()
				}
			} else {
				select {
				case <-promptRelease:
					return
				case <-ctx.Done():
				}
			}
			if promptCtxCancelled != nil {
				select {
				case promptCtxCancelled <- struct{}{}:
				default:
				}
			}
		}()
		return ch, nil
	}

	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (f *fakeSessionManager) ApprovePermission(context.Context, string, acp.ApproveRequest) error {
	return nil
}

func (f *fakeSessionManager) createCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.createCalls)
}

func (f *fakeSessionManager) createCall(index int) session.CreateOpts {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls[index]
}

type fakeNetworkBindableSessionManager struct {
	*fakeSessionManager
	networkPeers    session.NetworkPeerLifecycle
	turnEndNotifier session.TurnEndNotifier
	promptNetworkFn func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	prompting       map[string]bool
}

func newFakeNetworkBindableSessionManager() *fakeNetworkBindableSessionManager {
	return &fakeNetworkBindableSessionManager{
		fakeSessionManager: &fakeSessionManager{},
		prompting:          make(map[string]bool),
	}
}

func (f *fakeNetworkBindableSessionManager) SetNetworkPeerLifecycle(lifecycle session.NetworkPeerLifecycle) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.networkPeers = lifecycle
}

func (f *fakeNetworkBindableSessionManager) currentNetworkPeerLifecycle() session.NetworkPeerLifecycle {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.networkPeers
}

func (f *fakeNetworkBindableSessionManager) SetTurnEndNotifier(fn session.TurnEndNotifier) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.turnEndNotifier = fn
}

func (f *fakeNetworkBindableSessionManager) currentTurnEndNotifier() session.TurnEndNotifier {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.turnEndNotifier
}

func (f *fakeNetworkBindableSessionManager) PromptNetwork(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	f.mu.Lock()
	fn := f.promptNetworkFn
	f.mu.Unlock()
	if fn != nil {
		return fn(ctx, id, msg)
	}

	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (f *fakeNetworkBindableSessionManager) IsPrompting(sessionID string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prompting[sessionID]
}

type fakeNetworkRuntime struct {
	mu          sync.Mutex
	status      *network.NetworkStatus
	statusErr   error
	sendID      string
	sendErr     error
	sendCalls   []network.SendRequest
	joinCalls   []fakeNetworkJoinCall
	leaveCalls  []string
	turnEnds    []string
	inboxes     map[string][]network.Envelope
	shutdownErr error
	onShutdown  func()
}

type fakeNetworkJoinCall struct {
	sessionID string
	peerID    string
	channel   string
}

func (f *fakeNetworkRuntime) Send(_ context.Context, req network.SendRequest) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sendCalls = append(f.sendCalls, req)
	if f.sendErr != nil {
		return "", f.sendErr
	}
	if strings.TrimSpace(f.sendID) != "" {
		return f.sendID, nil
	}
	return "msg-test", nil
}

func (f *fakeNetworkRuntime) ListPeers(context.Context, string) ([]network.PeerInfo, error) {
	return nil, nil
}

func (f *fakeNetworkRuntime) ListChannels(context.Context) ([]network.ChannelInfo, error) {
	return nil, nil
}

func (f *fakeNetworkRuntime) Status(context.Context) (*network.NetworkStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	if f.status == nil {
		return nil, nil
	}
	status := *f.status
	return &status, nil
}

func (f *fakeNetworkRuntime) Inbox(_ context.Context, sessionID string) ([]network.Envelope, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.inboxes) == 0 {
		return nil, nil
	}
	return append([]network.Envelope(nil), f.inboxes[sessionID]...), nil
}

func (f *fakeNetworkRuntime) JoinChannel(_ context.Context, sessionID string, peerID string, channel string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.joinCalls = append(f.joinCalls, fakeNetworkJoinCall{
		sessionID: sessionID,
		peerID:    peerID,
		channel:   channel,
	})
	return nil
}

func (f *fakeNetworkRuntime) LeaveChannel(_ context.Context, sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.leaveCalls = append(f.leaveCalls, sessionID)
	return nil
}

func (f *fakeNetworkRuntime) OnTurnEnd(sessionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.turnEnds = append(f.turnEnds, sessionID)
}

func (f *fakeNetworkRuntime) Shutdown(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.onShutdown != nil {
		f.onShutdown()
	}
	return f.shutdownErr
}

type fakeObserver struct {
	reconciled bool
	result     store.ReconcileResult
	err        error
}

func (f *fakeObserver) QueryEvents(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
	return nil, nil
}

func (f *fakeObserver) QueryHookCatalog(context.Context, hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error) {
	return nil, nil
}

func (f *fakeObserver) QueryHookRuns(context.Context, store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
	return nil, nil
}

func (f *fakeObserver) QueryHookEvents(context.Context, hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error) {
	return nil, nil
}

func (f *fakeObserver) QueryBridgeHealth(context.Context) ([]observe.BridgeInstanceHealth, error) {
	return nil, nil
}

func (f *fakeObserver) Health(context.Context) (observe.Health, error) {
	return observe.Health{Status: "ok"}, nil
}

func (f *fakeObserver) Reconcile(context.Context) (store.ReconcileResult, error) {
	f.reconciled = true
	return f.result, f.err
}

func (f *fakeObserver) OnSessionCreated(context.Context, *session.Session) {}

func (f *fakeObserver) OnSessionStopped(context.Context, *session.Session) {}

func (f *fakeObserver) OnAgentEvent(context.Context, string, any) {}

type fakeServer struct {
	name       string
	onShutdown func()
}

func (f *fakeServer) Start(context.Context) error {
	return nil
}

func (f *fakeServer) Shutdown(context.Context) error {
	if f.onShutdown != nil {
		f.onShutdown()
	}
	return nil
}

type recordingRegistry struct {
	path    string
	onClose func()
}

func (r *recordingRegistry) Path() string {
	return r.path
}

func (r *recordingRegistry) InsertWorkspace(context.Context, workspacepkg.Workspace) error {
	return nil
}

func (r *recordingRegistry) UpdateWorkspace(context.Context, workspacepkg.Workspace) error {
	return nil
}

func (r *recordingRegistry) DeleteWorkspace(context.Context, string) error {
	return nil
}

func (r *recordingRegistry) GetWorkspace(context.Context, string) (workspacepkg.Workspace, error) {
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *recordingRegistry) GetWorkspaceByPath(context.Context, string) (workspacepkg.Workspace, error) {
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *recordingRegistry) GetWorkspaceByName(context.Context, string) (workspacepkg.Workspace, error) {
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *recordingRegistry) ListWorkspaces(context.Context) ([]workspacepkg.Workspace, error) {
	return nil, nil
}

func (r *recordingRegistry) RegisterSession(context.Context, store.SessionInfo) error {
	return nil
}

func (r *recordingRegistry) UpdateSessionState(context.Context, store.SessionStateUpdate) error {
	return nil
}

func (r *recordingRegistry) ListSessions(context.Context, store.SessionListQuery) ([]store.SessionInfo, error) {
	return nil, nil
}

func (r *recordingRegistry) ReconcileSessions(context.Context, []store.SessionInfo) (store.ReconcileResult, error) {
	return store.ReconcileResult{}, nil
}

func (r *recordingRegistry) WriteEventSummary(context.Context, store.EventSummary) error {
	return nil
}

func (r *recordingRegistry) ListEventSummaries(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
	return nil, nil
}

func (r *recordingRegistry) UpdateTokenStats(context.Context, store.TokenStatsUpdate) error {
	return nil
}

func (r *recordingRegistry) ListTokenStats(context.Context, store.TokenStatsQuery) ([]store.TokenStats, error) {
	return nil, nil
}

func (r *recordingRegistry) WritePermissionLog(context.Context, store.PermissionLogEntry) error {
	return nil
}

func (r *recordingRegistry) ListPermissionLog(context.Context, store.PermissionLogQuery) ([]store.PermissionLogEntry, error) {
	return nil, nil
}

func (r *recordingRegistry) WriteNetworkAudit(context.Context, store.NetworkAuditEntry) error {
	return nil
}

func (r *recordingRegistry) ListNetworkAudit(context.Context, store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error) {
	return nil, nil
}

func (r *recordingRegistry) Close(context.Context) error {
	if r.onClose != nil {
		r.onClose()
	}
	return nil
}

type recordingNotifier struct {
	events []string
}

func (n *recordingNotifier) OnSessionCreated(context.Context, *session.Session) {
	n.events = append(n.events, "created")
}

func (n *recordingNotifier) OnSessionStopped(context.Context, *session.Session) {
	n.events = append(n.events, "stopped")
}

func (n *recordingNotifier) OnAgentEvent(context.Context, string, any) {
	n.events = append(n.events, "agent")
}

type recordingHookTelemetrySink struct {
	calls []struct {
		sessionID string
		record    hookspkg.HookRunRecord
	}
}

func (s *recordingHookTelemetrySink) WriteHookRecord(_ context.Context, sessionID string, record hookspkg.HookRunRecord) error {
	s.calls = append(s.calls, struct {
		sessionID string
		record    hookspkg.HookRunRecord
	}{
		sessionID: sessionID,
		record:    record,
	})
	return nil
}

func (s *recordingHookTelemetrySink) count() int {
	if s == nil {
		return 0
	}
	return len(s.calls)
}

type noopMemoryObserver struct{}

func (noopMemoryObserver) OnMemoryConsolidated(context.Context, automationpkg.MemoryConsolidatedEvent) error {
	return nil
}

type fakeAutomationManager struct {
	jobs              []automationpkg.Job
	triggers          []automationpkg.Trigger
	runs              []automationpkg.Run
	status            automationpkg.ManagerStatus
	startCount        int
	shutdownCount     int
	startErr          error
	shutdownErr       error
	onStart           func()
	onShutdown        func()
	sessionObserver   session.Notifier
	hookTelemetrySink hookspkg.TelemetrySink
}

func (f *fakeAutomationManager) Start(context.Context) error {
	f.startCount++
	if f.onStart != nil {
		f.onStart()
	}
	f.status.Running = true
	return f.startErr
}

func (f *fakeAutomationManager) Shutdown(context.Context) error {
	f.shutdownCount++
	if f.onShutdown != nil {
		f.onShutdown()
	}
	f.status.Running = false
	f.status.SchedulerRunning = false
	return f.shutdownErr
}

func (f *fakeAutomationManager) Jobs(context.Context) ([]automationpkg.Job, error) {
	return append([]automationpkg.Job(nil), f.jobs...), nil
}

func (f *fakeAutomationManager) ListJobs(_ context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error) {
	jobs := make([]automationpkg.Job, 0, len(f.jobs))
	for _, job := range f.jobs {
		if query.Scope != "" && job.Scope != query.Scope {
			continue
		}
		if query.WorkspaceID != "" && job.WorkspaceID != query.WorkspaceID {
			continue
		}
		if query.Source != "" && job.Source != query.Source {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (f *fakeAutomationManager) GetJob(_ context.Context, id string) (automationpkg.Job, error) {
	for _, job := range f.jobs {
		if job.ID == strings.TrimSpace(id) {
			return job, nil
		}
	}
	return automationpkg.Job{}, automationpkg.ErrJobNotFound
}

func (f *fakeAutomationManager) CreateJob(_ context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	f.jobs = append(f.jobs, job)
	return job, nil
}

func (f *fakeAutomationManager) UpdateJob(_ context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	for i := range f.jobs {
		if f.jobs[i].ID == strings.TrimSpace(job.ID) {
			f.jobs[i] = job
			return job, nil
		}
	}
	return automationpkg.Job{}, automationpkg.ErrJobNotFound
}

func (f *fakeAutomationManager) DeleteJob(_ context.Context, id string) error {
	for i := range f.jobs {
		if f.jobs[i].ID == strings.TrimSpace(id) {
			f.jobs = append(f.jobs[:i], f.jobs[i+1:]...)
			return nil
		}
	}
	return automationpkg.ErrJobNotFound
}

func (f *fakeAutomationManager) TriggerJob(_ context.Context, id string) (automationpkg.Run, error) {
	run := automationpkg.Run{ID: "run-" + strings.TrimSpace(id), JobID: strings.TrimSpace(id), Status: automationpkg.RunCompleted, Attempt: 1}
	f.runs = append(f.runs, run)
	return run, nil
}

func (f *fakeAutomationManager) Triggers(context.Context) ([]automationpkg.Trigger, error) {
	return append([]automationpkg.Trigger(nil), f.triggers...), nil
}

func (f *fakeAutomationManager) ListTriggers(_ context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error) {
	triggers := make([]automationpkg.Trigger, 0, len(f.triggers))
	for _, trigger := range f.triggers {
		if query.Scope != "" && trigger.Scope != query.Scope {
			continue
		}
		if query.WorkspaceID != "" && trigger.WorkspaceID != query.WorkspaceID {
			continue
		}
		if query.Event != "" && trigger.Event != query.Event {
			continue
		}
		if query.Source != "" && trigger.Source != query.Source {
			continue
		}
		triggers = append(triggers, trigger)
	}
	return triggers, nil
}

func (f *fakeAutomationManager) GetTrigger(_ context.Context, id string) (automationpkg.Trigger, error) {
	for _, trigger := range f.triggers {
		if trigger.ID == strings.TrimSpace(id) {
			return trigger, nil
		}
	}
	return automationpkg.Trigger{}, automationpkg.ErrTriggerNotFound
}

func (f *fakeAutomationManager) CreateTrigger(_ context.Context, trigger automationpkg.Trigger, _ string) (automationpkg.Trigger, error) {
	f.triggers = append(f.triggers, trigger)
	return trigger, nil
}

func (f *fakeAutomationManager) UpdateTrigger(_ context.Context, trigger automationpkg.Trigger, _ *string) (automationpkg.Trigger, error) {
	for i := range f.triggers {
		if f.triggers[i].ID == strings.TrimSpace(trigger.ID) {
			f.triggers[i] = trigger
			return trigger, nil
		}
	}
	return automationpkg.Trigger{}, automationpkg.ErrTriggerNotFound
}

func (f *fakeAutomationManager) DeleteTrigger(_ context.Context, id string) error {
	for i := range f.triggers {
		if f.triggers[i].ID == strings.TrimSpace(id) {
			f.triggers = append(f.triggers[:i], f.triggers[i+1:]...)
			return nil
		}
	}
	return automationpkg.ErrTriggerNotFound
}

func (f *fakeAutomationManager) Runs(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error) {
	return append([]automationpkg.Run(nil), f.runs...), nil
}

func (f *fakeAutomationManager) ListRuns(_ context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
	runs := make([]automationpkg.Run, 0, len(f.runs))
	for _, run := range f.runs {
		if query.JobID != "" && run.JobID != query.JobID {
			continue
		}
		if query.TriggerID != "" && run.TriggerID != query.TriggerID {
			continue
		}
		if query.Status != "" && run.Status != query.Status {
			continue
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (f *fakeAutomationManager) GetRun(_ context.Context, id string) (automationpkg.Run, error) {
	for _, run := range f.runs {
		if run.ID == strings.TrimSpace(id) {
			return run, nil
		}
	}
	return automationpkg.Run{}, automationpkg.ErrRunNotFound
}

func (f *fakeAutomationManager) Status(context.Context) (automationpkg.ManagerStatus, error) {
	return f.status, nil
}

func (f *fakeAutomationManager) SetJobEnabled(context.Context, string, bool) (automationpkg.Job, error) {
	return automationpkg.Job{}, nil
}

func (f *fakeAutomationManager) SetTriggerEnabled(context.Context, string, bool) (automationpkg.Trigger, error) {
	return automationpkg.Trigger{}, nil
}

func (f *fakeAutomationManager) HandleWebhook(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
	return automationpkg.TriggerResult{}, nil
}

func (f *fakeAutomationManager) FireExtensionTrigger(_ context.Context, request automationpkg.ExtensionTriggerRequest) (automationpkg.TriggerResult, error) {
	return automationpkg.TriggerResult{
		Matched: 0,
		Runs:    append([]automationpkg.Run(nil), f.runs...),
	}, request.Validate("extension_trigger")
}

func (f *fakeAutomationManager) SessionObserver() session.Notifier {
	if f.sessionObserver != nil {
		return f.sessionObserver
	}
	return &recordingNotifier{}
}

func (f *fakeAutomationManager) HookTelemetrySink() hookspkg.TelemetrySink {
	if f.hookTelemetrySink != nil {
		return f.hookTelemetrySink
	}
	return &recordingHookTelemetrySink{}
}

func (*fakeAutomationManager) MemoryObserver() automationpkg.MemoryConsolidationObserver {
	return noopMemoryObserver{}
}

type fakeHookRuntime struct {
	version          int64
	onRebuild        func(context.Context) error
	onClose          func()
	onDispatchCreate func(context.Context, hookspkg.SessionPostCreatePayload) error
	onDispatchStop   func(context.Context, hookspkg.SessionPostStopPayload) error
	onTurnStart      func(context.Context, hookspkg.TurnStartPayload) error
	onTurnEnd        func(context.Context, hookspkg.TurnEndPayload) error
	onMessageStart   func(context.Context, hookspkg.MessageStartPayload) error
	onMessageDelta   func(context.Context, hookspkg.MessageDeltaPayload) error
	onMessageEnd     func(context.Context, hookspkg.MessageEndPayload) error
	onPreCompact     func(context.Context, hookspkg.ContextPreCompactPayload) error
	onPostCompact    func(context.Context, hookspkg.ContextPostCompactPayload) error
	onAgentEvent     func(context.Context, string, any)
}

func (f *fakeHookRuntime) Rebuild(ctx context.Context) error {
	if f.onRebuild != nil {
		return f.onRebuild(ctx)
	}
	return nil
}

func (f *fakeHookRuntime) Close() {
	if f.onClose != nil {
		f.onClose()
	}
}

func (f *fakeHookRuntime) Version() int64 {
	return f.version
}

func (f *fakeHookRuntime) DispatchSessionPreCreate(_ context.Context, payload hookspkg.SessionPreCreatePayload) (hookspkg.SessionPreCreatePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchSessionPostCreate(ctx context.Context, payload hookspkg.SessionPostCreatePayload) (hookspkg.SessionPostCreatePayload, error) {
	if f.onDispatchCreate != nil {
		return payload, f.onDispatchCreate(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchSessionPreResume(_ context.Context, payload hookspkg.SessionPreResumePayload) (hookspkg.SessionPreResumePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchSessionPostResume(_ context.Context, payload hookspkg.SessionPostResumePayload) (hookspkg.SessionPostResumePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchSessionPreStop(_ context.Context, payload hookspkg.SessionPreStopPayload) (hookspkg.SessionPreStopPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchSessionPostStop(ctx context.Context, payload hookspkg.SessionPostStopPayload) (hookspkg.SessionPostStopPayload, error) {
	if f.onDispatchStop != nil {
		return payload, f.onDispatchStop(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchInputPreSubmit(_ context.Context, payload hookspkg.InputPreSubmitPayload) (hookspkg.InputPreSubmitPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchPromptPostAssemble(_ context.Context, payload hookspkg.PromptPayload) (hookspkg.PromptPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchEventPreRecord(_ context.Context, payload hookspkg.EventPreRecordPayload) (hookspkg.EventPreRecordPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchEventPostRecord(_ context.Context, payload hookspkg.EventPostRecordPayload) (hookspkg.EventPostRecordPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationJobPreFire(_ context.Context, payload hookspkg.AutomationJobPreFirePayload) (hookspkg.AutomationJobPreFirePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationJobPostFire(_ context.Context, payload hookspkg.AutomationJobPostFirePayload) (hookspkg.AutomationJobPostFirePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationTriggerPreFire(_ context.Context, payload hookspkg.AutomationTriggerPreFirePayload) (hookspkg.AutomationTriggerPreFirePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationTriggerPostFire(_ context.Context, payload hookspkg.AutomationTriggerPostFirePayload) (hookspkg.AutomationTriggerPostFirePayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationRunCompleted(_ context.Context, payload hookspkg.AutomationRunCompletedPayload) (hookspkg.AutomationRunCompletedPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAutomationRunFailed(_ context.Context, payload hookspkg.AutomationRunFailedPayload) (hookspkg.AutomationRunFailedPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAgentPreStart(_ context.Context, payload hookspkg.AgentPreStartPayload) (hookspkg.AgentPreStartPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAgentSpawned(_ context.Context, payload hookspkg.AgentSpawnedPayload) (hookspkg.AgentSpawnedPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAgentCrashed(_ context.Context, payload hookspkg.AgentCrashedPayload) (hookspkg.AgentCrashedPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchAgentStopped(_ context.Context, payload hookspkg.AgentStoppedPayload) (hookspkg.AgentStoppedPayload, error) {
	return payload, nil
}

func (f *fakeHookRuntime) DispatchTurnStart(ctx context.Context, payload hookspkg.TurnStartPayload) (hookspkg.TurnStartPayload, error) {
	if f.onTurnStart != nil {
		return payload, f.onTurnStart(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchTurnEnd(ctx context.Context, payload hookspkg.TurnEndPayload) (hookspkg.TurnEndPayload, error) {
	if f.onTurnEnd != nil {
		return payload, f.onTurnEnd(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchMessageStart(ctx context.Context, payload hookspkg.MessageStartPayload) (hookspkg.MessageStartPayload, error) {
	if f.onMessageStart != nil {
		return payload, f.onMessageStart(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchMessageDelta(ctx context.Context, payload hookspkg.MessageDeltaPayload) (hookspkg.MessageDeltaPayload, error) {
	if f.onMessageDelta != nil {
		return payload, f.onMessageDelta(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchMessageEnd(ctx context.Context, payload hookspkg.MessageEndPayload) (hookspkg.MessageEndPayload, error) {
	if f.onMessageEnd != nil {
		return payload, f.onMessageEnd(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchContextPreCompact(ctx context.Context, payload hookspkg.ContextPreCompactPayload) (hookspkg.ContextPreCompactPayload, error) {
	if f.onPreCompact != nil {
		return payload, f.onPreCompact(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) DispatchContextPostCompact(ctx context.Context, payload hookspkg.ContextPostCompactPayload) (hookspkg.ContextPostCompactPayload, error) {
	if f.onPostCompact != nil {
		return payload, f.onPostCompact(ctx, payload)
	}
	return payload, nil
}

func (f *fakeHookRuntime) OnAgentEvent(ctx context.Context, sessionID string, event any) {
	if f.onAgentEvent != nil {
		f.onAgentEvent(ctx, sessionID, event)
	}
}

func testHookExecutorResolver(native map[string]hookspkg.Executor) hookspkg.ExecutorResolver {
	return func(decl hookspkg.HookDecl) (hookspkg.Executor, error) {
		if decl.ExecutorKind == hookspkg.HookExecutorNative {
			executor := native[strings.TrimSpace(decl.Name)]
			if executor == nil {
				return nil, errors.New("missing native executor")
			}
			return executor, nil
		}
		return defaultDaemonExecutorResolver(decl)
	}
}

type fakeDreamService struct {
	mu             sync.Mutex
	shouldRun      bool
	shouldRunErr   error
	runErr         error
	shouldRunCalls int
	runCalls       int
	runHook        func(context.Context, memory.SessionSpawner, string) error
}

func (f *fakeDreamService) ShouldRun() (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shouldRunCalls++
	return f.shouldRun, f.shouldRunErr
}

func (f *fakeDreamService) Run(ctx context.Context, spawn memory.SessionSpawner, workspace string) error {
	f.mu.Lock()
	f.runCalls++
	runHook := f.runHook
	runErr := f.runErr
	f.mu.Unlock()

	if runHook != nil {
		return runHook(ctx, spawn, workspace)
	}
	return runErr
}

func (f *fakeDreamService) runCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.runCalls
}

type portReportingServer struct {
	port int
}

func (s portReportingServer) Start(context.Context) error {
	return nil
}

func (s portReportingServer) Shutdown(context.Context) error {
	return nil
}

func (s portReportingServer) Port() int {
	return s.port
}

const (
	daemonExtensionHelperEnvKey      = "AGH_TEST_DAEMON_EXTENSION_HELPER"
	daemonExtensionHelperScenarioKey = "AGH_TEST_DAEMON_EXTENSION_SCENARIO"
	daemonExtensionHelperMarkerKey   = "AGH_TEST_DAEMON_EXTENSION_MARKER"
)

func TestDaemonExtensionHelperProcess(t *testing.T) {
	if os.Getenv(daemonExtensionHelperEnvKey) != "1" {
		return
	}

	server := newDaemonExtensionHelperServer(
		strings.TrimSpace(os.Getenv(daemonExtensionHelperScenarioKey)),
		strings.TrimSpace(os.Getenv(daemonExtensionHelperMarkerKey)),
	)
	os.Exit(server.run())
}

type fakeExtensionRuntime struct {
	startCount  int
	stopCount   int
	reloadCount int
	startErr    error
	stopErr     error
	reloadErr   error
	hookDecls   []hookspkg.HookDecl
	hookErr     error
	getExt      *extensionpkg.Extension
	getErr      error
	onStart     func()
	onStop      func()
}

func (f *fakeExtensionRuntime) Start(context.Context) error {
	f.startCount++
	if f.onStart != nil {
		f.onStart()
	}
	return f.startErr
}

func (f *fakeExtensionRuntime) Stop(context.Context) error {
	f.stopCount++
	if f.onStop != nil {
		f.onStop()
	}
	return f.stopErr
}

func (f *fakeExtensionRuntime) Reload(context.Context) error {
	f.reloadCount++
	return f.reloadErr
}

func (f *fakeExtensionRuntime) Get(string) (*extensionpkg.Extension, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.getExt, nil
}

func (f *fakeExtensionRuntime) HookDeclarations(context.Context) ([]hookspkg.HookDecl, error) {
	decls := make([]hookspkg.HookDecl, 0, len(f.hookDecls))
	for _, decl := range f.hookDecls {
		cloned := decl
		cloned.Args = append([]string(nil), decl.Args...)
		if len(decl.Env) > 0 {
			cloned.Env = make(map[string]string, len(decl.Env))
			for key, value := range decl.Env {
				cloned.Env[key] = value
			}
		}
		if len(decl.Metadata) > 0 {
			cloned.Metadata = make(map[string]string, len(decl.Metadata))
			for key, value := range decl.Metadata {
				cloned.Metadata[key] = value
			}
		}
		decls = append(decls, cloned)
	}
	return decls, f.hookErr
}

type daemonTestExtensionOptions struct {
	runtimeCommand string
	runtimeArgs    []string
	runtimeEnv     map[string]string
	hookCommand    string
	hookArgs       []string
	hookEvent      hookspkg.HookEvent
	capabilities   []string
	actions        []string
	security       []string
}

func openDaemonTestGlobalDB(t *testing.T) *globaldb.GlobalDB {
	t.Helper()

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), filepath.Join(t.TempDir(), store.GlobalDatabaseName))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("GlobalDB.Close() error = %v", err)
		}
	})
	return db
}

func installDaemonTestExtension(t *testing.T, db *globaldb.GlobalDB, name string, opts daemonTestExtensionOptions, enabled bool) string {
	t.Helper()

	if db == nil {
		t.Fatal("installDaemonTestExtension() db = nil")
	}

	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", dir, err)
	}
	manifestPath := filepath.Join(dir, "extension.toml")
	if err := os.WriteFile(manifestPath, []byte(daemonTestExtensionManifest(name, opts)), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", manifestPath, err)
	}

	manifest, err := extensionpkg.LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest(%q) error = %v", dir, err)
	}
	checksum, err := extensionpkg.ComputeDirectoryChecksum(dir)
	if err != nil {
		t.Fatalf("ComputeDirectoryChecksum(%q) error = %v", dir, err)
	}

	registry := extensionpkg.NewRegistry(db.DB())
	if err := registry.Install(manifest, dir, checksum); err != nil {
		t.Fatalf("Registry.Install(%q) error = %v", name, err)
	}
	if !enabled {
		if err := registry.Disable(name); err != nil {
			t.Fatalf("Registry.Disable(%q) error = %v", name, err)
		}
	}

	return dir
}

func daemonTestExtensionManifest(name string, opts daemonTestExtensionOptions) string {
	command := strings.TrimSpace(opts.runtimeCommand)
	if command == "" {
		command = "fake-extension"
	}
	capabilities := append([]string(nil), opts.capabilities...)
	if opts.capabilities == nil {
		capabilities = []string{"memory.backend"}
	}
	actions := append([]string(nil), opts.actions...)
	if opts.actions == nil {
		actions = []string{"sessions/list"}
	}
	security := append([]string(nil), opts.security...)
	if opts.security == nil {
		security = []string{"session.read"}
	}

	event := opts.hookEvent
	if event == "" {
		event = hookspkg.HookSessionPostCreate
	}

	var builder strings.Builder
	fmt.Fprintf(&builder, `[extension]
name = %q
version = "0.2.1"
description = "Daemon extension test fixture"
min_agh_version = "0.5.0"

[resources]
`, name)

	if strings.TrimSpace(opts.hookCommand) != "" {
		fmt.Fprintf(&builder, `
[[resources.hooks]]
name = %q
event = %q
mode = "sync"
executor.kind = "subprocess"
executor.command = %q
`, name+"-hook", string(event), opts.hookCommand)
		if len(opts.hookArgs) > 0 {
			builder.WriteString("executor.args = " + daemonTOMLStringArray(opts.hookArgs) + "\n")
		}
	}

	builder.WriteString(`
[capabilities]
provides = ` + daemonTOMLStringArray(capabilities) + `

[actions]
requires = ` + daemonTOMLStringArray(actions) + `

[subprocess]
command = ` + fmt.Sprintf("%q", command) + `
`)
	if len(opts.runtimeArgs) > 0 {
		builder.WriteString("args = " + daemonTOMLStringArray(opts.runtimeArgs) + "\n")
	}
	if len(opts.runtimeEnv) > 0 {
		builder.WriteString("\n[subprocess.env]\n")
		keys := make([]string, 0, len(opts.runtimeEnv))
		for key := range opts.runtimeEnv {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			fmt.Fprintf(&builder, "%s = %q\n", key, opts.runtimeEnv[key])
		}
	}

	builder.WriteString(`
[security]
capabilities = ` + daemonTOMLStringArray(security) + `
`)

	return builder.String()
}

func daemonTOMLStringArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func TestDaemonTestExtensionManifest(t *testing.T) {
	t.Run("ShouldApplyDefaultListsWhenOptionsAreNil", func(t *testing.T) {
		t.Parallel()

		manifest := daemonTestExtensionManifest("service-ext", daemonTestExtensionOptions{})
		for _, expected := range []string{
			`provides = ["memory.backend"]`,
			`requires = ["sessions/list"]`,
			`capabilities = ["session.read"]`,
		} {
			if !strings.Contains(manifest, expected) {
				t.Fatalf("daemonTestExtensionManifest() missing default %q in manifest %q", expected, manifest)
			}
		}
	})

	t.Run("ShouldPreserveExplicitEmptyLists", func(t *testing.T) {
		t.Parallel()

		manifest := daemonTestExtensionManifest("service-ext", daemonTestExtensionOptions{
			capabilities: []string{},
			actions:      []string{},
			security:     []string{},
		})

		for _, expected := range []string{
			"provides = []",
			"requires = []",
			"capabilities = []",
		} {
			if !strings.Contains(manifest, expected) {
				t.Fatalf("daemonTestExtensionManifest() missing explicit empty list %q in manifest %q", expected, manifest)
			}
		}
		for _, unexpected := range []string{"memory.backend", "sessions/list", "session.read"} {
			if strings.Contains(manifest, unexpected) {
				t.Fatalf("daemonTestExtensionManifest() unexpectedly injected %q into manifest %q", unexpected, manifest)
			}
		}
	})
}

func TestDaemonExtensionHelperHarness(t *testing.T) {
	t.Parallel()

	command := daemonExtensionHelperCommand(t)
	if strings.TrimSpace(command) == "" {
		t.Fatal("daemonExtensionHelperCommand() returned an empty path")
	}

	if got := daemonExtensionHelperArgs(); !testutil.EqualStringSlices(got, []string{"-test.run=TestDaemonExtensionHelperProcess"}) {
		t.Fatalf("daemonExtensionHelperArgs() = %#v, want helper test selector", got)
	}

	env := daemonExtensionHelperEnv("/tmp/daemon-helper-marker")
	if env[daemonExtensionHelperEnvKey] != "1" {
		t.Fatalf("daemonExtensionHelperEnv() helper flag = %q, want 1", env[daemonExtensionHelperEnvKey])
	}
	if env[daemonExtensionHelperMarkerKey] != "/tmp/daemon-helper-marker" {
		t.Fatalf("daemonExtensionHelperEnv() marker = %q, want /tmp/daemon-helper-marker", env[daemonExtensionHelperMarkerKey])
	}

	withoutMarker := daemonExtensionHelperEnv("")
	if _, ok := withoutMarker[daemonExtensionHelperMarkerKey]; ok {
		t.Fatalf("daemonExtensionHelperEnv(\"\") unexpectedly set %q", daemonExtensionHelperMarkerKey)
	}

	withScenario := daemonExtensionHelperScenarioEnv("record_initialize", "/tmp/daemon-helper-scenario")
	if got := withScenario[daemonExtensionHelperScenarioKey]; got != "record_initialize" {
		t.Fatalf("daemonExtensionHelperScenarioEnv() scenario = %q, want record_initialize", got)
	}
}

func daemonExtensionHelperCommand(t *testing.T) string {
	t.Helper()

	command, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	return command
}

func daemonExtensionHelperArgs() []string {
	return []string{"-test.run=TestDaemonExtensionHelperProcess"}
}

func daemonExtensionHelperEnv(markerPath string) map[string]string {
	env := map[string]string{
		daemonExtensionHelperEnvKey: "1",
	}
	if strings.TrimSpace(markerPath) != "" {
		env[daemonExtensionHelperMarkerKey] = markerPath
	}
	return env
}

func daemonExtensionHelperScenarioEnv(scenario string, markerPath string) map[string]string {
	env := daemonExtensionHelperEnv(markerPath)
	if strings.TrimSpace(scenario) != "" {
		env[daemonExtensionHelperScenarioKey] = scenario
	}
	return env
}

func TestDaemonExtensionHelperShutdownAppendsMarkerLine(t *testing.T) {
	t.Parallel()

	marker := filepath.Join(t.TempDir(), "helper-marker.jsonl")
	if err := appendMarkerLine(marker, `{"event":"initialize"}`); err != nil {
		t.Fatalf("appendMarkerLine(initialize) error = %v", err)
	}
	if err := appendMarkerLine(marker, `{"event":"delivery"}`); err != nil {
		t.Fatalf("appendMarkerLine(delivery) error = %v", err)
	}

	server := newDaemonExtensionHelperServer("", marker)
	server.encoder = json.NewEncoder(io.Discard)

	exit, err := server.handleRequest(daemonExtensionHelperRequest{ID: "1", Method: "shutdown"})
	if err != nil {
		t.Fatalf("handleRequest(shutdown) error = %v", err)
	}
	if exit {
		t.Fatal("handleRequest(shutdown) exit = true, want false")
	}

	payload, readErr := os.ReadFile(marker)
	if readErr != nil {
		t.Fatalf("os.ReadFile(marker) error = %v", readErr)
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if got, want := len(lines), 3; got != want {
		t.Fatalf("marker line count = %d, want %d; payload=%q", got, want, string(payload))
	}
	if got, want := lines[2], "shutdown"; got != want {
		t.Fatalf("marker final line = %q, want %q", got, want)
	}
}

func TestDaemonExtensionHelperHandleRequest(t *testing.T) {
	t.Run("ShouldRejectInvalidDeliveryRequestsBeforeRecordingOrAcking", func(t *testing.T) {
		t.Parallel()

		marker := filepath.Join(t.TempDir(), "helper-marker.jsonl")
		var output bytes.Buffer

		server := newDaemonExtensionHelperServer("", marker)
		server.encoder = json.NewEncoder(&output)

		params, err := json.Marshal(bridgepkg.DeliveryRequest{
			Event: bridgepkg.DeliveryEvent{
				DeliveryID:       "delivery-1",
				BridgeInstanceID: "brg-1",
				RoutingKey: bridgepkg.RoutingKey{
					Scope:            bridgepkg.ScopeGlobal,
					BridgeInstanceID: "brg-1",
					PeerID:           "peer-1",
				},
				DeliveryTarget: bridgepkg.DeliveryTarget{
					BridgeInstanceID: "brg-1",
					PeerID:           "peer-1",
					Mode:             bridgepkg.DeliveryModeDirectSend,
				},
				Seq:       1,
				EventType: bridgepkg.DeliveryEventTypeResume,
			},
		})
		if err != nil {
			t.Fatalf("json.Marshal(delivery request) error = %v", err)
		}

		exit, err := server.handleRequest(daemonExtensionHelperRequest{
			ID:     "1",
			Method: "bridges/deliver",
			Params: params,
		})
		if exit {
			t.Fatal("handleRequest(bridges/deliver) exit = true, want false")
		}
		if err == nil {
			t.Fatal("handleRequest(bridges/deliver) error = nil, want delivery validation failure")
		}
		if !strings.Contains(err.Error(), "validate bridges/deliver request") {
			t.Fatalf("handleRequest(bridges/deliver) error = %q, want validation context", err)
		}

		payload, readErr := os.ReadFile(marker)
		if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
			t.Fatalf("os.ReadFile(marker) error = %v", readErr)
		}
		if strings.TrimSpace(string(payload)) != "" {
			t.Fatalf("marker payload = %q, want no recorded delivery", string(payload))
		}
		if strings.TrimSpace(output.String()) != "" {
			t.Fatalf("helper output = %q, want no ACK payload", output.String())
		}
	})
}

func TestDaemonExtensionHelperMarkerRecording(t *testing.T) {
	t.Run("ShouldWrapInitializeMarkerFailuresWithOperationContext", func(t *testing.T) {
		t.Parallel()

		marker := filepath.Join(t.TempDir(), "marker-dir")
		if err := os.Mkdir(marker, 0o755); err != nil {
			t.Fatalf("os.Mkdir(marker) error = %v", err)
		}

		server := newDaemonExtensionHelperServer("", marker)
		err := server.recordInitialize(subprocess.InitializeRequest{}, subprocess.InitializeResponse{})
		if err == nil {
			t.Fatal("recordInitialize() error = nil, want marker append failure")
		}
		if !strings.Contains(err.Error(), "record initialize marker") {
			t.Fatalf("recordInitialize() error = %q, want initialize context", err)
		}
		if !strings.Contains(err.Error(), "append marker line") {
			t.Fatalf("recordInitialize() error = %q, want append context", err)
		}
	})

	t.Run("ShouldWrapDeliveryMarkerFailuresWithOperationContext", func(t *testing.T) {
		t.Parallel()

		marker := filepath.Join(t.TempDir(), "marker-dir")
		if err := os.Mkdir(marker, 0o755); err != nil {
			t.Fatalf("os.Mkdir(marker) error = %v", err)
		}

		server := newDaemonExtensionHelperServer("", marker)
		err := server.recordDelivery(bridgepkg.DeliveryRequest{})
		if err == nil {
			t.Fatal("recordDelivery() error = nil, want marker append failure")
		}
		if !strings.Contains(err.Error(), "record delivery marker") {
			t.Fatalf("recordDelivery() error = %q, want delivery context", err)
		}
		if !strings.Contains(err.Error(), "append marker line") {
			t.Fatalf("recordDelivery() error = %q, want append context", err)
		}
	})
}

type daemonExtensionHelperServer struct {
	scenario                string
	marker                  string
	scanner                 *bufio.Scanner
	encoder                 *json.Encoder
	slowDeliveryRelease     chan struct{}
	slowDeliveryReleaseOnce sync.Once
	slowDeliveryWG          sync.WaitGroup
	mu                      sync.Mutex
}

type daemonExtensionHelperRequest struct {
	ID     any             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func newDaemonExtensionHelperServer(scenario string, marker string) *daemonExtensionHelperServer {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)

	return &daemonExtensionHelperServer{
		scenario:            scenario,
		marker:              marker,
		scanner:             scanner,
		encoder:             encoder,
		slowDeliveryRelease: make(chan struct{}),
	}
}

func (h *daemonExtensionHelperServer) run() int {
	for h.scanner.Scan() {
		var req daemonExtensionHelperRequest
		if err := json.Unmarshal(h.scanner.Bytes(), &req); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		exitProcess, err := h.handleRequest(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if exitProcess {
			return 1
		}
	}
	if err := h.scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func (h *daemonExtensionHelperServer) handleRequest(req daemonExtensionHelperRequest) (bool, error) {
	switch req.Method {
	case "initialize":
		var params subprocess.InitializeRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false, err
		}
		response := daemonExtensionInitializeResponse(params)
		if err := h.sendResult(req.ID, response); err != nil {
			return false, err
		}
		if h.scenario == "record_initialize" || h.scenario == "auto_exit_record_initialize" {
			if err := h.recordInitialize(params, response); err != nil {
				return false, err
			}
		}
		return h.scenario == "auto_exit_record_initialize", nil
	case "health_check":
		return false, h.sendResult(req.ID, subprocess.HealthCheckResponse{Healthy: true})
	case "bridges/deliver":
		var params bridgepkg.DeliveryRequest
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return false, fmt.Errorf("decode bridges/deliver request: %w", err)
		}
		if err := params.Validate(); err != nil {
			return false, fmt.Errorf("validate bridges/deliver request: %w", err)
		}
		if err := h.recordDelivery(params); err != nil {
			return false, err
		}

		ack := bridgepkg.DeliveryAck{
			DeliveryID: strings.TrimSpace(params.Event.DeliveryID),
			Seq:        params.Event.Seq,
		}
		if ack.Seq > 0 {
			ack.RemoteMessageID = fmt.Sprintf("remote-%d", ack.Seq)
		}
		if ack.Seq > 1 {
			ack.ReplaceRemoteMessageID = fmt.Sprintf("remote-%d", ack.Seq-1)
		}
		switch h.scenario {
		case "slow_record_deliveries":
			h.sendDelayedDeliveryResult(req.ID, ack)
			return false, nil
		case "exit_once_record_deliveries":
			if markerLineCount(h.marker) == 1 {
				return true, nil
			}
		}
		return false, h.sendResult(req.ID, ack)
	case "shutdown":
		h.releaseSlowDeliveries()
		h.waitSlowDeliveries()
		if strings.TrimSpace(h.marker) != "" {
			if err := appendMarkerLine(h.marker, "shutdown"); err != nil {
				return false, err
			}
		}
		return false, h.sendResult(req.ID, subprocess.ShutdownResponse{Acknowledged: true})
	default:
		return false, h.sendResult(req.ID, map[string]any{})
	}
}

func (h *daemonExtensionHelperServer) sendDelayedDeliveryResult(id any, ack bridgepkg.DeliveryAck) {
	h.slowDeliveryWG.Add(1)
	go func() {
		defer h.slowDeliveryWG.Done()
		<-h.slowDeliveryRelease
		if err := h.sendResult(id, ack); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}()
}

func (h *daemonExtensionHelperServer) releaseSlowDeliveries() {
	h.slowDeliveryReleaseOnce.Do(func() {
		close(h.slowDeliveryRelease)
	})
}

func (h *daemonExtensionHelperServer) waitSlowDeliveries() {
	h.slowDeliveryWG.Wait()
}

func (h *daemonExtensionHelperServer) sendResult(id any, result any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.encoder.Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

func (h *daemonExtensionHelperServer) recordInitialize(
	request subprocess.InitializeRequest,
	response subprocess.InitializeResponse,
) error {
	if strings.TrimSpace(h.marker) == "" {
		return nil
	}

	payload, err := json.Marshal(daemonInitializeMarker{
		Request:  request,
		Response: response,
	})
	if err != nil {
		return fmt.Errorf("record initialize marker: marshal payload: %w", err)
	}
	if err := appendMarkerLine(h.marker, string(payload)); err != nil {
		return fmt.Errorf("record initialize marker: %w", err)
	}
	return nil
}

func (h *daemonExtensionHelperServer) recordDelivery(request bridgepkg.DeliveryRequest) error {
	if strings.TrimSpace(h.marker) == "" {
		return nil
	}

	payload, err := json.Marshal(daemonDeliveryMarker{
		PID:     os.Getpid(),
		Request: request,
	})
	if err != nil {
		return fmt.Errorf("record delivery marker: marshal payload: %w", err)
	}
	if err := appendMarkerLine(h.marker, string(payload)); err != nil {
		return fmt.Errorf("record delivery marker: %w", err)
	}
	return nil
}

func daemonExtensionInitializeResponse(req subprocess.InitializeRequest) subprocess.InitializeResponse {
	implementedMethods := []string{"health_check", "shutdown"}
	implementedMethods = append(implementedMethods, extensionprotocol.CapabilityServiceMethods(req.Capabilities.Provides)...)

	return subprocess.InitializeResponse{
		ProtocolVersion: req.ProtocolVersion,
		ExtensionInfo: subprocess.InitializeExtensionInfo{
			Name:    req.Extension.Name,
			Version: req.Extension.Version,
		},
		AcceptedCapabilities: subprocess.AcceptedCapabilities{
			Provides: append([]string(nil), req.Capabilities.Provides...),
			Actions:  append([]extensionprotocol.HostAPIMethod(nil), req.Capabilities.GrantedActions...),
			Security: append([]string(nil), req.Capabilities.GrantedSecurity...),
		},
		ImplementedMethods:  implementedMethods,
		SupportedHookEvents: []string{string(hookspkg.HookSessionPostCreate)},
		Supports: subprocess.InitializeSupports{
			HealthCheck: true,
		},
	}
}

type daemonInitializeMarker struct {
	Request  subprocess.InitializeRequest  `json:"request"`
	Response subprocess.InitializeResponse `json:"response"`
}

type daemonDeliveryMarker struct {
	PID     int                       `json:"pid"`
	Request bridgepkg.DeliveryRequest `json:"request"`
}

func appendMarkerLine(path string, line string) (err error) {
	target := strings.TrimSpace(path)
	if target == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("append marker line: create marker directory: %w", err)
	}
	file, err := os.OpenFile(target, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("append marker line: open marker file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("append marker line: close marker file: %w", closeErr)
		}
	}()
	_, err = fmt.Fprintf(file, "%s\n", strings.TrimSpace(line))
	if err != nil {
		return fmt.Errorf("append marker line: write marker file: %w", err)
	}
	return nil
}

func markerLineCount(path string) int {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		// The helper treats missing or unreadable markers as an empty state file.
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(payload), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		count++
	}
	return count
}
