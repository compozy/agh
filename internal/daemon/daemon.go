package daemon

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/httpapi"
	aghlogger "github.com/pedronauck/agh/internal/logger"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/udsapi"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	defaultShutdownTimeout = 10 * time.Second
	moduleImportPath       = "github.com/pedronauck/agh"
	orphanCleanupGraceWait = 2 * time.Second
	orphanCleanupPollWait  = 100 * time.Millisecond
)

// Option customizes daemon construction.
type Option func(*Daemon)

// ConfigLoader resolves the daemon-level runtime configuration.
type ConfigLoader func() (aghconfig.Config, error)

// SessionManager is the session lifecycle surface consumed by daemon/.
type SessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	List() []*session.SessionInfo
	ListAll(ctx context.Context) ([]*session.SessionInfo, error)
	Status(ctx context.Context, id string) (*session.SessionInfo, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error)
	Transcript(ctx context.Context, id string) ([]session.TranscriptMessage, error)
	Stop(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error
}

// Observer is the observability surface consumed by daemon/.
type Observer interface {
	session.Notifier
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	Health(ctx context.Context) (observe.Health, error)
	Reconcile(ctx context.Context) (store.ReconcileResult, error)
}

// Registry is the shared global database surface consumed by daemon/.
type Registry interface {
	store.SessionRegistry
	workspacepkg.WorkspaceStore
	Path() string
	Close(ctx context.Context) error
}

// Server is a daemon-owned runtime component with explicit start and shutdown phases.
type Server interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// RuntimeDeps captures the composition-root dependencies available to server factories.
type RuntimeDeps struct {
	Config            aghconfig.Config
	HomePaths         aghconfig.HomePaths
	Logger            *slog.Logger
	Sessions          SessionManager
	Observer          Observer
	Registry          Registry
	MemoryStore       *memory.Store
	WorkspaceResolver workspacepkg.WorkspaceResolver
	WorkspaceService  *workspacepkg.Resolver
	DreamTrigger      DreamTrigger
	StartedAt         time.Time
}

// ServerFactory constructs runtime components such as HTTP and UDS servers.
type ServerFactory func(ctx context.Context, deps RuntimeDeps) (Server, error)

// DreamTrigger exposes consolidation controls and health state to transport layers.
type DreamTrigger interface {
	Trigger(ctx context.Context, workspace string) (bool, string, error)
	LastConsolidatedAt() (time.Time, error)
	Enabled() bool
}

type registryOpener func(ctx context.Context, path string) (Registry, error)
type sessionManagerFactory func(ctx context.Context, deps SessionManagerDeps) (SessionManager, error)
type observerFactory func(ctx context.Context, deps RuntimeDeps) (Observer, error)
type dreamServiceFactory func(opts ...memory.Option) dreamService

type dreamService interface {
	ShouldRun() (bool, error)
	Run(ctx context.Context, spawn memory.SessionSpawner, workspace string) error
}

type runtimeDreamTrigger struct {
	enabled            bool
	service            dreamService
	spawner            memory.SessionSpawner
	lastConsolidatedAt func() (time.Time, error)
}

func (t runtimeDreamTrigger) Trigger(ctx context.Context, workspace string) (bool, string, error) {
	if !t.Enabled() || t.service == nil || t.spawner == nil {
		return false, "dream consolidation is disabled", nil
	}

	shouldRun, err := t.service.ShouldRun()
	if err != nil {
		return false, "", err
	}
	if !shouldRun {
		return false, "dream consolidation gates are not satisfied", nil
	}
	if err := t.service.Run(ctx, t.spawner, strings.TrimSpace(workspace)); err != nil {
		if errors.Is(err, memory.ErrLockUnavailable) {
			return false, "dream consolidation is already running", nil
		}
		return false, "", err
	}

	return true, "", nil
}

func (t runtimeDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	if t.lastConsolidatedAt == nil {
		return time.Time{}, nil
	}
	return t.lastConsolidatedAt()
}

func (t runtimeDreamTrigger) Enabled() bool {
	return t.enabled
}

// SessionManagerDeps captures the composition-root dependencies needed to create a session manager.
type SessionManagerDeps struct {
	HomePaths         aghconfig.HomePaths
	Logger            *slog.Logger
	Notifier          session.Notifier
	PromptAssembler   session.PromptAssembler
	WorkspaceResolver workspacepkg.WorkspaceResolver
}

type processInfo struct {
	PID  int
	PPID int
}

type dreamCheckRequest struct {
	reason       string
	workspaceRef string
}

type notifierFanout struct {
	notifiers        []session.Notifier
	onSessionStopped func(context.Context, *session.Session)
}

