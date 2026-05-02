package contract

import (
	"encoding/json"
	"time"

	apicontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/memory"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod = extensionprotocol.HostAPIMethod

const (
	HostAPIMethodSessionsList                = extensionprotocol.HostAPIMethodSessionsList
	HostAPIMethodSessionsCreate              = extensionprotocol.HostAPIMethodSessionsCreate
	HostAPIMethodSessionsPrompt              = extensionprotocol.HostAPIMethodSessionsPrompt
	HostAPIMethodSessionsStop                = extensionprotocol.HostAPIMethodSessionsStop
	HostAPIMethodSessionsStatus              = extensionprotocol.HostAPIMethodSessionsStatus
	HostAPIMethodSessionsEvents              = extensionprotocol.HostAPIMethodSessionsEvents
	HostAPIMethodSessionsSoulRefresh         = extensionprotocol.HostAPIMethodSessionsSoulRefresh
	HostAPIMethodSessionsHealthGet           = extensionprotocol.HostAPIMethodSessionsHealthGet
	HostAPIMethodSessionsStatusGet           = extensionprotocol.HostAPIMethodSessionsStatusGet
	HostAPIMethodSandboxList                 = extensionprotocol.HostAPIMethodSandboxList
	HostAPIMethodSandboxInfo                 = extensionprotocol.HostAPIMethodSandboxInfo
	HostAPIMethodSandboxExec                 = extensionprotocol.HostAPIMethodSandboxExec
	HostAPIMethodMemoryRecall                = extensionprotocol.HostAPIMethodMemoryRecall
	HostAPIMethodMemoryStore                 = extensionprotocol.HostAPIMethodMemoryStore
	HostAPIMethodMemoryForget                = extensionprotocol.HostAPIMethodMemoryForget
	HostAPIMethodObserveHealth               = extensionprotocol.HostAPIMethodObserveHealth
	HostAPIMethodObserveEvents               = extensionprotocol.HostAPIMethodObserveEvents
	HostAPIMethodSkillsList                  = extensionprotocol.HostAPIMethodSkillsList
	HostAPIMethodAgentsSoulGet               = extensionprotocol.HostAPIMethodAgentsSoulGet
	HostAPIMethodAgentsSoulValidate          = extensionprotocol.HostAPIMethodAgentsSoulValidate
	HostAPIMethodAgentsSoulPut               = extensionprotocol.HostAPIMethodAgentsSoulPut
	HostAPIMethodAgentsSoulDelete            = extensionprotocol.HostAPIMethodAgentsSoulDelete
	HostAPIMethodAgentsSoulHistory           = extensionprotocol.HostAPIMethodAgentsSoulHistory
	HostAPIMethodAgentsSoulRollback          = extensionprotocol.HostAPIMethodAgentsSoulRollback
	HostAPIMethodAgentsHeartbeatGet          = extensionprotocol.HostAPIMethodAgentsHeartbeatGet
	HostAPIMethodAgentsHeartbeatValidate     = extensionprotocol.HostAPIMethodAgentsHeartbeatValidate
	HostAPIMethodAgentsHeartbeatPut          = extensionprotocol.HostAPIMethodAgentsHeartbeatPut
	HostAPIMethodAgentsHeartbeatDelete       = extensionprotocol.HostAPIMethodAgentsHeartbeatDelete
	HostAPIMethodAgentsHeartbeatHistory      = extensionprotocol.HostAPIMethodAgentsHeartbeatHistory
	HostAPIMethodAgentsHeartbeatRollback     = extensionprotocol.HostAPIMethodAgentsHeartbeatRollback
	HostAPIMethodAgentsHeartbeatStatus       = extensionprotocol.HostAPIMethodAgentsHeartbeatStatus
	HostAPIMethodAgentsHeartbeatWake         = extensionprotocol.HostAPIMethodAgentsHeartbeatWake
	HostAPIMethodAutomationJobs              = extensionprotocol.HostAPIMethodAutomationJobs
	HostAPIMethodAutomationJobsGet           = extensionprotocol.HostAPIMethodAutomationJobsGet
	HostAPIMethodAutomationJobsCreate        = extensionprotocol.HostAPIMethodAutomationJobsCreate
	HostAPIMethodAutomationJobsUpdate        = extensionprotocol.HostAPIMethodAutomationJobsUpdate
	HostAPIMethodAutomationJobsDelete        = extensionprotocol.HostAPIMethodAutomationJobsDelete
	HostAPIMethodAutomationJobsTrigger       = extensionprotocol.HostAPIMethodAutomationJobsTrigger
	HostAPIMethodAutomationJobsRuns          = extensionprotocol.HostAPIMethodAutomationJobsRuns
	HostAPIMethodAutomationTriggers          = extensionprotocol.HostAPIMethodAutomationTriggers
	HostAPIMethodAutomationTriggersGet       = extensionprotocol.HostAPIMethodAutomationTriggersGet
	HostAPIMethodAutomationTriggersCreate    = extensionprotocol.HostAPIMethodAutomationTriggersCreate
	HostAPIMethodAutomationTriggersUpdate    = extensionprotocol.HostAPIMethodAutomationTriggersUpdate
	HostAPIMethodAutomationTriggersDelete    = extensionprotocol.HostAPIMethodAutomationTriggersDelete
	HostAPIMethodAutomationTriggersRuns      = extensionprotocol.HostAPIMethodAutomationTriggersRuns
	HostAPIMethodAutomationTriggersFire      = extensionprotocol.HostAPIMethodAutomationTriggersFire
	HostAPIMethodAutomationRuns              = extensionprotocol.HostAPIMethodAutomationRuns
	HostAPIMethodTasks                       = extensionprotocol.HostAPIMethodTasks
	HostAPIMethodTasksGet                    = extensionprotocol.HostAPIMethodTasksGet
	HostAPIMethodTasksTimeline               = extensionprotocol.HostAPIMethodTasksTimeline
	HostAPIMethodTasksTree                   = extensionprotocol.HostAPIMethodTasksTree
	HostAPIMethodTasksDashboard              = extensionprotocol.HostAPIMethodTasksDashboard
	HostAPIMethodTasksInbox                  = extensionprotocol.HostAPIMethodTasksInbox
	HostAPIMethodTasksCreate                 = extensionprotocol.HostAPIMethodTasksCreate
	HostAPIMethodTasksUpdate                 = extensionprotocol.HostAPIMethodTasksUpdate
	HostAPIMethodTasksCancel                 = extensionprotocol.HostAPIMethodTasksCancel
	HostAPIMethodTasksRuns                   = extensionprotocol.HostAPIMethodTasksRuns
	HostAPIMethodTasksRunsGet                = extensionprotocol.HostAPIMethodTasksRunsGet
	HostAPIMethodTasksRunsEnqueue            = extensionprotocol.HostAPIMethodTasksRunsEnqueue
	HostAPIMethodTasksRunsClaim              = extensionprotocol.HostAPIMethodTasksRunsClaim
	HostAPIMethodTasksRunsStart              = extensionprotocol.HostAPIMethodTasksRunsStart
	HostAPIMethodTasksRunsAttachSession      = extensionprotocol.HostAPIMethodTasksRunsAttachSession
	HostAPIMethodTasksRunsComplete           = extensionprotocol.HostAPIMethodTasksRunsComplete
	HostAPIMethodTasksRunsFail               = extensionprotocol.HostAPIMethodTasksRunsFail
	HostAPIMethodTasksRunsCancel             = extensionprotocol.HostAPIMethodTasksRunsCancel
	HostAPIMethodResourcesList               = extensionprotocol.HostAPIMethodResourcesList
	HostAPIMethodResourcesGet                = extensionprotocol.HostAPIMethodResourcesGet
	HostAPIMethodResourcesSnapshot           = extensionprotocol.HostAPIMethodResourcesSnapshot
	HostAPIMethodBridgesInstancesList        = extensionprotocol.HostAPIMethodBridgesInstancesList
	HostAPIMethodBridgesMessagesIngest       = extensionprotocol.HostAPIMethodBridgesMessagesIngest
	HostAPIMethodBridgesInstancesGet         = extensionprotocol.HostAPIMethodBridgesInstancesGet
	HostAPIMethodBridgesInstancesReportState = extensionprotocol.HostAPIMethodBridgesInstancesReportState
)

// NamedType links a generated TypeScript export name to a Go type.
type NamedType struct {
	Name  string
	Value any
}

// HostAPIMethodSpec describes one Host API request/response contract.
type HostAPIMethodSpec struct {
	Method         HostAPIMethod
	Params         NamedType
	Result         NamedType
	OptionalParams bool
}

// EmptyResult is the empty JSON-RPC result for methods without payloads.
type EmptyResult struct{}

// SessionsListParams filters visible sessions.
type SessionsListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// SessionsCreateParams starts a new session.
type SessionsCreateParams struct {
	Agent     string `json:"agent"`
	Prompt    string `json:"prompt,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Workspace string `json:"workspace,omitempty"`
}

// SessionsPromptParams submits one prompt to an existing session.
type SessionsPromptParams struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// SessionTargetParams identifies an existing session.
type SessionTargetParams struct {
	SessionID string `json:"session_id"`
}

// SessionEventsParams filters persisted session events.
type SessionEventsParams struct {
	SessionID string    `json:"session_id"`
	Type      string    `json:"type,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	TurnID    string    `json:"turn_id,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int64     `json:"offset,omitempty"`
	Since     time.Time `json:"since"`
}

// SessionSoulRefreshParams refreshes one session's Soul snapshot through managed CAS.
type SessionSoulRefreshParams struct {
	SessionID string `json:"session_id"`
	apicontract.SessionSoulRefreshRequest
}

// SessionHealthGetParams identifies one session health row.
type SessionHealthGetParams = SessionTargetParams

// SessionStatusGetParams identifies one authored-context session status row.
type SessionStatusGetParams = SessionTargetParams

// SandboxListParams filters active sandboxes.
type SandboxListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// SandboxInfoParams identifies one session sandbox.
type SandboxInfoParams struct {
	SessionID string `json:"session_id"`
}

// SandboxExecParams executes one command inside a session sandbox.
type SandboxExecParams struct {
	SessionID string `json:"session_id"`
	Command   string `json:"command"`
	Timeout   int    `json:"timeout,omitempty"`
}

// MemoryStoreParams persists one memory document.
type MemoryStoreParams struct {
	Key       string       `json:"key"`
	Content   string       `json:"content"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
	Tags      []string     `json:"tags,omitempty"`
}

// MemoryRecallParams queries stored memory documents.
type MemoryRecallParams struct {
	Query     string       `json:"query"`
	Limit     int          `json:"limit,omitempty"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
}

// MemoryForgetParams removes one stored memory document.
type MemoryForgetParams struct {
	Key       string       `json:"key"`
	Scope     memory.Scope `json:"scope,omitempty"`
	Workspace string       `json:"workspace,omitempty"`
}

// ObserveEventsParams filters global observability events.
type ObserveEventsParams struct {
	SessionID string    `json:"session_id,omitempty"`
	AgentName string    `json:"agent_name,omitempty"`
	Type      string    `json:"type,omitempty"`
	Since     time.Time `json:"since"`
	Limit     int       `json:"limit,omitempty"`
}

// SkillsListParams filters skills by workspace scope.
type SkillsListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// AgentSoulGetParams identifies one workspace-visible Soul read model.
type AgentSoulGetParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name"`
}

