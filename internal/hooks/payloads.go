package hooks

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrAutomationFireCancelled reports that a sync automation pre-fire hook canceled the dispatch.
	ErrAutomationFireCancelled = errors.New("hooks: automation fire canceled")
)

// PayloadBase carries the common identifiers attached to every hook payload.
type PayloadBase struct {
	Event     HookEvent `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

// SessionContext carries the common session-scoped hook attributes.
type SessionContext struct {
	SessionID    string `json:"session_id,omitempty"`
	SessionName  string `json:"session_name,omitempty"`
	SessionType  string `json:"session_type,omitempty"`
	AgentName    string `json:"agent_name,omitempty"`
	WorkspaceID  string `json:"workspace_id,omitempty"`
	Workspace    string `json:"workspace,omitempty"`
	ACPSessionID string `json:"acp_session_id,omitempty"`
	State        string `json:"state,omitempty"`
	*SessionSoulContext
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionSoulContext carries optional authored Soul provenance on session-scoped hooks.
type SessionSoulContext struct {
	SoulSnapshotID string `json:"soul_snapshot_id,omitempty"`
	SoulDigest     string `json:"soul_digest,omitempty"`
}

// TurnContext carries the current turn identifier.
type TurnContext struct {
	TurnID string `json:"turn_id,omitempty"`
}

// ContextBlock is a typed free-form context fragment attached to inputs or prompts.
type ContextBlock struct {
	Kind     string            `json:"kind,omitempty"`
	Text     string            `json:"text,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ToolCallRef identifies a tool invocation in hook payloads.
type ToolCallRef struct {
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolID     string `json:"tool_id,omitempty"`
	ReadOnly   bool   `json:"read_only,omitempty"`
}

// ToolLocation captures one path-scoped tool location.
type ToolLocation struct {
	Path      string `json:"path,omitempty"`
	StartLine int    `json:"start_line,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
}

// PermissionOption carries one interactive permission option.
type PermissionOption struct {
	Decision string `json:"decision,omitempty"`
	OptionID string `json:"option_id,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Label    string `json:"label,omitempty"`
}

// PermissionToolCall carries the tool details attached to a permission request.
type PermissionToolCall struct {
	ID        string         `json:"id,omitempty"`
	Kind      string         `json:"kind,omitempty"`
	Title     string         `json:"title,omitempty"`
	Status    string         `json:"status,omitempty"`
	Locations []ToolLocation `json:"locations,omitempty"`
}

// ControlPatch carries the common deny surface shared by mutable hook families.
type ControlPatch struct {
	Deny       bool   `json:"deny,omitempty"`
	DenyReason string `json:"deny_reason,omitempty"`
}

// SessionPreCreatePayload is delivered before a session is created.
type SessionPreCreatePayload struct {
	PayloadBase
	SessionContext
}

// SessionLifecyclePayload is shared by post-create, resume, and stop events.
type SessionLifecyclePayload struct {
	PayloadBase
	SessionContext
}

// SessionPostCreatePayload is delivered after a session is created.
type SessionPostCreatePayload = SessionLifecyclePayload

// SessionPreResumePayload is delivered before a session resumes.
type SessionPreResumePayload = SessionLifecyclePayload

// SessionPostResumePayload is delivered after a session resumes.
type SessionPostResumePayload = SessionLifecyclePayload

// SessionPreStopPayload is delivered before a session stops.
type SessionPreStopPayload = SessionLifecyclePayload

// SessionPostStopPayload is delivered after a session stops.
type SessionPostStopPayload = SessionLifecyclePayload

// SessionMessagePersistedPayload is delivered after an assistant message is durably persisted.
type SessionMessagePersistedPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	MessageID       string          `json:"message_id,omitempty"`
	MessageSeq      int64           `json:"message_seq,omitempty"`
	Role            string          `json:"role,omitempty"`
	Text            string          `json:"text,omitempty"`
	Raw             json.RawMessage `json:"raw,omitempty"`
	Persisted       json.RawMessage `json:"persisted,omitempty"`
	RootSessionID   string          `json:"root_session_id,omitempty"`
	ParentSessionID string          `json:"parent_session_id,omitempty"`
	ActorKind       string          `json:"actor_kind,omitempty"`
	ActorID         string          `json:"actor_id,omitempty"`
}

// SessionCreatePatch mutates or denies session lifecycle operations.
type SessionCreatePatch struct {
	ControlPatch
	SessionName *string `json:"session_name,omitempty"`
	SessionType *string `json:"session_type,omitempty"`
	AgentName   *string `json:"agent_name,omitempty"`
	WorkspaceID *string `json:"workspace_id,omitempty"`
	Workspace   *string `json:"workspace,omitempty"`
}

// SessionPostCreatePatch is the post-create patch surface.
type SessionPostCreatePatch = SessionCreatePatch

// SessionPreResumePatch is the pre-resume patch surface.
type SessionPreResumePatch = SessionCreatePatch

// SessionPostResumePatch is the post-resume patch surface.
type SessionPostResumePatch = SessionCreatePatch

// SessionPreStopPatch is the pre-stop patch surface.
type SessionPreStopPatch = SessionCreatePatch

// SessionPostStopPatch is the post-stop patch surface.
type SessionPostStopPatch = SessionCreatePatch

