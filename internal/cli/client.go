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
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/sse"
)

const (
	baseURL              = "http://unix"
	defaultUserAgentName = "agh-cli"
)

// DaemonClient is the CLI transport surface for talking to the AGH daemon over UDS.
type DaemonClient interface {
	DaemonStatus(ctx context.Context) (DaemonStatus, error)
	NetworkStatus(ctx context.Context) (NetworkStatusRecord, error)
	NetworkPeers(ctx context.Context, query NetworkPeersQuery) ([]NetworkPeerRecord, error)
	NetworkChannels(ctx context.Context) ([]NetworkChannelRecord, error)
	NetworkSend(ctx context.Context, request NetworkSendRequest) (NetworkSendRecord, error)
	NetworkInbox(ctx context.Context, sessionID string) ([]NetworkEnvelopeRecord, error)
	ListExtensions(ctx context.Context) ([]ExtensionRecord, error)
	InstallExtension(ctx context.Context, request InstallExtensionRequest) (ExtensionRecord, error)
	EnableExtension(ctx context.Context, name string) (ExtensionRecord, error)
	DisableExtension(ctx context.Context, name string) (ExtensionRecord, error)
	ExtensionStatus(ctx context.Context, name string) (ExtensionRecord, error)
	ListBridges(ctx context.Context) ([]BridgeRecord, error)
	CreateBridge(ctx context.Context, request CreateBridgeRequest) (BridgeRecord, error)
	GetBridge(ctx context.Context, id string) (BridgeRecord, error)
	UpdateBridge(ctx context.Context, id string, request UpdateBridgeRequest) (BridgeRecord, error)
	EnableBridge(ctx context.Context, id string) (BridgeRecord, error)
	DisableBridge(ctx context.Context, id string) (BridgeRecord, error)
	RestartBridge(ctx context.Context, id string) (BridgeRecord, error)
	BridgeRoutes(ctx context.Context, id string) ([]BridgeRouteRecord, error)
	TestBridgeDelivery(
		ctx context.Context,
		id string,
		request BridgeTestDeliveryRequest,
	) (BridgeTestDeliveryRecord, error)
	ListSessions(ctx context.Context, query SessionListQuery) ([]SessionRecord, error)
	CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error)
	GetSession(ctx context.Context, id string) (SessionRecord, error)
	StopSession(ctx context.Context, id string) error
	ResumeSession(ctx context.Context, id string) (SessionRecord, error)
	PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error)
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
	ListAgents(ctx context.Context) ([]AgentRecord, error)
	GetAgent(ctx context.Context, name string) (AgentRecord, error)
	HookCatalog(ctx context.Context, query HookCatalogQuery) ([]HookCatalogRecord, error)
	HookRuns(ctx context.Context, query HookRunsQuery) ([]HookRunRecord, error)
	HookEvents(ctx context.Context, query HookEventsQuery) ([]HookEventRecord, error)
	ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error)
	StreamObserveEvents(ctx context.Context, query ObserveEventQuery, lastEventID string, handler SSEHandler) error
	ObserveHealth(ctx context.Context) (HealthStatus, error)
	MemoryHealth(ctx context.Context, workspace string) (MemoryHealthRecord, error)
	MemoryHistory(ctx context.Context, query MemoryHistoryQuery) ([]MemoryHistoryRecord, error)
	ListMemory(ctx context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error)
	SearchMemory(ctx context.Context, query string, opts MemorySearchQuery) ([]MemorySearchRecord, error)
	ReadMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryReadRecord, error)
	WriteMemory(ctx context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error)
	DeleteMemory(
		ctx context.Context,
		filename string,
		scope memory.Scope,
		workspace string,
	) (MemoryMutationRecord, error)
	ReindexMemory(ctx context.Context, request MemoryReindexRequest) (MemoryReindexRecord, error)
	ConsolidateMemory(ctx context.Context, workspace string) (MemoryConsolidateRecord, error)
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
}

// CreateSessionRequest captures the shared daemon session creation payload.
type CreateSessionRequest = contract.CreateSessionRequest

// SessionListQuery captures the CLI filters for session list queries.
type SessionListQuery struct {
	Workspace string
}

// SessionRecord is the shared daemon session payload.
type SessionRecord = contract.SessionPayload

// ACPCapsRecord captures optional runtime capabilities exposed by the daemon API.
type ACPCapsRecord = contract.ACPCapsPayload

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

