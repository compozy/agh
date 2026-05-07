package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	mcppkg "github.com/pedronauck/agh/internal/mcp"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/sse"
)

const (
	baseURL                        = "http://unix"
	defaultUnixSocketClientTimeout = 30 * time.Second
	defaultUserAgentName           = "agh-cli"
)

// DaemonClient is the CLI transport surface for talking to the AGH daemon over UDS.
type DaemonClient interface {
	DaemonStatus(ctx context.Context) (DaemonStatus, error)
	TriggerSettingsRestart(ctx context.Context) (SettingsRestartActionRecord, error)
	GetSettingsRestartStatus(ctx context.Context, operationID string) (SettingsRestartStatusRecord, error)
	GetSettingsUpdate(ctx context.Context) (SettingsUpdateRecord, error)
	UpdateSettingsSkills(ctx context.Context, request UpdateSettingsSkillsRequest) (SettingsMutationRecord, error)
	ListProviderModels(ctx context.Context, query ProviderModelListQuery) (ProviderModelListRecord, error)
	RefreshProviderModels(ctx context.Context, providerID string, request ProviderModelRefreshRequest) (
		ProviderModelRefreshRecord,
		error,
	)
	ProviderModelStatus(ctx context.Context, providerID string) (ProviderModelStatusRecord, error)
	ListVaultSecrets(ctx context.Context, query VaultListQuery) ([]VaultRecord, error)
	GetVaultSecret(ctx context.Context, ref string) (VaultRecord, error)
	PutVaultSecret(ctx context.Context, request PutVaultSecretRequest) (VaultRecord, error)
	DeleteVaultSecret(ctx context.Context, ref string) error
	NetworkStatus(ctx context.Context) (NetworkStatusRecord, error)
	NetworkPeers(ctx context.Context, query NetworkPeersQuery) ([]NetworkPeerRecord, error)
	NetworkChannels(ctx context.Context) ([]NetworkChannelRecord, error)
	NetworkThreads(ctx context.Context, query NetworkThreadsQuery) ([]NetworkThreadRecord, error)
	NetworkThread(ctx context.Context, channel string, threadID string) (NetworkThreadRecord, error)
	NetworkThreadMessages(
		ctx context.Context,
		query NetworkConversationMessagesQuery,
	) ([]NetworkConversationMessageRecord, error)
	NetworkDirects(ctx context.Context, query NetworkDirectsQuery) ([]NetworkDirectRoomRecord, error)
	NetworkDirectResolve(
		ctx context.Context,
		channel string,
		request NetworkDirectResolveRequest,
	) (NetworkDirectRoomRecord, error)
	NetworkDirect(ctx context.Context, channel string, directID string) (NetworkDirectRoomRecord, error)
	NetworkDirectMessages(
		ctx context.Context,
		query NetworkConversationMessagesQuery,
	) ([]NetworkConversationMessageRecord, error)
	NetworkWork(ctx context.Context, workID string) (NetworkWorkRecord, error)
	NetworkSend(ctx context.Context, request NetworkSendRequest) (NetworkSendRecord, error)
	NetworkInbox(ctx context.Context, sessionID string) ([]NetworkEnvelopeRecord, error)
	ListExtensions(ctx context.Context) ([]ExtensionRecord, error)
	InstallExtension(ctx context.Context, request InstallExtensionRequest) (ExtensionRecord, error)
	EnableExtension(ctx context.Context, name string) (ExtensionRecord, error)
	DisableExtension(ctx context.Context, name string) (ExtensionRecord, error)
	ExtensionStatus(ctx context.Context, name string) (ExtensionRecord, error)
	ListBundleCatalog(ctx context.Context) ([]BundleCatalogRecord, error)
	PreviewBundleActivation(ctx context.Context, request ActivateBundleRequest) (BundleActivationRecord, error)
	ActivateBundle(ctx context.Context, request ActivateBundleRequest) (BundleActivationRecord, error)
	ListBundleActivations(ctx context.Context) ([]BundleActivationRecord, error)
	GetBundleActivation(ctx context.Context, id string) (BundleActivationRecord, error)
	UpdateBundleActivation(
		ctx context.Context,
		id string,
		request UpdateBundleActivationRequest,
	) (BundleActivationRecord, error)
	DeactivateBundle(ctx context.Context, id string) error
	BundleNetworkSettings(ctx context.Context) (BundleNetworkSettingsRecord, error)
	ListBridges(ctx context.Context) ([]BridgeRecord, error)
	CreateBridge(ctx context.Context, request CreateBridgeRequest) (BridgeRecord, error)
	GetBridge(ctx context.Context, id string) (BridgeRecord, error)
	UpdateBridge(ctx context.Context, id string, request UpdateBridgeRequest) (BridgeRecord, error)
	EnableBridge(ctx context.Context, id string) (BridgeRecord, error)
	DisableBridge(ctx context.Context, id string) (BridgeRecord, error)
	RestartBridge(ctx context.Context, id string) (BridgeRecord, error)
	BridgeRoutes(ctx context.Context, id string) ([]BridgeRouteRecord, error)
	ListBridgeSecretBindings(ctx context.Context, id string) ([]BridgeSecretBindingRecord, error)
	PutBridgeSecretBinding(
		ctx context.Context,
		id string,
		bindingName string,
		request BridgeSecretBindingRequest,
	) (BridgeSecretBindingRecord, error)
	DeleteBridgeSecretBinding(ctx context.Context, id string, bindingName string) error
	TestBridgeDelivery(
		ctx context.Context,
		id string,
		request BridgeTestDeliveryRequest,
	) (BridgeTestDeliveryRecord, error)
	ListSessions(ctx context.Context, query SessionListQuery) ([]SessionRecord, error)
	CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error)
	GetSession(ctx context.Context, id string) (SessionRecord, error)
	GetSessionHealth(ctx context.Context, id string) (SessionHealthRecord, error)
	GetSessionStatus(ctx context.Context, id string) (SessionStatusRecord, error)
	InspectSession(ctx context.Context, id string, query SessionInspectQuery) (SessionInspectRecord, error)
	RefreshSessionSoul(ctx context.Context, id string, request SessionSoulRefreshRequest) (AgentSoulRecord, error)
	StopSession(ctx context.Context, id string) error
	ResumeSession(ctx context.Context, id string) (SessionRecord, error)
	RepairSession(ctx context.Context, id string, query SessionRepairQuery) (SessionRepairRecord, error)
	ApproveSession(ctx context.Context, id string, request SessionApprovalRequest) (SessionApprovalRecord, error)
	PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error)
	StreamPromptSession(ctx context.Context, id string, message string, handler SSEHandler) error
	SessionEvents(ctx context.Context, id string, query SessionEventQuery) ([]SessionEventRecord, error)
	StreamSessionEvents(
		ctx context.Context,
		id string,
		query SessionEventQuery,
		lastEventID string,
		handler SSEHandler,
	) error
	SessionHistory(ctx context.Context, id string, query SessionEventQuery) ([]TurnHistoryRecord, error)
	CreateWorkspace(ctx context.Context, request WorkspaceCreateRequest) (WorkspaceRecord, error)
	ListWorkspaces(ctx context.Context) ([]WorkspaceRecord, error)
	GetWorkspace(ctx context.Context, ref string) (WorkspaceDetailRecord, error)
	UpdateWorkspace(ctx context.Context, ref string, request WorkspaceUpdateRequest) (WorkspaceRecord, error)
	DeleteWorkspace(ctx context.Context, ref string) error
	ListAgents(ctx context.Context, query AgentQuery) ([]AgentRecord, error)
	GetAgent(ctx context.Context, name string, query AgentQuery) (AgentRecord, error)
	GetAgentSoul(ctx context.Context, name string, query AgentQuery) (AgentSoulRecord, error)
	ValidateAgentSoul(ctx context.Context, name string, request AgentSoulValidateRequest) (AgentSoulRecord, error)
	PutAgentSoul(ctx context.Context, name string, request AgentSoulPutRequest) (AgentSoulMutationRecord, error)
	DeleteAgentSoul(ctx context.Context, name string, request AgentSoulDeleteRequest) (AgentSoulMutationRecord, error)
	ListAgentSoulHistory(
		ctx context.Context,
		name string,
		request AgentSoulHistoryRequest,
	) (AgentSoulHistoryRecord, error)
	RollbackAgentSoul(
		ctx context.Context,
		name string,
		request AgentSoulRollbackRequest,
	) (AgentSoulMutationRecord, error)
	GetAgentHeartbeat(
		ctx context.Context,
		name string,
		query AgentQuery,
	) (AgentHeartbeatRecord, error)
	ValidateAgentHeartbeat(
		ctx context.Context,
		name string,
		request AgentHeartbeatValidateRequest,
	) (AgentHeartbeatRecord, error)
	PutAgentHeartbeat(
		ctx context.Context,
		name string,
		request AgentHeartbeatPutRequest,
	) (AgentHeartbeatMutationRecord, error)
	DeleteAgentHeartbeat(
		ctx context.Context,
		name string,
		request AgentHeartbeatDeleteRequest,
	) (AgentHeartbeatMutationRecord, error)
	ListAgentHeartbeatHistory(
		ctx context.Context,
		name string,
		request AgentHeartbeatHistoryRequest,
	) (AgentHeartbeatHistoryRecord, error)
	RollbackAgentHeartbeat(
		ctx context.Context,
		name string,
		request AgentHeartbeatRollbackRequest,
	) (AgentHeartbeatMutationRecord, error)
	GetAgentHeartbeatStatus(
		ctx context.Context,
		name string,
		request AgentHeartbeatStatusRequest,
	) (AgentHeartbeatStatusRecord, error)
	WakeAgentHeartbeat(
		ctx context.Context,
		name string,
		request AgentHeartbeatWakeRequest,
	) (AgentHeartbeatWakeDecisionRecord, error)
	ListResources(ctx context.Context, query ResourceListQuery) ([]ResourceRecord, error)
	GetResource(ctx context.Context, kind string, id string) (ResourceRecord, error)
	PutResource(ctx context.Context, kind string, id string, request ResourcePutRequest) (ResourceRecord, error)
	DeleteResource(ctx context.Context, kind string, id string, request ResourceDeleteRequest) error
	ListSkills(ctx context.Context, query SkillQuery) ([]SkillRecord, error)
	GetSkill(ctx context.Context, name string, query SkillQuery) (SkillRecord, error)
	GetSkillContent(ctx context.Context, name string, query SkillQuery) (string, error)
	EnableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error)
	DisableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error)
	ListTools(ctx context.Context, query ToolQuery) (ToolsResponseRecord, error)
	SearchTools(ctx context.Context, request ToolSearchRequest) (ToolsResponseRecord, error)
	GetTool(ctx context.Context, id string, query ToolQuery) (ToolResponseRecord, error)
	CreateToolApproval(ctx context.Context, id string, request ToolApprovalRequest) (ToolApprovalRecord, error)
	InvokeTool(ctx context.Context, id string, request ToolInvokeRequest) (ToolInvokeResponseRecord, error)
	ListToolsets(ctx context.Context, query ToolQuery) (ToolsetsResponseRecord, error)
	GetToolset(ctx context.Context, id string, query ToolQuery) (ToolsetResponseRecord, error)
	HookCatalog(ctx context.Context, query HookCatalogQuery) ([]HookCatalogRecord, error)
	HookRuns(ctx context.Context, query HookRunsQuery) ([]HookRunRecord, error)
	HookEvents(ctx context.Context, query HookEventsQuery) ([]HookEventRecord, error)
	ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error)
	StreamObserveEvents(ctx context.Context, query ObserveEventQuery, lastEventID string, handler SSEHandler) error
	ObserveHealth(ctx context.Context) (HealthStatus, error)
	MemoryHealth(ctx context.Context, workspace string) (MemoryHealthRecord, error)
	MemoryHistory(ctx context.Context, query MemoryHistoryQuery) ([]MemoryHistoryRecord, error)
	ListMemory(ctx context.Context, query MemoryListQuery) (MemoryListRecord, error)
	ShowMemory(ctx context.Context, filename string, query MemorySelectorQuery) (MemoryEntryRecord, error)
	CreateMemory(ctx context.Context, request MemoryCreateRequest) (MemoryMutationRecord, error)
	EditMemory(ctx context.Context, filename string, request MemoryEditRequest) (MemoryMutationRecord, error)
	DeleteMemory(ctx context.Context, filename string, query MemorySelectorQuery) (MemoryDeleteRecord, error)
	SearchMemory(ctx context.Context, request MemorySearchRequest) (MemorySearchRecord, error)
	ReindexMemory(ctx context.Context, request MemoryReindexRequest) (MemoryReindexRecord, error)
	PromoteMemory(ctx context.Context, request MemoryPromoteRequest) (MemoryPromoteRecord, error)
	ResetMemory(ctx context.Context, request MemoryResetRequest) (MemoryResetRecord, error)
	ReloadMemory(ctx context.Context, request MemorySelectorQuery) (MemoryReloadRecord, error)
	MemoryScopeShow(ctx context.Context, query MemorySelectorQuery) (MemoryScopeShowRecord, error)
	ListMemoryDecisions(ctx context.Context, query MemoryDecisionListQuery) (MemoryDecisionListRecord, error)
	GetMemoryDecision(ctx context.Context, id string) (MemoryDecisionRecord, error)
	RevertMemoryDecision(
		ctx context.Context,
		id string,
		request MemoryDecisionRevertRequest,
	) (MemoryDecisionRevertRecord, error)
	GetMemoryRecallTrace(ctx context.Context, sessionID string, turnSeq int64) (MemoryRecallTraceRecord, error)
	ListMemoryDreams(ctx context.Context) (MemoryDreamListRecord, error)
	GetMemoryDream(ctx context.Context, id string) (MemoryDreamRecord, error)
	TriggerMemoryDream(ctx context.Context, request MemoryDreamTriggerRequest) (MemoryDreamTriggerRecord, error)
	RetryMemoryDream(ctx context.Context, id string, request MemoryDreamRetryRequest) (MemoryDreamRetryRecord, error)
	GetMemoryDreamStatus(ctx context.Context) (MemoryDreamListRecord, error)
	ListMemoryDailyLogs(ctx context.Context, query MemorySelectorQuery) (MemoryDailyLogListRecord, error)
	GetMemoryExtractorStatus(ctx context.Context, sessionID string) (MemoryExtractorStatusRecord, error)
	ListMemoryExtractorFailures(ctx context.Context) (MemoryExtractorFailuresRecord, error)
	RetryMemoryExtractor(ctx context.Context, request MemoryExtractorRetryRequest) (MemoryExtractorRetryRecord, error)
	DrainMemoryExtractor(ctx context.Context) (MemoryExtractorDrainRecord, error)
	ListMemoryProviders(ctx context.Context) (MemoryProviderListRecord, error)
	GetMemoryProvider(ctx context.Context, name string) (MemoryProviderRecord, error)
	SelectMemoryProvider(
		ctx context.Context,
		request MemoryProviderSelectRequest,
	) (MemoryProviderLifecycleRecord, error)
	EnableMemoryProvider(
		ctx context.Context,
		name string,
		request MemoryProviderLifecycleRequest,
	) (MemoryProviderLifecycleRecord, error)
	DisableMemoryProvider(
		ctx context.Context,
		name string,
		request MemoryProviderLifecycleRequest,
	) (MemoryProviderLifecycleRecord, error)
	CreateMemoryAdhocNote(ctx context.Context, request MemoryAdhocNoteRequest) (MemoryAdhocNoteRecord, error)
	ListAutomationJobs(ctx context.Context, query AutomationJobQuery) ([]JobRecord, error)
	CreateAutomationJob(ctx context.Context, request AutomationJobCreateRequest) (JobRecord, error)
	GetAutomationJob(ctx context.Context, id string) (JobRecord, error)
	UpdateAutomationJob(ctx context.Context, id string, request AutomationJobUpdateRequest) (JobRecord, error)
	DeleteAutomationJob(ctx context.Context, id string) error
	TriggerAutomationJob(ctx context.Context, id string) (RunRecord, error)
	AutomationJobRuns(ctx context.Context, id string, query AutomationRunQuery) ([]RunRecord, error)
	ListAutomationTriggers(ctx context.Context, query AutomationTriggerQuery) ([]TriggerRecord, error)
	CreateAutomationTrigger(ctx context.Context, request AutomationTriggerCreateRequest) (TriggerRecord, error)
	GetAutomationTrigger(ctx context.Context, id string) (TriggerRecord, error)
	UpdateAutomationTrigger(
		ctx context.Context,
		id string,
		request AutomationTriggerUpdateRequest,
	) (TriggerRecord, error)
	DeleteAutomationTrigger(ctx context.Context, id string) error
	AutomationTriggerRuns(ctx context.Context, id string, query AutomationRunQuery) ([]RunRecord, error)
	ListAutomationRuns(ctx context.Context, query AutomationRunQuery) ([]RunRecord, error)
	GetAutomationRun(ctx context.Context, id string) (RunRecord, error)
	ListTasks(ctx context.Context, query TaskListQuery) ([]TaskSummaryRecord, error)
	CreateTask(ctx context.Context, request CreateTaskRequest) (TaskRecord, error)
	GetTask(ctx context.Context, id string) (TaskDetailRecord, error)
	UpdateTask(ctx context.Context, id string, request UpdateTaskRequest) (TaskRecord, error)
	DeleteTask(ctx context.Context, id string) error
	GetTaskExecutionProfile(ctx context.Context, id string) (TaskExecutionProfileRecord, error)
	SetTaskExecutionProfile(
		ctx context.Context,
		id string,
		request *TaskExecutionProfileRequest,
	) (TaskExecutionProfileRecord, error)
	DeleteTaskExecutionProfile(ctx context.Context, id string) error
	CreateTaskBridgeNotificationSubscription(
		ctx context.Context,
		taskID string,
		request *TaskBridgeNotificationSubscriptionRequest,
	) (TaskBridgeNotificationSubscriptionRecord, error)
	ListTaskBridgeNotificationSubscriptions(
		ctx context.Context,
		taskID string,
		query TaskBridgeNotificationSubscriptionQuery,
	) ([]TaskBridgeNotificationSubscriptionRecord, error)
	GetTaskBridgeNotificationSubscription(
		ctx context.Context,
		taskID string,
		subscriptionID string,
	) (TaskBridgeNotificationSubscriptionRecord, error)
	DeleteTaskBridgeNotificationSubscription(ctx context.Context, taskID string, subscriptionID string) error
	RequestTaskRunReview(
		ctx context.Context,
		runID string,
		request *TaskRunReviewRequest,
	) (TaskRunReviewRequestRecord, error)
	ListTaskRunReviews(ctx context.Context, query TaskRunReviewListQuery) ([]TaskRunReviewRecord, error)
	GetTaskRunReview(ctx context.Context, reviewID string) (TaskRunReviewRecord, error)
	SubmitTaskRunReviewVerdict(
		ctx context.Context,
		reviewID string,
		request *TaskRunReviewVerdictRequest,
	) (TaskRunReviewVerdictRecord, error)
	PublishTask(ctx context.Context, id string, request TaskExecutionRequest) (TaskExecutionRecord, error)
	StartTask(ctx context.Context, id string, request TaskExecutionRequest) (TaskExecutionRecord, error)
	ApproveTask(ctx context.Context, id string, request TaskExecutionRequest) (TaskExecutionRecord, error)
	RejectTask(ctx context.Context, id string) (TaskRecord, error)
	CancelTask(ctx context.Context, id string, request CancelTaskRequest) (TaskRecord, error)
	CreateChildTask(ctx context.Context, id string, request CreateTaskChildRequest) (TaskRecord, error)
	AddTaskDependency(ctx context.Context, id string, request AddTaskDependencyRequest) (TaskDetailRecord, error)
	RemoveTaskDependency(ctx context.Context, id string, dependsOnID string) (TaskDetailRecord, error)
	EnqueueTaskRun(ctx context.Context, id string, request EnqueueTaskRunRequest) (TaskRunRecord, error)
	ListTaskRuns(ctx context.Context, id string, query TaskRunListQuery) ([]TaskRunRecord, error)
	ClaimTaskRun(ctx context.Context, id string, request ClaimTaskRunRequest) (TaskRunRecord, error)
	StartTaskRun(ctx context.Context, id string, request StartTaskRunRequest) (TaskRunRecord, error)
	AttachTaskRunSession(ctx context.Context, id string, request AttachTaskRunSessionRequest) (TaskRunRecord, error)
	CompleteTaskRun(ctx context.Context, id string, request CompleteTaskRunRequest) (TaskRunRecord, error)
	FailTaskRun(ctx context.Context, id string, request FailTaskRunRequest) (TaskRunRecord, error)
	CancelTaskRun(ctx context.Context, id string, request CancelTaskRunRequest) (TaskRunRecord, error)
	AgentMe(ctx context.Context, credentials agentidentity.Credentials) (AgentMeRecord, error)
	AgentContext(ctx context.Context, credentials agentidentity.Credentials) (AgentContextRecord, error)
	AgentSpawn(
		ctx context.Context,
		request AgentSpawnRequest,
		credentials agentidentity.Credentials,
	) (AgentSpawnRecord, error)
	AgentChannels(ctx context.Context, credentials agentidentity.Credentials) ([]AgentChannelRecord, error)
	AgentChannelRecv(
		ctx context.Context,
		channel string,
		query AgentChannelRecvQuery,
		credentials agentidentity.Credentials,
	) ([]AgentChannelMessageRecord, error)
	AgentChannelSend(
		ctx context.Context,
		channel string,
		request AgentChannelSendRequest,
		credentials agentidentity.Credentials,
	) (AgentChannelMessageRecord, error)
	AgentChannelReply(
		ctx context.Context,
		request AgentChannelReplyRequest,
		credentials agentidentity.Credentials,
	) (AgentChannelMessageRecord, error)
	AgentTaskClaimNext(
		ctx context.Context,
		request AgentTaskClaimNextRequest,
		credentials agentidentity.Credentials,
	) (AgentTaskNextRecord, error)
	AgentTaskHeartbeat(
		ctx context.Context,
		runID string,
		request AgentTaskHeartbeatRequest,
		credentials agentidentity.Credentials,
	) (AgentTaskLeaseRecord, error)
	AgentTaskComplete(
		ctx context.Context,
		runID string,
		request AgentTaskCompleteRequest,
		credentials agentidentity.Credentials,
	) (AgentTaskLeaseRecord, error)
	AgentTaskFail(
		ctx context.Context,
		runID string,
		request AgentTaskFailRequest,
		credentials agentidentity.Credentials,
	) (AgentTaskLeaseRecord, error)
	AgentTaskRelease(
		ctx context.Context,
		runID string,
		request AgentTaskReleaseRequest,
		credentials agentidentity.Credentials,
	) (AgentTaskLeaseRecord, error)
}

