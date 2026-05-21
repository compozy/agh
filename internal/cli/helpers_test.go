package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/agentidentity"
	"github.com/pedronauck/agh/internal/api/contract"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/testutil"
)

var fixedTestNow = time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

type stubClient struct {
	statusFn                    func(context.Context) (StatusRecord, error)
	doctorFn                    func(context.Context, DoctorQuery) (DoctorRecord, error)
	daemonStatusFn              func(context.Context) (DaemonStatus, error)
	triggerSettingsRestartFn    func(context.Context) (SettingsRestartActionRecord, error)
	getSettingsRestartStatusFn  func(context.Context, string) (SettingsRestartStatusRecord, error)
	createSupportBundleFn       func(context.Context, CreateSupportBundleRequest) (SupportBundleOperationRecord, error)
	getSupportBundleFn          func(context.Context, string) (SupportBundleOperationRecord, error)
	downloadSupportBundleFn     func(context.Context, string, io.Writer) error
	getSettingsUpdateFn         func(context.Context) (SettingsUpdateRecord, error)
	updateSettingsSkillsFn      func(context.Context, UpdateSettingsSkillsRequest) (SettingsMutationRecord, error)
	reloadSettingsFn            func(context.Context) (SettingsMutationRecord, error)
	listSettingsApplyRecordsFn  func(context.Context, SettingsApplyHistoryQuery) (SettingsApplyHistoryRecord, error)
	listProvidersFn             func(context.Context) (contract.ProviderListResponse, error)
	probeProviderAuthFn         func(context.Context, string) (contract.ProviderAuthProbeResponse, error)
	listProviderModelsFn        func(context.Context, ProviderModelListQuery) (ProviderModelListRecord, error)
	refreshProviderModelsFn     func(context.Context, string, ProviderModelRefreshRequest) (ProviderModelRefreshRecord, error)
	providerModelStatusFn       func(context.Context, string) (ProviderModelStatusRecord, error)
	listVaultSecretsFn          func(context.Context, VaultListQuery) ([]VaultRecord, error)
	getVaultSecretFn            func(context.Context, string) (VaultRecord, error)
	putVaultSecretFn            func(context.Context, PutVaultSecretRequest) (VaultRecord, error)
	deleteVaultSecretFn         func(context.Context, string) error
	networkStatusFn             func(context.Context) (NetworkStatusRecord, error)
	networkPeersFn              func(context.Context, NetworkPeersQuery) ([]NetworkPeerRecord, error)
	networkChannelsFn           func(context.Context, string) ([]NetworkChannelRecord, error)
	createNetworkChannelFn      func(context.Context, string, CreateNetworkChannelRequest) (NetworkChannelDetailRecord, error)
	networkThreadsFn            func(context.Context, NetworkThreadsQuery) ([]NetworkThreadRecord, error)
	networkThreadFn             func(context.Context, string, string, string) (NetworkThreadRecord, error)
	networkThreadMessagesFn     func(context.Context, NetworkConversationMessagesQuery) ([]NetworkConversationMessageRecord, error)
	networkDirectsFn            func(context.Context, NetworkDirectsQuery) ([]NetworkDirectRoomRecord, error)
	networkDirectResolveFn      func(context.Context, string, string, NetworkDirectResolveRequest) (NetworkDirectRoomRecord, error)
	networkDirectFn             func(context.Context, string, string, string) (NetworkDirectRoomRecord, error)
	networkDirectMessagesFn     func(context.Context, NetworkConversationMessagesQuery) ([]NetworkConversationMessageRecord, error)
	networkWorkFn               func(context.Context, string, string) (NetworkWorkRecord, error)
	networkSendFn               func(context.Context, NetworkSendRequest) (NetworkSendRecord, error)
	networkInboxFn              func(context.Context, string, string) ([]NetworkEnvelopeRecord, error)
	listExtensionsFn            func(context.Context) ([]ExtensionRecord, error)
	installExtensionFn          func(context.Context, InstallExtensionRequest) (ExtensionRecord, error)
	enableExtensionFn           func(context.Context, string) (ExtensionRecord, error)
	disableExtensionFn          func(context.Context, string) (ExtensionRecord, error)
	extensionStatusFn           func(context.Context, string) (ExtensionRecord, error)
	listBundleCatalogFn         func(context.Context) ([]BundleCatalogRecord, error)
	previewBundleActivationFn   func(context.Context, ActivateBundleRequest) (BundleActivationRecord, error)
	activateBundleFn            func(context.Context, ActivateBundleRequest) (BundleActivationRecord, error)
	listBundleActivationsFn     func(context.Context) ([]BundleActivationRecord, error)
	getBundleActivationFn       func(context.Context, string) (BundleActivationRecord, error)
	updateBundleActivationFn    func(context.Context, string, UpdateBundleActivationRequest) (BundleActivationRecord, error)
	deactivateBundleFn          func(context.Context, string) error
	bundleNetworkSettingsFn     func(context.Context) (BundleNetworkSettingsRecord, error)
	listBridgesFn               func(context.Context) ([]BridgeRecord, error)
	createBridgeFn              func(context.Context, CreateBridgeRequest) (BridgeRecord, error)
	getBridgeFn                 func(context.Context, string) (BridgeRecord, error)
	updateBridgeFn              func(context.Context, string, UpdateBridgeRequest) (BridgeRecord, error)
	enableBridgeFn              func(context.Context, string) (BridgeRecord, error)
	disableBridgeFn             func(context.Context, string) (BridgeRecord, error)
	restartBridgeFn             func(context.Context, string) (BridgeRecord, error)
	bridgeRoutesFn              func(context.Context, string) ([]BridgeRouteRecord, error)
	listBridgeSecretBindingsFn  func(context.Context, string) ([]BridgeSecretBindingRecord, error)
	putBridgeSecretBindingFn    func(context.Context, string, string, BridgeSecretBindingRequest) (BridgeSecretBindingRecord, error)
	deleteBridgeSecretBindingFn func(context.Context, string, string) error
	testBridgeDeliveryFn        func(context.Context, string, BridgeTestDeliveryRequest) (BridgeTestDeliveryRecord, error)
	listSessionsFn              func(context.Context, SessionListQuery) ([]SessionRecord, error)
	createSessionFn             func(context.Context, CreateSessionRequest) (SessionRecord, error)
	getSessionFn                func(context.Context, string) (SessionRecord, error)
	getSessionHealthFn          func(context.Context, string) (SessionHealthRecord, error)
	getSessionStatusFn          func(context.Context, string) (SessionStatusRecord, error)
	inspectSessionFn            func(context.Context, string, SessionInspectQuery) (SessionInspectRecord, error)
	refreshSessionSoulFn        func(context.Context, string, SessionSoulRefreshRequest) (AgentSoulRecord, error)
	stopSessionFn               func(context.Context, string) error
	resumeSessionFn             func(context.Context, string) (SessionRecord, error)
	sessionRecapFn              func(context.Context, string, int) (SessionRecapRecord, error)
	repairSessionFn             func(context.Context, string, SessionRepairQuery) (SessionRepairRecord, error)
	approveSessionFn            func(context.Context, string, SessionApprovalRequest) (SessionApprovalRecord, error)
	promptSessionFn             func(context.Context, string, string) ([]AgentEventRecord, error)
	streamPromptSessionFn       func(context.Context, string, string, SSEHandler) error
	sessionEventsFn             func(context.Context, string, SessionEventQuery) ([]SessionEventRecord, error)
	streamSessionFn             func(context.Context, string, SessionEventQuery, string, SSEHandler) error
	sessionHistoryFn            func(context.Context, string, SessionEventQuery) ([]TurnHistoryRecord, error)
	createWorkspaceFn           func(context.Context, WorkspaceCreateRequest) (WorkspaceRecord, error)
	listWorkspacesFn            func(context.Context) ([]WorkspaceRecord, error)
	getWorkspaceFn              func(context.Context, string) (WorkspaceDetailRecord, error)
	updateWorkspaceFn           func(context.Context, string, WorkspaceUpdateRequest) (WorkspaceRecord, error)
	deleteWorkspaceFn           func(context.Context, string) error
	listAgentsFn                func(context.Context, AgentQuery) ([]AgentRecord, error)
	getAgentFn                  func(context.Context, string, AgentQuery) (AgentRecord, error)
	getAgentSoulFn              func(context.Context, string, AgentQuery) (AgentSoulRecord, error)
	validateAgentSoulFn         func(context.Context, string, AgentSoulValidateRequest) (AgentSoulRecord, error)
	putAgentSoulFn              func(context.Context, string, AgentSoulPutRequest) (AgentSoulMutationRecord, error)
	deleteAgentSoulFn           func(context.Context, string, AgentSoulDeleteRequest) (AgentSoulMutationRecord, error)
	listAgentSoulHistoryFn      func(context.Context, string, AgentSoulHistoryRequest) (AgentSoulHistoryRecord, error)
	rollbackAgentSoulFn         func(context.Context, string, AgentSoulRollbackRequest) (AgentSoulMutationRecord, error)
	getAgentHeartbeatFn         func(context.Context, string, AgentQuery) (AgentHeartbeatRecord, error)
	validateAgentHeartbeatFn    func(context.Context, string, AgentHeartbeatValidateRequest) (AgentHeartbeatRecord, error)
	putAgentHeartbeatFn         func(context.Context, string, AgentHeartbeatPutRequest) (AgentHeartbeatMutationRecord, error)
	deleteAgentHeartbeatFn      func(context.Context, string, AgentHeartbeatDeleteRequest) (AgentHeartbeatMutationRecord, error)
	listAgentHeartbeatHistoryFn func(
		context.Context,
		string,
		AgentHeartbeatHistoryRequest,
	) (AgentHeartbeatHistoryRecord, error)
	rollbackAgentHeartbeatFn      func(context.Context, string, AgentHeartbeatRollbackRequest) (AgentHeartbeatMutationRecord, error)
	getAgentHeartbeatStatusFn     func(context.Context, string, AgentHeartbeatStatusRequest) (AgentHeartbeatStatusRecord, error)
	wakeAgentHeartbeatFn          func(context.Context, string, AgentHeartbeatWakeRequest) (AgentHeartbeatWakeDecisionRecord, error)
	listResourcesFn               func(context.Context, ResourceListQuery) ([]ResourceRecord, error)
	getResourceFn                 func(context.Context, string, string) (ResourceRecord, error)
	putResourceFn                 func(context.Context, string, string, ResourcePutRequest) (ResourceRecord, error)
	deleteResourceFn              func(context.Context, string, string, ResourceDeleteRequest) error
	listSkillsFn                  func(context.Context, SkillQuery) ([]SkillRecord, error)
	getSkillFn                    func(context.Context, string, SkillQuery) (SkillRecord, error)
	getSkillContentFn             func(context.Context, string, SkillQuery) (string, error)
	enableSkillFn                 func(context.Context, string, SkillQuery) (SkillActionRecord, error)
	disableSkillFn                func(context.Context, string, SkillQuery) (SkillActionRecord, error)
	listToolsFn                   func(context.Context, ToolQuery) (ToolsResponseRecord, error)
	searchToolsFn                 func(context.Context, ToolSearchRequest) (ToolsResponseRecord, error)
	getToolFn                     func(context.Context, string, ToolQuery) (ToolResponseRecord, error)
	createToolApprovalFn          func(context.Context, string, ToolApprovalRequest) (ToolApprovalRecord, error)
	invokeToolFn                  func(context.Context, string, ToolInvokeRequest) (ToolInvokeResponseRecord, error)
	listToolsetsFn                func(context.Context, ToolQuery) (ToolsetsResponseRecord, error)
	getToolsetFn                  func(context.Context, string, ToolQuery) (ToolsetResponseRecord, error)
	hookCatalogFn                 func(context.Context, HookCatalogQuery) ([]HookCatalogRecord, error)
	hookRunsFn                    func(context.Context, string, HookRunsQuery) ([]HookRunRecord, error)
	hookEventsFn                  func(context.Context, HookEventsQuery) ([]HookEventRecord, error)
	listLogsFn                    func(context.Context, LogsListQuery) ([]LogEventRecord, error)
	streamLogsFn                  func(context.Context, LogsListQuery, string, SSEHandler) error
	memoryHealthFn                func(context.Context, string) (MemoryHealthRecord, error)
	memoryHistoryFn               func(context.Context, MemoryHistoryQuery) ([]MemoryHistoryRecord, error)
	listMemoryFn                  func(context.Context, MemoryListQuery) (MemoryListRecord, error)
	showMemoryFn                  func(context.Context, string, MemorySelectorQuery) (MemoryEntryRecord, error)
	createMemoryFn                func(context.Context, MemoryCreateRequest) (MemoryMutationRecord, error)
	editMemoryFn                  func(context.Context, string, MemoryEditRequest) (MemoryMutationRecord, error)
	deleteMemoryFn                func(context.Context, string, MemorySelectorQuery) (MemoryDeleteRecord, error)
	searchMemoryFn                func(context.Context, MemorySearchRequest) (MemorySearchRecord, error)
	reindexMemoryFn               func(context.Context, MemoryReindexRequest) (MemoryReindexRecord, error)
	promoteMemoryFn               func(context.Context, MemoryPromoteRequest) (MemoryPromoteRecord, error)
	resetMemoryFn                 func(context.Context, MemoryResetRequest) (MemoryResetRecord, error)
	reloadMemoryFn                func(context.Context, MemorySelectorQuery) (MemoryReloadRecord, error)
	memoryScopeShowFn             func(context.Context, MemorySelectorQuery) (MemoryScopeShowRecord, error)
	listMemoryDecisionsFn         func(context.Context, MemoryDecisionListQuery) (MemoryDecisionListRecord, error)
	getMemoryDecisionFn           func(context.Context, string) (MemoryDecisionRecord, error)
	revertMemoryDecisionFn        func(context.Context, string, MemoryDecisionRevertRequest) (MemoryDecisionRevertRecord, error)
	getMemoryRecallTraceFn        func(context.Context, string, int64) (MemoryRecallTraceRecord, error)
	listMemoryDreamsFn            func(context.Context) (MemoryDreamListRecord, error)
	getMemoryDreamFn              func(context.Context, string) (MemoryDreamRecord, error)
	triggerMemoryDreamFn          func(context.Context, MemoryDreamTriggerRequest) (MemoryDreamTriggerRecord, error)
	retryMemoryDreamFn            func(context.Context, string, MemoryDreamRetryRequest) (MemoryDreamRetryRecord, error)
	getMemoryDreamStatusFn        func(context.Context) (MemoryDreamListRecord, error)
	listMemoryDailyLogsFn         func(context.Context, MemorySelectorQuery) (MemoryDailyLogListRecord, error)
	getMemoryExtractorStatusFn    func(context.Context, string) (MemoryExtractorStatusRecord, error)
	listMemoryExtractorFailuresFn func(context.Context) (MemoryExtractorFailuresRecord, error)
	retryMemoryExtractorFn        func(context.Context, MemoryExtractorRetryRequest) (MemoryExtractorRetryRecord, error)
	drainMemoryExtractorFn        func(context.Context) (MemoryExtractorDrainRecord, error)
	listMemoryProvidersFn         func(context.Context) (MemoryProviderListRecord, error)
	getMemoryProviderFn           func(context.Context, string) (MemoryProviderRecord, error)
	selectMemoryProviderFn        func(context.Context, MemoryProviderSelectRequest) (MemoryProviderLifecycleRecord, error)
	enableMemoryProviderFn        func(context.Context, string, MemoryProviderLifecycleRequest) (MemoryProviderLifecycleRecord, error)
	disableMemoryProviderFn       func(context.Context, string, MemoryProviderLifecycleRequest) (MemoryProviderLifecycleRecord, error)
	createMemoryAdhocNoteFn       func(context.Context, MemoryAdhocNoteRequest) (MemoryAdhocNoteRecord, error)
	listAutomationJobsFn          func(context.Context, AutomationJobQuery) ([]JobRecord, error)
	createAutomationJobFn         func(context.Context, AutomationJobCreateRequest) (JobRecord, error)
	getAutomationJobFn            func(context.Context, string) (JobRecord, error)
	updateAutomationJobFn         func(context.Context, string, AutomationJobUpdateRequest) (JobRecord, error)
	deleteAutomationJobFn         func(context.Context, string) error
	triggerAutomationJobFn        func(context.Context, string) (RunRecord, error)
	automationJobRunsFn           func(context.Context, string, AutomationRunQuery) ([]RunRecord, error)
	listAutomationTriggersFn      func(context.Context, AutomationTriggerQuery) ([]TriggerRecord, error)
	createAutomationTriggerFn     func(context.Context, AutomationTriggerCreateRequest) (TriggerRecord, error)
	getAutomationTriggerFn        func(context.Context, string) (TriggerRecord, error)
	updateAutomationTriggerFn     func(context.Context, string, AutomationTriggerUpdateRequest) (TriggerRecord, error)
	deleteAutomationTriggerFn     func(context.Context, string) error
	automationTriggerRunsFn       func(context.Context, string, AutomationRunQuery) ([]RunRecord, error)
	listAutomationRunsFn          func(context.Context, AutomationRunQuery) ([]RunRecord, error)
	getAutomationRunFn            func(context.Context, string) (RunRecord, error)
	listTasksFn                   func(context.Context, TaskListQuery) ([]TaskSummaryRecord, error)
	createTaskFn                  func(context.Context, CreateTaskRequest) (TaskRecord, error)
	createTaskAsAgentFn           func(context.Context, CreateTaskRequest, agentidentity.Credentials) (TaskRecord, error)
	getTaskFn                     func(context.Context, string) (TaskDetailRecord, error)
	updateTaskFn                  func(context.Context, string, UpdateTaskRequest) (TaskRecord, error)
	deleteTaskFn                  func(context.Context, string) error
	getTaskExecutionProfileFn     func(context.Context, string) (TaskExecutionProfileRecord, error)
	setTaskExecutionProfileFn     func(
		context.Context,
		string,
		*TaskExecutionProfileRequest,
	) (TaskExecutionProfileRecord, error)
	deleteTaskExecutionProfileFn               func(context.Context, string) error
	createTaskBridgeNotificationSubscriptionFn func(
		context.Context,
		string,
		*TaskBridgeNotificationSubscriptionRequest,
	) (TaskBridgeNotificationSubscriptionRecord, error)
	listTaskBridgeNotificationSubscriptionsFn func(
		context.Context,
		string,
		TaskBridgeNotificationSubscriptionQuery,
	) ([]TaskBridgeNotificationSubscriptionRecord, error)
	getTaskBridgeNotificationSubscriptionFn    func(context.Context, string, string) (TaskBridgeNotificationSubscriptionRecord, error)
	deleteTaskBridgeNotificationSubscriptionFn func(context.Context, string, string) error
	requestTaskRunReviewFn                     func(context.Context, string, *TaskRunReviewRequest) (TaskRunReviewRequestRecord, error)
	requestTaskRunReviewAsAgentFn              func(
		context.Context,
		string,
		*TaskRunReviewRequest,
		agentidentity.Credentials,
	) (TaskRunReviewRequestRecord, error)
	listTaskRunReviewsFn         func(context.Context, TaskRunReviewListQuery) ([]TaskRunReviewRecord, error)
	getTaskRunReviewFn           func(context.Context, string) (TaskRunReviewRecord, error)
	submitTaskRunReviewVerdictFn func(
		context.Context,
		string,
		*TaskRunReviewVerdictRequest,
	) (TaskRunReviewVerdictRecord, error)
	submitTaskRunReviewVerdictAsAgentFn func(
		context.Context,
		string,
		*TaskRunReviewVerdictRequest,
		agentidentity.Credentials,
	) (TaskRunReviewVerdictRecord, error)
	publishTaskFn          func(context.Context, string, TaskExecutionRequest) (TaskExecutionRecord, error)
	startTaskFn            func(context.Context, string, TaskExecutionRequest) (TaskExecutionRecord, error)
	approveTaskFn          func(context.Context, string, TaskExecutionRequest) (TaskExecutionRecord, error)
	rejectTaskFn           func(context.Context, string) (TaskRecord, error)
	cancelTaskFn           func(context.Context, string, CancelTaskRequest) (TaskRecord, error)
	createChildTaskFn      func(context.Context, string, CreateTaskChildRequest) (TaskRecord, error)
	addTaskDependencyFn    func(context.Context, string, AddTaskDependencyRequest) (TaskDetailRecord, error)
	removeTaskDependencyFn func(context.Context, string, string) (TaskDetailRecord, error)
	enqueueTaskRunFn       func(context.Context, string, EnqueueTaskRunRequest) (TaskRunRecord, error)
	listTaskRunsFn         func(context.Context, string, TaskRunListQuery) ([]TaskRunRecord, error)
	claimTaskRunFn         func(context.Context, string, ClaimTaskRunRequest) (TaskRunRecord, error)
	startTaskRunFn         func(context.Context, string, StartTaskRunRequest) (TaskRunRecord, error)
	attachTaskRunSessionFn func(context.Context, string, AttachTaskRunSessionRequest) (TaskRunRecord, error)
	completeTaskRunFn      func(context.Context, string, CompleteTaskRunRequest) (TaskRunRecord, error)
	failTaskRunFn          func(context.Context, string, FailTaskRunRequest) (TaskRunRecord, error)
	cancelTaskRunFn        func(context.Context, string, CancelTaskRunRequest) (TaskRunRecord, error)
	agentMeFn              func(context.Context, agentidentity.Credentials) (AgentMeRecord, error)
	agentContextFn         func(context.Context, agentidentity.Credentials) (AgentContextRecord, error)
	agentSpawnFn           func(context.Context, AgentSpawnRequest, agentidentity.Credentials) (AgentSpawnRecord, error)
	agentChannelsFn        func(context.Context, agentidentity.Credentials) ([]AgentChannelRecord, error)
	agentChannelRecvFn     func(context.Context, string, AgentChannelRecvQuery, agentidentity.Credentials) ([]AgentChannelMessageRecord, error)
	agentChannelSendFn     func(context.Context, string, AgentChannelSendRequest, agentidentity.Credentials) (AgentChannelMessageRecord, error)
	agentChannelReplyFn    func(context.Context, AgentChannelReplyRequest, agentidentity.Credentials) (AgentChannelMessageRecord, error)
	agentTaskClaimNextFn   func(context.Context, AgentTaskClaimNextRequest, agentidentity.Credentials) (AgentTaskNextRecord, error)
	agentTaskHeartbeatFn   func(context.Context, string, AgentTaskHeartbeatRequest, agentidentity.Credentials) (AgentTaskLeaseRecord, error)
	agentTaskCompleteFn    func(context.Context, string, AgentTaskCompleteRequest, agentidentity.Credentials) (AgentTaskLeaseRecord, error)
	agentTaskFailFn        func(context.Context, string, AgentTaskFailRequest, agentidentity.Credentials) (AgentTaskLeaseRecord, error)
	agentTaskReleaseFn     func(context.Context, string, AgentTaskReleaseRequest, agentidentity.Credentials) (AgentTaskLeaseRecord, error)
}