// AgentMCPServer is one MCP server entry returned by the daemon API.
type AgentMCPServer = contract.AgentMCPServerJSON

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

// MemoryHeaderRecord is one memory header returned by the daemon API.
type MemoryHeaderRecord = memory.Header

// MemoryHealthRecord is the shared daemon memory health payload.
type MemoryHealthRecord = contract.MemoryHealthPayload

// MemoryHistoryQuery captures filters for memory operation history.
type MemoryHistoryQuery struct {
	Scope     memory.Scope
	Workspace string
	Operation string
	Since     time.Time
	Limit     int
}

// MemoryHistoryRecord is one redacted memory operation history row.
type MemoryHistoryRecord = contract.MemoryOperationPayload

// MemoryReadRecord is the shared daemon memory document payload.
type MemoryReadRecord = contract.MemoryReadResponse

// MemorySearchQuery captures filters for durable memory search.
type MemorySearchQuery struct {
	Scope     memory.Scope
	Workspace string
	Limit     int
}

// MemorySearchRecord is one ranked durable memory search hit.
type MemorySearchRecord = memory.SearchResult

// MemoryWriteRequest captures the daemon API write payload.
type MemoryWriteRequest = contract.MemoryWriteRequest

// MemoryMutationRecord captures the daemon API write/delete response.
type MemoryMutationRecord = contract.MemoryMutationResponse

// MemoryReindexRequest captures the daemon API memory reindex payload.
type MemoryReindexRequest = contract.MemoryReindexRequest

// MemoryReindexRecord captures the daemon API memory reindex response.
type MemoryReindexRecord = memory.ReindexResult

// MemoryConsolidateRecord captures the daemon API consolidation response.
type MemoryConsolidateRecord = contract.MemoryConsolidateResponse

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

// TaskEventRecord is the shared task audit-event payload.
type TaskEventRecord = contract.TaskEventPayload

// TaskListQuery captures CLI filters for task list calls.
type TaskListQuery = contract.TaskListQuery

// TaskRunListQuery captures CLI filters for task-run list calls.
type TaskRunListQuery = contract.TaskRunListQuery

// CreateTaskRequest captures the shared task-create payload.
type CreateTaskRequest = contract.CreateTaskRequest

// CreateTaskChildRequest captures the shared child-task create payload.
type CreateTaskChildRequest = contract.CreateTaskChildRequest

// UpdateTaskRequest captures mutable task fields.
type UpdateTaskRequest = contract.UpdateTaskRequest

// CancelTaskRequest captures the shared task-cancel payload.
type CancelTaskRequest = contract.CancelTaskRequest

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

// NetworkEnvelopeRecord is the shared surfaced envelope payload.
type NetworkEnvelopeRecord = contract.NetworkEnvelopePayload

// NetworkPeersQuery captures CLI filters for peer listing.
type NetworkPeersQuery struct {
	Channel string
}

// InstallExtensionRequest captures the shared extension install payload.
type InstallExtensionRequest = contract.InstallExtensionRequest

// ExtensionRecord is the shared extension response payload.
type ExtensionRecord = contract.ExtensionPayload

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

// SSEEvent is one parsed server-sent event frame.
type SSEEvent = sse.Event
type SSEHandler = sse.Handler

type unixSocketClient struct {
	socketPath string
	httpClient *http.Client
}

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
		socketPath: path,
		httpClient: &http.Client{Transport: transport},
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