// SandboxProfilePayload is the sandbox profile snapshot exposed to sandbox hooks.
type SandboxProfilePayload struct {
	Profile        string            `json:"profile,omitempty"`
	Backend        string            `json:"backend,omitempty"`
	SyncMode       string            `json:"sync_mode,omitempty"`
	Persistence    string            `json:"persistence,omitempty"`
	RuntimeRootDir string            `json:"runtime_root,omitempty"`
	DestroyOnStop  bool              `json:"destroy_on_stop,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	SecretEnv      map[string]string `json:"secret_env,omitempty"`
}

// SandboxPreparePayload is delivered before a session sandbox is prepared.
type SandboxPreparePayload struct {
	PayloadBase
	SessionContext
	SandboxID           string                `json:"sandbox_id,omitempty"`
	Backend             string                `json:"backend,omitempty"`
	Profile             SandboxProfilePayload `json:"profile"`
	LocalRootDir        string                `json:"local_root,omitempty"`
	LocalAdditionalDirs []string              `json:"local_additional_dirs,omitempty"`
	AgentCommand        string                `json:"agent_command,omitempty"`
	AgentEnv            []string              `json:"agent_env,omitempty"`
	Permissions         string                `json:"permissions,omitempty"`
	ResumeACPState      string                `json:"resume_acp_state,omitempty"`
	EnvOverrides        map[string]string     `json:"env_overrides,omitempty"`
	Denied              bool                  `json:"denied,omitempty"`
	DenyReason          string                `json:"deny_reason,omitempty"`
}

// SandboxReadyPayload is delivered after a sandbox has been prepared and synchronized.
type SandboxReadyPayload struct {
	PayloadBase
	SessionContext
	SandboxID             string   `json:"sandbox_id,omitempty"`
	Backend               string   `json:"backend,omitempty"`
	Profile               string   `json:"profile,omitempty"`
	InstanceID            string   `json:"instance_id,omitempty"`
	RuntimeRootDir        string   `json:"runtime_root,omitempty"`
	RuntimeAdditionalDirs []string `json:"runtime_additional_dirs,omitempty"`
}

// SandboxSyncBeforePayload is delivered before a sandbox sync operation runs.
type SandboxSyncBeforePayload struct {
	PayloadBase
	SessionContext
	SandboxID       string   `json:"sandbox_id,omitempty"`
	Backend         string   `json:"backend,omitempty"`
	Profile         string   `json:"profile,omitempty"`
	InstanceID      string   `json:"instance_id,omitempty"`
	RuntimeRootDir  string   `json:"runtime_root,omitempty"`
	Direction       string   `json:"direction,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	FileCount       int      `json:"file_count,omitempty"`
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
	Denied          bool     `json:"denied,omitempty"`
	DenyReason      string   `json:"deny_reason,omitempty"`
}

// SandboxSyncAfterPayload is delivered after a sandbox sync operation finishes.
type SandboxSyncAfterPayload struct {
	PayloadBase
	SessionContext
	SandboxID        string   `json:"sandbox_id,omitempty"`
	Backend          string   `json:"backend,omitempty"`
	Profile          string   `json:"profile,omitempty"`
	InstanceID       string   `json:"instance_id,omitempty"`
	RuntimeRootDir   string   `json:"runtime_root,omitempty"`
	Direction        string   `json:"direction,omitempty"`
	Reason           string   `json:"reason,omitempty"`
	FilesSynced      int      `json:"files_synced,omitempty"`
	BytesTransferred int64    `json:"bytes_transferred,omitempty"`
	DurationMS       int64    `json:"duration_ms,omitempty"`
	Errors           []string `json:"errors,omitempty"`
}

// SandboxStopPayload is delivered before sandbox teardown.
type SandboxStopPayload struct {
	PayloadBase
	SessionContext
	SandboxID      string `json:"sandbox_id,omitempty"`
	Backend        string `json:"backend,omitempty"`
	Profile        string `json:"profile,omitempty"`
	InstanceID     string `json:"instance_id,omitempty"`
	RuntimeRootDir string `json:"runtime_root,omitempty"`
	StopReason     string `json:"stop_reason,omitempty"`
	WillDestroy    bool   `json:"will_destroy,omitempty"`
	Denied         bool   `json:"denied,omitempty"`
	DenyReason     string `json:"deny_reason,omitempty"`
}

// SandboxPreparePatch mutates or denies sandbox preparation.
type SandboxPreparePatch struct {
	ControlPatch
	EnvOverrides map[string]string `json:"env_overrides,omitempty"`
}

// SandboxSyncBeforePatch mutates or denies sandbox sync.
type SandboxSyncBeforePatch struct {
	ControlPatch
	ExcludePatterns []string `json:"exclude_patterns,omitempty"`
}

// SandboxObservationPatch is the no-op patch surface for sandbox observation hooks.
type SandboxObservationPatch struct{}

// SandboxReadyPatch is the ready patch surface.
type SandboxReadyPatch = SandboxObservationPatch

// SandboxSyncAfterPatch is the sync-after patch surface.
type SandboxSyncAfterPatch = SandboxObservationPatch

// SandboxStopPatch mutates or denies sandbox teardown.
type SandboxStopPatch struct {
	ControlPatch
}

// InputPreSubmitPayload is delivered before prompt submission.
type InputPreSubmitPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	InputClass    string         `json:"input_class,omitempty"`
	Message       string         `json:"message,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// InputPreSubmitPatch mutates or denies the submitted input.
type InputPreSubmitPatch struct {
	ControlPatch
	Message       *string        `json:"message,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// PromptPayload is delivered after prompt assembly.
type PromptPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	InputClass    string         `json:"input_class,omitempty"`
	Prompt        string         `json:"prompt,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// PromptPatch mutates or denies the assembled prompt.
type PromptPatch struct {
	ControlPatch
	Prompt        *string        `json:"prompt,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// EventRecordPayload is shared by event pre/post-record hooks.
type EventRecordPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	RecordType string          `json:"record_type,omitempty"`
	Sequence   int64           `json:"sequence,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
}

// EventPreRecordPayload is delivered before an event record is written.
type EventPreRecordPayload = EventRecordPayload

// EventPostRecordPayload is delivered after an event record is written.
type EventPostRecordPayload = EventRecordPayload

// EventRecordPatch captures the optional observation patch surface for event hooks.
type EventRecordPatch struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// EventPreRecordPatch is the pre-record patch surface.
type EventPreRecordPatch = EventRecordPatch

// EventPostRecordPatch is the post-record patch surface.
type EventPostRecordPatch = EventRecordPatch

// AutomationSchedulePayload carries the serializable schedule shape exposed to automation hooks.
type AutomationSchedulePayload struct {
	Mode     string `json:"mode,omitempty"`
	Expr     string `json:"expr,omitempty"`
	Interval string `json:"interval,omitempty"`
	Time     string `json:"time,omitempty"`
}

// AutomationJobPreFirePayload is delivered before a job fire dispatches.
type AutomationJobPreFirePayload struct {
	JobID       string                     `json:"job_id"`
	JobName     string                     `json:"job_name,omitempty"`
	AgentName   string                     `json:"agent_name,omitempty"`
	WorkspaceID string                     `json:"workspace_id,omitempty"`
	Prompt      string                     `json:"prompt,omitempty"`
	Schedule    *AutomationSchedulePayload `json:"schedule,omitempty"`
	Payload     map[string]any             `json:"payload,omitempty"`
	Attempt     int                        `json:"attempt,omitempty"`
}

// AutomationJobPostFirePayload is delivered after a job fire hands off to a session.
type AutomationJobPostFirePayload struct {
	JobID       string `json:"job_id"`
	JobName     string `json:"job_name,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

