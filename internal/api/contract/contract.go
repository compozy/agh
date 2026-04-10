// Package contract defines the canonical shared daemon API request and response DTOs.
package contract

import (
	"encoding/json"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// CreateSessionRequest is the shared session creation request payload.
type CreateSessionRequest struct {
	AgentName     string `json:"agent_name"`
	Name          string `json:"name"`
	Workspace     string `json:"workspace"`
	WorkspacePath string `json:"workspace_path"`
}

// ApproveSessionRequest is the interactive permission approval payload.
type ApproveSessionRequest struct {
	RequestID string `json:"request_id"`
	TurnID    string `json:"turn_id"`
	Decision  string `json:"decision"`
}

// SessionPayload is the shared session response payload.
type SessionPayload struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	AgentName     string `json:"agent_name"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	WorkspacePath string `json:"workspace_path,omitempty"`
	State         string `json:"state"`
	// StopReason is the session-level stop classification, distinct from AgentEventPayload.StopReason.
	StopReason string `json:"stop_reason,omitempty"`
	// StopDetail is the session-level stop context paired with StopReason.
	StopDetail   string          `json:"stop_detail,omitempty"`
	ACPSessionID string          `json:"acp_session_id,omitempty"`
	ACPCaps      *ACPCapsPayload `json:"acp_caps,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ACPCapsPayload is the JSON representation of ACP capabilities.
type ACPCapsPayload struct {
	SupportsLoadSession bool     `json:"supports_load_session"`
	SupportedModes      []string `json:"supported_modes,omitempty"`
	SupportedModels     []string `json:"supported_models,omitempty"`
}

// SessionEventPayload is the shared session event response payload.
type SessionEventPayload struct {
	ID            string          `json:"id"`
	SessionID     string          `json:"session_id"`
	Sequence      int64           `json:"sequence"`
	TurnID        string          `json:"turn_id"`
	Type          string          `json:"type"`
	AgentName     string          `json:"agent_name"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	WorkspacePath string          `json:"workspace_path,omitempty"`
	Content       json.RawMessage `json:"content"`
	Timestamp     time.Time       `json:"timestamp"`
}

// TurnHistoryPayload is the shared turn history response payload.
type TurnHistoryPayload struct {
	TurnID string                `json:"turn_id"`
	Events []SessionEventPayload `json:"events"`
}

// AgentPayload is the shared agent definition response payload.
type AgentPayload struct {
	Name        string               `json:"name"`
	Provider    string               `json:"provider"`
	Command     string               `json:"command,omitempty"`
	Model       string               `json:"model,omitempty"`
	Tools       []string             `json:"tools,omitempty"`
	Permissions string               `json:"permissions,omitempty"`
	MCPServers  []AgentMCPServerJSON `json:"mcp_servers,omitempty"`
	Prompt      string               `json:"prompt"`
}

// AgentMCPServerJSON is the shared MCP server response payload.
type AgentMCPServerJSON struct {
	Name    string            `json:"name"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// AgentEventPayload is the shared raw agent-event streaming payload.
type AgentEventPayload struct {
	Type       string             `json:"type"`
	SessionID  string             `json:"session_id,omitempty"`
	TurnID     string             `json:"turn_id,omitempty"`
	RequestID  string             `json:"request_id,omitempty"`
	Timestamp  time.Time          `json:"timestamp"`
	Text       string             `json:"text,omitempty"`
	Title      string             `json:"title,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	StopReason string             `json:"stop_reason,omitempty"`
	Action     string             `json:"action,omitempty"`
	Resource   string             `json:"resource,omitempty"`
	Decision   string             `json:"decision,omitempty"`
	Error      string             `json:"error,omitempty"`
	Usage      *TokenUsagePayload `json:"usage,omitempty"`
	Raw        json.RawMessage    `json:"raw,omitempty"`
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
	Status             string `json:"status"`
	UptimeSeconds      int64  `json:"uptime_seconds"`
	ActiveSessions     int    `json:"active_sessions"`
	ActiveAgents       int    `json:"active_agents"`
	GlobalDBSizeBytes  int64  `json:"global_db_size_bytes"`
	SessionDBSizeBytes int64  `json:"session_db_size_bytes"`
	Version            string `json:"version"`
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
	Matcher      hookspkg.HookMatcher `json:"matcher,omitempty"`
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
	Status         string    `json:"status"`
	PID            int       `json:"pid"`
	StartedAt      time.Time `json:"started_at"`
	Socket         string    `json:"socket"`
	HTTPHost       string    `json:"http_host"`
	HTTPPort       int       `json:"http_port"`
	ActiveSessions int       `json:"active_sessions"`
	TotalSessions  int       `json:"total_sessions"`
	Version        string    `json:"version,omitempty"`
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

// MemoryHealthPayload is the shared memory health response payload.
type MemoryHealthPayload struct {
	GlobalFiles       int        `json:"global_files"`
	WorkspaceFiles    int        `json:"workspace_files"`
	LastConsolidation *time.Time `json:"last_consolidation"`
	DreamEnabled      bool       `json:"dream_enabled"`
}

// CreateWorkspaceRequest is the shared workspace creation request payload.
type CreateWorkspaceRequest struct {
	RootDir      string   `json:"root_dir"`
	Name         string   `json:"name"`
	AddDirs      []string `json:"add_dirs"`
	DefaultAgent string   `json:"default_agent"`
}

// UpdateWorkspaceRequest is the shared workspace update request payload.
type UpdateWorkspaceRequest struct {
	Name         *string   `json:"name"`
	AddDirs      *[]string `json:"add_dirs"`
	DefaultAgent *string   `json:"default_agent"`
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
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// WorkspaceSkillPayload is the shared workspace skill response payload.
type WorkspaceSkillPayload struct {
	Name   string `json:"name"`
	Dir    string `json:"dir"`
	Source string `json:"source"`
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
	Workspace WorkspacePayload        `json:"workspace"`
	Sessions  []SessionPayload        `json:"sessions,omitempty"`
	Agents    []AgentPayload          `json:"agents,omitempty"`
	Skills    []WorkspaceSkillPayload `json:"skills,omitempty"`
}
