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
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

const (
	defaultPollInterval     = 100 * time.Millisecond
	defaultReadHeaderTimout = 5 * time.Second
	defaultIdleTimeout      = 60 * time.Second
)

// Option customizes UDS server construction.
type Option func(*Server)

// AgentLoader loads one parsed AGENT.md definition.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// SessionManager is the runtime session surface exposed over UDS.
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
}

// Observer is the observability surface exposed over UDS.
type Observer interface {
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	Health(ctx context.Context) (observe.Health, error)
}

// DreamTrigger exposes consolidation controls and state to the UDS API.
type DreamTrigger interface {
	Trigger(ctx context.Context, workspace string) (bool, string, error)
	LastConsolidatedAt() (time.Time, error)
	Enabled() bool
}

// Server exposes the daemon API over a Unix domain socket.
type Server struct {
	mu sync.Mutex

	homePaths    aghconfig.HomePaths
	config       aghconfig.Config
	socketPath   string
	logger       *slog.Logger
	startedAt    time.Time
	now          func() time.Time
	pollInterval time.Duration
	sessions     SessionManager
	observer     Observer
	memoryStore  *memory.Store
	dreamTrigger DreamTrigger
	agentLoader  AgentLoader

	engine       *gin.Engine
	handlers     *Handlers
	httpServer   *http.Server
	listener     net.Listener
	serveDone    chan struct{}
	serveErr     error
	streamCancel context.CancelFunc
	started      bool
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

// WithMemoryStore injects the memory store surfaced by the daemon.
func WithMemoryStore(store *memory.Store) Option {
	return func(server *Server) {
		server.memoryStore = store
	}
}

// WithDreamTrigger injects the dream-consolidation trigger surfaced by the daemon.
func WithDreamTrigger(trigger DreamTrigger) Option {
	return func(server *Server) {
		server.dreamTrigger = trigger
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

// New constructs a Unix domain socket API server.
func New(opts ...Option) (*Server, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("udsapi: resolve home paths: %w", err)
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
		return nil, errors.New("udsapi: session manager is required")
	}
	if server.observer == nil {
		return nil, errors.New("udsapi: observer is required")
	}
	if strings.TrimSpace(server.config.Daemon.Socket) == "" {
		server.config.Daemon.Socket = server.homePaths.DaemonSocket
	}
	if strings.TrimSpace(server.socketPath) == "" {
		server.socketPath = strings.TrimSpace(server.config.Daemon.Socket)
	}
	if strings.TrimSpace(server.socketPath) == "" {
		return nil, errors.New("udsapi: socket path is required")
	}
	if server.engine == nil {
		server.engine = gin.New()
		server.engine.Use(gin.Recovery())
	}

	server.handlers = newHandlers(handlerConfig{
		sessions:     server.sessions,
		observer:     server.observer,
		memoryStore:  server.memoryStore,
		dreamTrigger: server.dreamTrigger,
		homePaths:    server.homePaths,
		config:       server.config,
		logger:       server.logger,
		startedAt:    server.startedAt,
		now:          server.now,
		pollInterval: server.pollInterval,
		agentLoader:  server.agentLoader,
	})
	RegisterRoutes(server.engine, server.handlers)

	return server, nil
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

	ln, err := net.Listen("unix", socketPath)
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
		ReadHeaderTimeout: defaultReadHeaderTimout,
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
		if err := httpServer.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
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
