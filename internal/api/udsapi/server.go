// Package udsapi serves the AGH transport API over a Unix domain socket.
package udsapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	core "github.com/compozy/agh/internal/api/core"
	"github.com/compozy/agh/internal/api/ginutil"
	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/doctor"
	mcppkg "github.com/compozy/agh/internal/mcp"
	"github.com/compozy/agh/internal/memory"
	"github.com/compozy/agh/internal/store"
	toolspkg "github.com/compozy/agh/internal/tools"
	workspacepkg "github.com/compozy/agh/internal/workspace"
	"github.com/gin-gonic/gin"
)

const (
	defaultPollInterval      = 100 * time.Millisecond
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	maxSocketPathBytes       = 103
)

var (
	ErrSessionManagerRequired    = errors.New("udsapi: session manager is required")
	ErrTaskServiceRequired       = errors.New("udsapi: task service is required")
	ErrObserverRequired          = errors.New("udsapi: observer is required")
	ErrWorkspaceResolverRequired = errors.New("udsapi: workspace resolver is required")
	ErrSocketPathTooLong         = errors.New("udsapi: socket path too long")
)

type serverState uint8

const (
	serverStateStopped serverState = iota
	serverStateRunning
	serverStateStopping
)

// Option customizes UDS server construction.
type Option func(*Server)

// Server exposes the daemon API over a Unix domain socket.
type Server struct {
	mu sync.Mutex

	homePaths          aghconfig.HomePaths
	config             aghconfig.Config
	configSet          bool
	socketPath         string
	logger             *slog.Logger
	startedAt          time.Time
	now                func() time.Time
	pollInterval       time.Duration
	sessions           core.SessionManager
	drainController    core.DaemonDrainController
	sessionCatalog     core.SessionCatalog
	tasks              core.TaskService
	network            core.NetworkService
	networkStore       core.NetworkStore
	networkUsage       store.NetworkUsageStore
	coordination       workspacepkg.CoordinationCommands
	observer           core.Observer
	schemaStreams      core.SchemaStreamStatusReader
	automation         core.AutomationManager
	loops              core.LoopService
	bridges            core.BridgeService
	notifications      core.NotificationPresetService
	bundles            core.BundleService
	supportBundles     core.SupportBundleService
	tools              core.ToolRegistry
	toolArtifacts      toolspkg.ToolArtifactStore
	toolsets           core.ToolsetRegistry
	toolApprovals      core.ToolApprovalIssuer
	approvalGrants     core.ToolApprovalGrantService
	clarify            toolspkg.ClarifyBroker
	settings           core.SettingsService
	settingsRestart    core.SettingsRestartController
	settingsUpdate     core.SettingsUpdateController
	vault              core.VaultService
	workspaces         core.WorkspaceService
	onboarding         core.OnboardingStore
	agentCatalog       core.AgentCatalog
	agentSync          core.AgentDefinitionSync
	modelCatalog       core.ModelCatalogService
	marketplaceCatalog core.MarketplaceCatalogService
	agentContext       core.AgentContextService
	soulAuthoring      core.SoulAuthoringService
	soulHistoryPurger  core.SoulHistoryPurger
	soulRefresher      core.SoulRefresher
	heartbeatAuthor    core.HeartbeatAuthoringService
	heartbeatPurger    core.HeartbeatHistoryPurger
	heartbeatStatus    core.HeartbeatStatusService
	heartbeatWake      core.HeartbeatWakeService
	sessionHealth      core.SessionHealthReader
	wakeEvents         core.HeartbeatWakeEventReader
	coordinatorConfig  core.CoordinatorConfigResolver
	skillsRegistry     core.SkillsRegistry
	skillResources     core.SkillResourceSyncer
	memoryStore        *memory.Store
	dreamTrigger       core.DreamTrigger
	memoryExtractor    core.MemoryExtractorService
	memoryProviders    core.MemoryProviderService
	memoryLedger       core.MemorySessionLedgerService
	runtimeMemory      doctor.RuntimeMemorySnapshotSource
	deadEntities       doctor.DeadEntitySource
	agentLoader        core.AgentLoader
	udsExtendedServices

	engine       *gin.Engine
	handlers     *Handlers
	httpServer   *http.Server
	listener     net.Listener
	serveDone    chan struct{}
	serveErr     error
	streamCancel context.CancelFunc
	state        serverState
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	*core.BaseHandlers
	Extensions ExtensionService
	HostedMCP  *mcppkg.HostedService
	MCPHostAPI mcppkg.HostAPIInvoker
}

