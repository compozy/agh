package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/compozy/agh/internal/acp"
	"github.com/compozy/agh/internal/admission"
	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/httpapi"
	"github.com/compozy/agh/internal/api/udsapi"
	automationpkg "github.com/compozy/agh/internal/automation"
	bridgepkg "github.com/compozy/agh/internal/bridges"
	bundlepkg "github.com/compozy/agh/internal/bundles"
	aghconfig "github.com/compozy/agh/internal/config"
	extensionpkg "github.com/compozy/agh/internal/extension"
	"github.com/compozy/agh/internal/heartbeat"
	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/memory/consolidation"
	"github.com/compozy/agh/internal/network"
	"github.com/compozy/agh/internal/observe"
	"github.com/compozy/agh/internal/procutil"
	"github.com/compozy/agh/internal/resources"
	"github.com/compozy/agh/internal/sandbox"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/situation"
	"github.com/compozy/agh/internal/skills"
	"github.com/compozy/agh/internal/soul"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/toolruntime"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

const defaultShutdownTimeout = 10 * time.Second

var (
	errMissingNetworkBindingSurface = errors.New(
		"daemon: session manager does not implement the network binding surface",
	)
	errMissingWorkspaceRemovalPreparation = errors.New(
		"daemon: session manager does not implement workspace removal preparation",
	)
)

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
	store.SessionCatalog
	store.NetworkAuditStore
	store.NetworkChannelStore
	store.NetworkConversationStore
	store.NetworkMessageStore
	store.NetworkPreferenceStore
	store.NetworkAvailabilityStore
	store.NetworkUsageStore
	store.OnboardingStore
	workspacepkg.Store
	workspacepkg.CoordinationSettings
	workspacepkg.CoordinationCommandStore
}

// Server is a daemon-owned runtime component with explicit start and shutdown phases.
type Server interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
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
type resourceReconcileDriverFactory func(
	ctx context.Context,
	deps resourceReconcileDriverDeps,
) (resources.ReconcileDriver, error)

type networkRuntime interface {
	core.NetworkService
	session.NetworkPeerLifecycle
	Shutdown(context.Context) error
	OnTurnEnd(string)
	SendFromRuntimePeer(context.Context, network.RuntimeSendRequest) (string, error)
}

