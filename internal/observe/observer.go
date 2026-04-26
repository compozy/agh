// Package observe records global AGH observability data derived from live sessions.
package observe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/version"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// Registry is the narrowed global persistence surface consumed by observe/.
type Registry interface {
	RegisterSession(ctx context.Context, session store.SessionInfo) error
	UpdateSessionState(ctx context.Context, update store.SessionStateUpdate) error
	ListSessions(ctx context.Context, query store.SessionListQuery) ([]store.SessionInfo, error)
	ReconcileSessions(ctx context.Context, sessions []store.SessionInfo) (store.ReconcileResult, error)
	WriteEventSummary(ctx context.Context, summary store.EventSummary) error
	ListEventSummaries(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	UpdateTokenStats(ctx context.Context, update store.TokenStatsUpdate) error
	ListTokenStats(ctx context.Context, query store.TokenStatsQuery) ([]store.TokenStats, error)
	WritePermissionLog(ctx context.Context, entry store.PermissionLogEntry) error
	ListPermissionLog(ctx context.Context, query store.PermissionLogQuery) ([]store.PermissionLogEntry, error)
	ListNetworkAudit(ctx context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error)
	ListTasks(ctx context.Context, query taskpkg.Query) ([]taskpkg.Summary, error)
	CountDependencies(ctx context.Context, taskID string) (int, error)
	ListTaskRuns(ctx context.Context, query taskpkg.RunQuery) ([]taskpkg.Run, error)
	ListTaskEvents(ctx context.Context, query taskpkg.EventQuery) ([]taskpkg.Event, error)
	ListTaskTriageStates(ctx context.Context, actor taskpkg.ActorIdentity) ([]taskpkg.TriageState, error)
	Path() string
	Close(ctx context.Context) error
}

// SessionSource reports the currently active in-memory sessions.
type SessionSource interface {
	List() []*session.Info
}

// PermissionModeResolver resolves the effective permission mode for a live
// session using its durable workspace reference.
type PermissionModeResolver func(ctx context.Context, agentName, workspaceID string) (string, error)

// VersionSource returns the current daemon build metadata.
type VersionSource func() version.Info

// HookCatalogSource provides resolved hook catalog views from the live runtime.
type HookCatalogSource interface {
	Catalog(filter hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error)
}

// HookRunStore is the session-scoped storage surface used for hook run audits.
type HookRunStore interface {
	RecordHookRun(context.Context, hookspkg.HookRunRecord) error
	QueryHookRuns(context.Context, store.HookRunQuery) ([]hookspkg.HookRunRecord, error)
	Close(context.Context) error
}

// HookStoreOpener opens the per-session store used for hook run audit queries.
type HookStoreOpener func(ctx context.Context, sessionID string, path string) (HookRunStore, error)

// Option customizes Observer construction.
type Option func(*Observer)

type observedSession struct {
	agentName      string
	workspaceID    string
	permissionMode string
}

// TaskHealthConfig controls task-run stuck detection in the read-side health view.
type TaskHealthConfig struct {
	ClaimedStuckAfter  time.Duration
	StartingStuckAfter time.Duration
	RunningStuckAfter  time.Duration
}

// TaskDashboardConfig controls task dashboard freshness/backlog thresholds and active-run list size.
type TaskDashboardConfig struct {
	ActiveRunLimit   int
	BacklogWarnAfter time.Duration
	StaleAfter       time.Duration
}

// AgentProbeTargetSource resolves the currently configured ACP-compatible
// agent/provider commands that should be checked in observe health.
type AgentProbeTargetSource func(ctx context.Context) ([]acp.ProbeTarget, error)

type taskDashboardConfig struct {
	activeRunLimit   int
	backlogWarnAfter time.Duration
	staleAfter       time.Duration
}

// Observer implements session.Notifier and exposes query/health helpers for global observability.
type Observer struct {
	mu sync.RWMutex

	registry              Registry
	homePaths             aghconfig.HomePaths
	sessionSource         SessionSource
	resolvePermissionMode PermissionModeResolver
	workspaceResolver     workspacepkg.RuntimeResolver
	now                   func() time.Time
	startedAt             time.Time
	logger                *slog.Logger
	versionSource         VersionSource
	sessions              map[string]observedSession
	bridgeSource          BridgeSource
	bridgeState           map[string]observedBridgeState
	hookCatalogSource     HookCatalogSource
	openHookStore         HookStoreOpener
	taskHealthConfig      TaskHealthConfig
	taskDashboardConfig   taskDashboardConfig
	retention             RetentionConfig
	retentionHealth       RetentionHealth
	retentionCancel       context.CancelFunc
	retentionWG           sync.WaitGroup
	retentionMu           sync.RWMutex
	agentProbeSource      AgentProbeTargetSource
	agentProbeTimeout     time.Duration
}

var _ session.Notifier = (*Observer)(nil)
var _ session.AgentEventNotifier = (*Observer)(nil)

// WithRegistry injects the global registry implementation used by observe/.
func WithRegistry(registry Registry) Option {
	return func(observer *Observer) {
		observer.registry = registry
	}
}

// WithHomePaths overrides the AGH home layout used by observe/.
func WithHomePaths(homePaths aghconfig.HomePaths) Option {
	return func(observer *Observer) {
		observer.homePaths = homePaths
	}
}

// WithSessionSource injects the active session source used for health metrics.
func WithSessionSource(source SessionSource) Option {
	return func(observer *Observer) {
		observer.sessionSource = source
	}
}

// WithPermissionModeResolver injects custom permission resolution logic.
func WithPermissionModeResolver(resolver PermissionModeResolver) Option {
	return func(observer *Observer) {
		observer.resolvePermissionMode = resolver
	}
}

// WithWorkspaceResolver injects workspace resolution for config lookups that
// need a filesystem root.
func WithWorkspaceResolver(resolver workspacepkg.RuntimeResolver) Option {
	return func(observer *Observer) {
		observer.workspaceResolver = resolver
	}
}

// WithLogger injects the logger used for best-effort observer failures.
func WithLogger(logger *slog.Logger) Option {
	return func(observer *Observer) {
		observer.logger = logger
	}
}

// WithNow overrides the observer clock, mainly for tests.
func WithNow(now func() time.Time) Option {
	return func(observer *Observer) {
		observer.now = now
	}
}

// WithStartTime overrides the recorded daemon start time.
func WithStartTime(startedAt time.Time) Option {
	return func(observer *Observer) {
		observer.startedAt = startedAt
	}
}

// WithVersionSource overrides the build metadata source.
func WithVersionSource(source VersionSource) Option {
	return func(observer *Observer) {
		observer.versionSource = source
	}
}

// WithHookCatalogSource injects the runtime hook catalog source used by hook introspection.
func WithHookCatalogSource(source HookCatalogSource) Option {
	return func(observer *Observer) {
		observer.hookCatalogSource = source
	}
}

// WithHookStoreOpener overrides the per-session hook run store opener, mainly for tests.
func WithHookStoreOpener(opener HookStoreOpener) Option {
	return func(observer *Observer) {
		observer.openHookStore = opener
	}
}

// WithTaskHealthConfig overrides the task-run stuck thresholds used by the
// observer health view.
func WithTaskHealthConfig(cfg TaskHealthConfig) Option {
	return func(observer *Observer) {
		observer.taskHealthConfig = cfg
	}
}

// WithTaskDashboardConfig overrides the dashboard thresholds and active-run
// list sizing used by the observer task dashboard view.
func WithTaskDashboardConfig(cfg TaskDashboardConfig) Option {
	return func(observer *Observer) {
		observer.taskDashboardConfig = normalizeTaskDashboardConfig(observer.taskDashboardConfig, cfg)
	}
}

// WithObservabilityConfig applies observability settings that affect observer-owned background work.
func WithObservabilityConfig(cfg aghconfig.ObservabilityConfig) Option {
	return func(observer *Observer) {
		observer.retention = RetentionConfigFromObservability(cfg)
	}
}

// WithRetentionConfig overrides retention behavior, mainly for tests.
func WithRetentionConfig(cfg RetentionConfig) Option {
	return func(observer *Observer) {
		observer.retention = cfg
	}
}

// WithAgentProbeSource injects the downstream ACP command source used by health.
func WithAgentProbeSource(source AgentProbeTargetSource, timeout time.Duration) Option {
	return func(observer *Observer) {
		observer.agentProbeSource = source
		observer.agentProbeTimeout = timeout
	}
}

func defaultTaskDashboardConfig() taskDashboardConfig {
	return taskDashboardConfig{
		activeRunLimit:   4,
		backlogWarnAfter: 10 * time.Minute,
		staleAfter:       2 * time.Minute,
	}
}

func normalizeTaskDashboardConfig(base taskDashboardConfig, cfg TaskDashboardConfig) taskDashboardConfig {
	normalized := base
	if normalized.activeRunLimit <= 0 {
		normalized = defaultTaskDashboardConfig()
	}
	if cfg.ActiveRunLimit > 0 {
		normalized.activeRunLimit = cfg.ActiveRunLimit
	}
	if cfg.BacklogWarnAfter > 0 {
		normalized.backlogWarnAfter = cfg.BacklogWarnAfter
	}
	if cfg.StaleAfter > 0 {
		normalized.staleAfter = cfg.StaleAfter
	}
	return normalized
}

// New constructs an Observer and opens the global AGH database when needed.
func New(ctx context.Context, opts ...Option) (*Observer, error) {
	if ctx == nil {
		return nil, errors.New("observe: context is required")
	}

	homePaths, err := aghconfig.ResolveHomePaths()
	if err != nil {
		return nil, fmt.Errorf("observe: resolve home paths: %w", err)
	}

	observer := &Observer{
		homePaths: homePaths,
		now: func() time.Time {
			return time.Now().UTC()
		},
		logger:        slog.Default(),
		versionSource: version.Current,
		sessions:      make(map[string]observedSession),
		bridgeState:   make(map[string]observedBridgeState),
		taskHealthConfig: TaskHealthConfig{
			ClaimedStuckAfter:  5 * time.Minute,
			StartingStuckAfter: 5 * time.Minute,
			RunningStuckAfter:  30 * time.Minute,
		},
		taskDashboardConfig: defaultTaskDashboardConfig(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(observer)
		}
	}

	if observer.now == nil {
		observer.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if observer.startedAt.IsZero() {
		observer.startedAt = observer.now()
	}
	if observer.logger == nil {
		observer.logger = slog.Default()
	}
	if observer.versionSource == nil {
		observer.versionSource = version.Current
	}
	if observer.sessions == nil {
		observer.sessions = make(map[string]observedSession)
	}
	if observer.bridgeState == nil {
		observer.bridgeState = make(map[string]observedBridgeState)
	}
	if observer.resolvePermissionMode == nil {
		observer.resolvePermissionMode = defaultPermissionModeResolver(observer.homePaths, observer.workspaceResolver)
	}
	if observer.openHookStore == nil {
		observer.openHookStore = func(ctx context.Context, sessionID string, path string) (HookRunStore, error) {
			return sessiondb.OpenSessionDB(ctx, sessionID, path)
		}
	}
	observer.retention = normalizeRetentionConfig(observer.retention)
	observer.setRetentionHealth(observer.initialRetentionHealth())

	if observer.registry == nil {
		if err := aghconfig.EnsureHomeLayout(observer.homePaths); err != nil {
			return nil, fmt.Errorf("observe: ensure home layout: %w", err)
		}

		registry, err := globaldb.OpenGlobalDB(ctx, observer.homePaths.DatabaseFile)
		if err != nil {
			return nil, fmt.Errorf("observe: open global database: %w", err)
		}
		observer.registry = registry
	}

	return observer, nil
}

// AttachHooks swaps in the live hook catalog source after the hook runtime is built.
func (o *Observer) AttachHooks(source HookCatalogSource) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.hookCatalogSource = source
}

func (o *Observer) hookDBPath(sessionID string) string {
	return store.SessionDBFile(filepath.Join(o.homePaths.SessionsDir, strings.TrimSpace(sessionID)))
}

func (o *Observer) openHookRunStore(ctx context.Context, sessionID string) (HookRunStore, func() error, error) {
	if o == nil {
		return nil, nil, errors.New("observe: observer is required")
	}
	if ctx == nil {
		return nil, nil, errors.New("observe: hook run context is required")
	}

	target, err := sanitizeHookSessionID(sessionID)
	if err != nil {
		return nil, nil, err
	}

	dbPath := o.hookDBPath(target)
	if _, err := os.Stat(dbPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil, os.ErrNotExist
		}
		return nil, nil, fmt.Errorf("observe: stat hook database for %q: %w", target, err)
	}

	openStore := o.openHookStore
	if openStore == nil {
		return nil, nil, errors.New("observe: hook store opener is required")
	}

	storeHandle, err := openStore(ctx, target, dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("observe: open hook database for %q: %w", target, err)
	}

	cleanup := func() error {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return storeHandle.Close(closeCtx)
	}
	return storeHandle, cleanup, nil
}

