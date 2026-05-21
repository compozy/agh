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
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	"github.com/pedronauck/agh/internal/heartbeat"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/procutil"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/sandbox"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/situation"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/toolruntime"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const defaultShutdownTimeout = 10 * time.Second

var errMissingNetworkBindingSurface = errors.New(
	"daemon: session manager does not implement the network binding surface",
)

// Option customizes daemon construction.
type Option func(*Daemon)

// ConfigLoader resolves the daemon-level runtime configuration.
type ConfigLoader func() (aghconfig.Config, error)

// SessionManager is the shared transport-facing session surface consumed by daemon/.
type SessionManager = core.SessionManager

type sandboxExecSessionManager interface {
	ExecSandbox(context.Context, session.SandboxExecRequest) (session.SandboxExecResult, error)
}

type hostAPIExtensionSessionManager interface {
	SessionManager
	ExecSandbox(context.Context, session.SandboxExecRequest) (session.SandboxExecResult, error)
}

type hostAPIBridgePromptSessionManager interface {
	PromptNetwork(
		ctx context.Context,
		sessionID string,
		message string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
	IsPrompting(sessionID string) bool
}

type hostAPISessionManagerAdapter struct {
	core.SessionManager
	exec sandboxExecSessionManager
}

type hostAPINetworkSessionManagerAdapter struct {
	hostAPISessionManagerAdapter
	bridgePrompts hostAPIBridgePromptSessionManager
}

func newHostAPISessionManagerAdapter(sessions SessionManager) hostAPIExtensionSessionManager {
	adapter := hostAPISessionManagerAdapter{SessionManager: sessions}
	if exec, ok := sessions.(sandboxExecSessionManager); ok {
		adapter.exec = exec
	}
	if bridgePrompts, ok := sessions.(hostAPIBridgePromptSessionManager); ok {
		return hostAPINetworkSessionManagerAdapter{
			hostAPISessionManagerAdapter: adapter,
			bridgePrompts:                bridgePrompts,
		}
	}
	return adapter
}

func (a hostAPISessionManagerAdapter) ExecSandbox(
	ctx context.Context,
	req session.SandboxExecRequest,
) (session.SandboxExecResult, error) {
	if a.exec == nil {
		return session.SandboxExecResult{}, session.ErrSessionNotActive
	}
	return a.exec.ExecSandbox(ctx, req)
}

func (a hostAPINetworkSessionManagerAdapter) PromptNetwork(
	ctx context.Context,
	sessionID string,
	message string,
	meta ...acp.PromptNetworkMeta,
) (<-chan acp.AgentEvent, error) {
	return a.bridgePrompts.PromptNetwork(ctx, sessionID, message, meta...)
}

func (a hostAPINetworkSessionManagerAdapter) IsPrompting(sessionID string) bool {
	return a.bridgePrompts.IsPrompting(sessionID)
}

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
	store.NetworkChannelStore
	store.NetworkConversationStore
	store.NetworkMessageStore
	workspacepkg.Store
}

