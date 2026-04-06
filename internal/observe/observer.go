// Package observe records global AGH observability data derived from live sessions.
package observe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/version"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// Registry is the global persistence surface consumed by observe/.
type Registry interface {
	store.SessionRegistry
	Path() string
}

// SessionSource reports the currently active in-memory sessions.
type SessionSource interface {
	List() []*session.SessionInfo
}

// PermissionModeResolver resolves the effective permission mode for a live
// session using its durable workspace reference.
type PermissionModeResolver func(ctx context.Context, agentName, workspaceID string) (string, error)

// VersionSource returns the current daemon build metadata.
type VersionSource func() version.Info

// Option customizes Observer construction.
type Option func(*Observer)

type observedSession struct {
	agentName      string
	workspaceID    string
	permissionMode string
}

// Observer implements session.Notifier and exposes query/health helpers for global observability.
type Observer struct {
	mu sync.RWMutex

	registry              Registry
	homePaths             aghconfig.HomePaths
	sessionSource         SessionSource
	resolvePermissionMode PermissionModeResolver
	workspaceResolver     workspacepkg.WorkspaceResolver
	now                   func() time.Time
	startedAt             time.Time
	logger                *slog.Logger
	versionSource         VersionSource
	sessions              map[string]observedSession
}

var _ session.Notifier = (*Observer)(nil)

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
func WithWorkspaceResolver(resolver workspacepkg.WorkspaceResolver) Option {
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
	if observer.resolvePermissionMode == nil {
		observer.resolvePermissionMode = defaultPermissionModeResolver(observer.homePaths, observer.workspaceResolver)
	}

	if observer.registry == nil {
		if err := aghconfig.EnsureHomeLayout(observer.homePaths); err != nil {
			return nil, fmt.Errorf("observe: ensure home layout: %w", err)
		}

		registry, err := store.OpenGlobalDB(ctx, observer.homePaths.DatabaseFile)
		if err != nil {
			return nil, fmt.Errorf("observe: open global database: %w", err)
		}
		observer.registry = registry
	}

	return observer, nil
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
			o.logger.Warn("observe: resolve permission mode failed", "session_id", info.ID, "agent_name", info.AgentName, "workspace_id", info.WorkspaceID, "error", err)
		} else {
			snapshot.permissionMode = strings.TrimSpace(permissionMode)
		}
	}

	o.trackSession(info.ID, snapshot)

	if err := o.registry.RegisterSession(ctx, sessionInfoFromSession(info)); err != nil {
		o.logger.Warn("observe: register session failed", "session_id", info.ID, "agent_name", info.AgentName, "workspace_id", info.WorkspaceID, "error", err)
	}
}

// OnSessionStopped updates the session state in the global observability database.
func (o *Observer) OnSessionStopped(ctx context.Context, sess *session.Session) {
	info := sess.Info()

	if err := o.registry.UpdateSessionState(ctx, store.SessionStateUpdate{
		ID:           info.ID,
		State:        string(info.State),
		ACPSessionID: stringPointer(info.ACPSessionID),
		UpdatedAt:    info.UpdatedAt,
	}); err != nil {
		o.logger.Warn("observe: update session state failed", "session_id", info.ID, "agent_name", info.AgentName, "workspace_id", info.WorkspaceID, "state", info.State, "error", err)
	}

	o.untrackSession(info.ID)
}

