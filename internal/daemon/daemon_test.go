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
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
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

	if err := d.boot(testContext(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testContext(t)); err != nil {
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
	if got, want := signals, []string{"terminated:1001", "killed:1001"}; !equalStrings(got, want) {
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

	if err := d.cleanupOrphans(testContext(t), 444); err != nil {
		t.Fatalf("cleanupOrphans() error = %v", err)
	}
	if got, want := signals, []string{"terminated:1001"}; !equalStrings(got, want) {
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
		firstBoot <- d.boot(testContext(t))
	}()

	<-loadStarted
	if err := d.boot(testContext(t)); err == nil || !strings.Contains(err.Error(), "already booted") {
		t.Fatalf("concurrent boot error = %v, want already booted", err)
	}

	close(releaseLoad)
	if err := <-firstBoot; err != nil {
		t.Fatalf("first boot error = %v", err)
	}
	if err := d.Shutdown(testContext(t)); err != nil {
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

	if err := d.Shutdown(testContext(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	want := []string{"session:sess-a", "session:sess-b", "http", "uds", "db", "lock", "logger"}
	if !equalStrings(events, want) {
		t.Fatalf("Shutdown() order = %#v, want %#v", events, want)
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

	if err := d.boot(testContext(t)); err == nil || !strings.Contains(err.Error(), "uds boom") {
		t.Fatalf("boot() error = %v, want uds boom", err)
	}

	want := []string{"http", "db", "lock", "logger"}
	if !equalStrings(events, want) {
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

	if err := d.boot(testContext(t)); err == nil || !strings.Contains(err.Error(), "daemon info") {
		t.Fatalf("boot() error = %v, want daemon info failure", err)
	}

	want := []string{"uds", "http", "db", "lock", "logger"}
	if !equalStrings(events, want) {
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

func TestStopSessionsIgnoresNotFoundAndHandlesNilManager(t *testing.T) {
	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.stopSessions(testContext(t), nil); err != nil {
		t.Fatalf("stopSessions(nil) error = %v", err)
	}

	manager := &fakeSessionManager{
		infos: []*session.SessionInfo{{ID: "sess-a"}},
		stopErr: func(id string) error {
			return fmt.Errorf("%w: %s", session.ErrSessionNotFound, id)
		},
	}
	if err := d.stopSessions(testContext(t), manager); err != nil {
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
	if err := d.cleanupOrphans(testContext(t), 1); err == nil || !strings.Contains(err.Error(), "ps failed") {
		t.Fatalf("cleanupOrphans(list failure) error = %v, want ps failed", err)
	}

	d.listProcesses = func(context.Context) ([]processInfo, error) {
		return []processInfo{{PID: 10, PPID: 5}}, nil
	}
	d.signalProcess = func(int, syscall.Signal) error {
		return errors.New("signal failed")
	}
	if err := d.cleanupOrphans(testContext(t), 5); err == nil || !strings.Contains(err.Error(), "signal failed") {
		t.Fatalf("cleanupOrphans(signal failure) error = %v, want signal failed", err)
	}
	if err := d.cleanupOrphans(testContext(t), 0); err != nil {
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

	if err := d.Boundaries(testContext(t)); err != nil {
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

	if err := d.Boundaries(testContext(t)); err == nil {
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
	if err := d.Boundaries(testContext(t)); err != nil {
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

func TestNormalizeAbsolutePathVariants(t *testing.T) {
	if got, err := normalizeAbsolutePath(""); err != nil || got != "" {
		t.Fatalf("normalizeAbsolutePath(blank) = %q, %v, want empty nil", got, err)
	}

	got, err := normalizeAbsolutePath("daemon.sock")
	if err != nil {
		t.Fatalf("normalizeAbsolutePath(relative) error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("normalizeAbsolutePath(relative) = %q, want absolute path", got)
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

func TestNotifierFanoutDispatchesEvents(t *testing.T) {
	first := &recordingNotifier{}
	second := &recordingNotifier{}
	fanout := notifierFanout{notifiers: []session.Notifier{first, second}}

	fanout.OnSessionCreated(testContext(t), &session.Session{ID: "sess-1"})
	fanout.OnSessionStopped(testContext(t), &session.Session{ID: "sess-2"})
	fanout.OnAgentEvent(testContext(t), "sess-3", acp.AgentEvent{Type: "message"})

	if got, want := first.events, []string{"created", "stopped", "agent"}; !equalStrings(got, want) {
		t.Fatalf("first notifier events = %#v, want %#v", got, want)
	}
	if got, want := second.events, []string{"created", "stopped", "agent"}; !equalStrings(got, want) {
		t.Fatalf("second notifier events = %#v, want %#v", got, want)
	}
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

			if err := d.boot(testContext(t)); err != nil {
				t.Fatalf("boot() error = %v", err)
			}
			t.Cleanup(func() {
				if err := d.Shutdown(testContext(t)); err != nil {
					t.Fatalf("Shutdown() error = %v", err)
				}
			})

			if capturedDeps.PromptAssembler == nil {
				t.Fatal("boot() did not inject the composed prompt assembler")
			}
			if got := d.memoryStore != nil; got != tc.wantMemory {
				t.Fatalf("memory store initialized = %t, want %t", got, tc.wantMemory)
			}
			if got := d.skillsRegistry != nil; got != tc.wantSkills {
				t.Fatalf("skills registry initialized = %t, want %t", got, tc.wantSkills)
			}

			workspace := filepath.Join(t.TempDir(), "workspace")
			writeDaemonMemoryIndex(t, cfg.Memory.GlobalDir, workspace)

			prompt, err := capturedDeps.PromptAssembler.Assemble(context.Background(), testPromptAgent("Base prompt."), workspace)
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

	if err := d.boot(testContext(t)); err != nil {
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

	if err := d.Shutdown(testContext(t)); err != nil {
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

	if err := d.boot(testContext(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	skillsDone = d.skillsDone
	if skillsDone == nil {
		t.Fatal("boot() did not start the skills watcher")
	}

	if err := d.Shutdown(testContext(t)); err != nil {
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

func TestUserAgentsSkillsDirFallsBackToUserHome(t *testing.T) {
	t.Parallel()

	d, err := New(WithLogger(discardLogger()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.getenv = func(string) string { return "" }

	got, err := d.userAgentsSkillsDir()
	if err != nil {
		t.Fatalf("userAgentsSkillsDir() error = %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error = %v", err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", home, err)
	}
	if want := filepath.Join(absHome, ".agents", "skills"); got != want {
		t.Fatalf("userAgentsSkillsDir() = %q, want %q", got, want)
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
				return d.dreamService == nil && d.dreamCheckCh == nil
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
	d.newDreamService = func(opts ...memory.Option) dreamService {
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
		return d.dreamCheckCh != nil
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
	var notifier session.Notifier

	d := newTestDaemon(t, homePaths, cfg)
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		notifier = deps.Notifier
		return sessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.newDreamService = func(opts ...memory.Option) dreamService {
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
		return d.dreamCheckCh != nil
	})
	if notifier == nil {
		t.Fatal("session manager notifier = nil")
	}

	notifier.OnSessionStopped(context.Background(), &session.Session{ID: "sess-user", Workspace: workspace, Type: session.SessionTypeUser})
	waitForCondition(t, "dream run from session stop", func() bool {
		return dream.runCount() == 1
	})
	waitForCondition(t, "dream session workspace propagated", func() bool {
		return sessions.createCount() == 1
	})
	if got := sessions.createCall(0).Workspace; got != workspace {
		t.Fatalf("Create() workspace = %q, want %q", got, workspace)
	}

	notifier.OnSessionStopped(context.Background(), &session.Session{ID: "sess-dream", Type: session.SessionTypeDream})
	time.Sleep(20 * time.Millisecond)
	if got := dream.runCount(); got != 1 {
		t.Fatalf("dream run count after dream-session stop = %d, want 1", got)
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestDreamSpawnerCreatesDreamSession(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	sessions := &fakeSessionManager{}
	workspace := filepath.Join(t.TempDir(), "workspace")

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

	if err := d.boot(testContext(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testContext(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.dreamSpawner == nil {
		t.Fatal("boot() did not configure the dream spawner")
	}

	if err := d.dreamSpawner(testContext(t), "memory-consolidation", "summarize recent sessions", workspace); err != nil {
		t.Fatalf("dream spawner error = %v", err)
	}
	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create() calls = %d, want 1", got)
	}
	if got := sessions.createCall(0).Type; got != session.SessionTypeDream {
		t.Fatalf("Create() session type = %q, want %q", got, session.SessionTypeDream)
	}
	if got := sessions.createCall(0).AgentName; got != cfg.Memory.Dream.Agent {
		t.Fatalf("Create() agent = %q, want %q", got, cfg.Memory.Dream.Agent)
	}
	if got := sessions.createCall(0).Workspace; got != workspace {
		t.Fatalf("Create() workspace = %q, want %q", got, workspace)
	}
	if got := sessions.promptCount(); got != 1 || sessions.promptCall(0).msg != "summarize recent sessions" {
		t.Fatalf("Prompt() calls = %d, want one prompt payload", got)
	}
	if got := sessions.stopCount(); got != 1 || sessions.stopCall(0) != "dream-1" {
		t.Fatalf("Stop() calls = %d, want stop for created dream session", got)
	}
}

func TestDreamSpawnerDerivesRecentWorkspacesFromSessions(t *testing.T) {
	t.Parallel()

	homePaths := testHomePaths(t)
	cfg := testConfig(t, homePaths)
	workspaceA := filepath.Join(t.TempDir(), "workspace-a")
	workspaceB := filepath.Join(t.TempDir(), "workspace-b")
	sessions := &fakeSessionManager{
		infos: []*session.SessionInfo{
			{ID: "dream-old", Workspace: workspaceA, Type: session.SessionTypeDream, UpdatedAt: time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)},
			{ID: "user-old", Workspace: workspaceA, Type: session.SessionTypeUser, UpdatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)},
			{ID: "user-new", Workspace: workspaceB, Type: session.SessionTypeUser, UpdatedAt: time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)},
			{ID: "user-dup", Workspace: workspaceA, Type: session.SessionTypeUser, UpdatedAt: time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC)},
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

	if err := d.boot(testContext(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testContext(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	globalMemoryDir := cfg.Memory.GlobalDir
	if strings.TrimSpace(globalMemoryDir) == "" {
		globalMemoryDir = homePaths.MemoryDir
	}
	lockPath := memory.ConsolidationLockPath(globalMemoryDir)
	prior := time.Date(2026, 4, 4, 8, 0, 0, 0, time.UTC)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(lock dir) error = %v", err)
	}
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatalf("os.WriteFile(lock) error = %v", err)
	}
	if err := os.Chtimes(lockPath, prior, prior); err != nil {
		t.Fatalf("os.Chtimes(lock) error = %v", err)
	}

	if err := d.dreamSpawner(testContext(t), "memory-consolidation", "summarize recent sessions", ""); err != nil {
		t.Fatalf("dream spawner error = %v", err)
	}

	if got := sessions.createCount(); got != 2 {
		t.Fatalf("Create() calls = %d, want 2", got)
	}
	if got := sessions.createCall(0).Workspace; got != workspaceB {
		t.Fatalf("Create() workspace[0] = %q, want %q", got, workspaceB)
	}
	if got := sessions.createCall(1).Workspace; got != workspaceA {
		t.Fatalf("Create() workspace[1] = %q, want %q", got, workspaceA)
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
	processes, err := listProcesses(testContext(t))
	if err != nil {
		t.Fatalf("listProcesses() error = %v", err)
	}
	if len(processes) == 0 {
		t.Fatal("listProcesses() returned no processes")
	}

	if err := signalProcess(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Fatalf("signalProcess(self, 0) error = %v", err)
	}
	if err := signalProcess(0, syscall.SIGTERM); err == nil {
		t.Fatal("signalProcess(invalid pid) error = nil, want non-nil")
	}
}

func TestProcessAliveAndRuntimeLoggerHelpers(t *testing.T) {
	if processAlive(0) {
		t.Fatal("processAlive(0) = true, want false")
	}
	if !processAlive(os.Getpid()) {
		t.Fatal("processAlive(self) = false, want true")
	}
	if processAlive(999999) && runtime.GOOS != "windows" {
		t.Fatal("processAlive(999999) = true, want false")
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

func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
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

func equalStrings(got []string, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
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
	return &session.Session{
		ID:        sessionID,
		AgentName: opts.AgentName,
		Workspace: opts.Workspace,
		Type:      opts.Type,
		State:     session.StateActive,
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

func (f *fakeSessionManager) promptCall(index int) struct {
	id  string
	msg string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.promptCalls[index]
}

func (f *fakeSessionManager) stopCall(index int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopCalls[index]
}

func (f *fakeSessionManager) promptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.promptCalls)
}

func (f *fakeSessionManager) stopCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.stopCalls)
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

func (f *fakeObserver) OnAgentEvent(context.Context, string, acp.AgentEvent) {}

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

func (n *recordingNotifier) OnAgentEvent(context.Context, string, acp.AgentEvent) {
	n.events = append(n.events, "agent")
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
