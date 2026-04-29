package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/sandbox"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	defaultLifecycleTimeout = 5 * time.Second
	defaultPromptBufferSize = 128
)

var (
	// ErrSessionNotFound reports that the requested active session does not exist.
	ErrSessionNotFound = errors.New("session: session not found")
	// ErrSessionNotActive reports that a known session cannot accept live approvals or prompts.
	ErrSessionNotActive = errors.New("session: session is not active")
	// ErrMaxSessionsReached reports that the active plus pending session count hit the configured limit.
	ErrMaxSessionsReached = errors.New("session: max sessions reached")
	// ErrPendingPermissionNotFound reports that no waiting permission matched the approval request.
	ErrPendingPermissionNotFound = errors.New("session: pending permission not found")
	// ErrPendingPermissionConflict reports that the approval request matched multiple pending permissions.
	ErrPendingPermissionConflict = errors.New("session: pending permission lookup is ambiguous")
)

// CreateOpts defines the inputs required to create a new session.
type CreateOpts struct {
	AgentName     string
	Provider      string
	Name          string
	Workspace     string
	WorkspacePath string
	Channel       string
	PromptOverlay string
	Type          Type
	Lineage       *store.SessionLineage
}

// StoreOpener opens the per-session events store for a session directory.
type StoreOpener func(ctx context.Context, sessionID string, path string) (EventRecorder, error)

// IDGenerator returns unique identifiers for sessions and prompt turns.
type IDGenerator func() string

// HostedMCPLauncher mints and releases session-bound hosted MCP launch records.
type HostedMCPLauncher interface {
	Launch(ctx context.Context, req HostedMCPLaunchRequest) (aghconfig.MCPServer, error)
	CancelLaunch(sessionID string)
	ReleaseSession(sessionID string)
}

// HostedMCPLaunchRequest describes the session identity for a hosted MCP entry.
type HostedMCPLaunchRequest struct {
	SessionID   string
	WorkspaceID string
	AgentName   string
}

// Option customizes the session manager.
type Option func(*Manager)

// Manager owns active session lifecycle and runtime orchestration.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	pending    map[string]struct{}
	finalizing map[string]chan struct{}
	spawnMu    sync.Mutex

	syntheticMu          sync.Mutex
	syntheticQueues      map[string][]queuedSyntheticPrompt
	syntheticDispatching map[string]bool

	logger          *slog.Logger
	driver          AgentDriver
	notifier        Notifier
	networkPeers    NetworkPeerLifecycle
	turnEndNotifier TurnEndNotifier
	inputAugmenter  PromptInputAugmenter
	startupOverlay  StartupPromptOverlay
	hooks           HookSet
	sandbox         *sandbox.Registry
	agentResolver   AgentResolver
	skillRegistry   SkillRegistry
	mcpResolver     MCPResolver
	hostedMCP       HostedMCPLauncher
	homePaths       aghconfig.HomePaths
	workspace       workspacepkg.RuntimeResolver
	openStore       StoreOpener
	assembler       PromptAssembler
	supervision     aghconfig.SessionSupervisionConfig
	lifecycleCtx    context.Context
	now             func() time.Time
	newSessionID    IDGenerator
	newSandboxID    IDGenerator
	newTurnID       IDGenerator
	maxSessions     int
	promptBufSize   int
}

// WithSandboxRegistry injects the runtime sandbox provider registry.
func WithSandboxRegistry(registry *sandbox.Registry) Option {
	return func(manager *Manager) {
		manager.sandbox = registry
	}
}

// WithDriver injects the runtime driver used for session lifecycle operations.
func WithDriver(driver AgentDriver) Option {
	return func(manager *Manager) {
		manager.driver = driver
	}
}

// WithStore injects the opener used to create per-session event recorders.
func WithStore(opener StoreOpener) Option {
	return func(manager *Manager) {
		manager.openStore = opener
	}
}

// WithPromptAssembler injects prompt assembly for session startup.
func WithPromptAssembler(assembler PromptAssembler) Option {
	return func(manager *Manager) {
		manager.assembler = assembler
	}
}