// Close flushes and closes the backing global registry.
func (o *Observer) Close(ctx context.Context) error {
	return o.registry.Close(ctx)
}

// OnSessionCreated registers the session in the global observability database.
func (o *Observer) OnSessionCreated(ctx context.Context, sess *session.Session) {
	info := sess.Info()
	snapshot := observedSession{
		agentName:   info.AgentName,
		workspaceID: info.WorkspaceID,
	}
	if o.resolvePermissionMode != nil {
		permissionMode, err := o.resolvePermissionMode(ctx, info.AgentName, info.WorkspaceID)
		if err != nil {
			o.logger.Warn(
				"observe: resolve permission mode failed",
				"session_id",
				info.ID,
				"agent_name",
				info.AgentName,
				"workspace_id",
				info.WorkspaceID,
				"error",
				err,
			)
		} else {
			snapshot.permissionMode = strings.TrimSpace(permissionMode)
		}
	}

	o.trackSession(info.ID, snapshot)

	if err := o.registry.RegisterSession(ctx, sessionInfoFromSession(info)); err != nil {
		o.logger.Warn(
			"observe: register session failed",
			"session_id",
			info.ID,
			"agent_name",
			info.AgentName,
			"workspace_id",
			info.WorkspaceID,
			"error",
			err,
		)
	}
}

