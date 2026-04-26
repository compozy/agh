package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/agentidentity"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/testutil"
)

var fixedTestNow = time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

type stubClient struct {
	daemonStatusFn            func(context.Context) (DaemonStatus, error)
	networkStatusFn           func(context.Context) (NetworkStatusRecord, error)
	networkPeersFn            func(context.Context, NetworkPeersQuery) ([]NetworkPeerRecord, error)
	networkChannelsFn         func(context.Context) ([]NetworkChannelRecord, error)
	networkSendFn             func(context.Context, NetworkSendRequest) (NetworkSendRecord, error)
	networkInboxFn            func(context.Context, string) ([]NetworkEnvelopeRecord, error)
	listExtensionsFn          func(context.Context) ([]ExtensionRecord, error)
	installExtensionFn        func(context.Context, InstallExtensionRequest) (ExtensionRecord, error)
	enableExtensionFn         func(context.Context, string) (ExtensionRecord, error)
	disableExtensionFn        func(context.Context, string) (ExtensionRecord, error)
	extensionStatusFn         func(context.Context, string) (ExtensionRecord, error)
	listBridgesFn             func(context.Context) ([]BridgeRecord, error)
	createBridgeFn            func(context.Context, CreateBridgeRequest) (BridgeRecord, error)
	getBridgeFn               func(context.Context, string) (BridgeRecord, error)
	updateBridgeFn            func(context.Context, string, UpdateBridgeRequest) (BridgeRecord, error)
	enableBridgeFn            func(context.Context, string) (BridgeRecord, error)
	disableBridgeFn           func(context.Context, string) (BridgeRecord, error)
	restartBridgeFn           func(context.Context, string) (BridgeRecord, error)
	bridgeRoutesFn            func(context.Context, string) ([]BridgeRouteRecord, error)
	testBridgeDeliveryFn      func(context.Context, string, BridgeTestDeliveryRequest) (BridgeTestDeliveryRecord, error)
	listSessionsFn            func(context.Context, SessionListQuery) ([]SessionRecord, error)
	createSessionFn           func(context.Context, CreateSessionRequest) (SessionRecord, error)
	getSessionFn              func(context.Context, string) (SessionRecord, error)
	stopSessionFn             func(context.Context, string) error
	resumeSessionFn           func(context.Context, string) (SessionRecord, error)
	promptSessionFn           func(context.Context, string, string) ([]AgentEventRecord, error)
	sessionEventsFn           func(context.Context, string, SessionEventQuery) ([]SessionEventRecord, error)
	streamSessionFn           func(context.Context, string, SessionEventQuery, string, SSEHandler) error
	sessionHistoryFn          func(context.Context, string, SessionEventQuery) ([]TurnHistoryRecord, error)
	createWorkspaceFn         func(context.Context, WorkspaceCreateRequest) (WorkspaceRecord, error)
	listWorkspacesFn          func(context.Context) ([]WorkspaceRecord, error)
	getWorkspaceFn            func(context.Context, string) (WorkspaceDetailRecord, error)
	updateWorkspaceFn         func(context.Context, string, WorkspaceUpdateRequest) (WorkspaceRecord, error)
	deleteWorkspaceFn         func(context.Context, string) error
	listAgentsFn              func(context.Context) ([]AgentRecord, error)
	getAgentFn                func(context.Context, string) (AgentRecord, error)
	hookCatalogFn             func(context.Context, HookCatalogQuery) ([]HookCatalogRecord, error)
	hookRunsFn                func(context.Context, HookRunsQuery) ([]HookRunRecord, error)
	hookEventsFn              func(context.Context, HookEventsQuery) ([]HookEventRecord, error)
	observeEventsFn           func(context.Context, ObserveEventQuery) ([]ObserveEventRecord, error)
	streamObserveEventsFn     func(context.Context, ObserveEventQuery, string, SSEHandler) error
	observeHealthFn           func(context.Context) (HealthStatus, error)
	memoryHealthFn            func(context.Context, string) (MemoryHealthRecord, error)
	memoryHistoryFn           func(context.Context, MemoryHistoryQuery) ([]MemoryHistoryRecord, error)
	listMemoryFn              func(context.Context, memory.Scope, string) ([]MemoryHeaderRecord, error)
	searchMemoryFn            func(context.Context, string, MemorySearchQuery) ([]MemorySearchRecord, error)
	readMemoryFn              func(context.Context, string, memory.Scope, string) (MemoryReadRecord, error)
	writeMemoryFn             func(context.Context, string, MemoryWriteRequest) (MemoryMutationRecord, error)
	deleteMemoryFn            func(context.Context, string, memory.Scope, string) (MemoryMutationRecord, error)
	reindexMemoryFn           func(context.Context, MemoryReindexRequest) (MemoryReindexRecord, error)
	consolidateMemoryFn       func(context.Context, string) (MemoryConsolidateRecord, error)
	listAutomationJobsFn      func(context.Context, AutomationJobQuery) ([]JobRecord, error)
	createAutomationJobFn     func(context.Context, AutomationJobCreateRequest) (JobRecord, error)
	getAutomationJobFn        func(context.Context, string) (JobRecord, error)
	updateAutomationJobFn     func(context.Context, string, AutomationJobUpdateRequest) (JobRecord, error)
	deleteAutomationJobFn     func(context.Context, string) error
	triggerAutomationJobFn    func(context.Context, string) (RunRecord, error)
	automationJobRunsFn       func(context.Context, string, AutomationRunQuery) ([]RunRecord, error)
	listAutomationTriggersFn  func(context.Context, AutomationTriggerQuery) ([]TriggerRecord, error)
	createAutomationTriggerFn func(context.Context, AutomationTriggerCreateRequest) (TriggerRecord, error)
	getAutomationTriggerFn    func(context.Context, string) (TriggerRecord, error)
	updateAutomationTriggerFn func(context.Context, string, AutomationTriggerUpdateRequest) (TriggerRecord, error)
	deleteAutomationTriggerFn func(context.Context, string) error
	automationTriggerRunsFn   func(context.Context, string, AutomationRunQuery) ([]RunRecord, error)
	listAutomationRunsFn      func(context.Context, AutomationRunQuery) ([]RunRecord, error)
	getAutomationRunFn        func(context.Context, string) (RunRecord, error)
	listTasksFn               func(context.Context, TaskListQuery) ([]TaskSummaryRecord, error)
	createTaskFn              func(context.Context, CreateTaskRequest) (TaskRecord, error)
	getTaskFn                 func(context.Context, string) (TaskDetailRecord, error)
	updateTaskFn              func(context.Context, string, UpdateTaskRequest) (TaskRecord, error)
	cancelTaskFn              func(context.Context, string, CancelTaskRequest) (TaskRecord, error)
	createChildTaskFn         func(context.Context, string, CreateTaskChildRequest) (TaskRecord, error)
	addTaskDependencyFn       func(context.Context, string, AddTaskDependencyRequest) (TaskDetailRecord, error)
	removeTaskDependencyFn    func(context.Context, string, string) (TaskDetailRecord, error)
	enqueueTaskRunFn          func(context.Context, string, EnqueueTaskRunRequest) (TaskRunRecord, error)
	listTaskRunsFn            func(context.Context, string, TaskRunListQuery) ([]TaskRunRecord, error)
	claimTaskRunFn            func(context.Context, string, ClaimTaskRunRequest) (TaskRunRecord, error)
	startTaskRunFn            func(context.Context, string, StartTaskRunRequest) (TaskRunRecord, error)
	attachTaskRunSessionFn    func(context.Context, string, AttachTaskRunSessionRequest) (TaskRunRecord, error)
	completeTaskRunFn         func(context.Context, string, CompleteTaskRunRequest) (TaskRunRecord, error)
	failTaskRunFn             func(context.Context, string, FailTaskRunRequest) (TaskRunRecord, error)
	cancelTaskRunFn           func(context.Context, string, CancelTaskRunRequest) (TaskRunRecord, error)
	agentMeFn                 func(context.Context, agentidentity.Credentials) (AgentMeRecord, error)
	agentContextFn            func(context.Context, agentidentity.Credentials) (AgentContextRecord, error)
	agentChannelsFn           func(context.Context, agentidentity.Credentials) ([]AgentChannelRecord, error)
	agentChannelRecvFn        func(context.Context, string, AgentChannelRecvQuery, agentidentity.Credentials) ([]AgentChannelMessageRecord, error)
	agentChannelSendFn        func(context.Context, string, AgentChannelSendRequest, agentidentity.Credentials) (AgentChannelMessageRecord, error)
	agentChannelReplyFn       func(context.Context, AgentChannelReplyRequest, agentidentity.Credentials) (AgentChannelMessageRecord, error)
}