// WithLifecycleContext injects the daemon-owned lifecycle context used by background goroutines.
func WithLifecycleContext(ctx context.Context) Option {
	return func(manager *Manager) {
		manager.lifecycleCtx = ctx
	}
}

// WithNotifier injects the async notification fan-out implementation.
func WithNotifier(notifier Notifier) Option {
	return func(manager *Manager) {
		manager.notifier = notifier
	}
}

// WithHookSet injects the grouped hook dispatch surface used by the session
// manager for lifecycle and runtime hook points.
func WithHookSet(hooks HookSet) Option {
	return func(manager *Manager) {
		manager.hooks = hooks
	}
}

// WithSkillRegistry injects the active-skill registry used during session start.
func WithSkillRegistry(registry SkillRegistry) Option {
	return func(manager *Manager) {
		manager.skillRegistry = registry
	}
}

// WithAgentResolver injects the daemon-authoritative agent definition resolver.
func WithAgentResolver(resolver AgentResolver) Option {
	return func(manager *Manager) {
		manager.agentResolver = resolver
	}
}

// WithMCPResolver injects the skill MCP resolver used during session start.
func WithMCPResolver(resolver MCPResolver) Option {
	return func(manager *Manager) {
		manager.mcpResolver = resolver
	}
}

// WithHostedMCPLauncher injects the session-bound AGH-hosted MCP launcher.
func WithHostedMCPLauncher(launcher HostedMCPLauncher) Option {
	return func(manager *Manager) {
		manager.hostedMCP = launcher
	}
}

// WithLogger injects the logger used by the session manager.
func WithLogger(logger *slog.Logger) Option {
	return func(manager *Manager) {
		manager.logger = logger
	}
}

// WithHomePaths overrides the resolved AGH home directory layout.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(manager *Manager) {
		manager.homePaths = homePaths
	}
}

// WithWorkspaceResolver injects workspace resolution for create/resume flows.
func WithWorkspaceResolver(resolver workspacepkg.RuntimeResolver) Option {
	return func(manager *Manager) {
		manager.workspace = resolver
	}
}

// WithPromptInputAugmenter injects a bounded pre-dispatch message augmenter.
func WithPromptInputAugmenter(augmenter PromptInputAugmenter) Option {
	return func(manager *Manager) {
		manager.inputAugmenter = augmenter
	}
}

// WithStartupPromptOverlay injects a daemon-owned startup prompt overlay.
func WithStartupPromptOverlay(overlay StartupPromptOverlay) Option {
	return func(manager *Manager) {
		manager.startupOverlay = overlay
	}
}

// WithNow overrides the manager clock, mainly for tests.
func WithNow(now func() time.Time) Option {
	return func(manager *Manager) {
		manager.now = now
	}
}

// WithSessionIDGenerator overrides session id allocation.
func WithSessionIDGenerator(generator IDGenerator) Option {
	return func(manager *Manager) {
		manager.newSessionID = generator
	}
}

// WithSandboxIDGenerator overrides sandbox id allocation.
func WithSandboxIDGenerator(generator IDGenerator) Option {
	return func(manager *Manager) {
		manager.newSandboxID = generator
	}
}

// WithTurnIDGenerator overrides prompt turn id allocation.
func WithTurnIDGenerator(generator IDGenerator) Option {
	return func(manager *Manager) {
		manager.newTurnID = generator
	}
}

// WithMaxSessions overrides the config-derived max session limit.
func WithMaxSessions(limit int) Option {
	return func(manager *Manager) {
		manager.maxSessions = limit
	}
}

// WithPromptBufferSize overrides the size of the returned prompt event buffer.
func WithPromptBufferSize(size int) Option {
	return func(manager *Manager) {
		manager.promptBufSize = size
	}
}

// WithSessionSupervision overrides runtime activity supervision settings.
func WithSessionSupervision(config aghconfig.SessionSupervisionConfig) Option {
	return func(manager *Manager) {
		manager.supervision = config
	}
}

