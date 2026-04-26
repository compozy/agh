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

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

const (
	defaultPollInterval      = 100 * time.Millisecond
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

var (
	ErrSessionManagerRequired    = errors.New("udsapi: session manager is required")
	ErrTaskServiceRequired       = errors.New("udsapi: task service is required")
	ErrObserverRequired          = errors.New("udsapi: observer is required")
	ErrWorkspaceResolverRequired = errors.New("udsapi: workspace resolver is required")
)

// Option customizes UDS server construction.
type Option func(*Server)

// ExtensionService exposes daemon-backed extension management to the UDS API.
type ExtensionService interface {
	List(ctx context.Context) ([]contract.ExtensionPayload, error)
	Install(ctx context.Context, req contract.InstallExtensionRequest) (contract.ExtensionPayload, error)
	Enable(ctx context.Context, name string) (contract.ExtensionPayload, error)
	Disable(ctx context.Context, name string) (contract.ExtensionPayload, error)
	Status(ctx context.Context, name string) (contract.ExtensionPayload, error)
}

// Server exposes the daemon API over a Unix domain socket.
type Server struct {
	mu sync.Mutex

	homePaths       aghconfig.HomePaths
	config          aghconfig.Config
	socketPath      string
	logger          *slog.Logger
	startedAt       time.Time
	now             func() time.Time
	pollInterval    time.Duration
	sessions        core.SessionManager
	tasks           core.TaskService
	network         core.NetworkService
	networkStore    core.NetworkStore
	observer        core.Observer
	resources       core.ResourceService
	automation      core.AutomationManager
	bridges         core.BridgeService
	bundles         core.BundleService
	settings        core.SettingsService
	settingsRestart core.SettingsRestartController
	workspaces      core.WorkspaceService
	agentCatalog    core.AgentCatalog
	agentContext    core.AgentContextService
	skillsRegistry  core.SkillsRegistry
	memoryStore     *memory.Store
	dreamTrigger    core.DreamTrigger
	agentLoader     core.AgentLoader
	extensions      ExtensionService

	engine       *gin.Engine
	handlers     *Handlers
	httpServer   *http.Server
	listener     net.Listener
	serveDone    chan struct{}
	serveErr     error
	streamCancel context.CancelFunc
	started      bool
}

type handlerConfig struct {
	sessions        core.SessionManager
	tasks           core.TaskService
	network         core.NetworkService
	networkStore    core.NetworkStore
	observer        core.Observer
	resources       core.ResourceService
	automation      core.AutomationManager
	bridges         core.BridgeService
	bundles         core.BundleService
	settings        core.SettingsService
	settingsRestart core.SettingsRestartController
	workspaces      core.WorkspaceService
	agentCatalog    core.AgentCatalog
	agentContext    core.AgentContextService
	skillsRegistry  core.SkillsRegistry
	memoryStore     *memory.Store
	dreamTrigger    core.DreamTrigger
	homePaths       aghconfig.HomePaths
	config          aghconfig.Config
	logger          *slog.Logger
	startedAt       time.Time
	now             func() time.Time
	pollInterval    time.Duration
	agentLoader     core.AgentLoader
	extensions      ExtensionService
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	*core.BaseHandlers
	Extensions    ExtensionService
	promptDrainWG sync.WaitGroup
}

// WithHomePaths overrides the resolved AGH home layout.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(server *Server) {
		server.homePaths = homePaths
	}
}

// WithConfig overrides the runtime configuration used by the server.
func WithConfig(cfg *aghconfig.Config) Option {
	return func(server *Server) {
		if cfg != nil {
			server.config = *cfg
		}
	}
}