var _ DaemonClient = (*stubClient)(nil)

func (s *stubClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	if s.daemonStatusFn != nil {
		return s.daemonStatusFn(ctx)
	}
	return DaemonStatus{}, errors.New("unexpected DaemonStatus call")
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

func (s *stubClient) NetworkChannels(ctx context.Context) ([]NetworkChannelRecord, error) {
	if s.networkChannelsFn != nil {
		return s.networkChannelsFn(ctx)
	}
	return nil, errors.New("unexpected NetworkChannels call")
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
	sessionID string,
) ([]NetworkEnvelopeRecord, error) {
	if s.networkInboxFn != nil {
		return s.networkInboxFn(ctx, sessionID)
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

func (s *stubClient) ListAgents(ctx context.Context) ([]AgentRecord, error) {
	if s.listAgentsFn != nil {
		return s.listAgentsFn(ctx)
	}
	return nil, errors.New("unexpected ListAgents call")
}

func (s *stubClient) GetAgent(ctx context.Context, name string) (AgentRecord, error) {
	if s.getAgentFn != nil {
		return s.getAgentFn(ctx, name)
	}
	return AgentRecord{}, errors.New("unexpected GetAgent call")
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

func (s *stubClient) HookRuns(ctx context.Context, query HookRunsQuery) ([]HookRunRecord, error) {
	if s.hookRunsFn != nil {
		return s.hookRunsFn(ctx, query)
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

func (s *stubClient) ObserveEvents(
	ctx context.Context,
	query ObserveEventQuery,
) ([]ObserveEventRecord, error) {
	if s.observeEventsFn != nil {
		return s.observeEventsFn(ctx, query)
	}
	return nil, errors.New("unexpected ObserveEvents call")
}

func (s *stubClient) StreamObserveEvents(
	ctx context.Context,
	query ObserveEventQuery,
	lastEventID string,
	handler SSEHandler,
) error {
	if s.streamObserveEventsFn != nil {
		return s.streamObserveEventsFn(ctx, query, lastEventID, handler)
	}
	return errors.New("unexpected StreamObserveEvents call")
}

func (s *stubClient) ObserveHealth(ctx context.Context) (HealthStatus, error) {
	if s.observeHealthFn != nil {
		return s.observeHealthFn(ctx)
	}
	return HealthStatus{}, errors.New("unexpected ObserveHealth call")
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
	scope memory.Scope,
	workspace string,
) ([]MemoryHeaderRecord, error) {
	if s.listMemoryFn != nil {
		return s.listMemoryFn(ctx, scope, workspace)
	}
	return nil, errors.New("unexpected ListMemory call")
}

func (s *stubClient) SearchMemory(
	ctx context.Context,
	query string,
	opts MemorySearchQuery,
) ([]MemorySearchRecord, error) {
	if s.searchMemoryFn != nil {
		return s.searchMemoryFn(ctx, query, opts)
	}
	return nil, errors.New("unexpected SearchMemory call")
}

func (s *stubClient) ReadMemory(
	ctx context.Context,
	filename string,
	scope memory.Scope,
	workspace string,
) (MemoryReadRecord, error) {
	if s.readMemoryFn != nil {
		return s.readMemoryFn(ctx, filename, scope, workspace)
	}
	return MemoryReadRecord{}, errors.New("unexpected ReadMemory call")
}

func (s *stubClient) WriteMemory(
	ctx context.Context,
	filename string,
	request MemoryWriteRequest,
) (MemoryMutationRecord, error) {
	if s.writeMemoryFn != nil {
		return s.writeMemoryFn(ctx, filename, request)
	}
	return MemoryMutationRecord{}, errors.New("unexpected WriteMemory call")
}

func (s *stubClient) DeleteMemory(
	ctx context.Context,
	filename string,
	scope memory.Scope,
	workspace string,
) (MemoryMutationRecord, error) {
	if s.deleteMemoryFn != nil {
		return s.deleteMemoryFn(ctx, filename, scope, workspace)
	}
	return MemoryMutationRecord{}, errors.New("unexpected DeleteMemory call")
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

func (s *stubClient) ConsolidateMemory(
	ctx context.Context,
	workspace string,
) (MemoryConsolidateRecord, error) {
	if s.consolidateMemoryFn != nil {
		return s.consolidateMemoryFn(ctx, workspace)
	}
	return MemoryConsolidateRecord{}, errors.New("unexpected ConsolidateMemory call")
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
		ensureHome: func(aghconfig.HomePaths) error { return nil },
		newClient: func(string) (DaemonClient, error) {
			return client, nil
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
		return 1, stdout, fmt.Sprintf("%serror: %v\n", stderr, err)
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