// OnSessionStopped updates the session state in the global observability database.
func (o *Observer) OnSessionStopped(ctx context.Context, sess *session.Session) {
	info := sess.Info()

	if err := o.registry.UpdateSessionState(ctx, store.SessionStateUpdate{
		ID:            info.ID,
		State:         string(info.State),
		ACPSessionID:  stringPointer(info.ACPSessionID),
		StopReasonSet: true,
		StopReason:    stringPointer(string(info.StopReason)),
		StopDetail:    info.StopDetail,
		FailureSet:    true,
		Failure:       store.CloneSessionFailure(info.Failure),
		Liveness:      store.CloneSessionLivenessMeta(info.Liveness),
		Environment:   cloneSessionEnvironmentMeta(info.Environment),
		UpdatedAt:     info.UpdatedAt,
	}); err != nil {
		o.logger.Warn(
			"observe: update session state failed",
			"session_id",
			info.ID,
			"agent_name",
			info.AgentName,
			"workspace_id",
			info.WorkspaceID,
			"state",
			info.State,
			"error",
			err,
		)
	}

	o.untrackSession(info.ID)
}

// OnAgentEvent records one lightweight cross-session event summary and any derived aggregates.
func (o *Observer) OnAgentEvent(ctx context.Context, sessionID string, payload any) {
	o.observeAgentEvent(ctx, strings.TrimSpace(sessionID), payload)
}