// WithResourceService injects the shared operator-facing desired-state resource service.
func WithResourceService(service core.ResourceService) Option {
	return func(server *Server) {
		server.resources = service
	}
}

// WithBridgeService injects the daemon-owned bridge runtime.
func WithBridgeService(bridges core.BridgeService) Option {
	return func(server *Server) {
		server.bridges = bridges
	}
}

// WithNotificationPresetService injects the daemon-owned notification preset runtime.
func WithNotificationPresetService(service core.NotificationPresetService) Option {
	return func(server *Server) {
		server.notifications = service
	}
}

// WithBundleService injects the daemon-owned bundle runtime.
func WithBundleService(service core.BundleService) Option {
	return func(server *Server) {
		server.bundles = service
	}
}

// WithSupportBundleService injects the daemon-owned support bundle operation service.
func WithSupportBundleService(service core.SupportBundleService) Option {
	return func(server *Server) {
		server.supportBundles = service
	}
}

// WithToolRegistry injects the executable tool registry.
func WithToolRegistry(registry core.ToolRegistry) Option {
	return func(server *Server) {
		server.tools = registry
	}
}

// WithToolsetRegistry injects the named toolset projection registry.
func WithToolsetRegistry(registry core.ToolsetRegistry) Option {
	return func(server *Server) {
		server.toolsets = registry
	}
}

// WithSettingsService injects the daemon-owned settings service.
func WithSettingsService(service core.SettingsService) Option {
	return func(server *Server) {
		server.settings = service
	}
}

// WithSettingsRestartController injects the daemon-owned restart action surface for settings handlers.
func WithSettingsRestartController(controller core.SettingsRestartController) Option {
	return func(server *Server) {
		server.settingsRestart = controller
	}
}

// WithSettingsUpdateController injects the daemon-owned update status surface for settings handlers.
func WithSettingsUpdateController(controller core.SettingsUpdateController) Option {
	return func(server *Server) {
		server.settingsUpdate = controller
	}
}

// WithVaultService injects the daemon-owned vault control surface.
func WithVaultService(service core.VaultService) Option {
	return func(server *Server) {
		server.vault = service
	}
}

// WithWorkspaceResolver injects the runtime workspace resolver/service.
func WithWorkspaceResolver(workspaces core.WorkspaceService) Option {
	return func(server *Server) {
		server.workspaces = workspaces
	}
}

// WithOnboardingStore injects the first-run onboarding completion store.
func WithOnboardingStore(store core.OnboardingStore) Option {
	return func(server *Server) {
		server.onboarding = store
	}
}

// WithSkillsRegistry injects the skills registry surfaced by the daemon.
func WithSkillsRegistry(registry core.SkillsRegistry) Option {
	return func(server *Server) {
		server.skillsRegistry = registry
	}
}

// WithSkillResourceSyncer injects the resource-backed skill syncer surfaced by the daemon.
func WithSkillResourceSyncer(syncer core.SkillResourceSyncer) Option {
	return func(server *Server) {
		server.skillResources = syncer
	}
}

// WithAgentContext injects the bounded agent situation context service.
func WithAgentContext(service core.AgentContextService) Option {
	return func(server *Server) {
		server.agentContext = service
	}
}

// WithSoulAuthoring injects the managed Soul authoring surface.
func WithSoulAuthoring(service core.SoulAuthoringService) Option {
	return func(server *Server) {
		server.soulAuthoring = service
	}
}