var _ DaemonClient = (*stubClient)(nil)

func (s *stubClient) Status(ctx context.Context) (StatusRecord, error) {
	if s.statusFn != nil {
		return s.statusFn(ctx)
	}
	return StatusRecord{}, errors.New("unexpected Status call")
}

func (s *stubClient) Doctor(ctx context.Context, query DoctorQuery) (DoctorRecord, error) {
	if s.doctorFn != nil {
		return s.doctorFn(ctx, query)
	}
	return DoctorRecord{}, errors.New("unexpected Doctor call")
}

func (s *stubClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	if s.daemonStatusFn != nil {
		return s.daemonStatusFn(ctx)
	}
	return DaemonStatus{}, errors.New("unexpected DaemonStatus call")
}

func (s *stubClient) TriggerSettingsRestart(ctx context.Context) (SettingsRestartActionRecord, error) {
	if s.triggerSettingsRestartFn != nil {
		return s.triggerSettingsRestartFn(ctx)
	}
	return SettingsRestartActionRecord{}, errors.New("unexpected TriggerSettingsRestart call")
}

func (s *stubClient) GetSettingsRestartStatus(
	ctx context.Context,
	operationID string,
) (SettingsRestartStatusRecord, error) {
	if s.getSettingsRestartStatusFn != nil {
		return s.getSettingsRestartStatusFn(ctx, operationID)
	}
	return SettingsRestartStatusRecord{}, errors.New("unexpected GetSettingsRestartStatus call")
}

