// Package httpapi serves the AGH transport API over TCP HTTP/SSE.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

const (
	defaultPollInterval      = 100 * time.Millisecond
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

// Option customizes HTTP server construction.
type Option func(*Server)

// Server exposes the daemon API over TCP HTTP.
type Server struct {
	mu sync.Mutex

	homePaths       aghconfig.HomePaths
	config          aghconfig.Config
	configSet       bool
	host            string
	port            int
	logger          *slog.Logger
	startedAt       time.Time
	now             func() time.Time
	pollInterval    time.Duration
	sessions        core.SessionManager
	tasks           core.TaskService
	network         core.NetworkService
	networkStore    core.NetworkStore
	observer        core.Observer
	automation      core.AutomationManager
	bridges         core.BridgeService
	bundles         core.BundleService
	tools           core.ToolRegistry
	toolsets        core.ToolsetRegistry
	toolApprovals   core.ToolApprovalIssuer
	settings        core.SettingsService
	settingsRestart core.SettingsRestartController
	vault           core.VaultService
	workspaces      core.WorkspaceService
	agentCatalog    core.AgentCatalog
	agentContext    core.AgentContextService
	soulAuthoring   core.SoulAuthoringService
	soulRefresher   core.SoulRefresher
	heartbeatAuthor core.HeartbeatAuthoringService
	heartbeatStatus core.HeartbeatStatusService
	heartbeatWake   core.HeartbeatWakeService
	sessionHealth   core.SessionHealthReader
	wakeEvents      core.HeartbeatWakeEventReader
	skillsRegistry  core.SkillsRegistry
	memoryStore     *memory.Store
	dreamTrigger    core.DreamTrigger
	agentLoader     core.AgentLoader
	resources       core.ResourceService
	resourceAuth    []gin.HandlerFunc
	extensions      ExtensionService

	engine       *gin.Engine
	handlers     *Handlers
	httpServer   *http.Server
	listener     net.Listener
	serveDone    chan struct{}
	serveErr     error
	streamCancel context.CancelFunc
	started      bool
	actualPort   int
}

// WithHomePaths overrides the resolved AGH home layout.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(server *Server) {
		server.homePaths = homePaths
		if !server.configSet {
			server.config = aghconfig.DefaultWithHome(homePaths)
		}
	}
}

// WithConfig overrides the runtime configuration used by the server.
func WithConfig(cfg *aghconfig.Config) Option {
	return func(server *Server) {
		if cfg != nil {
			server.config = *cfg
			server.configSet = true
		}
	}
}

// WithHost overrides the HTTP bind host.
func WithHost(host string) Option {
	return func(server *Server) {
		server.host = strings.TrimSpace(host)
	}
}

// WithPort overrides the HTTP bind port.
func WithPort(port int) Option {
	return func(server *Server) {
		server.port = port
	}
}

// WithLogger injects the server logger.
func WithLogger(logger *slog.Logger) Option {
	return func(server *Server) {
		server.logger = logger
	}
}

// WithStartedAt overrides the daemon start time reported by the API.
func WithStartedAt(startedAt time.Time) Option {
	return func(server *Server) {
		server.startedAt = startedAt
	}
}

// WithNow overrides the server clock, mainly for tests.
func WithNow(now func() time.Time) Option {
	return func(server *Server) {
		server.now = now
	}
}

// WithPollInterval overrides the SSE poll cadence.
func WithPollInterval(interval time.Duration) Option {
	return func(server *Server) {
		server.pollInterval = interval
	}
}

// WithSessionManager injects the runtime session manager.
func WithSessionManager(manager core.SessionManager) Option {
	return func(server *Server) {
		server.sessions = manager
	}
}

// WithTaskService injects the daemon-owned task service.
func WithTaskService(service core.TaskService) Option {
	return func(server *Server) {
		server.tasks = service
	}
}

// WithNetworkService injects the runtime network manager.
func WithNetworkService(service core.NetworkService) Option {
	return func(server *Server) {
		server.network = service
	}
}

// WithNetworkStore injects the persisted network query store.
func WithNetworkStore(store core.NetworkStore) Option {
	return func(server *Server) {
		server.networkStore = store
	}
}

// WithObserver injects the runtime observer.
func WithObserver(observer core.Observer) Option {
	return func(server *Server) {
		server.observer = observer
	}
}

// WithAutomation injects the daemon-owned automation manager.
func WithAutomation(manager core.AutomationManager) Option {
	return func(server *Server) {
		server.automation = manager
	}
}