type networkBindableSessionManager interface {
	Resume(ctx context.Context, sessionID string) (*session.Session, error)
	PromptNetwork(
		ctx context.Context,
		sessionID string,
		message string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
	CancelPrompt(ctx context.Context, sessionID string) error
	IsPrompting(sessionID string) bool
	SetNetworkPeerLifecycle(session.NetworkPeerLifecycle)
	SetTurnEndNotifier(session.TurnEndNotifier)
}

type memoryProviderShutdowner interface {
	Shutdown(context.Context) error
}

type observerRetentionStarter interface {
	StartRetention(context.Context) error
}

type observerRetentionStopper interface {
	ShutdownRetention(context.Context) error
}

type resourceReconcileDriverDeps struct {
	Config           aghconfig.Config
	Logger           *slog.Logger
	Registry         Registry
	ResourceStore    resources.RawStore
	CodecRegistry    *resources.CodecRegistry
	Hooks            *hookspkg.Hooks
	AgentCatalog     *resourceCatalog[aghconfig.AgentDef]
	SoulCatalog      *resourceCatalog[soul.ResourceSpec]
	HeartbeatCatalog *resourceCatalog[heartbeat.ResourceSpec]
	ToolCatalog      *resourceCatalog[toolspkg.Tool]
	MCPServerCatalog *resourceCatalog[aghconfig.MCPServer]
	LoopCatalog      *resourceCatalog[looppkg.ResourceSpec]
	SkillsRegistry   *skills.Registry
	Automation       automationResourceProjectorTarget
	Bridges          bridgeResourceProjectorTarget
	Bundles          resources.BundleActivationProjector[bundlepkg.ActivationResourceSpec, bundlepkg.BundleResourceSpec]
}

type extensionRuntime interface {
	Start(context.Context) error
	Stop(context.Context) error
	Reload(context.Context) error
	Get(string) (*extensionpkg.Extension, error)
	HookDeclarations(context.Context) ([]hookspkg.HookDecl, error)
}

type extensionManagerDeps struct {
	Registry               *extensionpkg.Registry
	Extensions             aghconfig.ExtensionsConfig
	Sessions               SessionManager
	Clarify                toolspkg.ClarifyBroker
	Automation             func() extensionpkg.HostAPIAutomationManager
	Tasks                  taskpkg.Manager
	Network                core.NetworkService
	NetworkStore           store.NetworkConversationStore
	ModelCatalog           core.ModelCatalogService
	MemoryStore            *memory.Store
	MemoryProviderRegistry *extensionpkg.MemoryProviderRegistry
	Observer               Observer
	SkillsRegistry         *skills.Registry
	WorkspaceResolver      workspacepkg.RuntimeResolver
	Logger                 *slog.Logger
	BridgeRegistry         bridgepkg.Registry
	BridgeDedupStore       bridgeDedupStore
	BridgeBroker           *bridgepkg.Broker
	BridgeRuntime          extensionpkg.BridgeRuntimeResolver
	ResourceStore          resources.RawStore
	SourceSessions         resources.SourceSessionManager
	ResourceCodecs         *resources.CodecRegistry
	ResourceTrigger        func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	SoulAuthoring          core.SoulAuthoringService
	SoulRefresher          core.SoulRefresher
	HeartbeatAuthor        core.HeartbeatAuthoringService
	HeartbeatStatus        core.HeartbeatStatusService
	HeartbeatWake          core.HeartbeatWakeService
	SessionHealth          core.SessionHealthReader
	WakeEvents             core.HeartbeatWakeEventReader
	ProcessRegistry        *toolruntime.Registry
	SecretResolver         extensionpkg.SecretRefResolver
	AGHExecutable          func() (string, error)
}

// Daemon is the sole AGH composition root.
type Daemon struct {
	mu sync.Mutex

	homePaths                    aghconfig.HomePaths
	loadConfig                   ConfigLoader
	logger                       *slog.Logger
	closeLogger                  func() error
	now                          func() time.Time
	pid                          func() int
	acquireLock                  func(path string, pid int) (*Lock, error)
	openRegistry                 registryOpener
	newSessionManager            sessionManagerFactory
	newDreamService              consolidation.ServiceFactory
	newObserver                  observerFactory
	newExtensionManager          extensionManagerFactory
	newAutomationManager         automationManagerFactory
	newResourceReconcile         resourceReconcileDriverFactory
	httpFactory                  ServerFactory
	udsFactory                   ServerFactory
	listProcesses                func(context.Context) ([]processInfo, error)
	signalProcess                func(int, syscall.Signal) error
	processAlive                 func(int) bool
	executable                   func() (string, error)
	startDetached                detachedStartFunc
	signalCh                     <-chan os.Signal
	verifyBoundaries             bool
	boundaryRoot                 string
	getenv                       func(string) string
	bridgeSecretResolver         BridgeSecretResolver
	bridgeSecretResolverExplicit bool
	readyCh                      chan struct{}
	readyClosed                  bool
	booting                      bool
	orphanGraceWait              time.Duration
	orphanPollWait               time.Duration
	config                       aghconfig.Config
	startedAt                    time.Time
	info                         Info
	admission                    admission.Gate
	lock                         *Lock
	harnessResolver              *HarnessContextResolver
	registry                     Registry
	memoryStore                  *memory.Store
	memoryProviderRegistry       *extensionpkg.MemoryProviderRegistry
	memoryExtractor              *daemonMemoryExtractor
	runtimeWorkers               daemonRuntimeWorkers
	localMemoryProvider          memoryProviderShutdowner
	situationContext             *situation.Service
	sessions                     SessionManager
	tasks                        *taskRuntime
	coordinator                  *coordinatorRuntime
	spawnReaper                  *spawnReaper
	scheduler                    *schedulerRuntime
	network                      networkRuntime
	networkWakeRunner            *networkWakeRunner
	toolRegistry                 toolspkg.Registry
	clarify                      *clarifyBridge
	hooks                        hookRuntime
	extensions                   extensionRuntime
	observer                     Observer
	resourceReconcile            resources.ReconcileDriver
	agentCatalog                 *resourceCatalog[aghconfig.AgentDef]
	soulCatalog                  *resourceCatalog[soul.ResourceSpec]
	heartbeatCatalog             *resourceCatalog[heartbeat.ResourceSpec]
	toolCatalog                  *resourceCatalog[toolspkg.Tool]
	mcpServerCatalog             *resourceCatalog[aghconfig.MCPServer]
	loopCatalog                  *resourceCatalog[looppkg.ResourceSpec]
	automation                   automationRuntime
	bridges                      *bridgeRuntime
	httpServer                   Server
	udsServer                    Server
	dreamRuntime                 *consolidation.Runtime
	workspaceResolver            workspacepkg.RuntimeResolver
	sandboxRegistry              *sandbox.Registry
	desktopStateRuntime
	skillsRegistry   *skills.Registry
	modelCatalog     *modelCatalogRuntime
	marketplace      *marketplaceRuntime
	skillsCancel     context.CancelFunc
	skillsDone       chan struct{}
	loopsCancel      context.CancelFunc
	loopsDone        chan struct{}
	goalOutboxCancel context.CancelFunc
	goalOutboxDone   chan struct{}
}

// WithHomePaths overrides the resolved AGH home layout.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(d *Daemon) {
		d.homePaths = homePaths
	}
}

