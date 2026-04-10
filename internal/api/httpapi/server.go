// Package httpapi serves the AGH transport API over TCP HTTP/SSE.
package httpapi

import (
	"context"
	"errors"
	"fmt"
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

	homePaths      aghconfig.HomePaths
	config         aghconfig.Config
	host           string
	port           int
	logger         *slog.Logger
	startedAt      time.Time
	now            func() time.Time
	pollInterval   time.Duration
	sessions       core.SessionManager
	observer       core.Observer
	workspaces     core.WorkspaceService
	skillsRegistry core.SkillsRegistry
	memoryStore    *memory.Store
	dreamTrigger   core.DreamTrigger
	agentLoader    core.AgentLoader

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
	}
}

// WithConfig overrides the runtime configuration used by the server.
func WithConfig(cfg aghconfig.Config) Option {
	return func(server *Server) {
		server.config = cfg
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

// WithObserver injects the runtime observer.
func WithObserver(observer core.Observer) Option {
	return func(server *Server) {
		server.observer = observer
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

	server := &Server{
		homePaths: homePaths,
		config:    aghconfig.DefaultWithHome(homePaths),
		logger:    slog.Default(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		pollInterval: defaultPollInterval,
		agentLoader:  aghconfig.LoadAgentDef,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(server)
		}
	}

	if server.logger == nil {
		server.logger = slog.Default()
	}
	if server.now == nil {
		server.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if server.pollInterval <= 0 {
		server.pollInterval = defaultPollInterval
	}
	if server.startedAt.IsZero() {
		server.startedAt = server.now()
	}
	if server.agentLoader == nil {
		server.agentLoader = aghconfig.LoadAgentDef
	}
	if server.sessions == nil {
		return nil, errors.New("httpapi: session manager is required")
	}
	if server.observer == nil {
		return nil, errors.New("httpapi: observer is required")
	}
	if server.workspaces == nil {
		return nil, errors.New("httpapi: workspace resolver is required")
	}
	if strings.TrimSpace(server.config.HTTP.Host) == "" {
		server.config.HTTP.Host = "localhost"
	}
	if server.config.HTTP.Port <= 0 {
		server.config.HTTP.Port = 2123
	}
	if strings.TrimSpace(server.host) == "" {
		server.host = strings.TrimSpace(server.config.HTTP.Host)
	}
	if server.port <= 0 {
		server.port = server.config.HTTP.Port
	}
	staticFS, err := newStaticFS()
	if err != nil {
		return nil, fmt.Errorf("httpapi: load embedded frontend bundle: %w", err)
	}
	if server.engine == nil {
		server.engine = gin.New()
		server.engine.Use(gin.Recovery())
		server.engine.Use(requestLoggingMiddleware(server.logger))
		server.engine.Use(corsMiddleware(server.host))
		server.engine.Use(errorMiddleware())
	}

	server.handlers = newHandlers(handlerConfig{
		sessions:       server.sessions,
		observer:       server.observer,
		workspaces:     server.workspaces,
		skillsRegistry: server.skillsRegistry,
		memoryStore:    server.memoryStore,
		dreamTrigger:   server.dreamTrigger,
		staticFS:       staticFS,
		homePaths:      server.homePaths,
		config:         server.config,
		logger:         server.logger,
		startedAt:      server.startedAt,
		now:            server.now,
		pollInterval:   server.pollInterval,
		agentLoader:    server.agentLoader,
		httpPort:       server.port,
	})
	RegisterRoutes(server.engine, server.handlers)

	return server, nil
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
	ln, err := net.Listen("tcp", address)
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
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
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