// CreateSessionRequest captures the shared daemon session creation payload.
type CreateSessionRequest = contract.CreateSessionRequest

// SessionListQuery captures the CLI filters for session list queries.
type SessionListQuery struct {
	Workspace string
}

// SessionRecord is the shared daemon session payload.
type SessionRecord = contract.SessionPayload

// SessionSoulRefreshRequest captures the managed session Soul refresh payload.
type SessionSoulRefreshRequest = contract.SessionSoulRefreshRequest

// SessionHealthRecord is the shared session-health payload.
type SessionHealthRecord = contract.SessionHealthPayload

// SessionStatusRecord is the shared compact session status payload.
type SessionStatusRecord = contract.SessionStatusResponse

// SessionInspectRecord is the shared detailed session inspect payload.
type SessionInspectRecord = contract.SessionInspectResponse

// SessionInspectQuery captures optional session inspect expansion fields.
type SessionInspectQuery struct {
	IncludeRecentWakeEvents bool
}

// ACPCapsRecord captures optional runtime capabilities exposed by the daemon API.
type ACPCapsRecord = contract.ACPCapsPayload

// SessionRepairRecord reports one session repair pass returned by the daemon API.
type SessionRepairRecord = contract.SessionRepairPayload

// SessionRepairIssueRecord is one inconsistency reported by session repair.
type SessionRepairIssueRecord = contract.SessionRepairIssuePayload

// SessionRepairActionRecord is one planned or persisted repair action.
type SessionRepairActionRecord = contract.SessionRepairActionPayload

// SessionRepairQuery captures CLI repair modifiers.
type SessionRepairQuery struct {
	DryRun bool
	Force  bool
}

// SessionApprovalRequest captures an interactive permission decision.
type SessionApprovalRequest = contract.ApproveSessionRequest

// SessionApprovalRecord is the shared session approval response payload.
type SessionApprovalRecord = contract.SessionApprovalResponse

// SessionEventRecord is one persisted session event row returned by the daemon API.
type SessionEventRecord = contract.SessionEventPayload

// TurnHistoryRecord groups session events by turn.
type TurnHistoryRecord = contract.TurnHistoryPayload

// SessionEventQuery captures the CLI filters for session event/history queries.
type SessionEventQuery struct {
	Type          string
	AgentName     string
	TurnID        string
	Since         time.Time
	Last          int
	AfterSequence int64
}

// AgentRecord is the shared daemon agent definition payload.
type AgentRecord = contract.AgentPayload

// AgentSoulRecord is the dedicated managed Soul read model.
type AgentSoulRecord = contract.AgentSoulPayload

// AgentSoulValidateRequest captures a managed Soul validation payload.
type AgentSoulValidateRequest = contract.AgentSoulValidateRequest

// AgentSoulPutRequest captures a managed Soul write payload.
type AgentSoulPutRequest = contract.AgentSoulPutRequest

// AgentSoulDeleteRequest captures a managed Soul delete payload.
type AgentSoulDeleteRequest = contract.AgentSoulDeleteRequest

// AgentSoulRollbackRequest captures a managed Soul rollback payload.
type AgentSoulRollbackRequest = contract.AgentSoulRollbackRequest

// AgentSoulHistoryRequest captures managed Soul history filters.
type AgentSoulHistoryRequest = contract.AgentSoulHistoryRequest

// AgentSoulHistoryRecord is the managed Soul history response.
type AgentSoulHistoryRecord = contract.AgentSoulHistoryResponse

// AgentSoulRevisionRecord is one managed Soul authoring revision.
type AgentSoulRevisionRecord = contract.AgentSoulRevisionPayload

// AgentSoulMutationRecord is the managed Soul mutation response.
type AgentSoulMutationRecord = contract.AgentSoulMutationResponse

// AgentHeartbeatRecord is the dedicated managed Heartbeat policy read model.
type AgentHeartbeatRecord = contract.HeartbeatPolicyPayload

// AgentHeartbeatValidateRequest captures a managed Heartbeat validation payload.
type AgentHeartbeatValidateRequest = contract.HeartbeatValidateRequest

// AgentHeartbeatPutRequest captures a managed Heartbeat write payload.
type AgentHeartbeatPutRequest = contract.HeartbeatPutRequest

// AgentHeartbeatDeleteRequest captures a managed Heartbeat delete payload.
type AgentHeartbeatDeleteRequest = contract.HeartbeatDeleteRequest

// AgentHeartbeatRollbackRequest captures a managed Heartbeat rollback payload.
type AgentHeartbeatRollbackRequest = contract.HeartbeatRollbackRequest

// AgentHeartbeatHistoryRequest captures managed Heartbeat history filters.
type AgentHeartbeatHistoryRequest = contract.HeartbeatHistoryRequest

// AgentHeartbeatHistoryRecord is the managed Heartbeat history response.
type AgentHeartbeatHistoryRecord = contract.HeartbeatHistoryResponse

// AgentHeartbeatRevisionRecord is one managed Heartbeat authoring revision.
type AgentHeartbeatRevisionRecord = contract.HeartbeatRevisionPayload

// AgentHeartbeatMutationRecord is the managed Heartbeat mutation response.
type AgentHeartbeatMutationRecord = contract.HeartbeatMutationResponse

// AgentHeartbeatStatusRequest captures Heartbeat status filters.
type AgentHeartbeatStatusRequest = contract.HeartbeatStatusRequest

// AgentHeartbeatStatusRecord is the shared Heartbeat status payload.
type AgentHeartbeatStatusRecord = contract.HeartbeatStatusResponse

// AgentHeartbeatWakeRequest captures one manual Heartbeat wake request.
type AgentHeartbeatWakeRequest = contract.HeartbeatWakeRequest

// AgentHeartbeatWakeDecisionRecord is one manual Heartbeat wake decision payload.
type AgentHeartbeatWakeDecisionRecord = contract.HeartbeatWakeDecisionPayload

// AgentMCPServer is one MCP server entry returned by the daemon API.
type AgentMCPServer = contract.AgentMCPServerJSON

// AgentQuery captures agent definition filters.
type AgentQuery struct {
	Workspace string
}

// SkillRecord is the shared daemon skill payload.
type SkillRecord = contract.SkillPayload

// SkillQuery captures daemon skill filters.
type SkillQuery struct {
	Workspace string
	ForAgent  string
}

// SkillActionRecord is the shared skill enable/disable response payload.
type SkillActionRecord = contract.SkillActionResponse

// WorkspaceCreateRequest captures the shared workspace registration payload.
type WorkspaceCreateRequest = contract.CreateWorkspaceRequest

// WorkspaceUpdateRequest captures mutable workspace fields.
type WorkspaceUpdateRequest = contract.UpdateWorkspaceRequest

// WorkspaceRecord is the shared daemon workspace registration payload.
type WorkspaceRecord = contract.WorkspacePayload

// WorkspaceSkillRecord is one resolved workspace skill returned by the daemon API.
type WorkspaceSkillRecord = contract.WorkspaceSkillPayload

// WorkspaceDetailRecord captures the workspace info payload returned by the daemon API.
type WorkspaceDetailRecord = contract.WorkspaceDetailPayload

// AgentEventRecord is one prompt-stream event returned by the daemon API.
type AgentEventRecord = contract.AgentEventPayload

// TokenUsageRecord is the prompt usage payload returned by the daemon API.
type TokenUsageRecord = contract.TokenUsagePayload

// HookCatalogQuery captures the CLI filters for resolved hook catalog queries.
type HookCatalogQuery = contract.HookCatalogQuery

// HookCatalogRecord is one resolved hook returned by the daemon API.
type HookCatalogRecord = contract.HookCatalogPayload

// HookRunsQuery captures the CLI filters for hook execution history queries.
type HookRunsQuery = contract.HookRunsQuery

// HookRunRecord is one persisted hook execution audit record.
type HookRunRecord = contract.HookRunPayload

// HookEventsQuery captures the CLI filters for hook taxonomy queries.
type HookEventsQuery = contract.HookEventsQuery

// HookEventRecord is one supported hook taxonomy row returned by the daemon API.
type HookEventRecord = contract.HookEventPayload

// ObserveEventRecord is one cross-session observability event row.
type ObserveEventRecord = contract.ObserveEventPayload

// ObserveEventQuery captures the CLI filters for cross-session observability queries.
type ObserveEventQuery struct {
	SessionID string
	AgentName string
	Type      string
	Since     time.Time
	Last      int
}

// MemoryHealthRecord is the shared daemon memory health payload.
type MemoryHealthRecord = contract.MemoryHealthPayload

// MemorySelectorQuery captures scope selectors sent through Memory v2 query parameters.
type MemorySelectorQuery struct {
	Scope         memcontract.Scope
	WorkspaceID   string
	AgentName     string
	AgentTier     memcontract.AgentTier
	IncludeSystem bool
}

// MemoryHistoryQuery captures filters for memory operation history.
type MemoryHistoryQuery struct {
	Scope       memcontract.Scope
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	Operation   string
	Since       time.Time
	Limit       int
}

// MemoryHistoryRecord is one redacted memory operation history row.
type MemoryHistoryRecord = contract.MemoryOperationHistoryPayload

// MemoryDecisionListQuery captures filters for controller decision history.
type MemoryDecisionListQuery struct {
	Scope       memcontract.Scope
	WorkspaceID string
	AgentName   string
	AgentTier   memcontract.AgentTier
	Operation   string
	Since       time.Time
	Reason      string
}

// MemoryListQuery captures filters for Memory v2 list calls.
type MemoryListQuery struct {
	MemorySelectorQuery
	Type            memcontract.Type
	IncludeShadowed bool
}

// MemoryListRecord wraps Memory v2 list output.
type MemoryListRecord = contract.MemoryListResponse

// MemoryEntryRecord wraps one Memory v2 entry.
type MemoryEntryRecord = contract.MemoryEntryResponse

// MemoryCreateRequest captures the daemon API memory create/propose payload.
type MemoryCreateRequest = contract.MemoryCreateRequest

// MemoryEditRequest captures the daemon API memory edit/propose payload.
type MemoryEditRequest = contract.MemoryEditRequest

// MemoryDeleteRecord captures the daemon API memory delete response.
type MemoryDeleteRecord = contract.MemoryDeleteResponse

// MemoryMutationRecord captures the daemon API memory write/edit response.
type MemoryMutationRecord = contract.MemoryMutationDecisionResponse

// MemorySearchRequest captures the daemon API deterministic recall/search payload.
type MemorySearchRequest = contract.MemorySearchRequest

// MemorySearchRecord wraps deterministic recall/search output.
type MemorySearchRecord = contract.MemorySearchResponse

// MemoryReindexRequest captures the daemon API memory reindex payload.
type MemoryReindexRequest = contract.MemoryReindexV2Request

// MemoryReindexRecord captures the daemon API memory reindex response.
type MemoryReindexRecord = contract.MemoryReindexResponse

// MemoryPromoteRequest captures the daemon API memory promotion payload.
type MemoryPromoteRequest = contract.MemoryPromoteRequest

// MemoryPromoteRecord captures the daemon API memory promotion response.
type MemoryPromoteRecord = contract.MemoryPromoteResponse

// MemoryResetRequest captures the daemon API memory reset payload.
type MemoryResetRequest = contract.MemoryResetRequest

// MemoryResetRecord captures the daemon API memory reset response.
type MemoryResetRecord = contract.MemoryResetResponse

// MemoryReloadRecord captures the daemon API memory reload response.
type MemoryReloadRecord = contract.MemoryReloadResponse