// Daemon is the sole AGH composition root.
type Daemon struct {
	mu sync.Mutex

	homePaths         aghconfig.HomePaths
	loadConfig        ConfigLoader
	logger            *slog.Logger
	closeLogger       func() error
	now               func() time.Time
	pid               func() int
	acquireLock       func(path string, pid int) (*Lock, error)
	openRegistry      registryOpener
	newSessionManager sessionManagerFactory
	newDreamService   dreamServiceFactory
	newObserver       observerFactory
	httpFactory       ServerFactory
	udsFactory        ServerFactory
	listProcesses     func(context.Context) ([]processInfo, error)
	signalProcess     func(int, syscall.Signal) error
	processAlive      func(int) bool
	signalCh          <-chan os.Signal
	verifyBoundaries  bool
	boundaryRoot      string
	getenv            func(string) string
	readyCh           chan struct{}
	readyClosed       bool
	booting           bool
	orphanGraceWait   time.Duration
	orphanPollWait    time.Duration
	config            aghconfig.Config
	startedAt         time.Time
	info              Info
	lock              *Lock
	registry          Registry
	memoryStore       *memory.Store
	sessions          SessionManager
	observer          Observer
	httpServer        Server
	udsServer         Server
	dreamService      dreamService
	dreamSpawner      memory.SessionSpawner
	dreamCheckCh      chan dreamCheckRequest
	dreamCancel       context.CancelFunc
	dreamWG           sync.WaitGroup
	workspaceResolver workspacepkg.WorkspaceResolver
	skillsRegistry    *skills.Registry
	skillsCancel      context.CancelFunc
	skillsDone        chan struct{}
}

// WithHomePaths overrides the resolved AGH home layout.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(d *Daemon) {
		d.homePaths = homePaths
	}
}

// WithConfig overrides daemon-level configuration loading.
func WithConfig(cfg aghconfig.Config) Option {
	return func(d *Daemon) {
		d.loadConfig = func() (aghconfig.Config, error) {
			return cfg, nil
		}
	}
}

// WithConfigLoader overrides daemon-level configuration loading.
func WithConfigLoader(loader ConfigLoader) Option {
	return func(d *Daemon) {
		d.loadConfig = loader
	}
}

// WithLogger injects the daemon logger.
func WithLogger(logger *slog.Logger) Option {
	return func(d *Daemon) {
		d.logger = logger
		d.closeLogger = func() error { return nil }
	}
}

// WithNow overrides the daemon clock, mainly for tests.
func WithNow(now func() time.Time) Option {
	return func(d *Daemon) {
		d.now = now
	}
}

// WithHTTPServerFactory overrides HTTP server construction.
func WithHTTPServerFactory(factory ServerFactory) Option {
	return func(d *Daemon) {
		d.httpFactory = factory
	}
}

// WithUDSServerFactory overrides UDS server construction.
func WithUDSServerFactory(factory ServerFactory) Option {
	return func(d *Daemon) {
		d.udsFactory = factory
	}
}

// WithSignalChannel overrides OS signal delivery, mainly for tests.
func WithSignalChannel(ch <-chan os.Signal) Option {
	return func(d *Daemon) {
		d.signalCh = ch
	}
}

// WithBoundaryVerification enables best-effort import boundary verification on boot.
func WithBoundaryVerification(enabled bool) Option {
	return func(d *Daemon) {
		d.verifyBoundaries = enabled
	}
}

