// Package store provides shared persistence types, validation, and helper primitives.
package store

import (
	"context"
	"errors"
	"path/filepath"
	"time"
)

const (
	// SessionDatabaseName is the filename for per-session event storage.
	SessionDatabaseName = "events.db"
	// GlobalDatabaseName is the filename for the global AGH index database.
	GlobalDatabaseName = "agh.db"
	// SessionMetaName is the filename for quick session metadata lookups.
	SessionMetaName = "meta.json"
	// DefaultSQLiteBusyTimeoutMS is the shared SQLite busy timeout in milliseconds.
	DefaultSQLiteBusyTimeoutMS = defaultBusyTimeoutMS

	sqliteDriverName       = "sqlite"
	defaultBusyTimeoutMS   = 5000
	defaultMaxOpenConns    = 8
	defaultMaxIdleConns    = 8
	defaultWriteBufferSize = 256
	defaultDrainTimeout    = 5 * time.Second
)

var (
	// ErrClosed reports that a session database no longer accepts writes.
	ErrClosed = errors.New("store: session database closed")
	// ErrDrainTimeout reports that shutdown timed out before queued writes drained.
	ErrDrainTimeout = errors.New("store: writer drain timeout")
	// ErrNetworkConversationNotFound reports a missing network conversation container.
	ErrNetworkConversationNotFound = errors.New("store: network conversation not found")
	// ErrNetworkDirectRoomCollision reports a direct_id bound to another peer pair.
	ErrNetworkDirectRoomCollision = errors.New("store: network direct room collision")
	// ErrNetworkWorkContainerMismatch reports a work_id used outside its bound container.
	ErrNetworkWorkContainerMismatch = errors.New("store: network work container mismatch")
	// ErrNetworkWorkClosed reports a non-duplicate message for terminal work.
	ErrNetworkWorkClosed = errors.New("store: network work closed")
	// ErrNetworkCursorInvalid reports a malformed or query-mismatched network list cursor.
	ErrNetworkCursorInvalid = errors.New("store: network cursor invalid")
	// ErrNetworkChannelExists reports an attempted create for an existing workspace channel.
	ErrNetworkChannelExists = errors.New("store: network channel already exists")
)

// EventRecorder captures session events and token usage in the per-session database.
type EventRecorder interface {
	Record(ctx context.Context, event SessionEvent) error
	RecordTokenUsage(ctx context.Context, usage TokenUsage) error
	Query(ctx context.Context, query EventQuery) ([]SessionEvent, error)
	History(ctx context.Context, query EventQuery) ([]TurnHistory, error)
	Close(ctx context.Context) error
}

// SessionCatalog manages global session index records.
type SessionCatalog interface {
	RegisterSession(ctx context.Context, session SessionInfo) error
	UpdateSessionState(ctx context.Context, update SessionStateUpdate) error
	DeleteSession(ctx context.Context, sessionID string) error
	ListSessions(ctx context.Context, query SessionListQuery) ([]SessionInfo, error)
	AttachSession(ctx context.Context, req SessionAttachRequest) (SessionAttach, error)
	ReconcileSessions(ctx context.Context, sessions []SessionInfo) (ReconcileResult, error)
}

// SessionTranscriptEpochStore manages destructive transcript reset epochs.
type SessionTranscriptEpochStore interface {
	SessionTranscriptEpoch(ctx context.Context, sessionID string) (int64, error)
	EnsureSessionTranscriptEpoch(ctx context.Context, update SessionTranscriptEpochUpdate) (int64, error)
}

// EventSummaryStore manages persisted observability event summaries.
type EventSummaryStore interface {
	WriteEventSummary(ctx context.Context, summary EventSummary) error
	ListEventSummaries(ctx context.Context, query EventSummaryQuery) ([]EventSummary, error)
}

// TokenStatsStore manages aggregated token usage rows.
type TokenStatsStore interface {
	UpdateTokenStats(ctx context.Context, update TokenStatsUpdate) error
	ListTokenStats(ctx context.Context, query TokenStatsQuery) ([]TokenStats, error)
}