// MemoryScopeShowRecord captures effective memory scope resolution.
type MemoryScopeShowRecord = contract.MemoryScopeShowResponse

// MemoryDecisionListRecord wraps controller decision history.
type MemoryDecisionListRecord = contract.MemoryDecisionListResponse

// MemoryDecisionRecord wraps one controller decision.
type MemoryDecisionRecord = contract.MemoryDecisionResponse

// MemoryDecisionRevertRequest captures a decision revert request.
type MemoryDecisionRevertRequest = contract.MemoryDecisionRevertRequest

// MemoryDecisionRevertRecord captures a decision revert response.
type MemoryDecisionRevertRecord = contract.MemoryDecisionRevertResponse

// MemoryRecallTraceRecord captures one redaction-safe recall trace.
type MemoryRecallTraceRecord = contract.MemoryRecallTraceResponse

// MemoryDreamListRecord wraps dreaming runtime records.
type MemoryDreamListRecord = contract.MemoryDreamListResponse

// MemoryDreamRecord wraps one dreaming runtime record.
type MemoryDreamRecord = contract.MemoryDreamResponse

// MemoryDreamTriggerRequest captures a dreaming trigger request.
type MemoryDreamTriggerRequest = contract.MemoryDreamTriggerRequest

// MemoryDreamTriggerRecord captures a dreaming trigger response.
type MemoryDreamTriggerRecord = contract.MemoryDreamTriggerResponse

// MemoryDreamRetryRequest captures a dreaming retry request.
type MemoryDreamRetryRequest = contract.MemoryDreamRetryRequest

// MemoryDreamRetryRecord captures a dreaming retry response.
type MemoryDreamRetryRecord = contract.MemoryDreamRetryResponse

// MemoryDailyLogListRecord wraps daily memory log artifacts.
type MemoryDailyLogListRecord = contract.MemoryDailyLogListResponse

// MemoryExtractorStatusRecord wraps extractor runtime status.
type MemoryExtractorStatusRecord = contract.MemoryExtractorStatusResponse

// MemoryExtractorFailuresRecord wraps extractor DLQ records.
type MemoryExtractorFailuresRecord = contract.MemoryExtractorFailuresResponse

// MemoryExtractorRetryRequest captures an extractor retry request.
type MemoryExtractorRetryRequest = contract.MemoryExtractorRetryRequest

// MemoryExtractorRetryRecord captures extractor retry results.
type MemoryExtractorRetryRecord = contract.MemoryExtractorRetryResponse

// MemoryExtractorDrainRecord captures extractor drain completion.
type MemoryExtractorDrainRecord = contract.MemoryExtractorDrainResponse

// MemoryProviderListRecord wraps registered memory providers.
type MemoryProviderListRecord = contract.MemoryProviderListResponse

// MemoryProviderRecord wraps one memory provider.
type MemoryProviderRecord = contract.MemoryProviderResponse

// MemoryProviderSelectRequest captures active-provider selection.
type MemoryProviderSelectRequest = contract.MemoryProviderSelectRequest

// MemoryProviderLifecycleRequest captures provider lifecycle mutation.
type MemoryProviderLifecycleRequest = contract.MemoryProviderLifecycleRequest

// MemoryProviderLifecycleRecord captures provider lifecycle state after mutation.
type MemoryProviderLifecycleRecord = contract.MemoryProviderLifecycleResponse

// MemoryAdhocNoteRequest captures the ad-hoc memory note write surface.
type MemoryAdhocNoteRequest = contract.MemoryAdhocNoteRequest

// MemoryAdhocNoteRecord captures the created ad-hoc memory note artifact.
type MemoryAdhocNoteRecord = contract.MemoryAdhocNoteResponse

// AutomationJobQuery captures CLI filters for automation job list calls.
type AutomationJobQuery = automationpkg.JobListQuery

// AutomationTriggerQuery captures CLI filters for automation trigger list calls.
type AutomationTriggerQuery = automationpkg.TriggerListQuery

// AutomationRunQuery captures CLI filters for automation run history calls.
type AutomationRunQuery = automationpkg.RunQuery

// AutomationJobCreateRequest captures the shared automation job create payload.
type AutomationJobCreateRequest = contract.CreateJobRequest

// AutomationJobUpdateRequest captures mutable automation job fields.
type AutomationJobUpdateRequest = contract.UpdateJobRequest

// AutomationTriggerCreateRequest captures the shared automation trigger create payload.
type AutomationTriggerCreateRequest = contract.CreateTriggerRequest

// AutomationTriggerUpdateRequest captures mutable automation trigger fields.
type AutomationTriggerUpdateRequest = contract.UpdateTriggerRequest

// JobRecord is the shared automation job payload.
type JobRecord = contract.JobPayload

// TriggerRecord is the shared automation trigger payload.
type TriggerRecord = contract.TriggerPayload

// RunRecord is the shared automation run payload.
type RunRecord = contract.RunPayload

// TaskSummaryRecord is the shared list-oriented task payload.
type TaskSummaryRecord = contract.TaskSummaryPayload

// TaskRecord is the shared single-task payload.
type TaskRecord = contract.TaskPayload

// TaskDetailRecord is the shared expanded task payload.
type TaskDetailRecord = contract.TaskDetailPayload

// TaskDependencyRecord is the shared dependency-edge payload.
type TaskDependencyRecord = contract.TaskDependencyPayload

// TaskRunRecord is the shared task-run payload.
type TaskRunRecord = contract.TaskRunPayload

// TaskExecutionRecord is the shared task execution-boundary payload.
type TaskExecutionRecord = contract.TaskExecutionResponse

// TaskExecutionProfileRecord is the shared task execution profile payload.
type TaskExecutionProfileRecord = contract.TaskExecutionProfilePayload

// TaskExecutionProfileRequest captures a task execution profile replacement.
type TaskExecutionProfileRequest = contract.SetTaskExecutionProfileRequest

// TaskBridgeNotificationSubscriptionRecord is one task terminal bridge
// notification subscription payload.
type TaskBridgeNotificationSubscriptionRecord = contract.TaskBridgeNotificationSubscriptionPayload

// TaskBridgeNotificationSubscriptionRequest captures one task terminal bridge
// notification subscription request.
type TaskBridgeNotificationSubscriptionRequest = contract.CreateTaskBridgeNotificationSubscriptionRequest

// TaskBridgeNotificationSubscriptionQuery captures CLI filters for bridge
// terminal notification subscriptions.
type TaskBridgeNotificationSubscriptionQuery struct {
	BridgeInstanceID string
	Scope            bridgepkg.Scope
	WorkspaceID      string
	Limit            int
}

// TaskRunReviewRecord is the shared task-run review payload.
type TaskRunReviewRecord = contract.TaskRunReviewPayload

// TaskRunReviewRequest captures one task-run review request payload.
type TaskRunReviewRequest = contract.CreateTaskRunReviewRequest

// TaskRunReviewRequestRecord captures one task-run review request result.
type TaskRunReviewRequestRecord = contract.TaskRunReviewRequestResponse

// TaskRunReviewVerdictRequest captures one task-run review verdict payload.
type TaskRunReviewVerdictRequest = contract.SubmitTaskRunReviewVerdictRequest

// TaskRunReviewVerdictRecord captures one task-run review verdict result.
type TaskRunReviewVerdictRecord = contract.TaskRunReviewVerdictResponse

// AgentMeRecord is the shared agent caller identity payload.
type AgentMeRecord = contract.AgentMePayload

// AgentContextRecord is the shared bounded agent situation payload.
type AgentContextRecord = contract.AgentContextPayload

// AgentSpawnRequest captures one bounded child-session spawn request.
type AgentSpawnRequest = contract.AgentSpawnRequest

// AgentSpawnRecord is the stable child-session spawn response projection.
type AgentSpawnRecord = contract.AgentSpawnPayload

// SpawnPermissionPolicyRecord captures concrete spawn permission atoms.
type SpawnPermissionPolicyRecord = contract.SpawnPermissionPolicyPayload

// AgentChannelRecord is one discoverable coordination channel payload.
type AgentChannelRecord = contract.CoordinationChannelPayload

// AgentChannelMessageRecord is one safe agent channel message payload.
type AgentChannelMessageRecord = contract.AgentChannelMessagePayload

// AgentChannelSendRequest captures one agent channel send payload.
type AgentChannelSendRequest = contract.AgentChannelSendRequest

// AgentChannelReplyRequest captures one agent channel reply payload.
type AgentChannelReplyRequest = contract.AgentChannelReplyRequest

// AgentTaskClaimNextRequest captures one agent next-work request.
type AgentTaskClaimNextRequest = contract.AgentTaskClaimNextRequest

// AgentTaskHeartbeatRequest captures one agent lease heartbeat request.
type AgentTaskHeartbeatRequest = contract.AgentTaskHeartbeatRequest

// AgentTaskCompleteRequest captures one agent lease completion request.
type AgentTaskCompleteRequest = contract.AgentTaskCompleteRequest

// AgentTaskFailRequest captures one agent lease failure request.
type AgentTaskFailRequest = contract.AgentTaskFailRequest

// AgentTaskReleaseRequest captures one agent lease release request.
type AgentTaskReleaseRequest = contract.AgentTaskReleaseRequest

// AgentTaskClaimRecord is the synchronous claim payload returned to agents.
type AgentTaskClaimRecord = contract.AgentTaskClaimPayload

// AgentTaskLeaseRecord is the safe lease payload returned by agent lease mutations.
type AgentTaskLeaseRecord = contract.TaskRunLeaseSummaryPayload

// AgentTaskNextRecord is the stable CLI/client wrapper for next-work polling.
type AgentTaskNextRecord struct {
	Claimed bool                  `json:"claimed"`
	Claim   *AgentTaskClaimRecord `json:"claim,omitempty"`
}

// AgentChannelRecvQuery captures receive options for agent channel messages.
type AgentChannelRecvQuery struct {
	Wait  bool
	Limit int
}

// TaskEventRecord is the shared task audit-event payload.
type TaskEventRecord = contract.TaskEventPayload

// TaskListQuery captures CLI filters for task list calls.
type TaskListQuery = contract.TaskListQuery

// TaskRunListQuery captures CLI filters for task-run list calls.
type TaskRunListQuery = contract.TaskRunListQuery

// TaskRunReviewListQuery captures CLI filters for task-run review list calls.
type TaskRunReviewListQuery = contract.TaskRunReviewListQuery

// CreateTaskRequest captures the shared task-create payload.
type CreateTaskRequest = contract.CreateTaskRequest

// CreateTaskChildRequest captures the shared child-task create payload.
type CreateTaskChildRequest = contract.CreateTaskChildRequest

// UpdateTaskRequest captures mutable task fields.
type UpdateTaskRequest = contract.UpdateTaskRequest

// CancelTaskRequest captures the shared task-cancel payload.
type CancelTaskRequest = contract.CancelTaskRequest

// TaskExecutionRequest captures the shared task publish/start/approval payload.
type TaskExecutionRequest = contract.TaskExecutionRequest

// AddTaskDependencyRequest captures the shared dependency-create payload.
type AddTaskDependencyRequest = contract.AddTaskDependencyRequest

// EnqueueTaskRunRequest captures the shared run-enqueue payload.
type EnqueueTaskRunRequest = contract.EnqueueTaskRunRequest

// ClaimTaskRunRequest captures the shared run-claim payload.
type ClaimTaskRunRequest = contract.ClaimTaskRunRequest

// StartTaskRunRequest captures the shared run-start payload.
type StartTaskRunRequest = contract.StartTaskRunRequest

// AttachTaskRunSessionRequest captures the shared run-session attach payload.
type AttachTaskRunSessionRequest = contract.AttachTaskRunSessionRequest

// CompleteTaskRunRequest captures the shared run-complete payload.
type CompleteTaskRunRequest = contract.CompleteTaskRunRequest

// FailTaskRunRequest captures the shared run-fail payload.
type FailTaskRunRequest = contract.FailTaskRunRequest

// CancelTaskRunRequest captures the shared run-cancel payload.
type CancelTaskRunRequest = contract.CancelTaskRunRequest

// HealthStatus is the daemon API observability health payload.
type HealthStatus = contract.ObserveHealthPayload

// DaemonStatus is the shared daemon status payload.
type DaemonStatus = contract.DaemonStatusPayload

// SettingsRestartActionRecord is the shared restart action response payload.
type SettingsRestartActionRecord = contract.RestartActionResponse

// SettingsRestartStatusRecord is the shared restart polling payload.
type SettingsRestartStatusRecord = contract.RestartActionStatus

// SettingsUpdateRecord is the shared settings update status payload.
type SettingsUpdateRecord = contract.SettingsUpdateResponse

// SettingsMutationRecord is the shared settings mutation response payload.
type SettingsMutationRecord = contract.SettingsSkillsMutationResult

// UpdateSettingsSkillsRequest captures the shared skills settings update payload.
type UpdateSettingsSkillsRequest = contract.UpdateSettingsSkillsRequest

// VaultRecord is one redacted vault secret metadata row.
type VaultRecord = contract.VaultSecretPayload

// PutVaultSecretRequest captures a write-only vault secret write payload.
type PutVaultSecretRequest = contract.PutVaultSecretRequest

// VaultListQuery captures CLI filters for vault metadata listing.
type VaultListQuery struct {
	Prefix    string
	Namespace string
}

// NetworkStatusRecord is the shared network status payload.
type NetworkStatusRecord = contract.NetworkStatusPayload

// NetworkKindMetricRecord is one per-kind network metric row.
type NetworkKindMetricRecord = contract.NetworkKindMetricPayload

// NetworkSendRequest captures one outbound network send payload.
type NetworkSendRequest = contract.NetworkSendRequest

// NetworkSendRecord is the shared network send response payload.
type NetworkSendRecord = contract.NetworkSendPayload

// NetworkPeerRecord is the shared visible-peer payload.
type NetworkPeerRecord = contract.NetworkPeerPayload

// NetworkPeerCardRecord is the shared peer-card payload nested under peers.
type NetworkPeerCardRecord = contract.NetworkPeerCardPayload

// NetworkChannelRecord is the shared active-channel payload.
type NetworkChannelRecord = contract.NetworkChannelPayload

// NetworkThreadRecord is the shared public-thread summary payload.
type NetworkThreadRecord = contract.NetworkThreadSummaryPayload

// NetworkDirectRoomRecord is the shared direct-room summary payload.
type NetworkDirectRoomRecord = contract.NetworkDirectRoomPayload

// NetworkConversationMessageRecord is the shared conversation message payload.
type NetworkConversationMessageRecord = contract.NetworkConversationMessagePayload

// NetworkWorkRecord is the shared network work payload.
type NetworkWorkRecord = contract.NetworkWorkPayload

// NetworkDirectResolveRequest captures direct-room resolution inputs.
type NetworkDirectResolveRequest = contract.NetworkDirectResolveRequest

// NetworkEnvelopeRecord is the shared surfaced envelope payload.
type NetworkEnvelopeRecord = contract.NetworkEnvelopePayload

// NetworkPeersQuery captures CLI filters for peer listing.
type NetworkPeersQuery struct {
	Channel string
}

// NetworkThreadsQuery captures CLI filters for public-thread listing.
type NetworkThreadsQuery struct {
	Channel string
	Limit   int
	After   string
}

// NetworkDirectsQuery captures CLI filters for direct-room listing.
type NetworkDirectsQuery struct {
	Channel string
	PeerID  string
	Limit   int
	After   string
}

// NetworkConversationMessagesQuery captures CLI filters for conversation messages.
type NetworkConversationMessagesQuery struct {
	Channel  string
	ThreadID string
	DirectID string
	Limit    int
	Before   string
	After    string
	Kind     string
	WorkID   string
}

// InstallExtensionRequest captures the shared extension install payload.
type InstallExtensionRequest = contract.InstallExtensionRequest

// ExtensionRecord is the shared extension response payload.
type ExtensionRecord = contract.ExtensionPayload

// BundleCatalogRecord is one extension bundle catalog entry.
type BundleCatalogRecord = contract.BundleCatalogPayload

// BundleActivationRecord is one concrete or previewed bundle activation payload.
type BundleActivationRecord = contract.BundleActivationPayload

// BundleNetworkSettingsRecord captures bundle-derived network defaults.
type BundleNetworkSettingsRecord = contract.BundleNetworkSettingsPayload

// BundleChannelRecord is one channel declared by a bundle profile.
type BundleChannelRecord = contract.BundleChannelPayload

// BundleProfileCatalogRecord is one bundle profile catalog summary.
type BundleProfileCatalogRecord = contract.BundleProfileCatalogPayload

// BundleAgentRecord is one agent declared by a bundle profile.
type BundleAgentRecord = contract.BundleAgentPayload

// BundleJobRecord is one automation job declared by a bundle profile.
type BundleJobRecord = contract.BundleJobPayload

// BundleTriggerRecord is one automation trigger declared by a bundle profile.
type BundleTriggerRecord = contract.BundleTriggerPayload

// BundleBridgeRecord is one bridge preset declared by a bundle profile.
type BundleBridgeRecord = contract.BundleBridgePayload