// AutomationTriggerPreFirePayload is delivered before a trigger fire dispatches.
type AutomationTriggerPreFirePayload struct {
	TriggerID   string         `json:"trigger_id"`
	TriggerName string         `json:"trigger_name,omitempty"`
	Event       string         `json:"event,omitempty"`
	AgentName   string         `json:"agent_name,omitempty"`
	WorkspaceID string         `json:"workspace_id,omitempty"`
	Prompt      string         `json:"prompt,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
	Attempt     int            `json:"attempt,omitempty"`
}

// AutomationTriggerPostFirePayload is delivered after a trigger fire hands off to a session.
type AutomationTriggerPostFirePayload struct {
	TriggerID   string `json:"trigger_id"`
	TriggerName string `json:"trigger_name,omitempty"`
	Event       string `json:"event,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

// AutomationRunCompletedPayload is delivered after an automation run finishes successfully.
type AutomationRunCompletedPayload struct {
	RunID       string `json:"run_id"`
	JobID       string `json:"job_id,omitempty"`
	TriggerID   string `json:"trigger_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Attempt     int    `json:"attempt,omitempty"`
	DurationMS  int64  `json:"duration_ms,omitempty"`
}

// AutomationRunFailedPayload is delivered after an automation run fails.
type AutomationRunFailedPayload struct {
	RunID       string `json:"run_id"`
	JobID       string `json:"job_id,omitempty"`
	TriggerID   string `json:"trigger_id,omitempty"`
	AgentName   string `json:"agent_name,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Error       string `json:"error,omitempty"`
	Attempt     int    `json:"attempt,omitempty"`
	WillRetry   bool   `json:"will_retry,omitempty"`
}

// AutomationFirePatch mutates or cancels one automation pre-fire dispatch.
type AutomationFirePatch struct {
	Prompt *string `json:"prompt,omitempty"`
	Cancel bool    `json:"cancel,omitempty"`
}

// AutomationObservationPatch is the no-op patch surface for async automation observation hooks.
type AutomationObservationPatch struct{}

// AgentPreStartPayload is delivered before an agent process starts.
type AgentPreStartPayload struct {
	PayloadBase
	SessionContext
	Command  string   `json:"command,omitempty"`
	Args     []string `json:"args,omitempty"`
	Cwd      string   `json:"cwd,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
}

// AgentLifecyclePayload is shared by spawned, crashed, and stopped hooks.
type AgentLifecyclePayload struct {
	PayloadBase
	SessionContext
	Command  string   `json:"command,omitempty"`
	Args     []string `json:"args,omitempty"`
	Cwd      string   `json:"cwd,omitempty"`
	PID      int      `json:"pid,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Error    string   `json:"error,omitempty"`
}

// AgentSpawnedPayload is delivered after an agent process starts.
type AgentSpawnedPayload = AgentLifecyclePayload

// AgentCrashedPayload is delivered when an agent crashes.
type AgentCrashedPayload = AgentLifecyclePayload

// AgentStoppedPayload is delivered after an agent stops.
type AgentStoppedPayload = AgentLifecyclePayload

// AgentStartPatch mutates or denies a pre-start operation.
type AgentStartPatch struct {
	ControlPatch
	Command *string  `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Cwd     *string  `json:"cwd,omitempty"`
}

// AgentLifecyclePatch captures optional labels for observation events.
type AgentLifecyclePatch struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// AgentSpawnedPatch is the spawned patch surface.
type AgentSpawnedPatch = AgentLifecyclePatch

// AgentCrashedPatch is the crashed patch surface.
type AgentCrashedPatch = AgentLifecyclePatch

// AgentStoppedPatch is the stopped patch surface.
type AgentStoppedPatch = AgentLifecyclePatch

// AuthoredContextProvenance carries redacted Soul/Heartbeat source identity.
type AuthoredContextProvenance struct {
	WorkspaceID      string `json:"workspace_id,omitempty"`
	AgentName        string `json:"agent_name,omitempty"`
	SourcePath       string `json:"source_path,omitempty"`
	SnapshotID       string `json:"snapshot_id,omitempty"`
	Digest           string `json:"digest,omitempty"`
	ConfigDigest     string `json:"config_digest,omitempty"`
	ValidationStatus string `json:"validation_status,omitempty"`
	Valid            bool   `json:"valid"`
	Active           bool   `json:"active"`
	Reason           string `json:"reason,omitempty"`
}

// AuthoredMutationProvenance records who caused a managed authored-context mutation.
type AuthoredMutationProvenance struct {
	ActorKind  string `json:"actor_kind,omitempty"`
	ActorID    string `json:"actor_id,omitempty"`
	OriginKind string `json:"origin_kind,omitempty"`
	OriginRef  string `json:"origin_ref,omitempty"`
}

// AgentSoulSnapshotResolvedPayload is delivered after Soul snapshot/read-model resolution.
type AgentSoulSnapshotResolvedPayload struct {
	PayloadBase
	AuthoredContextProvenance
}

// AgentSoulMutationAfterPayload is delivered after a managed SOUL.md mutation commits.
type AgentSoulMutationAfterPayload struct {
	PayloadBase
	AuthoredContextProvenance
	AuthoredMutationProvenance
	RevisionID     string `json:"revision_id,omitempty"`
	Action         string `json:"action,omitempty"`
	PreviousDigest string `json:"previous_digest,omitempty"`
	NewDigest      string `json:"new_digest,omitempty"`
}

// AgentHeartbeatPolicyResolvedPayload is delivered after Heartbeat policy/status resolution.
type AgentHeartbeatPolicyResolvedPayload struct {
	PayloadBase
	AuthoredContextProvenance
	Summary string `json:"summary,omitempty"`
}

// AgentHeartbeatWakeBeforePayload is delivered before a managed Heartbeat wake decision.
type AgentHeartbeatWakeBeforePayload struct {
	PayloadBase
	SessionContext
	PolicySnapshotID string `json:"policy_snapshot_id,omitempty"`
	PolicyDigest     string `json:"policy_digest,omitempty"`
	ConfigDigest     string `json:"config_digest,omitempty"`
	Source           string `json:"source,omitempty"`
	DryRun           bool   `json:"dry_run,omitempty"`
}

// AgentHeartbeatWakeAfterPayload is delivered after a managed Heartbeat wake decision.
type AgentHeartbeatWakeAfterPayload struct {
	PayloadBase
	SessionContext
	WakeEventID       string `json:"wake_event_id,omitempty"`
	Result            string `json:"result,omitempty"`
	Reason            string `json:"reason,omitempty"`
	PolicySnapshotID  string `json:"policy_snapshot_id,omitempty"`
	PolicyDigest      string `json:"policy_digest,omitempty"`
	ConfigDigest      string `json:"config_digest,omitempty"`
	SyntheticPromptID string `json:"synthetic_prompt_id,omitempty"`
	Source            string `json:"source,omitempty"`
}

// SessionHealthUpdateAfterPayload is delivered after metadata-only session health changes.
type SessionHealthUpdateAfterPayload struct {
	PayloadBase
	SessionContext
	Health              string    `json:"health,omitempty"`
	ActivePrompt        bool      `json:"active_prompt,omitempty"`
	Attachable          bool      `json:"attachable,omitempty"`
	EligibleForWake     bool      `json:"eligible_for_wake,omitempty"`
	IneligibilityReason string    `json:"ineligibility_reason,omitempty"`
	LastActivityAt      time.Time `json:"last_activity_at"`
	LastPresenceAt      time.Time `json:"last_presence_at"`
	LastError           string    `json:"last_error,omitempty"`
}

// NetworkPayload is the shared observation payload for committed network conversation writes.
type NetworkPayload struct {
	PayloadBase
	WorkspaceID string     `json:"workspace_id,omitempty"`
	SessionID   string     `json:"session_id,omitempty"`
	Channel     string     `json:"channel,omitempty"`
	Surface     string     `json:"surface,omitempty"`
	ThreadID    string     `json:"thread_id,omitempty"`
	DirectID    string     `json:"direct_id,omitempty"`
	MessageID   string     `json:"message_id,omitempty"`
	Kind        string     `json:"kind,omitempty"`
	Direction   string     `json:"direction,omitempty"`
	WorkID      string     `json:"work_id,omitempty"`
	WorkState   string     `json:"work_state,omitempty"`
	PeerID      string     `json:"peer_id,omitempty"`
	PeerFrom    string     `json:"peer_from,omitempty"`
	PeerTo      string     `json:"peer_to,omitempty"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
	TraceID     string     `json:"trace_id,omitempty"`
	CausationID string     `json:"causation_id,omitempty"`
}

// NetworkThreadOpenedPayload observes a newly opened public thread.
type NetworkThreadOpenedPayload = NetworkPayload

// NetworkDirectRoomOpenedPayload observes a newly opened direct room.
type NetworkDirectRoomOpenedPayload = NetworkPayload

// NetworkMessagePersistedPayload observes a committed conversation message.
type NetworkMessagePersistedPayload = NetworkPayload

// NetworkWorkOpenedPayload observes a newly opened work item.
type NetworkWorkOpenedPayload = NetworkPayload

// NetworkWorkTransitionedPayload observes a work lifecycle transition.
type NetworkWorkTransitionedPayload = NetworkPayload

// NetworkWorkClosedPayload observes a terminal work lifecycle transition.
type NetworkWorkClosedPayload = NetworkPayload

// NetworkPeerJoinedPayload observes a peer becoming visible on a runtime channel.
type NetworkPeerJoinedPayload = NetworkPayload

// NetworkPeerLeftPayload observes a peer leaving or expiring from a runtime channel.
type NetworkPeerLeftPayload = NetworkPayload

// NetworkObservationPatch captures optional labels for network observation hooks.
type NetworkObservationPatch struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// AuthoredContextObservationPatch is the no-op patch surface for authored-context observation hooks.
type AuthoredContextObservationPatch = AutonomyObservationPatch

// TurnPayload is shared by turn start and end events.
type TurnPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	InputClass  string `json:"input_class,omitempty"`
	UserMessage string `json:"user_message,omitempty"`
}