// New constructs the daemon composition root.
func New(opts ...Option) (*Daemon, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("daemon: resolve home paths: %w", err)
	}

	d := &Daemon{
		homePaths: homePaths,
		readyCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	if d.now == nil {
		d.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if d.pid == nil {
		d.pid = os.Getpid
	}
	if d.acquireLock == nil {
		d.acquireLock = AcquireLock
	}
	if d.openRegistry == nil {
		d.openRegistry = func(ctx context.Context, path string) (Registry, error) {
			return store.OpenGlobalDB(ctx, path)
		}
	}
	if d.newSessionManager == nil {
		d.newSessionManager = func(ctx context.Context, deps SessionManagerDeps) (SessionManager, error) {
			return session.NewManager(
				session.WithHomePaths(deps.HomePaths),
				session.WithLogger(deps.Logger),
				session.WithNotifier(deps.Notifier),
				session.WithPromptAssembler(deps.PromptAssembler),
				session.WithWorkspaceResolver(deps.WorkspaceResolver),
			)
		}
	}
	if d.newDreamService == nil {
		d.newDreamService = func(opts ...memory.Option) dreamService {
			return memory.NewService(opts...)
		}
	}
	if d.newObserver == nil {
		d.newObserver = func(ctx context.Context, deps RuntimeDeps) (Observer, error) {
			source, ok := deps.Sessions.(observe.SessionSource)
			if !ok {
				return nil, errors.New("daemon: session manager does not implement observe session source")
			}
			return observe.New(
				ctx,
				observe.WithRegistry(deps.Registry),
				observe.WithHomePaths(deps.HomePaths),
				observe.WithSessionSource(source),
				observe.WithWorkspaceResolver(deps.WorkspaceResolver),
				observe.WithLogger(deps.Logger),
				observe.WithStartTime(deps.StartedAt),
			)
		}
	}
	if d.httpFactory == nil {
		d.httpFactory = func(_ context.Context, deps RuntimeDeps) (Server, error) {
			return httpapi.New(
				httpapi.WithHomePaths(deps.HomePaths),
				httpapi.WithConfig(deps.Config),
				httpapi.WithLogger(deps.Logger),
				httpapi.WithStartedAt(deps.StartedAt),
				httpapi.WithSessionManager(deps.Sessions),
				httpapi.WithObserver(deps.Observer),
				httpapi.WithWorkspaceResolver(deps.WorkspaceService),
				httpapi.WithMemoryStore(deps.MemoryStore),
				httpapi.WithDreamTrigger(deps.DreamTrigger),
			)
		}
	}
	if d.udsFactory == nil {
		d.udsFactory = func(_ context.Context, deps RuntimeDeps) (Server, error) {
			return udsapi.New(
				udsapi.WithHomePaths(deps.HomePaths),
				udsapi.WithConfig(deps.Config),
				udsapi.WithLogger(deps.Logger),
				udsapi.WithStartedAt(deps.StartedAt),
				udsapi.WithSessionManager(deps.Sessions),
				udsapi.WithObserver(deps.Observer),
				udsapi.WithWorkspaceResolver(deps.WorkspaceService),
				udsapi.WithMemoryStore(deps.MemoryStore),
				udsapi.WithDreamTrigger(deps.DreamTrigger),
			)
		}
	}
	if d.listProcesses == nil {
		d.listProcesses = listProcesses
	}
	if d.signalProcess == nil {
		d.signalProcess = signalProcess
	}
	if d.processAlive == nil {
		d.processAlive = processAlive
	}
	if d.getenv == nil {
		d.getenv = os.Getenv
	}
	if d.closeLogger == nil {
		d.closeLogger = func() error { return nil }
	}
	if d.loadConfig == nil {
		d.loadConfig = func() (aghconfig.Config, error) {
			return loadConfigFromHome(d.homePaths)
		}
	}
	if d.orphanGraceWait <= 0 {
		d.orphanGraceWait = orphanCleanupGraceWait
	}
	if d.orphanPollWait <= 0 {
		d.orphanPollWait = orphanCleanupPollWait
	}

	return d, nil
}

// Run boots the daemon, blocks until signal or context cancellation, then performs graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("daemon: run context is required")
	}
	if err := d.boot(ctx); err != nil {
		return err
	}
	d.startDreamLoop(ctx)

	sigCh, stopSignals := d.signalSource()
	defer stopSignals()

	select {
	case <-ctx.Done():
	case sig, ok := <-sigCh:
		if ok && sig != nil {
			d.runtimeLogger().Info("daemon: received shutdown signal", "signal", sig.String())
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	return d.Shutdown(shutdownCtx)
}

// Shutdown gracefully tears down the daemon in the required order.
func (d *Daemon) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	d.mu.Lock()
	sessions := d.sessions
	httpServer := d.httpServer
	udsServer := d.udsServer
	registry := d.registry
	lock := d.lock
	closeLogger := d.closeLogger
	infoPath := d.homePaths.DaemonInfo
	dreamCancel := d.dreamCancel
	skillsCancel := d.skillsCancel
	skillsDone := d.skillsDone

	d.sessions = nil
	d.httpServer = nil
	d.udsServer = nil
	d.observer = nil
	d.registry = nil
	d.memoryStore = nil
	d.skillsRegistry = nil
	d.lock = nil
	d.booting = false
	d.info = Info{}
	d.startedAt = time.Time{}
	d.closeLogger = func() error { return nil }
	d.dreamService = nil
	d.dreamSpawner = nil
	d.dreamCheckCh = nil
	d.dreamCancel = nil
	d.workspaceResolver = nil
	d.skillsCancel = nil
	d.skillsDone = nil
	d.mu.Unlock()

	var errs []error
	if dreamCancel != nil {
		dreamCancel()
		d.dreamWG.Wait()
	}
	stopSkillsWatcher(skillsCancel, skillsDone)
	if err := d.stopSessions(ctx, sessions); err != nil {
		errs = append(errs, err)
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: shutdown http server: %w", err))
		}
	}
	if udsServer != nil {
		if err := udsServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: shutdown uds server: %w", err))
		}
	}
	if err := RemoveInfo(infoPath); err != nil {
		errs = append(errs, err)
	}
	if registry != nil {
		if err := registry.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: close global database: %w", err))
		}
	}
	if lock != nil {
		if err := lock.Release(); err != nil {
			errs = append(errs, err)
		}
	}
	if closeLogger != nil {
		if err := closeLogger(); err != nil {
			errs = append(errs, fmt.Errorf("daemon: close logger: %w", err))
		}
	}

	return errors.Join(errs...)
}

// Boundaries performs a best-effort import boundary verification for local source checkouts.
func (d *Daemon) Boundaries(context.Context) error {
	root := strings.TrimSpace(d.boundaryRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("daemon: resolve working directory for boundary check: %w", err)
		}
		root = cwd
	}

	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("daemon: stat go.mod for boundary check: %w", err)
	}

	violations, err := verifyImportBoundaries(root)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}

	return errors.Join(violations...)
}