// OnAgentEventForSession records event summaries and refreshes the indexed
// liveness state for the active session.
func (o *Observer) OnAgentEventForSession(ctx context.Context, sess *session.Session, payload any) {
	if sess == nil {
		return
	}
	info := sess.Info()
	if info == nil {
		return
	}
	if err := o.registry.UpdateSessionState(ctx, store.SessionStateUpdate{
		ID:           info.ID,
		State:        string(info.State),
		ACPSessionID: stringPointer(info.ACPSessionID),
		Liveness:     store.CloneSessionLivenessMeta(info.Liveness),
		Environment:  cloneSessionEnvironmentMeta(info.Environment),
		UpdatedAt:    info.UpdatedAt,
	}); err != nil {
		o.logger.Warn(
			"observe: update session liveness failed",
			"session_id", info.ID,
			"state", info.State,
			"error", err,
		)
	}
	o.observeAgentEvent(ctx, info.ID, payload)
}

func (o *Observer) observeAgentEvent(ctx context.Context, sessionID string, payload any) {
	event, ok := normalizeObservedAgentEvent(payload)
	if !ok {
		o.logger.Warn("observe: skipped unsupported agent event payload", "session_id", strings.TrimSpace(sessionID))
		return
	}

	id, snapshot, ok := o.validateObservedEvent(sessionID, event)
	if !ok {
		return
	}

	timestamp := observedEventTimestamp(event, o.now)

	if err := o.writeObservedEventSummary(ctx, id, snapshot, event, timestamp); err != nil {
		o.logObservedEventFailure("observe: write event summary failed", id, snapshot, event, err)
	}
	if err := o.aggregateObservedUsage(ctx, id, snapshot, event, timestamp); err != nil {
		o.logger.Warn(
			"observe: update token stats failed",
			"session_id",
			id,
			"agent_name",
			snapshot.agentName,
			"workspace_id",
			snapshot.workspaceID,
			"turn_id",
			event.TurnID,
			"error",
			err,
		)
	}
	if err := o.writeObservedPermissionLog(ctx, id, snapshot, event, timestamp); err != nil {
		o.logger.Warn(
			"observe: write permission log failed",
			"session_id",
			id,
			"agent_name",
			snapshot.agentName,
			"workspace_id",
			snapshot.workspaceID,
			"error",
			err,
		)
	}
}