// BundleInventoryRecord is one resource owned by a bundle activation.
type BundleInventoryRecord = contract.BundleInventoryPayload

// DeclaredNetworkChannelRecord is one bundle-declared network channel.
type DeclaredNetworkChannelRecord = contract.DeclaredNetworkChannelPayload

// ActivateBundleRequest captures bundle preview and activation inputs.
type ActivateBundleRequest = contract.ActivateBundleRequest

// UpdateBundleActivationRequest captures mutable bundle activation overlays.
type UpdateBundleActivationRequest = contract.UpdateBundleActivationRequest

// CreateBridgeRequest captures the shared bridge-instance creation payload.
type CreateBridgeRequest = contract.CreateBridgeRequest

// UpdateBridgeRequest captures mutable bridge-instance fields.
type UpdateBridgeRequest = contract.UpdateBridgeRequest

// BridgeTestDeliveryRequest captures the typed bridge delivery-target dry-run request.
type BridgeTestDeliveryRequest = contract.BridgeTestDeliveryRequest

// BridgeDeliveryTargetInput captures the typed bridge delivery-target override input.
type BridgeDeliveryTargetInput = contract.BridgeDeliveryTargetInput

// BridgeRecord is the shared bridge-instance response payload.
type BridgeRecord = bridgepkg.BridgeInstance

// BridgeRouteRecord is one persisted bridge route returned by the daemon API.
type BridgeRouteRecord = bridgepkg.BridgeRoute

// BridgeSecretBindingRequest captures one bridge secret binding write payload.
type BridgeSecretBindingRequest = contract.PutBridgeSecretBindingRequest

// BridgeSecretBindingRecord is one bridge secret binding payload.
type BridgeSecretBindingRecord = bridgepkg.BridgeSecretBinding

// DeliveryTargetRecord is the resolved typed outbound target returned by the daemon API.
type DeliveryTargetRecord = bridgepkg.DeliveryTarget

// BridgeTestDeliveryRecord is the shared dry-run bridge delivery response payload.
type BridgeTestDeliveryRecord = contract.BridgeTestDeliveryResponse

// IdentityRecord is the local agent identity exposed by `agh whoami`.
type IdentityRecord struct {
	SessionID string `json:"session_id,omitempty"`
	Agent     string `json:"agent,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
}

// ResourceRecord is one desired-state resource payload.
type ResourceRecord = contract.ResourceRecordPayload

// ResourcePutRequest captures one desired-state resource upsert.
type ResourcePutRequest = contract.PutResourceRequest

// ResourceDeleteRequest captures one desired-state resource delete request.
type ResourceDeleteRequest = contract.DeleteResourceRequest

// ResourceListQuery captures CLI filters for resource list calls.
type ResourceListQuery struct {
	Kind       resources.ResourceKind
	ScopeKind  resources.ResourceScopeKind
	ScopeID    string
	OwnerKind  resources.ResourceOwnerKind
	OwnerID    string
	SourceKind resources.ResourceSourceKind
	SourceID   string
	Limit      int
}

// ToolApprovalRequest captures one local approval-token mint request.
type ToolApprovalRequest = contract.ToolApprovalRequest

// ToolApprovalRecord is the shared tool approval payload.
type ToolApprovalRecord = contract.ToolApprovalPayload

// SSEEvent is one parsed server-sent event frame.
type SSEEvent = sse.Event
type SSEHandler = sse.Handler

type unixSocketClient struct {
	socketPath   string
	httpClient   *http.Client
	streamClient *http.Client
}

var _ DaemonClient = (*unixSocketClient)(nil)
var _ mcppkg.HostedProxyClient = (*unixSocketClient)(nil)

var errStopSSE = sse.ErrStop

// NewClient constructs a daemon client that talks HTTP over a Unix domain socket.
func NewClient(socketPath string) (DaemonClient, error) {
	path := strings.TrimSpace(socketPath)
	if path == "" {
		return nil, errors.New("cli: daemon socket path is required")
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, "unix", path)
		},
	}

	return &unixSocketClient{
		socketPath:   path,
		httpClient:   &http.Client{Transport: transport, Timeout: defaultUnixSocketClientTimeout},
		streamClient: &http.Client{Transport: transport},
	}, nil
}

func (c *unixSocketClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	var response struct {
		Daemon DaemonStatus `json:"daemon"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/daemon/status", nil, nil, &response); err != nil {
		return DaemonStatus{}, err
	}
	return response.Daemon, nil
}