// PermissionLogStore manages permission decision audit entries.
type PermissionLogStore interface {
	WritePermissionLog(ctx context.Context, entry PermissionLogEntry) error
	ListPermissionLog(ctx context.Context, query PermissionLogQuery) ([]PermissionLogEntry, error)
}

// NetworkAuditStore manages network message audit entries.
type NetworkAuditStore interface {
	WriteNetworkAudit(ctx context.Context, entry NetworkAuditEntry) error
	ListNetworkAudit(ctx context.Context, query NetworkAuditQuery) ([]NetworkAuditEntry, error)
}

// NetworkChannelStore manages durable network channel metadata.
type NetworkChannelStore interface {
	CreateNetworkChannel(ctx context.Context, entry NetworkChannelEntry) error
	WriteNetworkChannel(ctx context.Context, entry NetworkChannelEntry) error
	GetNetworkChannel(ctx context.Context, ref NetworkChannelRef) (NetworkChannelEntry, error)
	ListNetworkChannels(ctx context.Context, query NetworkChannelQuery) ([]NetworkChannelEntry, error)
	PatchNetworkChannel(ctx context.Context, ref NetworkChannelRef, patch NetworkChannelPatch) error
	DeleteNetworkChannel(ctx context.Context, ref NetworkChannelRef) error
}

// NetworkChannelProjectionStore reads materialized channel timeline totals.
type NetworkChannelProjectionStore interface {
	ListNetworkChannelProjections(
		ctx context.Context,
		query NetworkChannelProjectionQuery,
	) ([]NetworkChannelProjection, error)
	ListNetworkChannelKindCounts(
		ctx context.Context,
		ref NetworkChannelRef,
	) ([]NetworkChannelKindCount, error)
}

// NetworkRecentStore reads cross-channel recent conversation summaries.
type NetworkRecentStore interface {
	ListNetworkRecents(ctx context.Context, query NetworkRecentQuery) ([]NetworkRecentSummary, error)
}

// AppMetadataStore manages a small global key-value table for instance-level flags.
type AppMetadataStore interface {
	GetAppMetadata(ctx context.Context, key string) (string, bool, error)
	SetAppMetadata(ctx context.Context, key string, value string) error
	DeleteAppMetadata(ctx context.Context, key string) error
}

// OnboardingStatus describes the global first-run onboarding completion state.
type OnboardingStatus struct {
	Completed   bool
	CompletedAt string
}

// OnboardingStore manages the domain lifecycle for first-run onboarding.
type OnboardingStore interface {
	GetOnboardingStatus(ctx context.Context) (OnboardingStatus, error)
	CompleteOnboarding(ctx context.Context, completedAt string) (OnboardingStatus, error)
	ResetOnboarding(ctx context.Context) (OnboardingStatus, error)
}

// NetworkMessageStore manages persisted network timeline messages.
type NetworkMessageStore interface {
	WriteNetworkMessage(ctx context.Context, entry NetworkMessageEntry) error
	ListNetworkMessages(ctx context.Context, query NetworkMessageQuery) ([]NetworkMessageEntry, error)
}

// NetworkConversationStore manages durable conversation containers and work rows.
type NetworkConversationStore interface {
	ResolveDirectRoom(ctx context.Context, entry NetworkDirectRoomEntry) (NetworkDirectRoomSummary, error)
	WriteConversationMessage(
		ctx context.Context,
		entry NetworkConversationMessage,
	) (NetworkConversationWriteResult, error)
	ListThreads(ctx context.Context, ref NetworkChannelRef, query NetworkThreadQuery) (NetworkThreadPage, error)
	GetThread(ctx context.Context, ref NetworkChannelRef, threadID string) (NetworkThreadSummary, error)
	ListDirectRooms(
		ctx context.Context,
		ref NetworkChannelRef,
		query NetworkDirectRoomQuery,
	) (NetworkDirectRoomPage, error)
	GetDirectRoom(ctx context.Context, ref NetworkChannelRef, directID string) (NetworkDirectRoomSummary, error)
	ListConversationMessages(
		ctx context.Context,
		ref NetworkConversationRef,
		query NetworkConversationMessageQuery,
	) ([]NetworkConversationMessage, error)
	GetWork(ctx context.Context, workspaceID string, workID string) (NetworkWorkEntry, error)
}

