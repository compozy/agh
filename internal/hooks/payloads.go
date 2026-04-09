package hooks

import (
	"encoding/json"
	"time"
)

// PayloadBase carries the common identifiers attached to every hook payload.
type PayloadBase struct {
	Event     HookEvent `json:"event"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// SessionContext carries the common session-scoped hook attributes.
type SessionContext struct {
	SessionID    string    `json:"session_id,omitempty"`
	SessionName  string    `json:"session_name,omitempty"`
	SessionType  string    `json:"session_type,omitempty"`
	AgentName    string    `json:"agent_name,omitempty"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	Workspace    string    `json:"workspace,omitempty"`
	ACPSessionID string    `json:"acp_session_id,omitempty"`
	State        string    `json:"state,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
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
	ToolCallID    string `json:"tool_call_id,omitempty"`
	ToolName      string `json:"tool_name,omitempty"`
	ToolNamespace string `json:"tool_namespace,omitempty"`
	ReadOnly      bool   `json:"read_only,omitempty"`
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
	ToolName      *string         `json:"tool_name,omitempty"`
	ToolNamespace *string         `json:"tool_namespace,omitempty"`
	ReadOnly      *bool           `json:"read_only,omitempty"`
	ToolInput     json.RawMessage `json:"tool_input,omitempty"`
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
	ToolCall      PermissionToolCall `json:"tool_call,omitempty"`
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
	ToolCall      PermissionToolCall `json:"tool_call,omitempty"`
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