func (d *Daemon) boot(ctx context.Context) (err error) {
	if ctx == nil {
		return errors.New("daemon: boot context is required")
	}

	d.mu.Lock()
	if d.booting || d.lock != nil || d.registry != nil || d.sessions != nil || d.observer != nil {
		d.mu.Unlock()
		return errors.New("daemon: already booted")
	}
	d.booting = true
	d.mu.Unlock()
	defer func() {
		if err == nil {
			return
		}
		d.mu.Lock()
		d.booting = false
		d.mu.Unlock()
	}()

	cfg, err := d.loadConfig()
	if err != nil {
		return err
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("daemon: validate config: %w", err)
	}
	if err := aghconfig.EnsureHomeLayout(d.homePaths); err != nil {
		return fmt.Errorf("daemon: ensure home layout: %w", err)
	}

	logger := d.logger
	closeLogger := d.closeLogger
	if logger == nil {
		logger, closeLogger, err = aghlogger.New(
			aghlogger.WithLevel(cfg.Log.Level),
			aghlogger.WithFile(d.homePaths.LogFile),
		)
		if err != nil {
			return fmt.Errorf("daemon: create logger: %w", err)
		}
	}
	if closeLogger == nil {
		closeLogger = func() error { return nil }
	}

	var (
		memoryStore      *memory.Store
		skillsRegistry   *skills.Registry
		dreamSvc         dreamService
		globalMemoryDir  string
		skillsCancel     context.CancelFunc
		skillsDone       chan struct{}
		prependProviders []session.PromptProvider
		appendProviders  []session.PromptProvider
	)
	if cfg.Memory.Enabled {
		globalMemoryDir = strings.TrimSpace(cfg.Memory.GlobalDir)
		if globalMemoryDir == "" {
			globalMemoryDir = d.homePaths.MemoryDir
		}
		memoryStore = memory.NewStore(globalMemoryDir)
		if err := memoryStore.EnsureDirs(); err != nil {
			return fmt.Errorf("daemon: ensure memory store directories: %w", err)
		}
		prependProviders = append(prependProviders, memory.NewAssembler(memoryStore))
	}

	cleanupFns := make([]func(context.Context) error, 0, 8)
	defer func() {
		if err == nil {
			return
		}
		var cleanupErrs []error
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			if cleanupErr := cleanupFns[i](context.Background()); cleanupErr != nil {
				cleanupErrs = append(cleanupErrs, cleanupErr)
			}
		}
		err = errors.Join(err, errors.Join(cleanupErrs...))
	}()
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return closeLogger()
	})

	if cfg.Skills.Enabled {
		skillsCfg, err := d.skillsRegistryConfig(cfg)
		if err != nil {
			return err
		}

		skillsRegistry = skills.NewRegistry(skillsCfg, skills.WithLogger(logger))
		if err := skillsRegistry.LoadAll(ctx); err != nil {
			return fmt.Errorf("daemon: load skills registry: %w", err)
		}

		skillsCancel, skillsDone = startSkillsWatcher(ctx, skillsRegistry, cfg.Skills.PollInterval)
		cleanupFns = append(cleanupFns, func(context.Context) error {
			stopSkillsWatcher(skillsCancel, skillsDone)
			return nil
		})
		appendProviders = append(appendProviders, skills.NewCatalogProvider(skillsRegistry))
	}

	promptAssembler := NewComposedAssembler(
		WithPrependPromptProviders(prependProviders...),
		WithAppendPromptProviders(appendProviders...),
	)

	pid := d.pid()
	lock, err := d.acquireLock(d.homePaths.DaemonLock, pid)
	if err != nil {
		return err
	}
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return lock.Release()
	})

	stalePID := lock.StalePID()
	if stalePID == 0 {
		existingInfo, readErr := ReadInfo(d.homePaths.DaemonInfo)
		switch {
		case readErr == nil && existingInfo.PID > 0 && existingInfo.PID != pid && !d.processAlive(existingInfo.PID):
			stalePID = existingInfo.PID
		case readErr != nil && !errors.Is(readErr, os.ErrNotExist):
			logger.Warn("daemon: read stale daemon info failed", "path", d.homePaths.DaemonInfo, "error", readErr)
		}
	}
	if stalePID > 0 {
		if cleanupErr := d.cleanupOrphans(ctx, stalePID); cleanupErr != nil {
			logger.Warn("daemon: cleanup orphan processes failed", "stale_pid", stalePID, "error", cleanupErr)
		}
	}

	if err := removeStaleSocket(cfg.Daemon.Socket); err != nil {
		return err
	}

	registry, err := d.openRegistry(ctx, d.homePaths.DatabaseFile)
	if err != nil {
		return fmt.Errorf("daemon: open global database %q: %w", d.homePaths.DatabaseFile, err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return registry.Close(ctx)
	})

	workspaceResolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(d.homePaths),
		workspacepkg.WithLogger(logger),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(d.homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		return fmt.Errorf("daemon: create workspace resolver: %w", err)
	}

	if cfg.Memory.Enabled && cfg.Memory.Dream.Enabled {
		dreamSvc = d.newDreamService(
			memory.WithMemoryStore(memoryStore),
			memory.WithSessionsDir(d.homePaths.SessionsDir),
			memory.WithMinHours(cfg.Memory.Dream.MinHours),
			memory.WithMinSessions(cfg.Memory.Dream.MinSessions),
			memory.WithLogger(logger),
			memory.WithWorkspaceResolver(workspaceResolver),
		)
	}

	startedAt := d.now().UTC()
	fanout := notifierFanout{}
	sessions, err := d.newSessionManager(ctx, SessionManagerDeps{
		HomePaths:         d.homePaths,
		Logger:            logger,
		Notifier:          &fanout,
		PromptAssembler:   promptAssembler,
		WorkspaceResolver: workspaceResolver,
	})
	if err != nil {
		return fmt.Errorf("daemon: create session manager: %w", err)
	}

	dreamSpawner := d.makeDreamSpawner(sessions, workspaceResolver, cfg, globalMemoryDir)
	var dreamTrigger DreamTrigger
	if dreamSvc != nil {
		lockPath := memory.ConsolidationLockPath(globalMemoryDir)
		dreamTrigger = runtimeDreamTrigger{
			enabled: cfg.Memory.Dream.Enabled,
			service: dreamSvc,
			spawner: dreamSpawner,
			lastConsolidatedAt: func() (time.Time, error) {
				return memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
			},
		}
	}

	deps := RuntimeDeps{
		Config:            cfg,
		HomePaths:         d.homePaths,
		Logger:            logger,
		Sessions:          sessions,
		Registry:          registry,
		MemoryStore:       memoryStore,
		WorkspaceResolver: workspaceResolver,
		WorkspaceService:  workspaceResolver,
		DreamTrigger:      dreamTrigger,
		StartedAt:         startedAt,
	}

	observer, err := d.newObserver(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create observer: %w", err)
	}
	fanout.notifiers = append(fanout.notifiers, observer)
	deps.Observer = observer
	if dreamSvc != nil {
		fanout.onSessionStopped = func(_ context.Context, sess *session.Session) {
			info := sess.Info()
			if info == nil || info.Type == session.SessionTypeDream || strings.TrimSpace(info.WorkspaceID) == "" {
				return
			}
			d.enqueueDreamCheck("session_stop", info.WorkspaceID)
		}
	}

	httpServer, err := d.httpFactory(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create http server: %w", err)
	}
	if err := httpServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start http server: %w", err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	})

	udsServer, err := d.udsFactory(ctx, deps)
	if err != nil {
		return fmt.Errorf("daemon: create uds server: %w", err)
	}
	if err := udsServer.Start(ctx); err != nil {
		return fmt.Errorf("daemon: start uds server: %w", err)
	}
	cleanupFns = append(cleanupFns, func(ctx context.Context) error {
		return udsServer.Shutdown(ctx)
	})

	info := Info{
		PID:       pid,
		Port:      resolveDaemonPort(cfg.HTTP.Port, httpServer),
		StartedAt: startedAt,
	}
	if err := WriteInfo(d.homePaths.DaemonInfo, info); err != nil {
		return err
	}
	cleanupFns = append(cleanupFns, func(context.Context) error {
		return RemoveInfo(d.homePaths.DaemonInfo)
	})

	reconcileResult, err := observer.Reconcile(ctx)
	if err != nil {
		return fmt.Errorf("daemon: reconcile sessions: %w", err)
	}
	logger.Info(
		"daemon: boot reconciliation complete",
		"indexed_sessions", len(reconcileResult.Indexed),
		"orphaned_sessions", len(reconcileResult.Orphaned),
	)

	if d.shouldVerifyBoundaries() {
		if boundaryErr := d.Boundaries(ctx); boundaryErr != nil {
			logger.Warn("daemon: boundary verification warning", "error", boundaryErr)
		}
	}

	d.mu.Lock()
	d.config = cfg
	d.logger = logger
	d.closeLogger = closeLogger
	d.booting = false
	d.lock = lock
	d.registry = registry
	d.memoryStore = memoryStore
	d.sessions = sessions
	d.observer = observer
	d.httpServer = httpServer
	d.udsServer = udsServer
	d.dreamService = dreamSvc
	d.dreamSpawner = dreamSpawner
	d.workspaceResolver = workspaceResolver
	d.skillsRegistry = skillsRegistry
	d.skillsCancel = skillsCancel
	d.skillsDone = skillsDone
	d.startedAt = startedAt
	d.info = info
	if !d.readyClosed {
		close(d.readyCh)
		d.readyClosed = true
	}
	d.mu.Unlock()

	return nil
}