// NewManager constructs a session manager with sensible defaults.
func NewManager(opts ...Option) (*Manager, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("session: resolve home paths: %w", err)
	}

	manager := &Manager{
		sessions:             make(map[string]*Session),
		pending:              make(map[string]struct{}),
		finalizing:           make(map[string]chan struct{}),
		syntheticQueues:      make(map[string][]queuedSyntheticPrompt),
		syntheticDispatching: make(map[string]bool),
		logger:               slog.Default(),
		driver:               NewACPDriverAdapter(acp.New()),
		homePaths:            homePaths,
		openStore: func(ctx context.Context, sessionID string, path string) (EventRecorder, error) {
			return sessiondb.OpenSessionDB(ctx, sessionID, path)
		},
		supervision:  aghconfig.DefaultSessionSupervisionConfig(),
		lifecycleCtx: context.Background(),
		now: func() time.Time {
			return time.Now().UTC()
		},
		newSessionID: func() string {
			return newID("sess")
		},
		newSandboxID: func() string {
			return newID("env")
		},
		newTurnID: func() string {
			return newID("turn")
		},
		promptBufSize: defaultPromptBufferSize,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(manager)
		}
	}

	if err := manager.applyRuntimeDefaults(); err != nil {
		return nil, err
	}
	if err := aghconfig.EnsureHomeLayout(manager.homePaths); err != nil {
		return nil, fmt.Errorf("session: ensure home layout: %w", err)
	}

	return manager, nil
}

func (m *Manager) applyRuntimeDefaults() error {
	if m.logger == nil {
		m.logger = slog.Default()
	}
	if m.driver == nil {
		return errors.New("session: agent driver is required")
	}
	if m.openStore == nil {
		return errors.New("session: store opener is required")
	}
	if m.lifecycleCtx == nil {
		m.lifecycleCtx = context.Background()
	}
	if m.now == nil {
		m.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if m.newSessionID == nil {
		m.newSessionID = func() string {
			return newID("sess")
		}
	}
	if m.newSandboxID == nil {
		m.newSandboxID = func() string {
			return newID("env")
		}
	}
	if m.newTurnID == nil {
		m.newTurnID = func() string {
			return newID("turn")
		}
	}
	if m.promptBufSize <= 0 {
		m.promptBufSize = defaultPromptBufferSize
	}
	if m.supervision == (aghconfig.SessionSupervisionConfig{}) {
		m.supervision = aghconfig.DefaultSessionSupervisionConfig()
	}
	if err := m.supervision.Validate(); err != nil {
		return fmt.Errorf("session: %w", err)
	}
	return nil
}

// Get returns the active in-memory session by id.
func (m *Manager) Get(id string) (*Session, bool) {
	target := strings.TrimSpace(id)
	if target == "" {
		return nil, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[target]
	return session, ok
}

func (m *Manager) isPending(id string) bool {
	target := strings.TrimSpace(id)
	if target == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.pending[target]
	return ok
}

// SetNetworkPeerLifecycle installs the late-bound network join/leave callbacks
// used after session activation and before final stop cleanup.
func (m *Manager) SetNetworkPeerLifecycle(lifecycle NetworkPeerLifecycle) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.networkPeers = lifecycle
}

// SetTurnEndNotifier installs a post-construction callback invoked after each
// prompt turn finishes.
func (m *Manager) SetTurnEndNotifier(fn TurnEndNotifier) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.turnEndNotifier = fn
}

// IsPrompting reports whether the target session currently has an in-flight
// prompt setup or active turn.
func (m *Manager) IsPrompting(id string) bool {
	session, ok := m.Get(id)
	if !ok {
		return false
	}
	return session.IsPrompting()
}

func (m *Manager) currentNetworkPeerLifecycle() NetworkPeerLifecycle {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.networkPeers
}

func (m *Manager) currentTurnEndNotifier() TurnEndNotifier {
	if m == nil {
		return nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.turnEndNotifier
}

// List returns active in-memory sessions in stable order.
func (m *Manager) List() []*Info {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	infos := make([]*Info, 0, len(sessions))
	for _, session := range sessions {
		infos = append(infos, session.Info())
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].CreatedAt.Equal(infos[j].CreatedAt) {
			return infos[i].ID < infos[j].ID
		}
		return infos[i].CreatedAt.Before(infos[j].CreatedAt)
	})

	return infos
}