// AgentSoulValidateParams validates current or proposed SOUL.md content.
type AgentSoulValidateParams = apicontract.AgentSoulValidateRequest

// AgentSoulPutParams creates or replaces SOUL.md through managed authoring.
type AgentSoulPutParams = apicontract.AgentSoulPutRequest

// AgentSoulDeleteParams deletes SOUL.md through managed authoring.
type AgentSoulDeleteParams = apicontract.AgentSoulDeleteRequest

// AgentSoulHistoryParams lists managed Soul authoring revisions.
type AgentSoulHistoryParams = apicontract.AgentSoulHistoryRequest

// AgentSoulRollbackParams restores a prior managed Soul revision.
type AgentSoulRollbackParams = apicontract.AgentSoulRollbackRequest

// AgentHeartbeatGetParams identifies one workspace-visible Heartbeat policy.
type AgentHeartbeatGetParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	AgentName   string `json:"agent_name"`
}

// AgentHeartbeatValidateParams validates proposed HEARTBEAT.md content.
type AgentHeartbeatValidateParams = apicontract.HeartbeatValidateRequest

// AgentHeartbeatPutParams creates or replaces HEARTBEAT.md through managed authoring.
type AgentHeartbeatPutParams = apicontract.HeartbeatPutRequest

// AgentHeartbeatDeleteParams deletes HEARTBEAT.md through managed authoring.
type AgentHeartbeatDeleteParams = apicontract.HeartbeatDeleteRequest