// TurnStartPayload is delivered at turn start.
type TurnStartPayload = TurnPayload

// TurnEndPayload is delivered at turn end.
type TurnEndPayload = TurnPayload

// TurnPatch mutates or denies turn-scoped operations.
type TurnPatch struct {
	ControlPatch
	Labels map[string]string `json:"labels,omitempty"`
}

// TurnStartPatch is the turn-start patch surface.
type TurnStartPatch = TurnPatch

// TurnEndPatch is the turn-end patch surface.
type TurnEndPatch = TurnPatch

// MessagePayload is shared by message start, delta, and end events.
type MessagePayload struct {
	PayloadBase
	SessionContext
	TurnContext
	MessageID string          `json:"message_id,omitempty"`
	Role      string          `json:"role,omitempty"`
	DeltaType string          `json:"delta_type,omitempty"`
	Text      string          `json:"text,omitempty"`
	Raw       json.RawMessage `json:"raw,omitempty"`
}

// MessageStartPayload is delivered when a message begins.
type MessageStartPayload = MessagePayload

// MessageDeltaPayload is delivered for streaming message deltas.
type MessageDeltaPayload = MessagePayload

// MessageEndPayload is delivered when a message finishes.
type MessageEndPayload = MessagePayload

// MessagePatch mutates or denies message-scoped operations.
type MessagePatch struct {
	ControlPatch
	Role      *string `json:"role,omitempty"`
	DeltaType *string `json:"delta_type,omitempty"`
	Text      *string `json:"text,omitempty"`
}

// MessageStartPatch is the message-start patch surface.
type MessageStartPatch = MessagePatch

// MessageDeltaPatch is the message-delta patch surface.
type MessageDeltaPatch = MessagePatch

// MessageEndPatch is the message-end patch surface.
type MessageEndPatch = MessagePatch

// ToolPreCallPayload is delivered before a tool runs.
type ToolPreCallPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	ToolCallRef
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
}

// ToolPostCallPayload is delivered after a tool completes successfully.
type ToolPostCallPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	ToolCallRef
	Title      string          `json:"title,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolResult json.RawMessage `json:"tool_result,omitempty"`
}

// ToolPostErrorPayload is delivered after a tool fails.
type ToolPostErrorPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	ToolCallRef
	Title     string          `json:"title,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// ToolCallPatch mutates or denies tool invocation inputs.
type ToolCallPatch struct {
	ControlPatch
	ToolID    *string         `json:"tool_id,omitempty"`
	ReadOnly  *bool           `json:"read_only,omitempty"`
	ToolInput json.RawMessage `json:"tool_input,omitempty"`
}

// ToolResultPatch mutates or denies tool outputs.
type ToolResultPatch struct {
	ControlPatch
	Title      *string         `json:"title,omitempty"`
	ToolResult json.RawMessage `json:"tool_result,omitempty"`
	Error      *string         `json:"error,omitempty"`
}

// ToolPostErrorPatch is the post-error patch surface.
type ToolPostErrorPatch = ToolResultPatch

// PermissionRequestPayload is delivered before a permission decision resolves.
type PermissionRequestPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	RequestID     string             `json:"request_id,omitempty"`
	Action        string             `json:"action,omitempty"`
	Resource      string             `json:"resource,omitempty"`
	Decision      string             `json:"decision,omitempty"`
	DecisionClass string             `json:"decision_class,omitempty"`
	ToolInput     json.RawMessage    `json:"tool_input,omitempty"`
	ToolCall      PermissionToolCall `json:"tool_call"`
	Options       []PermissionOption `json:"options,omitempty"`
}

// PermissionResolutionPayload is shared by resolved and denied events.
type PermissionResolutionPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	RequestID     string             `json:"request_id,omitempty"`
	Action        string             `json:"action,omitempty"`
	Resource      string             `json:"resource,omitempty"`
	Decision      string             `json:"decision,omitempty"`
	DecisionClass string             `json:"decision_class,omitempty"`
	ToolInput     json.RawMessage    `json:"tool_input,omitempty"`
	ToolCall      PermissionToolCall `json:"tool_call"`
}