// WithBridgeService injects the daemon-owned bridge runtime.
func WithBridgeService(bridges core.BridgeService) Option {
	return func(server *Server) {
		server.bridges = bridges
	}
}

// WithBundleService injects the daemon-owned bundle runtime.
func WithBundleService(service core.BundleService) Option {
	return func(server *Server) {
		server.bundles = service
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

// WithToolApprovalIssuer injects the local approval-token issuer.
func WithToolApprovalIssuer(issuer core.ToolApprovalIssuer) Option {
	return func(server *Server) {
		server.toolApprovals = issuer
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

// WithMemoryStore injects the memory store surfaced by the daemon.
func WithMemoryStore(store *memory.Store) Option {
	return func(server *Server) {
		server.memoryStore = store
	}
}

// WithSkillsRegistry injects the skills registry surfaced by the daemon.
func WithSkillsRegistry(registry core.SkillsRegistry) Option {
	return func(server *Server) {
		server.skillsRegistry = registry
	}
}

// WithAgentCatalog injects the projected resource-backed agent catalog.
func WithAgentCatalog(catalog core.AgentCatalog) Option {
	return func(server *Server) {
		server.agentCatalog = catalog
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

// WithDreamTrigger injects the dream-consolidation trigger surfaced by the daemon.
func WithDreamTrigger(trigger core.DreamTrigger) Option {
	return func(server *Server) {
		server.dreamTrigger = trigger
	}
}

// WithAgentLoader overrides agent definition loading.
func WithAgentLoader(loader core.AgentLoader) Option {
	return func(server *Server) {
		server.agentLoader = loader
	}
}

// WithResourceService injects the shared operator-facing desired-state resource service.
func WithResourceService(service core.ResourceService) Option {
	return func(server *Server) {
		server.resources = service
	}
}

// WithResourceOperatorAuth gates HTTP resource routes behind explicit operator auth middleware.
func WithResourceOperatorAuth(middleware ...gin.HandlerFunc) Option {
	return func(server *Server) {
		server.resourceAuth = append([]gin.HandlerFunc(nil), middleware...)
	}
}

// WithExtensionService injects daemon-backed extension management handlers.
func WithExtensionService(service ExtensionService) Option {
	return func(server *Server) {
		server.extensions = service
	}
}

// WithEngine overrides the Gin engine used by the server, mainly for tests.
func WithEngine(engine *gin.Engine) Option {
	return func(server *Server) {
		server.engine = engine
	}
}

// New constructs an HTTP API server.
func New(opts ...Option) (*Server, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("httpapi: resolve home paths: %w", err)
	}

	server := newDefaultServer(homePaths)
	applyOptions(server, opts)
	if err := server.finalize(); err != nil {
		return nil, err
	}

	staticFS, err := newStaticFS()
	if err != nil {
		return nil, fmt.Errorf("httpapi: load embedded frontend bundle: %w", err)
	}
	server.ensureEngine()
	server.handlers = newHandlers(server.handlerConfig(staticFS))
	RegisterRoutes(server.engine, server.handlers)

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

func (s *Server) finalize() error {
	s.applyDefaults()
	if err := s.validateRequired(); err != nil {
		return err
	}
	s.configureAddress()
	return nil
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
	if strings.TrimSpace(s.config.HTTP.Host) == "" {
		s.config.HTTP.Host = "localhost"
	}
	if s.config.HTTP.Port <= 0 {
		s.config.HTTP.Port = 2123
	}
}

func (s *Server) validateRequired() error {
	switch {
	case len(s.resourceAuth) > 0 && s.resources == nil:
		return errors.New("httpapi: resource service is required when resource operator auth is configured")
	case s.sessions == nil:
		return errors.New("httpapi: session manager is required")
	case s.tasks == nil:
		return errors.New("httpapi: task service is required")
	case s.observer == nil:
		return errors.New("httpapi: observer is required")
	case s.workspaces == nil:
		return errors.New("httpapi: workspace resolver is required")
	default:
		return nil
	}
}

func (s *Server) configureAddress() {
	if strings.TrimSpace(s.host) == "" {
		s.host = strings.TrimSpace(s.config.HTTP.Host)
	}
	if s.port <= 0 {
		s.port = s.config.HTTP.Port
	}
}

func (s *Server) ensureEngine() {
	if s.engine != nil {
		return
	}

	s.engine = gin.New()
	s.engine.Use(gin.Recovery())
	s.engine.Use(requestLoggingMiddleware(s.logger))
	s.engine.Use(corsMiddleware(s.host))
	s.engine.Use(requestBodyLimitMiddleware(maxAPIRequestBodyBytes))
	s.engine.Use(errorMiddleware())
}

func (s *Server) handlerConfig(staticFS fs.FS) *handlerConfig {
	return &handlerConfig{
		sessions:        s.sessions,
		tasks:           s.tasks,
		network:         s.network,
		networkStore:    s.networkStore,
		observer:        s.observer,
		resources:       s.resources,
		automation:      s.automation,
		bridges:         s.bridges,
		bundles:         s.bundles,
		tools:           s.tools,
		toolsets:        s.toolsets,
		toolApprovals:   s.toolApprovals,
		settings:        s.settings,
		settingsRestart: s.settingsRestart,
		vault:           s.vault,
		workspaces:      s.workspaces,
		agentCatalog:    s.agentCatalog,
		agentContext:    s.agentContext,
		soulAuthoring:   s.soulAuthoring,
		soulRefresher:   s.soulRefresher,
		heartbeatAuthor: s.heartbeatAuthor,
		heartbeatStatus: s.heartbeatStatus,
		heartbeatWake:   s.heartbeatWake,
		sessionHealth:   s.sessionHealth,
		wakeEvents:      s.wakeEvents,
		skillsRegistry:  s.skillsRegistry,
		memoryStore:     s.memoryStore,
		dreamTrigger:    s.dreamTrigger,
		staticFS:        staticFS,
		homePaths:       s.homePaths,
		config:          s.config,
		boundHost:       s.host,
		logger:          s.logger,
		startedAt:       s.startedAt,
		now:             s.now,
		pollInterval:    s.pollInterval,
		agentLoader:     s.agentLoader,
		httpPort:        s.port,
		resourceAuth:    append([]gin.HandlerFunc(nil), s.resourceAuth...),
		extensions:      s.extensions,
	}
}

// Port reports the effective HTTP port.
func (s *Server) Port() int {
	if s == nil {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.actualPort > 0 {
		return s.actualPort
	}
	return s.port
}

// Start begins serving the API over the configured TCP address.
func (s *Server) Start(ctx context.Context) error {
	if s == nil {
		return errors.New("httpapi: server is required")
	}
	if ctx == nil {
		return errors.New("httpapi: start context is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	address := net.JoinHostPort(strings.TrimSpace(s.host), strconv.Itoa(s.port))
	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("httpapi: listen on %q: %w", address, err)
	}

	streamCtx, streamCancel := context.WithCancel(context.Background())
	httpServer := &http.Server{
		Handler:           s.engine,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
	serveDone := make(chan struct{})

	actualPort := s.port
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok && tcpAddr.Port > 0 {
		actualPort = tcpAddr.Port
	}

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		streamCancel()
		_ = ln.Close()
		return errors.New("httpapi: server already started")
	}
	s.handlers.setStreamDone(streamCtx.Done())
	s.handlers.setHTTPPort(actualPort)
	s.httpServer = httpServer
	s.listener = ln
	s.serveDone = serveDone
	s.serveErr = nil
	s.streamCancel = streamCancel
	s.started = true
	s.actualPort = actualPort
	s.mu.Unlock()

	go func() {
		defer close(serveDone)
		if err := httpServer.Serve(
			ln,
		); err != nil && !errors.Is(err, http.ErrServerClosed) &&
			!errors.Is(err, net.ErrClosed) {
			s.mu.Lock()
			s.serveErr = fmt.Errorf("httpapi: serve %q: %w", address, err)
			s.mu.Unlock()
		}
	}()

	return nil
}

// Shutdown stops accepting new requests and drains active ones.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	httpServer := s.httpServer
	listener := s.listener
	serveDone := s.serveDone
	streamCancel := s.streamCancel
	serveErr := s.serveErr
	handlers := s.handlers
	s.httpServer = nil
	s.listener = nil
	s.serveDone = nil
	s.streamCancel = nil
	s.serveErr = nil
	s.started = false
	s.actualPort = 0
	s.mu.Unlock()

	var errs []error
	if streamCancel != nil {
		streamCancel()
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("httpapi: shutdown http server: %w", err))
		}
	}
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, fmt.Errorf("httpapi: close listener: %w", err))
		}
	}
	if serveDone != nil {
		if err := waitForServeDone(ctx, serveDone); err != nil {
			errs = append(errs, err)
		}
	}
	if handlers != nil {
		if err := handlers.waitForPromptDrains(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if serveErr != nil {
		errs = append(errs, serveErr)
	}

	return errors.Join(errs...)
}

func waitForServeDone(ctx context.Context, done <-chan struct{}) error {
	if done == nil {
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("httpapi: wait for serve shutdown: %w", ctx.Err())
	}
}