// WithConfig overrides daemon-level configuration loading.
func WithConfig(cfg *aghconfig.Config) Option {
	return func(d *Daemon) {
		if cfg == nil {
			return
		}
		cfgCopy := *cfg
		d.loadConfig = func() (aghconfig.Config, error) {
			return cfgCopy, nil
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
// bridge secret bindings into launch-time bound secret material. When this
// option is not supplied, daemon boot wires the canonical vault-backed resolver.
func WithBridgeSecretResolver(resolver BridgeSecretResolver) Option {
	return func(d *Daemon) {
		d.bridgeSecretResolver = resolver
		d.bridgeSecretResolverExplicit = true
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

	d.applyDefaults()

	return d, nil
}

func (d *Daemon) applyDefaults() {
	d.applyCoreDefaults()
	d.applyRuntimeFactoryDefaults()
	d.applyServerFactoryDefaults()
	d.applySystemDefaults()
	d.applyTimingDefaults()
}

func (d *Daemon) applyCoreDefaults() {
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
}

func (d *Daemon) applyRuntimeFactoryDefaults() {
	d.applySessionManagerFactoryDefault()
	if d.newDreamService == nil {
		d.newDreamService = func(opts ...memory.Option) consolidation.Service {
			return memory.NewService(opts...)
		}
	}
	d.applyObserverFactoryDefault()
	d.applyExtensionManagerFactoryDefault()
	d.applyAutomationManagerFactoryDefault()
	d.applyResourceReconcileDriverFactoryDefault()
}

func (d *Daemon) applyAutomationManagerFactoryDefault() {
	if d.newAutomationManager != nil {
		return
	}
	d.newAutomationManager = func(deps automationManagerDeps) (automationRuntime, error) {
		jobStore, triggerStore, err := automationResourceStores(deps.ResourceStore, deps.ResourceCodecs)
		if err != nil {
			return nil, err
		}
		resourceOpts := []automationpkg.Option(nil)
		if jobStore != nil && triggerStore != nil {
			resourceOpts = append(resourceOpts, automationpkg.WithResourceDefinitions(
				jobStore,
				triggerStore,
				resourceReconcileActor(),
				deps.ResourceTrigger,
			))
		}
		loopStarter, err := newAutomationLoopStarter(
			deps.Store,
			deps.LoopCatalog,
			deps.ToolRegistry,
			d.homePaths,
			deps.WorkspaceResolver,
			deps.ParticipationResolver,
		)
		if err != nil {
			return nil, err
		}

		managerOpts := []automationpkg.Option{
			automationpkg.WithStore(deps.Store),
			automationpkg.WithSessions(deps.Sessions),
			automationpkg.WithTasks(deps.Tasks),
			automationpkg.WithWorkspaceResolver(deps.WorkspaceResolver),
			automationpkg.WithConfig(deps.Config),
			automationpkg.WithHooks(deps.Hooks),
			automationpkg.WithWebhookSecretStore(deps.WebhookSecrets),
			automationpkg.WithLogger(deps.Logger),
			automationpkg.WithGlobalWorkspacePath(deps.GlobalWorkspacePath),
		}
		if loopStarter != nil {
			managerOpts = append(managerOpts, automationpkg.WithLoopStarter(loopStarter))
		}
		managerOpts = append(managerOpts, resourceOpts...)

		manager, err := automationpkg.New(managerOpts...)
		if err != nil {
			return nil, err
		}
		return manager, nil
	}
}

func (d *Daemon) applyResourceReconcileDriverFactoryDefault() {
	if d.newResourceReconcile != nil {
		return
	}
	d.newResourceReconcile = func(
		_ context.Context,
		deps resourceReconcileDriverDeps,
	) (resources.ReconcileDriver, error) {
		if deps.ResourceStore == nil || deps.CodecRegistry == nil {
			return resources.NewReconcileDriver(
				nil,
				resources.MutationActor{},
				nil,
				resources.WithReconcileLogger(deps.Logger),
			)
		}

		registrations, err := buildResourceProjectorRegistrations(&deps)
		if err != nil {
			return nil, err
		}

		return resources.NewReconcileDriver(
			deps.ResourceStore,
			resourceReconcileActor(),
			registrations,
			resources.WithReconcileLogger(deps.Logger),
		)
	}
}

func buildResourceProjectorRegistrations(
	deps *resourceReconcileDriverDeps,
) ([]resources.ProjectorRegistration, error) {
	var registrations []resources.ProjectorRegistration
	var err error
	registrations, err = appendCoreProjectorRegistrations(registrations, deps)
	if err != nil {
		return nil, err
	}
	if deps.Automation != nil {
		registrations, err = appendAutomationProjectorRegistrations(registrations, deps)
		if err != nil {
			return nil, err
		}
	}
	if deps.Bridges != nil {
		registrations, err = appendBridgeProjectorRegistration(registrations, deps)
		if err != nil {
			return nil, err
		}
	}
	if deps.Bundles != nil {
		registrations, err = appendBundleProjectorRegistrations(registrations, deps)
		if err != nil {
			return nil, err
		}
	}
	return registrations, nil
}

func appendCoreProjectorRegistrations(
	registrations []resources.ProjectorRegistration,
	deps *resourceReconcileDriverDeps,
) ([]resources.ProjectorRegistration, error) {
	var err error
	if deps.Hooks != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			hookBindingResourceKind,
			newHookBindingProjector(deps.Hooks),
		)
	}
	if err != nil {
		return nil, err
	}
	if deps.AgentCatalog != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			aghconfig.AgentResourceKind,
			newAgentProjector(deps.AgentCatalog),
		)
	}
	if err != nil {
		return nil, err
	}
	if deps.SoulCatalog != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			soul.ResourceKind,
			newSoulProjector(deps.SoulCatalog),
		)
	}
	if err != nil {
		return nil, err
	}
	if deps.HeartbeatCatalog != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			heartbeat.ResourceKind,
			newHeartbeatProjector(deps.HeartbeatCatalog),
		)
	}
	if err != nil {
		return nil, err
	}
	if deps.ToolCatalog != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			toolspkg.ToolResourceKind,
			newToolProjector(deps.ToolCatalog),
		)
	}
	if err != nil {
		return nil, err
	}
	if deps.MCPServerCatalog != nil {
		registrations, err = appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			aghconfig.MCPServerResourceKind,
			newMCPServerProjector(deps.MCPServerCatalog),
		)
	}
	if err != nil {
		return nil, err
	}
	registrations, err = appendLoopProjectorRegistration(registrations, deps)
	if err != nil {
		return nil, err
	}
	return appendSkillProjectorRegistration(registrations, deps)
}