// AgentHeartbeatHistoryParams lists managed Heartbeat authoring revisions.
type AgentHeartbeatHistoryParams = apicontract.HeartbeatHistoryRequest

// AgentHeartbeatRollbackParams restores a prior Heartbeat revision or snapshot digest.
type AgentHeartbeatRollbackParams = apicontract.HeartbeatRollbackRequest

// AgentHeartbeatStatusParams composes Heartbeat policy, wake state, health, and wake audit.
type AgentHeartbeatStatusParams = apicontract.HeartbeatStatusRequest

// AgentHeartbeatWakeParams requests one managed advisory wake decision.
type AgentHeartbeatWakeParams = apicontract.HeartbeatWakeRequest

// AutomationJobsParams filters visible automation jobs.
type AutomationJobsParams struct {
	Scope       automationpkg.Scope `json:"scope,omitempty"`
	WorkspaceID string              `json:"workspace_id,omitempty"`
	Enabled     *bool               `json:"enabled,omitempty"`
}

// AutomationTriggersParams filters visible automation triggers.
type AutomationTriggersParams struct {
	Scope       automationpkg.Scope `json:"scope,omitempty"`
	WorkspaceID string              `json:"workspace_id,omitempty"`
	Event       string              `json:"event,omitempty"`
	Enabled     *bool               `json:"enabled,omitempty"`
}

// AutomationRunsParams filters visible automation runs.
type AutomationRunsParams struct {
	JobID     string                  `json:"job_id,omitempty"`
	TriggerID string                  `json:"trigger_id,omitempty"`
	Status    automationpkg.RunStatus `json:"status,omitempty"`
	Limit     int                     `json:"limit,omitempty"`
}

// AutomationTargetParams identifies one automation resource by id.
type AutomationTargetParams struct {
	ID string `json:"id"`
}

// AutomationJobCreateParams starts a new dynamic automation job.
type AutomationJobCreateParams = apicontract.CreateJobRequest

// AutomationJobUpdateParams patches one automation job definition by id.
type AutomationJobUpdateParams struct {
	ID string `json:"id"`
	apicontract.UpdateJobRequest
}

// AutomationJobTriggerParams forces one immediate automation job run.
type AutomationJobTriggerParams struct {
	ID      string         `json:"id"`
	Payload map[string]any `json:"payload,omitempty"`
}

// AutomationJobRunsParams filters run history for one automation job.
type AutomationJobRunsParams struct {
	ID     string                  `json:"id"`
	Status automationpkg.RunStatus `json:"status,omitempty"`
	Limit  int                     `json:"limit,omitempty"`
}

// AutomationTriggerCreateParams starts a new dynamic automation trigger.
type AutomationTriggerCreateParams = apicontract.CreateTriggerRequest

