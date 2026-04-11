package contract

import (
	"github.com/pedronauck/agh/internal/transcript"
)

// SessionsResponse wraps the shared session list payload.
type SessionsResponse struct {
	Sessions []SessionPayload `json:"sessions"`
}

// SessionResponse wraps one shared session payload.
type SessionResponse struct {
	Session SessionPayload `json:"session"`
}

// SessionEventsResponse wraps the shared session events payload.
type SessionEventsResponse struct {
	Events []SessionEventPayload `json:"events"`
}

// SessionHistoryResponse wraps the shared grouped turn history payload.
type SessionHistoryResponse struct {
	History []TurnHistoryPayload `json:"history"`
}

// SessionTranscriptResponse wraps the canonical transcript payload.
type SessionTranscriptResponse struct {
	Messages []transcript.Message `json:"messages"`
}

// SessionApprovalResponse wraps the approve-session success payload.
type SessionApprovalResponse struct {
	Status string `json:"status"`
}

// AgentsResponse wraps the shared agent list payload.
type AgentsResponse struct {
	Agents []AgentPayload `json:"agents"`
}

// AgentResponse wraps one shared agent payload.
type AgentResponse struct {
	Agent AgentPayload `json:"agent"`
}

// JobsResponse wraps the shared automation job list payload.
type JobsResponse struct {
	Jobs []JobPayload `json:"jobs"`
}

// JobResponse wraps one shared automation job payload.
type JobResponse struct {
	Job JobPayload `json:"job"`
}

// TriggersResponse wraps the shared automation trigger list payload.
type TriggersResponse struct {
	Triggers []TriggerPayload `json:"triggers"`
}

// TriggerResponse wraps one shared automation trigger payload.
type TriggerResponse struct {
	Trigger TriggerPayload `json:"trigger"`
}

// RunsResponse wraps the shared automation run list payload.
type RunsResponse struct {
	Runs []RunPayload `json:"runs"`
}

// RunResponse wraps one shared automation run payload.
type RunResponse struct {
	Run RunPayload `json:"run"`
}

// WebhookDeliveryResponse wraps the shared webhook delivery result payload.
type WebhookDeliveryResponse struct {
	Result WebhookDeliveryPayload `json:"result"`
}

// HookCatalogResponse wraps the resolved hook catalog payload.
type HookCatalogResponse struct {
	Hooks []HookCatalogPayload `json:"hooks"`
}

// HookRunsResponse wraps the hook run history payload.
type HookRunsResponse struct {
	Runs []HookRunPayload `json:"runs"`
}

// HookEventsResponse wraps the hook taxonomy payload.
type HookEventsResponse struct {
	Events []HookEventPayload `json:"events"`
}

// ObserveEventsResponse wraps the observe events payload.
type ObserveEventsResponse struct {
	Events []ObserveEventPayload `json:"events"`
}

// HealthResponse wraps daemon health plus memory health.
type HealthResponse struct {
	Health     ObserveHealthPayload    `json:"health"`
	Memory     MemoryHealthPayload     `json:"memory"`
	Automation AutomationHealthPayload `json:"automation"`
}

// DaemonStatusResponse wraps the daemon status payload.
type DaemonStatusResponse struct {
	Daemon DaemonStatusPayload `json:"daemon"`
}

// NetworkStatusResponse wraps the network runtime status payload.
type NetworkStatusResponse struct {
	Network NetworkStatusPayload `json:"network"`
}

// NetworkPeersResponse wraps the visible peer list payload.
type NetworkPeersResponse struct {
	Peers []NetworkPeerPayload `json:"peers"`
}

// NetworkSpacesResponse wraps the active space list payload.
type NetworkSpacesResponse struct {
	Spaces []NetworkSpacePayload `json:"spaces"`
}

// NetworkSendResponse wraps the outbound send result payload.
type NetworkSendResponse struct {
	Message NetworkSendPayload `json:"message"`
}

// NetworkInboxResponse wraps the queued inbox payload.
type NetworkInboxResponse struct {
	Messages []NetworkEnvelopePayload `json:"messages"`
}

// WorkspacesResponse wraps the shared workspace list payload.
type WorkspacesResponse struct {
	Workspaces []WorkspacePayload `json:"workspaces"`
}

// WorkspaceResponse wraps one shared workspace payload.
type WorkspaceResponse struct {
	Workspace WorkspacePayload `json:"workspace"`
}

// SkillsResponse wraps the shared skill list payload.
type SkillsResponse struct {
	Skills []SkillPayload `json:"skills"`
}

// SkillResponse wraps one shared skill payload.
type SkillResponse struct {
	Skill SkillPayload `json:"skill"`
}

// ExtensionsResponse wraps the extension list payload.
type ExtensionsResponse struct {
	Extensions []ExtensionPayload `json:"extensions"`
}

// ExtensionResponse wraps one extension payload.
type ExtensionResponse struct {
	Extension ExtensionPayload `json:"extension"`
}