func (d *Daemon) skillsRegistryConfig(cfg aghconfig.Config) (skills.RegistryConfig, error) {
	userAgentsDir, err := d.userAgentsSkillsDir()
	if err != nil {
		return skills.RegistryConfig{}, err
	}

	return skills.RegistryConfig{
		BundledFS:      bundled.FS(),
		UserSkillsDir:  d.homePaths.SkillsDir,
		UserAgentsDir:  userAgentsDir,
		DisabledSkills: append([]string(nil), cfg.Skills.DisabledSkills...),
	}, nil
}

func (d *Daemon) userAgentsSkillsDir() (string, error) {
	if d.getenv != nil {
		if home := strings.TrimSpace(d.getenv("HOME")); home != "" {
			absHome, err := filepath.Abs(home)
			if err != nil {
				return "", fmt.Errorf("daemon: resolve HOME for user agent skills: %w", err)
			}
			return filepath.Join(absHome, ".agents", "skills"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("daemon: resolve user home for agent skills: %w", err)
	}

	absHome, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("daemon: resolve user home for agent skills: %w", err)
	}

	return filepath.Join(absHome, ".agents", "skills"), nil
}

func startSkillsWatcher(ctx context.Context, registry *skills.Registry, interval time.Duration) (context.CancelFunc, chan struct{}) {
	if registry == nil {
		return nil, nil
	}

	watcherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	watcher := skills.NewWatcher(registry, interval)
	go func() {
		defer close(done)
		watcher.Start(watcherCtx)
	}()
	return cancel, done
}

func stopSkillsWatcher(cancel context.CancelFunc, done <-chan struct{}) {
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (d *Daemon) startDreamLoop(parent context.Context) {
	d.mu.Lock()
	if d.dreamService == nil || d.dreamSpawner == nil || d.dreamCheckCh != nil {
		d.mu.Unlock()
		return
	}

	dreamCtx, cancel := context.WithCancel(parent)
	dreamCheckCh := make(chan dreamCheckRequest, 1)
	d.dreamCancel = cancel
	d.dreamCheckCh = dreamCheckCh
	service := d.dreamService
	spawner := d.dreamSpawner
	logger := d.logger
	interval := d.config.Memory.Dream.CheckInterval
	d.dreamWG.Add(1)
	d.mu.Unlock()
	if logger == nil {
		logger = slog.Default()
	}

	go func() {
		defer d.dreamWG.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-dreamCtx.Done():
				return
			case <-ticker.C:
				d.runDreamCheck(dreamCtx, logger, service, spawner, "ticker", "")
			case request := <-dreamCheckCh:
				d.runDreamCheck(dreamCtx, logger, service, spawner, request.reason, request.workspaceRef)
			}
		}
	}()
}