// PermissionResolvedPayload is delivered after a permission decision resolves.
type PermissionResolvedPayload = PermissionResolutionPayload

// PermissionDeniedPayload is delivered after a permission denial resolves.
type PermissionDeniedPayload = PermissionResolutionPayload

// PermissionRequestPatch mutates or denies the permission-request surface.
type PermissionRequestPatch struct {
	ControlPatch
	Decision      *string `json:"decision,omitempty"`
	DecisionClass *string `json:"decision_class,omitempty"`
	Reason        *string `json:"reason,omitempty"`
}

// PermissionResolvedPatch is the resolved patch surface.
type PermissionResolvedPatch struct{}

// PermissionDeniedPatch is the denied patch surface.
type PermissionDeniedPatch struct{}

// ContextCompactPayload is shared by context compaction hooks.
type ContextCompactPayload struct {
	PayloadBase
	SessionContext
	TurnContext
	Reason        string         `json:"reason,omitempty"`
	Strategy      string         `json:"strategy,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// ContextPreCompactPayload is delivered before compaction.
type ContextPreCompactPayload = ContextCompactPayload

// ContextPostCompactPayload is delivered after compaction.
type ContextPostCompactPayload = ContextCompactPayload

// ContextCompactionPatch mutates or denies compaction behavior.
type ContextCompactionPatch struct {
	ControlPatch
	Reason        *string        `json:"reason,omitempty"`
	Strategy      *string        `json:"strategy,omitempty"`
	ContextBlocks []ContextBlock `json:"context_blocks,omitempty"`
}

// ContextPreCompactPatch is the pre-compact patch surface.
type ContextPreCompactPatch = ContextCompactionPatch

// ContextPostCompactPatch is the post-compact patch surface.
type ContextPostCompactPatch = ContextCompactionPatch

// AutonomyObservationPatch captures optional labels for committed autonomy lifecycle events.
type AutonomyObservationPatch struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// CoordinatorContext carries the coordinator identifiers shared across coordinator hooks.
type CoordinatorContext struct {
	WorkspaceID           string `json:"workspace_id,omitempty"`
	Workspace             string `json:"workspace,omitempty"`
	AgentName             string `json:"agent_name,omitempty"`
	CoordinatorSessionID  string `json:"coordinator_session_id,omitempty"`
	TaskID                string `json:"task_id,omitempty"`
	RunID                 string `json:"run_id,omitempty"`
	WorkflowID            string `json:"workflow_id,omitempty"`
	CoordinationChannelID string `json:"coordination_channel_id,omitempty"`
	Provider              string `json:"provider,omitempty"`
	Model                 string `json:"model,omitempty"`
}

// CoordinatorPreSpawnPayload is delivered before the daemon creates a coordinator session.
type CoordinatorPreSpawnPayload struct {
	PayloadBase
	CoordinatorContext
	Reason     string `json:"reason,omitempty"`
	Denied     bool   `json:"denied,omitempty"`
	DenyReason string `json:"deny_reason,omitempty"`
}

// CoordinatorLifecyclePayload is shared by committed coordinator lifecycle hooks.
type CoordinatorLifecyclePayload struct {
	PayloadBase
	CoordinatorContext
	DecisionKind string `json:"decision_kind,omitempty"`
	Decision     string `json:"decision,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
	Error        string `json:"error,omitempty"`
}

// CoordinatorSpawnedPayload is delivered after a coordinator session is created.
type CoordinatorSpawnedPayload = CoordinatorLifecyclePayload

// CoordinatorDecisionPayload is delivered when a coordinator records a semantic decision.
type CoordinatorDecisionPayload = CoordinatorLifecyclePayload

// CoordinatorStoppedPayload is delivered after a coordinator session stops.
type CoordinatorStoppedPayload = CoordinatorLifecyclePayload

// CoordinatorFailedPayload is delivered after a coordinator lifecycle failure.
type CoordinatorFailedPayload = CoordinatorLifecyclePayload

// CoordinatorSpawnPatch mutates or denies coordinator spawn requests.
type CoordinatorSpawnPatch struct {
	ControlPatch
	AgentName *string `json:"agent_name,omitempty"`
	Provider  *string `json:"provider,omitempty"`
	Model     *string `json:"model,omitempty"`
}

// CoordinatorObservationPatch is the observation patch surface for committed coordinator hooks.
type CoordinatorObservationPatch = AutonomyObservationPatch

// TaskRunClaimCriteria carries the mutable claim criteria exposed to task-run pre-claim hooks.
type TaskRunClaimCriteria struct {
	WorkspaceID           string   `json:"workspace_id,omitempty"`
	ClaimerSessionID      string   `json:"claimer_session_id,omitempty"`
	AgentName             string   `json:"agent_name,omitempty"`
	RequiredCapabilities  []string `json:"required_capabilities,omitempty"`
	PriorityMin           int      `json:"priority_min,omitempty"`
	CoordinationChannelID string   `json:"coordination_channel_id,omitempty"`
}