func (o *Observer) validateObservedEvent(
	sessionID string,
	event acp.AgentEvent,
) (string, observedSession, bool) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		o.logger.Warn("observe: skipped agent event with empty session id", "event_type", event.Type)
		return "", observedSession{}, false
	}

	snapshot, ok := o.sessionSnapshot(id)
	if !ok {
		o.logger.Warn("observe: skipped agent event for unknown session", "session_id", id, "event_type", event.Type)
		return "", observedSession{}, false
	}
	if strings.TrimSpace(event.Type) == "" {
		o.logger.Warn(
			"observe: skipped agent event with empty type",
			"session_id",
			id,
			"agent_name",
			snapshot.agentName,
			"workspace_id",
			snapshot.workspaceID,
		)
		return "", observedSession{}, false
	}

	return id, snapshot, true
}

func observedEventTimestamp(event acp.AgentEvent, now func() time.Time) time.Time {
	if !event.Timestamp.IsZero() {
		return event.Timestamp
	}
	return now()
}

func (o *Observer) writeObservedEventSummary(
	ctx context.Context,
	sessionID string,
	snapshot observedSession,
	event acp.AgentEvent,
	timestamp time.Time,
) error {
	return o.registry.WriteEventSummary(ctx, store.EventSummary{
		SessionID: sessionID,
		Type:      strings.TrimSpace(event.Type),
		AgentName: snapshot.agentName,
		Summary:   summarizeEvent(event),
		Timestamp: timestamp,
	})
}

func (o *Observer) aggregateObservedUsage(
	ctx context.Context,
	sessionID string,
	snapshot observedSession,
	event acp.AgentEvent,
	timestamp time.Time,
) error {
	if !shouldAggregateUsage(event) {
		return nil
	}

	usageTimestamp := timestamp
	if !event.Usage.Timestamp.IsZero() {
		usageTimestamp = event.Usage.Timestamp
	}

	return o.registry.UpdateTokenStats(ctx, store.TokenStatsUpdate{
		SessionID:    sessionID,
		AgentName:    snapshot.agentName,
		InputTokens:  event.Usage.InputTokens,
		OutputTokens: event.Usage.OutputTokens,
		TotalTokens:  event.Usage.TotalTokens,
		CostAmount:   event.Usage.CostAmount,
		CostCurrency: event.Usage.CostCurrency,
		Turns:        1,
		UpdatedAt:    usageTimestamp,
	})
}

