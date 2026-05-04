// Package contract defines the canonical shared daemon API request and response DTOs.
package contract

import (
	"encoding/json"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// CreateSessionRequest is the shared session creation request payload.
type CreateSessionRequest struct {
	AgentName     string `json:"agent_name,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Name          string `json:"name,omitempty"`
	Workspace     string `json:"workspace,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	Channel       string `json:"channel,omitempty"`
}

// ApproveSessionRequest is the interactive permission approval payload.
type ApproveSessionRequest struct {
	RequestID string `json:"request_id"`
	TurnID    string `json:"turn_id"`
	Decision  string `json:"decision"`
}

// SessionPayload is the shared session response payload.
type SessionPayload struct {
	ID            string        `json:"id"`
	Name          string        `json:"name,omitempty"`
	AgentName     string        `json:"agent_name"`
	Provider      string        `json:"provider"`
	WorkspaceID   string        `json:"workspace_id,omitempty"`
	WorkspacePath string        `json:"workspace_path,omitempty"`
	Channel       string        `json:"channel,omitempty"`
	Type          session.Type  `json:"type,omitempty"`
	State         session.State `json:"state"`
	// StopReason is the session-level stop classification, distinct from AgentEventPayload.StopReason.
	StopReason store.StopReason `json:"stop_reason,omitempty"`
	// StopDetail is the session-level stop context paired with StopReason.
	StopDetail   string                  `json:"stop_detail,omitempty"`
	Failure      *SessionFailurePayload  `json:"failure,omitempty"`
	ACPSessionID string                  `json:"acp_session_id,omitempty"`
	ACPCaps      *ACPCapsPayload         `json:"acp_caps,omitempty"`
	Activity     *RuntimeActivityPayload `json:"activity,omitempty"`
	Sandbox      *SessionSandboxPayload  `json:"sandbox,omitempty"`
	Lineage      *SessionLineagePayload  `json:"lineage,omitempty"`
	Health       *SessionHealthPayload   `json:"health,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
}

// SessionFailurePayload is the redacted lifecycle failure diagnostic shared by
// session read paths, event streams, and health summaries.
type SessionFailurePayload struct {
	Kind            store.FailureKind `json:"kind"`
	Summary         string            `json:"summary,omitempty"`
	CrashBundlePath string            `json:"crash_bundle_path,omitempty"`
}

// RuntimeActivityPayload is the shared JSON representation of active prompt supervision state.
type RuntimeActivityPayload struct {
	TurnID             string     `json:"turn_id,omitempty"`
	TurnSource         string     `json:"turn_source,omitempty"`
	TurnStartedAt      *time.Time `json:"turn_started_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
	LastActivityKind   string     `json:"last_activity_kind,omitempty"`
	LastActivityDetail string     `json:"last_activity_detail,omitempty"`
	CurrentTool        string     `json:"current_tool,omitempty"`
	ToolCallID         string     `json:"tool_call_id,omitempty"`
	LastProgressAt     *time.Time `json:"last_progress_at,omitempty"`
	IterationCurrent   int        `json:"iteration_current"`
	IterationMax       int        `json:"iteration_max"`
	IdleSeconds        int64      `json:"idle_seconds"`
	ElapsedSeconds     int64      `json:"elapsed_seconds"`
}

// SessionSandboxPayload is the shared session sandbox response payload.
type SessionSandboxPayload struct {
	SandboxID         string          `json:"sandbox_id,omitempty"`
	Backend           string          `json:"backend,omitempty"`
	Profile           string          `json:"profile,omitempty"`
	State             string          `json:"state,omitempty"`
	InstanceID        string          `json:"instance_id,omitempty"`
	LastSyncError     string          `json:"last_sync_error,omitempty"`
	ProviderStateJSON json.RawMessage `json:"provider_state_json,omitempty"`
}

// ACPCapsPayload is the JSON representation of ACP capabilities.
type ACPCapsPayload struct {
	SupportsLoadSession bool     `json:"supports_load_session"`
	SupportedModes      []string `json:"supported_modes,omitempty"`
	SupportedModels     []string `json:"supported_models,omitempty"`
}

// SessionEventPayload is the shared session event response payload.
type SessionEventPayload struct {
	ID            string                 `json:"id"`
	SessionID     string                 `json:"session_id"`
	Sequence      int64                  `json:"sequence"`
	TurnID        string                 `json:"turn_id"`
	Type          string                 `json:"type"`
	AgentName     string                 `json:"agent_name"`
	WorkspaceID   string                 `json:"workspace_id,omitempty"`
	WorkspacePath string                 `json:"workspace_path,omitempty"`
	Content       json.RawMessage        `json:"content"`
	StopReason    store.StopReason       `json:"stop_reason,omitempty"`
	StopDetail    string                 `json:"stop_detail,omitempty"`
	Failure       *SessionFailurePayload `json:"failure,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

// TurnHistoryPayload is the shared turn history response payload.
type TurnHistoryPayload struct {
	TurnID string                `json:"turn_id"`
	Events []SessionEventPayload `json:"events"`
}

// SessionRepairPayload reports one dry-run or persisted session repair pass.
type SessionRepairPayload struct {
	SessionID string                       `json:"session_id"`
	Issues    []SessionRepairIssuePayload  `json:"issues"`
	Actions   []SessionRepairActionPayload `json:"actions"`
	Persisted bool                         `json:"persisted"`
}

// SessionRepairIssuePayload is one inconsistency found during session repair.
type SessionRepairIssuePayload struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	TurnID   string `json:"turn_id,omitempty"`
	EventID  string `json:"event_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// SessionRepairActionPayload is one append-only repair action.
type SessionRepairActionPayload struct {
	Code       string `json:"code"`
	TurnID     string `json:"turn_id"`
	EventID    string `json:"event_id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	Persisted  bool   `json:"persisted"`
}

// AgentPayload is the shared agent definition response payload.
type AgentPayload struct {
	Name        string                   `json:"name"`
	Provider    string                   `json:"provider"`
	Command     string                   `json:"command,omitempty"`
	Model       string                   `json:"model,omitempty"`
	Tools       []string                 `json:"tools,omitempty"`
	Toolsets    []string                 `json:"toolsets,omitempty"`
	DenyTools   []string                 `json:"deny_tools,omitempty"`
	Permissions string                   `json:"permissions,omitempty"`
	MCPServers  []AgentMCPServerJSON     `json:"mcp_servers,omitempty"`
	Prompt      string                   `json:"prompt"`
	Diagnostics []AgentDiagnosticPayload `json:"diagnostics,omitempty"`
}

// AgentDiagnosticPayload reports one malformed agent definition encountered during discovery.
type AgentDiagnosticPayload struct {
	Path      string `json:"path"`
	ErrorKind string `json:"error_kind"`
	Message   string `json:"message"`
}

// AgentMCPServerJSON is the shared MCP server response payload.
type AgentMCPServerJSON struct {
	Name      string                        `json:"name"`
	Transport string                        `json:"transport,omitempty"`
	Command   string                        `json:"command,omitempty"`
	Args      []string                      `json:"args,omitempty"`
	Env       map[string]string             `json:"env,omitempty"`
	SecretEnv map[string]string             `json:"secret_env,omitempty"`
	URL       string                        `json:"url,omitempty"`
	Auth      *SettingsMCPAuthConfigPayload `json:"auth,omitempty"`
}

// AgentEventPayload is the shared raw agent-event streaming payload.
type AgentEventPayload struct {
	Type       string                  `json:"type"`
	SessionID  string                  `json:"session_id,omitempty"`
	TurnID     string                  `json:"turn_id,omitempty"`
	RequestID  string                  `json:"request_id,omitempty"`
	Timestamp  time.Time               `json:"timestamp"`
	Text       string                  `json:"text,omitempty"`
	Title      string                  `json:"title,omitempty"`
	ToolCallID string                  `json:"tool_call_id,omitempty"`
	StopReason string                  `json:"stop_reason,omitempty"`
	Action     string                  `json:"action,omitempty"`
	Resource   string                  `json:"resource,omitempty"`
	Decision   string                  `json:"decision,omitempty"`
	Error      string                  `json:"error,omitempty"`
	Failure    *SessionFailurePayload  `json:"failure,omitempty"`
	Usage      *TokenUsagePayload      `json:"usage,omitempty"`
	Runtime    *RuntimeActivityPayload `json:"runtime,omitempty"`
	Raw        json.RawMessage         `json:"raw,omitempty"`
}

// TokenUsagePayload is the shared token-usage response payload.
type TokenUsagePayload struct {
	TurnID           string    `json:"turn_id,omitempty"`
	InputTokens      *int64    `json:"input_tokens,omitempty"`
	OutputTokens     *int64    `json:"output_tokens,omitempty"`
	TotalTokens      *int64    `json:"total_tokens,omitempty"`
	ThoughtTokens    *int64    `json:"thought_tokens,omitempty"`
	CacheReadTokens  *int64    `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens *int64    `json:"cache_write_tokens,omitempty"`
	ContextUsed      *int64    `json:"context_used,omitempty"`
	ContextSize      *int64    `json:"context_size,omitempty"`
	CostAmount       *float64  `json:"cost_amount,omitempty"`
	CostCurrency     *string   `json:"cost_currency,omitempty"`
	Timestamp        time.Time `json:"timestamp"`
}

// ObserveEventPayload is the shared observability event response payload.
type ObserveEventPayload struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	AgentName string    `json:"agent_name"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ObserveHealthPayload is the shared observability health response payload.
type ObserveHealthPayload struct {
	Status             string                          `json:"status"`
	UptimeSeconds      int64                           `json:"uptime_seconds"`
	ActiveSessions     int                             `json:"active_sessions"`
	ActiveAgents       int                             `json:"active_agents"`
	GlobalDBSizeBytes  int64                           `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64                           `json:"session_db_size_bytes"`
	Persistence        ObservePersistenceHealthPayload `json:"persistence"`
	Retention          ObserveRetentionHealthPayload   `json:"retention"`
	Failures           ObserveFailureHealthPayload     `json:"failures"`
	AgentProbes        []AgentProbeHealthPayload       `json:"agent_probes,omitempty"`
	Bridges            BridgeAggregateHealthPayload    `json:"bridges"`
	Activities         []SessionActivityHealthPayload  `json:"activities,omitempty"`
	Version            string                          `json:"version"`
}

// ObserveFailureHealthPayload summarizes persisted lifecycle failures.
type ObserveFailureHealthPayload struct {
	Status string                        `json:"status"`
	Total  int                           `json:"total"`
	ByKind map[store.FailureKind]int     `json:"by_kind,omitempty"`
	Recent []SessionFailureHealthPayload `json:"recent,omitempty"`
}

// SessionFailureHealthPayload exposes one compact lifecycle failure health row.
type SessionFailureHealthPayload struct {
	SessionID       string            `json:"session_id"`
	AgentName       string            `json:"agent_name,omitempty"`
	Provider        string            `json:"provider,omitempty"`
	WorkspaceID     string            `json:"workspace_id,omitempty"`
	State           string            `json:"state,omitempty"`
	FailureKind     store.FailureKind `json:"failure_kind"`
	Summary         string            `json:"summary,omitempty"`
	CrashBundlePath string            `json:"crash_bundle_path,omitempty"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// AgentProbeHealthPayload exposes one downstream ACP command probe result.
type AgentProbeHealthPayload struct {
	AgentName  string    `json:"agent_name,omitempty"`
	Provider   string    `json:"provider,omitempty"`
	Command    string    `json:"command,omitempty"`
	Executable string    `json:"executable,omitempty"`
	Status     string    `json:"status"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
	DurationMS int64     `json:"duration_ms"`
}

// ObservePersistenceHealthPayload captures store health fields shared by
// lifecycle, memory, and operator diagnostics.
type ObservePersistenceHealthPayload struct {
	Status             string `json:"status"`
	GlobalDBSizeBytes  int64  `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64  `json:"session_db_size_bytes"`
}

// ObserveRetentionHealthPayload captures the observable state of configured
// retention sweeps.
type ObserveRetentionHealthPayload struct {
	Enabled                  bool       `json:"enabled"`
	RetentionDays            int        `json:"retention_days"`
	SweepIntervalSeconds     int64      `json:"sweep_interval_seconds"`
	LastSweepStatus          string     `json:"last_sweep_status"`
	LastSweepAt              *time.Time `json:"last_sweep_at,omitempty"`
	LastCutoffAt             *time.Time `json:"last_cutoff_at,omitempty"`
	LastSweepError           string     `json:"last_sweep_error,omitempty"`
	DeletedEventSummaries    int64      `json:"deleted_event_summaries"`
	DeletedTokenStats        int64      `json:"deleted_token_stats"`
	DeletedPermissionLogRows int64      `json:"deleted_permission_log_rows"`
}

// SessionActivityHealthPayload exposes active runtime supervision state in the
// observability health response.
type SessionActivityHealthPayload struct {
	SessionID          string     `json:"session_id"`
	TurnID             string     `json:"turn_id,omitempty"`
	TurnSource         string     `json:"turn_source,omitempty"`
	TurnStartedAt      *time.Time `json:"turn_started_at,omitempty"`
	LastActivityAt     *time.Time `json:"last_activity_at,omitempty"`
	LastActivityKind   string     `json:"last_activity_kind,omitempty"`
	LastActivityDetail string     `json:"last_activity_detail,omitempty"`
	CurrentTool        string     `json:"current_tool,omitempty"`
	ToolCallID         string     `json:"tool_call_id,omitempty"`
	LastProgressAt     *time.Time `json:"last_progress_at,omitempty"`
	IterationCurrent   int        `json:"iteration_current"`
	IterationMax       int        `json:"iteration_max"`
	IdleSeconds        int64      `json:"idle_seconds"`
	ElapsedSeconds     int64      `json:"elapsed_seconds"`
	Status             string     `json:"status"`
	StallState         string     `json:"stall_state,omitempty"`
	StallReason        string     `json:"stall_reason,omitempty"`
}

// HookCatalogQuery captures the shared resolved-hook catalog filters.
type HookCatalogQuery struct {
	Workspace string
	Agent     string
	Event     string
	Source    string
	Mode      string
}

// HookRunsQuery captures the shared hook execution history filters.
type HookRunsQuery struct {
	Session string
	Event   string
	Outcome string
	Since   string
	Last    int
}

// HookEventsQuery captures the shared hook taxonomy filters.
type HookEventsQuery struct {
	Family   string
	SyncOnly bool
}

// HookCatalogPayload is the shared resolved-hook catalog response payload.
type HookCatalogPayload struct {
	Order        int                  `json:"order"`
	Name         string               `json:"name"`
	Event        string               `json:"event"`
	Source       string               `json:"source"`
	SkillSource  string               `json:"skill_source,omitempty"`
	Mode         string               `json:"mode"`
	Required     bool                 `json:"required"`
	Priority     int                  `json:"priority"`
	TimeoutMS    int64                `json:"timeout_ms,omitempty"`
	ExecutorKind string               `json:"executor_kind,omitempty"`
	Matcher      hookspkg.HookMatcher `json:"matcher"`
	Metadata     map[string]string    `json:"metadata,omitempty"`
}

// HookRunPayload is the shared hook execution history response payload.
type HookRunPayload struct {
	HookName      string          `json:"hook_name"`
	Event         string          `json:"event"`
	Source        string          `json:"source"`
	Mode          string          `json:"mode"`
	DurationMS    int64           `json:"duration_ms"`
	Outcome       string          `json:"outcome"`
	DispatchDepth int             `json:"dispatch_depth"`
	PatchApplied  json.RawMessage `json:"patch_applied,omitempty"`
	Error         string          `json:"error,omitempty"`
	Required      bool            `json:"required,omitempty"`
	RecordedAt    time.Time       `json:"recorded_at"`
}

// HookEventPayload is the shared hook taxonomy response payload.
type HookEventPayload struct {
	Event         string `json:"event"`
	Family        string `json:"family"`
	SyncEligible  bool   `json:"sync_eligible"`
	PayloadSchema string `json:"payload_schema"`
	PatchSchema   string `json:"patch_schema,omitempty"`
}

// DaemonStatusPayload is the shared daemon status response payload.
type DaemonStatusPayload struct {
	Status         string                `json:"status"`
	PID            int                   `json:"pid"`
	StartedAt      time.Time             `json:"started_at"`
	Socket         string                `json:"socket"`
	HTTPHost       string                `json:"http_host"`
	HTTPPort       int                   `json:"http_port"`
	UserHomeDir    string                `json:"user_home_dir"`
	ActiveSessions int                   `json:"active_sessions"`
	TotalSessions  int                   `json:"total_sessions"`
	Version        string                `json:"version,omitempty"`
	Network        *NetworkStatusPayload `json:"network,omitempty"`
}

// NetworkStatusPayload is the shared network diagnostics response payload.
type NetworkStatusPayload struct {
	Enabled                  bool                            `json:"enabled"`
	Status                   string                          `json:"status"`
	ConfiguredDefaultChannel string                          `json:"configured_default_channel,omitempty"`
	EffectiveDefaultChannel  string                          `json:"effective_default_channel,omitempty"`
	EffectiveDefaultSource   string                          `json:"effective_default_source,omitempty"`
	ListenerHost             string                          `json:"listener_host,omitempty"`
	ListenerPort             int                             `json:"listener_port,omitempty"`
	LocalPeers               int                             `json:"local_peers,omitempty"`
	RemotePeers              int                             `json:"remote_peers,omitempty"`
	Channels                 int                             `json:"channels,omitempty"`
	QueuedMessages           int                             `json:"queued_messages,omitempty"`
	QueuedSessions           int                             `json:"queued_sessions,omitempty"`
	DeliveryWorkers          int                             `json:"delivery_workers,omitempty"`
	MessagesSent             int64                           `json:"messages_sent,omitempty"`
	MessagesReceived         int64                           `json:"messages_received,omitempty"`
	MessagesRejected         int64                           `json:"messages_rejected,omitempty"`
	MessagesDelivered        int64                           `json:"messages_delivered,omitempty"`
	WorkflowTaggedEvents     int64                           `json:"workflow_tagged_events,omitempty"`
	HandoffTaggedEvents      int64                           `json:"handoff_tagged_events,omitempty"`
	LastDisconnect           string                          `json:"last_disconnect,omitempty"`
	DeclaredChannels         []DeclaredNetworkChannelPayload `json:"declared_channels,omitempty"`
	KindMetrics              []NetworkKindMetricPayload      `json:"kind_metrics,omitempty"`
}

// NetworkKindMetricPayload is the per-kind network runtime metric snapshot.
type NetworkKindMetricPayload struct {
	Kind      string `json:"kind"`
	Sent      int64  `json:"sent,omitempty"`
	Received  int64  `json:"received,omitempty"`
	Rejected  int64  `json:"rejected,omitempty"`
	Delivered int64  `json:"delivered,omitempty"`
}

// NetworkSendRequest is the shared daemon network send request payload.
type NetworkSendRequest struct {
	SessionID     string                     `json:"session_id"`
	Channel       string                     `json:"channel"`
	Kind          string                     `json:"kind"`
	To            string                     `json:"to,omitempty"`
	Body          json.RawMessage            `json:"body"`
	InteractionID string                     `json:"interaction_id,omitempty"`
	ReplyTo       string                     `json:"reply_to,omitempty"`
	TraceID       string                     `json:"trace_id,omitempty"`
	CausationID   string                     `json:"causation_id,omitempty"`
	ExpiresAt     *int64                     `json:"expires_at,omitempty"`
	ID            string                     `json:"id,omitempty"`
	Ext           map[string]json.RawMessage `json:"ext,omitempty"`
}

// NetworkSendPayload is the shared daemon network send response payload.
type NetworkSendPayload struct {
	ID            string                     `json:"id"`
	SessionID     string                     `json:"session_id"`
	Channel       string                     `json:"channel"`
	Kind          string                     `json:"kind"`
	To            string                     `json:"to,omitempty"`
	InteractionID string                     `json:"interaction_id,omitempty"`
	ReplyTo       string                     `json:"reply_to,omitempty"`
	TraceID       string                     `json:"trace_id,omitempty"`
	CausationID   string                     `json:"causation_id,omitempty"`
	ExpiresAt     *int64                     `json:"expires_at,omitempty"`
	Ext           map[string]json.RawMessage `json:"ext,omitempty"`
}

// CreateNetworkChannelRequest is the shared network channel creation payload.
type CreateNetworkChannelRequest struct {
	Channel     string   `json:"channel"`
	WorkspaceID string   `json:"workspace_id"`
	Purpose     string   `json:"purpose"`
	AgentNames  []string `json:"agent_names"`
}

// NetworkCapabilityBriefPayload is the shared brief discovery projection for
// one peer capability.
type NetworkCapabilityBriefPayload struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

// NetworkCapabilityPayload is the shared rich capability payload surfaced by
// daemon APIs.
type NetworkCapabilityPayload struct {
	ID                string   `json:"id"`
	Summary           string   `json:"summary"`
	Outcome           string   `json:"outcome"`
	Version           string   `json:"version,omitempty"`
	Digest            string   `json:"digest,omitempty"`
	ContextNeeded     []string `json:"context_needed,omitempty"`
	ArtifactsExpected []string `json:"artifacts_expected,omitempty"`
	ExecutionOutline  []string `json:"execution_outline,omitempty"`
	Constraints       []string `json:"constraints,omitempty"`
	Examples          []string `json:"examples,omitempty"`
	Requirements      []string `json:"requirements,omitempty"`
}

// NetworkCapabilityCatalogPayload is the shared rich discovery catalog surfaced
// by peer-detail APIs when explicit rich capability data is available.
type NetworkCapabilityCatalogPayload struct {
	Capabilities []NetworkCapabilityPayload `json:"capabilities"`
}

// NetworkPeerCardPayload is the shared JSON representation of one peer card.
type NetworkPeerCardPayload struct {
	PeerID              string                          `json:"peer_id"`
	DisplayName         *string                         `json:"display_name,omitempty"`
	ProfilesSupported   []string                        `json:"profiles_supported"`
	Capabilities        []NetworkCapabilityBriefPayload `json:"capabilities"`
	ArtifactsSupported  []string                        `json:"artifacts_supported"`
	TrustModesSupported []string                        `json:"trust_modes_supported"`
	Ext                 map[string]json.RawMessage      `json:"ext,omitempty"`
}

// NetworkPeerPayload is the shared JSON representation of one visible peer.
type NetworkPeerPayload struct {
	SessionID   *string                `json:"session_id,omitempty"`
	PeerID      string                 `json:"peer_id"`
	DisplayName string                 `json:"display_name,omitempty"`
	Channel     string                 `json:"channel"`
	Local       bool                   `json:"local"`
	PeerCard    NetworkPeerCardPayload `json:"peer_card"`
	JoinedAt    *time.Time             `json:"joined_at,omitempty"`
	LastSeen    *time.Time             `json:"last_seen,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// NetworkChannelPayload is the shared JSON representation of one active channel.
type NetworkChannelPayload struct {
	Channel                    string     `json:"channel"`
	WorkspaceID                string     `json:"workspace_id,omitempty"`
	Purpose                    string     `json:"purpose,omitempty"`
	CreatedBy                  string     `json:"created_by,omitempty"`
	CreatedAt                  *time.Time `json:"created_at,omitempty"`
	PeerCount                  int        `json:"peer_count"`
	LocalPeerCount             int        `json:"local_peer_count,omitempty"`
	RemotePeerCount            int        `json:"remote_peer_count,omitempty"`
	SessionCount               int        `json:"session_count,omitempty"`
	MessageCount               int        `json:"message_count,omitempty"`
	PresenceCount              int        `json:"presence_count,omitempty"`
	HistoricalParticipantCount int        `json:"historical_participant_count,omitempty"`
	LastActivityAt             *time.Time `json:"last_activity_at,omitempty"`
	LastPresenceAt             *time.Time `json:"last_presence_at,omitempty"`
	LastMessagePreview         string     `json:"last_message_preview,omitempty"`
}

// NetworkEnvelopePayload is the shared JSON representation of one surfaced
// network envelope used by inbox and audit-facing views.
type NetworkEnvelopePayload struct {
	Protocol      string                     `json:"protocol"`
	ID            string                     `json:"id"`
	Kind          string                     `json:"kind"`
	Channel       string                     `json:"channel"`
	From          string                     `json:"from"`
	To            *string                    `json:"to,omitempty"`
	InteractionID *string                    `json:"interaction_id,omitempty"`
	ReplyTo       *string                    `json:"reply_to,omitempty"`
	TraceID       *string                    `json:"trace_id,omitempty"`
	CausationID   *string                    `json:"causation_id,omitempty"`
	TS            int64                      `json:"ts"`
	ExpiresAt     *int64                     `json:"expires_at,omitempty"`
	Body          json.RawMessage            `json:"body"`
	Proof         map[string]json.RawMessage `json:"proof,omitempty"`
	Ext           map[string]json.RawMessage `json:"ext,omitempty"`
}

// NetworkChannelDetailPayload is the shared channel detail payload used by the network UI.
type NetworkChannelDetailPayload struct {
	Channel                    string                           `json:"channel"`
	WorkspaceID                string                           `json:"workspace_id,omitempty"`
	Purpose                    string                           `json:"purpose,omitempty"`
	CreatedBy                  string                           `json:"created_by,omitempty"`
	CreatedAt                  *time.Time                       `json:"created_at,omitempty"`
	PeerCount                  int                              `json:"peer_count"`
	LocalPeerCount             int                              `json:"local_peer_count,omitempty"`
	RemotePeerCount            int                              `json:"remote_peer_count,omitempty"`
	SessionCount               int                              `json:"session_count,omitempty"`
	MessageCount               int                              `json:"message_count,omitempty"`
	PresenceCount              int                              `json:"presence_count,omitempty"`
	HistoricalParticipantCount int                              `json:"historical_participant_count,omitempty"`
	LastActivityAt             *time.Time                       `json:"last_activity_at,omitempty"`
	LastPresenceAt             *time.Time                       `json:"last_presence_at,omitempty"`
	LastMessagePreview         string                           `json:"last_message_preview,omitempty"`
	KindCounts                 []NetworkChannelKindCountPayload `json:"kind_counts,omitempty"`
	Sessions                   []SessionPayload                 `json:"sessions,omitempty"`
	Peers                      []NetworkPeerPayload             `json:"peers,omitempty"`
}

// NetworkChannelKindCountPayload reports one channel-level kind count.
type NetworkChannelKindCountPayload struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// NetworkChannelMessagePayload is the shared network room timeline payload.
type NetworkChannelMessagePayload struct {
	MessageID          string          `json:"message_id"`
	Channel            string          `json:"channel"`
	Kind               string          `json:"kind"`
	Direction          string          `json:"direction"`
	PeerFrom           string          `json:"peer_from"`
	PeerTo             string          `json:"peer_to,omitempty"`
	DisplayName        string          `json:"display_name,omitempty"`
	SessionID          string          `json:"session_id,omitempty"`
	Local              bool            `json:"local,omitempty"`
	InteractionID      string          `json:"interaction_id,omitempty"`
	ReplyTo            string          `json:"reply_to,omitempty"`
	TraceID            string          `json:"trace_id,omitempty"`
	CausationID        string          `json:"causation_id,omitempty"`
	Intent             string          `json:"intent,omitempty"`
	Text               string          `json:"text,omitempty"`
	PreviewText        string          `json:"preview_text,omitempty"`
	PresenceCount      int             `json:"presence_count,omitempty"`
	PresenceStartedAt  *time.Time      `json:"presence_started_at,omitempty"`
	PresenceLastSeenAt *time.Time      `json:"presence_last_seen_at,omitempty"`
	Body               json.RawMessage `json:"body"`
	Timestamp          time.Time       `json:"timestamp"`
}

// NetworkPeerMetricsPayload is the shared peer-level counter payload.
type NetworkPeerMetricsPayload struct {
	Sent      int64 `json:"sent,omitempty"`
	Received  int64 `json:"received,omitempty"`
	Rejected  int64 `json:"rejected,omitempty"`
	Delivered int64 `json:"delivered,omitempty"`
}

// NetworkPeerDetailPayload is the shared selected-peer detail payload.
type NetworkPeerDetailPayload struct {
	SessionID         *string                          `json:"session_id,omitempty"`
	PeerID            string                           `json:"peer_id"`
	DisplayName       string                           `json:"display_name,omitempty"`
	Channel           string                           `json:"channel,omitempty"`
	Local             bool                             `json:"local,omitempty"`
	PeerCard          NetworkPeerCardPayload           `json:"peer_card"`
	CapabilityCatalog *NetworkCapabilityCatalogPayload `json:"capability_catalog,omitempty"`
	JoinedAt          *time.Time                       `json:"joined_at,omitempty"`
	LastSeen          *time.Time                       `json:"last_seen,omitempty"`
	ExpiresAt         *time.Time                       `json:"expires_at,omitempty"`
	Metrics           NetworkPeerMetricsPayload        `json:"metrics"`
}

// InstallExtensionRequest is the shared extension install request payload.
type InstallExtensionRequest struct {
	Path     string `json:"path"`
	Checksum string `json:"checksum"`
}

// ExtensionPayload is the shared extension response payload surfaced by CLI APIs.
type ExtensionPayload struct {
	Name          string                          `json:"name"`
	Version       string                          `json:"version"`
	Type          string                          `json:"type"`
	Source        string                          `json:"source"`
	Enabled       bool                            `json:"enabled"`
	State         string                          `json:"state"`
	Capabilities  []string                        `json:"capabilities,omitempty"`
	Actions       []string                        `json:"actions,omitempty"`
	RequiresEnv   []string                        `json:"requires_env,omitempty"`
	MissingEnv    []string                        `json:"missing_env,omitempty"`
	PID           int                             `json:"pid,omitempty"`
	UptimeSeconds int64                           `json:"uptime_seconds,omitempty"`
	Health        string                          `json:"health,omitempty"`
	HealthMessage string                          `json:"health_message,omitempty"`
	LastError     string                          `json:"last_error,omitempty"`
	DaemonRunning bool                            `json:"daemon_running"`
	Bundles       []ExtensionBundleSummaryPayload `json:"bundles,omitempty"`
}

// ExtensionBundleSummaryPayload captures the installed bundle catalog surfaced
// alongside extension status.
type ExtensionBundleSummaryPayload struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Profiles    []string `json:"profiles,omitempty"`
}

// ErrorPayload is the shared error response payload.
type ErrorPayload struct {
	Error string `json:"error"`
}

// MemoryWriteRequest is the shared memory write request payload.
type MemoryWriteRequest struct {
	Content   string `json:"content"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// MemoryReadResponse is the shared memory read response payload.
type MemoryReadResponse struct {
	Content string `json:"content"`
}

// MemoryMutationResponse is the shared memory mutation response payload.
type MemoryMutationResponse struct {
	OK bool `json:"ok"`
}

// MemoryConsolidateRequest is the shared memory consolidation request payload.
type MemoryConsolidateRequest struct {
	Workspace string `json:"workspace,omitempty"`
}

// MemoryConsolidateResponse is the shared memory consolidation response payload.
type MemoryConsolidateResponse struct {
	Triggered bool   `json:"triggered"`
	Reason    string `json:"reason,omitempty"`
}

// MemoryReindexRequest is the shared memory-catalog reindex request payload.
type MemoryReindexRequest struct {
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// MemoryOperationPayload is one redacted memory operation history row.
type MemoryOperationPayload struct {
	ID        string    `json:"id"`
	Operation string    `json:"operation"`
	Scope     string    `json:"scope,omitempty"`
	Workspace string    `json:"workspace,omitempty"`
	Filename  string    `json:"filename,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// MemoryHistoryResponse wraps the bounded memory operation history payload.
type MemoryHistoryResponse struct {
	Operations []MemoryOperationPayload `json:"operations"`
}

// MemoryHealthPayload is the shared memory health response payload.
type MemoryHealthPayload struct {
	Status             string     `json:"status"`
	Reason             string     `json:"reason,omitempty"`
	Enabled            bool       `json:"enabled"`
	Configured         bool       `json:"configured"`
	GlobalDir          string     `json:"global_dir,omitempty"`
	GlobalFiles        int        `json:"global_files"`
	WorkspaceFiles     int        `json:"workspace_files"`
	WorkspaceCount     int        `json:"workspace_count"`
	DreamEnabled       bool       `json:"dream_enabled"`
	DreamAgent         string     `json:"dream_agent,omitempty"`
	DreamMinHours      float64    `json:"dream_min_hours,omitempty"`
	DreamMinSessions   int        `json:"dream_min_sessions,omitempty"`
	DreamCheckInterval string     `json:"dream_check_interval,omitempty"`
	IndexedFiles       int        `json:"indexed_files"`
	OrphanedFiles      int        `json:"orphaned_files"`
	LastReindex        *time.Time `json:"last_reindex"`
	OperationCount     int        `json:"operation_count"`
	LastOperationAt    *time.Time `json:"last_operation_at"`
	LastConsolidation  *time.Time `json:"last_consolidation"`
}

// CreateWorkspaceRequest is the shared workspace creation request payload.
type CreateWorkspaceRequest struct {
	RootDir      string   `json:"root_dir"`
	Name         string   `json:"name,omitempty"`
	AddDirs      []string `json:"add_dirs,omitempty"`
	DefaultAgent string   `json:"default_agent,omitempty"`
	SandboxRef   string   `json:"sandbox_ref,omitempty"`
}

// UpdateWorkspaceRequest is the shared workspace update request payload.
type UpdateWorkspaceRequest struct {
	Name         *string   `json:"name"`
	AddDirs      *[]string `json:"add_dirs"`
	DefaultAgent *string   `json:"default_agent"`
	SandboxRef   *string   `json:"sandbox_ref"`
}

// ResolveWorkspaceRequest is the shared workspace resolve request payload.
type ResolveWorkspaceRequest struct {
	Path string `json:"path"`
}

// WorkspacePayload is the shared workspace response payload.
type WorkspacePayload struct {
	ID           string    `json:"id"`
	RootDir      string    `json:"root_dir"`
	AddDirs      []string  `json:"add_dirs"`
	Name         string    `json:"name"`
	DefaultAgent string    `json:"default_agent,omitempty"`
	SandboxRef   string    `json:"sandbox_ref,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkspaceSkillPayload is the shared workspace skill response payload.
type WorkspaceSkillPayload struct {
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Source string `json:"source"`
}

// SessionProviderOptionPayload is one workspace-visible session provider option.
type SessionProviderOptionPayload struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name,omitempty"`
	Harness         string `json:"harness,omitempty"`
	RuntimeProvider string `json:"runtime_provider,omitempty"`
	DefaultModel    string `json:"default_model,omitempty"`
	AuthMode        string `json:"auth_mode,omitempty"`
	EnvPolicy       string `json:"env_policy,omitempty"`
	HomePolicy      string `json:"home_policy,omitempty"`
}

// SkillPayload is the HTTP response type for a skill.
type SkillPayload struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Version     string             `json:"version,omitempty"`
	Source      string             `json:"source"`
	Enabled     bool               `json:"enabled"`
	Dir         string             `json:"dir"`
	Metadata    map[string]any     `json:"metadata,omitempty"`
	Provenance  *ProvenancePayload `json:"provenance,omitempty"`
}

// SkillContentResponse is the explicit response type for one skill body.
type SkillContentResponse struct {
	Content string `json:"content"`
}

// ProvenancePayload is the nested provenance metadata for marketplace skills.
type ProvenancePayload struct {
	Slug        string    `json:"slug"`
	Registry    string    `json:"registry"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

// SkillActionResponse is the shared skill enable/disable response payload.
type SkillActionResponse struct {
	OK bool `json:"ok"`
}

// WorkspaceDetailPayload is the shared resolved workspace detail response payload.
type WorkspaceDetailPayload struct {
	Workspace WorkspacePayload               `json:"workspace"`
	Sessions  []SessionPayload               `json:"sessions,omitempty"`
	Agents    []AgentPayload                 `json:"agents,omitempty"`
	Skills    []WorkspaceSkillPayload        `json:"skills,omitempty"`
	Providers []SessionProviderOptionPayload `json:"providers,omitempty"`
}