// TaskRunContext carries task-run identifiers shared across task-run hooks.
type TaskRunContext struct {
	TaskID                string    `json:"task_id,omitempty"`
	RunID                 string    `json:"run_id,omitempty"`
	WorkspaceID           string    `json:"workspace_id,omitempty"`
	WorkflowID            string    `json:"workflow_id,omitempty"`
	CoordinationChannelID string    `json:"coordination_channel_id,omitempty"`
	NetworkChannel        string    `json:"network_channel,omitempty"`
	AgentName             string    `json:"agent_name,omitempty"`
	SessionID             string    `json:"session_id,omitempty"`
	ActorKind             string    `json:"actor_kind,omitempty"`
	ActorID               string    `json:"actor_id,omitempty"`
	OriginKind            string    `json:"origin_kind,omitempty"`
	OriginRef             string    `json:"origin_ref,omitempty"`
	TaskStatus            string    `json:"task_status,omitempty"`
	RunStatus             string    `json:"run_status,omitempty"`
	SoulSnapshotID        string    `json:"soul_snapshot_id,omitempty"`
	SoulDigest            string    `json:"soul_digest,omitempty"`
	Attempt               int       `json:"attempt,omitempty"`
	LeaseUntil            time.Time `json:"lease_until"`
	ReleaseReason         string    `json:"release_reason,omitempty"`
	Error                 string    `json:"error,omitempty"`
}