func (s *stubClient) CreateSupportBundle(
	ctx context.Context,
	request CreateSupportBundleRequest,
) (SupportBundleOperationRecord, error) {
	if s.createSupportBundleFn != nil {
		return s.createSupportBundleFn(ctx, request)
	}
	return SupportBundleOperationRecord{}, errors.New("unexpected CreateSupportBundle call")
}

func (s *stubClient) GetSupportBundle(
	ctx context.Context,
	operationID string,
) (SupportBundleOperationRecord, error) {
	if s.getSupportBundleFn != nil {
		return s.getSupportBundleFn(ctx, operationID)
	}
	return SupportBundleOperationRecord{}, errors.New("unexpected GetSupportBundle call")
}

func (s *stubClient) DownloadSupportBundle(
	ctx context.Context,
	operationID string,
	dst io.Writer,
) error {
	if s.downloadSupportBundleFn != nil {
		return s.downloadSupportBundleFn(ctx, operationID, dst)
	}
	return errors.New("unexpected DownloadSupportBundle call")
}

func (s *stubClient) GetSettingsUpdate(ctx context.Context) (SettingsUpdateRecord, error) {
	if s.getSettingsUpdateFn != nil {
		return s.getSettingsUpdateFn(ctx)
	}
	return SettingsUpdateRecord{}, errors.New("unexpected GetSettingsUpdate call")
}

func (s *stubClient) UpdateSettingsSkills(
	ctx context.Context,
	request UpdateSettingsSkillsRequest,
) (SettingsMutationRecord, error) {
	if s.updateSettingsSkillsFn != nil {
		return s.updateSettingsSkillsFn(ctx, request)
	}
	return SettingsMutationRecord{}, errors.New("unexpected UpdateSettingsSkills call")
}

func (s *stubClient) ReloadSettings(ctx context.Context) (SettingsMutationRecord, error) {
	if s.reloadSettingsFn != nil {
		return s.reloadSettingsFn(ctx)
	}
	return SettingsMutationRecord{}, errors.New("unexpected ReloadSettings call")
}

func (s *stubClient) ListSettingsApplyRecords(
	ctx context.Context,
	query SettingsApplyHistoryQuery,
) (SettingsApplyHistoryRecord, error) {
	if s.listSettingsApplyRecordsFn != nil {
		return s.listSettingsApplyRecordsFn(ctx, query)
	}
	return SettingsApplyHistoryRecord{}, errors.New("unexpected ListSettingsApplyRecords call")
}

func (s *stubClient) ListProviders(ctx context.Context) (contract.ProviderListResponse, error) {
	if s.listProvidersFn != nil {
		return s.listProvidersFn(ctx)
	}
	return contract.ProviderListResponse{}, errors.New("unexpected ListProviders call")
}

func (s *stubClient) ProbeProviderAuth(
	ctx context.Context,
	providerID string,
) (contract.ProviderAuthProbeResponse, error) {
	if s.probeProviderAuthFn != nil {
		return s.probeProviderAuthFn(ctx, providerID)
	}
	return contract.ProviderAuthProbeResponse{}, errors.New("unexpected ProbeProviderAuth call")
}

func (s *stubClient) ListProviderModels(
	ctx context.Context,
	query ProviderModelListQuery,
) (ProviderModelListRecord, error) {
	if s.listProviderModelsFn != nil {
		return s.listProviderModelsFn(ctx, query)
	}
	return ProviderModelListRecord{}, errors.New("unexpected ListProviderModels call")
}

func (s *stubClient) RefreshProviderModels(
	ctx context.Context,
	providerID string,
	request ProviderModelRefreshRequest,
) (ProviderModelRefreshRecord, error) {
	if s.refreshProviderModelsFn != nil {
		return s.refreshProviderModelsFn(ctx, providerID, request)
	}
	return ProviderModelRefreshRecord{}, errors.New("unexpected RefreshProviderModels call")
}

func (s *stubClient) ProviderModelStatus(
	ctx context.Context,
	providerID string,
) (ProviderModelStatusRecord, error) {
	if s.providerModelStatusFn != nil {
		return s.providerModelStatusFn(ctx, providerID)
	}
	return ProviderModelStatusRecord{}, errors.New("unexpected ProviderModelStatus call")
}

