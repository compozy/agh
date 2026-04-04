package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
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
	AgentName string
	Name      string
	Workspace string
	Type      SessionType
}

// ConfigLoader resolves the effective runtime config for a workspace.
type ConfigLoader func(workspace string) (aghconfig.Config, error)

// AgentLoader loads a parsed AGENT.md definition by name.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// StoreOpener opens the per-session events store for a session directory.
type StoreOpener func(ctx context.Context, sessionID string, path string) (EventRecorder, error)

// IDGenerator returns unique identifiers for sessions and prompt turns.
type IDGenerator func() string

// Option customizes the session manager.
type Option func(*Manager)

// Manager owns active session lifecycle and runtime orchestration.
type Manager struct {
	mu         sync.RWMutex
	sessions   map[string]*Session
	pending    map[string]struct{}
	finalizing map[string]struct{}

	logger        *slog.Logger
	driver        AgentDriver
	notifier      Notifier
	homePaths     aghconfig.HomePaths
	loadConfig    ConfigLoader
	loadAgent     AgentLoader
	openStore     StoreOpener
	assembler     PromptAssembler
	now           func() time.Time
	newSessionID  IDGenerator
	newTurnID     IDGenerator
	maxSessions   int
	promptBufSize int
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

// WithNotifier injects the async notification fan-out implementation.
func WithNotifier(notifier Notifier) Option {
	return func(manager *Manager) {
		manager.notifier = notifier
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

// WithConfigLoader overrides workspace config resolution.
func WithConfigLoader(loader ConfigLoader) Option {
	return func(manager *Manager) {
		manager.loadConfig = loader
	}
}

// WithConfig injects a static runtime config for all session operations.
func WithConfig(cfg aghconfig.Config) Option {
	return func(manager *Manager) {
		manager.loadConfig = func(string) (aghconfig.Config, error) {
			return cfg, nil
		}
	}
}

// WithAgentLoader overrides agent definition resolution.
func WithAgentLoader(loader AgentLoader) Option {
	return func(manager *Manager) {
		manager.loadAgent = loader
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

// NewManager constructs a session manager with sensible defaults.
func NewManager(opts ...Option) (*Manager, error) {
	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("session: resolve home paths: %w", err)
	}

	manager := &Manager{
		sessions:   make(map[string]*Session),
		pending:    make(map[string]struct{}),
		finalizing: make(map[string]struct{}),
		logger:     slog.Default(),
		driver:     NewACPDriverAdapter(acp.New()),
		homePaths:  homePaths,
		loadConfig: func(workspace string) (aghconfig.Config, error) {
			if strings.TrimSpace(workspace) == "" {
				return aghconfig.Load()
			}
			return aghconfig.Load(aghconfig.WithWorkspaceRoot(workspace))
		},
		loadAgent: aghconfig.LoadAgentDef,
		openStore: func(ctx context.Context, sessionID string, path string) (EventRecorder, error) {
			return store.OpenSessionDB(ctx, sessionID, path)
		},
		now: func() time.Time {
			return time.Now().UTC()
		},
		newSessionID: func() string {
			return newID("sess")
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

	if manager.logger == nil {
		manager.logger = slog.Default()
	}
	if manager.driver == nil {
		return nil, errors.New("session: agent driver is required")
	}
	if manager.loadConfig == nil {
		return nil, errors.New("session: config loader is required")
	}
	if manager.loadAgent == nil {
		return nil, errors.New("session: agent loader is required")
	}
	if manager.openStore == nil {
		return nil, errors.New("session: store opener is required")
	}
	if manager.now == nil {
		manager.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if manager.newSessionID == nil {
		manager.newSessionID = func() string {
			return newID("sess")
		}
	}
	if manager.newTurnID == nil {
		manager.newTurnID = func() string {
			return newID("turn")
		}
	}
	if manager.promptBufSize <= 0 {
		manager.promptBufSize = defaultPromptBufferSize
	}
	if err := aghconfig.EnsureHomeLayout(manager.homePaths); err != nil {
		return nil, fmt.Errorf("session: ensure home layout: %w", err)
	}

	return manager, nil
}

// Create resolves an agent definition, opens the session store, and starts a new runtime session.
func (m *Manager) Create(ctx context.Context, opts CreateOpts) (_ *Session, err error) {
	if ctx == nil {
		return nil, errors.New("session: create context is required")
	}

	agentName := strings.TrimSpace(opts.AgentName)
	if agentName == "" {
		return nil, errors.New("session: agent name is required")
	}

	workspace, err := resolveWorkspace(opts.Workspace)
	if err != nil {
		return nil, err
	}

	cfg, err := m.loadConfig(workspace)
	if err != nil {
		return nil, fmt.Errorf("session: load config for %q: %w", workspace, err)
	}

	agentDef, err := m.loadAgent(agentName, m.homePaths)
	if err != nil {
		return nil, fmt.Errorf("session: load agent %q: %w", agentName, err)
	}
	if m.assembler != nil {
		assembledPrompt, assembleErr := m.assembler.Assemble(ctx, agentDef, workspace)
		if assembleErr != nil {
			return nil, fmt.Errorf("session: assemble prompt for %q: %w", agentName, assembleErr)
		}
		if strings.TrimSpace(assembledPrompt) != "" {
			agentDef.Prompt = strings.TrimSpace(assembledPrompt)
		}
	}

	resolved, err := cfg.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", agentName, err)
	}

	sessionID := strings.TrimSpace(m.newSessionID())
	if sessionID == "" {
		return nil, errors.New("session: session id generator returned empty id")
	}

	if err := m.reserve(sessionID, m.effectiveMaxSessions(cfg)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(sessionID)
		}
	}()

	sessionDir := filepath.Join(m.homePaths.SessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("session: create session directory %q: %w", sessionDir, err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, sessionID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}
		err = errors.Join(err, m.cleanupFailedCreate(sessionDir, recorder, proc))
	}()

	now := m.now()
	session := &Session{
		ID:         sessionID,
		Name:       strings.TrimSpace(opts.Name),
		AgentName:  resolved.Name,
		Workspace:  workspace,
		Type:       normalizeSessionType(opts.Type),
		State:      StateStarting,
		CreatedAt:  now,
		UpdatedAt:  now,
		sessionDir: sessionDir,
		metaPath:   store.SessionMetaFile(sessionDir),
		dbPath:     dbPath,
		recorder:   recorder,
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, acp.StartOpts{
		AgentName:   resolved.Name,
		Command:     resolved.Command,
		Cwd:         workspace,
		MCPServers:  append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions: aghconfig.PermissionMode(resolved.Permissions),
	})
	if err != nil {
		return nil, fmt.Errorf("session: start agent for %q: %w", sessionID, err)
	}

	session.updateFromProcess(proc, m.now())
	if err := session.activate(m.now()); err != nil {
		return nil, err
	}
	if err := m.writeMeta(session); err != nil {
		return nil, err
	}
	if err := m.activate(session); err != nil {
		return nil, err
	}

	m.watchProcess(session)
	if m.notifier != nil {
		m.notifier.OnSessionCreated(ctx, session)
	}

	return session, nil
}

// Stop stops an active session and persists the stopped state to disk.
func (m *Manager) Stop(ctx context.Context, id string) error {
	if ctx == nil {
		return errors.New("session: stop context is required")
	}

	session, err := m.lookup(id)
	if err != nil {
		return err
	}

	state := session.Info().State
	if state == StateStopped {
		return nil
	}
	if state == StateActive {
		if err := session.beginStopping(m.now()); err != nil {
			return err
		}
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}

	proc := session.processHandle()
	if proc == nil {
		return m.finalizeStopped(ctx, session, nil)
	}

	stopErr := m.driver.Stop(ctx, proc)
	if !isProcessDone(proc) {
		return stopErr
	}

	return errors.Join(stopErr, m.finalizeStopped(ctx, session, nil))
}

// Resume restarts a stopped session from its persisted metadata and event history.
func (m *Manager) Resume(ctx context.Context, id string) (_ *Session, err error) {
	if ctx == nil {
		return nil, errors.New("session: resume context is required")
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return nil, errors.New("session: session id is required")
	}

	if session, ok := m.Get(target); ok {
		return session, nil
	}

	sessionDir := filepath.Join(m.homePaths.SessionsDir, target)
	metaPath := store.SessionMetaFile(sessionDir)
	meta, err := store.ReadSessionMeta(metaPath)
	if err != nil {
		return nil, fmt.Errorf("session: read session meta %q: %w", metaPath, err)
	}

	workspace, err := resolveWorkspace(meta.Workspace)
	if err != nil {
		return nil, err
	}

	cfg, err := m.loadConfig(workspace)
	if err != nil {
		return nil, fmt.Errorf("session: load config for %q: %w", workspace, err)
	}

	agentDef, err := m.loadAgent(meta.AgentName, m.homePaths)
	if err != nil {
		return nil, fmt.Errorf("session: load agent %q: %w", meta.AgentName, err)
	}

	resolved, err := cfg.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", meta.AgentName, err)
	}

	if err := m.reserve(meta.ID, m.effectiveMaxSessions(cfg)); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			m.releaseReservation(meta.ID)
		}
	}()

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := m.openStore(ctx, meta.ID, dbPath)
	if err != nil {
		return nil, fmt.Errorf("session: open session store %q: %w", dbPath, err)
	}

	var proc *AgentProcess
	defer func() {
		if err == nil {
			return
		}
		err = errors.Join(err, m.cleanupFailedResume(recorder, proc))
	}()

	createdAt := meta.CreatedAt
	if createdAt.IsZero() {
		createdAt = m.now()
	}
	session := &Session{
		ID:           meta.ID,
		Name:         meta.Name,
		AgentName:    meta.AgentName,
		Workspace:    workspace,
		Type:         normalizeSessionType(SessionType(meta.SessionType)),
		State:        StateStarting,
		ACPSessionID: derefString(meta.ACPSessionID),
		CreatedAt:    createdAt,
		UpdatedAt:    m.now(),
		sessionDir:   sessionDir,
		metaPath:     metaPath,
		dbPath:       dbPath,
		recorder:     recorder,
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, acp.StartOpts{
		AgentName:       resolved.Name,
		Command:         resolved.Command,
		Cwd:             workspace,
		MCPServers:      append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions:     aghconfig.PermissionMode(resolved.Permissions),
		ResumeSessionID: derefString(meta.ACPSessionID),
	})
	if err != nil {
		return nil, fmt.Errorf("session: resume agent for %q: %w", meta.ID, err)
	}

	session.updateFromProcess(proc, m.now())
	if err := session.activate(m.now()); err != nil {
		return nil, err
	}
	if err := m.writeMeta(session); err != nil {
		return nil, err
	}
	if err := m.activate(session); err != nil {
		return nil, err
	}

	m.watchProcess(session)
	if m.notifier != nil {
		m.notifier.OnSessionCreated(ctx, session)
	}

	return session, nil
}

// Prompt sends one prompt turn to an active session and mirrors the runtime stream into storage and observers.
func (m *Manager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if ctx == nil {
		return nil, errors.New("session: prompt context is required")
	}

	message := strings.TrimSpace(msg)
	if message == "" {
		return nil, errors.New("session: prompt message is required")
	}

	session, err := m.lookup(id)
	if err != nil {
		return nil, err
	}

	turnID := strings.TrimSpace(m.newTurnID())
	if turnID == "" {
		turnID = newID("turn")
	}

	session.mu.RLock()
	if session.State != StateActive {
		session.mu.RUnlock()
		return nil, fmt.Errorf("session: session %q is not active", id)
	}

	proc := session.process
	if proc == nil {
		session.mu.RUnlock()
		return nil, errors.New("session: agent process is not available")
	}

	source, err := m.driver.Prompt(ctx, proc, acp.PromptRequest{TurnID: turnID, Message: message})
	session.mu.RUnlock()
	if err != nil {
		return nil, fmt.Errorf("session: prompt session %q: %w", id, err)
	}

	out := make(chan acp.AgentEvent, m.promptBufSize)
	go m.pumpPrompt(ctx, session, turnID, source, out)
	return out, nil
}

// ApprovePermission resolves one pending interactive permission request for an active session.
func (m *Manager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if ctx == nil {
		return errors.New("session: approval context is required")
	}
	if err := req.Validate(); err != nil {
		return err
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return errors.New("session: session id is required")
	}

	session, ok := m.Get(target)
	if !ok {
		meta, err := m.readMeta(target)
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: %s (%s)", ErrSessionNotActive, target, meta.State)
	}

	if err := session.ApprovePermission(ctx, req); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotActive):
			return err
		case errors.Is(err, acp.ErrPendingPermissionNotFound):
			return fmt.Errorf("%w: %s", ErrPendingPermissionNotFound, target)
		case errors.Is(err, acp.ErrPendingPermissionConflict):
			return fmt.Errorf("%w: %s", ErrPendingPermissionConflict, target)
		default:
			return err
		}
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

// List returns active in-memory sessions in stable order.
func (m *Manager) List() []*SessionInfo {
	m.mu.RLock()
	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.mu.RUnlock()

	infos := make([]*SessionInfo, 0, len(sessions))
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

func (m *Manager) reserve(id string, max int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; ok {
		return fmt.Errorf("session: session %q is already active", id)
	}
	if _, ok := m.pending[id]; ok {
		return fmt.Errorf("session: session %q is already pending", id)
	}

	active := len(m.sessions) + len(m.pending)
	if active >= max {
		return maxSessionsReachedError{active: active, limit: max}
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
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
	delete(m.pending, id)
	delete(m.finalizing, id)
}

func (m *Manager) claimFinalization(session *Session) bool {
	if session == nil {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.sessions[session.ID]
	if !ok || current != session {
		return false
	}
	if _, ok := m.finalizing[session.ID]; ok {
		return false
	}

	m.finalizing[session.ID] = struct{}{}
	return true
}

func (m *Manager) effectiveMaxSessions(cfg aghconfig.Config) int {
	if m.maxSessions > 0 {
		return m.maxSessions
	}
	if cfg.Limits.MaxSessions > 0 {
		return cfg.Limits.MaxSessions
	}
	return aghconfig.DefaultWithHome(m.homePaths).Limits.MaxSessions
}

func (m *Manager) writeMeta(session *Session) error {
	if session == nil {
		return errors.New("session: session is required")
	}
	if err := store.WriteSessionMeta(session.MetaPath(), session.meta()); err != nil {
		return fmt.Errorf("session: write meta for %q: %w", session.ID, err)
	}
	return nil
}

func (m *Manager) pumpPrompt(ctx context.Context, session *Session, turnID string, source <-chan acp.AgentEvent, out chan<- acp.AgentEvent) {
	defer close(out)

	for event := range source {
		normalized := m.normalizeEvent(session, turnID, event)
		if err := m.recordEvent(ctx, session, normalized); err != nil {
			m.sessionLogger(session).Warn("session: record prompt event failed", "turn_id", turnID, "error", err)
		}
		if m.notifier != nil {
			m.notifier.OnAgentEvent(ctx, session.ID, normalized)
		}

		select {
		case out <- normalized:
		case <-ctx.Done():
		}
	}
}

func (m *Manager) normalizeEvent(session *Session, turnID string, event acp.AgentEvent) acp.AgentEvent {
	normalized := event
	if strings.TrimSpace(normalized.TurnID) == "" {
		normalized.TurnID = turnID
	}
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = m.now()
	}
	if session != nil {
		info := session.Info()
		if strings.TrimSpace(normalized.SessionID) == "" {
			normalized.SessionID = info.ACPSessionID
		}
	}
	return normalized
}

func (m *Manager) recordEvent(ctx context.Context, session *Session, event acp.AgentEvent) error {
	recorder := session.recorderHandle()
	if recorder == nil {
		return errors.New("session: event recorder is not available")
	}

	payload, err := marshalAgentEvent(event)
	if err != nil {
		return err
	}

	if err := recorder.Record(ctx, store.SessionEvent{
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: session.Info().AgentName,
		Content:   payload,
		Timestamp: event.Timestamp,
	}); err != nil {
		return err
	}

	if event.Usage != nil {
		if err := recorder.RecordTokenUsage(ctx, store.TokenUsage{
			TurnID:           event.Usage.TurnID,
			InputTokens:      event.Usage.InputTokens,
			OutputTokens:     event.Usage.OutputTokens,
			TotalTokens:      event.Usage.TotalTokens,
			ThoughtTokens:    event.Usage.ThoughtTokens,
			CacheReadTokens:  event.Usage.CacheReadTokens,
			CacheWriteTokens: event.Usage.CacheWriteTokens,
			ContextUsed:      event.Usage.ContextUsed,
			ContextSize:      event.Usage.ContextSize,
			CostAmount:       event.Usage.CostAmount,
			CostCurrency:     event.Usage.CostCurrency,
			Timestamp:        event.Usage.Timestamp,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) watchProcess(session *Session) {
	proc := session.processHandle()
	if proc == nil {
		return
	}

	go func() {
		waitErr := proc.Wait()
		if err := m.handleProcessExit(session, waitErr); err != nil {
			m.sessionLogger(session).Warn("session: process exit handling failed", "error", err)
		}
	}()
}

func (m *Manager) handleProcessExit(session *Session, waitErr error) error {
	if session == nil {
		return nil
	}

	state := session.Info().State
	if state != StateActive && state != StateStopping {
		return nil
	}

	return m.finalizeStopped(context.Background(), session, waitErr)
}

func (m *Manager) finalizeStopped(ctx context.Context, session *Session, waitErr error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if session == nil {
		return nil
	}
	if !m.claimFinalization(session) {
		return nil
	}

	var errs []error
	state := session.Info().State
	if state == StateActive {
		if err := session.beginStopping(m.now()); err != nil {
			errs = append(errs, err)
		} else if err := m.writeMeta(session); err != nil {
			errs = append(errs, err)
		}
	}

	if waitErr != nil {
		event := acp.AgentEvent{
			Type:      acp.EventTypeError,
			TurnID:    newID("turn"),
			Timestamp: m.now(),
			Error:     waitErr.Error(),
			Text:      session.processHandle().Stderr(),
		}
		normalized := m.normalizeEvent(session, event.TurnID, event)
		if err := m.recordEvent(ctx, session, normalized); err != nil {
			errs = append(errs, err)
		}
		if m.notifier != nil {
			m.notifier.OnAgentEvent(ctx, session.ID, normalized)
		}
	}

	stopEvent := acp.AgentEvent{
		Type:      EventTypeSessionStopped,
		TurnID:    newID("turn"),
		Timestamp: m.now(),
	}
	if waitErr != nil {
		stopEvent.Error = waitErr.Error()
		if proc := session.processHandle(); proc != nil {
			stopEvent.Text = proc.Stderr()
		}
	}
	normalizedStop := m.normalizeEvent(session, stopEvent.TurnID, stopEvent)
	if err := m.recordEvent(ctx, session, normalizedStop); err != nil {
		errs = append(errs, err)
	}
	if m.notifier != nil {
		m.notifier.OnAgentEvent(ctx, session.ID, normalizedStop)
	}

	if recorder := session.recorderHandle(); recorder != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := recorder.Close(closeCtx); err != nil {
			errs = append(errs, err)
		}
		cancel()
		session.setRecorder(nil)
	}

	session.clearProcess(m.now())
	if err := session.markStopped(m.now()); err != nil {
		errs = append(errs, err)
	} else if err := m.writeMeta(session); err != nil {
		errs = append(errs, err)
	}

	m.remove(session.ID)
	if m.notifier != nil {
		m.notifier.OnSessionStopped(ctx, session)
	}

	return errors.Join(errs...)
}

func (m *Manager) cleanupFailedCreate(sessionDir string, recorder EventRecorder, proc *AgentProcess) error {
	var errs []error
	if proc != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := m.driver.Stop(stopCtx, proc); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	if recorder != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := recorder.Close(closeCtx); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	if strings.TrimSpace(sessionDir) != "" {
		if err := os.RemoveAll(sessionDir); err != nil {
			errs = append(errs, fmt.Errorf("session: remove failed session directory %q: %w", sessionDir, err))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) cleanupFailedResume(recorder EventRecorder, proc *AgentProcess) error {
	var errs []error
	if proc != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := m.driver.Stop(stopCtx, proc); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	if recorder != nil {
		closeCtx, cancel := context.WithTimeout(context.Background(), defaultLifecycleTimeout)
		if err := recorder.Close(closeCtx); err != nil {
			errs = append(errs, err)
		}
		cancel()
	}
	return errors.Join(errs...)
}

func (m *Manager) sessionLogger(session *Session) *slog.Logger {
	logger := m.logger
	if logger == nil {
		logger = slog.Default()
	}
	if session == nil {
		return logger
	}

	info := session.Info()
	return logger.With("session_id", info.ID, "agent_name", info.AgentName)
}

func resolveWorkspace(workspace string) (string, error) {
	target := strings.TrimSpace(workspace)
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("session: resolve current workspace: %w", err)
		}
		target = cwd
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("session: resolve workspace %q: %w", target, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("session: stat workspace %q: %w", absPath, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("session: workspace %q is not a directory", absPath)
	}
	return absPath, nil
}

func marshalAgentEvent(event acp.AgentEvent) (string, error) {
	if len(event.Raw) > 0 {
		return string(event.Raw), nil
	}

	payload := map[string]any{
		"type":       event.Type,
		"session_id": event.SessionID,
		"turn_id":    event.TurnID,
		"timestamp":  event.Timestamp,
	}
	if strings.TrimSpace(event.RequestID) != "" {
		payload["request_id"] = event.RequestID
	}
	if strings.TrimSpace(event.Text) != "" {
		payload["text"] = event.Text
	}
	if strings.TrimSpace(event.Title) != "" {
		payload["title"] = event.Title
	}
	if strings.TrimSpace(event.ToolCallID) != "" {
		payload["tool_call_id"] = event.ToolCallID
	}
	if strings.TrimSpace(event.StopReason) != "" {
		payload["stop_reason"] = event.StopReason
	}
	if strings.TrimSpace(event.Action) != "" {
		payload["action"] = event.Action
	}
	if strings.TrimSpace(event.Resource) != "" {
		payload["resource"] = event.Resource
	}
	if strings.TrimSpace(event.Decision) != "" {
		payload["decision"] = event.Decision
	}
	if strings.TrimSpace(event.Error) != "" {
		payload["error"] = event.Error
	}
	if event.Usage != nil {
		payload["usage"] = event.Usage
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("session: marshal agent event: %w", err)
	}
	return string(data), nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isProcessDone(proc *AgentProcess) bool {
	if proc == nil {
		return true
	}
	select {
	case <-proc.Done():
		return true
	default:
		return false
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

func newID(prefix string) string {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		now := time.Now().UTC().UnixNano()
		if strings.TrimSpace(prefix) == "" {
			return fmt.Sprintf("%d", now)
		}
		return fmt.Sprintf("%s-%d", prefix, now)
	}

	if strings.TrimSpace(prefix) == "" {
		return hex.EncodeToString(random[:])
	}
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(random[:]))
}