// WithSoulRefresher injects the session Soul refresh surface.
func WithSoulRefresher(service core.SoulRefresher) Option {
	return func(server *Server) {
		server.soulRefresher = service
	}
}

// WithHeartbeatAuthoring injects the managed Heartbeat authoring surface.
func WithHeartbeatAuthoring(service core.HeartbeatAuthoringService) Option {
	return func(server *Server) {
		server.heartbeatAuthor = service
	}
}

// WithHeartbeatStatus injects the Heartbeat status/read surface.
func WithHeartbeatStatus(service core.HeartbeatStatusService) Option {
	return func(server *Server) {
		server.heartbeatStatus = service
	}
}

// WithHeartbeatWake injects the manual Heartbeat wake surface.
func WithHeartbeatWake(service core.HeartbeatWakeService) Option {
	return func(server *Server) {
		server.heartbeatWake = service
	}
}

// WithSessionHealthReader injects the metadata-only session health reader.
func WithSessionHealthReader(reader core.SessionHealthReader) Option {
	return func(server *Server) {
		server.sessionHealth = reader
	}
}

// WithHeartbeatWakeEventReader injects the retained Heartbeat wake audit reader.
func WithHeartbeatWakeEventReader(reader core.HeartbeatWakeEventReader) Option {
	return func(server *Server) {
		server.wakeEvents = reader
	}
}

// WithCoordinatorConfig injects the resolved coordinator policy reader.
func WithCoordinatorConfig(resolver core.CoordinatorConfigResolver) Option {
	return func(server *Server) {
		server.coordinatorConfig = resolver
	}
}

// WithAgentLoader overrides agent definition loading.
func WithAgentLoader(loader core.AgentLoader) Option {
	return func(server *Server) {
		server.agentLoader = loader
	}
}

// WithEngine overrides the Gin engine used by the server, mainly for tests.
func WithEngine(engine *gin.Engine) Option {
	return func(server *Server) {
		server.engine = engine
	}
}

// New constructs a Unix domain socket API server.
func New(opts ...Option) (*Server, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("udsapi: resolve home paths: %w", err)
	}

	server := newDefaultServer(homePaths)
	applyOptions(server, opts)
	if err := server.finalize(); err != nil {
		return nil, err
	}

	server.ensureEngine()
	server.handlers = newHandlers(server.handlerConfig())
	registerRoutes(server.engine, server.handlers)

	return server, nil
}

func newDefaultServer(homePaths aghconfig.HomePaths) *Server {
	return &Server{
		homePaths: homePaths,
		config:    aghconfig.DefaultWithHome(homePaths),
		logger:    slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		pollInterval: defaultPollInterval,
		agentLoader:  aghconfig.LoadAgentDef,
	}
}

func applyOptions(server *Server, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}
}

func registerRoutes(engine *gin.Engine, handlers *Handlers) {
	ginutil.QuietDebug(func() {
		RegisterRoutes(engine, handlers)
	})
}

func (s *Server) finalize() error {
	s.applyDefaults()
	if err := s.validateRequired(); err != nil {
		return err
	}
	return s.configureSocketPath()
}