// OnAgentEvent records one lightweight cross-session event summary and any derived aggregates.
func (o *Observer) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		o.logger.Warn("observe: skipped agent event with empty session id", "event_type", event.Type)
		return
	}

	snapshot, ok := o.sessionSnapshot(id)
	if !ok {
		o.logger.Warn("observe: skipped agent event for unknown session", "session_id", id, "event_type", event.Type)
		return
	}
	if strings.TrimSpace(event.Type) == "" {
		o.logger.Warn("observe: skipped agent event with empty type", "session_id", id, "agent_name", snapshot.agentName, "workspace_id", snapshot.workspaceID)
		return
	}

	timestamp := event.Timestamp
	if timestamp.IsZero() {
		timestamp = o.now()
	}

	if err := o.registry.WriteEventSummary(ctx, store.EventSummary{
		SessionID: id,
		Type:      strings.TrimSpace(event.Type),
		AgentName: snapshot.agentName,
		Summary:   summarizeEvent(event),
		Timestamp: timestamp,
	}); err != nil {
		o.logger.Warn("observe: write event summary failed", "session_id", id, "agent_name", snapshot.agentName, "workspace_id", snapshot.workspaceID, "event_type", event.Type, "error", err)
	}

	if shouldAggregateUsage(event) {
		usageTimestamp := timestamp
		if !event.Usage.Timestamp.IsZero() {
			usageTimestamp = event.Usage.Timestamp
		}
		if err := o.registry.UpdateTokenStats(ctx, store.TokenStatsUpdate{
			SessionID:    id,
			AgentName:    snapshot.agentName,
			InputTokens:  event.Usage.InputTokens,
			OutputTokens: event.Usage.OutputTokens,
			TotalTokens:  event.Usage.TotalTokens,
			CostAmount:   event.Usage.CostAmount,
			CostCurrency: event.Usage.CostCurrency,
			Turns:        1,
			UpdatedAt:    usageTimestamp,
		}); err != nil {
			o.logger.Warn("observe: update token stats failed", "session_id", id, "agent_name", snapshot.agentName, "workspace_id", snapshot.workspaceID, "turn_id", event.TurnID, "error", err)
		}
	}

	if strings.TrimSpace(event.Type) != acp.EventTypePermission {
		return
	}

	policyUsed := strings.TrimSpace(snapshot.permissionMode)
	if policyUsed == "" {
		o.logger.Warn("observe: skipped permission log without resolved policy", "session_id", id, "agent_name", snapshot.agentName, "workspace_id", snapshot.workspaceID)
		return
	}
	if strings.TrimSpace(event.Decision) == "" {
		return
	}

	if err := o.registry.WritePermissionLog(ctx, store.PermissionLogEntry{
		SessionID:  id,
		AgentName:  snapshot.agentName,
		Action:     strings.TrimSpace(event.Action),
		Resource:   strings.TrimSpace(event.Resource),
		Decision:   strings.TrimSpace(event.Decision),
		PolicyUsed: policyUsed,
		Timestamp:  timestamp,
	}); err != nil {
		o.logger.Warn("observe: write permission log failed", "session_id", id, "agent_name", snapshot.agentName, "workspace_id", snapshot.workspaceID, "error", err)
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

func defaultPermissionModeResolver(homePaths aghconfig.HomePaths, resolver workspacepkg.WorkspaceResolver) PermissionModeResolver {
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

func sessionInfoFromSession(info *session.SessionInfo) store.SessionInfo {
	if info == nil {
		return store.SessionInfo{}
	}

	return store.SessionInfo{
		ID:           info.ID,
		Name:         info.Name,
		AgentName:    info.AgentName,
		WorkspaceID:  info.WorkspaceID,
		SessionType:  string(info.Type),
		State:        string(info.State),
		ACPSessionID: stringPointer(info.ACPSessionID),
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

func summarizeEvent(event acp.AgentEvent) string {
	candidates := []string{
		strings.TrimSpace(event.Text),
		strings.TrimSpace(event.Title),
		strings.TrimSpace(event.Error),
		strings.TrimSpace(event.Resource),
		strings.TrimSpace(event.StopReason),
		strings.TrimSpace(event.ToolCallID),
	}
	if strings.TrimSpace(event.Type) == acp.EventTypePermission {
		candidates = append([]string{
			strings.TrimSpace(event.Title),
			strings.TrimSpace(event.Resource),
			strings.TrimSpace(event.Decision),
		}, candidates...)
	}

	for _, candidate := range candidates {
		if candidate != "" {
			return truncateSummary(candidate)
		}
	}

	if len(event.Raw) > 0 {
		return truncateSummary(string(event.Raw))
	}
	return ""
}

func truncateSummary(summary string) string {
	const maxRunes = 240

	clean := strings.TrimSpace(summary)
	if clean == "" {
		return ""
	}

	runes := []rune(clean)
	if len(runes) <= maxRunes {
		return clean
	}

	return string(runes[:maxRunes-3]) + "..."
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