func (c *unixSocketClient) TriggerSettingsRestart(ctx context.Context) (SettingsRestartActionRecord, error) {
	var response SettingsRestartActionRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/settings/actions/restart", nil, nil, &response); err != nil {
		return SettingsRestartActionRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetSettingsRestartStatus(
	ctx context.Context,
	operationID string,
) (SettingsRestartStatusRecord, error) {
	path := "/api/settings/actions/restart/" + url.PathEscape(strings.TrimSpace(operationID))
	var response SettingsRestartStatusRecord
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return SettingsRestartStatusRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetSettingsUpdate(ctx context.Context) (SettingsUpdateRecord, error) {
	var response SettingsUpdateRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/settings/update", nil, nil, &response); err != nil {
		return SettingsUpdateRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) UpdateSettingsSkills(
	ctx context.Context,
	request UpdateSettingsSkillsRequest,
) (SettingsMutationRecord, error) {
	var response SettingsMutationRecord
	if err := c.doJSON(ctx, http.MethodPatch, "/api/settings/skills", nil, request, &response); err != nil {
		return SettingsMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListVaultSecrets(ctx context.Context, query VaultListQuery) ([]VaultRecord, error) {
	var response struct {
		Secrets []VaultRecord `json:"secrets"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/vault/secrets",
		vaultListValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Secrets, nil
}

func (c *unixSocketClient) GetVaultSecret(ctx context.Context, ref string) (VaultRecord, error) {
	trimmedRef, err := requireVaultRef(ref)
	if err != nil {
		return VaultRecord{}, err
	}
	var response struct {
		Secret VaultRecord `json:"secret"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/vault/secrets/metadata",
		vaultRefValues(trimmedRef),
		nil,
		&response,
	); err != nil {
		return VaultRecord{}, err
	}
	return response.Secret, nil
}

func (c *unixSocketClient) PutVaultSecret(
	ctx context.Context,
	request PutVaultSecretRequest,
) (VaultRecord, error) {
	var response struct {
		Secret VaultRecord `json:"secret"`
	}
	if err := c.doJSON(ctx, http.MethodPut, "/api/vault/secrets", nil, request, &response); err != nil {
		return VaultRecord{}, err
	}
	return response.Secret, nil
}

func (c *unixSocketClient) DeleteVaultSecret(ctx context.Context, ref string) error {
	trimmedRef, err := requireVaultRef(ref)
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, "/api/vault/secrets", vaultRefValues(trimmedRef), nil, nil)
}

func (c *unixSocketClient) NetworkStatus(ctx context.Context) (NetworkStatusRecord, error) {
	var response struct {
		Network NetworkStatusRecord `json:"network"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/network/status", nil, nil, &response); err != nil {
		return NetworkStatusRecord{}, err
	}
	return response.Network, nil
}

func (c *unixSocketClient) NetworkPeers(ctx context.Context, query NetworkPeersQuery) ([]NetworkPeerRecord, error) {
	var response struct {
		Peers []NetworkPeerRecord `json:"peers"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/network/peers",
		networkPeersValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Peers, nil
}

func (c *unixSocketClient) NetworkChannels(ctx context.Context) ([]NetworkChannelRecord, error) {
	var response struct {
		Channels []NetworkChannelRecord `json:"channels"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/network/channels", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Channels, nil
}

func (c *unixSocketClient) NetworkThreads(
	ctx context.Context,
	query NetworkThreadsQuery,
) ([]NetworkThreadRecord, error) {
	channel, err := requireNetworkPathValue("channel", query.Channel)
	if err != nil {
		return nil, err
	}
	var response struct {
		Threads []NetworkThreadRecord `json:"threads"`
	}
	path := "/api/network/channels/" + url.PathEscape(channel) + "/threads"
	if err := c.doJSON(ctx, http.MethodGet, path, networkThreadsValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Threads, nil
}

func (c *unixSocketClient) NetworkThread(
	ctx context.Context,
	channel string,
	threadID string,
) (NetworkThreadRecord, error) {
	path, err := networkThreadPath(channel, threadID)
	if err != nil {
		return NetworkThreadRecord{}, err
	}
	var response struct {
		Thread NetworkThreadRecord `json:"thread"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return NetworkThreadRecord{}, err
	}
	return response.Thread, nil
}

func (c *unixSocketClient) NetworkThreadMessages(
	ctx context.Context,
	query NetworkConversationMessagesQuery,
) ([]NetworkConversationMessageRecord, error) {
	path, err := networkThreadMessagesPath(query.Channel, query.ThreadID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Messages []NetworkConversationMessageRecord `json:"messages"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		path,
		networkConversationMessagesValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

func (c *unixSocketClient) NetworkDirects(
	ctx context.Context,
	query NetworkDirectsQuery,
) ([]NetworkDirectRoomRecord, error) {
	channel, err := requireNetworkPathValue("channel", query.Channel)
	if err != nil {
		return nil, err
	}
	var response struct {
		Directs []NetworkDirectRoomRecord `json:"directs"`
	}
	path := "/api/network/channels/" + url.PathEscape(channel) + "/directs"
	if err := c.doJSON(ctx, http.MethodGet, path, networkDirectsValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Directs, nil
}

func (c *unixSocketClient) NetworkDirectResolve(
	ctx context.Context,
	channel string,
	request NetworkDirectResolveRequest,
) (NetworkDirectRoomRecord, error) {
	channel, err := requireNetworkPathValue("channel", channel)
	if err != nil {
		return NetworkDirectRoomRecord{}, err
	}
	var response struct {
		Direct NetworkDirectRoomRecord `json:"direct"`
	}
	path := "/api/network/channels/" + url.PathEscape(channel) + "/directs/resolve"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return NetworkDirectRoomRecord{}, err
	}
	return response.Direct, nil
}

func (c *unixSocketClient) NetworkDirect(
	ctx context.Context,
	channel string,
	directID string,
) (NetworkDirectRoomRecord, error) {
	path, err := networkDirectPath(channel, directID)
	if err != nil {
		return NetworkDirectRoomRecord{}, err
	}
	var response struct {
		Direct NetworkDirectRoomRecord `json:"direct"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return NetworkDirectRoomRecord{}, err
	}
	return response.Direct, nil
}

func (c *unixSocketClient) NetworkDirectMessages(
	ctx context.Context,
	query NetworkConversationMessagesQuery,
) ([]NetworkConversationMessageRecord, error) {
	path, err := networkDirectMessagesPath(query.Channel, query.DirectID)
	if err != nil {
		return nil, err
	}
	var response struct {
		Messages []NetworkConversationMessageRecord `json:"messages"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		path,
		networkConversationMessagesValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

func (c *unixSocketClient) NetworkWork(ctx context.Context, workID string) (NetworkWorkRecord, error) {
	workID, err := requireNetworkPathValue("work_id", workID)
	if err != nil {
		return NetworkWorkRecord{}, err
	}
	var response struct {
		Work NetworkWorkRecord `json:"work"`
	}
	path := "/api/network/work/" + url.PathEscape(workID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return NetworkWorkRecord{}, err
	}
	return response.Work, nil
}

func (c *unixSocketClient) NetworkSend(ctx context.Context, request NetworkSendRequest) (NetworkSendRecord, error) {
	var response struct {
		Message NetworkSendRecord `json:"message"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/network/send", nil, request, &response); err != nil {
		return NetworkSendRecord{}, err
	}
	return response.Message, nil
}

func (c *unixSocketClient) NetworkInbox(ctx context.Context, sessionID string) ([]NetworkEnvelopeRecord, error) {
	var response struct {
		Messages []NetworkEnvelopeRecord `json:"messages"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/network/inbox",
		networkInboxValues(sessionID),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

func (c *unixSocketClient) ListExtensions(ctx context.Context) ([]ExtensionRecord, error) {
	var response struct {
		Extensions []ExtensionRecord `json:"extensions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/extensions", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Extensions, nil
}

func (c *unixSocketClient) InstallExtension(
	ctx context.Context,
	request InstallExtensionRequest,
) (ExtensionRecord, error) {
	var response struct {
		Extension ExtensionRecord `json:"extension"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/extensions", nil, request, &response); err != nil {
		return ExtensionRecord{}, err
	}
	return response.Extension, nil
}

func (c *unixSocketClient) EnableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	return c.extensionAction(ctx, strings.TrimSpace(name), "enable")
}

func (c *unixSocketClient) DisableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	return c.extensionAction(ctx, strings.TrimSpace(name), "disable")
}

func (c *unixSocketClient) ExtensionStatus(ctx context.Context, name string) (ExtensionRecord, error) {
	var response struct {
		Extension ExtensionRecord `json:"extension"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/extensions/"+url.PathEscape(strings.TrimSpace(name)),
		nil,
		nil,
		&response,
	); err != nil {
		return ExtensionRecord{}, err
	}
	return response.Extension, nil
}

func (c *unixSocketClient) ListBundleCatalog(ctx context.Context) ([]BundleCatalogRecord, error) {
	var response struct {
		Bundles []BundleCatalogRecord `json:"bundles"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/bundles/catalog", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Bundles, nil
}

func (c *unixSocketClient) PreviewBundleActivation(
	ctx context.Context,
	request ActivateBundleRequest,
) (BundleActivationRecord, error) {
	var response struct {
		Activation BundleActivationRecord `json:"activation"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/bundles/preview", nil, request, &response); err != nil {
		return BundleActivationRecord{}, err
	}
	return response.Activation, nil
}

func (c *unixSocketClient) ActivateBundle(
	ctx context.Context,
	request ActivateBundleRequest,
) (BundleActivationRecord, error) {
	var response struct {
		Activation BundleActivationRecord `json:"activation"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/bundles/activations", nil, request, &response); err != nil {
		return BundleActivationRecord{}, err
	}
	return response.Activation, nil
}

func (c *unixSocketClient) ListBundleActivations(ctx context.Context) ([]BundleActivationRecord, error) {
	var response struct {
		Activations []BundleActivationRecord `json:"activations"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/bundles/activations", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Activations, nil
}

func (c *unixSocketClient) GetBundleActivation(ctx context.Context, id string) (BundleActivationRecord, error) {
	var response struct {
		Activation BundleActivationRecord `json:"activation"`
	}
	path := "/api/bundles/activations/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return BundleActivationRecord{}, err
	}
	return response.Activation, nil
}

func (c *unixSocketClient) UpdateBundleActivation(
	ctx context.Context,
	id string,
	request UpdateBundleActivationRequest,
) (BundleActivationRecord, error) {
	var response struct {
		Activation BundleActivationRecord `json:"activation"`
	}
	path := "/api/bundles/activations/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return BundleActivationRecord{}, err
	}
	return response.Activation, nil
}

func (c *unixSocketClient) DeactivateBundle(ctx context.Context, id string) error {
	path := "/api/bundles/activations/" + url.PathEscape(strings.TrimSpace(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) BundleNetworkSettings(ctx context.Context) (BundleNetworkSettingsRecord, error) {
	var response struct {
		Network BundleNetworkSettingsRecord `json:"network"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/bundles/network/settings", nil, nil, &response); err != nil {
		return BundleNetworkSettingsRecord{}, err
	}
	return response.Network, nil
}

func (c *unixSocketClient) ListBridges(ctx context.Context) ([]BridgeRecord, error) {
	var response struct {
		Bridges []BridgeRecord `json:"bridges"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/bridges", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Bridges, nil
}

func (c *unixSocketClient) CreateBridge(ctx context.Context, request CreateBridgeRequest) (BridgeRecord, error) {
	var response struct {
		Bridge BridgeRecord `json:"bridge"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/bridges", nil, request, &response); err != nil {
		return BridgeRecord{}, err
	}
	return response.Bridge, nil
}

func (c *unixSocketClient) GetBridge(ctx context.Context, id string) (BridgeRecord, error) {
	var response struct {
		Bridge BridgeRecord `json:"bridge"`
	}
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return BridgeRecord{}, err
	}
	return response.Bridge, nil
}

func (c *unixSocketClient) UpdateBridge(
	ctx context.Context,
	id string,
	request UpdateBridgeRequest,
) (BridgeRecord, error) {
	var response struct {
		Bridge BridgeRecord `json:"bridge"`
	}
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return BridgeRecord{}, err
	}
	return response.Bridge, nil
}

func (c *unixSocketClient) EnableBridge(ctx context.Context, id string) (BridgeRecord, error) {
	return c.bridgeAction(ctx, strings.TrimSpace(id), "enable")
}

func (c *unixSocketClient) DisableBridge(ctx context.Context, id string) (BridgeRecord, error) {
	return c.bridgeAction(ctx, strings.TrimSpace(id), "disable")
}

func (c *unixSocketClient) RestartBridge(ctx context.Context, id string) (BridgeRecord, error) {
	return c.bridgeAction(ctx, strings.TrimSpace(id), "restart")
}

func (c *unixSocketClient) BridgeRoutes(ctx context.Context, id string) ([]BridgeRouteRecord, error) {
	var response struct {
		Routes []BridgeRouteRecord `json:"routes"`
	}
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id)) + "/routes"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Routes, nil
}

func (c *unixSocketClient) ListBridgeSecretBindings(
	ctx context.Context,
	id string,
) ([]BridgeSecretBindingRecord, error) {
	var response struct {
		Bindings []BridgeSecretBindingRecord `json:"bindings"`
	}
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id)) + "/secret-bindings"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Bindings, nil
}

func (c *unixSocketClient) PutBridgeSecretBinding(
	ctx context.Context,
	id string,
	bindingName string,
	request BridgeSecretBindingRequest,
) (BridgeSecretBindingRecord, error) {
	var response struct {
		Binding BridgeSecretBindingRecord `json:"binding"`
	}
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id)) +
		"/secret-bindings/" + url.PathEscape(strings.TrimSpace(bindingName))
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return BridgeSecretBindingRecord{}, err
	}
	return response.Binding, nil
}

func (c *unixSocketClient) DeleteBridgeSecretBinding(ctx context.Context, id string, bindingName string) error {
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id)) +
		"/secret-bindings/" + url.PathEscape(strings.TrimSpace(bindingName))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) TestBridgeDelivery(
	ctx context.Context,
	id string,
	request BridgeTestDeliveryRequest,
) (BridgeTestDeliveryRecord, error) {
	var response BridgeTestDeliveryRecord
	path := "/api/bridges/" + url.PathEscape(strings.TrimSpace(id)) + "/test-delivery"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return BridgeTestDeliveryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListSessions(ctx context.Context, query SessionListQuery) ([]SessionRecord, error) {
	var response struct {
		Sessions []SessionRecord `json:"sessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/sessions", sessionListValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Sessions, nil
}

func (c *unixSocketClient) CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/sessions", nil, request, &response); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) GetSession(ctx context.Context, id string) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id)),
		nil,
		nil,
		&response,
	); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) GetSessionHealth(ctx context.Context, id string) (SessionHealthRecord, error) {
	var response struct {
		Health SessionHealthRecord `json:"health"`
	}
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/health"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return SessionHealthRecord{}, err
	}
	return response.Health, nil
}

func (c *unixSocketClient) GetSessionStatus(ctx context.Context, id string) (SessionStatusRecord, error) {
	var response SessionStatusRecord
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/status"
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return SessionStatusRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) InspectSession(
	ctx context.Context,
	id string,
	query SessionInspectQuery,
) (SessionInspectRecord, error) {
	var response SessionInspectRecord
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/inspect"
	if err := c.doJSON(ctx, http.MethodGet, path, sessionInspectValues(query), nil, &response); err != nil {
		return SessionInspectRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RefreshSessionSoul(
	ctx context.Context,
	id string,
	request SessionSoulRefreshRequest,
) (AgentSoulRecord, error) {
	var response AgentSoulRecord
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/soul/refresh"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentSoulRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) StopSession(ctx context.Context, id string) error {
	return c.doJSON(
		ctx,
		http.MethodPost,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/stop",
		nil,
		nil,
		nil,
	)
}

func (c *unixSocketClient) ResumeSession(ctx context.Context, id string) (SessionRecord, error) {
	var response struct {
		Session SessionRecord `json:"session"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/resume",
		nil,
		nil,
		&response,
	); err != nil {
		return SessionRecord{}, err
	}
	return response.Session, nil
}

func (c *unixSocketClient) RepairSession(
	ctx context.Context,
	id string,
	query SessionRepairQuery,
) (SessionRepairRecord, error) {
	var response struct {
		Repair SessionRepairRecord `json:"repair"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/repair",
		sessionRepairValues(query),
		nil,
		&response,
	); err != nil {
		return SessionRepairRecord{}, err
	}
	return response.Repair, nil
}

func (c *unixSocketClient) ApproveSession(
	ctx context.Context,
	id string,
	request SessionApprovalRequest,
) (SessionApprovalRecord, error) {
	var response SessionApprovalRecord
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/approve"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return SessionApprovalRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error) {
	var events []AgentEventRecord
	query := url.Values{}
	query.Set("format", "raw")
	err := c.doSSE(
		ctx,
		http.MethodPost,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/prompt",
		query,
		map[string]string{"message": message},
		"",
		func(event SSEEvent) error {
			var payload AgentEventRecord
			if len(event.Data) > 0 {
				if err := json.Unmarshal(event.Data, &payload); err != nil {
					return fmt.Errorf("cli: decode prompt event: %w", err)
				}
			}
			if payload.Type == "" {
				payload.Type = event.Event
			}
			events = append(events, payload)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (c *unixSocketClient) StreamPromptSession(
	ctx context.Context,
	id string,
	message string,
	handler SSEHandler,
) error {
	return c.doSSE(
		ctx,
		http.MethodPost,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/prompt",
		nil,
		map[string]string{"message": message},
		"",
		handler,
	)
}

func (c *unixSocketClient) SessionEvents(
	ctx context.Context,
	id string,
	query SessionEventQuery,
) ([]SessionEventRecord, error) {
	var response struct {
		Events []SessionEventRecord `json:"events"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/events",
		sessionEventValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *unixSocketClient) StreamSessionEvents(
	ctx context.Context,
	id string,
	query SessionEventQuery,
	lastEventID string,
	handler SSEHandler,
) error {
	return c.doSSE(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/stream",
		sessionEventValues(query),
		nil,
		lastEventID,
		handler,
	)
}

func (c *unixSocketClient) SessionHistory(
	ctx context.Context,
	id string,
	query SessionEventQuery,
) ([]TurnHistoryRecord, error) {
	var response struct {
		History []TurnHistoryRecord `json:"history"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/sessions/"+url.PathEscape(strings.TrimSpace(id))+"/history",
		sessionEventValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.History, nil
}

func (c *unixSocketClient) BindHostedMCP(
	ctx context.Context,
	request mcppkg.HostedBindRequest,
) (mcppkg.HostedBindResponse, error) {
	var response mcppkg.HostedBindResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/internal/hosted-mcp/bind", nil, request, &response); err != nil {
		return mcppkg.HostedBindResponse{}, err
	}
	return response, nil
}

func (c *unixSocketClient) HostedMCPProjection(
	ctx context.Context,
	bindID string,
) (mcppkg.HostedProjectionResponse, error) {
	query := url.Values{}
	query.Set("bind_id", strings.TrimSpace(bindID))
	var response mcppkg.HostedProjectionResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/internal/hosted-mcp/projection", query, nil, &response); err != nil {
		return mcppkg.HostedProjectionResponse{}, err
	}
	return response, nil
}

func (c *unixSocketClient) StreamHostedMCPProjection(
	ctx context.Context,
	bindID string,
	lastDigest string,
	handler mcppkg.HostedProjectionHandler,
) error {
	query := url.Values{}
	query.Set("bind_id", strings.TrimSpace(bindID))
	if trimmed := strings.TrimSpace(lastDigest); trimmed != "" {
		query.Set("last_digest", trimmed)
	}
	return c.doSSE(
		ctx,
		http.MethodGet,
		"/api/internal/hosted-mcp/projection/stream",
		query,
		nil,
		"",
		func(event SSEEvent) error {
			if event.Event == "error" {
				return readAPIErrorBody(0, "", event.Data)
			}
			if event.Event != "projection" {
				return nil
			}
			var snapshot mcppkg.HostedProjectionResponse
			if err := json.Unmarshal(event.Data, &snapshot); err != nil {
				return fmt.Errorf("cli: decode hosted MCP projection event: %w", err)
			}
			if handler == nil {
				return nil
			}
			return handler(snapshot)
		},
	)
}

func (c *unixSocketClient) CallHostedMCP(
	ctx context.Context,
	request mcppkg.HostedCallRequest,
) (mcppkg.HostedCallResponse, error) {
	var response mcppkg.HostedCallResponse
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/internal/hosted-mcp/tools/call",
		nil,
		request,
		&response,
	); err != nil {
		return mcppkg.HostedCallResponse{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ReleaseHostedMCP(
	ctx context.Context,
	request mcppkg.HostedReleaseRequest,
) error {
	return c.doJSON(ctx, http.MethodPost, "/api/internal/hosted-mcp/release", nil, request, nil)
}

func (c *unixSocketClient) CreateWorkspace(
	ctx context.Context,
	request WorkspaceCreateRequest,
) (WorkspaceRecord, error) {
	var response struct {
		Workspace WorkspaceRecord `json:"workspace"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/workspaces", nil, request, &response); err != nil {
		return WorkspaceRecord{}, err
	}
	return response.Workspace, nil
}

func (c *unixSocketClient) ListWorkspaces(ctx context.Context) ([]WorkspaceRecord, error) {
	var response struct {
		Workspaces []WorkspaceRecord `json:"workspaces"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/workspaces", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Workspaces, nil
}

func (c *unixSocketClient) GetWorkspace(ctx context.Context, ref string) (WorkspaceDetailRecord, error) {
	routeRef, err := c.workspaceRouteRef(ctx, ref)
	if err != nil {
		return WorkspaceDetailRecord{}, err
	}
	var response WorkspaceDetailRecord
	path := "/api/workspaces/" + url.PathEscape(routeRef)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return WorkspaceDetailRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) workspaceRouteRef(ctx context.Context, ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", errors.New("cli: workspace reference is required")
	}
	if !workspaceRefLooksLikePath(trimmed) {
		return trimmed, nil
	}
	target, err := canonicalCLIWorkspacePath(trimmed)
	if err != nil {
		return "", err
	}
	workspaces, err := c.ListWorkspaces(ctx)
	if err != nil {
		return "", fmt.Errorf("cli: list workspaces before resolving path %q: %w", target, err)
	}
	for _, workspace := range workspaces {
		root, err := canonicalCLIWorkspacePath(workspace.RootDir)
		if err != nil {
			continue
		}
		if root == target {
			return workspace.ID, nil
		}
	}
	return "", fmt.Errorf("cli: workspace path %q is not registered", target)
}

func workspaceRefLooksLikePath(ref string) bool {
	return filepath.IsAbs(ref) ||
		strings.HasPrefix(ref, ".") ||
		strings.Contains(ref, string(filepath.Separator))
}

func canonicalCLIWorkspacePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("cli: workspace path is required")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("cli: resolve workspace path %q: %w", path, err)
	}
	cleaned := filepath.Clean(abs)
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err == nil {
		return resolved, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return cleaned, nil
	}
	return "", fmt.Errorf("cli: resolve workspace path %q: %w", cleaned, err)
}

func (c *unixSocketClient) UpdateWorkspace(
	ctx context.Context,
	ref string,
	request WorkspaceUpdateRequest,
) (WorkspaceRecord, error) {
	var response struct {
		Workspace WorkspaceRecord `json:"workspace"`
	}
	path := "/api/workspaces/" + url.PathEscape(strings.TrimSpace(ref))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return WorkspaceRecord{}, err
	}
	return response.Workspace, nil
}

func (c *unixSocketClient) DeleteWorkspace(ctx context.Context, ref string) error {
	path := "/api/workspaces/" + url.PathEscape(strings.TrimSpace(ref))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) ListAgents(ctx context.Context, query AgentQuery) ([]AgentRecord, error) {
	var response struct {
		Agents []AgentRecord `json:"agents"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/agents", agentValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Agents, nil
}

func (c *unixSocketClient) GetAgent(ctx context.Context, name string, query AgentQuery) (AgentRecord, error) {
	var response struct {
		Agent AgentRecord `json:"agent"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/agents/"+url.PathEscape(strings.TrimSpace(name)),
		agentValues(query),
		nil,
		&response,
	); err != nil {
		return AgentRecord{}, err
	}
	return response.Agent, nil
}

func (c *unixSocketClient) GetAgentSoul(
	ctx context.Context,
	name string,
	query AgentQuery,
) (AgentSoulRecord, error) {
	var response AgentSoulRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul"
	if err := c.doJSON(ctx, http.MethodGet, path, agentValues(query), nil, &response); err != nil {
		return AgentSoulRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ValidateAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulValidateRequest,
) (AgentSoulRecord, error) {
	var response AgentSoulRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul/validate"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentSoulRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PutAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulPutRequest,
) (AgentSoulMutationRecord, error) {
	var response AgentSoulMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul"
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return AgentSoulMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DeleteAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulDeleteRequest,
) (AgentSoulMutationRecord, error) {
	var response AgentSoulMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul"
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, request, &response); err != nil {
		return AgentSoulMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListAgentSoulHistory(
	ctx context.Context,
	name string,
	request AgentSoulHistoryRequest,
) (AgentSoulHistoryRecord, error) {
	var response AgentSoulHistoryRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul/history"
	if err := c.doJSON(ctx, http.MethodGet, path, agentSoulHistoryValues(request), nil, &response); err != nil {
		return AgentSoulHistoryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RollbackAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulRollbackRequest,
) (AgentSoulMutationRecord, error) {
	var response AgentSoulMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/soul/rollback"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentSoulMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetAgentHeartbeat(
	ctx context.Context,
	name string,
	query AgentQuery,
) (AgentHeartbeatRecord, error) {
	var response AgentHeartbeatRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat"
	if err := c.doJSON(ctx, http.MethodGet, path, agentValues(query), nil, &response); err != nil {
		return AgentHeartbeatRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ValidateAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatValidateRequest,
) (AgentHeartbeatRecord, error) {
	var response AgentHeartbeatRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat/validate"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentHeartbeatRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PutAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatPutRequest,
) (AgentHeartbeatMutationRecord, error) {
	var response AgentHeartbeatMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat"
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return AgentHeartbeatMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DeleteAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatDeleteRequest,
) (AgentHeartbeatMutationRecord, error) {
	var response AgentHeartbeatMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat"
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, request, &response); err != nil {
		return AgentHeartbeatMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListAgentHeartbeatHistory(
	ctx context.Context,
	name string,
	request AgentHeartbeatHistoryRequest,
) (AgentHeartbeatHistoryRecord, error) {
	var response AgentHeartbeatHistoryRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat/history"
	if err := c.doJSON(ctx, http.MethodGet, path, agentHeartbeatHistoryValues(request), nil, &response); err != nil {
		return AgentHeartbeatHistoryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RollbackAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatRollbackRequest,
) (AgentHeartbeatMutationRecord, error) {
	var response AgentHeartbeatMutationRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat/rollback"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentHeartbeatMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetAgentHeartbeatStatus(
	ctx context.Context,
	name string,
	request AgentHeartbeatStatusRequest,
) (AgentHeartbeatStatusRecord, error) {
	var response AgentHeartbeatStatusRecord
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat/status"
	if err := c.doJSON(ctx, http.MethodGet, path, agentHeartbeatStatusValues(request), nil, &response); err != nil {
		return AgentHeartbeatStatusRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) WakeAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatWakeRequest,
) (AgentHeartbeatWakeDecisionRecord, error) {
	var response contract.HeartbeatWakeResponse
	path := "/api/agents/" + url.PathEscape(strings.TrimSpace(name)) + "/heartbeat/wake"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return AgentHeartbeatWakeDecisionRecord{}, err
	}
	return response.Decision, nil
}

func (c *unixSocketClient) ListResources(ctx context.Context, query ResourceListQuery) ([]ResourceRecord, error) {
	var response struct {
		Records []ResourceRecord `json:"records"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/resources", resourceListValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Records, nil
}

func (c *unixSocketClient) GetResource(ctx context.Context, kind string, id string) (ResourceRecord, error) {
	var response struct {
		Record ResourceRecord `json:"record"`
	}
	path := "/api/resources/" + url.PathEscape(strings.TrimSpace(kind)) + "/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return ResourceRecord{}, err
	}
	return response.Record, nil
}

func (c *unixSocketClient) PutResource(
	ctx context.Context,
	kind string,
	id string,
	request ResourcePutRequest,
) (ResourceRecord, error) {
	var response struct {
		Record ResourceRecord `json:"record"`
	}
	path := "/api/resources/" + url.PathEscape(strings.TrimSpace(kind)) + "/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return ResourceRecord{}, err
	}
	return response.Record, nil
}

func (c *unixSocketClient) DeleteResource(
	ctx context.Context,
	kind string,
	id string,
	request ResourceDeleteRequest,
) error {
	path := "/api/resources/" + url.PathEscape(strings.TrimSpace(kind)) + "/" + url.PathEscape(strings.TrimSpace(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, request, nil)
}

func (c *unixSocketClient) ListSkills(ctx context.Context, query SkillQuery) ([]SkillRecord, error) {
	var response struct {
		Skills []SkillRecord `json:"skills"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/skills", skillValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Skills, nil
}

func (c *unixSocketClient) GetSkill(ctx context.Context, name string, query SkillQuery) (SkillRecord, error) {
	var response struct {
		Skill SkillRecord `json:"skill"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/skills/"+url.PathEscape(strings.TrimSpace(name)),
		skillValues(query),
		nil,
		&response,
	); err != nil {
		return SkillRecord{}, err
	}
	return response.Skill, nil
}

func (c *unixSocketClient) GetSkillContent(ctx context.Context, name string, query SkillQuery) (string, error) {
	var response struct {
		Content string `json:"content"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/skills/"+url.PathEscape(strings.TrimSpace(name))+"/content",
		skillValues(query),
		nil,
		&response,
	); err != nil {
		return "", err
	}
	return response.Content, nil
}

func (c *unixSocketClient) EnableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error) {
	return c.skillAction(ctx, strings.TrimSpace(name), "enable", query)
}

func (c *unixSocketClient) DisableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error) {
	return c.skillAction(ctx, strings.TrimSpace(name), "disable", query)
}

func (c *unixSocketClient) HookCatalog(ctx context.Context, query HookCatalogQuery) ([]HookCatalogRecord, error) {
	var response struct {
		Hooks []HookCatalogRecord `json:"hooks"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/hooks/catalog",
		hookCatalogValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Hooks, nil
}

func (c *unixSocketClient) HookRuns(ctx context.Context, query HookRunsQuery) ([]HookRunRecord, error) {
	var response struct {
		Runs []HookRunRecord `json:"runs"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/hooks/runs", hookRunsValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *unixSocketClient) HookEvents(ctx context.Context, query HookEventsQuery) ([]HookEventRecord, error) {
	var response struct {
		Events []HookEventRecord `json:"events"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/hooks/events", hookEventsValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *unixSocketClient) ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error) {
	var response struct {
		Events []ObserveEventRecord `json:"events"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/observe/events",
		observeEventValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (c *unixSocketClient) StreamObserveEvents(
	ctx context.Context,
	query ObserveEventQuery,
	lastEventID string,
	handler SSEHandler,
) error {
	return c.doSSE(
		ctx,
		http.MethodGet,
		"/api/observe/events/stream",
		observeEventValues(query),
		nil,
		lastEventID,
		handler,
	)
}

func (c *unixSocketClient) ObserveHealth(ctx context.Context) (HealthStatus, error) {
	var response struct {
		Health HealthStatus `json:"health"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/observe/health", nil, nil, &response); err != nil {
		return HealthStatus{}, err
	}
	return response.Health, nil
}

func (c *unixSocketClient) MemoryHealth(ctx context.Context, workspace string) (MemoryHealthRecord, error) {
	var response MemoryHealthRecord
	values := url.Values{}
	if trimmed := strings.TrimSpace(workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/health", values, nil, &response); err != nil {
		return MemoryHealthRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) MemoryHistory(
	ctx context.Context,
	query MemoryHistoryQuery,
) ([]MemoryHistoryRecord, error) {
	var response contract.MemoryOperationHistoryResponse
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/history",
		memoryHistoryValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Operations, nil
}

func (c *unixSocketClient) ListMemory(
	ctx context.Context,
	query MemoryListQuery,
) (MemoryListRecord, error) {
	var response MemoryListRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory", memoryListValues(query), nil, &response); err != nil {
		return MemoryListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ShowMemory(
	ctx context.Context,
	filename string,
	query MemorySelectorQuery,
) (MemoryEntryRecord, error) {
	var response MemoryEntryRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/"+url.PathEscape(strings.TrimSpace(filename)),
		memorySelectorValues(query),
		nil,
		&response,
	); err != nil {
		return MemoryEntryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) CreateMemory(
	ctx context.Context,
	request MemoryCreateRequest,
) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory", nil, request, &response); err != nil {
		return MemoryMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) EditMemory(
	ctx context.Context,
	filename string,
	request MemoryEditRequest,
) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(
		ctx,
		http.MethodPatch,
		"/api/memory/"+url.PathEscape(strings.TrimSpace(filename)),
		nil,
		request,
		&response,
	); err != nil {
		return MemoryMutationRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DeleteMemory(
	ctx context.Context,
	filename string,
	query MemorySelectorQuery,
) (MemoryDeleteRecord, error) {
	var response MemoryDeleteRecord
	if err := c.doJSON(
		ctx,
		http.MethodDelete,
		"/api/memory/"+url.PathEscape(strings.TrimSpace(filename)),
		memorySelectorValues(query),
		nil,
		&response,
	); err != nil {
		return MemoryDeleteRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) SearchMemory(
	ctx context.Context,
	request MemorySearchRequest,
) (MemorySearchRecord, error) {
	var response MemorySearchRecord
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/memory/search",
		nil,
		request,
		&response,
	); err != nil {
		return MemorySearchRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ReindexMemory(
	ctx context.Context,
	request MemoryReindexRequest,
) (MemoryReindexRecord, error) {
	var response MemoryReindexRecord
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/memory/reindex",
		nil,
		request,
		&response,
	); err != nil {
		return MemoryReindexRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PromoteMemory(
	ctx context.Context,
	request MemoryPromoteRequest,
) (MemoryPromoteRecord, error) {
	var response MemoryPromoteRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/promote", nil, request, &response); err != nil {
		return MemoryPromoteRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ResetMemory(ctx context.Context, request MemoryResetRequest) (MemoryResetRecord, error) {
	var response MemoryResetRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/reset", nil, request, &response); err != nil {
		return MemoryResetRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ReloadMemory(ctx context.Context, request MemorySelectorQuery) (MemoryReloadRecord, error) {
	var response MemoryReloadRecord
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/memory/reload",
		memorySelectorValues(request),
		nil,
		&response,
	); err != nil {
		return MemoryReloadRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) MemoryScopeShow(
	ctx context.Context,
	query MemorySelectorQuery,
) (MemoryScopeShowRecord, error) {
	var response MemoryScopeShowRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/scope-show",
		memorySelectorValues(query),
		nil,
		&response,
	); err != nil {
		return MemoryScopeShowRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListMemoryDecisions(
	ctx context.Context,
	query MemoryDecisionListQuery,
) (MemoryDecisionListRecord, error) {
	var response MemoryDecisionListRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/decisions",
		memoryDecisionValues(query),
		nil,
		&response,
	); err != nil {
		return MemoryDecisionListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryDecision(ctx context.Context, id string) (MemoryDecisionRecord, error) {
	var response MemoryDecisionRecord
	path := "/api/memory/decisions/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return MemoryDecisionRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RevertMemoryDecision(
	ctx context.Context,
	id string,
	request MemoryDecisionRevertRequest,
) (MemoryDecisionRevertRecord, error) {
	var response MemoryDecisionRevertRecord
	path := "/api/memory/decisions/" + url.PathEscape(strings.TrimSpace(id)) + "/revert"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return MemoryDecisionRevertRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryRecallTrace(
	ctx context.Context,
	sessionID string,
	turnSeq int64,
) (MemoryRecallTraceRecord, error) {
	var response MemoryRecallTraceRecord
	path := fmt.Sprintf(
		"/api/memory/recall-traces/%s/%d",
		url.PathEscape(strings.TrimSpace(sessionID)),
		turnSeq,
	)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return MemoryRecallTraceRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListMemoryDreams(ctx context.Context) (MemoryDreamListRecord, error) {
	var response MemoryDreamListRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/dreams", nil, nil, &response); err != nil {
		return MemoryDreamListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryDream(ctx context.Context, id string) (MemoryDreamRecord, error) {
	var response MemoryDreamRecord
	path := "/api/memory/dreams/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return MemoryDreamRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) TriggerMemoryDream(
	ctx context.Context,
	request MemoryDreamTriggerRequest,
) (MemoryDreamTriggerRecord, error) {
	var response MemoryDreamTriggerRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/dreams/trigger", nil, request, &response); err != nil {
		return MemoryDreamTriggerRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RetryMemoryDream(
	ctx context.Context,
	id string,
	request MemoryDreamRetryRequest,
) (MemoryDreamRetryRecord, error) {
	var response MemoryDreamRetryRecord
	path := "/api/memory/dreams/" + url.PathEscape(strings.TrimSpace(id)) + "/retry"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return MemoryDreamRetryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryDreamStatus(ctx context.Context) (MemoryDreamListRecord, error) {
	var response MemoryDreamListRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/dreams/status", nil, nil, &response); err != nil {
		return MemoryDreamListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListMemoryDailyLogs(
	ctx context.Context,
	query MemorySelectorQuery,
) (MemoryDailyLogListRecord, error) {
	var response MemoryDailyLogListRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/daily",
		memorySelectorValues(query),
		nil,
		&response,
	); err != nil {
		return MemoryDailyLogListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryExtractorStatus(
	ctx context.Context,
	sessionID string,
) (MemoryExtractorStatusRecord, error) {
	var response MemoryExtractorStatusRecord
	values := url.Values{}
	if trimmed := strings.TrimSpace(sessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/extractor/status", values, nil, &response); err != nil {
		return MemoryExtractorStatusRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListMemoryExtractorFailures(ctx context.Context) (MemoryExtractorFailuresRecord, error) {
	var response MemoryExtractorFailuresRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/extractor/failures", nil, nil, &response); err != nil {
		return MemoryExtractorFailuresRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) RetryMemoryExtractor(
	ctx context.Context,
	request MemoryExtractorRetryRequest,
) (MemoryExtractorRetryRecord, error) {
	var response MemoryExtractorRetryRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/extractor/retry", nil, request, &response); err != nil {
		return MemoryExtractorRetryRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DrainMemoryExtractor(ctx context.Context) (MemoryExtractorDrainRecord, error) {
	var response MemoryExtractorDrainRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/extractor/drain", nil, nil, &response); err != nil {
		return MemoryExtractorDrainRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListMemoryProviders(ctx context.Context) (MemoryProviderListRecord, error) {
	var response MemoryProviderListRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory/providers", nil, nil, &response); err != nil {
		return MemoryProviderListRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) GetMemoryProvider(ctx context.Context, name string) (MemoryProviderRecord, error) {
	var response MemoryProviderRecord
	path := "/api/memory/providers/" + url.PathEscape(strings.TrimSpace(name))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return MemoryProviderRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) SelectMemoryProvider(
	ctx context.Context,
	request MemoryProviderSelectRequest,
) (MemoryProviderLifecycleRecord, error) {
	var response MemoryProviderLifecycleRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/providers/select", nil, request, &response); err != nil {
		return MemoryProviderLifecycleRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) EnableMemoryProvider(
	ctx context.Context,
	name string,
	request MemoryProviderLifecycleRequest,
) (MemoryProviderLifecycleRecord, error) {
	var response MemoryProviderLifecycleRecord
	path := "/api/memory/providers/" + url.PathEscape(strings.TrimSpace(name)) + "/enable"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return MemoryProviderLifecycleRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) DisableMemoryProvider(
	ctx context.Context,
	name string,
	request MemoryProviderLifecycleRequest,
) (MemoryProviderLifecycleRecord, error) {
	var response MemoryProviderLifecycleRecord
	path := "/api/memory/providers/" + url.PathEscape(strings.TrimSpace(name)) + "/disable"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return MemoryProviderLifecycleRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) CreateMemoryAdhocNote(
	ctx context.Context,
	request MemoryAdhocNoteRequest,
) (MemoryAdhocNoteRecord, error) {
	var response MemoryAdhocNoteRecord
	if err := c.doJSON(ctx, http.MethodPost, "/api/memory/ad-hoc", nil, request, &response); err != nil {
		return MemoryAdhocNoteRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListAutomationJobs(ctx context.Context, query AutomationJobQuery) ([]JobRecord, error) {
	var response contract.JobsResponse
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/automation/jobs",
		automationJobValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Jobs, nil
}

func (c *unixSocketClient) CreateAutomationJob(
	ctx context.Context,
	request AutomationJobCreateRequest,
) (JobRecord, error) {
	var response contract.JobResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/automation/jobs", nil, request, &response); err != nil {
		return JobRecord{}, err
	}
	return response.Job, nil
}

func (c *unixSocketClient) GetAutomationJob(ctx context.Context, id string) (JobRecord, error) {
	var response contract.JobResponse
	path := "/api/automation/jobs/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return JobRecord{}, err
	}
	return response.Job, nil
}

func (c *unixSocketClient) UpdateAutomationJob(
	ctx context.Context,
	id string,
	request AutomationJobUpdateRequest,
) (JobRecord, error) {
	var response contract.JobResponse
	path := "/api/automation/jobs/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return JobRecord{}, err
	}
	return response.Job, nil
}

func (c *unixSocketClient) DeleteAutomationJob(ctx context.Context, id string) error {
	path := "/api/automation/jobs/" + url.PathEscape(strings.TrimSpace(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) TriggerAutomationJob(ctx context.Context, id string) (RunRecord, error) {
	var response contract.RunResponse
	path := "/api/automation/jobs/" + url.PathEscape(strings.TrimSpace(id)) + "/trigger"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return RunRecord{}, err
	}
	return response.Run, nil
}

func (c *unixSocketClient) AutomationJobRuns(
	ctx context.Context,
	id string,
	query AutomationRunQuery,
) ([]RunRecord, error) {
	var response contract.RunsResponse
	path := "/api/automation/jobs/" + url.PathEscape(strings.TrimSpace(id)) + "/runs"
	if err := c.doJSON(ctx, http.MethodGet, path, automationRunValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *unixSocketClient) ListAutomationTriggers(
	ctx context.Context,
	query AutomationTriggerQuery,
) ([]TriggerRecord, error) {
	var response contract.TriggersResponse
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/automation/triggers",
		automationTriggerValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Triggers, nil
}

func (c *unixSocketClient) CreateAutomationTrigger(
	ctx context.Context,
	request AutomationTriggerCreateRequest,
) (TriggerRecord, error) {
	var response contract.TriggerResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/automation/triggers", nil, request, &response); err != nil {
		return TriggerRecord{}, err
	}
	return response.Trigger, nil
}

func (c *unixSocketClient) GetAutomationTrigger(ctx context.Context, id string) (TriggerRecord, error) {
	var response contract.TriggerResponse
	path := "/api/automation/triggers/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return TriggerRecord{}, err
	}
	return response.Trigger, nil
}

func (c *unixSocketClient) UpdateAutomationTrigger(
	ctx context.Context,
	id string,
	request AutomationTriggerUpdateRequest,
) (TriggerRecord, error) {
	var response contract.TriggerResponse
	path := "/api/automation/triggers/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return TriggerRecord{}, err
	}
	return response.Trigger, nil
}

func (c *unixSocketClient) DeleteAutomationTrigger(ctx context.Context, id string) error {
	path := "/api/automation/triggers/" + url.PathEscape(strings.TrimSpace(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) AutomationTriggerRuns(
	ctx context.Context,
	id string,
	query AutomationRunQuery,
) ([]RunRecord, error) {
	var response contract.RunsResponse
	path := "/api/automation/triggers/" + url.PathEscape(strings.TrimSpace(id)) + "/runs"
	if err := c.doJSON(ctx, http.MethodGet, path, automationRunValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *unixSocketClient) ListAutomationRuns(ctx context.Context, query AutomationRunQuery) ([]RunRecord, error) {
	var response contract.RunsResponse
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/automation/runs",
		automationRunValues(query),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *unixSocketClient) GetAutomationRun(ctx context.Context, id string) (RunRecord, error) {
	var response contract.RunResponse
	path := "/api/automation/runs/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return RunRecord{}, err
	}
	return response.Run, nil
}

func (c *unixSocketClient) ListTasks(ctx context.Context, query TaskListQuery) ([]TaskSummaryRecord, error) {
	var response contract.TasksResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/tasks", taskValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Tasks, nil
}

func (c *unixSocketClient) CreateTask(ctx context.Context, request CreateTaskRequest) (TaskRecord, error) {
	var response contract.TaskResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/tasks", nil, request, &response); err != nil {
		return TaskRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) GetTask(ctx context.Context, id string) (TaskDetailRecord, error) {
	var response contract.TaskDetailResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return TaskDetailRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) UpdateTask(ctx context.Context, id string, request UpdateTaskRequest) (TaskRecord, error) {
	var response contract.TaskResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, request, &response); err != nil {
		return TaskRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) GetTaskExecutionProfile(
	ctx context.Context,
	id string,
) (TaskExecutionProfileRecord, error) {
	var response contract.TaskExecutionProfileResponse
	path := taskExecutionProfilePath(id)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return TaskExecutionProfileRecord{}, err
	}
	return response.Profile, nil
}

func (c *unixSocketClient) SetTaskExecutionProfile(
	ctx context.Context,
	id string,
	request *TaskExecutionProfileRequest,
) (TaskExecutionProfileRecord, error) {
	if request == nil {
		return TaskExecutionProfileRecord{}, errors.New("cli: task execution profile request is required")
	}
	var response contract.TaskExecutionProfileResponse
	path := taskExecutionProfilePath(id)
	if err := c.doJSON(ctx, http.MethodPut, path, nil, request, &response); err != nil {
		return TaskExecutionProfileRecord{}, err
	}
	return response.Profile, nil
}

func (c *unixSocketClient) DeleteTaskExecutionProfile(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, taskExecutionProfilePath(id), nil, nil, nil)
}

func taskExecutionProfilePath(id string) string {
	return "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/execution-profile"
}

func (c *unixSocketClient) CreateTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	request *TaskBridgeNotificationSubscriptionRequest,
) (TaskBridgeNotificationSubscriptionRecord, error) {
	if request == nil {
		return TaskBridgeNotificationSubscriptionRecord{}, errors.New(
			"cli: task bridge notification subscription request is required",
		)
	}
	var response contract.TaskBridgeNotificationSubscriptionResponse
	path := taskBridgeNotificationSubscriptionsPath(taskID)
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskBridgeNotificationSubscriptionRecord{}, err
	}
	return response.Subscription, nil
}

func (c *unixSocketClient) ListTaskBridgeNotificationSubscriptions(
	ctx context.Context,
	taskID string,
	query TaskBridgeNotificationSubscriptionQuery,
) ([]TaskBridgeNotificationSubscriptionRecord, error) {
	var response contract.TaskBridgeNotificationSubscriptionsResponse
	path := taskBridgeNotificationSubscriptionsPath(taskID)
	values := taskBridgeNotificationSubscriptionValues(query)
	if err := c.doJSON(ctx, http.MethodGet, path, values, nil, &response); err != nil {
		return nil, err
	}
	return response.Subscriptions, nil
}

func (c *unixSocketClient) GetTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	subscriptionID string,
) (TaskBridgeNotificationSubscriptionRecord, error) {
	var response contract.TaskBridgeNotificationSubscriptionResponse
	path := taskBridgeNotificationSubscriptionPath(taskID, subscriptionID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return TaskBridgeNotificationSubscriptionRecord{}, err
	}
	return response.Subscription, nil
}

func (c *unixSocketClient) DeleteTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	subscriptionID string,
) error {
	path := taskBridgeNotificationSubscriptionPath(taskID, subscriptionID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func taskBridgeNotificationSubscriptionsPath(taskID string) string {
	return "/api/tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/notifications/bridges"
}

func taskBridgeNotificationSubscriptionPath(taskID string, subscriptionID string) string {
	return taskBridgeNotificationSubscriptionsPath(taskID) + "/" + url.PathEscape(strings.TrimSpace(subscriptionID))
}

func (c *unixSocketClient) RequestTaskRunReview(
	ctx context.Context,
	runID string,
	request *TaskRunReviewRequest,
) (TaskRunReviewRequestRecord, error) {
	if request == nil {
		return TaskRunReviewRequestRecord{}, errors.New("cli: task run review request is required")
	}
	var response contract.TaskRunReviewRequestResponse
	path := "/api/task-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/reviews"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskRunReviewRequestRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) ListTaskRunReviews(
	ctx context.Context,
	query TaskRunReviewListQuery,
) ([]TaskRunReviewRecord, error) {
	var response contract.TaskRunReviewsResponse
	path, values, err := taskRunReviewListPathAndValues(query)
	if err != nil {
		return nil, err
	}
	if err := c.doJSON(ctx, http.MethodGet, path, values, nil, &response); err != nil {
		return nil, err
	}
	return response.Reviews, nil
}

func (c *unixSocketClient) GetTaskRunReview(
	ctx context.Context,
	reviewID string,
) (TaskRunReviewRecord, error) {
	var response contract.TaskRunReviewResponse
	path := "/api/task-reviews/" + url.PathEscape(strings.TrimSpace(reviewID))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return TaskRunReviewRecord{}, err
	}
	return response.Review, nil
}

func (c *unixSocketClient) SubmitTaskRunReviewVerdict(
	ctx context.Context,
	reviewID string,
	request *TaskRunReviewVerdictRequest,
) (TaskRunReviewVerdictRecord, error) {
	if request == nil {
		return TaskRunReviewVerdictRecord{}, errors.New("cli: task run review verdict request is required")
	}
	var response contract.TaskRunReviewVerdictResponse
	path := "/api/task-reviews/" + url.PathEscape(strings.TrimSpace(reviewID)) + "/verdict"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskRunReviewVerdictRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) PublishTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	return c.taskExecutionAction(ctx, strings.TrimSpace(id), "publish", request)
}

func (c *unixSocketClient) StartTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	return c.taskExecutionAction(ctx, strings.TrimSpace(id), "start", request)
}

func (c *unixSocketClient) ApproveTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	return c.taskExecutionAction(ctx, strings.TrimSpace(id), "approve", request)
}

func (c *unixSocketClient) DeleteTask(ctx context.Context, id string) error {
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *unixSocketClient) RejectTask(ctx context.Context, id string) (TaskRecord, error) {
	var response contract.TaskResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/reject"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return TaskRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) CancelTask(ctx context.Context, id string, request CancelTaskRequest) (TaskRecord, error) {
	var response contract.TaskResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/cancel"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) CreateChildTask(
	ctx context.Context,
	id string,
	request CreateTaskChildRequest,
) (TaskRecord, error) {
	var response contract.TaskResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/children"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) AddTaskDependency(
	ctx context.Context,
	id string,
	request AddTaskDependencyRequest,
) (TaskDetailRecord, error) {
	var response contract.TaskDetailResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/dependencies"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskDetailRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) RemoveTaskDependency(
	ctx context.Context,
	id string,
	dependsOnID string,
) (TaskDetailRecord, error) {
	var response contract.TaskDetailResponse
	path := "/api/tasks/" + url.PathEscape(
		strings.TrimSpace(id),
	) + "/dependencies/" + url.PathEscape(
		strings.TrimSpace(dependsOnID),
	)
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil, &response); err != nil {
		return TaskDetailRecord{}, err
	}
	return response.Task, nil
}

func (c *unixSocketClient) EnqueueTaskRun(
	ctx context.Context,
	id string,
	request EnqueueTaskRunRequest,
) (TaskRunRecord, error) {
	var response contract.TaskRunResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/runs"
	if err := c.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return TaskRunRecord{}, err
	}
	return response.Run, nil
}

func (c *unixSocketClient) ListTaskRuns(
	ctx context.Context,
	id string,
	query TaskRunListQuery,
) ([]TaskRunRecord, error) {
	var response contract.TaskRunsResponse
	path := "/api/tasks/" + url.PathEscape(strings.TrimSpace(id)) + "/runs"
	if err := c.doJSON(ctx, http.MethodGet, path, taskRunValues(query), nil, &response); err != nil {
		return nil, err
	}
	return response.Runs, nil
}

func (c *unixSocketClient) ClaimTaskRun(
	ctx context.Context,
	id string,
	request ClaimTaskRunRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "claim", request)
}

func (c *unixSocketClient) StartTaskRun(
	ctx context.Context,
	id string,
	request StartTaskRunRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "start", request)
}

func (c *unixSocketClient) AttachTaskRunSession(
	ctx context.Context,
	id string,
	request AttachTaskRunSessionRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "attach-session", request)
}

func (c *unixSocketClient) CompleteTaskRun(
	ctx context.Context,
	id string,
	request CompleteTaskRunRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "complete", request)
}

func (c *unixSocketClient) FailTaskRun(
	ctx context.Context,
	id string,
	request FailTaskRunRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "fail", request)
}

func (c *unixSocketClient) CancelTaskRun(
	ctx context.Context,
	id string,
	request CancelTaskRunRequest,
) (TaskRunRecord, error) {
	return c.taskRunAction(ctx, strings.TrimSpace(id), "cancel", request)
}

func (c *unixSocketClient) AgentMe(
	ctx context.Context,
	credentials agentidentity.Credentials,
) (AgentMeRecord, error) {
	var response contract.AgentMeResponse
	if err := c.doAgentJSON(ctx, http.MethodGet, "/api/agent/me", nil, nil, credentials, &response); err != nil {
		return AgentMeRecord{}, err
	}
	return response.Me, nil
}

func (c *unixSocketClient) AgentContext(
	ctx context.Context,
	credentials agentidentity.Credentials,
) (AgentContextRecord, error) {
	var response contract.AgentContextResponse
	if err := c.doAgentJSON(ctx, http.MethodGet, "/api/agent/context", nil, nil, credentials, &response); err != nil {
		return AgentContextRecord{}, err
	}
	return response.Context, nil
}

func (c *unixSocketClient) AgentSpawn(
	ctx context.Context,
	request AgentSpawnRequest,
	credentials agentidentity.Credentials,
) (AgentSpawnRecord, error) {
	var response contract.AgentSpawnResponse
	if err := c.doAgentJSON(
		ctx,
		http.MethodPost,
		"/api/agent/spawn",
		nil,
		request,
		credentials,
		&response,
	); err != nil {
		return AgentSpawnRecord{}, err
	}
	return response.Spawn, nil
}

func (c *unixSocketClient) AgentChannels(
	ctx context.Context,
	credentials agentidentity.Credentials,
) ([]AgentChannelRecord, error) {
	var response contract.AgentChannelsResponse
	if err := c.doAgentJSON(ctx, http.MethodGet, "/api/agent/channels", nil, nil, credentials, &response); err != nil {
		return nil, err
	}
	return response.Channels, nil
}

func (c *unixSocketClient) AgentChannelRecv(
	ctx context.Context,
	channel string,
	query AgentChannelRecvQuery,
	credentials agentidentity.Credentials,
) ([]AgentChannelMessageRecord, error) {
	var response contract.AgentChannelMessagesResponse
	path := "/api/agent/channels/" + url.PathEscape(strings.TrimSpace(channel)) + "/recv"
	if err := c.doAgentJSON(
		ctx,
		http.MethodGet,
		path,
		agentChannelRecvValues(query),
		nil,
		credentials,
		&response,
	); err != nil {
		return nil, err
	}
	return response.Messages, nil
}

func (c *unixSocketClient) AgentChannelSend(
	ctx context.Context,
	channel string,
	request AgentChannelSendRequest,
	credentials agentidentity.Credentials,
) (AgentChannelMessageRecord, error) {
	var response contract.AgentChannelMessageResponse
	path := "/api/agent/channels/" + url.PathEscape(strings.TrimSpace(channel)) + "/send"
	if err := c.doAgentJSON(
		ctx,
		http.MethodPost,
		path,
		nil,
		request,
		credentials,
		&response,
	); err != nil {
		return AgentChannelMessageRecord{}, err
	}
	return response.Message, nil
}

func (c *unixSocketClient) AgentChannelReply(
	ctx context.Context,
	request AgentChannelReplyRequest,
	credentials agentidentity.Credentials,
) (AgentChannelMessageRecord, error) {
	var response contract.AgentChannelMessageResponse
	if err := c.doAgentJSON(
		ctx,
		http.MethodPost,
		"/api/agent/channels/reply",
		nil,
		request,
		credentials,
		&response,
	); err != nil {
		return AgentChannelMessageRecord{}, err
	}
	return response.Message, nil
}

func (c *unixSocketClient) AgentTaskClaimNext(
	ctx context.Context,
	request AgentTaskClaimNextRequest,
	credentials agentidentity.Credentials,
) (AgentTaskNextRecord, error) {
	const path = "/api/agent/tasks/claim-next"
	response, err := c.doRequestWithCredentials(ctx, http.MethodPost, path, nil, request, "", credentials)
	if err != nil {
		return AgentTaskNextRecord{}, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode == http.StatusNoContent {
		if err := drainResponseBody(http.MethodPost, path, response.Body); err != nil {
			return AgentTaskNextRecord{}, err
		}
		return AgentTaskNextRecord{Claimed: false}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return AgentTaskNextRecord{}, readAPIError(response)
	}

	var decoded contract.AgentTaskClaimResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return AgentTaskNextRecord{}, fmt.Errorf("cli: decode %s %s response: %w", http.MethodPost, path, err)
	}
	claim := decoded.Claim
	return AgentTaskNextRecord{Claimed: true, Claim: &claim}, nil
}

func (c *unixSocketClient) AgentTaskHeartbeat(
	ctx context.Context,
	runID string,
	request AgentTaskHeartbeatRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	return c.agentTaskLeaseAction(ctx, strings.TrimSpace(runID), "heartbeat", request, credentials)
}

func (c *unixSocketClient) AgentTaskComplete(
	ctx context.Context,
	runID string,
	request AgentTaskCompleteRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	return c.agentTaskLeaseAction(ctx, strings.TrimSpace(runID), "complete", request, credentials)
}

func (c *unixSocketClient) AgentTaskFail(
	ctx context.Context,
	runID string,
	request AgentTaskFailRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	return c.agentTaskLeaseAction(ctx, strings.TrimSpace(runID), "fail", request, credentials)
}

func (c *unixSocketClient) AgentTaskRelease(
	ctx context.Context,
	runID string,
	request AgentTaskReleaseRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	return c.agentTaskLeaseAction(ctx, strings.TrimSpace(runID), "release", request, credentials)
}

func (c *unixSocketClient) agentTaskLeaseAction(
	ctx context.Context,
	runID string,
	action string,
	request any,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	var response contract.AgentTaskLeaseResponse
	path := "/api/agent/tasks/" + url.PathEscape(runID) + "/" + strings.TrimSpace(action)
	if err := c.doAgentJSON(ctx, http.MethodPost, path, nil, request, credentials, &response); err != nil {
		return AgentTaskLeaseRecord{}, err
	}
	return response.Lease, nil
}

func (c *unixSocketClient) extensionAction(ctx context.Context, name string, action string) (ExtensionRecord, error) {
	var response struct {
		Extension ExtensionRecord `json:"extension"`
	}
	path := "/api/extensions/" + url.PathEscape(name) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return ExtensionRecord{}, err
	}
	return response.Extension, nil
}

func (c *unixSocketClient) bridgeAction(ctx context.Context, id string, action string) (BridgeRecord, error) {
	var response struct {
		Bridge BridgeRecord `json:"bridge"`
	}
	path := "/api/bridges/" + url.PathEscape(id) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return BridgeRecord{}, err
	}
	return response.Bridge, nil
}

func (c *unixSocketClient) skillAction(
	ctx context.Context,
	name string,
	action string,
	query SkillQuery,
) (SkillActionRecord, error) {
	var response SkillActionRecord
	path := "/api/skills/" + url.PathEscape(name) + "/" + strings.TrimSpace(action)
	if err := c.doJSON(ctx, http.MethodPost, path, skillValues(query), nil, &response); err != nil {
		return SkillActionRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) taskRunAction(
	ctx context.Context,
	id string,
	action string,
	requestBody any,
) (TaskRunRecord, error) {
	var response contract.TaskRunResponse
	path := "/api/task-runs/" + url.PathEscape(id) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, path, nil, requestBody, &response); err != nil {
		return TaskRunRecord{}, err
	}
	return response.Run, nil
}

func (c *unixSocketClient) taskExecutionAction(
	ctx context.Context,
	id string,
	action string,
	requestBody TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	var response contract.TaskExecutionResponse
	path := "/api/tasks/" + url.PathEscape(id) + "/" + strings.TrimSpace(action)
	if err := c.doJSON(ctx, http.MethodPost, path, nil, requestBody, &response); err != nil {
		return TaskExecutionRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) doJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	responseBody any,
) error {
	response, err := c.doRequest(ctx, method, path, query, requestBody, "")
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	return c.decodeJSONResponse(ctx, method, path, response, responseBody)
}

func (c *unixSocketClient) doAgentJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	credentials agentidentity.Credentials,
	responseBody any,
) error {
	response, err := c.doRequestWithCredentials(ctx, method, path, query, requestBody, "", credentials)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	return c.decodeJSONResponse(ctx, method, path, response, responseBody)
}

func (c *unixSocketClient) decodeJSONResponse(
	_ context.Context,
	method string,
	path string,
	response *http.Response,
	responseBody any,
) error {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}
	if responseBody == nil {
		return drainResponseBody(method, path, response.Body)
	}
	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("cli: decode %s %s response: %w", method, path, err)
	}
	return nil
}

func (c *unixSocketClient) doSSE(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	lastEventID string,
	handler SSEHandler,
) error {
	response, err := c.doRequestWithCredentialsAndClient(
		ctx,
		method,
		path,
		query,
		requestBody,
		lastEventID,
		agentidentity.Credentials{},
		c.streamHTTPClient(),
	)
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}

	if handler == nil {
		return drainResponseBody(method, path, response.Body)
	}
	return decodeSSE(ctx, response.Body, handler)
}

func (c *unixSocketClient) doRequest(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	lastEventID string,
) (*http.Response, error) {
	return c.doRequestWithCredentials(
		ctx,
		method,
		path,
		query,
		requestBody,
		lastEventID,
		agentidentity.Credentials{},
	)
}

func (c *unixSocketClient) doRequestWithCredentials(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	lastEventID string,
	credentials agentidentity.Credentials,
) (*http.Response, error) {
	return c.doRequestWithCredentialsAndClient(
		ctx,
		method,
		path,
		query,
		requestBody,
		lastEventID,
		credentials,
		c.httpClient,
	)
}

// doRequestWithCredentialsAndClient lets SSE streams opt out of the JSON request timeout.
func (c *unixSocketClient) doRequestWithCredentialsAndClient(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	requestBody any,
	lastEventID string,
	credentials agentidentity.Credentials,
	client *http.Client,
) (*http.Response, error) {
	if ctx == nil {
		return nil, errors.New("cli: context is required")
	}
	if client == nil {
		return nil, errors.New("cli: http client is required")
	}

	target := baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body io.Reader
	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return nil, fmt.Errorf("cli: encode %s %s request: %w", method, path, err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("cli: build %s %s request: %w", method, path, err)
	}
	req.Header.Set("User-Agent", defaultUserAgentName)
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(lastEventID) != "" {
		req.Header.Set("Last-Event-ID", strings.TrimSpace(lastEventID))
	}
	setAgentIdentityHeaders(req, credentials)

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cli: %s %s via %s: %w", method, path, c.socketPath, err)
	}
	return response, nil
}

// streamHTTPClient preserves long-lived streams when no dedicated client has been configured.
func (c *unixSocketClient) streamHTTPClient() *http.Client {
	if c != nil && c.streamClient != nil {
		return c.streamClient
	}
	if c == nil {
		return nil
	}
	return c.httpClient
}

func setAgentIdentityHeaders(req *http.Request, credentials agentidentity.Credentials) {
	if req == nil {
		return
	}
	if sessionID := strings.TrimSpace(credentials.SessionID); sessionID != "" {
		req.Header.Set(agentidentity.HeaderSessionID, sessionID)
	}
	if agentName := strings.TrimSpace(credentials.AgentName); agentName != "" {
		req.Header.Set(agentidentity.HeaderAgent, agentName)
	}
	if workspaceID := strings.TrimSpace(credentials.WorkspaceID); workspaceID != "" {
		req.Header.Set(agentidentity.HeaderWorkspaceID, workspaceID)
	}
}

func decodeSSE(ctx context.Context, body io.Reader, handler SSEHandler) error {
	if ctx == nil {
		return fmt.Errorf("sse: context is required")
	}
	if cliReaderIsNil(body) {
		return fmt.Errorf("sse: body is required")
	}
	if handler == nil {
		return fmt.Errorf("sse: handler is required")
	}

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	event := SSEEvent{}
	dataLines := make([]string, 0, 4)
	emit := func() error {
		if event.ID == "" && event.Event == "" && len(dataLines) == 0 {
			return nil
		}
		if len(dataLines) > 0 {
			event.Data = json.RawMessage(strings.Join(dataLines, "\n"))
		}
		err := handler(event)
		event = SSEEvent{}
		dataLines = dataLines[:0]
		return err
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		if decodeSSELine(scanner.Text(), &event, &dataLines) {
			err := emit()
			if errors.Is(err, errStopSSE) {
				return nil
			}
			if err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("sse: read stream: %w", err)
	}

	err := emit()
	if errors.Is(err, errStopSSE) {
		return nil
	}
	return err
}

func decodeSSELine(line string, event *SSEEvent, dataLines *[]string) bool {
	if line == "" {
		return true
	}
	if strings.HasPrefix(line, ":") {
		return false
	}

	field, value, found := strings.Cut(line, ":")
	if !found {
		return false
	}

	switch field {
	case "id":
		event.ID = strings.TrimPrefix(value, " ")
	case "event":
		event.Event = strings.TrimPrefix(value, " ")
	case "data":
		*dataLines = append(*dataLines, strings.TrimPrefix(value, " "))
	}

	return false
}

func cliReaderIsNil(reader io.Reader) bool {
	if reader == nil {
		return true
	}

	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func sessionListValues(query SessionListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	return values
}

func sessionRepairValues(query SessionRepairQuery) url.Values {
	values := url.Values{}
	if query.DryRun {
		values.Set("dry_run", "true")
	}
	if query.Force {
		values.Set("force", "true")
	}
	return values
}

func sessionInspectValues(query SessionInspectQuery) url.Values {
	values := url.Values{}
	if query.IncludeRecentWakeEvents {
		values.Set("include_recent_wake_events", "true")
	}
	return values
}

func networkPeersValues(query NetworkPeersQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Channel); trimmed != "" {
		values.Set("channel", trimmed)
	}
	return values
}

func networkThreadsValues(query NetworkThreadsQuery) url.Values {
	return networkListValues(query.Limit, query.After)
}

func networkDirectsValues(query NetworkDirectsQuery) url.Values {
	values := networkListValues(query.Limit, query.After)
	if trimmed := strings.TrimSpace(query.PeerID); trimmed != "" {
		values.Set("peer_id", trimmed)
	}
	return values
}

func networkConversationMessagesValues(query NetworkConversationMessagesQuery) url.Values {
	values := networkListValues(query.Limit, query.After)
	if trimmed := strings.TrimSpace(query.Before); trimmed != "" {
		values.Set("before", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Kind); trimmed != "" {
		values.Set("kind", trimmed)
	}
	if trimmed := strings.TrimSpace(query.WorkID); trimmed != "" {
		values.Set("work_id", trimmed)
	}
	return values
}

func networkListValues(limit int, after string) url.Values {
	values := url.Values{}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(limit))
	}
	if trimmed := strings.TrimSpace(after); trimmed != "" {
		values.Set("after", trimmed)
	}
	return values
}

func networkThreadPath(channel string, threadID string) (string, error) {
	channel, err := requireNetworkPathValue("channel", channel)
	if err != nil {
		return "", err
	}
	threadID, err = requireNetworkPathValue("thread_id", threadID)
	if err != nil {
		return "", err
	}
	return "/api/network/channels/" + url.PathEscape(channel) + "/threads/" + url.PathEscape(threadID), nil
}

func networkThreadMessagesPath(channel string, threadID string) (string, error) {
	path, err := networkThreadPath(channel, threadID)
	if err != nil {
		return "", err
	}
	return path + "/messages", nil
}

func networkDirectPath(channel string, directID string) (string, error) {
	channel, err := requireNetworkPathValue("channel", channel)
	if err != nil {
		return "", err
	}
	directID, err = requireNetworkPathValue("direct_id", directID)
	if err != nil {
		return "", err
	}
	return "/api/network/channels/" + url.PathEscape(channel) + "/directs/" + url.PathEscape(directID), nil
}

func networkDirectMessagesPath(channel string, directID string) (string, error) {
	path, err := networkDirectPath(channel, directID)
	if err != nil {
		return "", err
	}
	return path + "/messages", nil
}

func requireNetworkPathValue(name string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("cli: %s is required", name)
	}
	return trimmed, nil
}

func networkInboxValues(sessionID string) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(sessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	return values
}

func vaultListValues(query VaultListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Prefix); trimmed != "" {
		values.Set("prefix", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Namespace); trimmed != "" {
		values.Set("namespace", trimmed)
	}
	return values
}

func vaultRefValues(ref string) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(ref); trimmed != "" {
		values.Set("ref", trimmed)
	}
	return values
}

func requireVaultRef(ref string) (string, error) {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return "", errors.New("cli: vault ref is required")
	}
	return trimmed, nil
}

func agentValues(query AgentQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	return values
}

func agentSoulHistoryValues(request AgentSoulHistoryRequest) url.Values {
	values := agentValues(AgentQuery{Workspace: request.WorkspaceID})
	if request.Limit > 0 {
		values.Set("limit", strconv.Itoa(request.Limit))
	}
	if trimmed := strings.TrimSpace(request.Cursor); trimmed != "" {
		values.Set("cursor", trimmed)
	}
	return values
}

func agentHeartbeatHistoryValues(request AgentHeartbeatHistoryRequest) url.Values {
	values := agentValues(AgentQuery{Workspace: request.WorkspaceID})
	if request.Limit > 0 {
		values.Set("limit", strconv.Itoa(request.Limit))
	}
	if trimmed := strings.TrimSpace(request.Cursor); trimmed != "" {
		values.Set("cursor", trimmed)
	}
	return values
}

func agentHeartbeatStatusValues(request AgentHeartbeatStatusRequest) url.Values {
	values := agentValues(AgentQuery{Workspace: request.WorkspaceID})
	if trimmed := strings.TrimSpace(request.SessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if request.IncludeSessionHealth {
		values.Set("include_session_health", "true")
	}
	if request.IncludeRecentWakeEvents {
		values.Set("include_recent_wake_events", "true")
	}
	return values
}

func skillValues(query SkillQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	if trimmed := strings.TrimSpace(query.ForAgent); trimmed != "" {
		values.Set("for_agent", trimmed)
	}
	return values
}

func resourceListValues(query ResourceListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Kind)); trimmed != "" {
		values.Set("kind", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.ScopeKind)); trimmed != "" {
		values.Set("scope_kind", trimmed)
	}
	if trimmed := strings.TrimSpace(query.ScopeID); trimmed != "" {
		values.Set("scope_id", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.OwnerKind)); trimmed != "" {
		values.Set("owner_kind", trimmed)
	}
	if trimmed := strings.TrimSpace(query.OwnerID); trimmed != "" {
		values.Set("owner_id", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.SourceKind)); trimmed != "" {
		values.Set("source_kind", trimmed)
	}
	if trimmed := strings.TrimSpace(query.SourceID); trimmed != "" {
		values.Set("source_id", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func agentChannelRecvValues(query AgentChannelRecvQuery) url.Values {
	values := url.Values{}
	if query.Wait {
		values.Set("wait", "true")
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func sessionEventValues(query SessionEventQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Type); trimmed != "" {
		values.Set("type", trimmed)
	}
	if trimmed := strings.TrimSpace(query.AgentName); trimmed != "" {
		values.Set("agent_name", trimmed)
	}
	if trimmed := strings.TrimSpace(query.TurnID); trimmed != "" {
		values.Set("turn_id", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if query.Last > 0 {
		values.Set("limit", strconv.Itoa(query.Last))
	}
	if query.AfterSequence > 0 {
		values.Set("after_sequence", strconv.FormatInt(query.AfterSequence, 10))
	}
	return values
}

func observeEventValues(query ObserveEventQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.SessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.AgentName); trimmed != "" {
		values.Set("agent_name", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Type); trimmed != "" {
		values.Set("type", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if query.Last > 0 {
		values.Set("limit", strconv.Itoa(query.Last))
	}
	return values
}

func hookCatalogValues(query HookCatalogQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Agent); trimmed != "" {
		values.Set("agent", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Event); trimmed != "" {
		values.Set("event", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Source); trimmed != "" {
		values.Set("source", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Mode); trimmed != "" {
		values.Set("mode", trimmed)
	}
	return values
}

func hookRunsValues(query HookRunsQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Session); trimmed != "" {
		values.Set("session", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Event); trimmed != "" {
		values.Set("event", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Outcome); trimmed != "" {
		values.Set("outcome", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Since); trimmed != "" {
		values.Set("since", trimmed)
	}
	if query.Last > 0 {
		values.Set("last", strconv.Itoa(query.Last))
	}
	return values
}

func hookEventsValues(query HookEventsQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Family); trimmed != "" {
		values.Set("family", trimmed)
	}
	if query.SyncOnly {
		values.Set("sync_only", strconv.FormatBool(query.SyncOnly))
	}
	return values
}

func memorySelectorValues(query MemorySelectorQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(query.WorkspaceID); trimmed != "" {
		values.Set("workspace_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.AgentName); trimmed != "" {
		values.Set("agent_name", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.AgentTier)); trimmed != "" {
		values.Set("agent_tier", trimmed)
	}
	if query.IncludeSystem {
		values.Set("include_system", strconv.FormatBool(query.IncludeSystem))
	}
	return values
}

func memoryListValues(query MemoryListQuery) url.Values {
	values := memorySelectorValues(query.MemorySelectorQuery)
	if trimmed := strings.TrimSpace(string(query.Type)); trimmed != "" {
		values.Set("type", trimmed)
	}
	if query.IncludeShadowed {
		values.Set("include_shadowed", strconv.FormatBool(query.IncludeShadowed))
	}
	return values
}

func memoryHistoryValues(query MemoryHistoryQuery) url.Values {
	values := memorySelectorValues(MemorySelectorQuery{
		Scope:       query.Scope,
		WorkspaceID: query.WorkspaceID,
		AgentName:   query.AgentName,
		AgentTier:   query.AgentTier,
	})
	if trimmed := strings.TrimSpace(query.Operation); trimmed != "" {
		values.Set("operation", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func memoryDecisionValues(query MemoryDecisionListQuery) url.Values {
	values := memorySelectorValues(MemorySelectorQuery{
		Scope:       query.Scope,
		WorkspaceID: query.WorkspaceID,
		AgentName:   query.AgentName,
		AgentTier:   query.AgentTier,
	})
	if trimmed := strings.TrimSpace(query.Operation); trimmed != "" {
		values.Set("op", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if trimmed := strings.TrimSpace(query.Reason); trimmed != "" {
		values.Set("reason", trimmed)
	}
	return values
}

func automationJobValues(query AutomationJobQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(query.WorkspaceID); trimmed != "" {
		values.Set("workspace_id", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.Source)); trimmed != "" {
		values.Set("source", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func automationTriggerValues(query AutomationTriggerQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(query.WorkspaceID); trimmed != "" {
		values.Set("workspace_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Event); trimmed != "" {
		values.Set("event", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.Source)); trimmed != "" {
		values.Set("source", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func automationRunValues(query AutomationRunQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.JobID); trimmed != "" {
		values.Set("job_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.TriggerID); trimmed != "" {
		values.Set("trigger_id", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.Status)); trimmed != "" {
		values.Set("status", trimmed)
	}
	if !query.Since.IsZero() {
		values.Set("since", query.Since.UTC().Format(time.RFC3339Nano))
	}
	if !query.Until.IsZero() {
		values.Set("until", query.Until.UTC().Format(time.RFC3339Nano))
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func taskValues(query TaskListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(query.Workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.Status)); trimmed != "" {
		values.Set("status", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.OwnerKind)); trimmed != "" {
		values.Set("owner_kind", trimmed)
	}
	if trimmed := strings.TrimSpace(query.OwnerRef); trimmed != "" {
		values.Set("owner_ref", trimmed)
	}
	if trimmed := strings.TrimSpace(query.ParentTaskID); trimmed != "" {
		values.Set("parent_task_id", trimmed)
	}
	if trimmed := strings.TrimSpace(query.NetworkChannel); trimmed != "" {
		values.Set("network_channel", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func taskRunValues(query TaskRunListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Status)); trimmed != "" {
		values.Set("status", trimmed)
	}
	if trimmed := strings.TrimSpace(query.SessionID); trimmed != "" {
		values.Set("session_id", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func taskRunReviewListPathAndValues(query TaskRunReviewListQuery) (string, url.Values, error) {
	taskID := strings.TrimSpace(query.TaskID)
	runID := strings.TrimSpace(query.RunID)
	switch {
	case taskID != "" && runID != "":
		return "", nil, errors.New("cli: choose either --task or --run when listing task reviews")
	case taskID != "":
		return "/api/tasks/" + url.PathEscape(taskID) + "/reviews", taskRunReviewValues(query), nil
	case runID != "":
		return "/api/task-runs/" + url.PathEscape(runID) + "/reviews", taskRunReviewValues(query), nil
	default:
		return "", nil, errors.New("cli: task review list requires --task or --run")
	}
}

func taskRunReviewValues(query TaskRunReviewListQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(query.Status)); trimmed != "" {
		values.Set("status", trimmed)
	}
	if trimmed := strings.TrimSpace(query.ReviewerSessionID); trimmed != "" {
		values.Set("reviewer_session_id", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func taskBridgeNotificationSubscriptionValues(query TaskBridgeNotificationSubscriptionQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.BridgeInstanceID); trimmed != "" {
		values.Set("bridge_instance_id", trimmed)
	}
	if trimmed := strings.TrimSpace(string(query.Scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(query.WorkspaceID); trimmed != "" {
		values.Set("workspace_id", trimmed)
	}
	if query.Limit > 0 {
		values.Set("limit", strconv.Itoa(query.Limit))
	}
	return values
}

func readAPIError(response *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("cli: read api error response: %w", err)
	}
	return readAPIErrorBody(response.StatusCode, response.Status, body)
}

func readAPIErrorBody(statusCode int, status string, body []byte) error {
	var payload struct {
		Error string `json:"error"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return errors.New(redactToolDiagnostic(payload.Error))
	}
	var memoryPayload contract.MemoryErrorPayload
	if len(body) > 0 && json.Unmarshal(body, &memoryPayload) == nil &&
		strings.TrimSpace(memoryPayload.Code) != "" {
		message := strings.TrimSpace(memoryPayload.Message)
		if message == "" {
			message = strings.TrimSpace(memoryPayload.Code)
		}
		return fmt.Errorf("%s: %s", strings.TrimSpace(memoryPayload.Code), redactToolDiagnostic(message))
	}
	var toolPayload contract.ToolErrorResponse
	if len(body) > 0 && json.Unmarshal(body, &toolPayload) == nil && toolPayload.Error.Code != "" {
		return newToolAPIError(statusCode, status, toolPayload)
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = status
	}
	message = redactToolDiagnostic(message)
	if strings.TrimSpace(status) == "" {
		return errors.New(message)
	}
	return fmt.Errorf("daemon api %s: %s", status, message)
}

func drainResponseBody(method string, path string, body io.Reader) error {
	if _, err := io.Copy(io.Discard, body); err != nil {
		return fmt.Errorf("cli: drain %s %s response: %w", method, path, err)
	}
	return nil
}