// TaskRunEnqueuedPayload is delivered after a task run is enqueued and its audit event is committed.
type TaskRunEnqueuedPayload struct {
	PayloadBase
	TaskRunContext
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// TaskRunPreClaimPayload is delivered before a task run claim commits.
type TaskRunPreClaimPayload struct {
	PayloadBase
	TaskRunContext
	Criteria   TaskRunClaimCriteria `json:"criteria"`
	Denied     bool                 `json:"denied,omitempty"`
	DenyReason string               `json:"deny_reason,omitempty"`
}

// TaskRunPostClaimPayload is delivered after a task run claim and audit event commit.
type TaskRunPostClaimPayload struct {
	PayloadBase
	TaskRunContext
	ClaimedAt time.Time `json:"claimed_at"`
}

// TaskRunLeasePayload is shared by committed task-run lease lifecycle hooks.
type TaskRunLeasePayload struct {
	PayloadBase
	TaskRunContext
	PreviousRunStatus string `json:"previous_run_status,omitempty"`
	PreviousSessionID string `json:"previous_session_id,omitempty"`
	RecoveryAction    string `json:"recovery_action,omitempty"`
	RecoveryReason    string `json:"recovery_reason,omitempty"`
}

// TaskRunLeaseExtendedPayload is delivered after a task-run lease is extended.
type TaskRunLeaseExtendedPayload = TaskRunLeasePayload

// TaskRunLeaseExpiredPayload is delivered after a task-run lease expires.
type TaskRunLeaseExpiredPayload = TaskRunLeasePayload

// TaskRunLeaseRecoveredPayload is delivered after lease recovery commits.
type TaskRunLeaseRecoveredPayload = TaskRunLeasePayload

// TaskRunReleasedPayload is delivered after a task run lease is released.
type TaskRunReleasedPayload = TaskRunLeasePayload

// TaskRunCompletedPayload is delivered after a token-fenced task run completion.
type TaskRunCompletedPayload = TaskRunLeasePayload

// TaskRunFailedPayload is delivered after a token-fenced task run failure.
type TaskRunFailedPayload = TaskRunLeasePayload

// TaskRunPreClaimPatch denies or narrows task-run claim criteria.
type TaskRunPreClaimPatch struct {
	ControlPatch
	AddRequiredCapabilities []string `json:"add_required_capabilities,omitempty"`
	PriorityMin             *int     `json:"priority_min,omitempty"`
}

// TaskRunObservationPatch is the observation patch surface for committed task-run hooks.
type TaskRunObservationPatch = AutonomyObservationPatch

// PermissionSet captures concrete permission atoms that spawned children may only narrow.
type PermissionSet struct {
	Tools           []string `json:"tools,omitempty"`
	Skills          []string `json:"skills,omitempty"`
	MCPServers      []string `json:"mcp_servers,omitempty"`
	WorkspacePaths  []string `json:"workspace_paths,omitempty"`
	NetworkChannels []string `json:"network_channels,omitempty"`
	SandboxProfiles []string `json:"sandbox_profiles,omitempty"`
}

// SpawnContext carries spawn identifiers shared across spawn lifecycle hooks.
type SpawnContext struct {
	ParentSessionID       string `json:"parent_session_id,omitempty"`
	RootSessionID         string `json:"root_session_id,omitempty"`
	ChildSessionID        string `json:"child_session_id,omitempty"`
	WorkspaceID           string `json:"workspace_id,omitempty"`
	Workspace             string `json:"workspace,omitempty"`
	AgentName             string `json:"agent_name,omitempty"`
	SpawnRole             string `json:"spawn_role,omitempty"`
	SpawnDepth            int    `json:"spawn_depth,omitempty"`
	TTLSeconds            int64  `json:"ttl_seconds,omitempty"`
	AutoStopOnParent      bool   `json:"auto_stop_on_parent,omitempty"`
	TaskID                string `json:"task_id,omitempty"`
	RunID                 string `json:"run_id,omitempty"`
	WorkflowID            string `json:"workflow_id,omitempty"`
	CoordinationChannelID string `json:"coordination_channel_id,omitempty"`
	SoulSnapshotID        string `json:"soul_snapshot_id,omitempty"`
	SoulDigest            string `json:"soul_digest,omitempty"`
	ParentSoulDigest      string `json:"parent_soul_digest,omitempty"`
}

// SpawnPreCreatePayload is delivered before a child session is created.
type SpawnPreCreatePayload struct {
	PayloadBase
	SpawnContext
	ParentPermissions *PermissionSet `json:"parent_permissions"`
	ChildPermissions  *PermissionSet `json:"child_permissions"`
	Denied            bool           `json:"denied,omitempty"`
	DenyReason        string         `json:"deny_reason,omitempty"`
}

// SpawnLifecyclePayload is shared by committed spawn lifecycle hooks.
type SpawnLifecyclePayload struct {
	PayloadBase
	SpawnContext
	ParentPermissions *PermissionSet `json:"parent_permissions,omitempty"`
	ChildPermissions  *PermissionSet `json:"child_permissions,omitempty"`
	StopReason        string         `json:"stop_reason,omitempty"`
	ReapReason        string         `json:"reap_reason,omitempty"`
	Error             string         `json:"error,omitempty"`
}

// SpawnCreatedPayload is delivered after a child session is created.
type SpawnCreatedPayload = SpawnLifecyclePayload

// SpawnParentStoppedPayload is delivered when parent-stop reaps a child session.
type SpawnParentStoppedPayload = SpawnLifecyclePayload

// SpawnTTLExpiredPayload is delivered when TTL expiry reaps a child session.
type SpawnTTLExpiredPayload = SpawnLifecyclePayload

// SpawnReapedPayload is delivered after a child session is reaped.
type SpawnReapedPayload = SpawnLifecyclePayload

// SpawnCreatePatch mutates or denies child-session spawn requests.
type SpawnCreatePatch struct {
	ControlPatch
	AgentName        *string        `json:"agent_name,omitempty"`
	SpawnRole        *string        `json:"spawn_role,omitempty"`
	TTLSeconds       *int64         `json:"ttl_seconds,omitempty"`
	ChildPermissions *PermissionSet `json:"child_permissions,omitempty"`
}

// SpawnObservationPatch is the observation patch surface for committed spawn lifecycle hooks.
type SpawnObservationPatch = AutonomyObservationPatch

func (p SessionPreCreatePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SessionLifecyclePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SessionMessagePersistedPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SandboxPreparePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SandboxReadyPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SandboxSyncBeforePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SandboxSyncAfterPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SandboxStopPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p InputPreSubmitPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p PromptPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p EventRecordPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p AgentPreStartPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p AgentLifecyclePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p AgentHeartbeatWakeBeforePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p AgentHeartbeatWakeAfterPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p SessionHealthUpdateAfterPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p TurnPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p MessagePayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p ToolPreCallPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p ToolPostCallPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p ToolPostErrorPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p PermissionRequestPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p PermissionResolutionPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p ContextCompactPayload) hookSessionContext() SessionContext {
	return p.SessionContext
}

func (p CoordinatorPreSpawnPayload) hookSessionContext() SessionContext {
	return SessionContext{
		SessionID:   p.CoordinatorSessionID,
		AgentName:   p.AgentName,
		WorkspaceID: p.WorkspaceID,
		Workspace:   p.Workspace,
	}
}

func (p CoordinatorLifecyclePayload) hookSessionContext() SessionContext {
	return SessionContext{
		SessionID:   p.CoordinatorSessionID,
		AgentName:   p.AgentName,
		WorkspaceID: p.WorkspaceID,
		Workspace:   p.Workspace,
	}
}

func (p TaskRunEnqueuedPayload) hookSessionContext() SessionContext {
	return taskRunSessionContext(p.TaskRunContext)
}

func (p TaskRunPreClaimPayload) hookSessionContext() SessionContext {
	return taskRunSessionContext(p.TaskRunContext)
}

func (p TaskRunPostClaimPayload) hookSessionContext() SessionContext {
	return taskRunSessionContext(p.TaskRunContext)
}

func (p TaskRunLeasePayload) hookSessionContext() SessionContext {
	return taskRunSessionContext(p.TaskRunContext)
}

func (p SpawnPreCreatePayload) hookSessionContext() SessionContext {
	return spawnSessionContext(p.SpawnContext)
}

func (p SpawnLifecyclePayload) hookSessionContext() SessionContext {
	return spawnSessionContext(p.SpawnContext)
}

func taskRunSessionContext(ctx TaskRunContext) SessionContext {
	return SessionContext{
		SessionID:          ctx.SessionID,
		AgentName:          ctx.AgentName,
		WorkspaceID:        ctx.WorkspaceID,
		SessionSoulContext: optionalSessionSoulContext(ctx.SoulSnapshotID, ctx.SoulDigest),
	}
}

func spawnSessionContext(ctx SpawnContext) SessionContext {
	return SessionContext{
		SessionID:          ctx.ChildSessionID,
		AgentName:          ctx.AgentName,
		WorkspaceID:        ctx.WorkspaceID,
		Workspace:          ctx.Workspace,
		SessionSoulContext: optionalSessionSoulContext(ctx.SoulSnapshotID, ctx.SoulDigest),
	}
}

func optionalSessionSoulContext(snapshotID string, digest string) *SessionSoulContext {
	trimmedSnapshotID := strings.TrimSpace(snapshotID)
	trimmedDigest := strings.TrimSpace(digest)
	if trimmedSnapshotID == "" && trimmedDigest == "" {
		return nil
	}
	return &SessionSoulContext{
		SoulSnapshotID: trimmedSnapshotID,
		SoulDigest:     trimmedDigest,
	}
}