func (s *stubClient) ListVaultSecrets(
	ctx context.Context,
	query VaultListQuery,
) ([]VaultRecord, error) {
	if s.listVaultSecretsFn != nil {
		return s.listVaultSecretsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListVaultSecrets call")
}

func (s *stubClient) GetVaultSecret(ctx context.Context, ref string) (VaultRecord, error) {
	if s.getVaultSecretFn != nil {
		return s.getVaultSecretFn(ctx, ref)
	}
	return VaultRecord{}, errors.New("unexpected GetVaultSecret call")
}

func (s *stubClient) PutVaultSecret(
	ctx context.Context,
	request PutVaultSecretRequest,
) (VaultRecord, error) {
	if s.putVaultSecretFn != nil {
		return s.putVaultSecretFn(ctx, request)
	}
	return VaultRecord{}, errors.New("unexpected PutVaultSecret call")
}

func (s *stubClient) DeleteVaultSecret(ctx context.Context, ref string) error {
	if s.deleteVaultSecretFn != nil {
		return s.deleteVaultSecretFn(ctx, ref)
	}
	return errors.New("unexpected DeleteVaultSecret call")
}

func (s *stubClient) NetworkStatus(ctx context.Context) (NetworkStatusRecord, error) {
	if s.networkStatusFn != nil {
		return s.networkStatusFn(ctx)
	}
	return NetworkStatusRecord{}, errors.New("unexpected NetworkStatus call")
}

func (s *stubClient) NetworkPeers(
	ctx context.Context,
	query NetworkPeersQuery,
) ([]NetworkPeerRecord, error) {
	if s.networkPeersFn != nil {
		return s.networkPeersFn(ctx, query)
	}
	return nil, errors.New("unexpected NetworkPeers call")
}

func (s *stubClient) NetworkChannels(ctx context.Context, workspaceRef string) ([]NetworkChannelRecord, error) {
	if s.networkChannelsFn != nil {
		return s.networkChannelsFn(ctx, workspaceRef)
	}
	return nil, errors.New("unexpected NetworkChannels call")
}

func (s *stubClient) CreateNetworkChannel(
	ctx context.Context,
	workspaceRef string,
	request CreateNetworkChannelRequest,
) (NetworkChannelDetailRecord, error) {
	if s.createNetworkChannelFn != nil {
		return s.createNetworkChannelFn(ctx, workspaceRef, request)
	}
	return NetworkChannelDetailRecord{}, errors.New("unexpected CreateNetworkChannel call")
}

func (s *stubClient) NetworkThreads(
	ctx context.Context,
	query NetworkThreadsQuery,
) ([]NetworkThreadRecord, error) {
	if s.networkThreadsFn != nil {
		return s.networkThreadsFn(ctx, query)
	}
	return nil, errors.New("unexpected NetworkThreads call")
}

func (s *stubClient) NetworkThread(
	ctx context.Context,
	workspaceRef string,
	channel string,
	threadID string,
) (NetworkThreadRecord, error) {
	if s.networkThreadFn != nil {
		return s.networkThreadFn(ctx, workspaceRef, channel, threadID)
	}
	return NetworkThreadRecord{}, errors.New("unexpected NetworkThread call")
}

func (s *stubClient) NetworkThreadMessages(
	ctx context.Context,
	query NetworkConversationMessagesQuery,
) ([]NetworkConversationMessageRecord, error) {
	if s.networkThreadMessagesFn != nil {
		return s.networkThreadMessagesFn(ctx, query)
	}
	return nil, errors.New("unexpected NetworkThreadMessages call")
}

func (s *stubClient) NetworkDirects(
	ctx context.Context,
	query NetworkDirectsQuery,
) ([]NetworkDirectRoomRecord, error) {
	if s.networkDirectsFn != nil {
		return s.networkDirectsFn(ctx, query)
	}
	return nil, errors.New("unexpected NetworkDirects call")
}

func (s *stubClient) NetworkDirectResolve(
	ctx context.Context,
	workspaceRef string,
	channel string,
	request NetworkDirectResolveRequest,
) (NetworkDirectRoomRecord, error) {
	if s.networkDirectResolveFn != nil {
		return s.networkDirectResolveFn(ctx, workspaceRef, channel, request)
	}
	return NetworkDirectRoomRecord{}, errors.New("unexpected NetworkDirectResolve call")
}

func (s *stubClient) NetworkDirect(
	ctx context.Context,
	workspaceRef string,
	channel string,
	directID string,
) (NetworkDirectRoomRecord, error) {
	if s.networkDirectFn != nil {
		return s.networkDirectFn(ctx, workspaceRef, channel, directID)
	}
	return NetworkDirectRoomRecord{}, errors.New("unexpected NetworkDirect call")
}

func (s *stubClient) NetworkDirectMessages(
	ctx context.Context,
	query NetworkConversationMessagesQuery,
) ([]NetworkConversationMessageRecord, error) {
	if s.networkDirectMessagesFn != nil {
		return s.networkDirectMessagesFn(ctx, query)
	}
	return nil, errors.New("unexpected NetworkDirectMessages call")
}

func (s *stubClient) NetworkWork(ctx context.Context, workspaceRef string, workID string) (NetworkWorkRecord, error) {
	if s.networkWorkFn != nil {
		return s.networkWorkFn(ctx, workspaceRef, workID)
	}
	return NetworkWorkRecord{}, errors.New("unexpected NetworkWork call")
}

func (s *stubClient) NetworkSend(
	ctx context.Context,
	request NetworkSendRequest,
) (NetworkSendRecord, error) {
	if s.networkSendFn != nil {
		return s.networkSendFn(ctx, request)
	}
	return NetworkSendRecord{}, errors.New("unexpected NetworkSend call")
}

func (s *stubClient) NetworkInbox(
	ctx context.Context,
	workspaceRef string,
	sessionID string,
) ([]NetworkEnvelopeRecord, error) {
	if s.networkInboxFn != nil {
		return s.networkInboxFn(ctx, workspaceRef, sessionID)
	}
	return nil, errors.New("unexpected NetworkInbox call")
}

func (s *stubClient) ListExtensions(ctx context.Context) ([]ExtensionRecord, error) {
	if s.listExtensionsFn != nil {
		return s.listExtensionsFn(ctx)
	}
	return nil, errors.New("unexpected ListExtensions call")
}

func (s *stubClient) InstallExtension(
	ctx context.Context,
	request InstallExtensionRequest,
) (ExtensionRecord, error) {
	if s.installExtensionFn != nil {
		return s.installExtensionFn(ctx, request)
	}
	return ExtensionRecord{}, errors.New("unexpected InstallExtension call")
}

func (s *stubClient) EnableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.enableExtensionFn != nil {
		return s.enableExtensionFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected EnableExtension call")
}

func (s *stubClient) DisableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.disableExtensionFn != nil {
		return s.disableExtensionFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected DisableExtension call")
}

func (s *stubClient) ExtensionStatus(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.extensionStatusFn != nil {
		return s.extensionStatusFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected ExtensionStatus call")
}

func (s *stubClient) ListBundleCatalog(ctx context.Context) ([]BundleCatalogRecord, error) {
	if s.listBundleCatalogFn != nil {
		return s.listBundleCatalogFn(ctx)
	}
	return nil, errors.New("unexpected ListBundleCatalog call")
}

func (s *stubClient) PreviewBundleActivation(
	ctx context.Context,
	request ActivateBundleRequest,
) (BundleActivationRecord, error) {
	if s.previewBundleActivationFn != nil {
		return s.previewBundleActivationFn(ctx, request)
	}
	return BundleActivationRecord{}, errors.New("unexpected PreviewBundleActivation call")
}

func (s *stubClient) ActivateBundle(
	ctx context.Context,
	request ActivateBundleRequest,
) (BundleActivationRecord, error) {
	if s.activateBundleFn != nil {
		return s.activateBundleFn(ctx, request)
	}
	return BundleActivationRecord{}, errors.New("unexpected ActivateBundle call")
}

func (s *stubClient) ListBundleActivations(ctx context.Context) ([]BundleActivationRecord, error) {
	if s.listBundleActivationsFn != nil {
		return s.listBundleActivationsFn(ctx)
	}
	return nil, errors.New("unexpected ListBundleActivations call")
}

func (s *stubClient) GetBundleActivation(ctx context.Context, id string) (BundleActivationRecord, error) {
	if s.getBundleActivationFn != nil {
		return s.getBundleActivationFn(ctx, id)
	}
	return BundleActivationRecord{}, errors.New("unexpected GetBundleActivation call")
}

func (s *stubClient) UpdateBundleActivation(
	ctx context.Context,
	id string,
	request UpdateBundleActivationRequest,
) (BundleActivationRecord, error) {
	if s.updateBundleActivationFn != nil {
		return s.updateBundleActivationFn(ctx, id, request)
	}
	return BundleActivationRecord{}, errors.New("unexpected UpdateBundleActivation call")
}

func (s *stubClient) DeactivateBundle(ctx context.Context, id string) error {
	if s.deactivateBundleFn != nil {
		return s.deactivateBundleFn(ctx, id)
	}
	return errors.New("unexpected DeactivateBundle call")
}

func (s *stubClient) BundleNetworkSettings(ctx context.Context) (BundleNetworkSettingsRecord, error) {
	if s.bundleNetworkSettingsFn != nil {
		return s.bundleNetworkSettingsFn(ctx)
	}
	return BundleNetworkSettingsRecord{}, errors.New("unexpected BundleNetworkSettings call")
}

func (s *stubClient) ListBridges(ctx context.Context) ([]BridgeRecord, error) {
	if s.listBridgesFn != nil {
		return s.listBridgesFn(ctx)
	}
	return nil, errors.New("unexpected ListBridges call")
}

func (s *stubClient) CreateBridge(
	ctx context.Context,
	request CreateBridgeRequest,
) (BridgeRecord, error) {
	if s.createBridgeFn != nil {
		return s.createBridgeFn(ctx, request)
	}
	return BridgeRecord{}, errors.New("unexpected CreateBridge call")
}

func (s *stubClient) GetBridge(ctx context.Context, id string) (BridgeRecord, error) {
	if s.getBridgeFn != nil {
		return s.getBridgeFn(ctx, id)
	}
	return BridgeRecord{}, errors.New("unexpected GetBridge call")
}

func (s *stubClient) UpdateBridge(
	ctx context.Context,
	id string,
	request UpdateBridgeRequest,
) (BridgeRecord, error) {
	if s.updateBridgeFn != nil {
		return s.updateBridgeFn(ctx, id, request)
	}
	return BridgeRecord{}, errors.New("unexpected UpdateBridge call")
}

func (s *stubClient) EnableBridge(ctx context.Context, id string) (BridgeRecord, error) {
	if s.enableBridgeFn != nil {
		return s.enableBridgeFn(ctx, id)
	}
	return BridgeRecord{}, errors.New("unexpected EnableBridge call")
}

func (s *stubClient) DisableBridge(ctx context.Context, id string) (BridgeRecord, error) {
	if s.disableBridgeFn != nil {
		return s.disableBridgeFn(ctx, id)
	}
	return BridgeRecord{}, errors.New("unexpected DisableBridge call")
}

func (s *stubClient) RestartBridge(ctx context.Context, id string) (BridgeRecord, error) {
	if s.restartBridgeFn != nil {
		return s.restartBridgeFn(ctx, id)
	}
	return BridgeRecord{}, errors.New("unexpected RestartBridge call")
}

func (s *stubClient) BridgeRoutes(ctx context.Context, id string) ([]BridgeRouteRecord, error) {
	if s.bridgeRoutesFn != nil {
		return s.bridgeRoutesFn(ctx, id)
	}
	return nil, errors.New("unexpected BridgeRoutes call")
}

func (s *stubClient) ListBridgeSecretBindings(
	ctx context.Context,
	id string,
) ([]BridgeSecretBindingRecord, error) {
	if s.listBridgeSecretBindingsFn != nil {
		return s.listBridgeSecretBindingsFn(ctx, id)
	}
	return nil, errors.New("unexpected ListBridgeSecretBindings call")
}

func (s *stubClient) PutBridgeSecretBinding(
	ctx context.Context,
	id string,
	bindingName string,
	request BridgeSecretBindingRequest,
) (BridgeSecretBindingRecord, error) {
	if s.putBridgeSecretBindingFn != nil {
		return s.putBridgeSecretBindingFn(ctx, id, bindingName, request)
	}
	return BridgeSecretBindingRecord{}, errors.New("unexpected PutBridgeSecretBinding call")
}

func (s *stubClient) DeleteBridgeSecretBinding(ctx context.Context, id string, bindingName string) error {
	if s.deleteBridgeSecretBindingFn != nil {
		return s.deleteBridgeSecretBindingFn(ctx, id, bindingName)
	}
	return errors.New("unexpected DeleteBridgeSecretBinding call")
}

func (s *stubClient) TestBridgeDelivery(
	ctx context.Context,
	id string,
	request BridgeTestDeliveryRequest,
) (BridgeTestDeliveryRecord, error) {
	if s.testBridgeDeliveryFn != nil {
		return s.testBridgeDeliveryFn(ctx, id, request)
	}
	return BridgeTestDeliveryRecord{}, errors.New("unexpected TestBridgeDelivery call")
}

func (s *stubClient) ListSessions(
	ctx context.Context,
	query SessionListQuery,
) ([]SessionRecord, error) {
	if s.listSessionsFn != nil {
		return s.listSessionsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListSessions call")
}

func (s *stubClient) CreateSession(
	ctx context.Context,
	request CreateSessionRequest,
) (SessionRecord, error) {
	if s.createSessionFn != nil {
		return s.createSessionFn(ctx, request)
	}
	return SessionRecord{}, errors.New("unexpected CreateSession call")
}

func (s *stubClient) GetSession(ctx context.Context, id string) (SessionRecord, error) {
	if s.getSessionFn != nil {
		return s.getSessionFn(ctx, id)
	}
	return SessionRecord{}, errors.New("unexpected GetSession call")
}

func (s *stubClient) GetSessionHealth(ctx context.Context, id string) (SessionHealthRecord, error) {
	if s.getSessionHealthFn != nil {
		return s.getSessionHealthFn(ctx, id)
	}
	return SessionHealthRecord{}, errors.New("unexpected GetSessionHealth call")
}

func (s *stubClient) GetSessionStatus(ctx context.Context, id string) (SessionStatusRecord, error) {
	if s.getSessionStatusFn != nil {
		return s.getSessionStatusFn(ctx, id)
	}
	return SessionStatusRecord{}, errors.New("unexpected GetSessionStatus call")
}

func (s *stubClient) InspectSession(
	ctx context.Context,
	id string,
	query SessionInspectQuery,
) (SessionInspectRecord, error) {
	if s.inspectSessionFn != nil {
		return s.inspectSessionFn(ctx, id, query)
	}
	return SessionInspectRecord{}, errors.New("unexpected InspectSession call")
}

func (s *stubClient) RefreshSessionSoul(
	ctx context.Context,
	id string,
	request SessionSoulRefreshRequest,
) (AgentSoulRecord, error) {
	if s.refreshSessionSoulFn != nil {
		return s.refreshSessionSoulFn(ctx, id, request)
	}
	return AgentSoulRecord{}, errors.New("unexpected RefreshSessionSoul call")
}

func (s *stubClient) StopSession(ctx context.Context, id string) error {
	if s.stopSessionFn != nil {
		return s.stopSessionFn(ctx, id)
	}
	return errors.New("unexpected StopSession call")
}

func (s *stubClient) ResumeSession(ctx context.Context, id string) (SessionRecord, error) {
	if s.resumeSessionFn != nil {
		return s.resumeSessionFn(ctx, id)
	}
	return SessionRecord{}, errors.New("unexpected ResumeSession call")
}

func (s *stubClient) SessionRecap(ctx context.Context, id string, limit int) (SessionRecapRecord, error) {
	if s.sessionRecapFn != nil {
		return s.sessionRecapFn(ctx, id, limit)
	}
	return SessionRecapRecord{}, errors.New("unexpected SessionRecap call")
}

func (s *stubClient) RepairSession(
	ctx context.Context,
	id string,
	query SessionRepairQuery,
) (SessionRepairRecord, error) {
	if s.repairSessionFn != nil {
		return s.repairSessionFn(ctx, id, query)
	}
	return SessionRepairRecord{}, errors.New("unexpected RepairSession call")
}

func (s *stubClient) ApproveSession(
	ctx context.Context,
	id string,
	request SessionApprovalRequest,
) (SessionApprovalRecord, error) {
	if s.approveSessionFn != nil {
		return s.approveSessionFn(ctx, id, request)
	}
	return SessionApprovalRecord{}, errors.New("unexpected ApproveSession call")
}

func (s *stubClient) PromptSession(
	ctx context.Context,
	id string,
	message string,
) ([]AgentEventRecord, error) {
	if s.promptSessionFn != nil {
		return s.promptSessionFn(ctx, id, message)
	}
	return nil, errors.New("unexpected PromptSession call")
}

func (s *stubClient) StreamPromptSession(
	ctx context.Context,
	id string,
	message string,
	handler SSEHandler,
) error {
	if s.streamPromptSessionFn != nil {
		return s.streamPromptSessionFn(ctx, id, message, handler)
	}
	return errors.New("unexpected StreamPromptSession call")
}

func (s *stubClient) SessionEvents(
	ctx context.Context,
	id string,
	query SessionEventQuery,
) ([]SessionEventRecord, error) {
	if s.sessionEventsFn != nil {
		return s.sessionEventsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected SessionEvents call")
}

func (s *stubClient) StreamSessionEvents(
	ctx context.Context,
	id string,
	query SessionEventQuery,
	lastEventID string,
	handler SSEHandler,
) error {
	if s.streamSessionFn != nil {
		return s.streamSessionFn(ctx, id, query, lastEventID, handler)
	}
	return errors.New("unexpected StreamSessionEvents call")
}

func (s *stubClient) SessionHistory(
	ctx context.Context,
	id string,
	query SessionEventQuery,
) ([]TurnHistoryRecord, error) {
	if s.sessionHistoryFn != nil {
		return s.sessionHistoryFn(ctx, id, query)
	}
	return nil, errors.New("unexpected SessionHistory call")
}

func (s *stubClient) CreateWorkspace(
	ctx context.Context,
	request WorkspaceCreateRequest,
) (WorkspaceRecord, error) {
	if s.createWorkspaceFn != nil {
		return s.createWorkspaceFn(ctx, request)
	}
	return WorkspaceRecord{}, errors.New("unexpected CreateWorkspace call")
}

func (s *stubClient) ListWorkspaces(ctx context.Context) ([]WorkspaceRecord, error) {
	if s.listWorkspacesFn != nil {
		return s.listWorkspacesFn(ctx)
	}
	return nil, errors.New("unexpected ListWorkspaces call")
}

func (s *stubClient) GetWorkspace(ctx context.Context, ref string) (WorkspaceDetailRecord, error) {
	if s.getWorkspaceFn != nil {
		return s.getWorkspaceFn(ctx, ref)
	}
	return WorkspaceDetailRecord{}, errors.New("unexpected GetWorkspace call")
}

func (s *stubClient) UpdateWorkspace(
	ctx context.Context,
	ref string,
	request WorkspaceUpdateRequest,
) (WorkspaceRecord, error) {
	if s.updateWorkspaceFn != nil {
		return s.updateWorkspaceFn(ctx, ref, request)
	}
	return WorkspaceRecord{}, errors.New("unexpected UpdateWorkspace call")
}

func (s *stubClient) DeleteWorkspace(ctx context.Context, ref string) error {
	if s.deleteWorkspaceFn != nil {
		return s.deleteWorkspaceFn(ctx, ref)
	}
	return errors.New("unexpected DeleteWorkspace call")
}

func (s *stubClient) ListAgents(ctx context.Context, query AgentQuery) ([]AgentRecord, error) {
	if s.listAgentsFn != nil {
		return s.listAgentsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAgents call")
}

func (s *stubClient) GetAgent(ctx context.Context, name string, query AgentQuery) (AgentRecord, error) {
	if s.getAgentFn != nil {
		return s.getAgentFn(ctx, name, query)
	}
	return AgentRecord{}, errors.New("unexpected GetAgent call")
}

func (s *stubClient) GetAgentSoul(
	ctx context.Context,
	name string,
	query AgentQuery,
) (AgentSoulRecord, error) {
	if s.getAgentSoulFn != nil {
		return s.getAgentSoulFn(ctx, name, query)
	}
	return AgentSoulRecord{}, errors.New("unexpected GetAgentSoul call")
}

func (s *stubClient) ValidateAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulValidateRequest,
) (AgentSoulRecord, error) {
	if s.validateAgentSoulFn != nil {
		return s.validateAgentSoulFn(ctx, name, request)
	}
	return AgentSoulRecord{}, errors.New("unexpected ValidateAgentSoul call")
}

func (s *stubClient) PutAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulPutRequest,
) (AgentSoulMutationRecord, error) {
	if s.putAgentSoulFn != nil {
		return s.putAgentSoulFn(ctx, name, request)
	}
	return AgentSoulMutationRecord{}, errors.New("unexpected PutAgentSoul call")
}

func (s *stubClient) DeleteAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulDeleteRequest,
) (AgentSoulMutationRecord, error) {
	if s.deleteAgentSoulFn != nil {
		return s.deleteAgentSoulFn(ctx, name, request)
	}
	return AgentSoulMutationRecord{}, errors.New("unexpected DeleteAgentSoul call")
}

func (s *stubClient) ListAgentSoulHistory(
	ctx context.Context,
	name string,
	request AgentSoulHistoryRequest,
) (AgentSoulHistoryRecord, error) {
	if s.listAgentSoulHistoryFn != nil {
		return s.listAgentSoulHistoryFn(ctx, name, request)
	}
	return AgentSoulHistoryRecord{}, errors.New("unexpected ListAgentSoulHistory call")
}

func (s *stubClient) RollbackAgentSoul(
	ctx context.Context,
	name string,
	request AgentSoulRollbackRequest,
) (AgentSoulMutationRecord, error) {
	if s.rollbackAgentSoulFn != nil {
		return s.rollbackAgentSoulFn(ctx, name, request)
	}
	return AgentSoulMutationRecord{}, errors.New("unexpected RollbackAgentSoul call")
}

func (s *stubClient) GetAgentHeartbeat(
	ctx context.Context,
	name string,
	query AgentQuery,
) (AgentHeartbeatRecord, error) {
	if s.getAgentHeartbeatFn != nil {
		return s.getAgentHeartbeatFn(ctx, name, query)
	}
	return AgentHeartbeatRecord{}, errors.New("unexpected GetAgentHeartbeat call")
}

func (s *stubClient) ValidateAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatValidateRequest,
) (AgentHeartbeatRecord, error) {
	if s.validateAgentHeartbeatFn != nil {
		return s.validateAgentHeartbeatFn(ctx, name, request)
	}
	return AgentHeartbeatRecord{}, errors.New("unexpected ValidateAgentHeartbeat call")
}

func (s *stubClient) PutAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatPutRequest,
) (AgentHeartbeatMutationRecord, error) {
	if s.putAgentHeartbeatFn != nil {
		return s.putAgentHeartbeatFn(ctx, name, request)
	}
	return AgentHeartbeatMutationRecord{}, errors.New("unexpected PutAgentHeartbeat call")
}