func (c *unixSocketClient) PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error) {
	path := "/api/sessions/" + url.PathEscape(strings.TrimSpace(id)) + "/prompt"
	var events []AgentEventRecord
	err := c.doSSE(
		ctx,
		http.MethodPost,
		path,
		nil,
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
	var response WorkspaceDetailRecord
	path := "/api/workspaces/" + url.PathEscape(strings.TrimSpace(ref))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &response); err != nil {
		return WorkspaceDetailRecord{}, err
	}
	return response, nil
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

func (c *unixSocketClient) ListAgents(ctx context.Context) ([]AgentRecord, error) {
	var response struct {
		Agents []AgentRecord `json:"agents"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/agents", nil, nil, &response); err != nil {
		return nil, err
	}
	return response.Agents, nil
}

func (c *unixSocketClient) GetAgent(ctx context.Context, name string) (AgentRecord, error) {
	var response struct {
		Agent AgentRecord `json:"agent"`
	}
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/agents/"+url.PathEscape(strings.TrimSpace(name)),
		nil,
		nil,
		&response,
	); err != nil {
		return AgentRecord{}, err
	}
	return response.Agent, nil
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
	var response contract.MemoryHistoryResponse
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
	scope memory.Scope,
	workspace string,
) ([]MemoryHeaderRecord, error) {
	var response []MemoryHeaderRecord
	if err := c.doJSON(ctx, http.MethodGet, "/api/memory", memoryValues(scope, workspace), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *unixSocketClient) SearchMemory(
	ctx context.Context,
	query string,
	opts MemorySearchQuery,
) ([]MemorySearchRecord, error) {
	var response []MemorySearchRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/search",
		memorySearchValues(query, opts),
		nil,
		&response,
	); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *unixSocketClient) ReadMemory(
	ctx context.Context,
	filename string,
	scope memory.Scope,
	workspace string,
) (MemoryReadRecord, error) {
	var response MemoryReadRecord
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/api/memory/"+url.PathEscape(strings.TrimSpace(filename)),
		memoryValues(scope, workspace),
		nil,
		&response,
	); err != nil {
		return MemoryReadRecord{}, err
	}
	return response, nil
}

func (c *unixSocketClient) WriteMemory(
	ctx context.Context,
	filename string,
	request MemoryWriteRequest,
) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(
		ctx,
		http.MethodPut,
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
	scope memory.Scope,
	workspace string,
) (MemoryMutationRecord, error) {
	var response MemoryMutationRecord
	if err := c.doJSON(
		ctx,
		http.MethodDelete,
		"/api/memory/"+url.PathEscape(strings.TrimSpace(filename)),
		memoryValues(scope, workspace),
		nil,
		&response,
	); err != nil {
		return MemoryMutationRecord{}, err
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

func (c *unixSocketClient) ConsolidateMemory(ctx context.Context, workspace string) (MemoryConsolidateRecord, error) {
	var response MemoryConsolidateRecord
	if err := c.doJSON(
		ctx,
		http.MethodPost,
		"/api/memory/consolidate",
		nil,
		map[string]string{"workspace": strings.TrimSpace(workspace)},
		&response,
	); err != nil {
		return MemoryConsolidateRecord{}, err
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

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}
	defer func() {
		_ = response.Body.Close()
	}()
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
	response, err := c.doRequest(ctx, method, path, query, requestBody, lastEventID)
	if err != nil {
		return err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return readAPIError(response)
	}
	defer func() {
		_ = response.Body.Close()
	}()

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
	if ctx == nil {
		return nil, errors.New("cli: context is required")
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

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cli: %s %s via %s: %w", method, path, c.socketPath, err)
	}
	return response, nil
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

func networkPeersValues(query NetworkPeersQuery) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(query.Channel); trimmed != "" {
		values.Set("channel", trimmed)
	}
	return values
}

func networkInboxValues(sessionID string) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(sessionID); trimmed != "" {
		values.Set("session_id", trimmed)
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

func memoryValues(scope memory.Scope, workspace string) url.Values {
	values := url.Values{}
	if trimmed := strings.TrimSpace(string(scope)); trimmed != "" {
		values.Set("scope", trimmed)
	}
	if trimmed := strings.TrimSpace(workspace); trimmed != "" {
		values.Set("workspace", trimmed)
	}
	return values
}

func memorySearchValues(query string, opts MemorySearchQuery) url.Values {
	values := memoryValues(opts.Scope, opts.Workspace)
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		values.Set("q", trimmed)
	}
	if opts.Limit > 0 {
		values.Set("limit", strconv.Itoa(opts.Limit))
	}
	return values
}

func memoryHistoryValues(query MemoryHistoryQuery) url.Values {
	values := memoryValues(query.Scope, query.Workspace)
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

func readAPIError(response *http.Response) error {
	defer func() {
		_ = response.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("cli: read api error response: %w", err)
	}

	var payload struct {
		Error string `json:"error"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil && strings.TrimSpace(payload.Error) != "" {
		return errors.New(strings.TrimSpace(payload.Error))
	}

	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("daemon api %s: %s", response.Status, message)
}

func drainResponseBody(method string, path string, body io.Reader) error {
	if _, err := io.Copy(io.Discard, body); err != nil {
		return fmt.Errorf("cli: drain %s %s response: %w", method, path, err)
	}
	return nil
}