func (s *Server) applyDefaults() {
	if s.logger == nil {
		s.logger = slog.Default()
	}
	if s.now == nil {
		s.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if s.pollInterval <= 0 {
		s.pollInterval = defaultPollInterval
	}
	if s.startedAt.IsZero() {
		s.startedAt = s.now()
	}
	if s.agentLoader == nil {
		s.agentLoader = aghconfig.LoadAgentDef
	}
	if strings.TrimSpace(s.config.Daemon.Socket) == "" {
		s.config.Daemon.Socket = s.homePaths.DaemonSocket
	}
}

func (s *Server) validateRequired() error {
	switch {
	case s.sessions == nil:
		return ErrSessionManagerRequired
	case s.tasks == nil:
		return ErrTaskServiceRequired
	case s.observer == nil:
		return ErrObserverRequired
	case s.workspaces == nil:
		return ErrWorkspaceResolverRequired
	default:
		return nil
	}
}

func (s *Server) configureSocketPath() error {
	if strings.TrimSpace(s.socketPath) == "" {
		s.socketPath = strings.TrimSpace(s.config.Daemon.Socket)
	}
	if strings.TrimSpace(s.socketPath) == "" {
		return errors.New("udsapi: socket path is required")
	}
	if length := len([]byte(s.socketPath)); length > maxSocketPathBytes {
		return fmt.Errorf(
			"%w: %q is %d bytes; use %d bytes or fewer",
			ErrSocketPathTooLong,
			s.socketPath,
			length,
			maxSocketPathBytes,
		)
	}
	return nil
}

func (s *Server) ensureEngine() {
	if s.engine != nil {
		return
	}

	s.engine = ginutil.NewEngine()
	s.engine.Use(gin.Recovery())
}

// Path reports the served Unix domain socket path.
func (s *Server) Path() string {
	if s == nil {
		return ""
	}
	return s.socketPath
}

// Start begins serving the API over the configured Unix domain socket.
func (s *Server) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("udsapi: server is required")
	}
	if ctx == nil {
		return errors.New("udsapi: start context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	socketPath := strings.TrimSpace(s.socketPath)
	if socketPath == "" {
		return errors.New("udsapi: socket path is required")
	}

	s.mu.Lock()
	if s.state != serverStateStopped {
		err := errors.New("udsapi: server already started")
		if s.state == serverStateStopping {
			err = errors.New("udsapi: server shutdown in progress")
		}
		s.mu.Unlock()
		return err
	}
	if err := ensureSocketParentDir(socketPath); err != nil {
		s.mu.Unlock()
		return err
	}
	if err := removeSocketPath(socketPath); err != nil {
		s.mu.Unlock()
		return err
	}

	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(ctx, "unix", socketPath)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("udsapi: listen on %q: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		cleanupErr := cleanupSocketStartFailure(ln, socketPath)
		s.mu.Unlock()
		return errors.Join(fmt.Errorf("udsapi: chmod socket %q: %w", socketPath, err), cleanupErr)
	}

	streamCtx, streamCancel := context.WithCancel(context.WithoutCancel(ctx))
	httpServer := &http.Server{
		Handler:           s.engine,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ConnContext: func(ctx context.Context, conn net.Conn) context.Context {
			peer, err := mcppkg.PeerInfoFromConn(conn)
			return mcppkg.ContextWithPeerInfo(ctx, peer, err)
		},
	}
	serveDone := make(chan struct{})

	s.handlers.setStreamDone(streamCtx.Done())
	s.httpServer = httpServer
	s.listener = ln
	s.serveDone = serveDone
	s.serveErr = nil
	s.streamCancel = streamCancel
	s.state = serverStateRunning
	s.mu.Unlock()

	go func() {
		defer close(serveDone)
		if err := httpServer.Serve(
			ln,
		); err != nil && !errors.Is(err, http.ErrServerClosed) &&
			!errors.Is(err, net.ErrClosed) {
			s.mu.Lock()
			s.serveErr = fmt.Errorf("udsapi: serve socket %q: %w", socketPath, err)
			s.mu.Unlock()
		}
	}()

	return nil
}

func cleanupSocketStartFailure(ln net.Listener, socketPath string) error {
	var errs []error
	if ln != nil {
		if err := ln.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, fmt.Errorf("udsapi: close startup listener: %w", err))
		}
	}
	if err := removeSocketPath(socketPath); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func ensureSocketParentDir(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return errors.New("udsapi: socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(cleanPath), 0o755); err != nil {
		return fmt.Errorf("udsapi: create socket directory for %q: %w", cleanPath, err)
	}
	return nil
}

func removeSocketPath(path string) error {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil
	}

	info, err := os.Lstat(cleanPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("udsapi: stat socket %q: %w", cleanPath, err)
	case info.Mode()&os.ModeSocket == 0:
		return fmt.Errorf("udsapi: existing path %q is not a unix socket", cleanPath)
	}

	if err := os.Remove(cleanPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("udsapi: remove socket %q: %w", cleanPath, err)
	}
	return nil
}