// AutomationTriggerUpdateParams patches one automation trigger definition by id.
type AutomationTriggerUpdateParams struct {
	ID string `json:"id"`
	apicontract.UpdateTriggerRequest
}

// AutomationTriggerRunsParams filters run history for one automation trigger.
type AutomationTriggerRunsParams struct {
	ID     string                  `json:"id"`
	Status automationpkg.RunStatus `json:"status,omitempty"`
	Limit  int                     `json:"limit,omitempty"`
}

// AutomationTriggerFireParams injects one extension-originated trigger event.
type AutomationTriggerFireParams struct {
	Event       string              `json:"event"`
	Scope       automationpkg.Scope `json:"scope"`
	WorkspaceID string              `json:"workspace_id,omitempty"`
	Payload     map[string]any      `json:"payload,omitempty"`
}

// TasksParams filters visible tasks.
type TasksParams = apicontract.TaskListQuery

// TaskTargetParams identifies one task by id.
type TaskTargetParams struct {
	ID string `json:"id"`
}

// TaskTimelineParams queries one task timeline by task id.
type TaskTimelineParams struct {
	ID string `json:"id"`
	apicontract.TaskTimelineQuery
}

// TaskTreeParams queries one task tree by task id.
type TaskTreeParams = TaskTargetParams

// TaskDashboardParams filters observer-backed task dashboard reads.
type TaskDashboardParams = apicontract.TaskDashboardQuery

// TaskInboxParams filters observer-backed task inbox reads.
type TaskInboxParams = apicontract.TaskInboxQuery

// TaskCreateParams creates one task.
type TaskCreateParams = apicontract.CreateTaskRequest

// TaskUpdateParams patches one task.
type TaskUpdateParams struct {
	ID string `json:"id"`
	apicontract.UpdateTaskRequest
}

// TaskCancelParams requests cancellation for one task.
type TaskCancelParams struct {
	ID string `json:"id"`
	apicontract.CancelTaskRequest
}

// TaskRunsParams filters runs for one task.
type TaskRunsParams struct {
	ID string `json:"id"`
	apicontract.TaskRunListQuery
}

// TaskRunGetParams identifies one task run by id for richer detail reads.
type TaskRunGetParams struct {
	ID string `json:"id"`
}

// TaskRunEnqueueParams enqueues one run for a task.
type TaskRunEnqueueParams struct {
	TaskID string `json:"task_id"`
	apicontract.EnqueueTaskRunRequest
}

// TaskRunClaimParams claims one queued run.
type TaskRunClaimParams struct {
	ID string `json:"id"`
	apicontract.ClaimTaskRunRequest
}

// TaskRunStartParams starts one claimed run.
type TaskRunStartParams struct {
	ID string `json:"id"`
	apicontract.StartTaskRunRequest
}

// TaskRunAttachSessionParams attaches one existing session to a run.
type TaskRunAttachSessionParams struct {
	ID string `json:"id"`
	apicontract.AttachTaskRunSessionRequest
}

// TaskRunCompleteParams completes one run.
type TaskRunCompleteParams struct {
	ID string `json:"id"`
	apicontract.CompleteTaskRunRequest
}

// TaskRunFailParams fails one run.
type TaskRunFailParams struct {
	ID string `json:"id"`
	apicontract.FailTaskRunRequest
}

// TaskRunCancelParams cancels one run.
type TaskRunCancelParams struct {
	ID string `json:"id"`
	apicontract.CancelTaskRunRequest
}

// ResourcesListParams filters same-source resource visibility for one extension actor.
type ResourcesListParams struct {
	Kind  resources.ResourceKind   `json:"kind,omitempty"`
	Scope *resources.ResourceScope `json:"scope,omitempty"`
	Limit int                      `json:"limit,omitempty"`
}

// ResourceGetParams identifies one canonical resource record by kind and id.
type ResourceGetParams struct {
	Kind resources.ResourceKind `json:"kind"`
	ID   string                 `json:"id"`
}

// ResourceSnapshotRecord carries one snapshot-authored resource definition.
type ResourceSnapshotRecord struct {
	Kind  resources.ResourceKind  `json:"kind"`
	ID    string                  `json:"id"`
	Scope resources.ResourceScope `json:"scope"`
	Spec  json.RawMessage         `json:"spec"`
}

// ResourcesSnapshotParams replaces one extension source snapshot.
type ResourcesSnapshotParams struct {
	SourceVersion int64                    `json:"source_version"`
	Records       []ResourceSnapshotRecord `json:"records"`
}

// BridgesMessagesIngestParams carries one normalized inbound bridge message.
type BridgesMessagesIngestParams = bridgepkg.InboundMessageEnvelope

// BridgeInstanceTargetParams identifies one provider-owned bridge instance.
type BridgeInstanceTargetParams struct {
	BridgeInstanceID string `json:"bridge_instance_id"`
}

// BridgesInstancesReportStateParams reports one adapter-observed instance status update.
type BridgesInstancesReportStateParams struct {
	BridgeInstanceID string                       `json:"bridge_instance_id"`
	Status           bridgepkg.BridgeStatus       `json:"status"`
	Degradation      *bridgepkg.BridgeDegradation `json:"degradation,omitempty"`
	ClearDegradation bool                         `json:"clear_degradation,omitempty"`
}