func (o *Observer) writeObservedPermissionLog(
	ctx context.Context,
	sessionID string,
	snapshot observedSession,
	event acp.AgentEvent,
	timestamp time.Time,
) error {
	if strings.TrimSpace(event.Type) != acp.EventTypePermission {
		return nil
	}

	policyUsed := strings.TrimSpace(snapshot.permissionMode)
	if policyUsed == "" {
		o.logger.Warn(
			"observe: skipped permission log without resolved policy",
			"session_id",
			sessionID,
			"agent_name",
			snapshot.agentName,
			"workspace_id",
			snapshot.workspaceID,
		)
		return nil
	}
	if strings.TrimSpace(event.Decision) == "" {
		return nil
	}

	return o.registry.WritePermissionLog(ctx, store.PermissionLogEntry{
		SessionID:  sessionID,
		AgentName:  snapshot.agentName,
		Action:     strings.TrimSpace(event.Action),
		Resource:   strings.TrimSpace(event.Resource),
		Decision:   strings.TrimSpace(event.Decision),
		PolicyUsed: policyUsed,
		Timestamp:  timestamp,
	})
}

func (o *Observer) logObservedEventFailure(
	message string,
	sessionID string,
	snapshot observedSession,
	event acp.AgentEvent,
	err error,
) {
	o.logger.Warn(
		message,
		"session_id",
		sessionID,
		"agent_name",
		snapshot.agentName,
		"workspace_id",
		snapshot.workspaceID,
		"event_type",
		event.Type,
		"error",
		err,
	)
}

func normalizeObservedAgentEvent(payload any) (acp.AgentEvent, bool) {
	switch event := payload.(type) {
	case acp.AgentEvent:
		return event, true
	case *acp.AgentEvent:
		if event == nil {
			return acp.AgentEvent{}, false
		}
		return *event, true
	default:
		return acp.AgentEvent{}, false
	}
}

func (o *Observer) trackSession(id string, snapshot observedSession) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sessions[strings.TrimSpace(id)] = snapshot
}

func (o *Observer) untrackSession(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.sessions, strings.TrimSpace(id))
}

func (o *Observer) sessionSnapshot(id string) (observedSession, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	snapshot, ok := o.sessions[strings.TrimSpace(id)]
	return snapshot, ok
}

func defaultPermissionModeResolver(
	homePaths aghconfig.HomePaths,
	resolver workspacepkg.RuntimeResolver,
) PermissionModeResolver {
	return func(ctx context.Context, agentName, workspaceID string) (string, error) {
		if ctx == nil {
			return "", errors.New("observe: permission resolver context is required")
		}

		var (
			cfg      aghconfig.Config
			agentDef aghconfig.AgentDef
			err      error
		)
		if strings.TrimSpace(workspaceID) == "" {
			cfg, err = aghconfig.LoadForHome(homePaths)
			if err != nil {
				return "", fmt.Errorf("load config: %w", err)
			}
			agentDef, err = aghconfig.LoadAgentDef(agentName, homePaths)
		} else {
			if resolver == nil {
				return "", errors.New("observe: workspace resolver is required")
			}

			resolvedWorkspace, resolveErr := resolver.Resolve(ctx, workspaceID)
			if resolveErr != nil {
				return "", fmt.Errorf("resolve workspace %q: %w", workspaceID, resolveErr)
			}
			cfg, err = aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(resolvedWorkspace.RootDir))
			if err != nil {
				return "", fmt.Errorf("load config: %w", err)
			}
			agentDef, err = agentDefByName(resolvedWorkspace.Agents, agentName)
		}
		if err != nil {
			return "", fmt.Errorf("load agent %q: %w", agentName, err)
		}

		resolved, err := cfg.ResolveAgent(agentDef)
		if err != nil {
			return "", fmt.Errorf("resolve agent %q: %w", agentName, err)
		}

		return strings.TrimSpace(resolved.Permissions), nil
	}
}

func agentDefByName(agents []aghconfig.AgentDef, name string) (aghconfig.AgentDef, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return aghconfig.AgentDef{}, errors.New("agent name is required")
	}

	for _, agent := range agents {
		if strings.TrimSpace(agent.Name) == target {
			return agent, nil
		}
	}

	return aghconfig.AgentDef{}, workspacepkg.ErrAgentNotAvailable
}