func newHandlers(cfg *handlerConfig) *Handlers {
	if cfg == nil {
		cfg = &handlerConfig{}
	}

	if cfg.pollInterval <= 0 {
		cfg.pollInterval = defaultPollInterval
	}

	return &Handlers{
		BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:                "udsapi",
			MaskInternalErrors:           false,
			IncludeSessionWorkspaceInSSE: true,
			Sessions:                     cfg.sessions,
			DrainController:              cfg.drainController,
			SessionCatalog:               cfg.sessionCatalog,
			Tasks:                        cfg.tasks,
			Network:                      cfg.network,
			NetworkStore:                 cfg.networkStore,
			NetworkUsage:                 cfg.networkUsage,
			Coordination:                 cfg.coordination,
			Observer:                     cfg.observer,
			SchemaStreams:                cfg.schemaStreams,
			Resources:                    cfg.resources,
			DesktopState:                 cfg.desktopState,
			Extensions:                   cfg.extensions,
			Automation:                   cfg.automation,
			Loops:                        cfg.loops,
			Bridges:                      cfg.bridges,
			Notifications:                cfg.notifications,
			Bundles:                      cfg.bundles,
			SupportBundles:               cfg.supportBundles,
			Tools:                        cfg.tools,
			ToolArtifacts:                cfg.toolArtifacts,
			Toolsets:                     cfg.toolsets,
			ToolApprovals:                cfg.toolApprovals,
			ApprovalGrants:               cfg.approvalGrants,
			Clarify:                      cfg.clarify,
			Settings:                     cfg.settings,
			SettingsRestart:              cfg.settingsRestart,
			SettingsUpdate:               cfg.settingsUpdate,
			Vault:                        cfg.vault,
			Workspaces:                   cfg.workspaces,
			Onboarding:                   cfg.onboarding,
			AgentCatalog:                 cfg.agentCatalog,
			AgentDefinitionSync:          cfg.agentSync,
			ModelCatalog:                 cfg.modelCatalog,
			MarketplaceCatalog:           cfg.marketplaceCatalog,
			AgentContextService:          cfg.agentContext,
			SoulAuthoring:                cfg.soulAuthoring,
			SoulHistoryPurger:            cfg.soulHistoryPurger,
			SoulRefresher:                cfg.soulRefresher,
			HeartbeatAuthoring:           cfg.heartbeatAuthor,
			HeartbeatHistoryPurger:       cfg.heartbeatPurger,
			HeartbeatStatus:              cfg.heartbeatStatus,
			HeartbeatWake:                cfg.heartbeatWake,
			SessionHealth:                cfg.sessionHealth,
			HeartbeatWakeEvents:          cfg.wakeEvents,
			CoordinatorConfig:            cfg.coordinatorConfig,
			SkillsRegistry:               cfg.skillsRegistry,
			SkillResources:               cfg.skillResources,
			MemoryStore:                  cfg.memoryStore,
			DreamTrigger:                 cfg.dreamTrigger,
			MemoryExtractor:              cfg.memoryExtractor,
			MemoryProviders:              cfg.memoryProviders,
			MemorySessionLedger:          cfg.memoryLedger,
			RuntimeMemory:                cfg.runtimeMemory,
			DeadEntities:                 cfg.deadEntities,
			HomePaths:                    cfg.homePaths,
			Config:                       cfg.config,
			Logger:                       cfg.logger,
			StartedAt:                    cfg.startedAt,
			Now:                          cfg.now,
			PollInterval:                 cfg.pollInterval,
			AgentLoader:                  cfg.agentLoader,
		}),
		Extensions: cfg.extensions,
		HostedMCP:  cfg.hostedMCP,
		MCPHostAPI: cfg.mcpHostAPI,
	}
}

func (h *Handlers) setStreamDone(done <-chan struct{}) {
	if h != nil && h.BaseHandlers != nil {
		h.SetStreamDone(done)
	}
}