func (m *Manager) lookup(id string) (*Session, error) {
	target := strings.TrimSpace(id)
	if target == "" {
		return nil, errors.New("session: session id is required")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[target]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, target)
	}
	return session, nil
}

func (m *Manager) reserve(id string, maxSessions int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; ok {
		return fmt.Errorf("session: session %q is already active", id)
	}
	if _, ok := m.pending[id]; ok {
		return fmt.Errorf("session: session %q is already pending", id)
	}

	active := len(m.sessions) + len(m.pending)
	if active >= maxSessions {
		return maxSessionsReachedError{active: active, limit: maxSessions}
	}

	m.pending[id] = struct{}{}
	return nil
}

func (m *Manager) activate(session *Session) error {
	if session == nil {
		return errors.New("session: session is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.pending, session.ID)
	if _, ok := m.sessions[session.ID]; ok {
		return fmt.Errorf("session: session %q is already active", session.ID)
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *Manager) releaseReservation(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pending, id)
}

func (m *Manager) remove(id string) {
	target := strings.TrimSpace(id)

	m.mu.Lock()
	if done, ok := m.finalizing[target]; ok {
		close(done)
	}
	delete(m.sessions, target)
	delete(m.pending, target)
	delete(m.finalizing, target)
	m.mu.Unlock()

	m.emitDroppedSyntheticPrompts(m.takeQueuedSyntheticPrompts(target), ErrSessionNotFound)
}

func (m *Manager) removeActive(id string) {
	target := strings.TrimSpace(id)

	m.mu.Lock()
	delete(m.sessions, target)
	delete(m.pending, target)
	m.mu.Unlock()

	m.emitDroppedSyntheticPrompts(m.takeQueuedSyntheticPrompts(target), ErrSessionNotActive)
}

func (m *Manager) takeQueuedSyntheticPrompts(sessionID string) []queuedSyntheticPrompt {
	if m == nil {
		return nil
	}

	target := strings.TrimSpace(sessionID)
	if target == "" {
		return nil
	}

	m.syntheticMu.Lock()
	defer m.syntheticMu.Unlock()

	queue := append([]queuedSyntheticPrompt(nil), m.syntheticQueues[target]...)
	delete(m.syntheticQueues, target)
	delete(m.syntheticDispatching, target)
	return queue
}

func (m *Manager) emitDroppedSyntheticPrompts(items []queuedSyntheticPrompt, err error) {
	for _, item := range items {
		m.emitQueuedSyntheticDispatchError(item, err)
	}
}

func (m *Manager) finishFinalization(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if done, ok := m.finalizing[id]; ok {
		close(done)
	}
	delete(m.finalizing, id)
}

func (m *Manager) claimFinalization(session *Session) (bool, <-chan struct{}) {
	if session == nil {
		return false, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if done, ok := m.finalizing[session.ID]; ok {
		return false, done
	}

	current, ok := m.sessions[session.ID]
	if !ok || current != session {
		return false, nil
	}

	done := make(chan struct{})
	m.finalizing[session.ID] = done
	return true, done
}

// WaitForFinalizations blocks until all in-flight finalization routines finish.
func (m *Manager) WaitForFinalizations(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("session: wait for finalizations context is required")
	}

	for {
		m.mu.RLock()
		pending := make([]<-chan struct{}, 0, len(m.finalizing))
		for _, done := range m.finalizing {
			if done != nil {
				pending = append(pending, done)
			}
		}
		m.mu.RUnlock()

		if len(pending) == 0 {
			return nil
		}

		for _, done := range pending {
			select {
			case <-done:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

type maxSessionsReachedError struct {
	active int
	limit  int
}

func (e maxSessionsReachedError) Error() string {
	return fmt.Sprintf("session: max sessions reached (%d/%d)", e.active, e.limit)
}

func (e maxSessionsReachedError) Is(target error) bool {
	return target == ErrMaxSessionsReached
}