func (d *Daemon) enqueueDreamCheck(reason string, workspaceRef string) {
	d.mu.Lock()
	dreamCheckCh := d.dreamCheckCh
	d.mu.Unlock()

	if dreamCheckCh == nil {
		return
	}

	select {
	case dreamCheckCh <- dreamCheckRequest{
		reason:       strings.TrimSpace(reason),
		workspaceRef: strings.TrimSpace(workspaceRef),
	}:
	default:
		d.runtimeLogger().Debug("daemon: dream check already queued", "reason", reason, "workspace_ref", workspaceRef)
	}
}

func (d *Daemon) runDreamCheck(ctx context.Context, logger *slog.Logger, service dreamService, spawner memory.SessionSpawner, reason string, workspaceRef string) {
	if service == nil || spawner == nil {
		return
	}
	if logger == nil {
		logger = slog.Default()
	}

	logger.Debug("daemon: evaluating dream consolidation gates", "reason", reason, "workspace_ref", workspaceRef)
	shouldRun, err := service.ShouldRun()
	if err != nil {
		logger.Warn("daemon: dream gate evaluation failed", "reason", reason, "workspace_ref", workspaceRef, "error", err)
		return
	}
	if !shouldRun {
		logger.Debug("daemon: dream consolidation skipped", "reason", reason, "workspace_ref", workspaceRef)
		return
	}

	logger.Info("daemon: starting dream consolidation", "reason", reason, "workspace_ref", workspaceRef)
	if err := service.Run(ctx, spawner, workspaceRef); err != nil {
		if errors.Is(err, memory.ErrLockUnavailable) {
			logger.Debug("daemon: dream consolidation already running", "reason", reason, "workspace_ref", workspaceRef)
			return
		}
		logger.Warn("daemon: dream consolidation failed", "reason", reason, "workspace_ref", workspaceRef, "error", err)
		return
	}
	logger.Info("daemon: dream consolidation completed", "reason", reason, "workspace_ref", workspaceRef)
}

func (d *Daemon) makeDreamSpawner(sessions SessionManager, resolver workspacepkg.WorkspaceResolver, cfg aghconfig.Config, globalMemoryDir string) memory.SessionSpawner {
	if !cfg.Memory.Enabled || !cfg.Memory.Dream.Enabled || sessions == nil || resolver == nil {
		return nil
	}

	return func(ctx context.Context, goal, prompt, workspace string) error {
		workspaces, err := d.resolveDreamWorkspaces(ctx, sessions, resolver, globalMemoryDir, workspace)
		if err != nil {
			return err
		}

		for _, workspace := range workspaces {
			if err := spawnDreamSession(ctx, sessions, cfg.Memory.Dream.Agent, goal, prompt, workspace); err != nil {
				return err
			}
		}

		return nil
	}
}

