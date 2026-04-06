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
	Name          string
	Workspace     string
	WorkspacePath string
	Type          SessionType
}

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
	workspace     workspacepkg.WorkspaceResolver
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

// WithWorkspaceResolver injects workspace resolution for create/resume flows.
func WithWorkspaceResolver(resolver workspacepkg.WorkspaceResolver) Option {
	return func(manager *Manager) {
		manager.workspace = resolver
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

	resolvedWorkspace, err := m.resolveCreateWorkspace(ctx, opts)
	if err != nil {
		return nil, err
	}

	agentName, err := aghconfig.ResolveAgentName(opts.AgentName, resolvedWorkspace.Config)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent name: %w", err)
	}

	agentDef, err := resolveWorkspaceAgent(agentName, resolvedWorkspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve workspace agent %q: %w", agentName, err)
	}
	startupPrompt, err := m.startupPrompt(ctx, agentName, agentDef, resolvedWorkspace)
	if err != nil {
		return nil, err
	}
	agentDef.Prompt = startupPrompt

	resolved, err := resolvedWorkspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", agentName, err)
	}

	sessionID := strings.TrimSpace(m.newSessionID())
	if sessionID == "" {
		return nil, errors.New("session: session id generator returned empty id")
	}

	if err := m.reserve(sessionID, m.effectiveMaxSessions(resolvedWorkspace.Config)); err != nil {
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
		ID:          sessionID,
		Name:        strings.TrimSpace(opts.Name),
		AgentName:   resolved.Name,
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        normalizeSessionType(opts.Type),
		State:       StateStarting,
		CreatedAt:   now,
		UpdatedAt:   now,
		sessionDir:  sessionDir,
		metaPath:    store.SessionMetaFile(sessionDir),
		dbPath:      dbPath,
		recorder:    recorder,
	}

	if err := m.writeMeta(session); err != nil {
		return nil, err
	}

	proc, err = m.driver.Start(ctx, acp.StartOpts{
		AgentName:      resolved.Name,
		Command:        resolved.Command,
		Cwd:            resolvedWorkspace.RootDir,
		AdditionalDirs: append([]string(nil), resolvedWorkspace.AdditionalDirs...),
		MCPServers:     append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions:    m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:   resolved.Prompt,
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

	writeMeta, promptSetupDone, err := session.prepareStop(m.now())
	if err != nil {
		return err
	}
	if writeMeta {
		if err := m.writeMeta(session); err != nil {
			return err
		}
	}
	if err := waitForPromptSetup(ctx, session, promptSetupDone); err != nil {
		return err
	}

	state := session.Info().State
	if state == StateStopped {
		return nil
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

	resolvedWorkspace, err := m.resolveResumeWorkspace(ctx, meta)
	if err != nil {
		return nil, err
	}

	agentDef, err := resolveWorkspaceAgent(meta.AgentName, resolvedWorkspace)
	if err != nil {
		return nil, fmt.Errorf("session: resolve workspace agent %q: %w", meta.AgentName, err)
	}
	startupPrompt, err := m.startupPrompt(ctx, meta.AgentName, agentDef, resolvedWorkspace)
	if err != nil {
		return nil, err
	}
	agentDef.Prompt = startupPrompt

	resolved, err := resolvedWorkspace.Config.ResolveAgent(agentDef)
	if err != nil {
		return nil, fmt.Errorf("session: resolve agent %q: %w", meta.AgentName, err)
	}

	if err := m.reserve(meta.ID, m.effectiveMaxSessions(resolvedWorkspace.Config)); err != nil {
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
		WorkspaceID:  strings.TrimSpace(meta.WorkspaceID),
		Workspace:    resolvedWorkspace.RootDir,
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
		Cwd:             resolvedWorkspace.RootDir,
		AdditionalDirs:  append([]string(nil), resolvedWorkspace.AdditionalDirs...),
		MCPServers:      append([]aghconfig.MCPServer(nil), resolved.MCPServers...),
		Permissions:     m.startPermissions(session.Type, resolved.Permissions),
		SystemPrompt:    resolved.Prompt,
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

func (m *Manager) startupPrompt(ctx context.Context, agentName string, agent aghconfig.AgentDef, workspace workspacepkg.ResolvedWorkspace) (string, error) {
	prompt := strings.TrimSpace(agent.Prompt)
	if m.assembler == nil {
		return prompt, nil
	}

	assembledPrompt, err := m.assembler.Assemble(ctx, agent, workspace)
	if err != nil {
		return "", fmt.Errorf("session: assemble prompt for %q: %w", agentName, err)
	}
	if strings.TrimSpace(assembledPrompt) == "" {
		return prompt, nil
	}

	return strings.TrimSpace(assembledPrompt), nil
}

func (m *Manager) startPermissions(sessionType SessionType, configured string) aghconfig.PermissionMode {
	if normalizeSessionType(sessionType) == SessionTypeDream {
		return aghconfig.PermissionModeApproveAll
	}

	mode := aghconfig.PermissionMode(strings.TrimSpace(configured))
	if mode == "" {
		return aghconfig.PermissionModeApproveReads
	}
	return mode
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

	proc, err := session.beginPromptSetup()
	if err != nil {
		return nil, err
	}
	defer session.finishPromptSetup()

	userEvent := m.normalizeEvent(session, turnID, acp.AgentEvent{
		Type:      acp.EventTypeUserMessage,
		TurnID:    turnID,
		Timestamp: m.now(),
		Text:      message,
	})
	if err := m.recordEvent(ctx, session, userEvent); err != nil {
		return nil, fmt.Errorf("session: persist prompt message for %q: %w", id, err)
	}
	if m.notifier != nil {
		m.notifier.OnAgentEvent(ctx, session.ID, userEvent)
	}

	source, err := m.driver.Prompt(ctx, proc, acp.PromptRequest{TurnID: turnID, Message: message})
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

func (m *Manager) resolveCreateWorkspace(ctx context.Context, opts CreateOpts) (workspacepkg.ResolvedWorkspace, error) {
	resolver, err := m.requireWorkspaceResolver()
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}

	workspaceRef := strings.TrimSpace(opts.Workspace)
	workspacePath := strings.TrimSpace(opts.WorkspacePath)
	switch {
	case workspaceRef == "" && workspacePath == "":
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: workspace or workspace path is required")
	case workspaceRef != "" && workspacePath != "":
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: workspace and workspace path are mutually exclusive")
	case workspacePath != "":
		resolved, err := resolver.ResolveOrRegister(ctx, workspacePath)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("session: resolve workspace path %q: %w", workspacePath, err)
		}
		return resolved, nil
	default:
		resolved, err := resolver.Resolve(ctx, workspaceRef)
		if err != nil {
			return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("session: resolve workspace %q: %w", workspaceRef, err)
		}
		return resolved, nil
	}
}

func (m *Manager) resolveResumeWorkspace(ctx context.Context, meta store.SessionMeta) (workspacepkg.ResolvedWorkspace, error) {
	resolver, err := m.requireWorkspaceResolver()
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}

	workspaceID := strings.TrimSpace(meta.WorkspaceID)
	if workspaceID == "" {
		return workspacepkg.ResolvedWorkspace{}, errors.New("session: session workspace id is required")
	}

	resolved, err := resolver.Resolve(ctx, workspaceID)
	if err != nil {
		return workspacepkg.ResolvedWorkspace{}, fmt.Errorf("session: resolve workspace %q for session %q: %w", workspaceID, meta.ID, err)
	}
	return resolved, nil
}

func (m *Manager) requireWorkspaceResolver() (workspacepkg.WorkspaceResolver, error) {
	if m.workspace == nil {
		return nil, errors.New("session: workspace resolver is required")
	}
	return m.workspace, nil
}

func resolveWorkspaceAgent(agentName string, resolvedWorkspace workspacepkg.ResolvedWorkspace) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(agentName)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("session: agent name is required")
	}

	for _, agent := range resolvedWorkspace.Agents {
		if strings.TrimSpace(agent.Name) != target {
			continue
		}
		return agent, nil
	}

	return aghconfig.AgentDef{}, fmt.Errorf("%w: %s", workspacepkg.ErrAgentNotAvailable, target)
}