// Server is a daemon-owned runtime component with explicit start and shutdown phases.
type Server interface {
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

// RuntimeDeps captures the composition-root dependencies available to server factories.
type RuntimeDeps struct {
	Config              aghconfig.Config
	HomePaths           aghconfig.HomePaths
	Logger              *slog.Logger
	Sessions            SessionManager
	Tasks               taskpkg.Manager
	Network             core.NetworkService
	ToolRegistry        toolspkg.Registry
	Toolsets            core.ToolsetRegistry
	ToolApprovals       toolspkg.ApprovalTokenIssuer
	HostedMCP           *mcppkg.HostedService
	Observer            Observer
	Automation          core.AutomationManager
	Bridges             core.BridgeService
	Registry            Registry
	MemoryStore         *memory.Store
	MemoryExtractor     core.MemoryExtractorService
	MemoryProviders     core.MemoryProviderService
	MemorySessionLedger core.MemorySessionLedgerService
	WorkspaceResolver   workspacepkg.RuntimeResolver
	WorkspaceService    core.WorkspaceService
	AgentCatalog        core.AgentCatalog
	ModelCatalog        core.ModelCatalogService
	AgentContext        *situation.Service
	SoulAuthoring       core.SoulAuthoringService
	SoulRefresher       core.SoulRefresher
	HeartbeatAuthor     core.HeartbeatAuthoringService
	HeartbeatStatus     core.HeartbeatStatusService
	HeartbeatWake       core.HeartbeatWakeService
	SessionHealth       core.SessionHealthReader
	WakeEvents          core.HeartbeatWakeEventReader
	CoordinatorConfig   CoordinatorConfigResolver
	SkillsRegistry      core.SkillsRegistry
	DreamTrigger        DreamTrigger
	Settings            core.SettingsService
	SettingsRestart     core.SettingsRestartController
	SettingsUpdate      core.SettingsUpdateController
	SupportBundles      core.SupportBundleService
	Vault               core.VaultService
	Extensions          udsapi.ExtensionService
	Bundles             core.BundleService
	Resources           core.ResourceService
	StartedAt           time.Time
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
}

type networkBindableSessionManager interface {
	PromptNetwork(
		ctx context.Context,
		sessionID string,
		message string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
	IsPrompting(sessionID string) bool
	SetNetworkPeerLifecycle(session.NetworkPeerLifecycle)
	SetTurnEndNotifier(session.TurnEndNotifier)
}

type shutdownStopper interface {
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
}

type memoryProviderShutdowner interface {
	Shutdown(context.Context) error
}

type finalizationWaiter interface {
	WaitForFinalizations(ctx context.Context) error
}

type observerRetentionStarter interface {
	StartRetention(context.Context) error
}

type observerRetentionStopper interface {
	ShutdownRetention(context.Context) error
}

type extensionDBSource interface {
	DB() *sql.DB
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

func bridgeObserveSource(service core.BridgeService) observe.BridgeSource {
	if service == nil {
		return nil
	}
	source, ok := service.(observe.BridgeSource)
	if !ok {
		return nil
	}
	return source
}

type extensionManagerDeps struct {
	Registry               *extensionpkg.Registry
	Extensions             aghconfig.ExtensionsConfig
	Sessions               SessionManager
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
	WorkspaceResolver   workspacepkg.RuntimeResolver
	Config              aghconfig.AutomationConfig
	Hooks               automationpkg.HookDispatcher
	WebhookSecrets      automationpkg.WebhookSecretStore
	Logger              *slog.Logger
	GlobalWorkspacePath string
	ResourceStore       resources.RawStore
	ResourceCodecs      *resources.CodecRegistry
	ResourceTrigger     func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
}

// SessionManagerDeps captures the composition-root dependencies needed to create a session manager.
type SessionManagerDeps struct {
	HomePaths            aghconfig.HomePaths
	Logger               *slog.Logger
	Notifier             session.Notifier
	Hooks                session.HookSet
	PromptAssembler      session.PromptAssembler
	StartupPromptOverlay session.StartupPromptOverlay
	PromptInputAugmenter session.PromptInputAugmenter
	MemoryStore          *memory.Store
	LedgerMaterializer   session.LedgerMaterializer
	AgentResolver        session.AgentResolver
	SkillRegistry        session.SkillRegistry
	MCPResolver          session.MCPResolver
	WorkspaceResolver    workspacepkg.RuntimeResolver
	SandboxRegistry      *sandbox.Registry
	SessionSupervision   aghconfig.SessionSupervisionConfig
	SessionHealthConfig  aghconfig.HeartbeatConfig
	ProcessRegistry      *toolruntime.Registry
	HostedMCP            session.HostedMCPLauncher
	ProviderSecrets      session.ProviderSecretResolver
	SoulStore            session.SoulSnapshotStore
	SoulRunChecker       session.SoulRunActivityChecker
	SessionHealthStore   session.HealthStore
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
	lock                         *Lock
	harnessResolver              *HarnessContextResolver
	registry                     Registry
	memoryStore                  *memory.Store
	memoryProviderRegistry       *extensionpkg.MemoryProviderRegistry
	memoryExtractor              *daemonMemoryExtractor
	localMemoryProvider          memoryProviderShutdowner
	situationContext             *situation.Service
	sessions                     SessionManager
	tasks                        *taskRuntime
	spawnReaper                  *spawnReaper
	scheduler                    *schedulerRuntime
	network                      networkRuntime
	toolRegistry                 toolspkg.Registry
	hooks                        hookRuntime
	extensions                   extensionRuntime
	observer                     Observer
	resourceReconcile            resources.ReconcileDriver
	agentCatalog                 *resourceCatalog[aghconfig.AgentDef]
	soulCatalog                  *resourceCatalog[soul.ResourceSpec]
	heartbeatCatalog             *resourceCatalog[heartbeat.ResourceSpec]
	toolCatalog                  *resourceCatalog[toolspkg.Tool]
	mcpServerCatalog             *resourceCatalog[aghconfig.MCPServer]
	automation                   automationRuntime
	bridges                      *bridgeRuntime
	httpServer                   Server
	udsServer                    Server
	dreamRuntime                 *consolidation.Runtime
	workspaceResolver            workspacepkg.RuntimeResolver
	sandboxRegistry              *sandbox.Registry
	skillsRegistry               *skills.Registry
	modelCatalog                 *modelCatalogRuntime
	skillsCancel                 context.CancelFunc
	skillsDone                   chan struct{}
}

type shutdownTargets struct {
	scheduler           *schedulerRuntime
	spawnReaper         *spawnReaper
	tasks               *taskRuntime
	sessions            SessionManager
	network             networkRuntime
	hooks               hookRuntime
	extensions          extensionRuntime
	automation          automationRuntime
	resourceReconcile   resources.ReconcileDriver
	bridges             *bridgeRuntime
	httpServer          Server
	udsServer           Server
	registry            Registry
	lock                *Lock
	closeLogger         func() error
	infoPath            string
	dreamRuntime        *consolidation.Runtime
	memoryExtractor     *daemonMemoryExtractor
	memoryStore         *memory.Store
	localMemoryProvider memoryProviderShutdowner
	modelCatalog        *modelCatalogRuntime
	skillsCancel        context.CancelFunc
	skillsDone          chan struct{}
	retention           observerRetentionStopper
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

func (d *Daemon) applySessionManagerFactoryDefault() {
	if d.newSessionManager != nil {
		return
	}
	d.newSessionManager = func(ctx context.Context, deps SessionManagerDeps) (SessionManager, error) {
		return session.NewManager(
			session.WithHomePaths(deps.HomePaths),
			session.WithLifecycleContext(ctx),
			session.WithLogger(deps.Logger),
			session.WithNotifier(deps.Notifier),
			session.WithHookSet(deps.Hooks),
			session.WithPromptAssembler(deps.PromptAssembler),
			session.WithStartupPromptOverlay(deps.StartupPromptOverlay),
			session.WithPromptInputAugmenter(deps.PromptInputAugmenter),
			session.WithAgentResolver(deps.AgentResolver),
			session.WithSkillRegistry(deps.SkillRegistry),
			session.WithMCPResolver(deps.MCPResolver),
			session.WithWorkspaceResolver(deps.WorkspaceResolver),
			session.WithSandboxRegistry(deps.SandboxRegistry),
			session.WithSessionSupervision(deps.SessionSupervision),
			session.WithSessionHealthConfig(deps.SessionHealthConfig),
			session.WithSessionHealthStore(deps.SessionHealthStore),
			session.WithHostedMCPLauncher(deps.HostedMCP),
			session.WithProviderSecretResolver(deps.ProviderSecrets),
			session.WithSoulSnapshotStore(deps.SoulStore),
			session.WithSoulRunActivityChecker(deps.SoulRunChecker),
			session.WithLedgerMaterializer(deps.LedgerMaterializer),
			session.WithDriver(session.NewACPDriverAdapter(acp.New(
				acp.WithLogger(deps.Logger),
				acp.WithProcessRegistry(deps.ProcessRegistry),
			))),
		)
	}
}

func (d *Daemon) applyObserverFactoryDefault() {
	if d.newObserver != nil {
		return
	}
	d.newObserver = func(ctx context.Context, deps RuntimeDeps) (Observer, error) {
		source, ok := deps.Sessions.(observe.SessionSource)
		if !ok {
			return nil, errors.New("daemon: session manager does not implement observe session source")
		}
		opts := []observe.Option{
			observe.WithRegistry(deps.Registry),
			observe.WithHomePaths(deps.HomePaths),
			observe.WithSessionSource(source),
			observe.WithWorkspaceResolver(deps.WorkspaceResolver),
			observe.WithLogger(deps.Logger),
			observe.WithStartTime(deps.StartedAt),
			observe.WithBridgeSource(bridgeObserveSource(deps.Bridges)),
			observe.WithObservabilityConfig(deps.Config.Observability),
			observe.WithAgentProbeSource(
				agentProbeTargetSource(&deps.Config, deps.AgentCatalog, deps.Logger),
				deps.Config.Observability.AgentProbeTimeoutOrDefault(),
			),
		}
		if deps.MemoryStore != nil {
			opts = append(opts, observe.WithMemoryEventSource(deps.MemoryStore))
		}
		return observe.New(ctx, opts...)
	}
}

func (d *Daemon) applyExtensionManagerFactoryDefault() {
	if d.newExtensionManager != nil {
		return
	}
	d.newExtensionManager = func(deps extensionManagerDeps) extensionRuntime {
		if deps.Registry == nil || deps.ResourceStore == nil || deps.SourceSessions == nil {
			return nil
		}

		capChecker := &extensionpkg.CapabilityChecker{}
		capChecker.SetResourcePolicy(deps.Extensions.Resources)
		hostAPI := extensionpkg.NewHostAPIHandler(
			newHostAPISessionManagerAdapter(deps.Sessions),
			deps.MemoryStore,
			deps.Observer,
			deps.SkillsRegistry,
			buildHostAPIOptions(&deps, capChecker, deps.ResourceStore)...,
		)

		return extensionpkg.NewManager(
			deps.Registry,
			buildExtensionManagerOptions(&deps, capChecker, hostAPI, deps.SourceSessions)...,
		)
	}
}

func buildHostAPIOptions(
	deps *extensionManagerDeps,
	capChecker *extensionpkg.CapabilityChecker,
	resourceStore resources.RawStore,
) []extensionpkg.HostAPIOption {
	opts := []extensionpkg.HostAPIOption{
		extensionpkg.WithHostAPIAutomationGetter(deps.Automation),
		extensionpkg.WithHostAPITaskManager(deps.Tasks),
		extensionpkg.WithHostAPINetworkService(deps.Network),
		extensionpkg.WithHostAPINetworkStore(deps.NetworkStore),
		extensionpkg.WithHostAPIModelCatalogService(deps.ModelCatalog),
		extensionpkg.WithHostAPICapabilityChecker(capChecker),
		extensionpkg.WithHostAPIWorkspaceResolver(deps.WorkspaceResolver),
		extensionpkg.WithHostAPIResourceStore(resourceStore),
		extensionpkg.WithHostAPIResourceCodecRegistry(deps.ResourceCodecs),
		extensionpkg.WithHostAPIResourceTrigger(deps.ResourceTrigger),
		extensionpkg.WithHostAPISoulAuthoring(deps.SoulAuthoring),
		extensionpkg.WithHostAPISoulRefresher(deps.SoulRefresher),
		extensionpkg.WithHostAPIHeartbeatAuthoring(deps.HeartbeatAuthor),
		extensionpkg.WithHostAPIHeartbeatStatus(deps.HeartbeatStatus),
		extensionpkg.WithHostAPIHeartbeatWake(deps.HeartbeatWake),
		extensionpkg.WithHostAPISessionHealth(deps.SessionHealth),
		extensionpkg.WithHostAPIHeartbeatWakeEvents(deps.WakeEvents),
		extensionpkg.WithHostAPIMemoryProviderRegistry(deps.MemoryProviderRegistry),
	}
	if deps.BridgeRegistry != nil {
		opts = append(opts, extensionpkg.WithHostAPIBridgeRegistry(deps.BridgeRegistry))
	}
	if deps.BridgeDedupStore != nil {
		opts = append(opts, extensionpkg.WithHostAPIBridgeDedupStore(deps.BridgeDedupStore))
	}
	if deps.BridgeBroker != nil {
		opts = append(opts, extensionpkg.WithHostAPIDeliveryBroker(deps.BridgeBroker))
	}
	return opts
}

func buildExtensionManagerOptions(
	deps *extensionManagerDeps,
	capChecker *extensionpkg.CapabilityChecker,
	hostAPI *extensionpkg.HostAPIHandler,
	sourceSessions resources.SourceSessionManager,
) []extensionpkg.Option {
	opts := []extensionpkg.Option{
		extensionpkg.WithCapabilityChecker(capChecker),
		extensionpkg.WithLogger(deps.Logger),
		extensionpkg.WithSourceSessionManager(sourceSessions),
		extensionpkg.WithProcessRegistry(deps.ProcessRegistry),
	}
	if sink, ok := deps.Observer.(extensionpkg.BridgeTelemetrySink); ok {
		opts = append(opts, extensionpkg.WithBridgeTelemetrySink(sink))
	}
	if deps.BridgeRuntime != nil {
		opts = append(opts, extensionpkg.WithBridgeRuntimeResolver(deps.BridgeRuntime))
	}
	if deps.SecretResolver != nil {
		opts = append(opts, extensionpkg.WithSecretResolver(deps.SecretResolver))
	}
	for method, handler := range hostAPI.MethodHandlers() {
		opts = append(opts, extensionpkg.WithHostMethodHandler(method, handler))
	}
	return opts
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
	if deps.SkillsRegistry != nil {
		return appendTypedProjectorRegistration(
			registrations,
			deps.CodecRegistry,
			skills.SkillResourceKind,
			newSkillProjector(deps.SkillsRegistry),
		)
	}
	return registrations, nil
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

func httpServerOptions(deps *RuntimeDeps) []httpapi.Option {
	return []httpapi.Option{
		httpapi.WithHomePaths(deps.HomePaths),
		httpapi.WithConfig(&deps.Config),
		httpapi.WithLogger(deps.Logger),
		httpapi.WithStartedAt(deps.StartedAt),
		httpapi.WithSessionManager(deps.Sessions),
		httpapi.WithTaskService(deps.Tasks),
		httpapi.WithNetworkService(deps.Network),
		httpapi.WithNetworkStore(deps.Registry),
		httpapi.WithObserver(deps.Observer),
		httpapi.WithAutomation(deps.Automation),
		httpapi.WithBridgeService(deps.Bridges),
		httpapi.WithBundleService(deps.Bundles),
		httpapi.WithToolRegistry(deps.ToolRegistry),
		httpapi.WithToolsetRegistry(deps.Toolsets),
		httpapi.WithToolApprovalIssuer(deps.ToolApprovals),
		httpapi.WithSettingsService(deps.Settings),
		httpapi.WithSettingsRestartController(deps.SettingsRestart),
		httpapi.WithSettingsUpdateController(deps.SettingsUpdate),
		httpapi.WithSupportBundleService(deps.SupportBundles),
		httpapi.WithVaultService(deps.Vault),
		httpapi.WithResourceService(deps.Resources),
		httpapi.WithWorkspaceResolver(deps.WorkspaceService),
		httpapi.WithAgentCatalog(deps.AgentCatalog),
		httpapi.WithModelCatalogService(deps.ModelCatalog),
		httpapi.WithAgentContext(deps.AgentContext),
		httpapi.WithCoordinatorConfig(deps.CoordinatorConfig),
		httpapi.WithSoulAuthoring(deps.SoulAuthoring),
		httpapi.WithSoulRefresher(deps.SoulRefresher),
		httpapi.WithHeartbeatAuthoring(deps.HeartbeatAuthor),
		httpapi.WithHeartbeatStatus(deps.HeartbeatStatus),
		httpapi.WithHeartbeatWake(deps.HeartbeatWake),
		httpapi.WithSessionHealthReader(deps.SessionHealth),
		httpapi.WithHeartbeatWakeEventReader(deps.WakeEvents),
		httpapi.WithSkillsRegistry(deps.SkillsRegistry),
		httpapi.WithMemoryStore(deps.MemoryStore),
		httpapi.WithDreamTrigger(deps.DreamTrigger),
		httpapi.WithMemoryExtractorService(deps.MemoryExtractor),
		httpapi.WithMemoryProviderService(deps.MemoryProviders),
		httpapi.WithMemorySessionLedgerService(deps.MemorySessionLedger),
		httpapi.WithExtensionService(deps.Extensions),
	}
}

func udsServerOptions(deps *RuntimeDeps) []udsapi.Option {
	return []udsapi.Option{
		udsapi.WithHomePaths(deps.HomePaths),
		udsapi.WithConfig(&deps.Config),
		udsapi.WithLogger(deps.Logger),
		udsapi.WithStartedAt(deps.StartedAt),
		udsapi.WithSessionManager(deps.Sessions),
		udsapi.WithTaskService(deps.Tasks),
		udsapi.WithNetworkService(deps.Network),
		udsapi.WithNetworkStore(deps.Registry),
		udsapi.WithObserver(deps.Observer),
		udsapi.WithAutomation(deps.Automation),
		udsapi.WithBridgeService(deps.Bridges),
		udsapi.WithBundleService(deps.Bundles),
		udsapi.WithToolRegistry(deps.ToolRegistry),
		udsapi.WithToolsetRegistry(deps.Toolsets),
		udsapi.WithToolApprovalIssuer(deps.ToolApprovals),
		udsapi.WithSettingsService(deps.Settings),
		udsapi.WithSettingsRestartController(deps.SettingsRestart),
		udsapi.WithSettingsUpdateController(deps.SettingsUpdate),
		udsapi.WithSupportBundleService(deps.SupportBundles),
		udsapi.WithVaultService(deps.Vault),
		udsapi.WithResourceService(deps.Resources),
		udsapi.WithWorkspaceResolver(deps.WorkspaceService),
		udsapi.WithAgentCatalog(deps.AgentCatalog),
		udsapi.WithModelCatalogService(deps.ModelCatalog),
		udsapi.WithAgentContext(deps.AgentContext),
		udsapi.WithSoulAuthoring(deps.SoulAuthoring),
		udsapi.WithSoulRefresher(deps.SoulRefresher),
		udsapi.WithHeartbeatAuthoring(deps.HeartbeatAuthor),
		udsapi.WithHeartbeatStatus(deps.HeartbeatStatus),
		udsapi.WithHeartbeatWake(deps.HeartbeatWake),
		udsapi.WithSessionHealthReader(deps.SessionHealth),
		udsapi.WithHeartbeatWakeEventReader(deps.WakeEvents),
		udsapi.WithCoordinatorConfig(deps.CoordinatorConfig),
		udsapi.WithSkillsRegistry(deps.SkillsRegistry),
		udsapi.WithMemoryStore(deps.MemoryStore),
		udsapi.WithDreamTrigger(deps.DreamTrigger),
		udsapi.WithMemoryExtractorService(deps.MemoryExtractor),
		udsapi.WithMemoryProviderService(deps.MemoryProviders),
		udsapi.WithMemorySessionLedgerService(deps.MemorySessionLedger),
		udsapi.WithExtensionService(deps.Extensions),
		udsapi.WithHostedMCP(deps.HostedMCP),
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
	return d.shutdownDetached(ctx, d.detachShutdownTargets())
}

func daemonShutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithTimeout(context.WithoutCancel(parent), defaultShutdownTimeout)
}

func (d *Daemon) detachShutdownTargets() shutdownTargets {
	d.mu.Lock()
	defer d.mu.Unlock()

	targets := shutdownTargets{
		scheduler:           d.scheduler,
		spawnReaper:         d.spawnReaper,
		tasks:               d.tasks,
		sessions:            d.sessions,
		network:             d.network,
		hooks:               d.hooks,
		extensions:          d.extensions,
		automation:          d.automation,
		resourceReconcile:   d.resourceReconcile,
		bridges:             d.bridges,
		httpServer:          d.httpServer,
		udsServer:           d.udsServer,
		registry:            d.registry,
		lock:                d.lock,
		closeLogger:         d.closeLogger,
		infoPath:            d.homePaths.DaemonInfo,
		dreamRuntime:        d.dreamRuntime,
		memoryExtractor:     d.memoryExtractor,
		memoryStore:         d.memoryStore,
		localMemoryProvider: d.localMemoryProvider,
		modelCatalog:        d.modelCatalog,
		skillsCancel:        d.skillsCancel,
		skillsDone:          d.skillsDone,
	}
	if stopper, ok := d.observer.(observerRetentionStopper); ok {
		targets.retention = stopper
	}

	d.resetRuntimeStateLocked()
	return targets
}

func (d *Daemon) resetRuntimeStateLocked() {
	d.sessions = nil
	d.tasks = nil
	d.spawnReaper = nil
	d.scheduler = nil
	d.hooks = nil
	d.extensions = nil
	d.automation = nil
	d.resourceReconcile = nil
	d.httpServer = nil
	d.udsServer = nil
	d.observer = nil
	d.registry = nil
	d.harnessResolver = nil
	d.memoryStore = nil
	d.memoryProviderRegistry = nil
	d.memoryExtractor = nil
	d.localMemoryProvider = nil
	d.modelCatalog = nil
	d.skillsRegistry = nil
	d.lock = nil
	d.booting = false
	d.info = Info{}
	d.startedAt = time.Time{}
	d.closeLogger = func() error { return nil }
	d.dreamRuntime = nil
	d.workspaceResolver = nil
	d.sandboxRegistry = nil
	d.skillsCancel = nil
	d.skillsDone = nil
	d.bridges = nil
	d.network = nil
	d.toolRegistry = nil
}

func (d *Daemon) shutdownDetached(ctx context.Context, targets shutdownTargets) error {
	var errs []error
	d.shutdownRuntimeWorkers(ctx, targets, &errs)
	d.shutdownServersAndHooks(ctx, targets, &errs)
	d.shutdownPersistentResources(ctx, targets, &errs)
	return errors.Join(errs...)
}

func (d *Daemon) shutdownRuntimeWorkers(ctx context.Context, targets shutdownTargets, errs *[]error) {
	if targets.dreamRuntime != nil {
		targets.dreamRuntime.Shutdown()
	}
	if targets.memoryExtractor != nil {
		appendWrappedError(errs, "daemon: shutdown memory extractor", targets.memoryExtractor.Close(ctx))
	}
	if targets.memoryStore != nil {
		appendWrappedError(
			errs,
			"daemon: shutdown recall signal recorders",
			targets.memoryStore.CloseRecallSignalRecorders(ctx),
		)
	}
	if targets.modelCatalog != nil {
		appendWrappedError(errs, "daemon: shutdown model catalog", targets.modelCatalog.Shutdown(ctx))
	}
	appendWrappedError(
		errs,
		"daemon: stop skills watcher",
		stopSkillsWatcher(ctx, targets.skillsCancel, targets.skillsDone),
	)
	if targets.resourceReconcile != nil {
		appendWrappedError(errs, "daemon: close resource reconcile driver", targets.resourceReconcile.Close(ctx))
	}
	if targets.extensions != nil {
		appendWrappedError(errs, "daemon: stop extensions", targets.extensions.Stop(ctx))
	}
	if targets.automation != nil {
		appendWrappedError(errs, "daemon: shutdown automation", targets.automation.Shutdown(ctx))
	}
	if targets.retention != nil {
		appendWrappedError(errs, "daemon: shutdown observability retention", targets.retention.ShutdownRetention(ctx))
	}
	if targets.scheduler != nil {
		appendWrappedError(errs, "daemon: shutdown scheduler", targets.scheduler.stopLoop(ctx))
	}
	if targets.spawnReaper != nil {
		appendWrappedError(errs, "daemon: shutdown spawn reaper", targets.spawnReaper.shutdown(ctx))
	}
	if err := d.stopSessions(ctx, targets.sessions); err != nil {
		*errs = append(*errs, err)
	}
	if targets.scheduler != nil {
		appendWrappedError(errs, "daemon: shutdown scheduler wake dispatcher", targets.scheduler.shutdownWaker(ctx))
	}
	if targets.tasks != nil {
		targets.tasks.shutdown()
	}
	if targets.localMemoryProvider != nil {
		appendWrappedError(errs, "daemon: shutdown local memory provider", targets.localMemoryProvider.Shutdown(ctx))
	}
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

func (d *Daemon) shutdownPersistentResources(ctx context.Context, targets shutdownTargets, errs *[]error) {
	if err := RemoveInfo(targets.infoPath); err != nil {
		*errs = append(*errs, err)
	}
	if targets.registry != nil {
		appendWrappedError(errs, "daemon: close global database", targets.registry.Close(ctx))
	}
	if targets.lock != nil {
		if err := targets.lock.Release(); err != nil {
			*errs = append(*errs, err)
		}
	}
	if targets.closeLogger != nil {
		appendWrappedError(errs, "daemon: close logger", targets.closeLogger())
	}
}

func appendWrappedError(errs *[]error, prefix string, err error) {
	if errs == nil || err == nil {
		return
	}
	*errs = append(*errs, fmt.Errorf("%s: %w", prefix, err))
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