// SessionSummary is the lightweight host-visible session listing shape.
type SessionSummary struct {
	ID        string        `json:"id"`
	Name      string        `json:"name,omitempty"`
	Agent     string        `json:"agent"`
	Provider  string        `json:"provider"`
	Workspace string        `json:"workspace,omitempty"`
	State     session.State `json:"state"`
	CreatedAt time.Time     `json:"created_at"`
}

// SessionStatus is the detailed host-visible session status shape.
type SessionStatus struct {
	SessionID    string           `json:"session_id"`
	Name         string           `json:"name,omitempty"`
	Agent        string           `json:"agent"`
	Provider     string           `json:"provider"`
	WorkspaceID  string           `json:"workspace_id,omitempty"`
	Workspace    string           `json:"workspace,omitempty"`
	State        session.State    `json:"state"`
	StopReason   store.StopReason `json:"stop_reason,omitempty"`
	StopDetail   string           `json:"stop_detail,omitempty"`
	ACPSessionID string           `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// SessionEvent is the host-visible session or observe event record.
type SessionEvent struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// SessionCreateResult returns the created session identifier.
type SessionCreateResult struct {
	SessionID string `json:"session_id"`
	Provider  string `json:"provider"`
}

// SessionPromptResult returns the created turn identifier.
type SessionPromptResult struct {
	TurnID string `json:"turn_id"`
}

// SandboxSummary is one active sandbox in the host-visible list response.
type SandboxSummary struct {
	SessionID  string `json:"session_id"`
	SandboxID  string `json:"sandbox_id"`
	Backend    string `json:"backend"`
	Profile    string `json:"profile,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	State      string `json:"state"`
	SyncState  string `json:"sync_state,omitempty"`
}

// SandboxListResult returns active sandbox instances.
type SandboxListResult struct {
	Sandboxes []SandboxSummary `json:"sandboxes"`
}

// SandboxInfoResult returns detailed sandbox state for a session.
type SandboxInfoResult struct {
	SandboxID     string    `json:"sandbox_id"`
	Backend       string    `json:"backend"`
	Profile       string    `json:"profile"`
	InstanceID    string    `json:"instance_id"`
	RuntimeRoot   string    `json:"runtime_root"`
	SyncState     string    `json:"sync_state"`
	CreatedAt     time.Time `json:"created_at"`
	LastSyncError string    `json:"last_sync_error"`
}

// SandboxExecResult returns command execution output.
type SandboxExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
}

// MemoryRecallEntry is one scored memory lookup hit.
type MemoryRecallEntry struct {
	Key     string  `json:"key"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// SkillSummary is the lightweight host-visible skill listing shape.
type SkillSummary struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source"`
}

// ObserveHealth is the host-visible daemon health payload.
type ObserveHealth = observepkg.Health