func (s *stubClient) DeleteAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatDeleteRequest,
) (AgentHeartbeatMutationRecord, error) {
	if s.deleteAgentHeartbeatFn != nil {
		return s.deleteAgentHeartbeatFn(ctx, name, request)
	}
	return AgentHeartbeatMutationRecord{}, errors.New("unexpected DeleteAgentHeartbeat call")
}

func (s *stubClient) ListAgentHeartbeatHistory(
	ctx context.Context,
	name string,
	request AgentHeartbeatHistoryRequest,
) (AgentHeartbeatHistoryRecord, error) {
	if s.listAgentHeartbeatHistoryFn != nil {
		return s.listAgentHeartbeatHistoryFn(ctx, name, request)
	}
	return AgentHeartbeatHistoryRecord{}, errors.New("unexpected ListAgentHeartbeatHistory call")
}

func (s *stubClient) RollbackAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatRollbackRequest,
) (AgentHeartbeatMutationRecord, error) {
	if s.rollbackAgentHeartbeatFn != nil {
		return s.rollbackAgentHeartbeatFn(ctx, name, request)
	}
	return AgentHeartbeatMutationRecord{}, errors.New("unexpected RollbackAgentHeartbeat call")
}

func (s *stubClient) GetAgentHeartbeatStatus(
	ctx context.Context,
	name string,
	request AgentHeartbeatStatusRequest,
) (AgentHeartbeatStatusRecord, error) {
	if s.getAgentHeartbeatStatusFn != nil {
		return s.getAgentHeartbeatStatusFn(ctx, name, request)
	}
	return AgentHeartbeatStatusRecord{}, errors.New("unexpected GetAgentHeartbeatStatus call")
}

func (s *stubClient) WakeAgentHeartbeat(
	ctx context.Context,
	name string,
	request AgentHeartbeatWakeRequest,
) (AgentHeartbeatWakeDecisionRecord, error) {
	if s.wakeAgentHeartbeatFn != nil {
		return s.wakeAgentHeartbeatFn(ctx, name, request)
	}
	return AgentHeartbeatWakeDecisionRecord{}, errors.New("unexpected WakeAgentHeartbeat call")
}

func (s *stubClient) ListResources(ctx context.Context, query ResourceListQuery) ([]ResourceRecord, error) {
	if s.listResourcesFn != nil {
		return s.listResourcesFn(ctx, query)
	}
	return nil, errors.New("unexpected ListResources call")
}

func (s *stubClient) GetResource(ctx context.Context, kind string, id string) (ResourceRecord, error) {
	if s.getResourceFn != nil {
		return s.getResourceFn(ctx, kind, id)
	}
	return ResourceRecord{}, errors.New("unexpected GetResource call")
}

func (s *stubClient) PutResource(
	ctx context.Context,
	kind string,
	id string,
	request ResourcePutRequest,
) (ResourceRecord, error) {
	if s.putResourceFn != nil {
		return s.putResourceFn(ctx, kind, id, request)
	}
	return ResourceRecord{}, errors.New("unexpected PutResource call")
}

func (s *stubClient) DeleteResource(
	ctx context.Context,
	kind string,
	id string,
	request ResourceDeleteRequest,
) error {
	if s.deleteResourceFn != nil {
		return s.deleteResourceFn(ctx, kind, id, request)
	}
	return errors.New("unexpected DeleteResource call")
}

func (s *stubClient) ListSkills(ctx context.Context, query SkillQuery) ([]SkillRecord, error) {
	if s.listSkillsFn != nil {
		return s.listSkillsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListSkills call")
}

func (s *stubClient) GetSkill(ctx context.Context, name string, query SkillQuery) (SkillRecord, error) {
	if s.getSkillFn != nil {
		return s.getSkillFn(ctx, name, query)
	}
	return SkillRecord{}, errors.New("unexpected GetSkill call")
}

func (s *stubClient) GetSkillContent(ctx context.Context, name string, query SkillQuery) (string, error) {
	if s.getSkillContentFn != nil {
		return s.getSkillContentFn(ctx, name, query)
	}
	return "", errors.New("unexpected GetSkillContent call")
}

func (s *stubClient) EnableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error) {
	if s.enableSkillFn != nil {
		return s.enableSkillFn(ctx, name, query)
	}
	return SkillActionRecord{}, errors.New("unexpected EnableSkill call")
}

func (s *stubClient) DisableSkill(ctx context.Context, name string, query SkillQuery) (SkillActionRecord, error) {
	if s.disableSkillFn != nil {
		return s.disableSkillFn(ctx, name, query)
	}
	return SkillActionRecord{}, errors.New("unexpected DisableSkill call")
}

func (s *stubClient) ListTools(ctx context.Context, query ToolQuery) (ToolsResponseRecord, error) {
	if s.listToolsFn != nil {
		return s.listToolsFn(ctx, query)
	}
	return ToolsResponseRecord{}, errors.New("unexpected ListTools call")
}

func (s *stubClient) SearchTools(
	ctx context.Context,
	request ToolSearchRequest,
) (ToolsResponseRecord, error) {
	if s.searchToolsFn != nil {
		return s.searchToolsFn(ctx, request)
	}
	return ToolsResponseRecord{}, errors.New("unexpected SearchTools call")
}

func (s *stubClient) GetTool(
	ctx context.Context,
	id string,
	query ToolQuery,
) (ToolResponseRecord, error) {
	if s.getToolFn != nil {
		return s.getToolFn(ctx, id, query)
	}
	return ToolResponseRecord{}, errors.New("unexpected GetTool call")
}

func (s *stubClient) CreateToolApproval(
	ctx context.Context,
	id string,
	request ToolApprovalRequest,
) (ToolApprovalRecord, error) {
	if s.createToolApprovalFn != nil {
		return s.createToolApprovalFn(ctx, id, request)
	}
	return ToolApprovalRecord{}, errors.New("unexpected CreateToolApproval call")
}

func (s *stubClient) InvokeTool(
	ctx context.Context,
	id string,
	request ToolInvokeRequest,
) (ToolInvokeResponseRecord, error) {
	if s.invokeToolFn != nil {
		return s.invokeToolFn(ctx, id, request)
	}
	return ToolInvokeResponseRecord{}, errors.New("unexpected InvokeTool call")
}

func (s *stubClient) ListToolsets(ctx context.Context, query ToolQuery) (ToolsetsResponseRecord, error) {
	if s.listToolsetsFn != nil {
		return s.listToolsetsFn(ctx, query)
	}
	return ToolsetsResponseRecord{}, errors.New("unexpected ListToolsets call")
}

func (s *stubClient) GetToolset(
	ctx context.Context,
	id string,
	query ToolQuery,
) (ToolsetResponseRecord, error) {
	if s.getToolsetFn != nil {
		return s.getToolsetFn(ctx, id, query)
	}
	return ToolsetResponseRecord{}, errors.New("unexpected GetToolset call")
}

func (s *stubClient) HookCatalog(
	ctx context.Context,
	query HookCatalogQuery,
) ([]HookCatalogRecord, error) {
	if s.hookCatalogFn != nil {
		return s.hookCatalogFn(ctx, query)
	}
	return nil, errors.New("unexpected HookCatalog call")
}

