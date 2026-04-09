package daemon

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/gofrs/flock"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
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
	d.sessions = &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}, {ID: "sess-b"}},
		onStop: func(id string) {
			events = append(events, "session:"+id)
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

	want := []string{"session:sess-a", "session:sess-b", "hooks", "http", "uds", "db", "lock", "logger"}
	if !testutil.EqualStringSlices(events, want) {
		t.Fatalf("Shutdown() order = %#v, want %#v", events, want)
	}
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
		WithSignalChannel(signalCh),
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
		t.Fatal("WithSignalChannel() did not apply")
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
		t.Fatal("signalSource() channel = nil")
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
	var dispatcher session.HookDispatcher

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
	if dispatcher == nil {
		t.Fatal("session manager hook dispatcher = nil")
	}

	resolved := resolveDaemonWorkspace(t, d.workspaceResolver, workspace)
	if _, err := dispatcher.DispatchSessionPostStop(context.Background(), hookspkg.SessionPostStopPayload{
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

	if _, err := dispatcher.DispatchSessionPostStop(context.Background(), hookspkg.SessionPostStopPayload{
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
	mu          sync.Mutex
	infos       []*session.SessionInfo
	onStop      func(string)
	stopErr     func(string) error
	createCalls []session.CreateOpts
	promptCalls []struct {
		id  string
		msg string
	}
	stopCalls []string
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

func (f *fakeSessionManager) Resume(context.Context, string) (*session.Session, error) {
	return nil, nil
}

func (f *fakeSessionManager) Prompt(_ context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	f.mu.Lock()
	f.promptCalls = append(f.promptCalls, struct {
		id  string
		msg string
	}{id: id, msg: msg})
	f.mu.Unlock()
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

type fakeObserver struct {
	reconciled bool
	result     store.ReconcileResult
	err        error
}

func (f *fakeObserver) QueryEvents(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
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