func (d *Daemon) resolveDreamWorkspaces(ctx context.Context, sessions SessionManager, resolver workspacepkg.WorkspaceResolver, globalMemoryDir string, explicitWorkspace string) ([]string, error) {
	if resolver == nil {
		return nil, errors.New("daemon: workspace resolver is required for dream consolidation")
	}

	if workspaceRef := strings.TrimSpace(explicitWorkspace); workspaceRef != "" {
		resolvedRef, err := resolveDreamWorkspaceRef(ctx, resolver, workspaceRef)
		if err != nil {
			return nil, err
		}
		return []string{resolvedRef}, nil
	}

	lockPath := memory.ConsolidationLockPath(globalMemoryDir)
	lastConsolidatedAt, err := memory.NewConsolidationLock(lockPath).LastConsolidatedAt()
	if err != nil {
		return nil, fmt.Errorf("daemon: read dream consolidation lock: %w", err)
	}

	infos, err := sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list sessions for dream consolidation: %w", err)
	}

	type workspaceCandidate struct {
		id        string
		updatedAt time.Time
	}

	latestByWorkspace := make(map[string]time.Time, len(infos))
	for _, info := range infos {
		if info == nil || info.Type == session.SessionTypeDream {
			continue
		}

		workspaceID := strings.TrimSpace(info.WorkspaceID)
		if workspaceID == "" {
			continue
		}

		updatedAt := info.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = info.CreatedAt
		}
		if !lastConsolidatedAt.IsZero() && updatedAt.Before(lastConsolidatedAt) {
			continue
		}

		if latest, ok := latestByWorkspace[workspaceID]; !ok || updatedAt.After(latest) {
			latestByWorkspace[workspaceID] = updatedAt
		}
	}

	if len(latestByWorkspace) == 0 {
		return nil, errors.New("daemon: no recent workspaces available for dream consolidation")
	}

	candidates := make([]workspaceCandidate, 0, len(latestByWorkspace))
	for workspaceID, updatedAt := range latestByWorkspace {
		candidates = append(candidates, workspaceCandidate{id: workspaceID, updatedAt: updatedAt})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].updatedAt.Equal(candidates[j].updatedAt) {
			return candidates[i].id < candidates[j].id
		}
		return candidates[i].updatedAt.After(candidates[j].updatedAt)
	})

	workspaces := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		workspaces = append(workspaces, candidate.id)
	}
	return workspaces, nil
}

func resolveDreamWorkspaceRef(ctx context.Context, resolver workspacepkg.WorkspaceResolver, workspaceRef string) (string, error) {
	trimmedRef := strings.TrimSpace(workspaceRef)
	if trimmedRef == "" {
		return "", errors.New("daemon: dream workspace is required")
	}

	var (
		resolved workspacepkg.ResolvedWorkspace
		err      error
	)
	if isPathLikeWorkspaceRef(trimmedRef) {
		normalizedPath, normalizeErr := normalizeAbsolutePath(trimmedRef)
		if normalizeErr != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, normalizeErr)
		}
		resolved, err = resolver.ResolveOrRegister(ctx, normalizedPath)
		if err != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, err)
		}
	} else {
		resolved, err = resolver.Resolve(ctx, trimmedRef)
		if err != nil {
			return "", fmt.Errorf("daemon: resolve dream workspace %q: %w", workspaceRef, err)
		}
	}

	if strings.TrimSpace(resolved.ID) == "" {
		return "", errors.New("daemon: dream workspace id is required")
	}
	return resolved.ID, nil
}

func isPathLikeWorkspaceRef(ref string) bool {
	trimmedRef := strings.TrimSpace(ref)
	return filepath.IsAbs(trimmedRef) ||
		strings.HasPrefix(trimmedRef, ".") ||
		strings.HasPrefix(trimmedRef, "~") ||
		strings.Contains(trimmedRef, string(os.PathSeparator))
}

func spawnDreamSession(ctx context.Context, sessions SessionManager, agentName string, goal string, prompt string, workspace string) (err error) {
	dreamSession, err := sessions.Create(ctx, session.CreateOpts{
		AgentName: agentName,
		Name:      strings.TrimSpace(goal),
		Workspace: strings.TrimSpace(workspace),
		Type:      session.SessionTypeDream,
	})
	if err != nil {
		return fmt.Errorf("daemon: create dream session: %w", err)
	}
	defer func() {
		stopErr := sessions.Stop(ctx, dreamSession.ID)
		if stopErr != nil {
			err = errors.Join(err, fmt.Errorf("daemon: stop dream session %q: %w", dreamSession.ID, stopErr))
		}
	}()

	events, err := sessions.Prompt(ctx, dreamSession.ID, prompt)
	if err != nil {
		return fmt.Errorf("daemon: prompt dream session %q: %w", dreamSession.ID, err)
	}

	for range events {
	}
	return nil
}

func (d *Daemon) shouldVerifyBoundaries() bool {
	if d.verifyBoundaries {
		return true
	}

	value := strings.ToLower(strings.TrimSpace(d.getenv("AGH_DEV_VERIFY_BOUNDARIES")))
	return value == "1" || value == "true" || value == "yes"
}

func (d *Daemon) runtimeLogger() *slog.Logger {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.logger != nil {
		return d.logger
	}
	return slog.Default()
}

func (d *Daemon) signalSource() (<-chan os.Signal, func()) {
	if d.signalCh != nil {
		return d.signalCh, func() {}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch, func() {
		signal.Stop(ch)
	}
}

func (d *Daemon) stopSessions(ctx context.Context, sessions SessionManager) error {
	if sessions == nil {
		return nil
	}

	infos := sessions.List()
	var errs []error
	for _, info := range infos {
		if info == nil {
			continue
		}
		if err := sessions.Stop(ctx, info.ID); err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			errs = append(errs, fmt.Errorf("daemon: stop session %q: %w", info.ID, err))
		}
	}

	return errors.Join(errs...)
}