func (s *stubClient) HookRuns(
	ctx context.Context,
	workspaceRef string,
	query HookRunsQuery,
) ([]HookRunRecord, error) {
	if s.hookRunsFn != nil {
		if strings.TrimSpace(workspaceRef) == "" {
			return nil, errors.New("stub: workspaceRef is required")
		}
		return s.hookRunsFn(ctx, workspaceRef, query)
	}
	return nil, errors.New("unexpected HookRuns call")
}

func (s *stubClient) HookEvents(
	ctx context.Context,
	query HookEventsQuery,
) ([]HookEventRecord, error) {
	if s.hookEventsFn != nil {
		return s.hookEventsFn(ctx, query)
	}
	return nil, errors.New("unexpected HookEvents call")
}

func (s *stubClient) ListLogs(
	ctx context.Context,
	query LogsListQuery,
) ([]LogEventRecord, error) {
	if s.listLogsFn != nil {
		return s.listLogsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListLogs call")
}

func (s *stubClient) StreamLogs(
	ctx context.Context,
	query LogsListQuery,
	lastEventID string,
	handler SSEHandler,
) error {
	if s.streamLogsFn != nil {
		return s.streamLogsFn(ctx, query, lastEventID, handler)
	}
	return errors.New("unexpected StreamLogs call")
}

func (s *stubClient) MemoryHealth(ctx context.Context, workspace string) (MemoryHealthRecord, error) {
	if s.memoryHealthFn != nil {
		return s.memoryHealthFn(ctx, workspace)
	}
	return MemoryHealthRecord{}, errors.New("unexpected MemoryHealth call")
}

func (s *stubClient) MemoryHistory(
	ctx context.Context,
	query MemoryHistoryQuery,
) ([]MemoryHistoryRecord, error) {
	if s.memoryHistoryFn != nil {
		return s.memoryHistoryFn(ctx, query)
	}
	return nil, errors.New("unexpected MemoryHistory call")
}

func (s *stubClient) ListMemory(
	ctx context.Context,
	query MemoryListQuery,
) (MemoryListRecord, error) {
	if s.listMemoryFn != nil {
		return s.listMemoryFn(ctx, query)
	}
	return MemoryListRecord{}, errors.New("unexpected ListMemory call")
}

func (s *stubClient) ShowMemory(
	ctx context.Context,
	filename string,
	query MemorySelectorQuery,
) (MemoryEntryRecord, error) {
	if s.showMemoryFn != nil {
		return s.showMemoryFn(ctx, filename, query)
	}
	return MemoryEntryRecord{}, errors.New("unexpected ShowMemory call")
}

func (s *stubClient) CreateMemory(ctx context.Context, request MemoryCreateRequest) (MemoryMutationRecord, error) {
	if s.createMemoryFn != nil {
		return s.createMemoryFn(ctx, request)
	}
	return MemoryMutationRecord{}, errors.New("unexpected CreateMemory call")
}

func (s *stubClient) EditMemory(
	ctx context.Context,
	filename string,
	request MemoryEditRequest,
) (MemoryMutationRecord, error) {
	if s.editMemoryFn != nil {
		return s.editMemoryFn(ctx, filename, request)
	}
	return MemoryMutationRecord{}, errors.New("unexpected EditMemory call")
}

func (s *stubClient) DeleteMemory(
	ctx context.Context,
	filename string,
	query MemorySelectorQuery,
) (MemoryDeleteRecord, error) {
	if s.deleteMemoryFn != nil {
		return s.deleteMemoryFn(ctx, filename, query)
	}
	return MemoryDeleteRecord{}, errors.New("unexpected DeleteMemory call")
}

func (s *stubClient) SearchMemory(
	ctx context.Context,
	request MemorySearchRequest,
) (MemorySearchRecord, error) {
	if s.searchMemoryFn != nil {
		return s.searchMemoryFn(ctx, request)
	}
	return MemorySearchRecord{}, errors.New("unexpected SearchMemory call")
}

func (s *stubClient) ReindexMemory(
	ctx context.Context,
	request MemoryReindexRequest,
) (MemoryReindexRecord, error) {
	if s.reindexMemoryFn != nil {
		return s.reindexMemoryFn(ctx, request)
	}
	return MemoryReindexRecord{}, errors.New("unexpected ReindexMemory call")
}

func (s *stubClient) PromoteMemory(ctx context.Context, request MemoryPromoteRequest) (MemoryPromoteRecord, error) {
	if s.promoteMemoryFn != nil {
		return s.promoteMemoryFn(ctx, request)
	}
	return MemoryPromoteRecord{}, errors.New("unexpected PromoteMemory call")
}

func (s *stubClient) ResetMemory(ctx context.Context, request MemoryResetRequest) (MemoryResetRecord, error) {
	if s.resetMemoryFn != nil {
		return s.resetMemoryFn(ctx, request)
	}
	return MemoryResetRecord{}, errors.New("unexpected ResetMemory call")
}

func (s *stubClient) ReloadMemory(ctx context.Context, request MemorySelectorQuery) (MemoryReloadRecord, error) {
	if s.reloadMemoryFn != nil {
		return s.reloadMemoryFn(ctx, request)
	}
	return MemoryReloadRecord{}, errors.New("unexpected ReloadMemory call")
}

func (s *stubClient) MemoryScopeShow(ctx context.Context, query MemorySelectorQuery) (MemoryScopeShowRecord, error) {
	if s.memoryScopeShowFn != nil {
		return s.memoryScopeShowFn(ctx, query)
	}
	return MemoryScopeShowRecord{}, errors.New("unexpected MemoryScopeShow call")
}

func (s *stubClient) ListMemoryDecisions(
	ctx context.Context,
	query MemoryDecisionListQuery,
) (MemoryDecisionListRecord, error) {
	if s.listMemoryDecisionsFn != nil {
		return s.listMemoryDecisionsFn(ctx, query)
	}
	return MemoryDecisionListRecord{}, errors.New("unexpected ListMemoryDecisions call")
}

func (s *stubClient) GetMemoryDecision(ctx context.Context, id string) (MemoryDecisionRecord, error) {
	if s.getMemoryDecisionFn != nil {
		return s.getMemoryDecisionFn(ctx, id)
	}
	return MemoryDecisionRecord{}, errors.New("unexpected GetMemoryDecision call")
}

func (s *stubClient) RevertMemoryDecision(
	ctx context.Context,
	id string,
	request MemoryDecisionRevertRequest,
) (MemoryDecisionRevertRecord, error) {
	if s.revertMemoryDecisionFn != nil {
		return s.revertMemoryDecisionFn(ctx, id, request)
	}
	return MemoryDecisionRevertRecord{}, errors.New("unexpected RevertMemoryDecision call")
}

func (s *stubClient) GetMemoryRecallTrace(
	ctx context.Context,
	sessionID string,
	turnSeq int64,
) (MemoryRecallTraceRecord, error) {
	if s.getMemoryRecallTraceFn != nil {
		return s.getMemoryRecallTraceFn(ctx, sessionID, turnSeq)
	}
	return MemoryRecallTraceRecord{}, errors.New("unexpected GetMemoryRecallTrace call")
}

func (s *stubClient) ListMemoryDreams(ctx context.Context) (MemoryDreamListRecord, error) {
	if s.listMemoryDreamsFn != nil {
		return s.listMemoryDreamsFn(ctx)
	}
	return MemoryDreamListRecord{}, errors.New("unexpected ListMemoryDreams call")
}

func (s *stubClient) GetMemoryDream(ctx context.Context, id string) (MemoryDreamRecord, error) {
	if s.getMemoryDreamFn != nil {
		return s.getMemoryDreamFn(ctx, id)
	}
	return MemoryDreamRecord{}, errors.New("unexpected GetMemoryDream call")
}

func (s *stubClient) TriggerMemoryDream(
	ctx context.Context,
	request MemoryDreamTriggerRequest,
) (MemoryDreamTriggerRecord, error) {
	if s.triggerMemoryDreamFn != nil {
		return s.triggerMemoryDreamFn(ctx, request)
	}
	return MemoryDreamTriggerRecord{}, errors.New("unexpected TriggerMemoryDream call")
}

func (s *stubClient) RetryMemoryDream(
	ctx context.Context,
	id string,
	request MemoryDreamRetryRequest,
) (MemoryDreamRetryRecord, error) {
	if s.retryMemoryDreamFn != nil {
		return s.retryMemoryDreamFn(ctx, id, request)
	}
	return MemoryDreamRetryRecord{}, errors.New("unexpected RetryMemoryDream call")
}

func (s *stubClient) GetMemoryDreamStatus(ctx context.Context) (MemoryDreamListRecord, error) {
	if s.getMemoryDreamStatusFn != nil {
		return s.getMemoryDreamStatusFn(ctx)
	}
	return MemoryDreamListRecord{}, errors.New("unexpected GetMemoryDreamStatus call")
}

func (s *stubClient) ListMemoryDailyLogs(
	ctx context.Context,
	query MemorySelectorQuery,
) (MemoryDailyLogListRecord, error) {
	if s.listMemoryDailyLogsFn != nil {
		return s.listMemoryDailyLogsFn(ctx, query)
	}
	return MemoryDailyLogListRecord{}, errors.New("unexpected ListMemoryDailyLogs call")
}

func (s *stubClient) GetMemoryExtractorStatus(
	ctx context.Context,
	sessionID string,
) (MemoryExtractorStatusRecord, error) {
	if s.getMemoryExtractorStatusFn != nil {
		return s.getMemoryExtractorStatusFn(ctx, sessionID)
	}
	return MemoryExtractorStatusRecord{}, errors.New("unexpected GetMemoryExtractorStatus call")
}

func (s *stubClient) ListMemoryExtractorFailures(ctx context.Context) (MemoryExtractorFailuresRecord, error) {
	if s.listMemoryExtractorFailuresFn != nil {
		return s.listMemoryExtractorFailuresFn(ctx)
	}
	return MemoryExtractorFailuresRecord{}, errors.New("unexpected ListMemoryExtractorFailures call")
}

func (s *stubClient) RetryMemoryExtractor(
	ctx context.Context,
	request MemoryExtractorRetryRequest,
) (MemoryExtractorRetryRecord, error) {
	if s.retryMemoryExtractorFn != nil {
		return s.retryMemoryExtractorFn(ctx, request)
	}
	return MemoryExtractorRetryRecord{}, errors.New("unexpected RetryMemoryExtractor call")
}

func (s *stubClient) DrainMemoryExtractor(ctx context.Context) (MemoryExtractorDrainRecord, error) {
	if s.drainMemoryExtractorFn != nil {
		return s.drainMemoryExtractorFn(ctx)
	}
	return MemoryExtractorDrainRecord{}, errors.New("unexpected DrainMemoryExtractor call")
}

func (s *stubClient) ListMemoryProviders(ctx context.Context) (MemoryProviderListRecord, error) {
	if s.listMemoryProvidersFn != nil {
		return s.listMemoryProvidersFn(ctx)
	}
	return MemoryProviderListRecord{}, errors.New("unexpected ListMemoryProviders call")
}

func (s *stubClient) GetMemoryProvider(ctx context.Context, name string) (MemoryProviderRecord, error) {
	if s.getMemoryProviderFn != nil {
		return s.getMemoryProviderFn(ctx, name)
	}
	return MemoryProviderRecord{}, errors.New("unexpected GetMemoryProvider call")
}

func (s *stubClient) SelectMemoryProvider(
	ctx context.Context,
	request MemoryProviderSelectRequest,
) (MemoryProviderLifecycleRecord, error) {
	if s.selectMemoryProviderFn != nil {
		return s.selectMemoryProviderFn(ctx, request)
	}
	return MemoryProviderLifecycleRecord{}, errors.New("unexpected SelectMemoryProvider call")
}

func (s *stubClient) EnableMemoryProvider(
	ctx context.Context,
	name string,
	request MemoryProviderLifecycleRequest,
) (MemoryProviderLifecycleRecord, error) {
	if s.enableMemoryProviderFn != nil {
		return s.enableMemoryProviderFn(ctx, name, request)
	}
	return MemoryProviderLifecycleRecord{}, errors.New("unexpected EnableMemoryProvider call")
}

func (s *stubClient) DisableMemoryProvider(
	ctx context.Context,
	name string,
	request MemoryProviderLifecycleRequest,
) (MemoryProviderLifecycleRecord, error) {
	if s.disableMemoryProviderFn != nil {
		return s.disableMemoryProviderFn(ctx, name, request)
	}
	return MemoryProviderLifecycleRecord{}, errors.New("unexpected DisableMemoryProvider call")
}

func (s *stubClient) CreateMemoryAdhocNote(
	ctx context.Context,
	request MemoryAdhocNoteRequest,
) (MemoryAdhocNoteRecord, error) {
	if s.createMemoryAdhocNoteFn != nil {
		return s.createMemoryAdhocNoteFn(ctx, request)
	}
	return MemoryAdhocNoteRecord{}, errors.New("unexpected CreateMemoryAdhocNote call")
}

func (s *stubClient) ListAutomationJobs(
	ctx context.Context,
	query AutomationJobQuery,
) ([]JobRecord, error) {
	if s.listAutomationJobsFn != nil {
		return s.listAutomationJobsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationJobs call")
}

func (s *stubClient) CreateAutomationJob(
	ctx context.Context,
	request AutomationJobCreateRequest,
) (JobRecord, error) {
	if s.createAutomationJobFn != nil {
		return s.createAutomationJobFn(ctx, request)
	}
	return JobRecord{}, errors.New("unexpected CreateAutomationJob call")
}

func (s *stubClient) GetAutomationJob(ctx context.Context, id string) (JobRecord, error) {
	if s.getAutomationJobFn != nil {
		return s.getAutomationJobFn(ctx, id)
	}
	return JobRecord{}, errors.New("unexpected GetAutomationJob call")
}

func (s *stubClient) UpdateAutomationJob(
	ctx context.Context,
	id string,
	request AutomationJobUpdateRequest,
) (JobRecord, error) {
	if s.updateAutomationJobFn != nil {
		return s.updateAutomationJobFn(ctx, id, request)
	}
	return JobRecord{}, errors.New("unexpected UpdateAutomationJob call")
}

func (s *stubClient) DeleteAutomationJob(ctx context.Context, id string) error {
	if s.deleteAutomationJobFn != nil {
		return s.deleteAutomationJobFn(ctx, id)
	}
	return errors.New("unexpected DeleteAutomationJob call")
}

func (s *stubClient) TriggerAutomationJob(ctx context.Context, id string) (RunRecord, error) {
	if s.triggerAutomationJobFn != nil {
		return s.triggerAutomationJobFn(ctx, id)
	}
	return RunRecord{}, errors.New("unexpected TriggerAutomationJob call")
}

func (s *stubClient) AutomationJobRuns(
	ctx context.Context,
	id string,
	query AutomationRunQuery,
) ([]RunRecord, error) {
	if s.automationJobRunsFn != nil {
		return s.automationJobRunsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected AutomationJobRuns call")
}

func (s *stubClient) ListAutomationTriggers(
	ctx context.Context,
	query AutomationTriggerQuery,
) ([]TriggerRecord, error) {
	if s.listAutomationTriggersFn != nil {
		return s.listAutomationTriggersFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationTriggers call")
}

func (s *stubClient) CreateAutomationTrigger(
	ctx context.Context,
	request AutomationTriggerCreateRequest,
) (TriggerRecord, error) {
	if s.createAutomationTriggerFn != nil {
		return s.createAutomationTriggerFn(ctx, request)
	}
	return TriggerRecord{}, errors.New("unexpected CreateAutomationTrigger call")
}

func (s *stubClient) GetAutomationTrigger(ctx context.Context, id string) (TriggerRecord, error) {
	if s.getAutomationTriggerFn != nil {
		return s.getAutomationTriggerFn(ctx, id)
	}
	return TriggerRecord{}, errors.New("unexpected GetAutomationTrigger call")
}

func (s *stubClient) UpdateAutomationTrigger(
	ctx context.Context,
	id string,
	request AutomationTriggerUpdateRequest,
) (TriggerRecord, error) {
	if s.updateAutomationTriggerFn != nil {
		return s.updateAutomationTriggerFn(ctx, id, request)
	}
	return TriggerRecord{}, errors.New("unexpected UpdateAutomationTrigger call")
}

func (s *stubClient) DeleteAutomationTrigger(ctx context.Context, id string) error {
	if s.deleteAutomationTriggerFn != nil {
		return s.deleteAutomationTriggerFn(ctx, id)
	}
	return errors.New("unexpected DeleteAutomationTrigger call")
}

func (s *stubClient) AutomationTriggerRuns(
	ctx context.Context,
	id string,
	query AutomationRunQuery,
) ([]RunRecord, error) {
	if s.automationTriggerRunsFn != nil {
		return s.automationTriggerRunsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected AutomationTriggerRuns call")
}

func (s *stubClient) ListAutomationRuns(
	ctx context.Context,
	query AutomationRunQuery,
) ([]RunRecord, error) {
	if s.listAutomationRunsFn != nil {
		return s.listAutomationRunsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationRuns call")
}

func (s *stubClient) GetAutomationRun(ctx context.Context, id string) (RunRecord, error) {
	if s.getAutomationRunFn != nil {
		return s.getAutomationRunFn(ctx, id)
	}
	return RunRecord{}, errors.New("unexpected GetAutomationRun call")
}

func (s *stubClient) ListTasks(
	ctx context.Context,
	query TaskListQuery,
) ([]TaskSummaryRecord, error) {
	if s.listTasksFn != nil {
		return s.listTasksFn(ctx, query)
	}
	return nil, errors.New("unexpected ListTasks call")
}

func (s *stubClient) CreateTask(
	ctx context.Context,
	request CreateTaskRequest,
) (TaskRecord, error) {
	if s.createTaskFn != nil {
		return s.createTaskFn(ctx, request)
	}
	return TaskRecord{}, errors.New("unexpected CreateTask call")
}

func (s *stubClient) CreateTaskAsAgent(
	ctx context.Context,
	request CreateTaskRequest,
	credentials agentidentity.Credentials,
) (TaskRecord, error) {
	if s.createTaskAsAgentFn != nil {
		return s.createTaskAsAgentFn(ctx, request, credentials)
	}
	return TaskRecord{}, errors.New("unexpected CreateTaskAsAgent call")
}

func (s *stubClient) GetTask(ctx context.Context, id string) (TaskDetailRecord, error) {
	if s.getTaskFn != nil {
		return s.getTaskFn(ctx, id)
	}
	return TaskDetailRecord{}, errors.New("unexpected GetTask call")
}

func (s *stubClient) UpdateTask(
	ctx context.Context,
	id string,
	request UpdateTaskRequest,
) (TaskRecord, error) {
	if s.updateTaskFn != nil {
		return s.updateTaskFn(ctx, id, request)
	}
	return TaskRecord{}, errors.New("unexpected UpdateTask call")
}

func (s *stubClient) DeleteTask(ctx context.Context, id string) error {
	if s.deleteTaskFn != nil {
		return s.deleteTaskFn(ctx, id)
	}
	return errors.New("unexpected DeleteTask call")
}

func (s *stubClient) GetTaskExecutionProfile(
	ctx context.Context,
	id string,
) (TaskExecutionProfileRecord, error) {
	if s.getTaskExecutionProfileFn != nil {
		return s.getTaskExecutionProfileFn(ctx, id)
	}
	return TaskExecutionProfileRecord{}, errors.New("unexpected GetTaskExecutionProfile call")
}

func (s *stubClient) SetTaskExecutionProfile(
	ctx context.Context,
	id string,
	request *TaskExecutionProfileRequest,
) (TaskExecutionProfileRecord, error) {
	if s.setTaskExecutionProfileFn != nil {
		return s.setTaskExecutionProfileFn(ctx, id, request)
	}
	return TaskExecutionProfileRecord{}, errors.New("unexpected SetTaskExecutionProfile call")
}

func (s *stubClient) DeleteTaskExecutionProfile(ctx context.Context, id string) error {
	if s.deleteTaskExecutionProfileFn != nil {
		return s.deleteTaskExecutionProfileFn(ctx, id)
	}
	return errors.New("unexpected DeleteTaskExecutionProfile call")
}

func (s *stubClient) CreateTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	request *TaskBridgeNotificationSubscriptionRequest,
) (TaskBridgeNotificationSubscriptionRecord, error) {
	if s.createTaskBridgeNotificationSubscriptionFn != nil {
		return s.createTaskBridgeNotificationSubscriptionFn(ctx, taskID, request)
	}
	return TaskBridgeNotificationSubscriptionRecord{}, errors.New(
		"unexpected CreateTaskBridgeNotificationSubscription call",
	)
}

func (s *stubClient) ListTaskBridgeNotificationSubscriptions(
	ctx context.Context,
	taskID string,
	query TaskBridgeNotificationSubscriptionQuery,
) ([]TaskBridgeNotificationSubscriptionRecord, error) {
	if s.listTaskBridgeNotificationSubscriptionsFn != nil {
		return s.listTaskBridgeNotificationSubscriptionsFn(ctx, taskID, query)
	}
	return nil, errors.New("unexpected ListTaskBridgeNotificationSubscriptions call")
}

func (s *stubClient) GetTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	subscriptionID string,
) (TaskBridgeNotificationSubscriptionRecord, error) {
	if s.getTaskBridgeNotificationSubscriptionFn != nil {
		return s.getTaskBridgeNotificationSubscriptionFn(ctx, taskID, subscriptionID)
	}
	return TaskBridgeNotificationSubscriptionRecord{}, errors.New(
		"unexpected GetTaskBridgeNotificationSubscription call",
	)
}

func (s *stubClient) DeleteTaskBridgeNotificationSubscription(
	ctx context.Context,
	taskID string,
	subscriptionID string,
) error {
	if s.deleteTaskBridgeNotificationSubscriptionFn != nil {
		return s.deleteTaskBridgeNotificationSubscriptionFn(ctx, taskID, subscriptionID)
	}
	return errors.New("unexpected DeleteTaskBridgeNotificationSubscription call")
}

func (s *stubClient) RequestTaskRunReview(
	ctx context.Context,
	runID string,
	request *TaskRunReviewRequest,
) (TaskRunReviewRequestRecord, error) {
	if s.requestTaskRunReviewFn != nil {
		return s.requestTaskRunReviewFn(ctx, runID, request)
	}
	return TaskRunReviewRequestRecord{}, errors.New("unexpected RequestTaskRunReview call")
}

func (s *stubClient) RequestTaskRunReviewAsAgent(
	ctx context.Context,
	runID string,
	request *TaskRunReviewRequest,
	credentials agentidentity.Credentials,
) (TaskRunReviewRequestRecord, error) {
	if s.requestTaskRunReviewAsAgentFn != nil {
		return s.requestTaskRunReviewAsAgentFn(ctx, runID, request, credentials)
	}
	return TaskRunReviewRequestRecord{}, errors.New("unexpected RequestTaskRunReviewAsAgent call")
}

func (s *stubClient) ListTaskRunReviews(
	ctx context.Context,
	query TaskRunReviewListQuery,
) ([]TaskRunReviewRecord, error) {
	if s.listTaskRunReviewsFn != nil {
		return s.listTaskRunReviewsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListTaskRunReviews call")
}

func (s *stubClient) GetTaskRunReview(ctx context.Context, reviewID string) (TaskRunReviewRecord, error) {
	if s.getTaskRunReviewFn != nil {
		return s.getTaskRunReviewFn(ctx, reviewID)
	}
	return TaskRunReviewRecord{}, errors.New("unexpected GetTaskRunReview call")
}

func (s *stubClient) SubmitTaskRunReviewVerdict(
	ctx context.Context,
	reviewID string,
	request *TaskRunReviewVerdictRequest,
) (TaskRunReviewVerdictRecord, error) {
	if s.submitTaskRunReviewVerdictFn != nil {
		return s.submitTaskRunReviewVerdictFn(ctx, reviewID, request)
	}
	return TaskRunReviewVerdictRecord{}, errors.New("unexpected SubmitTaskRunReviewVerdict call")
}

func (s *stubClient) SubmitTaskRunReviewVerdictAsAgent(
	ctx context.Context,
	reviewID string,
	request *TaskRunReviewVerdictRequest,
	credentials agentidentity.Credentials,
) (TaskRunReviewVerdictRecord, error) {
	if s.submitTaskRunReviewVerdictAsAgentFn != nil {
		return s.submitTaskRunReviewVerdictAsAgentFn(ctx, reviewID, request, credentials)
	}
	return TaskRunReviewVerdictRecord{}, errors.New("unexpected SubmitTaskRunReviewVerdictAsAgent call")
}

func (s *stubClient) PublishTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	if s.publishTaskFn != nil {
		return s.publishTaskFn(ctx, id, request)
	}
	return TaskExecutionRecord{}, errors.New("unexpected PublishTask call")
}

func (s *stubClient) StartTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	if s.startTaskFn != nil {
		return s.startTaskFn(ctx, id, request)
	}
	return TaskExecutionRecord{}, errors.New("unexpected StartTask call")
}

func (s *stubClient) ApproveTask(
	ctx context.Context,
	id string,
	request TaskExecutionRequest,
) (TaskExecutionRecord, error) {
	if s.approveTaskFn != nil {
		return s.approveTaskFn(ctx, id, request)
	}
	return TaskExecutionRecord{}, errors.New("unexpected ApproveTask call")
}

func (s *stubClient) RejectTask(ctx context.Context, id string) (TaskRecord, error) {
	if s.rejectTaskFn != nil {
		return s.rejectTaskFn(ctx, id)
	}
	return TaskRecord{}, errors.New("unexpected RejectTask call")
}

func (s *stubClient) CancelTask(
	ctx context.Context,
	id string,
	request CancelTaskRequest,
) (TaskRecord, error) {
	if s.cancelTaskFn != nil {
		return s.cancelTaskFn(ctx, id, request)
	}
	return TaskRecord{}, errors.New("unexpected CancelTask call")
}

func (s *stubClient) CreateChildTask(
	ctx context.Context,
	id string,
	request CreateTaskChildRequest,
) (TaskRecord, error) {
	if s.createChildTaskFn != nil {
		return s.createChildTaskFn(ctx, id, request)
	}
	return TaskRecord{}, errors.New("unexpected CreateChildTask call")
}

func (s *stubClient) AddTaskDependency(
	ctx context.Context,
	id string,
	request AddTaskDependencyRequest,
) (TaskDetailRecord, error) {
	if s.addTaskDependencyFn != nil {
		return s.addTaskDependencyFn(ctx, id, request)
	}
	return TaskDetailRecord{}, errors.New("unexpected AddTaskDependency call")
}

func (s *stubClient) RemoveTaskDependency(
	ctx context.Context,
	id string,
	dependsOnID string,
) (TaskDetailRecord, error) {
	if s.removeTaskDependencyFn != nil {
		return s.removeTaskDependencyFn(ctx, id, dependsOnID)
	}
	return TaskDetailRecord{}, errors.New("unexpected RemoveTaskDependency call")
}

func (s *stubClient) EnqueueTaskRun(
	ctx context.Context,
	id string,
	request EnqueueTaskRunRequest,
) (TaskRunRecord, error) {
	if s.enqueueTaskRunFn != nil {
		return s.enqueueTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected EnqueueTaskRun call")
}

func (s *stubClient) ListTaskRuns(
	ctx context.Context,
	id string,
	query TaskRunListQuery,
) ([]TaskRunRecord, error) {
	if s.listTaskRunsFn != nil {
		return s.listTaskRunsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected ListTaskRuns call")
}

func (s *stubClient) ClaimTaskRun(
	ctx context.Context,
	id string,
	request ClaimTaskRunRequest,
) (TaskRunRecord, error) {
	if s.claimTaskRunFn != nil {
		return s.claimTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected ClaimTaskRun call")
}

func (s *stubClient) StartTaskRun(
	ctx context.Context,
	id string,
	request StartTaskRunRequest,
) (TaskRunRecord, error) {
	if s.startTaskRunFn != nil {
		return s.startTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected StartTaskRun call")
}

func (s *stubClient) AttachTaskRunSession(
	ctx context.Context,
	id string,
	request AttachTaskRunSessionRequest,
) (TaskRunRecord, error) {
	if s.attachTaskRunSessionFn != nil {
		return s.attachTaskRunSessionFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected AttachTaskRunSession call")
}

func (s *stubClient) CompleteTaskRun(
	ctx context.Context,
	id string,
	request CompleteTaskRunRequest,
) (TaskRunRecord, error) {
	if s.completeTaskRunFn != nil {
		return s.completeTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected CompleteTaskRun call")
}

func (s *stubClient) FailTaskRun(
	ctx context.Context,
	id string,
	request FailTaskRunRequest,
) (TaskRunRecord, error) {
	if s.failTaskRunFn != nil {
		return s.failTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected FailTaskRun call")
}

func (s *stubClient) CancelTaskRun(
	ctx context.Context,
	id string,
	request CancelTaskRunRequest,
) (TaskRunRecord, error) {
	if s.cancelTaskRunFn != nil {
		return s.cancelTaskRunFn(ctx, id, request)
	}
	return TaskRunRecord{}, errors.New("unexpected CancelTaskRun call")
}

func (s *stubClient) AgentMe(ctx context.Context, credentials agentidentity.Credentials) (AgentMeRecord, error) {
	if s.agentMeFn != nil {
		return s.agentMeFn(ctx, credentials)
	}
	return AgentMeRecord{}, errors.New("unexpected AgentMe call")
}

func (s *stubClient) AgentContext(
	ctx context.Context,
	credentials agentidentity.Credentials,
) (AgentContextRecord, error) {
	if s.agentContextFn != nil {
		return s.agentContextFn(ctx, credentials)
	}
	return AgentContextRecord{}, errors.New("unexpected AgentContext call")
}

func (s *stubClient) AgentSpawn(
	ctx context.Context,
	request AgentSpawnRequest,
	credentials agentidentity.Credentials,
) (AgentSpawnRecord, error) {
	if s.agentSpawnFn != nil {
		return s.agentSpawnFn(ctx, request, credentials)
	}
	return AgentSpawnRecord{}, errors.New("unexpected AgentSpawn call")
}

func (s *stubClient) AgentChannels(
	ctx context.Context,
	credentials agentidentity.Credentials,
) ([]AgentChannelRecord, error) {
	if s.agentChannelsFn != nil {
		return s.agentChannelsFn(ctx, credentials)
	}
	return nil, errors.New("unexpected AgentChannels call")
}

func (s *stubClient) AgentChannelRecv(
	ctx context.Context,
	channel string,
	query AgentChannelRecvQuery,
	credentials agentidentity.Credentials,
) ([]AgentChannelMessageRecord, error) {
	if s.agentChannelRecvFn != nil {
		return s.agentChannelRecvFn(ctx, channel, query, credentials)
	}
	return nil, errors.New("unexpected AgentChannelRecv call")
}

func (s *stubClient) AgentChannelSend(
	ctx context.Context,
	channel string,
	request AgentChannelSendRequest,
	credentials agentidentity.Credentials,
) (AgentChannelMessageRecord, error) {
	if s.agentChannelSendFn != nil {
		return s.agentChannelSendFn(ctx, channel, request, credentials)
	}
	return AgentChannelMessageRecord{}, errors.New("unexpected AgentChannelSend call")
}

func (s *stubClient) AgentChannelReply(
	ctx context.Context,
	request AgentChannelReplyRequest,
	credentials agentidentity.Credentials,
) (AgentChannelMessageRecord, error) {
	if s.agentChannelReplyFn != nil {
		return s.agentChannelReplyFn(ctx, request, credentials)
	}
	return AgentChannelMessageRecord{}, errors.New("unexpected AgentChannelReply call")
}

func (s *stubClient) AgentTaskClaimNext(
	ctx context.Context,
	request AgentTaskClaimNextRequest,
	credentials agentidentity.Credentials,
) (AgentTaskNextRecord, error) {
	if s.agentTaskClaimNextFn != nil {
		return s.agentTaskClaimNextFn(ctx, request, credentials)
	}
	return AgentTaskNextRecord{}, errors.New("unexpected AgentTaskClaimNext call")
}

func (s *stubClient) AgentTaskHeartbeat(
	ctx context.Context,
	runID string,
	request AgentTaskHeartbeatRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	if s.agentTaskHeartbeatFn != nil {
		return s.agentTaskHeartbeatFn(ctx, runID, request, credentials)
	}
	return AgentTaskLeaseRecord{}, errors.New("unexpected AgentTaskHeartbeat call")
}

func (s *stubClient) AgentTaskComplete(
	ctx context.Context,
	runID string,
	request AgentTaskCompleteRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	if s.agentTaskCompleteFn != nil {
		return s.agentTaskCompleteFn(ctx, runID, request, credentials)
	}
	return AgentTaskLeaseRecord{}, errors.New("unexpected AgentTaskComplete call")
}

func (s *stubClient) AgentTaskFail(
	ctx context.Context,
	runID string,
	request AgentTaskFailRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	if s.agentTaskFailFn != nil {
		return s.agentTaskFailFn(ctx, runID, request, credentials)
	}
	return AgentTaskLeaseRecord{}, errors.New("unexpected AgentTaskFail call")
}

func (s *stubClient) AgentTaskRelease(
	ctx context.Context,
	runID string,
	request AgentTaskReleaseRequest,
	credentials agentidentity.Credentials,
) (AgentTaskLeaseRecord, error) {
	if s.agentTaskReleaseFn != nil {
		return s.agentTaskReleaseFn(ctx, runID, request, credentials)
	}
	return AgentTaskLeaseRecord{}, errors.New("unexpected AgentTaskRelease call")
}

func newTestDeps(t *testing.T, client DaemonClient) commandDeps {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	return commandDeps{
		loadConfig: func() (aghconfig.Config, error) {
			return aghconfig.DefaultWithHome(homePaths), nil
		},
		resolveHome: func() (aghconfig.HomePaths, error) {
			return homePaths, nil
		},
		resolveHomeForWorkspace: func(string) (aghconfig.HomePaths, error) {
			return homePaths, nil
		},
		ensureHome: func(aghconfig.HomePaths) error { return nil },
		newClient: func(string) (DaemonClient, error) {
			return client, nil
		},
		processMatchesStartTime: func(int, time.Time) bool {
			return true
		},
		getwd: func() (string, error) {
			return "/workspace/project", nil
		},
		getenv: func(string) string { return "" },
		now: func() time.Time {
			return fixedTestNow
		},
	}
}

func executeRootCommand(t *testing.T, deps commandDeps, args ...string) (string, string, error) {
	t.Helper()

	cmd := newRootCommand(deps)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.ExecuteContext(testutil.Context(t))
	return stdout.String(), stderr.String(), err
}

func executeRootCommandWithExit(
	t *testing.T,
	deps commandDeps,
	args ...string,
) (int, string, string) {
	t.Helper()

	stdout, stderr, err := executeRootCommand(t, deps, args...)
	if err != nil {
		var rendered bytes.Buffer
		rendered.WriteString(stderr)
		return writeExecutionError(&rendered, args, err), stdout, rendered.String()
	}
	return 0, stdout, stderr
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return payload
}