// NetworkAcceptanceStore owns atomic message acceptance and wake accounting.
type NetworkAcceptanceStore interface {
	AcceptNetworkMessage(
		ctx context.Context,
		req AcceptNetworkMessageRequest,
	) (AcceptNetworkMessageResult, error)
}

// NetworkInboxStore reads immutable delivered dispositions by durable session identity.
type NetworkInboxStore interface {
	ListNetworkInbox(
		ctx context.Context,
		workspaceID string,
		sessionID string,
		channel string,
		limit int,
	) ([]NetworkMessageEntry, error)
}

// NetworkTaskStatusProjectionStore persists typed task transitions for subscribed recipients.
type NetworkTaskStatusProjectionStore interface {
	ListNetworkTaskStatusProjections(
		ctx context.Context,
		query NetworkTaskStatusProjectionQuery,
	) ([]NetworkTaskStatusProjection, error)
}

// NetworkCausationStore resolves durable wake ancestry for bounded reply chains.
type NetworkCausationStore interface {
	ResolveNetworkCausation(
		ctx context.Context,
		workspaceID string,
		causationID string,
	) (rootID string, depth int, found bool, err error)
}

// NetworkWakeQueueStore reads committed wake work for daemon execution.
type NetworkWakeQueueStore interface {
	ListQueuedNetworkWakes(
		ctx context.Context,
		limit int,
	) ([]CommittedNetworkNotification, error)
	LoadNetworkWake(
		ctx context.Context,
		workspaceID string,
		wakeID string,
	) (WakeReservation, []NetworkMessageEntry, error)
}

// NetworkUsageStore reads truthful wake-ledger detail and aggregates.
type NetworkUsageStore interface {
	GetNetworkUsage(ctx context.Context, query NetworkUsageQuery) (NetworkUsageReport, error)
}

// NetworkPreferenceStore manages network delivery preferences and network-task links.
type NetworkPreferenceStore interface {
	PutNetworkSubscription(ctx context.Context, entry NetworkSubscriptionEntry) error
	PutNetworkSubscriptionWithChannel(
		ctx context.Context,
		channel NetworkChannelEntry,
		entry NetworkSubscriptionEntry,
	) error
	ListNetworkSubscriptions(ctx context.Context, query NetworkSubscriptionQuery) ([]NetworkSubscriptionEntry, error)
	DeleteNetworkSubscription(ctx context.Context, ref NetworkSubscriptionRef) error
	PutNetworkTaskThreadOrigin(ctx context.Context, origin NetworkTaskThreadOrigin) error
	ListNetworkTaskThreadOrigins(
		ctx context.Context,
		query NetworkTaskThreadOriginQuery,
	) ([]NetworkTaskThreadOrigin, error)
	PutTaskDesignationRollup(ctx context.Context, rollup TaskDesignationRollup) error
	ListTaskDesignationRollups(ctx context.Context, query TaskDesignationRollupQuery) ([]TaskDesignationRollup, error)
}

// SessionRegistry composes the global persistence surfaces used by runtime consumers.
type SessionRegistry interface {
	SessionCatalog
	EventSummaryStore
	TokenStatsStore
	PermissionLogStore
	NetworkAuditStore
	NetworkChannelStore
	NetworkChannelProjectionStore
	NetworkRecentStore
	NetworkMessageStore
	NetworkConversationStore
	NetworkPreferenceStore
	Close(ctx context.Context) error
}

// SessionDBFile returns the canonical events database path for a session directory.
func SessionDBFile(sessionDir string) string {
	return filepath.Join(sessionDir, SessionDatabaseName)
}

// SessionMetaFile returns the canonical metadata file path for a session directory.
func SessionMetaFile(sessionDir string) string {
	return filepath.Join(sessionDir, SessionMetaName)
}
