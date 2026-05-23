package contract

import (
	"github.com/compozy/agh/internal/transcript"
)

// SessionsResponse wraps the shared session list payload.
type SessionsResponse struct {
	Sessions []SessionPayload `json:"sessions"`
}

// SessionResponse wraps one shared session payload.
type SessionResponse struct {
	Session SessionPayload `json:"session"`
}

// SessionAttachResponse wraps one explicit session attach lease.
type SessionAttachResponse struct {
	Session SessionPayload       `json:"session"`
	Attach  SessionAttachPayload `json:"attach"`
}

// SessionRecapResponse wraps one deterministic session recap.
type SessionRecapResponse struct {
	Recap RecapPayload `json:"recap"`
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
	Messages []transcript.UIMessage `json:"messages"`
}

// SessionRepairResponse wraps the repair report for one session.
type SessionRepairResponse struct {
	Repair SessionRepairPayload `json:"repair"`
}

// SendPromptResultResponse wraps non-streaming busy-input prompt outcomes.
type SendPromptResultResponse struct {
	Prompt SendPromptResultPayload `json:"prompt"`
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

// AgentMeResponse wraps the resolved caller payload.
type AgentMeResponse struct {
	Me AgentMePayload `json:"me"`
}

// AgentContextResponse wraps the bounded caller situation payload.
type AgentContextResponse struct {
	Context AgentContextPayload `json:"context"`
}

// AgentChannelsResponse wraps discoverable coordination channels for the caller.
type AgentChannelsResponse struct {
	Channels []CoordinationChannelPayload `json:"channels"`
}

// AgentChannelMessagesResponse wraps channel inbox messages.
type AgentChannelMessagesResponse struct {
	Messages []AgentChannelMessagePayload `json:"messages"`
}

// AgentChannelMessageResponse wraps one sent channel message.
type AgentChannelMessageResponse struct {
	Message AgentChannelMessagePayload `json:"message"`
}

// AgentTaskClaimResponse wraps the synchronous task claim response.
type AgentTaskClaimResponse struct {
	Claim AgentTaskClaimPayload `json:"claim"`
}

// AgentTaskLeaseResponse wraps a safe task-run lease projection.
type AgentTaskLeaseResponse struct {
	Lease TaskRunLeaseSummaryPayload `json:"lease"`
}

// AgentSpawnResponse wraps a safe spawn result.
type AgentSpawnResponse struct {
	Spawn AgentSpawnPayload `json:"spawn"`
}

// AgentCoordinatorConfigResponse wraps coordinator config read state.
type AgentCoordinatorConfigResponse struct {
	Coordinator CoordinatorConfigPayload `json:"coordinator"`
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

// TasksResponse wraps the shared task list payload.
type TasksResponse struct {
	Tasks []TaskSummaryPayload `json:"tasks"`
}

// TaskResponse wraps one shared task payload.
type TaskResponse struct {
	Task TaskPayload `json:"task"`
}

// TaskExecutionResponse wraps one explicit task execution-boundary result.
type TaskExecutionResponse struct {
	Task TaskPayload    `json:"task"`
	Run  TaskRunPayload `json:"run"`
}

// TaskDetailResponse wraps one shared expanded task payload.
type TaskDetailResponse struct {
	Task TaskDetailPayload `json:"task"`
}

// TaskTimelineResponse wraps the shared task timeline payload.
type TaskTimelineResponse struct {
	Timeline []TaskTimelineItemPayload `json:"timeline"`
}

// TaskTreeResponse wraps the shared task-tree payload.
type TaskTreeResponse struct {
	Tree TaskTreePayload `json:"tree"`
}

// TaskRunsResponse wraps the shared task-run list payload.
type TaskRunsResponse struct {
	Runs []TaskRunPayload `json:"runs"`
}

// TaskRunResponse wraps one shared task-run payload.
type TaskRunResponse struct {
	Run TaskRunPayload `json:"run"`
}

// RetryTaskRunResponse wraps one retry source and newly queued run payload.
type RetryTaskRunResponse struct {
	PreviousRun TaskRunPayload `json:"previous_run"`
	Run         TaskRunPayload `json:"run"`
}

// BulkForceTaskRunResponse wraps bounded per-row force-operation results.
type BulkForceTaskRunResponse struct {
	Results []BulkForceTaskRunItemPayload `json:"results"`
}

// TaskRunReviewRequestResponse wraps one review request and idempotent-create marker.
type TaskRunReviewRequestResponse struct {
	Review  TaskRunReviewPayload `json:"review"`
	Created bool                 `json:"created"`
}

// TaskRunReviewResponse wraps one task-run review payload.
type TaskRunReviewResponse struct {
	Review TaskRunReviewPayload `json:"review"`
}

// TaskRunReviewsResponse wraps a task-run review list.
type TaskRunReviewsResponse struct {
	Reviews []TaskRunReviewPayload `json:"reviews"`
}

// TaskRunReviewVerdictResponse wraps one recorded review verdict result.
type TaskRunReviewVerdictResponse struct {
	Review          TaskRunReviewPayload `json:"review"`
	ContinuationRun *TaskRunPayload      `json:"continuation_run,omitempty"`
	CircuitOpened   bool                 `json:"circuit_opened,omitempty"`
}

// TaskRunDetailResponse wraps one shared task-run detail payload.
type TaskRunDetailResponse struct {
	Run TaskRunDetailPayload `json:"run"`
}

// TaskDashboardResponse wraps the shared task dashboard payload.
type TaskDashboardResponse struct {
	Dashboard TaskDashboardPayload `json:"dashboard"`
}

// TaskInboxResponse wraps the shared task inbox payload.
type TaskInboxResponse struct {
	Inbox TaskInboxPayload `json:"inbox"`
}

// TaskTriageStateResponse wraps the shared task triage-state payload.
type TaskTriageStateResponse struct {
	Triage TaskTriageStatePayload `json:"triage"`
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

// LogsListResponse wraps the runtime logs payload.
type LogsListResponse struct {
	Events []LogEventPayload `json:"events"`
}

// NetworkStatusResponse wraps the network runtime status payload.
type NetworkStatusResponse struct {
	Network NetworkStatusPayload `json:"network"`
}

// NetworkPeersResponse wraps the visible peer list payload.
type NetworkPeersResponse struct {
	Peers []NetworkPeerPayload `json:"peers"`
}

// NetworkChannelsResponse wraps the active channel list payload.
type NetworkChannelsResponse struct {
	Channels []NetworkChannelPayload `json:"channels"`
}

// CreateNetworkChannelResponse wraps the created channel detail payload.
type CreateNetworkChannelResponse struct {
	Channel NetworkChannelDetailPayload `json:"channel"`
}

// NetworkChannelResponse wraps one channel detail payload.
type NetworkChannelResponse struct {
	Channel NetworkChannelDetailPayload `json:"channel"`
}

// NetworkChannelMessagesResponse wraps the read-only channel timeline payload.
type NetworkChannelMessagesResponse struct {
	Messages []NetworkConversationMessagePayload `json:"messages"`
}

// NetworkPeerResponse wraps one selected peer detail payload.
type NetworkPeerResponse struct {
	Peer NetworkPeerDetailPayload `json:"peer"`
}

// NetworkPeerMessagesResponse wraps the peer-room timeline payload.
type NetworkPeerMessagesResponse struct {
	Messages []NetworkConversationMessagePayload `json:"messages"`
}

// NetworkSendResponse wraps the outbound send result payload.
type NetworkSendResponse struct {
	Message NetworkSendPayload `json:"message"`
}

// NetworkThreadsResponse wraps public-thread summaries.
type NetworkThreadsResponse struct {
	Threads []NetworkThreadSummaryPayload `json:"threads"`
}

// NetworkThreadResponse wraps one public-thread summary.
type NetworkThreadResponse struct {
	Thread NetworkThreadSummaryPayload `json:"thread"`
}

// NetworkThreadMessagesResponse wraps one public-thread message timeline.
type NetworkThreadMessagesResponse struct {
	Messages []NetworkConversationMessagePayload `json:"messages"`
}

// NetworkDirectRoomsResponse wraps direct-room summaries.
type NetworkDirectRoomsResponse struct {
	Directs []NetworkDirectRoomPayload `json:"directs"`
}

// NetworkDirectRoomResponse wraps one direct-room summary.
type NetworkDirectRoomResponse struct {
	Direct NetworkDirectRoomPayload `json:"direct"`
}

// NetworkDirectRoomMessagesResponse wraps one direct-room message timeline.
type NetworkDirectRoomMessagesResponse struct {
	Messages []NetworkConversationMessagePayload `json:"messages"`
}

// NetworkWorkResponse wraps one network work lookup.
type NetworkWorkResponse struct {
	Work NetworkWorkPayload `json:"work"`
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

// SkillMarketplaceSearchResponse wraps remote marketplace skill search results.
type SkillMarketplaceSearchResponse struct {
	Skills []SkillMarketplaceListingPayload `json:"skills"`
}

// SkillMarketplaceDetailResponse wraps one remote marketplace skill detail.
type SkillMarketplaceDetailResponse struct {
	Skill SkillMarketplaceDetailPayload `json:"skill"`
}

// SkillMarketplaceInstallResponse wraps one marketplace install result.
type SkillMarketplaceInstallResponse struct {
	Skill SkillMarketplaceInstallPayload `json:"skill"`
}

// SkillMarketplaceUpdateResponse wraps marketplace update results.
type SkillMarketplaceUpdateResponse struct {
	Skills []SkillMarketplaceUpdatePayload `json:"skills"`
}

// SkillMarketplaceRemoveResponse wraps one marketplace removal result.
type SkillMarketplaceRemoveResponse struct {
	Skill SkillMarketplaceRemovePayload `json:"skill"`
}

// SkillResponse wraps one shared skill payload.
type SkillResponse struct {
	Skill SkillPayload `json:"skill"`
}

// SkillShadowsResponse wraps the resolver shadow evidence for one skill name.
type SkillShadowsResponse struct {
	Name    string                    `json:"name"`
	Winner  SkillShadowEntryPayload   `json:"winner"`
	Shadows []SkillShadowEntryPayload `json:"shadows"`
}

// ExtensionsResponse wraps the extension list payload.
type ExtensionsResponse struct {
	Extensions []ExtensionPayload `json:"extensions"`
}

// ExtensionResponse wraps one extension payload.
type ExtensionResponse struct {
	Extension ExtensionPayload `json:"extension"`
}

// ExtensionMarketplaceResponse wraps daemon-owned extension marketplace results.
type ExtensionMarketplaceResponse struct {
	Extensions []ExtensionMarketplaceEntry `json:"extensions"`
}

// ExtensionProvenanceResponse wraps one installed extension provenance record.
type ExtensionProvenanceResponse struct {
	Provenance ExtensionProvenancePayload `json:"provenance"`
}

// ExtensionUpdateResponse wraps one marketplace extension update result.
type ExtensionUpdateResponse struct {
	Update ManagedExtensionUpdatePayload `json:"update"`
}

// ExtensionRemoveResponse wraps one removed extension result.
type ExtensionRemoveResponse struct {
	Extension ManagedExtensionRemovePayload `json:"extension"`
}

// ManagedExtensionUpdatePayload describes one daemon-owned extension update.
type ManagedExtensionUpdatePayload struct {
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Registry       string `json:"registry"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Path           string `json:"path"`
	Status         string `json:"status"`
}

// ManagedExtensionRemovePayload describes one daemon-owned extension removal.
type ManagedExtensionRemovePayload struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

// ResourcesResponse wraps the shared desired-state resource list payload.
type ResourcesResponse struct {
	Records []ResourceRecordPayload `json:"records"`
}

// ResourceResponse wraps one desired-state resource payload.
type ResourceResponse struct {
	Record ResourceRecordPayload `json:"record"`
}