func sessionInfoFromSession(info *session.Info) store.SessionInfo {
	if info == nil {
		return store.SessionInfo{}
	}

	return store.SessionInfo{
		ID:           info.ID,
		Name:         info.Name,
		AgentName:    info.AgentName,
		Provider:     info.Provider,
		WorkspaceID:  info.WorkspaceID,
		Channel:      info.Channel,
		SessionType:  string(info.Type),
		Lineage:      store.CloneSessionLineage(info.Lineage),
		State:        string(info.State),
		ACPSessionID: stringPointer(info.ACPSessionID),
		StopReason:   info.StopReason,
		StopDetail:   info.StopDetail,
		Failure:      store.CloneSessionFailure(info.Failure),
		Liveness:     store.CloneSessionLivenessMeta(info.Liveness),
		Environment:  cloneSessionEnvironmentMeta(info.Environment),
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

// OnEnvironmentLifecycleEvent receives optional environment lifecycle spans from session orchestration.
func (o *Observer) OnEnvironmentLifecycleEvent(_ context.Context, event session.EnvironmentLifecycleEvent) {
	if o == nil || o.logger == nil {
		return
	}
	o.logger.Debug(
		"observe: environment lifecycle",
		"name", event.Name,
		"span", event.Span,
		"session_id", event.SessionID,
		"workspace_id", event.WorkspaceID,
		"environment_id", event.EnvironmentID,
		"backend", event.Backend,
		"profile", event.Profile,
		"instance_id", event.InstanceID,
		"duration_ms", event.Duration.Milliseconds(),
		"error_kind", event.ErrorKind,
		"error", event.Error,
	)
}

func summarizeEvent(event acp.AgentEvent) string {
	if strings.TrimSpace(event.Type) == acp.EventTypePermission {
		if summary := firstNonEmptySummary(
			event.Title,
			event.Resource,
			event.Decision,
			event.Text,
			event.Error,
			event.StopReason,
			event.ToolCallID,
		); summary != "" {
			return truncateSummary(summary)
		}
	}
	if summary := firstNonEmptySummary(
		event.Text,
		event.Title,
		event.Error,
		event.Resource,
		event.StopReason,
		event.ToolCallID,
	); summary != "" {
		return truncateSummary(summary)
	}

	if len(event.Raw) > 0 {
		return truncateSummary(string(event.Raw))
	}
	return ""
}

func firstNonEmptySummary(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateSummary(summary string) string {
	const maxRunes = 240

	clean := strings.TrimSpace(summary)
	if clean == "" {
		return ""
	}
	if len(clean) <= maxRunes {
		return clean
	}

	runes := []rune(clean)
	if len(runes) <= maxRunes {
		return clean
	}

	return string(runes[:maxRunes-3]) + "..."
}

func sanitizeHookSessionID(sessionID string) (string, error) {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return "", errors.New("observe: session id is required")
	}
	if target == "." || target == ".." || strings.ContainsAny(target, `/\`) {
		return "", fmt.Errorf("observe: invalid session id %q", sessionID)
	}
	return target, nil
}

func cloneSessionEnvironmentMeta(meta *store.SessionEnvironmentMeta) *store.SessionEnvironmentMeta {
	if meta == nil {
		return nil
	}
	cloned := *meta
	cloned.RuntimeAdditionalDirs = append([]string(nil), meta.RuntimeAdditionalDirs...)
	if meta.ProviderState != nil {
		cloned.ProviderState = append([]byte(nil), meta.ProviderState...)
	}
	if meta.SSHAccessExpiresAt != nil {
		expiresAt := *meta.SSHAccessExpiresAt
		cloned.SSHAccessExpiresAt = &expiresAt
	}
	if meta.LastSyncAt != nil {
		lastSyncAt := *meta.LastSyncAt
		cloned.LastSyncAt = &lastSyncAt
	}
	return &cloned
}

func shouldAggregateUsage(event acp.AgentEvent) bool {
	return strings.TrimSpace(event.Type) == acp.EventTypeDone && event.Usage != nil && !event.Usage.IsZero()
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	copyValue := value
	return &copyValue
}