// WithSocketPath overrides the Unix socket path served by the API.
func WithSocketPath(path string) Option {
	return func(server *Server) {
		server.socketPath = strings.TrimSpace(path)
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

// WithResourceService injects the shared operator-facing desired-state resource service.
func WithResourceService(service core.ResourceService) Option {
	return func(server *Server) {
		server.resources = service
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
	return nil
}

func (s *Server) ensureEngine() {
	if s.engine != nil {
		return
	}

	s.engine = gin.New()
	s.engine.Use(gin.Recovery())
}

func (s *Server) handlerConfig() *handlerConfig {
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
		settings:        s.settings,
		settingsRestart: s.settingsRestart,
		workspaces:      s.workspaces,
		agentCatalog:    s.agentCatalog,
		agentContext:    s.agentContext,
		skillsRegistry:  s.skillsRegistry,
		memoryStore:     s.memoryStore,
		dreamTrigger:    s.dreamTrigger,
		homePaths:       s.homePaths,
		config:          s.config,
		logger:          s.logger,
		startedAt:       s.startedAt,
		now:             s.now,
		pollInterval:    s.pollInterval,
		agentLoader:     s.agentLoader,
		extensions:      s.extensions,
	}
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
	if err := ensureSocketParentDir(socketPath); err != nil {
		return err
	}
	if err := removeSocketPath(socketPath); err != nil {
		return err
	}

	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(ctx, "unix", socketPath)
	if err != nil {
		return fmt.Errorf("udsapi: listen on %q: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return fmt.Errorf("udsapi: chmod socket %q: %w", socketPath, err)
	}

	streamCtx, streamCancel := context.WithCancel(context.Background())
	httpServer := &http.Server{
		Handler:           s.engine,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
	serveDone := make(chan struct{})

	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		streamCancel()
		_ = ln.Close()
		_ = os.Remove(socketPath)
		return errors.New("udsapi: server already started")
	}
	s.handlers.setStreamDone(streamCtx.Done())
	s.httpServer = httpServer
	s.listener = ln
	s.serveDone = serveDone
	s.serveErr = nil
	s.streamCancel = streamCancel
	s.started = true
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

// Shutdown stops accepting new requests, drains active ones, and removes the socket file.
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
	socketPath := s.socketPath
	serveErr := s.serveErr
	s.httpServer = nil
	s.listener = nil
	s.serveDone = nil
	s.streamCancel = nil
	s.serveErr = nil
	s.started = false
	s.mu.Unlock()

	var errs []error
	if streamCancel != nil {
		streamCancel()
	}
	if httpServer != nil {
		if err := httpServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("udsapi: shutdown http server: %w", err))
		}
	}
	if listener != nil {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, fmt.Errorf("udsapi: close listener: %w", err))
		}
	}
	if serveDone != nil {
		if err := waitForServeDone(ctx, serveDone); err != nil {
			errs = append(errs, err)
		}
	}
	if s.handlers != nil {
		if err := s.handlers.waitForPromptDrains(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := removeSocketPath(socketPath); err != nil {
		errs = append(errs, err)
	}
	if serveErr != nil {
		errs = append(errs, serveErr)
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

func waitForServeDone(ctx context.Context, done <-chan struct{}) error {
	if done == nil {
		return nil
	}

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("udsapi: wait for serve shutdown: %w", ctx.Err())
	}
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
			Tasks:                        cfg.tasks,
			Network:                      cfg.network,
			NetworkStore:                 cfg.networkStore,
			Observer:                     cfg.observer,
			Resources:                    cfg.resources,
			Automation:                   cfg.automation,
			Bridges:                      cfg.bridges,
			Bundles:                      cfg.bundles,
			Settings:                     cfg.settings,
			SettingsRestart:              cfg.settingsRestart,
			Workspaces:                   cfg.workspaces,
			AgentCatalog:                 cfg.agentCatalog,
			AgentContextService:          cfg.agentContext,
			SkillsRegistry:               cfg.skillsRegistry,
			MemoryStore:                  cfg.memoryStore,
			DreamTrigger:                 cfg.dreamTrigger,
			HomePaths:                    cfg.homePaths,
			Config:                       cfg.config,
			Logger:                       cfg.logger,
			StartedAt:                    cfg.startedAt,
			Now:                          cfg.now,
			PollInterval:                 cfg.pollInterval,
			AgentLoader:                  cfg.agentLoader,
		}),
		Extensions: cfg.extensions,
	}
}

func (h *Handlers) setStreamDone(done <-chan struct{}) {
	if h != nil && h.BaseHandlers != nil {
		h.SetStreamDone(done)
	}
}
