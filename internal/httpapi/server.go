// Package httpapi serves the AGH transport API over TCP HTTP/SSE.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

const (
	defaultPollInterval      = 100 * time.Millisecond
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

// Option customizes HTTP server construction.
type Option func(*Server)

// AgentLoader loads one parsed AGENT.md definition.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// SessionManager is the runtime session surface exposed over HTTP.
type SessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	List() []*session.SessionInfo
	ListAll(ctx context.Context) ([]*session.SessionInfo, error)
	Status(ctx context.Context, id string) (*session.SessionInfo, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error)
	Stop(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan session.AgentEvent, error)
	ApprovePermission(ctx context.Context, id string, req session.ApproveRequest) error
}

// Observer is the observability surface exposed over HTTP.
type Observer interface {
	QueryEvents(ctx context.Context, query observe.EventQuery) ([]observe.Event, error)
	Health(ctx context.Context) (observe.Health, error)
}

// Server exposes the daemon API over TCP HTTP.
type Server struct {
	mu sync.Mutex

	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	host         string
	port         int
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	sessions     SessionManager
	observer     Observer
	agentLoader  AgentLoader

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

type handlerConfig struct {
	sessions     SessionManager
	observer     Observer
	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	agentLoader  AgentLoader
	httpPort     int
}

// Handlers expose request/response and SSE endpoints for the AGH API.
type Handlers struct {
	sessions     SessionManager
	observer     Observer
	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	agentLoader  AgentLoader
	streamDone   <-chan struct{}
	httpPort     int
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
func WithSessionManager(manager SessionManager) Option {
	return func(server *Server) {
		server.sessions = manager
	}
}

// WithObserver injects the runtime observer.
func WithObserver(observer Observer) Option {
	return func(server *Server) {
		server.observer = observer
	}
}

// WithAgentLoader overrides agent definition loading.
func WithAgentLoader(loader AgentLoader) Option {
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
	if server.engine == nil {
		server.engine = gin.New()
		server.engine.Use(gin.Recovery())
		server.engine.Use(requestLoggingMiddleware(server.logger))
		server.engine.Use(corsMiddleware(server.host))
		server.engine.Use(errorMiddleware())
	}

	server.handlers = newHandlers(handlerConfig{
		sessions:     server.sessions,
		observer:     server.observer,
		homePaths:    server.homePaths,
		config:       server.config,
		logger:       server.logger,
		startedAt:    server.startedAt,
		now:          server.now,
		pollInterval: server.pollInterval,
		agentLoader:  server.agentLoader,
		httpPort:     server.port,
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

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

	sessions := api.Group("/sessions")
	{
		sessions.GET("", handlers.listSessions)
		sessions.POST("", handlers.createSession)
		sessions.GET("/:id", handlers.getSession)
		sessions.DELETE("/:id", handlers.stopSession)
		sessions.POST("/:id/resume", handlers.resumeSession)
		sessions.POST("/:id/prompt", handlers.promptSession)
		sessions.GET("/:id/events", handlers.sessionEvents)
		sessions.GET("/:id/history", handlers.sessionHistory)
		sessions.GET("/:id/stream", handlers.streamSession)
		sessions.POST("/:id/approve", handlers.approveSession)
	}

	agents := api.Group("/agents")
	{
		agents.GET("", handlers.listAgents)
		agents.GET("/:name", handlers.getAgent)
	}

	observeGroup := api.Group("/observe")
	{
		observeGroup.GET("/events", handlers.observeEvents)
		observeGroup.GET("/events/stream", handlers.streamObserveEvents)
		observeGroup.GET("/health", handlers.health)
	}

	daemonGroup := api.Group("/daemon")
	{
		daemonGroup.GET("/status", handlers.daemonStatus)
	}
}

func newHandlers(cfg handlerConfig) *Handlers {
	logger := cfg.logger
	if logger == nil {
		logger = slog.Default()
	}
	now := cfg.now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	agentLoader := cfg.agentLoader
	if agentLoader == nil {
		agentLoader = aghconfig.LoadAgentDef
	}
	if cfg.pollInterval <= 0 {
		cfg.pollInterval = defaultPollInterval
	}
	if cfg.startedAt.IsZero() {
		cfg.startedAt = now()
	}
	if cfg.httpPort <= 0 {
		cfg.httpPort = cfg.config.HTTP.Port
	}

	return &Handlers{
		sessions:     cfg.sessions,
		observer:     cfg.observer,
		homePaths:    cfg.homePaths,
		config:       cfg.config,
		logger:       logger,
		startedAt:    cfg.startedAt,
		now:          now,
		pollInterval: cfg.pollInterval,
		agentLoader:  agentLoader,
		httpPort:     cfg.httpPort,
	}
}

func (h *Handlers) setStreamDone(done <-chan struct{}) {
	h.streamDone = done
}

func (h *Handlers) setHTTPPort(port int) {
	if port > 0 {
		h.httpPort = port
	}
}

func requestLoggingMiddleware(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}

	return func(c *gin.Context) {
		started := time.Now()
		c.Next()

		logger.Info(
			"httpapi: request",
			"method", c.Request.Method,
			"path", c.FullPath(),
			"status", c.Writer.Status(),
			"latency_ms", time.Since(started).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func corsMiddleware(boundHost string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Headers", "Content-Type, Last-Event-ID, Accept")
		headers.Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		headers.Set("Access-Control-Expose-Headers", "Content-Type, Last-Event-ID, x-vercel-ai-ui-message-stream")
		headers.Set("Vary", "Origin")
		if origin != "" {
			allowedOrigin, ok := resolveAllowedOrigin(origin, c.Request.Host, boundHost)
			if !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, errorPayload{Error: "origin not allowed"})
				return
			}
			headers.Set("Access-Control-Allow-Origin", allowedOrigin)
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func resolveAllowedOrigin(origin string, requestHost string, boundHost string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}

	originHost := canonicalHost(parsed.Hostname())
	requestHostname := canonicalHost(hostOnly(requestHost))
	boundHostname := canonicalHost(hostOnly(boundHost))

	switch {
	case originHost == "" || requestHostname == "":
		return "", false
	case originHost == requestHostname:
		return origin, true
	case isLoopbackHost(originHost) && isLoopbackHost(requestHostname):
		return origin, true
	case boundHostname != "" && !isWildcardHost(boundHostname) && originHost == boundHostname:
		return origin, true
	default:
		return "", false
	}
}

func hostOnly(value string) string {
	host := strings.TrimSpace(value)
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}
	return host
}

func canonicalHost(value string) string {
	return strings.Trim(strings.TrimSpace(value), "[]")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWildcardHost(host string) bool {
	switch host {
	case "", "0.0.0.0", "::":
		return true
	default:
		return false
	}
}

func errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		respondError(c, http.StatusInternalServerError, c.Errors.Last())
	}
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