func (d *Daemon) cleanupOrphans(ctx context.Context, stalePID int) error {
	if stalePID <= 0 {
		return nil
	}

	processes, err := d.listProcesses(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, proc := range processes {
		if proc.PPID != stalePID || proc.PID <= 0 {
			continue
		}
		if err := d.signalProcess(proc.PID, syscall.SIGTERM); err != nil {
			errs = append(errs, fmt.Errorf("daemon: terminate orphan process %d: %w", proc.PID, err))
			continue
		}
		if d.waitForProcessExit(ctx, proc.PID) {
			continue
		}
		if d.processAlive(proc.PID) {
			if err := d.signalProcess(proc.PID, syscall.SIGKILL); err != nil {
				errs = append(errs, fmt.Errorf("daemon: kill orphan process %d: %w", proc.PID, err))
			}
		}
	}

	return errors.Join(errs...)
}

func (d *Daemon) waitForProcessExit(ctx context.Context, pid int) bool {
	if pid <= 0 {
		return true
	}
	if !d.processAlive(pid) {
		return true
	}
	if d.orphanGraceWait <= 0 || d.orphanPollWait <= 0 {
		return !d.processAlive(pid)
	}

	timer := time.NewTimer(d.orphanGraceWait)
	ticker := time.NewTicker(d.orphanPollWait)
	defer timer.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return !d.processAlive(pid)
		case <-ticker.C:
			if !d.processAlive(pid) {
				return true
			}
		case <-timer.C:
			return !d.processAlive(pid)
		}
	}
}

func removeStaleSocket(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil
	}

	if err := os.Remove(cleanPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("daemon: remove stale socket %q: %w", cleanPath, err)
	}
	return nil
}

func resolveDaemonPort(defaultPort int, server Server) int {
	type portReporter interface {
		Port() int
	}

	if reporter, ok := server.(portReporter); ok && reporter.Port() >= 0 {
		return reporter.Port()
	}
	return defaultPort
}

func loadConfigFromHome(homePaths aghconfig.HomePaths) (aghconfig.Config, error) {
	cfg := aghconfig.DefaultWithHome(homePaths)
	if err := aghconfig.ApplyConfigOverlayFile(homePaths.ConfigFile, &cfg); err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: load global config: %w", err)
	}

	socketPath, err := normalizeAbsolutePath(cfg.Daemon.Socket)
	if err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: normalize daemon socket path: %w", err)
	}
	if strings.TrimSpace(socketPath) != "" {
		cfg.Daemon.Socket = socketPath
	}

	if err := cfg.Validate(); err != nil {
		return aghconfig.Config{}, fmt.Errorf("daemon: validate config: %w", err)
	}

	return cfg, nil
}

func normalizeAbsolutePath(path string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", nil
	}
	if clean == "~" || strings.HasPrefix(clean, "~/") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if clean == "~" {
			clean = userHome
		} else {
			clean = filepath.Join(userHome, clean[2:])
		}
	}

	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path %q: %w", path, err)
	}
	return absPath, nil
}

func listProcesses(ctx context.Context) ([]processInfo, error) {
	command := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("daemon: list processes: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	processes := make([]processInfo, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		processes = append(processes, processInfo{PID: pid, PPID: ppid})
	}

	return processes, nil
}

func signalProcess(pid int, sig syscall.Signal) error {
	if pid <= 0 {
		return fmt.Errorf("daemon: invalid process pid %d", pid)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("daemon: find process %d: %w", pid, err)
	}
	if err := process.Signal(sig); err != nil {
		return fmt.Errorf("daemon: signal process %d with %s: %w", pid, sig.String(), err)
	}
	return nil
}

func verifyImportBoundaries(root string) ([]error, error) {
	internalRoot := filepath.Join(root, "internal")
	forbiddenImports := map[string]struct{}{
		moduleImportPath + "/internal/daemon":  {},
		moduleImportPath + "/internal/httpapi": {},
		moduleImportPath + "/internal/udsapi":  {},
		moduleImportPath + "/internal/cli":     {},
	}
	allowedPackages := map[string]struct{}{
		moduleImportPath + "/internal/daemon":  {},
		moduleImportPath + "/internal/httpapi": {},
		moduleImportPath + "/internal/udsapi":  {},
		moduleImportPath + "/internal/cli":     {},
	}

	violations := make([]error, 0)
	fileSet := token.NewFileSet()
	err := filepath.WalkDir(internalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		parsed, err := parser.ParseFile(fileSet, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("daemon: parse %q for boundary verification: %w", path, err)
		}

		dir := filepath.Dir(path)
		relDir, err := filepath.Rel(root, dir)
		if err != nil {
			return fmt.Errorf("daemon: resolve relative package path for %q: %w", dir, err)
		}
		importer := moduleImportPath + "/" + filepath.ToSlash(relDir)
		if _, ok := allowedPackages[importer]; ok {
			return nil
		}

		for _, spec := range parsed.Imports {
			target, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return fmt.Errorf("daemon: decode import path in %q: %w", path, err)
			}
			if _, forbidden := forbiddenImports[target]; forbidden {
				violations = append(violations, fmt.Errorf("daemon: boundary violation: %s imports %s", importer, target))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return violations, nil
}

func (f *notifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *notifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if f.onSessionStopped != nil {
		f.onSessionStopped(ctx, sess)
	}
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *notifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}
