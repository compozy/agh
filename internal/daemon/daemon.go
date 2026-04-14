package daemon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/httpapi"
	"github.com/pedronauck/agh/internal/api/udsapi"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const defaultShutdownTimeout = 10 * time.Second

var errMissingNetworkBindingSurface = errors.New("daemon: session manager does not implement the network binding surface")

// Option customizes daemon construction.
type Option func(*Daemon)

// ConfigLoader resolves the daemon-level runtime configuration.
type ConfigLoader func() (aghconfig.Config, error)

// SessionManager is the shared transport-facing session surface consumed by daemon/.
type SessionManager = core.SessionManager

// Observer is the daemon observer surface used for transport wiring and reconciliation.
type Observer interface {
	core.Observer
	session.Notifier
	Reconcile(ctx context.Context) (store.ReconcileResult, error)
}

// Registry is the narrowed global database surface shared by observe and workspace.
type Registry interface {
	observe.Registry
	store.NetworkAuditStore
	store.NetworkMessageStore
	workspacepkg.WorkspaceStore
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
	Tasks             taskpkg.Manager
	Network           core.NetworkService
	Observer          Observer
	Automation        core.AutomationManager
	Bridges           core.BridgeService
	Registry          Registry
	MemoryStore       *memory.Store
	WorkspaceResolver workspacepkg.WorkspaceResolver
	WorkspaceService  core.WorkspaceService
	SkillsRegistry    core.SkillsRegistry
	DreamTrigger      DreamTrigger
	Extensions        udsapi.ExtensionService
	StartedAt         time.Time
}

// ServerFactory constructs runtime components such as HTTP and UDS servers.
type ServerFactory func(ctx context.Context, deps RuntimeDeps) (Server, error)

// DreamTrigger exposes consolidation controls and health state to transport layers.
type DreamTrigger = core.DreamTrigger

type registryOpener func(ctx context.Context, path string) (Registry, error)
type sessionManagerFactory func(ctx context.Context, deps SessionManagerDeps) (SessionManager, error)
type observerFactory func(ctx context.Context, deps RuntimeDeps) (Observer, error)
type extensionManagerFactory func(deps extensionManagerDeps) extensionRuntime
type automationManagerFactory func(deps automationManagerDeps) (automationRuntime, error)

type networkRuntime interface {
	core.NetworkService
	session.NetworkPeerLifecycle
	Shutdown(context.Context) error
	OnTurnEnd(string)
}

type networkBindableSessionManager interface {
	PromptNetwork(ctx context.Context, sessionID string, message string) (<-chan acp.AgentEvent, error)
	IsPrompting(sessionID string) bool
	SetNetworkPeerLifecycle(session.NetworkPeerLifecycle)
	SetTurnEndNotifier(session.TurnEndNotifier)
}