func appendTypedProjectorRegistration[T any](
	registrations []resources.ProjectorRegistration,
	registry *resources.CodecRegistry,
	kind resources.ResourceKind,
	projector resources.TypedProjector[T],
) ([]resources.ProjectorRegistration, error) {
	codec, err := resources.ResolveCodec[T](registry, kind)
	if err != nil {
		return nil, err
	}
	registration, err := resources.NewTypedProjectorRegistration(codec, projector)
	if err != nil {
		return nil, err
	}
	return append(registrations, registration), nil
}

func appendAutomationProjectorRegistrations(
	registrations []resources.ProjectorRegistration,
	deps *resourceReconcileDriverDeps,
) ([]resources.ProjectorRegistration, error) {
	jobCodec, err := resources.ResolveCodec[automationpkg.Job](deps.CodecRegistry, automationpkg.JobResourceKind)
	if err != nil {
		return nil, err
	}
	jobRegistration, err := resources.NewTypedProjectorRegistration(
		jobCodec,
		newAutomationJobProjector(deps.Automation),
	)
	if err != nil {
		return nil, err
	}

	triggerCodec, err := resources.ResolveCodec[automationpkg.Trigger](
		deps.CodecRegistry,
		automationpkg.TriggerResourceKind,
	)
	if err != nil {
		return nil, err
	}
	triggerRegistration, err := resources.NewTypedProjectorRegistration(
		triggerCodec,
		newAutomationTriggerProjector(deps.Automation),
	)
	if err != nil {
		return nil, err
	}

	registrations = append(registrations, jobRegistration, triggerRegistration)
	return registrations, nil
}