// ResourceRecord is the generic Host API desired-state shape exposed to extensions.
type ResourceRecord struct {
	Kind      resources.ResourceKind   `json:"kind"`
	ID        string                   `json:"id"`
	Version   int64                    `json:"version"`
	Scope     resources.ResourceScope  `json:"scope"`
	Owner     resources.ResourceOwner  `json:"owner"`
	Source    resources.ResourceSource `json:"source"`
	Spec      json.RawMessage          `json:"spec"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
}

// BridgesMessagesIngestResult reports the resolved session association for one inbound message.
type BridgesMessagesIngestResult struct {
	SessionID    string               `json:"session_id"`
	RouteCreated bool                 `json:"route_created"`
	RoutingKey   bridgepkg.RoutingKey `json:"routing_key"`
}

var hostAPIMethodSpecs = []HostAPIMethodSpec{
	{
		Method:         HostAPIMethodSessionsList,
		Params:         NamedType{Name: "SessionsListParams", Value: SessionsListParams{}},
		Result:         NamedType{Name: "SessionSummary", Value: []SessionSummary{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodSessionsCreate,
		Params: NamedType{Name: "SessionsCreateParams", Value: SessionsCreateParams{}},
		Result: NamedType{Name: "SessionCreateResult", Value: SessionCreateResult{}},
	},
	{
		Method: HostAPIMethodSessionsPrompt,
		Params: NamedType{Name: "SessionsPromptParams", Value: SessionsPromptParams{}},
		Result: NamedType{Name: "SessionPromptResult", Value: SessionPromptResult{}},
	},
	{
		Method: HostAPIMethodSessionsStop,
		Params: NamedType{Name: "SessionTargetParams", Value: SessionTargetParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method: HostAPIMethodSessionsStatus,
		Params: NamedType{Name: "SessionTargetParams", Value: SessionTargetParams{}},
		Result: NamedType{Name: "SessionStatus", Value: SessionStatus{}},
	},
	{
		Method: HostAPIMethodSessionsEvents,
		Params: NamedType{Name: "SessionEventsParams", Value: SessionEventsParams{}},
		Result: NamedType{Name: "SessionEvent", Value: []SessionEvent{}},
	},
	{
		Method: HostAPIMethodSessionsSoulRefresh,
		Params: NamedType{Name: "SessionSoulRefreshParams", Value: SessionSoulRefreshParams{}},
		Result: NamedType{Name: "AgentSoulPayload", Value: apicontract.AgentSoulPayload{}},
	},
	{
		Method: HostAPIMethodSessionsHealthGet,
		Params: NamedType{Name: "SessionHealthGetParams", Value: SessionHealthGetParams{}},
		Result: NamedType{Name: "SessionHealthResponse", Value: apicontract.SessionHealthResponse{}},
	},
	{
		Method: HostAPIMethodSessionsStatusGet,
		Params: NamedType{Name: "SessionStatusGetParams", Value: SessionStatusGetParams{}},
		Result: NamedType{Name: "SessionStatusResponse", Value: apicontract.SessionStatusResponse{}},
	},
	{
		Method:         HostAPIMethodSandboxList,
		Params:         NamedType{Name: "SandboxListParams", Value: SandboxListParams{}},
		Result:         NamedType{Name: "SandboxListResult", Value: SandboxListResult{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodSandboxInfo,
		Params: NamedType{Name: "SandboxInfoParams", Value: SandboxInfoParams{}},
		Result: NamedType{Name: "SandboxInfoResult", Value: SandboxInfoResult{}},
	},
	{
		Method: HostAPIMethodSandboxExec,
		Params: NamedType{Name: "SandboxExecParams", Value: SandboxExecParams{}},
		Result: NamedType{Name: "SandboxExecResult", Value: SandboxExecResult{}},
	},
	{
		Method: HostAPIMethodMemoryRecall,
		Params: NamedType{Name: "MemoryRecallParams", Value: MemoryRecallParams{}},
		Result: NamedType{Name: "MemoryRecallEntry", Value: []MemoryRecallEntry{}},
	},
	{
		Method: HostAPIMethodMemoryStore,
		Params: NamedType{Name: "MemoryStoreParams", Value: MemoryStoreParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method: HostAPIMethodMemoryForget,
		Params: NamedType{Name: "MemoryForgetParams", Value: MemoryForgetParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method:         HostAPIMethodObserveHealth,
		Params:         NamedType{Name: "EmptyResult", Value: EmptyResult{}},
		Result:         NamedType{Name: "ObserveHealth", Value: ObserveHealth{}},
		OptionalParams: true,
	},
	{
		Method:         HostAPIMethodObserveEvents,
		Params:         NamedType{Name: "ObserveEventsParams", Value: ObserveEventsParams{}},
		Result:         NamedType{Name: "SessionEvent", Value: []SessionEvent{}},
		OptionalParams: true,
	},
	{
		Method:         HostAPIMethodSkillsList,
		Params:         NamedType{Name: "SkillsListParams", Value: SkillsListParams{}},
		Result:         NamedType{Name: "SkillSummary", Value: []SkillSummary{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodAgentsSoulGet,
		Params: NamedType{Name: "AgentSoulGetParams", Value: AgentSoulGetParams{}},
		Result: NamedType{Name: "AgentSoulPayload", Value: apicontract.AgentSoulPayload{}},
	},
	{
		Method: HostAPIMethodAgentsSoulValidate,
		Params: NamedType{Name: "AgentSoulValidateParams", Value: AgentSoulValidateParams{}},
		Result: NamedType{Name: "AgentSoulPayload", Value: apicontract.AgentSoulPayload{}},
	},
	{
		Method: HostAPIMethodAgentsSoulPut,
		Params: NamedType{Name: "AgentSoulPutParams", Value: AgentSoulPutParams{}},
		Result: NamedType{Name: "AgentSoulMutationResponse", Value: apicontract.AgentSoulMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsSoulDelete,
		Params: NamedType{Name: "AgentSoulDeleteParams", Value: AgentSoulDeleteParams{}},
		Result: NamedType{Name: "AgentSoulMutationResponse", Value: apicontract.AgentSoulMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsSoulHistory,
		Params: NamedType{Name: "AgentSoulHistoryParams", Value: AgentSoulHistoryParams{}},
		Result: NamedType{Name: "AgentSoulHistoryResponse", Value: apicontract.AgentSoulHistoryResponse{}},
	},
	{
		Method: HostAPIMethodAgentsSoulRollback,
		Params: NamedType{Name: "AgentSoulRollbackParams", Value: AgentSoulRollbackParams{}},
		Result: NamedType{Name: "AgentSoulMutationResponse", Value: apicontract.AgentSoulMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatGet,
		Params: NamedType{Name: "AgentHeartbeatGetParams", Value: AgentHeartbeatGetParams{}},
		Result: NamedType{Name: "HeartbeatPolicyPayload", Value: apicontract.HeartbeatPolicyPayload{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatValidate,
		Params: NamedType{Name: "AgentHeartbeatValidateParams", Value: AgentHeartbeatValidateParams{}},
		Result: NamedType{Name: "HeartbeatPolicyPayload", Value: apicontract.HeartbeatPolicyPayload{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatPut,
		Params: NamedType{Name: "AgentHeartbeatPutParams", Value: AgentHeartbeatPutParams{}},
		Result: NamedType{Name: "HeartbeatMutationResponse", Value: apicontract.HeartbeatMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatDelete,
		Params: NamedType{Name: "AgentHeartbeatDeleteParams", Value: AgentHeartbeatDeleteParams{}},
		Result: NamedType{Name: "HeartbeatMutationResponse", Value: apicontract.HeartbeatMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatHistory,
		Params: NamedType{Name: "AgentHeartbeatHistoryParams", Value: AgentHeartbeatHistoryParams{}},
		Result: NamedType{Name: "HeartbeatHistoryResponse", Value: apicontract.HeartbeatHistoryResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatRollback,
		Params: NamedType{Name: "AgentHeartbeatRollbackParams", Value: AgentHeartbeatRollbackParams{}},
		Result: NamedType{Name: "HeartbeatMutationResponse", Value: apicontract.HeartbeatMutationResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatStatus,
		Params: NamedType{Name: "AgentHeartbeatStatusParams", Value: AgentHeartbeatStatusParams{}},
		Result: NamedType{Name: "HeartbeatStatusResponse", Value: apicontract.HeartbeatStatusResponse{}},
	},
	{
		Method: HostAPIMethodAgentsHeartbeatWake,
		Params: NamedType{Name: "AgentHeartbeatWakeParams", Value: AgentHeartbeatWakeParams{}},
		Result: NamedType{Name: "HeartbeatWakeResponse", Value: apicontract.HeartbeatWakeResponse{}},
	},
	{
		Method:         HostAPIMethodAutomationJobs,
		Params:         NamedType{Name: "AutomationJobsParams", Value: AutomationJobsParams{}},
		Result:         NamedType{Name: "Job", Value: []automationpkg.Job{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodAutomationJobsGet,
		Params: NamedType{Name: "AutomationTargetParams", Value: AutomationTargetParams{}},
		Result: NamedType{Name: "Job", Value: automationpkg.Job{}},
	},
	{
		Method: HostAPIMethodAutomationJobsCreate,
		Params: NamedType{Name: "AutomationJobCreateParams", Value: AutomationJobCreateParams{}},
		Result: NamedType{Name: "Job", Value: automationpkg.Job{}},
	},
	{
		Method: HostAPIMethodAutomationJobsUpdate,
		Params: NamedType{Name: "AutomationJobUpdateParams", Value: AutomationJobUpdateParams{}},
		Result: NamedType{Name: "Job", Value: automationpkg.Job{}},
	},
	{
		Method: HostAPIMethodAutomationJobsDelete,
		Params: NamedType{Name: "AutomationTargetParams", Value: AutomationTargetParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method: HostAPIMethodAutomationJobsTrigger,
		Params: NamedType{Name: "AutomationJobTriggerParams", Value: AutomationJobTriggerParams{}},
		Result: NamedType{Name: "Run", Value: automationpkg.Run{}},
	},
	{
		Method: HostAPIMethodAutomationJobsRuns,
		Params: NamedType{Name: "AutomationJobRunsParams", Value: AutomationJobRunsParams{}},
		Result: NamedType{Name: "Run", Value: []automationpkg.Run{}},
	},
	{
		Method:         HostAPIMethodAutomationTriggers,
		Params:         NamedType{Name: "AutomationTriggersParams", Value: AutomationTriggersParams{}},
		Result:         NamedType{Name: "Trigger", Value: []apicontract.TriggerPayload{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodAutomationTriggersGet,
		Params: NamedType{Name: "AutomationTargetParams", Value: AutomationTargetParams{}},
		Result: NamedType{Name: "Trigger", Value: apicontract.TriggerPayload{}},
	},
	{
		Method: HostAPIMethodAutomationTriggersCreate,
		Params: NamedType{Name: "AutomationTriggerCreateParams", Value: AutomationTriggerCreateParams{}},
		Result: NamedType{Name: "Trigger", Value: apicontract.TriggerPayload{}},
	},
	{
		Method: HostAPIMethodAutomationTriggersUpdate,
		Params: NamedType{Name: "AutomationTriggerUpdateParams", Value: AutomationTriggerUpdateParams{}},
		Result: NamedType{Name: "Trigger", Value: apicontract.TriggerPayload{}},
	},
	{
		Method: HostAPIMethodAutomationTriggersDelete,
		Params: NamedType{Name: "AutomationTargetParams", Value: AutomationTargetParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method: HostAPIMethodAutomationTriggersRuns,
		Params: NamedType{Name: "AutomationTriggerRunsParams", Value: AutomationTriggerRunsParams{}},
		Result: NamedType{Name: "Run", Value: []automationpkg.Run{}},
	},
	{
		Method: HostAPIMethodAutomationTriggersFire,
		Params: NamedType{Name: "AutomationTriggerFireParams", Value: AutomationTriggerFireParams{}},
		Result: NamedType{Name: "TriggerResult", Value: automationpkg.TriggerResult{}},
	},
	{
		Method:         HostAPIMethodAutomationRuns,
		Params:         NamedType{Name: "AutomationRunsParams", Value: AutomationRunsParams{}},
		Result:         NamedType{Name: "Run", Value: []automationpkg.Run{}},
		OptionalParams: true,
	},
	{
		Method:         HostAPIMethodTasks,
		Params:         NamedType{Name: "TasksParams", Value: TasksParams{}},
		Result:         NamedType{Name: "TaskSummary", Value: []apicontract.TaskSummaryPayload{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodTasksGet,
		Params: NamedType{Name: "TaskTargetParams", Value: TaskTargetParams{}},
		Result: NamedType{Name: "TaskDetail", Value: apicontract.TaskDetailPayload{}},
	},
	{
		Method: HostAPIMethodTasksTimeline,
		Params: NamedType{Name: "TaskTimelineParams", Value: TaskTimelineParams{}},
		Result: NamedType{Name: "TaskTimelineItem", Value: []apicontract.TaskTimelineItemPayload{}},
	},
	{
		Method: HostAPIMethodTasksTree,
		Params: NamedType{Name: "TaskTreeParams", Value: TaskTreeParams{}},
		Result: NamedType{Name: "TaskTree", Value: apicontract.TaskTreePayload{}},
	},
	{
		Method:         HostAPIMethodTasksDashboard,
		Params:         NamedType{Name: "TaskDashboardParams", Value: TaskDashboardParams{}},
		Result:         NamedType{Name: "TaskDashboard", Value: apicontract.TaskDashboardPayload{}},
		OptionalParams: true,
	},
	{
		Method:         HostAPIMethodTasksInbox,
		Params:         NamedType{Name: "TaskInboxParams", Value: TaskInboxParams{}},
		Result:         NamedType{Name: "TaskInbox", Value: apicontract.TaskInboxPayload{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodTasksCreate,
		Params: NamedType{Name: "TaskCreateParams", Value: TaskCreateParams{}},
		Result: NamedType{Name: "Task", Value: apicontract.TaskPayload{}},
	},
	{
		Method: HostAPIMethodTasksUpdate,
		Params: NamedType{Name: "TaskUpdateParams", Value: TaskUpdateParams{}},
		Result: NamedType{Name: "Task", Value: apicontract.TaskPayload{}},
	},
	{
		Method: HostAPIMethodTasksCancel,
		Params: NamedType{Name: "TaskCancelParams", Value: TaskCancelParams{}},
		Result: NamedType{Name: "Task", Value: apicontract.TaskPayload{}},
	},
	{
		Method: HostAPIMethodTasksRuns,
		Params: NamedType{Name: "TaskRunsParams", Value: TaskRunsParams{}},
		Result: NamedType{Name: "TaskRun", Value: []apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsGet,
		Params: NamedType{Name: "TaskRunGetParams", Value: TaskRunGetParams{}},
		Result: NamedType{Name: "TaskRunDetail", Value: apicontract.TaskRunDetailPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsEnqueue,
		Params: NamedType{Name: "TaskRunEnqueueParams", Value: TaskRunEnqueueParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsClaim,
		Params: NamedType{Name: "TaskRunClaimParams", Value: TaskRunClaimParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsStart,
		Params: NamedType{Name: "TaskRunStartParams", Value: TaskRunStartParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsAttachSession,
		Params: NamedType{Name: "TaskRunAttachSessionParams", Value: TaskRunAttachSessionParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsComplete,
		Params: NamedType{Name: "TaskRunCompleteParams", Value: TaskRunCompleteParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsFail,
		Params: NamedType{Name: "TaskRunFailParams", Value: TaskRunFailParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method: HostAPIMethodTasksRunsCancel,
		Params: NamedType{Name: "TaskRunCancelParams", Value: TaskRunCancelParams{}},
		Result: NamedType{Name: "TaskRun", Value: apicontract.TaskRunPayload{}},
	},
	{
		Method:         HostAPIMethodResourcesList,
		Params:         NamedType{Name: "ResourcesListParams", Value: ResourcesListParams{}},
		Result:         NamedType{Name: "ResourceRecord", Value: []ResourceRecord{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodResourcesGet,
		Params: NamedType{Name: "ResourceGetParams", Value: ResourceGetParams{}},
		Result: NamedType{Name: "ResourceRecord", Value: ResourceRecord{}},
	},
	{
		Method: HostAPIMethodResourcesSnapshot,
		Params: NamedType{Name: "ResourcesSnapshotParams", Value: ResourcesSnapshotParams{}},
		Result: NamedType{Name: "EmptyResult", Value: EmptyResult{}},
	},
	{
		Method:         HostAPIMethodBridgesInstancesList,
		Params:         NamedType{Name: "EmptyResult", Value: EmptyResult{}},
		Result:         NamedType{Name: "BridgeInstance", Value: []bridgepkg.BridgeInstance{}},
		OptionalParams: true,
	},
	{
		Method: HostAPIMethodBridgesMessagesIngest,
		Params: NamedType{Name: "InboundMessageEnvelope", Value: bridgepkg.InboundMessageEnvelope{}},
		Result: NamedType{Name: "BridgesMessagesIngestResult", Value: BridgesMessagesIngestResult{}},
	},
	{
		Method: HostAPIMethodBridgesInstancesGet,
		Params: NamedType{Name: "BridgeInstanceTargetParams", Value: BridgeInstanceTargetParams{}},
		Result: NamedType{Name: "BridgeInstance", Value: bridgepkg.BridgeInstance{}},
	},
	{
		Method: HostAPIMethodBridgesInstancesReportState,
		Params: NamedType{Name: "BridgesInstancesReportStateParams", Value: BridgesInstancesReportStateParams{}},
		Result: NamedType{Name: "BridgeInstance", Value: bridgepkg.BridgeInstance{}},
	},
}

// HostAPIMethodSpecs returns the canonical Host API method registry in wire order.
func HostAPIMethodSpecs() []HostAPIMethodSpec {
	return append([]HostAPIMethodSpec(nil), hostAPIMethodSpecs...)
}