func marshalAgentEvent(event acp.AgentEvent) (string, error) {
	payload := canonicalEventPayload{
		Schema:     eventEnvelopeSchema,
		Type:       event.Type,
		SessionID:  event.SessionID,
		TurnID:     event.TurnID,
		RequestID:  event.RequestID,
		Timestamp:  event.Timestamp,
		Text:       event.Text,
		Title:      event.Title,
		ToolCallID: event.ToolCallID,
		StopReason: event.StopReason,
		Action:     event.Action,
		Resource:   event.Resource,
		Decision:   event.Decision,
		Error:      event.Error,
		Usage:      event.Usage,
	}

	if len(event.Raw) > 0 {
		if json.Valid(event.Raw) {
			payload.Raw = cloneRawMessage(event.Raw)
		} else {
			payload.Raw = rawMessageFromValue(string(event.Raw))
		}

		var rawPayload map[string]any
		if err := json.Unmarshal(event.Raw, &rawPayload); err == nil {
			payload.ToolName = legacyToolName(rawPayload)
			payload.ToolInput = cloneRawMessage(rawMessageFromValue(rawPayload["rawInput"]))
			if event.Type == acp.EventTypeToolResult {
				toolResult := buildToolResult(
					payload.ToolName,
					strings.EqualFold(nestedString(rawPayload, "status"), "failed"),
					extractLegacyContentText(rawPayload["content"]),
					rawPayload["rawOutput"],
				)
				payload.ToolResult = toolResult
				payload.ToolError = strings.EqualFold(nestedString(rawPayload, "status"), "failed")
			}
		}
	}

	if payload.ToolName == "" {
		payload.ToolName = event.Title
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

func waitForPromptSetup(ctx context.Context, session *Session, promptSetupDone <-chan struct{}) error {
	if promptSetupDone == nil {
		return nil
	}
	select {
	case <-promptSetupDone:
		return nil
	case <-ctx.Done():
		sessionID := ""
		if session != nil {
			sessionID = session.ID
		}
		return fmt.Errorf("session: wait for in-flight prompt setup for %q: %w", sessionID, ctx.Err())
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