func resourceReconcileActor() resources.MutationActor {
	return resources.MutationActor{
		Kind: resources.MutationActorKindDaemon,
		ID:   "daemon-control",
		Source: resources.ResourceSource{
			Kind: resources.ResourceSourceKind("daemon"),
			ID:   string(SessionClassSystem),
		},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}
}

func (d *Daemon) applyServerFactoryDefaults() {
	if d.httpFactory == nil {
		d.httpFactory = func(_ context.Context, deps RuntimeDeps) (Server, error) {
			return httpapi.New(httpServerOptions(&deps)...)
		}
	}
	if d.udsFactory == nil {
		d.udsFactory = func(_ context.Context, deps RuntimeDeps) (Server, error) {
			return udsapi.New(udsServerOptions(&deps)...)
		}
	}
}

func (d *Daemon) applySystemDefaults() {
	if d.listProcesses == nil {
		d.listProcesses = listProcesses
	}
	if d.signalProcess == nil {
		d.signalProcess = procutil.Signal
	}
	if d.processAlive == nil {
		d.processAlive = procutil.Alive
	}
	if d.executable == nil {
		d.executable = os.Executable
	}
	if d.startDetached == nil {
		d.startDetached = defaultDetachedStart
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
}

func (d *Daemon) applyTimingDefaults() {
	if d.orphanGraceWait <= 0 {
		d.orphanGraceWait = orphanCleanupGraceWait
	}
	if d.orphanPollWait <= 0 {
		d.orphanPollWait = orphanCleanupPollWait
	}
}

func (d *Daemon) startObserverRetention(ctx context.Context) error {
	d.mu.Lock()
	observer := d.observer
	d.mu.Unlock()

	starter, ok := observer.(observerRetentionStarter)
	if !ok {
		return nil
	}
	return starter.StartRetention(ctx)
}

// Run boots the daemon, blocks until signal or context cancellation, then performs graceful shutdown.
func (d *Daemon) Run(ctx context.Context) error {
	if ctx == nil {
		return errors.New("daemon: run context is required")
	}

	sigCh, stopSignals := d.signalSource()
	defer stopSignals()
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	receivedSignal := make(chan os.Signal, 1)
	signalDone := make(chan struct{})
	go func() {
		defer close(signalDone)
		select {
		case <-runCtx.Done():
			return
		case sig, ok := <-sigCh:
			if ok && sig != nil {
				select {
				case receivedSignal <- sig:
				default:
				}
				cancelRun()
			}
		}
	}()

	if err := d.boot(runCtx); err != nil {
		cancelRun()
		<-signalDone
		return err
	}
	if d.dreamRuntime != nil {
		d.dreamRuntime.Start(runCtx)
	}
	if d.memoryExtractor != nil {
		if err := d.memoryExtractor.Start(runCtx); err != nil {
			cancelRun()
			<-signalDone
			shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
			defer cancel()
			shutdownErr := d.Shutdown(shutdownCtx)
			return errors.Join(
				fmt.Errorf("daemon: start memory extractor: %w", err),
				shutdownErr,
			)
		}
	}
	if err := d.startObserverRetention(runCtx); err != nil {
		cancelRun()
		<-signalDone
		shutdownCtx, cancel := daemonShutdownContext(ctx)
		defer cancel()
		shutdownErr := d.Shutdown(shutdownCtx)
		return errors.Join(
			fmt.Errorf("daemon: start observability retention: %w", err),
			shutdownErr,
		)
	}

	select {
	case <-ctx.Done():
	case sig := <-receivedSignal:
		d.runtimeLogger().Info("daemon: received shutdown signal", "signal", sig.String())
	}
	cancelRun()
	<-signalDone

	shutdownCtx, cancel := daemonShutdownContext(ctx)
	defer cancel()

	return d.Shutdown(shutdownCtx)
}

// Shutdown gracefully tears down the daemon in the required order.
func (d *Daemon) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.TODO()
	}
	drainErr := d.Drain(context.WithoutCancel(ctx))
	return errors.Join(drainErr, d.shutdownDetached(ctx, d.detachShutdownTargets()))
}

func daemonShutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithTimeout(context.WithoutCancel(parent), defaultShutdownTimeout)
}

func (d *Daemon) shutdownDetached(ctx context.Context, targets shutdownTargets) error {
	var errs []error
	d.shutdownRuntimeWorkers(ctx, targets, &errs)
	d.shutdownServersAndHooks(ctx, targets, &errs)
	d.shutdownPersistentResources(ctx, targets, &errs)
	return errors.Join(errs...)
}

func (d *Daemon) shutdownServersAndHooks(ctx context.Context, targets shutdownTargets, errs *[]error) {
	if targets.httpServer != nil {
		appendWrappedError(errs, "daemon: shutdown http server", targets.httpServer.Shutdown(ctx))
	}
	if targets.udsServer != nil {
		appendWrappedError(errs, "daemon: shutdown uds server", targets.udsServer.Shutdown(ctx))
	}
	if targets.bridges != nil {
		targets.bridges.Close()
	}
	if targets.network != nil {
		appendWrappedError(errs, "daemon: shutdown network runtime", targets.network.Shutdown(ctx))
	}
	if targets.hooks != nil {
		targets.hooks.Close()
	}
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