type shutdownStopper interface {
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type finalizationWaiter interface {
	WaitForFinalizations(ctx context.Context) error
}

type extensionDBSource interface {
	DB() *sql.DB
}

type extensionRuntime interface {
	Start(context.Context) error
	Stop(context.Context) error
	Reload(context.Context) error
	Get(string) (*extensionpkg.Extension, error)
	HookDeclarations(context.Context) ([]hookspkg.HookDecl, error)
}

func bridgeObserveSource(service core.BridgeService) observe.BridgeSource {
	if service == nil {
		return nil
	}
	source, _ := service.(observe.BridgeSource)
	return source
}

type extensionManagerDeps struct {
	Registry          *extensionpkg.Registry
	Sessions          SessionManager
	Automation        func() extensionpkg.HostAPIAutomationManager
	Tasks             taskpkg.Manager
	MemoryStore       *memory.Store
	Observer          Observer
	SkillsRegistry    *skills.Registry
	WorkspaceResolver workspacepkg.WorkspaceResolver
	Logger            *slog.Logger
	BridgeRegistry    bridgepkg.Registry
	BridgeDedupStore  bridgeDedupStore
	BridgeBroker      *bridgepkg.Broker
	BridgeRuntime     extensionpkg.BridgeRuntimeResolver
}

type automationRuntime interface {
	core.AutomationManager
	extensionpkg.HostAPIAutomationManager
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
	SessionObserver() session.Notifier
	HookTelemetrySink() hookspkg.TelemetrySink
	MemoryObserver() automationpkg.MemoryConsolidationObserver
}

type automationManagerDeps struct {
	Store               automationpkg.Store
	Sessions            SessionManager
	Tasks               taskpkg.Manager
	WorkspaceResolver   workspacepkg.WorkspaceResolver
	Config              aghconfig.AutomationConfig
	Hooks               automationpkg.AutomationHookDispatcher
	Logger              *slog.Logger
	GlobalWorkspacePath string
}

// SessionManagerDeps captures the composition-root dependencies needed to create a session manager.
type SessionManagerDeps struct {
	HomePaths         aghconfig.HomePaths
	Logger            *slog.Logger
	Notifier          session.Notifier
	Hooks             session.HookSet
	PromptAssembler   session.PromptAssembler
	SkillRegistry     session.SkillRegistry
	MCPResolver       session.MCPResolver
	WorkspaceResolver workspacepkg.WorkspaceResolver
}

// Daemon is the sole AGH composition root.
type Daemon struct {
	mu sync.Mutex

	homePaths            aghconfig.HomePaths
	loadConfig           ConfigLoader
	logger               *slog.Logger
	closeLogger          func() error
	now                  func() time.Time
	pid                  func() int
	acquireLock          func(path string, pid int) (*Lock, error)
	openRegistry         registryOpener
	newSessionManager    sessionManagerFactory
	newDreamService      consolidation.ServiceFactory
	newObserver          observerFactory
	newExtensionManager  extensionManagerFactory
	newAutomationManager automationManagerFactory
	httpFactory          ServerFactory
	udsFactory           ServerFactory
	listProcesses        func(context.Context) ([]processInfo, error)
	signalProcess        func(int, syscall.Signal) error
	processAlive         func(int) bool
	signalCh             <-chan os.Signal
	verifyBoundaries     bool
	boundaryRoot         string
	getenv               func(string) string
	bridgeSecretResolver BridgeSecretResolver
	readyCh              chan struct{}
	readyClosed          bool
	booting              bool
	orphanGraceWait      time.Duration
	orphanPollWait       time.Duration
	config               aghconfig.Config
	startedAt            time.Time
	info                 Info
	lock                 *Lock
	registry             Registry
	memoryStore          *memory.Store
	sessions             SessionManager
	tasks                *taskRuntime
	network              networkRuntime
	hooks                hookRuntime
	extensions           extensionRuntime
	observer             Observer
	automation           automationRuntime
	bridges              *bridgeRuntime
	httpServer           Server
	udsServer            Server
	dreamRuntime         *consolidation.Runtime
	workspaceResolver    workspacepkg.WorkspaceResolver
	skillsRegistry       *skills.Registry
	skillsCancel         context.CancelFunc
	skillsDone           chan struct{}
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

// WithBridgeSecretResolver injects the daemon-owned resolver used to convert
// bridge secret bindings into launch-time bound secret material.
func WithBridgeSecretResolver(resolver BridgeSecretResolver) Option {
	return func(d *Daemon) {
		d.bridgeSecretResolver = resolver
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

// WithSignalBridge overrides OS signal delivery, mainly for tests.
func WithSignalBridge(ch <-chan os.Signal) Option {
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

	if err := d.applyDefaults(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Daemon) applyDefaults() error {
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
			return globaldb.OpenGlobalDB(ctx, path)
		}
	}
	if d.newSessionManager == nil {
		d.newSessionManager = func(ctx context.Context, deps SessionManagerDeps) (SessionManager, error) {
			return session.NewManager(
				session.WithHomePaths(deps.HomePaths),
				session.WithLifecycleContext(ctx),
				session.WithLogger(deps.Logger),
				session.WithNotifier(deps.Notifier),
				session.WithHookSet(deps.Hooks),
				session.WithPromptAssembler(deps.PromptAssembler),
				session.WithSkillRegistry(deps.SkillRegistry),
				session.WithMCPResolver(deps.MCPResolver),
				session.WithWorkspaceResolver(deps.WorkspaceResolver),
			)
		}
	}
	if d.newDreamService == nil {
		d.newDreamService = func(opts ...memory.Option) consolidation.Service {
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
				observe.WithBridgeSource(bridgeObserveSource(deps.Bridges)),
			)
		}
	}
	if d.newExtensionManager == nil {
		d.newExtensionManager = func(deps extensionManagerDeps) extensionRuntime {
			if deps.Registry == nil {
				return nil
			}

			capChecker := &extensionpkg.CapabilityChecker{}
			hostAPIOpts := []extensionpkg.HostAPIOption{
				extensionpkg.WithHostAPIAutomationGetter(deps.Automation),
				extensionpkg.WithHostAPITaskManager(deps.Tasks),
				extensionpkg.WithHostAPICapabilityChecker(capChecker),
				extensionpkg.WithHostAPIWorkspaceResolver(deps.WorkspaceResolver),
			}
			if deps.BridgeRegistry != nil {
				hostAPIOpts = append(hostAPIOpts, extensionpkg.WithHostAPIBridgeRegistry(deps.BridgeRegistry))
			}
			if deps.BridgeDedupStore != nil {
				hostAPIOpts = append(hostAPIOpts, extensionpkg.WithHostAPIBridgeDedupStore(deps.BridgeDedupStore))
			}
			if deps.BridgeBroker != nil {
				hostAPIOpts = append(hostAPIOpts, extensionpkg.WithHostAPIDeliveryBroker(deps.BridgeBroker))
			}

			hostAPI := extensionpkg.NewHostAPIHandler(
				deps.Sessions,
				deps.MemoryStore,
				deps.Observer,
				deps.SkillsRegistry,
				hostAPIOpts...,
			)

			opts := []extensionpkg.Option{
				extensionpkg.WithCapabilityChecker(capChecker),
				extensionpkg.WithSkillsRegistry(deps.SkillsRegistry),
				extensionpkg.WithLogger(deps.Logger),
			}
			if sink, ok := deps.Observer.(extensionpkg.BridgeTelemetrySink); ok {
				opts = append(opts, extensionpkg.WithBridgeTelemetrySink(sink))
			}
			if deps.BridgeRuntime != nil {
				opts = append(opts, extensionpkg.WithBridgeRuntimeResolver(deps.BridgeRuntime))
			}
			for method, handler := range hostAPI.MethodHandlers() {
				opts = append(opts, extensionpkg.WithHostMethodHandler(method, handler))
			}

			return extensionpkg.NewManager(deps.Registry, opts...)
		}
	}
	if d.newAutomationManager == nil {
		d.newAutomationManager = func(deps automationManagerDeps) (automationRuntime, error) {
			manager, err := automationpkg.New(
				automationpkg.WithStore(deps.Store),
				automationpkg.WithSessions(deps.Sessions),
				automationpkg.WithTasks(deps.Tasks),
				automationpkg.WithWorkspaceResolver(deps.WorkspaceResolver),
				automationpkg.WithConfig(deps.Config),
				automationpkg.WithHooks(deps.Hooks),
				automationpkg.WithLogger(deps.Logger),
				automationpkg.WithGlobalWorkspacePath(deps.GlobalWorkspacePath),
			)
			if err != nil {
				return nil, err
			}
			return manager, nil
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
				httpapi.WithTaskService(deps.Tasks),
				httpapi.WithNetworkService(deps.Network),
				httpapi.WithNetworkStore(deps.Registry),
				httpapi.WithObserver(deps.Observer),
				httpapi.WithAutomation(deps.Automation),
				httpapi.WithBridgeService(deps.Bridges),
				httpapi.WithWorkspaceResolver(deps.WorkspaceService),
				httpapi.WithSkillsRegistry(deps.SkillsRegistry),
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
				udsapi.WithTaskService(deps.Tasks),
				udsapi.WithNetworkService(deps.Network),
				udsapi.WithNetworkStore(deps.Registry),
				udsapi.WithObserver(deps.Observer),
				udsapi.WithAutomation(deps.Automation),
				udsapi.WithBridgeService(deps.Bridges),
				udsapi.WithWorkspaceResolver(deps.WorkspaceService),
				udsapi.WithSkillsRegistry(deps.SkillsRegistry),
				udsapi.WithMemoryStore(deps.MemoryStore),
				udsapi.WithDreamTrigger(deps.DreamTrigger),
				udsapi.WithExtensionService(deps.Extensions),
			)
		}
	}
	if d.listProcesses == nil {
		d.listProcesses = listProcesses
	}
	if d.signalProcess == nil {
		d.signalProcess = procutil.Signal
	}
	if d.processAlive == nil {
		d.processAlive = procutil.Alive
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

	return nil
}

// Run boots the daemon, blocks until signal or context cancellation, then performs graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("daemon: run context is required")
	}
	if err := d.boot(ctx); err != nil {
		return err
	}
	if d.dreamRuntime != nil {
		d.dreamRuntime.Start(ctx)
	}

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
	network := d.network
	hooks := d.hooks
	extensions := d.extensions
	automation := d.automation
	bridges := d.bridges
	httpServer := d.httpServer
	udsServer := d.udsServer
	registry := d.registry
	lock := d.lock
	closeLogger := d.closeLogger
	infoPath := d.homePaths.DaemonInfo
	dreamRuntime := d.dreamRuntime
	skillsCancel := d.skillsCancel
	skillsDone := d.skillsDone

	d.sessions = nil
	d.tasks = nil
	d.hooks = nil
	d.extensions = nil
	d.automation = nil
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
	d.dreamRuntime = nil
	d.workspaceResolver = nil
	d.skillsCancel = nil
	d.skillsDone = nil
	d.bridges = nil
	d.network = nil
	d.mu.Unlock()

	var errs []error
	if dreamRuntime != nil {
		dreamRuntime.Shutdown()
	}
	stopSkillsWatcher(skillsCancel, skillsDone)
	if extensions != nil {
		if err := extensions.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: stop extensions: %w", err))
		}
	}
	if automation != nil {
		if err := automation.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: shutdown automation: %w", err))
		}
	}
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
	if bridges != nil {
		bridges.Close()
	}
	if network != nil {
		if err := network.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: shutdown network runtime: %w", err))
		}
	}
	if hooks != nil {
		hooks.Close()
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
		var err error
		if stopper, ok := sessions.(shutdownStopper); ok {
			err = stopper.StopWithCause(ctx, info.ID, session.CauseShutdown, "daemon shutdown")
		} else {
			err = sessions.Stop(ctx, info.ID)
		}
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			errs = append(errs, fmt.Errorf("daemon: stop session %q: %w", info.ID, err))
		}
	}
	if waiter, ok := sessions.(finalizationWaiter); ok {
		if err := waiter.WaitForFinalizations(ctx); err != nil {
			errs = append(errs, fmt.Errorf("daemon: wait for session finalizations: %w", err))
		}
	}

	return errors.Join(errs...)
}
