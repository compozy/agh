package extensionpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"
	"sync"
	"time"

	memcontract "github.com/pedronauck/agh/internal/memory/contract"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/acp"
	apicontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/network"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const (
	hostAPIAgentNameKey                 = "agent_name"
	hostAPIAutomationJobsDeletePath     = "automation/jobs/delete"
	hostAPIAutomationJobsGetPath        = "automation/jobs/get"
	hostAPIAutomationJobsRunsPath       = "automation/jobs/runs"
	hostAPIAutomationJobsTriggerPath    = "automation/jobs/trigger"
	hostAPIAutomationJobsUpdatePath     = "automation/jobs/update"
	hostAPIAutomationRunsPath           = "automation/runs"
	hostAPIAutomationTriggersPath       = "automation/triggers"
	hostAPIAutomationTriggersCreatePath = "automation/triggers/create"
	hostAPIAutomationTriggersDeletePath = "automation/triggers/delete"
	hostAPIAutomationTriggersFirePath   = "automation/triggers/fire"
	hostAPIAutomationTriggersGetPath    = "automation/triggers/get"
	hostAPIAutomationTriggersRunsPath   = "automation/triggers/runs"
	hostAPIAutomationTriggersUpdatePath = "automation/triggers/update"
	hostAPIBridgesInstancesGetPath      = "bridges/instances/get"
	hostAPIBridgesMessagesIngestPath    = "bridges/messages/ingest"
	hostAPILimitKey                     = "limit"
	hostAPIMemoryStorePath              = "memory/store"
	hostAPIMethodKey                    = "method"
	hostAPIObserveHealthPath            = "observe/health"
	hostAPIResourceKey                  = "resource"
	hostAPIResourcesGetPath             = "resources/get"
	hostAPISandboxInfoPath              = "sandbox/info"
	hostAPISandboxListPath              = "sandbox/list"
	hostAPIScopeKey                     = "scope"
	hostAPISessionIDKey                 = "session_id"
	hostAPISessionsListPath             = "sessions/list"
	hostAPISessionsPromptPath           = "sessions/prompt"
	hostAPISessionsStatusPath           = "sessions/status"
	hostAPISessionsStopPath             = "sessions/stop"
	hostAPIWorkspaceIDKey               = "workspace_id"
)

const (
	// HostAPIRateLimitedCode is the protocol code for per-extension backpressure.
	HostAPIRateLimitedCode = -32002
	// HostAPIUnavailableCode reports a temporarily unavailable Host API resource.
	HostAPIUnavailableCode = -32005
	// HostAPINotFoundCode reports a missing Host API resource.
	HostAPINotFoundCode = -32006
	// HostAPIInvalidParamsCode is the JSON-RPC invalid params code used for bad request payloads.
	HostAPIInvalidParamsCode = -32602
	// HostAPIMethodNotFoundCode is the JSON-RPC method-not-found code for unknown Host API methods.
	HostAPIMethodNotFoundCode = -32601

	defaultHostAPIRateLimit             = 10
	defaultHostAPIBurst                 = 20
	defaultHostAPIDefaultLimit          = 100
	defaultHostAPIRecallLimit           = 10
	defaultHostAPIBridgeIngestDedupTTL  = 24 * time.Hour
	defaultHostAPIBridgeCleanupInterval = time.Hour
	maxMemoryDescriptionLength          = 160
	tagCommentPrefix                    = "<!-- agh-tags:"
	hostAPIUnknownExtensionName         = "unknown"
	hostAPISandboxStateSynced           = "synced"
	hostAPISandboxStatePending          = "pending"
)

type hostAPIContextKey string

const hostAPIExtensionNameContextKey hostAPIContextKey = "extension.host_api.extension_name"
const hostAPIBridgeRuntimeContextKey hostAPIContextKey = "extension.host_api.bridge_runtime"
const hostAPIResourceSessionContextKey hostAPIContextKey = "extension.host_api.resource_session"

// HostAPIOption customizes a HostAPIHandler.
type HostAPIOption func(*HostAPIHandler)

// HostAPIHandler handles extension -> AGH Host API JSON-RPC requests.
type HostAPIHandler struct {
	sessions         hostAPISessionManager
	automation       HostAPIAutomationManager
	tasks            hostAPITaskManager
	network          hostAPINetworkService
	networkStore     store.NetworkConversationStore
	memory           *memory.Store
	observer         hostAPIObserver
	skills           hostAPISkillsRegistry
	modelCatalog     hostAPIModelCatalogService
	workspaces       workspacepkg.RuntimeResolver
	bridges          hostAPIBridgeRegistry
	dedupStore       hostAPIBridgeDedupStore
	deliveryBroker   hostAPIDeliveryBroker
	resourceStore    resources.RawStore
	resourceCodecs   *resources.CodecRegistry
	soulAuthoring    hostAPISoulAuthoringService
	soulRefresher    hostAPISoulRefresher
	heartbeatAuthor  hostAPIHeartbeatAuthoringService
	heartbeatStatus  hostAPIHeartbeatStatusService
	heartbeatWake    hostAPIHeartbeatWakeService
	sessionHealth    hostAPISessionHealthReader
	wakeEvents       hostAPIHeartbeatWakeEventReader
	memoryProviders  *MemoryProviderRegistry
	capChecker       *CapabilityChecker
	limiter          *hostAPIRateLimiter
	automationGetter func() HostAPIAutomationManager
	resourceTrigger  func(context.Context, resources.ResourceKind, resources.ReconcileReason) error
	now              func() time.Time
	rateLimit        int
	rateBurst        int

	bridgeIngestDedupTTL  time.Duration
	bridgeCleanupInterval time.Duration
	bridgeLocks           *hostAPIKeyLocker
	bridgeCleanupMu       sync.Mutex
	bridgeLastCleanup     time.Time

	methods map[string]hostAPIMethodFunc
}

type hostAPIMethodFunc func(context.Context, json.RawMessage) (any, error)

type hostAPISessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
	Status(ctx context.Context, id string) (*session.Info, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	Stop(ctx context.Context, id string) error
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	ExecSandbox(ctx context.Context, req session.SandboxExecRequest) (session.SandboxExecResult, error)
}

type hostAPINetworkPromptSessionManager interface {
	PromptNetwork(
		ctx context.Context,
		id string,
		msg string,
		meta ...acp.PromptNetworkMeta,
	) (<-chan acp.AgentEvent, error)
}

type hostAPIPromptingSessionManager interface {
	IsPrompting(id string) bool
}

type hostAPINetworkService interface {
	Send(ctx context.Context, req network.SendRequest) (string, error)
	ListPeers(ctx context.Context, workspaceID string, channel string) ([]network.PeerInfo, error)
	ListChannels(ctx context.Context, workspaceID string) ([]network.ChannelInfo, error)
	Status(ctx context.Context) (*network.Status, error)
}

type hostAPIObserver interface {
	Health(ctx context.Context) (observepkg.Health, error)
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	QueryTaskDashboard(ctx context.Context, query observepkg.TaskDashboardQuery) (observepkg.TaskDashboardView, error)
	QueryTaskInbox(
		ctx context.Context,
		query observepkg.TaskInboxQuery,
		actor taskpkg.ActorIdentity,
	) (observepkg.TaskInboxView, error)
}

// HostAPIAutomationManager is the automation surface exposed to the extension Host API.
type HostAPIAutomationManager interface {
	ListJobs(ctx context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error)
	GetJob(ctx context.Context, id string) (automationpkg.Job, error)
	CreateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error)
	UpdateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error)
	DeleteJob(ctx context.Context, id string) error
	TriggerJob(ctx context.Context, id string) (automationpkg.Run, error)
	TriggerJobWithPayload(ctx context.Context, id string, payload map[string]any) (automationpkg.Run, error)
	ListTriggers(ctx context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	GetTrigger(ctx context.Context, id string) (automationpkg.Trigger, error)
	CreateTrigger(
		ctx context.Context,
		trigger automationpkg.Trigger,
		webhookSecret automationpkg.WebhookSecretWrite,
	) (automationpkg.Trigger, error)
	UpdateTrigger(
		ctx context.Context,
		trigger automationpkg.Trigger,
		webhookSecret *automationpkg.WebhookSecretWrite,
	) (automationpkg.Trigger, error)
	DeleteTrigger(ctx context.Context, id string) error
	ListRuns(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error)
	SetJobEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Job, error)
	SetTriggerEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Trigger, error)
	FireExtensionTrigger(
		ctx context.Context,
		request automationpkg.ExtensionTriggerRequest,
	) (automationpkg.TriggerResult, error)
}

type hostAPITaskManager interface {
	ListTasks(ctx context.Context, query taskpkg.Query, actor taskpkg.ActorContext) ([]taskpkg.Summary, error)
	GetTask(ctx context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.View, error)
	Timeline(
		ctx context.Context,
		taskID string,
		query taskpkg.TimelineQuery,
		actor taskpkg.ActorContext,
	) ([]taskpkg.TimelineItem, error)
	Tree(ctx context.Context, taskID string, actor taskpkg.ActorContext) (*taskpkg.TreeView, error)
	RunDetail(ctx context.Context, runID string, actor taskpkg.ActorContext) (*taskpkg.RunDetailView, error)
	ListTaskRuns(
		ctx context.Context,
		taskID string,
		query taskpkg.RunQuery,
		actor taskpkg.ActorContext,
	) ([]taskpkg.Run, error)
	CreateTask(ctx context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error)
	UpdateTask(
		ctx context.Context,
		id string,
		patch taskpkg.Patch,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	CancelTask(
		ctx context.Context,
		id string,
		req taskpkg.CancelTask,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	EnqueueRun(ctx context.Context, spec taskpkg.EnqueueRun, actor taskpkg.ActorContext) (*taskpkg.Run, error)
	ClaimRun(
		ctx context.Context,
		runID string,
		claim taskpkg.ClaimRun,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	StartRun(
		ctx context.Context,
		runID string,
		req taskpkg.StartRun,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	AttachRunSession(
		ctx context.Context,
		runID string,
		sessionID string,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	CompleteRun(
		ctx context.Context,
		runID string,
		result taskpkg.RunResult,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	FailRun(
		ctx context.Context,
		runID string,
		failure taskpkg.RunFailure,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
	CancelRun(
		ctx context.Context,
		runID string,
		req taskpkg.CancelRun,
		actor taskpkg.ActorContext,
	) (*taskpkg.Run, error)
}

type hostAPIDeliveryBroker interface {
	RegisterPromptDelivery(
		ctx context.Context,
		reg bridgepkg.PromptDeliveryRegistration,
	) (*bridgepkg.DeliverySnapshot, error)
	ProjectEvent(ctx context.Context, sessionID string, event bridgepkg.DeliveryProjectionEvent) error
}

type hostAPISkillsRegistry interface {
	List() []*skillspkg.Skill
	ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skillspkg.Skill, error)
	ForAgent(
		ctx context.Context,
		resolved *workspacepkg.ResolvedWorkspace,
		agentName string,
	) ([]*skillspkg.Skill, error)
}

// WithHostAPICapabilityChecker injects the capability checker used for Host API authorization.
func WithHostAPICapabilityChecker(checker *CapabilityChecker) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.capChecker = checker
	}
}

// WithHostAPIAutomationManager injects the automation manager used for automation Host API methods.
func WithHostAPIAutomationManager(manager HostAPIAutomationManager) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.automation = manager
	}
}

// WithHostAPITaskManager injects the task manager used for task Host API methods.
func WithHostAPITaskManager(manager hostAPITaskManager) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.tasks = manager
	}
}

// WithHostAPINetworkService injects the network runtime used by network Host API methods.
func WithHostAPINetworkService(service hostAPINetworkService) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.network = service
	}
}

// WithHostAPINetworkStore injects the durable conversation store used by network Host API methods.
func WithHostAPINetworkStore(networkStore store.NetworkConversationStore) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.networkStore = networkStore
	}
}

// WithHostAPIAutomationGetter injects a lazy automation lookup used when the runtime boots after extensions.
func WithHostAPIAutomationGetter(getter func() HostAPIAutomationManager) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.automationGetter = getter
	}
}

// WithHostAPIWorkspaceResolver injects workspace resolution for workspace-scoped Host API methods.
func WithHostAPIWorkspaceResolver(resolver workspacepkg.RuntimeResolver) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.workspaces = resolver
	}
}

// WithHostAPIBridgeRegistry injects the bridge registry used by bridge Host API methods.
func WithHostAPIBridgeRegistry(registry hostAPIBridgeRegistry) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.bridges = registry
	}
}

// WithHostAPIBridgeDedupStore injects the dedup persistence used by inbound bridge ingest.
func WithHostAPIBridgeDedupStore(store hostAPIBridgeDedupStore) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.dedupStore = store
	}
}

// WithHostAPIDeliveryBroker injects the session-to-bridge delivery projection broker.
func WithHostAPIDeliveryBroker(broker hostAPIDeliveryBroker) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.deliveryBroker = broker
	}
}

// WithHostAPIResourceStore injects the canonical raw resource store used by
// the extension resource Host API methods.
func WithHostAPIResourceStore(store resources.RawStore) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.resourceStore = store
	}
}

// WithHostAPIResourceCodecRegistry injects resource codecs used to validate
// and canonicalize snapshot specs before persistence.
func WithHostAPIResourceCodecRegistry(registry *resources.CodecRegistry) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.resourceCodecs = registry
	}
}

// WithHostAPIResourceTrigger injects the reconcile trigger used after
// successful snapshot writes.
func WithHostAPIResourceTrigger(
	trigger func(context.Context, resources.ResourceKind, resources.ReconcileReason) error,
) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.resourceTrigger = trigger
	}
}

// WithHostAPISoulAuthoring injects managed SOUL.md read and mutation support.
func WithHostAPISoulAuthoring(service hostAPISoulAuthoringService) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.soulAuthoring = service
	}
}

// WithHostAPISoulRefresher injects managed session Soul refresh support.
func WithHostAPISoulRefresher(refresher hostAPISoulRefresher) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.soulRefresher = refresher
	}
}

// WithHostAPIHeartbeatAuthoring injects managed HEARTBEAT.md mutation support.
func WithHostAPIHeartbeatAuthoring(service hostAPIHeartbeatAuthoringService) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.heartbeatAuthor = service
	}
}

// WithHostAPIHeartbeatStatus injects managed Heartbeat status support.
func WithHostAPIHeartbeatStatus(service hostAPIHeartbeatStatusService) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.heartbeatStatus = service
	}
}

// WithHostAPIHeartbeatWake injects managed Heartbeat wake support.
func WithHostAPIHeartbeatWake(service hostAPIHeartbeatWakeService) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.heartbeatWake = service
	}
}

// WithHostAPISessionHealth injects metadata-only session health reads.
func WithHostAPISessionHealth(reader hostAPISessionHealthReader) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.sessionHealth = reader
	}
}

// WithHostAPIHeartbeatWakeEvents injects retained wake audit reads.
func WithHostAPIHeartbeatWakeEvents(reader hostAPIHeartbeatWakeEventReader) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.wakeEvents = reader
	}
}

// WithHostAPIBridgeIngressConfig overrides dedup TTL and cleanup cadence for bridge ingest.
func WithHostAPIBridgeIngressConfig(dedupTTL time.Duration, cleanupInterval time.Duration) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.bridgeIngestDedupTTL = dedupTTL
		handler.bridgeCleanupInterval = cleanupInterval
	}
}

// WithHostAPIMemoryProviderRegistry injects MemoryProvider registration state.
func WithHostAPIMemoryProviderRegistry(registry *MemoryProviderRegistry) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.memoryProviders = registry
	}
}

// WithHostAPIRateLimit overrides the per-extension Host API token bucket settings.
func WithHostAPIRateLimit(limit int, burst int) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.rateLimit = limit
		handler.rateBurst = burst
	}
}

// WithHostAPINow overrides the handler clock, mainly for tests.
func WithHostAPINow(now func() time.Time) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.now = now
	}
}

// NewHostAPIHandler constructs a Host API handler with sensible defaults.
func NewHostAPIHandler(
	sessions hostAPISessionManager,
	memoryStore *memory.Store,
	observer hostAPIObserver,
	skillsRegistry hostAPISkillsRegistry,
	opts ...HostAPIOption,
) *HostAPIHandler {
	handler := newHostAPIHandlerDefaults(sessions, memoryStore, observer, skillsRegistry)

	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}

	normalizeHostAPIHandlerDefaults(handler)
	handler.limiter = newHostAPIRateLimiter(handler.rateLimit, handler.rateBurst, handler.now)
	handler.methods = hostAPIMethodHandlers(handler)

	return handler
}

func newHostAPIHandlerDefaults(
	sessions hostAPISessionManager,
	memoryStore *memory.Store,
	observer hostAPIObserver,
	skillsRegistry hostAPISkillsRegistry,
) *HostAPIHandler {
	return &HostAPIHandler{
		sessions:              sessions,
		memory:                memoryStore,
		observer:              observer,
		skills:                skillsRegistry,
		capChecker:            &CapabilityChecker{},
		rateLimit:             defaultHostAPIRateLimit,
		rateBurst:             defaultHostAPIBurst,
		bridgeIngestDedupTTL:  defaultHostAPIBridgeIngestDedupTTL,
		bridgeCleanupInterval: defaultHostAPIBridgeCleanupInterval,
		bridgeLocks:           newHostAPIKeyLocker(),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func normalizeHostAPIHandlerDefaults(handler *HostAPIHandler) {
	if handler == nil {
		return
	}
	if handler.now == nil {
		handler.now = func() time.Time {
			return time.Now().UTC()
		}
	}
	if handler.capChecker == nil {
		handler.capChecker = &CapabilityChecker{}
	}
	if handler.bridgeIngestDedupTTL <= 0 {
		handler.bridgeIngestDedupTTL = defaultHostAPIBridgeIngestDedupTTL
	}
	if handler.bridgeCleanupInterval <= 0 {
		handler.bridgeCleanupInterval = defaultHostAPIBridgeCleanupInterval
	}
	if handler.bridgeLocks == nil {
		handler.bridgeLocks = newHostAPIKeyLocker()
	}
}

func hostAPIMethodHandlers(handler *HostAPIHandler) map[string]hostAPIMethodFunc {
	handlers := map[string]hostAPIMethodFunc{
		hostAPIAutomationJobsPath:                                      handler.handleAutomationJobs,
		hostAPIAutomationJobsGetPath:                                   handler.handleAutomationJobsGet,
		hostAPIAutomationJobsCreatePath:                                handler.handleAutomationJobsCreate,
		hostAPIAutomationJobsUpdatePath:                                handler.handleAutomationJobsUpdate,
		hostAPIAutomationJobsDeletePath:                                handler.handleAutomationJobsDelete,
		hostAPIAutomationJobsTriggerPath:                               handler.handleAutomationJobsTrigger,
		hostAPIAutomationJobsRunsPath:                                  handler.handleAutomationJobsRuns,
		hostAPIAutomationTriggersPath:                                  handler.handleAutomationTriggers,
		hostAPIAutomationTriggersGetPath:                               handler.handleAutomationTriggersGet,
		hostAPIAutomationTriggersCreatePath:                            handler.handleAutomationTriggersCreate,
		hostAPIAutomationTriggersUpdatePath:                            handler.handleAutomationTriggersUpdate,
		hostAPIAutomationTriggersDeletePath:                            handler.handleAutomationTriggersDelete,
		hostAPIAutomationTriggersRunsPath:                              handler.handleAutomationTriggersRuns,
		hostAPIAutomationTriggersFirePath:                              handler.handleAutomationTriggersFire,
		hostAPIAutomationRunsPath:                                      handler.handleAutomationRuns,
		string(extensioncontract.HostAPIMethodTasks):                   handler.handleTasks,
		string(extensioncontract.HostAPIMethodTasksGet):                handler.handleTasksGet,
		string(extensioncontract.HostAPIMethodTasksTimeline):           handler.handleTasksTimeline,
		string(extensioncontract.HostAPIMethodTasksTree):               handler.handleTasksTree,
		string(extensioncontract.HostAPIMethodTasksDashboard):          handler.handleTasksDashboard,
		string(extensioncontract.HostAPIMethodTasksInbox):              handler.handleTasksInbox,
		string(extensioncontract.HostAPIMethodTasksCreate):             handler.handleTasksCreate,
		string(extensioncontract.HostAPIMethodTasksUpdate):             handler.handleTasksUpdate,
		string(extensioncontract.HostAPIMethodTasksCancel):             handler.handleTasksCancel,
		string(extensioncontract.HostAPIMethodTasksRuns):               handler.handleTasksRuns,
		string(extensioncontract.HostAPIMethodTasksRunsGet):            handler.handleTasksRunsGet,
		string(extensioncontract.HostAPIMethodTasksRunsEnqueue):        handler.handleTasksRunsEnqueue,
		string(extensioncontract.HostAPIMethodTasksRunsClaim):          handler.handleTasksRunsClaim,
		string(extensioncontract.HostAPIMethodTasksRunsStart):          handler.handleTasksRunsStart,
		string(extensioncontract.HostAPIMethodTasksRunsAttachSession):  handler.handleTasksRunsAttachSession,
		string(extensioncontract.HostAPIMethodTasksRunsComplete):       handler.handleTasksRunsComplete,
		string(extensioncontract.HostAPIMethodTasksRunsFail):           handler.handleTasksRunsFail,
		string(extensioncontract.HostAPIMethodTasksRunsCancel):         handler.handleTasksRunsCancel,
		hostAPIResourcesListPath:                                       handler.handleResourcesList,
		hostAPIResourcesGetPath:                                        handler.handleResourcesGet,
		hostAPIResourcesSnapshotPath:                                   handler.handleResourcesSnapshot,
		hostAPIBridgesInstancesListPath:                                handler.handleBridgesInstancesList,
		hostAPIBridgesInstancesGetPath:                                 handler.handleBridgesInstancesGet,
		hostAPIBridgesInstancesReportStatePath:                         handler.handleBridgesInstancesReportState,
		hostAPIBridgesMessagesIngestPath:                               handler.handleBridgesMessagesIngest,
		hostAPISandboxExecPath:                                         handler.handleSandboxExec,
		hostAPISandboxInfoPath:                                         handler.handleSandboxInfo,
		hostAPISandboxListPath:                                         handler.handleSandboxList,
		hostAPIMemoryForgetPath:                                        handler.handleMemoryForget,
		hostAPIMemoryRecallPath:                                        handler.handleMemoryRecall,
		hostAPIMemoryStorePath:                                         handler.handleMemoryStore,
		hostAPIObserveEventsPath:                                       handler.handleObserveEvents,
		hostAPIObserveHealthPath:                                       handler.handleObserveHealth,
		string(extensioncontract.HostAPIMethodModelsList):              handler.handleModelsList,
		string(extensioncontract.HostAPIMethodModelsRefresh):           handler.handleModelsRefresh,
		string(extensioncontract.HostAPIMethodModelsStatus):            handler.handleModelsStatus,
		string(extensioncontract.HostAPIMethodAgentsSoulGet):           handler.handleAgentsSoulGet,
		string(extensioncontract.HostAPIMethodAgentsSoulValidate):      handler.handleAgentsSoulValidate,
		string(extensioncontract.HostAPIMethodAgentsSoulPut):           handler.handleAgentsSoulPut,
		string(extensioncontract.HostAPIMethodAgentsSoulDelete):        handler.handleAgentsSoulDelete,
		string(extensioncontract.HostAPIMethodAgentsSoulHistory):       handler.handleAgentsSoulHistory,
		string(extensioncontract.HostAPIMethodAgentsSoulRollback):      handler.handleAgentsSoulRollback,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatGet):      handler.handleAgentsHeartbeatGet,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatValidate): handler.handleAgentsHeartbeatValidate,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatPut):      handler.handleAgentsHeartbeatPut,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatDelete):   handler.handleAgentsHeartbeatDelete,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatHistory):  handler.handleAgentsHeartbeatHistory,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatRollback): handler.handleAgentsHeartbeatRollback,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatStatus):   handler.handleAgentsHeartbeatStatus,
		string(extensioncontract.HostAPIMethodAgentsHeartbeatWake):     handler.handleAgentsHeartbeatWake,
		hostAPISessionsCreatePath:                                      handler.handleSessionsCreate,
		hostAPISessionsEventsPath:                                      handler.handleSessionsEvents,
		string(extensioncontract.HostAPIMethodSessionsSoulRefresh):     handler.handleSessionsSoulRefresh,
		string(extensioncontract.HostAPIMethodSessionsHealthGet):       handler.handleSessionsHealthGet,
		hostAPISessionsListPath:                                        handler.handleSessionsList,
		hostAPISessionsPromptPath:                                      handler.handleSessionsPrompt,
		hostAPISessionsStatusPath:                                      handler.handleSessionsStatus,
		string(extensioncontract.HostAPIMethodSessionsStatusGet):       handler.handleSessionsStatusGet,
		hostAPISessionsStopPath:                                        handler.handleSessionsStop,
		hostAPISkillsListPath:                                          handler.handleSkillsList,
	}
	registerHostAPINetworkMethodHandlers(handler, handlers)
	return handlers
}

// Handle dispatches one Host API request for the named extension.
func (h *HostAPIHandler) Handle(
	ctx context.Context,
	extName string,
	method string,
	params json.RawMessage,
) (any, error) {
	if h == nil {
		return nil, errors.New("extension: host api handler is required")
	}
	if ctx == nil {
		return nil, errors.New("extension: host api context is required")
	}

	method = strings.TrimSpace(method)
	handler, ok := h.methods[method]
	if !ok {
		return nil, methodNotFoundRPCError(method)
	}

	if err := h.capChecker.CheckHostAPI(extName, method); err != nil {
		return nil, rpcCapabilityDenied(err)
	}
	if err := h.limiter.Allow(extName, method); err != nil {
		return nil, normalizeHostAPIRPCError(method, err)
	}

	result, err := handler(withHostAPIExtensionName(ctx, extName), params)
	if err != nil {
		return nil, normalizeHostAPIRPCError(method, err)
	}
	return result, nil
}

// HandleMethod returns a subprocess-compatible handler for one Host API method.
func (h *HostAPIHandler) HandleMethod(method string) subprocess.HandlerFunc {
	method = strings.TrimSpace(method)
	return func(ctx context.Context, params json.RawMessage) (any, error) {
		return h.Handle(ctx, hostAPIExtensionNameFromContext(ctx), method, params)
	}
}

// MethodHandlers returns the subprocess-compatible handler set for every Host API method.
func (h *HostAPIHandler) MethodHandlers() map[string]subprocess.HandlerFunc {
	out := make(map[string]subprocess.HandlerFunc, len(h.methods))
	for method := range h.methods {
		out[method] = h.HandleMethod(method)
	}
	return out
}

func withHostAPIBridgeRuntime(ctx context.Context, bridgeRuntime *subprocess.InitializeBridgeRuntime) context.Context {
	if ctx == nil || bridgeRuntime == nil {
		return ctx
	}
	return context.WithValue(
		ctx,
		hostAPIBridgeRuntimeContextKey,
		subprocess.CloneInitializeBridgeRuntime(bridgeRuntime),
	)
}

type hostAPIResourceSession struct {
	Actor resources.MutationActor
}

func withHostAPIResourceSession(ctx context.Context, session *hostAPIResourceSession) context.Context {
	if ctx == nil || session == nil {
		return ctx
	}
	cloned := &hostAPIResourceSession{Actor: cloneResourceMutationActor(session.Actor)}
	return context.WithValue(ctx, hostAPIResourceSessionContextKey, cloned)
}

func hostAPIResourceSessionFromContext(ctx context.Context) (*hostAPIResourceSession, bool) {
	if ctx == nil {
		return nil, false
	}
	value, ok := ctx.Value(hostAPIResourceSessionContextKey).(*hostAPIResourceSession)
	if !ok || value == nil {
		return nil, false
	}
	return &hostAPIResourceSession{Actor: cloneResourceMutationActor(value.Actor)}, true
}

func cloneResourceMutationActor(actor resources.MutationActor) resources.MutationActor {
	return resources.MutationActor{
		Kind:          actor.Kind,
		ID:            actor.ID,
		SessionNonce:  actor.SessionNonce,
		Source:        actor.Source,
		MaxScope:      actor.MaxScope,
		GrantedKinds:  append([]resources.ResourceKind(nil), actor.GrantedKinds...),
		GrantedScopes: append([]resources.ResourceScopeKind(nil), actor.GrantedScopes...),
	}
}

type hostAPISessionsListParams = extensioncontract.SessionsListParams

type hostAPISessionCreateParams = extensioncontract.SessionsCreateParams

type hostAPISessionPromptParams = extensioncontract.SessionsPromptParams

type hostAPISessionTargetParams = extensioncontract.SessionTargetParams

type hostAPISessionEventsParams = extensioncontract.SessionEventsParams

type hostAPISandboxListParams = extensioncontract.SandboxListParams

type hostAPISandboxInfoParams = extensioncontract.SandboxInfoParams

type hostAPISandboxExecParams = extensioncontract.SandboxExecParams

type hostAPIMemoryStoreParams = extensioncontract.MemoryStoreParams

type hostAPIMemoryRecallParams = extensioncontract.MemoryRecallParams

type hostAPIMemoryForgetParams = extensioncontract.MemoryForgetParams

type hostAPIObserveEventsParams = extensioncontract.ObserveEventsParams

type hostAPISkillsListParams = extensioncontract.SkillsListParams

type hostAPISessionSummary = extensioncontract.SessionSummary

type hostAPISessionStatus = extensioncontract.SessionStatus

type hostAPISessionEvent = extensioncontract.SessionEvent

type hostAPISessionCreateResult = extensioncontract.SessionCreateResult

type hostAPISessionPromptResult = extensioncontract.SessionPromptResult

type hostAPISandboxListResult = extensioncontract.SandboxListResult

type hostAPISandboxSummary = extensioncontract.SandboxSummary

type hostAPISandboxInfoResult = extensioncontract.SandboxInfoResult

type hostAPISandboxExecResult = extensioncontract.SandboxExecResult

type hostAPIMemoryRecallEntry = extensioncontract.MemoryRecallEntry

type hostAPISkillSummary = extensioncontract.SkillSummary

type hostAPIAutomationJobsParams = extensioncontract.AutomationJobsParams

type hostAPIAutomationTriggersParams = extensioncontract.AutomationTriggersParams

type hostAPIAutomationRunsParams = extensioncontract.AutomationRunsParams

type hostAPIAutomationTargetParams = extensioncontract.AutomationTargetParams

type hostAPIAutomationJobCreateParams = extensioncontract.AutomationJobCreateParams

type hostAPIAutomationJobUpdateParams = extensioncontract.AutomationJobUpdateParams

type hostAPIAutomationJobTriggerParams = extensioncontract.AutomationJobTriggerParams

type hostAPIAutomationJobRunsParams = extensioncontract.AutomationJobRunsParams

type hostAPIAutomationTriggerCreateParams = extensioncontract.AutomationTriggerCreateParams

type hostAPIAutomationTriggerUpdateParams = extensioncontract.AutomationTriggerUpdateParams

type hostAPIAutomationTriggerRunsParams = extensioncontract.AutomationTriggerRunsParams

type hostAPIAutomationTriggerFireParams = extensioncontract.AutomationTriggerFireParams

type hostAPITasksParams = extensioncontract.TasksParams

type hostAPITaskTargetParams = extensioncontract.TaskTargetParams

type hostAPITaskTimelineParams = extensioncontract.TaskTimelineParams

type hostAPITaskTreeParams = extensioncontract.TaskTreeParams

type hostAPITaskDashboardParams = extensioncontract.TaskDashboardParams

type hostAPITaskInboxParams = extensioncontract.TaskInboxParams

type hostAPITaskCreateParams = extensioncontract.TaskCreateParams

type hostAPITaskUpdateParams = extensioncontract.TaskUpdateParams

type hostAPITaskCancelParams = extensioncontract.TaskCancelParams

type hostAPITaskRunsParams = extensioncontract.TaskRunsParams

type hostAPITaskRunGetParams = extensioncontract.TaskRunGetParams

type hostAPITaskRunEnqueueParams = extensioncontract.TaskRunEnqueueParams

type hostAPITaskRunClaimParams = extensioncontract.TaskRunClaimParams

type hostAPITaskRunStartParams = extensioncontract.TaskRunStartParams

type hostAPITaskRunAttachSessionParams = extensioncontract.TaskRunAttachSessionParams

type hostAPITaskRunCompleteParams = extensioncontract.TaskRunCompleteParams

type hostAPITaskRunFailParams = extensioncontract.TaskRunFailParams

type hostAPITaskRunCancelParams = extensioncontract.TaskRunCancelParams

type hostAPIResourcesListParams = extensioncontract.ResourcesListParams

type hostAPIResourceGetParams = extensioncontract.ResourceGetParams

type hostAPIResourcesSnapshotParams = extensioncontract.ResourcesSnapshotParams

type hostAPIBridgesMessagesIngestParams = extensioncontract.BridgesMessagesIngestParams

type hostAPIBridgesMessagesIngestResult = extensioncontract.BridgesMessagesIngestResult

type hostAPIBridgeInstanceTargetParams = extensioncontract.BridgeInstanceTargetParams

type hostAPIBridgesInstancesReportStateParams = extensioncontract.BridgesInstancesReportStateParams

type hostAPIBridgeInstance = bridgepkg.BridgeInstance

type hostAPIResourceRecord = extensioncontract.ResourceRecord

func (h *HostAPIHandler) handleSessionsList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPISessionsListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	infos, err := h.sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	filterWorkspaceID := ""
	filterWorkspaceRoot := ""
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if h.workspaces != nil {
			resolved, resolveErr := h.workspaces.Resolve(ctx, workspaceRef)
			if resolveErr != nil {
				return nil, resolveErr
			}
			filterWorkspaceID, resolveErr = hostAPIResolvedWorkspaceID(&resolved)
			if resolveErr != nil {
				return nil, resolveErr
			}
			filterWorkspaceRoot = strings.TrimSpace(resolved.RootDir)
		} else {
			filterWorkspaceID = workspaceRef
			filterWorkspaceRoot = workspaceRef
		}
	}

	result := make([]hostAPISessionSummary, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		if filterWorkspaceID != "" || filterWorkspaceRoot != "" {
			if info.WorkspaceID != filterWorkspaceID && info.Workspace != filterWorkspaceRoot {
				continue
			}
		}
		result = append(result, hostAPISessionSummary{
			ID:        info.ID,
			Name:      info.Name,
			Agent:     info.AgentName,
			Provider:  info.Provider,
			Workspace: info.Workspace,
			State:     info.State,
			CreatedAt: info.CreatedAt,
		})
	}

	return result, nil
}

func (h *HostAPIHandler) handleSessionsCreate(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPISessionCreateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Agent) == "" {
		return nil, invalidParamsRPCError(errors.New("agent is required"))
	}

	sess, err := h.sessions.Create(ctx, session.CreateOpts{
		AgentName:       strings.TrimSpace(params.Agent),
		Provider:        strings.TrimSpace(params.Provider),
		Model:           strings.TrimSpace(params.Model),
		ReasoningEffort: strings.TrimSpace(params.ReasoningEffort),
		Workspace:       strings.TrimSpace(params.Workspace),
		Type:            session.SessionTypeSystem,
	})
	if err != nil {
		return nil, err
	}

	if prompt := strings.TrimSpace(params.Prompt); prompt != "" {
		if _, err := h.submitPrompt(ctx, sess.ID, prompt); err != nil {
			return nil, err
		}
	}

	return hostAPISessionCreateResult{
		SessionID: sess.ID,
		Provider:  sess.Info().Provider,
	}, nil
}

func (h *HostAPIHandler) handleSessionsPrompt(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionPromptParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	if strings.TrimSpace(params.Message) == "" {
		return nil, invalidParamsRPCError(errors.New("message is required"))
	}
	if _, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, params.SessionID); err != nil {
		return nil, err
	}

	submission, err := h.submitPrompt(ctx, params.SessionID, params.Message)
	if err != nil {
		return nil, err
	}

	return hostAPISessionPromptResult{TurnID: submission.TurnID}, nil
}

func (h *HostAPIHandler) handleSessionsStop(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	if _, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, params.SessionID); err != nil {
		return nil, err
	}
	if err := h.sessions.Stop(ctx, params.SessionID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleSessionsStatus(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}

	info, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, params.SessionID)
	if err != nil {
		return nil, err
	}
	return hostAPISessionStatusFromInfo(info), nil
}

func (h *HostAPIHandler) handleSessionsEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPISessionEventsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	if strings.TrimSpace(params.SessionID) == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	if _, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, params.SessionID); err != nil {
		return nil, err
	}

	events, err := h.sessions.Events(ctx, params.SessionID, store.EventQuery{
		Type:          strings.TrimSpace(params.Type),
		AgentName:     strings.TrimSpace(params.AgentName),
		TurnID:        strings.TrimSpace(params.TurnID),
		Since:         params.Since,
		Limit:         params.Limit,
		AfterSequence: params.Offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISessionEvent, 0, len(events))
	for _, event := range events {
		result = append(result, hostAPISessionEvent{
			Type:      event.Type,
			Timestamp: event.Timestamp,
			Data:      decodeJSONValue(event.Content),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) handleSandboxList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPISandboxListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	filterWorkspaceID, filterWorkspaceRoot, err := h.resolveSandboxWorkspaceFilter(ctx, params.Workspace)
	if err != nil {
		return nil, err
	}

	infos, err := h.sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := hostAPISandboxListResult{
		Sandboxes: make([]hostAPISandboxSummary, 0, len(infos)),
	}
	for _, info := range infos {
		if info == nil || info.Sandbox == nil || info.State == session.StateStopped {
			continue
		}
		if filterWorkspaceID != "" || filterWorkspaceRoot != "" {
			if info.WorkspaceID != filterWorkspaceID && info.Workspace != filterWorkspaceRoot {
				continue
			}
		}
		result.Sandboxes = append(result.Sandboxes, hostAPISandboxSummary{
			SessionID:  info.ID,
			SandboxID:  strings.TrimSpace(info.Sandbox.SandboxID),
			Backend:    strings.TrimSpace(info.Sandbox.Backend),
			Profile:    strings.TrimSpace(info.Sandbox.Profile),
			InstanceID: strings.TrimSpace(info.Sandbox.InstanceID),
			State:      strings.TrimSpace(info.Sandbox.State),
			SyncState:  hostAPISandboxSyncState(info.Sandbox),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) handleSandboxInfo(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	var params hostAPISandboxInfoParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	info, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, sessionID)
	if err != nil {
		return nil, err
	}
	if info == nil || info.Sandbox == nil {
		return nil, notFoundRPCError("sandbox", sessionID, errors.New("sandbox is not configured"))
	}
	return hostAPISandboxInfoResult{
		SandboxID:     strings.TrimSpace(info.Sandbox.SandboxID),
		Backend:       strings.TrimSpace(info.Sandbox.Backend),
		Profile:       strings.TrimSpace(info.Sandbox.Profile),
		InstanceID:    strings.TrimSpace(info.Sandbox.InstanceID),
		RuntimeRoot:   strings.TrimSpace(info.Sandbox.RuntimeRootDir),
		SyncState:     hostAPISandboxSyncState(info.Sandbox),
		CreatedAt:     info.CreatedAt,
		LastSyncError: strings.TrimSpace(info.Sandbox.LastSyncError),
	}, nil
}

func (h *HostAPIHandler) handleSandboxExec(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	var params hostAPISandboxExecParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	command := strings.TrimSpace(params.Command)
	if command == "" {
		return nil, invalidParamsRPCError(errors.New("command is required"))
	}
	if params.Timeout < 0 {
		return nil, invalidParamsRPCError(errors.New("timeout must be non-negative"))
	}
	if _, err := h.requireHostAPISessionWorkspace(ctx, params.WorkspaceID, sessionID); err != nil {
		return nil, err
	}
	result, err := h.sessions.ExecSandbox(ctx, session.SandboxExecRequest{
		SessionID: sessionID,
		Command:   command,
		Timeout:   time.Duration(params.Timeout) * time.Second,
	})
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, notFoundRPCError("session", sessionID, err)
		}
		if errors.Is(err, session.ErrSessionNotActive) {
			return nil, unavailableRPCError(err)
		}
		return nil, err
	}
	return hostAPISandboxExecResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (h *HostAPIHandler) resolveSandboxWorkspaceFilter(
	ctx context.Context,
	workspace string,
) (string, string, error) {
	workspace = strings.TrimSpace(workspace)
	if workspace == "" {
		return "", "", nil
	}
	if h.workspaces == nil {
		return workspace, workspace, nil
	}
	resolved, err := h.workspaces.Resolve(ctx, workspace)
	if err != nil {
		return "", "", err
	}
	workspaceID, err := hostAPIResolvedWorkspaceID(&resolved)
	if err != nil {
		return "", "", err
	}
	return workspaceID, strings.TrimSpace(resolved.RootDir), nil
}

func hostAPISandboxSyncState(meta *store.SessionSandboxMeta) string {
	if meta == nil {
		return ""
	}
	if strings.TrimSpace(meta.LastSyncError) != "" {
		return extensionStateError
	}
	if meta.LastSyncAt != nil {
		return hostAPISandboxStateSynced
	}
	return hostAPISandboxStatePending
}

func (h *HostAPIHandler) handleMemoryStore(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryStoreParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Key) == "" {
		return nil, invalidParamsRPCError(errors.New("key is required"))
	}
	if strings.TrimSpace(params.Content) == "" {
		return nil, invalidParamsRPCError(errors.New("content is required"))
	}

	storeHandle, scope, err := h.memoryStoreFor(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}

	filename := normalizeMemoryFilename(params.Key)
	doc, err := renderMemoryDocument(hostAPIMemoryDocument{
		Key:     filename,
		Scope:   scope,
		Content: params.Content,
		Tags:    params.Tags,
	})
	if err != nil {
		return nil, err
	}
	result, err := storeHandle.ProposeWrite(ctx, scope, filename, []byte(doc), memcontract.OriginTool)
	if err != nil {
		return nil, err
	}
	if result.Decision.Op == memcontract.OpReject {
		return nil, invalidParamsRPCError(fmt.Errorf("memory write rejected: %s", result.Decision.Reason))
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleMemoryRecall(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryRecallParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	query := strings.TrimSpace(params.Query)
	if query == "" {
		return nil, invalidParamsRPCError(errors.New("query is required"))
	}

	limit := params.Limit
	if limit <= 0 {
		limit = defaultHostAPIRecallLimit
	}

	packaged, err := h.recallMemory(ctx, query, hostAPIMemoryRecallSelection{
		Limit:     limit,
		Scope:     params.Scope,
		Workspace: params.Workspace,
	})
	if err != nil {
		return nil, err
	}

	return hostAPIMemoryRecallEntries(packaged, limit), nil
}

func (h *HostAPIHandler) handleMemoryForget(ctx context.Context, raw json.RawMessage) (any, error) {
	var params hostAPIMemoryForgetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.Key) == "" {
		return nil, invalidParamsRPCError(errors.New("key is required"))
	}

	storeHandle, scope, err := h.memoryStoreFor(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}
	if _, err := storeHandle.ProposeDelete(
		ctx,
		scope,
		normalizeMemoryFilename(params.Key),
		memcontract.OriginTool,
	); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleObserveHealth(ctx context.Context, _ json.RawMessage) (any, error) {
	if h.observer == nil {
		return nil, errors.New("extension: observer is not configured")
	}
	return h.observer.Health(ctx)
}

func (h *HostAPIHandler) handleObserveEvents(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.observer == nil {
		return nil, errors.New("extension: observer is not configured")
	}

	var params hostAPIObserveEventsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	workspaceID, err := h.hostAPINetworkWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	events, err := h.observer.QueryEvents(ctx, store.EventSummaryQuery{
		WorkspaceID: workspaceID,
		SessionID:   strings.TrimSpace(params.SessionID),
		AgentName:   strings.TrimSpace(params.AgentName),
		Type:        strings.TrimSpace(params.Type),
		Since:       params.Since,
		Limit:       params.Limit,
	})
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISessionEvent, 0, len(events))
	for _, event := range events {
		result = append(result, hostAPISessionEvent{
			Type:      event.Type,
			Timestamp: event.Timestamp,
			Data: map[string]any{
				hostAPIWorkspaceIDKey: event.WorkspaceID,
				hostAPISessionIDKey:   event.SessionID,
				hostAPIAgentNameKey:   event.AgentName,
				"summary":             event.Summary,
			},
		})
	}

	return result, nil
}

func (h *HostAPIHandler) handleSkillsList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.skills == nil {
		return nil, errors.New("extension: skills registry is not configured")
	}

	var params hostAPISkillsListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	var (
		skills []*skillspkg.Skill
		err    error
	)
	if workspaceRef := strings.TrimSpace(params.Workspace); workspaceRef != "" {
		if h.workspaces == nil {
			return nil, errors.New("extension: workspace resolver is not configured")
		}
		resolved, resolveErr := h.workspaces.Resolve(ctx, workspaceRef)
		if resolveErr != nil {
			return nil, resolveErr
		}
		if agentName := strings.TrimSpace(params.ForAgent); agentName != "" {
			skills, err = h.skills.ForAgent(ctx, &resolved, agentName)
		} else {
			skills, err = h.skills.ForWorkspace(ctx, &resolved)
		}
	} else if agentName := strings.TrimSpace(params.ForAgent); agentName != "" {
		skills, err = h.skills.ForAgent(ctx, nil, agentName)
	} else {
		skills = h.skills.List()
	}
	if err != nil {
		return nil, err
	}

	result := make([]hostAPISkillSummary, 0, len(skills))
	for _, skill := range skills {
		if skill == nil {
			continue
		}
		result = append(result, hostAPISkillSummary{
			Name:        skill.Meta.Name,
			Description: skill.Meta.Description,
			Source:      skillspkg.SkillSourceName(skill.Source),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) handleAutomationJobs(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationJobsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	jobs, err := automation.ListJobs(ctx, automationpkg.JobListQuery{
		Scope:       params.Scope,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	if params.Enabled == nil {
		return jobs, nil
	}

	filtered := make([]automationpkg.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.Enabled == *params.Enabled {
			filtered = append(filtered, job)
		}
	}
	return filtered, nil
}

func (h *HostAPIHandler) handleAutomationJobsGet(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	return automation.GetJob(ctx, params.ID)
}

func (h *HostAPIHandler) handleAutomationJobsCreate(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationJobCreateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	job, err := h.jobFromCreateParams(ctx, params)
	if err != nil {
		return nil, err
	}
	if err := job.Validate("job"); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	return automation.CreateJob(ctx, job)
}

func (h *HostAPIHandler) handleAutomationJobsUpdate(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationJobUpdateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}
	if !params.HasChanges() {
		return nil, invalidParamsRPCError(errors.New("automation job update must include at least one field"))
	}

	current, err := automation.GetJob(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if current.Source == automationpkg.JobSourceConfig {
		if err := validateHostAPIConfigJobUpdate(params.UpdateJobRequest); err != nil {
			return nil, invalidParamsRPCError(err)
		}
		return automation.SetJobEnabled(ctx, current.ID, *params.Enabled)
	}

	next, err := h.applyJobUpdateParams(ctx, current, params.UpdateJobRequest)
	if err != nil {
		return nil, err
	}
	if err := next.Validate("job"); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	return automation.UpdateJob(ctx, next)
}

func (h *HostAPIHandler) handleAutomationJobsDelete(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	current, err := automation.GetJob(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if current.Source == automationpkg.JobSourceConfig {
		return nil, invalidParamsRPCError(errors.New("config-backed automation jobs cannot be deleted"))
	}
	if err := automation.DeleteJob(ctx, current.ID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleAutomationJobsTrigger(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationJobTriggerParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	return automation.TriggerJobWithPayload(ctx, params.ID, params.Payload)
}

func (h *HostAPIHandler) handleAutomationJobsRuns(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationJobRunsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	job, err := automation.GetJob(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	return automation.ListRuns(ctx, automationpkg.RunQuery{
		JobID:  job.ID,
		Status: params.Status,
		Limit:  params.Limit,
	})
}

func (h *HostAPIHandler) handleAutomationTriggers(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTriggersParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	triggers, err := automation.ListTriggers(ctx, automationpkg.TriggerListQuery{
		Scope:       params.Scope,
		WorkspaceID: workspaceID,
		Event:       strings.TrimSpace(params.Event),
	})
	if err != nil {
		return nil, err
	}

	if params.Enabled == nil {
		return apicontract.TriggerPayloadsFromTriggers(triggers), nil
	}

	filtered := make([]automationpkg.Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		if trigger.Enabled == *params.Enabled {
			filtered = append(filtered, trigger)
		}
	}
	return apicontract.TriggerPayloadsFromTriggers(filtered), nil
}

func (h *HostAPIHandler) handleAutomationTriggersGet(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	trigger, err := automation.GetTrigger(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	return apicontract.TriggerPayloadFromTrigger(trigger), nil
}

func (h *HostAPIHandler) handleAutomationTriggersCreate(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTriggerCreateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	trigger, webhookSecret, err := h.triggerFromCreateParams(ctx, params)
	if err != nil {
		return nil, err
	}

	created, err := automation.CreateTrigger(ctx, trigger, webhookSecret)
	if err != nil {
		return nil, err
	}
	return apicontract.TriggerPayloadFromTrigger(created), nil
}

func (h *HostAPIHandler) handleAutomationTriggersUpdate(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTriggerUpdateParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}
	if !params.HasChanges() {
		return nil, invalidParamsRPCError(errors.New("automation trigger update must include at least one field"))
	}

	current, err := automation.GetTrigger(ctx, params.ID)
	if err != nil {
		return nil, err
	}

	if current.Source == automationpkg.JobSourceConfig {
		if err := validateHostAPIConfigTriggerUpdate(params.UpdateTriggerRequest); err != nil {
			return nil, invalidParamsRPCError(err)
		}
		updated, err := automation.SetTriggerEnabled(ctx, current.ID, *params.Enabled)
		if err != nil {
			return nil, err
		}
		return apicontract.TriggerPayloadFromTrigger(updated), nil
	}

	next, webhookSecret, err := h.applyTriggerUpdateParams(ctx, current, params.UpdateTriggerRequest)
	if err != nil {
		return nil, err
	}

	updated, err := automation.UpdateTrigger(ctx, next, webhookSecret)
	if err != nil {
		return nil, err
	}
	return apicontract.TriggerPayloadFromTrigger(updated), nil
}

func (h *HostAPIHandler) handleAutomationTriggersDelete(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTargetParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	current, err := automation.GetTrigger(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	if current.Source == automationpkg.JobSourceConfig {
		return nil, invalidParamsRPCError(errors.New("config-backed automation triggers cannot be deleted"))
	}
	if err := automation.DeleteTrigger(ctx, current.ID); err != nil {
		return nil, err
	}
	return struct{}{}, nil
}

func (h *HostAPIHandler) handleAutomationTriggersRuns(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTriggerRunsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	if strings.TrimSpace(params.ID) == "" {
		return nil, invalidParamsRPCError(errors.New("id is required"))
	}

	trigger, err := automation.GetTrigger(ctx, params.ID)
	if err != nil {
		return nil, err
	}
	return automation.ListRuns(ctx, automationpkg.RunQuery{
		TriggerID: trigger.ID,
		Status:    params.Status,
		Limit:     params.Limit,
	})
}

func (h *HostAPIHandler) handleAutomationTriggersFire(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationTriggerFireParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, params.WorkspaceID)
	if err != nil {
		return nil, err
	}

	request := automationpkg.ExtensionTriggerRequest{
		Event:       strings.TrimSpace(params.Event),
		Scope:       params.Scope,
		WorkspaceID: workspaceID,
		Payload:     cloneJSONMap(params.Payload),
	}
	if err := request.Validate("trigger_fire"); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	return automation.FireExtensionTrigger(ctx, request)
}

func (h *HostAPIHandler) handleAutomationRuns(ctx context.Context, raw json.RawMessage) (any, error) {
	automation, err := h.automationManager()
	if err != nil {
		return nil, err
	}

	var params hostAPIAutomationRunsParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	return automation.ListRuns(ctx, automationpkg.RunQuery{
		JobID:     strings.TrimSpace(params.JobID),
		TriggerID: strings.TrimSpace(params.TriggerID),
		Status:    params.Status,
		Limit:     params.Limit,
	})
}

type hostAPIPromptSubmission struct {
	TurnID     string
	SeedEvents []bridgepkg.DeliveryProjectionEvent
}

func (h *HostAPIHandler) submitPrompt(
	ctx context.Context,
	sessionID string,
	message string,
) (hostAPIPromptSubmission, error) {
	if h.sessions == nil {
		return hostAPIPromptSubmission{}, errors.New("extension: session manager is not configured")
	}

	lastSequence, err := h.latestSessionSequence(ctx, sessionID)
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}

	promptCtx := context.WithoutCancel(ctx)
	eventsCh, err := h.sessions.Prompt(promptCtx, sessionID, message)
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}
	drainAgentEvents(eventsCh)

	events, err := h.sessions.Events(ctx, sessionID, store.EventQuery{
		AfterSequence: lastSequence,
	})
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}

	return promptSubmissionFromStoredEvents(events)
}

func promptSubmissionFromStoredEvents(events []store.SessionEvent) (hostAPIPromptSubmission, error) {
	turnID := promptTurnIDFromStoredEvents(events)
	if turnID == "" {
		return hostAPIPromptSubmission{}, errors.New("extension: prompt turn id not found after prompt submission")
	}

	seedEvents, err := promptSeedEventsFromStoredEvents(events, turnID)
	if err != nil {
		return hostAPIPromptSubmission{}, err
	}

	return hostAPIPromptSubmission{
		TurnID:     turnID,
		SeedEvents: seedEvents,
	}, nil
}

func promptTurnIDFromStoredEvents(events []store.SessionEvent) string {
	for _, event := range events {
		if !isPromptInitiatingStoredEventType(event.Type) {
			continue
		}
		turnID := strings.TrimSpace(event.TurnID)
		if turnID == "" {
			continue
		}
		return turnID
	}
	return ""
}

func isPromptInitiatingStoredEventType(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case acp.EventTypeUserMessage, acp.EventTypeSyntheticReentry:
		return true
	default:
		return false
	}
}

func promptSeedEventsFromStoredEvents(
	events []store.SessionEvent,
	turnID string,
) ([]bridgepkg.DeliveryProjectionEvent, error) {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" || len(events) == 0 {
		return nil, nil
	}

	seedEvents := make([]bridgepkg.DeliveryProjectionEvent, 0, len(events))
	for _, storedEvent := range events {
		if strings.TrimSpace(storedEvent.TurnID) != turnID {
			continue
		}

		projected, err := promptProjectionEventFromStoredEvent(storedEvent)
		if err != nil {
			return nil, err
		}
		seedEvents = append(seedEvents, projected)
	}
	return seedEvents, nil
}

func promptProjectionEventFromStoredEvent(storedEvent store.SessionEvent) (bridgepkg.DeliveryProjectionEvent, error) {
	decoded, err := transcript.UnmarshalAgentEvent(storedEvent.Content)
	if err != nil {
		return bridgepkg.DeliveryProjectionEvent{}, fmt.Errorf("extension: decode prompt seed event: %w", err)
	}
	if strings.TrimSpace(decoded.Type) == "" {
		decoded.Type = strings.TrimSpace(storedEvent.Type)
	}
	if strings.TrimSpace(decoded.TurnID) == "" {
		decoded.TurnID = strings.TrimSpace(storedEvent.TurnID)
	}
	if decoded.Timestamp.IsZero() {
		decoded.Timestamp = storedEvent.Timestamp
	}

	return bridgepkg.DeliveryProjectionEvent{
		Type:        decoded.Type,
		TurnID:      decoded.TurnID,
		Timestamp:   decoded.Timestamp,
		Text:        decoded.Text,
		Error:       decoded.Error,
		Fingerprint: strings.TrimSpace(storedEvent.Content),
	}, nil
}

func (h *HostAPIHandler) latestSessionSequence(ctx context.Context, sessionID string) (int64, error) {
	events, err := h.sessions.Events(ctx, sessionID, store.EventQuery{Limit: 1})
	if err != nil {
		return 0, err
	}
	if len(events) == 0 {
		return 0, nil
	}
	return events[len(events)-1].Sequence, nil
}

type hostAPIMemoryRecallSelection struct {
	Limit     int
	Scope     memcontract.Scope
	Workspace string
}

func (h *HostAPIHandler) recallMemory(
	ctx context.Context,
	query string,
	selection hostAPIMemoryRecallSelection,
) (memcontract.Packaged, error) {
	workspaceID, err := h.resolveWorkspaceID(ctx, selection.Workspace)
	if err != nil {
		return memcontract.Packaged{}, err
	}
	if selection.Scope.Normalize() == memcontract.ScopeGlobal {
		workspaceID = ""
	}
	if providerRecall, ok, err := h.recallMemoryFromProvider(
		ctx,
		query,
		workspaceID,
		selection.Limit,
	); ok ||
		err != nil {
		return providerRecall, err
	}
	return h.recallMemoryFromStore(ctx, query, workspaceID, selection)
}

func (h *HostAPIHandler) recallMemoryFromProvider(
	ctx context.Context,
	query string,
	workspaceID string,
	limit int,
) (memcontract.Packaged, bool, error) {
	if h.memoryProviders == nil {
		return memcontract.Packaged{}, false, nil
	}
	registration, err := h.memoryProviders.Select(ctx, workspaceID, "")
	if err != nil {
		if errors.Is(err, ErrMemoryProviderNotFound) {
			return memcontract.Packaged{}, false, nil
		}
		return memcontract.Packaged{}, true, err
	}
	recalled, err := registration.Provider.Recall(ctx, memcontract.RecallRequest{
		Query: memcontract.Query{
			WorkspaceID: workspaceID,
			QueryText:   query,
		},
		Options: memcontract.RecallOptions{TopK: limit},
	})
	if err != nil {
		if errors.Is(err, memcontract.ErrNotImplemented) {
			return memcontract.Packaged{}, false, nil
		}
		return memcontract.Packaged{}, true, err
	}
	return recalled.Packaged, true, nil
}

func (h *HostAPIHandler) recallMemoryFromStore(
	ctx context.Context,
	query string,
	workspaceID string,
	selection hostAPIMemoryRecallSelection,
) (memcontract.Packaged, error) {
	storeHandle, _, err := h.memoryStoreFor(ctx, string(selection.Scope), selection.Workspace)
	if err != nil {
		return memcontract.Packaged{}, err
	}
	return storeHandle.Recall(ctx, memcontract.Query{
		WorkspaceID: workspaceID,
		QueryText:   query,
	}, memcontract.RecallOptions{TopK: selection.Limit})
}

func (h *HostAPIHandler) memoryStoreFor(
	ctx context.Context,
	rawScope string,
	rawWorkspace string,
) (*memory.Store, memcontract.Scope, error) {
	if h.memory == nil {
		return nil, "", errors.New("extension: memory store is not configured")
	}

	scope := memcontract.Scope(strings.TrimSpace(rawScope)).Normalize()
	workspaceRoot, err := h.resolveWorkspaceRoot(ctx, rawWorkspace)
	if err != nil {
		return nil, "", err
	}
	if scope == "" {
		if workspaceRoot != "" {
			scope = memcontract.ScopeWorkspace
		} else {
			scope = memcontract.ScopeGlobal
		}
	}

	switch scope {
	case memcontract.ScopeGlobal:
		return h.memory, memcontract.ScopeGlobal, nil
	case memcontract.ScopeWorkspace:
		if workspaceRoot == "" {
			return nil, "", invalidParamsRPCError(errors.New("workspace is required for workspace memory scope"))
		}
		return h.memory.ForWorkspace(workspaceRoot), memcontract.ScopeWorkspace, nil
	default:
		return nil, "", invalidParamsRPCError(fmt.Errorf("memory scope must be one of global or workspace"))
	}
}

func (h *HostAPIHandler) resolveWorkspaceRoot(ctx context.Context, rawWorkspace string) (string, error) {
	if strings.TrimSpace(rawWorkspace) == "" {
		return "", nil
	}
	if h.workspaces == nil {
		return "", invalidParamsRPCError(errors.New("workspace resolver is not configured"))
	}
	resolved, err := h.workspaces.Resolve(ctx, rawWorkspace)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resolved.RootDir), nil
}

func (h *HostAPIHandler) resolveWorkspaceID(ctx context.Context, rawWorkspace string) (string, error) {
	trimmed := strings.TrimSpace(rawWorkspace)
	if trimmed == "" {
		return "", nil
	}
	if h.workspaces == nil {
		return trimmed, nil
	}
	resolved, err := h.workspaces.Resolve(ctx, trimmed)
	if err != nil {
		return "", err
	}
	return hostAPIResolvedWorkspaceID(&resolved)
}

func (h *HostAPIHandler) resolveRequiredWorkspaceID(ctx context.Context, rawWorkspace string) (string, error) {
	workspaceID, err := h.resolveWorkspaceID(ctx, rawWorkspace)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(workspaceID) == "" {
		return "", invalidParamsRPCError(errors.New("workspace_id is required"))
	}
	return strings.TrimSpace(workspaceID), nil
}

func (h *HostAPIHandler) requireHostAPISessionWorkspace(
	ctx context.Context,
	workspaceRef string,
	sessionID string,
) (*session.Info, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	workspaceID, err := h.resolveRequiredWorkspaceID(ctx, workspaceRef)
	if err != nil {
		return nil, err
	}
	info, err := h.sessions.Status(ctx, id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, notFoundRPCError("session", id, err)
		}
		return nil, err
	}
	if info == nil || strings.TrimSpace(info.WorkspaceID) != workspaceID {
		return nil, notFoundRPCError(
			"session",
			id,
			fmt.Errorf("%w: session=%q workspace_id=%q", session.ErrSessionNotFound, id, workspaceID),
		)
	}
	return info, nil
}

func (h *HostAPIHandler) automationManager() (HostAPIAutomationManager, error) {
	if h == nil {
		return nil, errors.New("extension: host api handler is required")
	}
	if h.automation != nil {
		return h.automation, nil
	}
	if h.automationGetter != nil {
		if automation := h.automationGetter(); automation != nil {
			return automation, nil
		}
	}
	return nil, errors.New("extension: automation manager is not configured")
}

func (h *HostAPIHandler) resolveAutomationWorkspaceID(ctx context.Context, rawWorkspace string) (string, error) {
	trimmed := strings.TrimSpace(rawWorkspace)
	if trimmed == "" {
		return "", nil
	}
	if h.workspaces == nil {
		return trimmed, nil
	}
	resolved, err := h.workspaces.Resolve(ctx, trimmed)
	if err != nil {
		return "", err
	}
	return hostAPIResolvedWorkspaceID(&resolved)
}

func hostAPIResolvedWorkspaceID(resolved *workspacepkg.ResolvedWorkspace) (string, error) {
	if resolved == nil {
		return "", errors.New("extension: resolved workspace is required")
	}
	workspaceID := strings.TrimSpace(resolved.WorkspaceID)
	if workspaceID == "" {
		return "", errors.New("extension: resolved workspace_id is empty")
	}
	return workspaceID, nil
}

func (h *HostAPIHandler) jobFromCreateParams(
	ctx context.Context,
	req hostAPIAutomationJobCreateParams,
) (automationpkg.Job, error) {
	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, req.WorkspaceID)
	if err != nil {
		return automationpkg.Job{}, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retry := automationpkg.DefaultRetryConfig()
	if req.Retry != nil {
		retry = *req.Retry
	}

	fireLimit := automationpkg.DefaultFireLimitConfig()
	if req.FireLimit != nil {
		fireLimit = *req.FireLimit
	}

	schedule := req.Schedule
	return automationpkg.Job{
		Scope:       req.Scope,
		Name:        strings.TrimSpace(req.Name),
		AgentName:   strings.TrimSpace(req.AgentName),
		WorkspaceID: workspaceID,
		Prompt:      strings.TrimSpace(req.Prompt),
		Schedule:    &schedule,
		Enabled:     enabled,
		Retry:       retry,
		FireLimit:   fireLimit,
		Source:      automationpkg.JobSourceDynamic,
	}, nil
}

func (h *HostAPIHandler) applyJobUpdateParams(
	ctx context.Context,
	current automationpkg.Job,
	req apicontract.UpdateJobRequest,
) (automationpkg.Job, error) {
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.AgentName != nil {
		next.AgentName = strings.TrimSpace(*req.AgentName)
	}
	if req.WorkspaceID != nil {
		workspaceID, err := h.resolveAutomationWorkspaceID(ctx, *req.WorkspaceID)
		if err != nil {
			return automationpkg.Job{}, err
		}
		next.WorkspaceID = workspaceID
	}
	if req.Prompt != nil {
		next.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.Schedule != nil {
		schedule := *req.Schedule
		next.Schedule = &schedule
	}
	if req.Enabled != nil {
		next.Enabled = *req.Enabled
	}
	if req.Retry != nil {
		next.Retry = *req.Retry
	}
	if req.FireLimit != nil {
		next.FireLimit = *req.FireLimit
	}
	return next, nil
}

func validateHostAPIConfigJobUpdate(req apicontract.UpdateJobRequest) error {
	switch {
	case req.Enabled == nil:
		return errors.New("config-backed automation jobs only accept enabled updates")
	case hostAPIJobUpdateTouchesImmutableConfigFields(req):
		return errors.New("config-backed automation jobs only accept enabled updates")
	default:
		return nil
	}
}

func (h *HostAPIHandler) triggerFromCreateParams(
	ctx context.Context,
	req hostAPIAutomationTriggerCreateParams,
) (automationpkg.Trigger, automationpkg.WebhookSecretWrite, error) {
	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, req.WorkspaceID)
	if err != nil {
		return automationpkg.Trigger{}, automationpkg.WebhookSecretWrite{}, err
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	retry := automationpkg.DefaultRetryConfig()
	if req.Retry != nil {
		retry = *req.Retry
	}

	fireLimit := automationpkg.DefaultFireLimitConfig()
	if req.FireLimit != nil {
		fireLimit = *req.FireLimit
	}

	trigger := automationpkg.Trigger{
		Scope:        req.Scope,
		Name:         strings.TrimSpace(req.Name),
		AgentName:    strings.TrimSpace(req.AgentName),
		WorkspaceID:  workspaceID,
		Prompt:       strings.TrimSpace(req.Prompt),
		Event:        strings.TrimSpace(req.Event),
		Filter:       cloneHostAPIStringMap(req.Filter),
		Enabled:      enabled,
		Retry:        retry,
		FireLimit:    fireLimit,
		Source:       automationpkg.JobSourceDynamic,
		WebhookID:    strings.TrimSpace(req.WebhookID),
		EndpointSlug: strings.TrimSpace(req.EndpointSlug),
	}
	write := automationpkg.WebhookSecretWrite{}
	if strings.TrimSpace(req.WebhookSecretValue) != "" {
		value := strings.TrimSpace(req.WebhookSecretValue)
		write.Value = &value
	}
	return trigger, write, nil
}

func (h *HostAPIHandler) applyTriggerUpdateParams(
	ctx context.Context,
	current automationpkg.Trigger,
	req apicontract.UpdateTriggerRequest,
) (automationpkg.Trigger, *automationpkg.WebhookSecretWrite, error) {
	next := current
	if req.Name != nil {
		next.Name = strings.TrimSpace(*req.Name)
	}
	if req.AgentName != nil {
		next.AgentName = strings.TrimSpace(*req.AgentName)
	}
	if req.WorkspaceID != nil {
		workspaceID, err := h.resolveAutomationWorkspaceID(ctx, *req.WorkspaceID)
		if err != nil {
			return automationpkg.Trigger{}, nil, err
		}
		next.WorkspaceID = workspaceID
	}
	if req.Prompt != nil {
		next.Prompt = strings.TrimSpace(*req.Prompt)
	}
	if req.Event != nil {
		next.Event = strings.TrimSpace(*req.Event)
	}
	if req.Filter != nil {
		next.Filter = cloneHostAPIStringMap(req.Filter)
	}
	if req.Enabled != nil {
		next.Enabled = *req.Enabled
	}
	if req.Retry != nil {
		next.Retry = *req.Retry
	}
	if req.FireLimit != nil {
		next.FireLimit = *req.FireLimit
	}

	webhookSecret := applyTriggerWebhookUpdateParams(&next, req)
	return next, webhookSecret, nil
}

func applyTriggerWebhookUpdateParams(
	next *automationpkg.Trigger,
	req apicontract.UpdateTriggerRequest,
) *automationpkg.WebhookSecretWrite {
	event := strings.TrimSpace(next.Event)
	if req.WebhookID != nil {
		next.WebhookID = strings.TrimSpace(*req.WebhookID)
	} else if !strings.EqualFold(event, "webhook") {
		next.WebhookID = ""
	}
	if req.EndpointSlug != nil {
		next.EndpointSlug = strings.TrimSpace(*req.EndpointSlug)
	} else if !strings.EqualFold(event, "webhook") {
		next.EndpointSlug = ""
	}
	if !strings.EqualFold(event, "webhook") {
		next.WebhookSecretRef = ""
	}

	if req.WebhookSecretValue == nil {
		return nil
	}
	write := automationpkg.WebhookSecretWrite{}
	value := strings.TrimSpace(*req.WebhookSecretValue)
	write.Value = &value
	return &write
}

func validateHostAPIConfigTriggerUpdate(req apicontract.UpdateTriggerRequest) error {
	switch {
	case req.Enabled == nil:
		return errors.New("config-backed automation triggers only accept enabled updates")
	case hostAPITriggerUpdateTouchesImmutableConfigFields(req):
		return errors.New("config-backed automation triggers only accept enabled updates")
	default:
		return nil
	}
}

type hostAPIMemoryDocument struct {
	Key     string
	Scope   memcontract.Scope
	Content string
	Tags    []string
}

func renderMemoryDocument(doc hostAPIMemoryDocument) (string, error) {
	header := memcontract.Header{
		Name:        memoryNameFromFilename(doc.Key),
		Description: memoryDescriptionFromContent(doc.Content),
		Type:        memoryTypeForScope(doc.Scope, doc.Tags),
	}
	if err := header.Validate(); err != nil {
		return "", invalidParamsRPCError(err)
	}

	metadata, err := yaml.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("extension: marshal memory frontmatter: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.Write(metadata)
	builder.WriteString("---\n\n")
	body := strings.TrimSpace(doc.Content)
	tags := normalizeUniqueStrings(doc.Tags)
	if len(tags) > 0 {
		builder.WriteString(tagCommentPrefix)
		builder.WriteByte(' ')
		builder.WriteString(strings.Join(tags, ", "))
		builder.WriteString(" -->\n\n")
	}
	builder.WriteString(body)
	return builder.String(), nil
}

func memoryTypeForScope(scope memcontract.Scope, tags []string) memcontract.Type {
	for _, tag := range normalizeUniqueStrings(tags) {
		switch memcontract.Type(tag).Normalize() {
		case memcontract.TypeUser, memcontract.TypeFeedback, memcontract.TypeProject, memcontract.TypeReference:
			return memcontract.Type(tag).Normalize()
		}
	}
	if scope == memcontract.ScopeWorkspace {
		return memcontract.TypeProject
	}
	return memcontract.TypeUser
}

func memoryNameFromFilename(filename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(filename)), filepath.Ext(strings.TrimSpace(filename)))
	if base == "" {
		return ""
	}

	normalized := strings.NewReplacer("-", " ", "_", " ", ".", " ").Replace(base)
	parts := strings.Fields(normalized)
	for i, part := range parts {
		parts[i] = titleCaseWord(part)
	}
	return strings.Join(parts, " ")
}

func titleCaseWord(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 1 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:1]) + strings.ToLower(trimmed[1:])
}

func memoryDescriptionFromContent(content string) string {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(content), "\n")[0])
	if len(firstLine) <= maxMemoryDescriptionLength {
		return firstLine
	}
	return strings.TrimSpace(firstLine[:maxMemoryDescriptionLength]) + "..."
}

func normalizeMemoryFilename(key string) string {
	filename := strings.TrimSpace(key)
	if filepath.Ext(filename) == "" {
		filename += ".md"
	}
	return filename
}

func hostAPIMemoryRecallEntries(packaged memcontract.Packaged, limit int) []hostAPIMemoryRecallEntry {
	entries := make([]hostAPIMemoryRecallEntry, 0)
	for _, block := range packaged.Blocks {
		for _, entry := range block.Entries {
			entries = append(entries, hostAPIMemoryRecallEntry{
				Key:     strings.TrimSpace(entry.ID),
				Content: taskpkg.RedactClaimTokens(strings.TrimSpace(entry.Body)),
				Score:   float64(len(entries) + 1),
			})
		}
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i].Score, entries[j].Score = entries[j].Score, entries[i].Score
	}
	if limit > 0 && len(entries) > limit {
		return entries[:limit]
	}
	return entries
}

func hostAPISessionStatusFromInfo(info *session.Info) hostAPISessionStatus {
	if info == nil {
		return hostAPISessionStatus{}
	}
	return hostAPISessionStatus{
		SessionID:    info.ID,
		Name:         info.Name,
		Agent:        info.AgentName,
		Provider:     info.Provider,
		WorkspaceID:  info.WorkspaceID,
		Workspace:    info.Workspace,
		State:        info.State,
		StopReason:   info.StopReason,
		StopDetail:   info.StopDetail,
		ACPSessionID: info.ACPSessionID,
		CreatedAt:    info.CreatedAt,
		UpdatedAt:    info.UpdatedAt,
	}
}

func decodeHostAPIParams(raw json.RawMessage, target any) error {
	if target == nil {
		return errors.New("extension: host api params target is required")
	}
	payload := bytes.TrimSpace(raw)
	if len(payload) == 0 || bytes.Equal(payload, []byte("null")) {
		payload = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return invalidParamsRPCError(fmt.Errorf("decode params: %w", err))
	}
	return nil
}

func decodeJSONValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded
	}
	return trimmed
}

func invalidParamsRPCError(err error) error {
	if err == nil {
		return subprocess.NewRPCError(HostAPIInvalidParamsCode, "Invalid params", nil)
	}
	return subprocess.NewRPCError(
		HostAPIInvalidParamsCode,
		"Invalid params",
		map[string]string{extensionStateError: err.Error()},
	)
}

func unavailableRPCError(err error) error {
	if err == nil {
		return subprocess.NewRPCError(HostAPIUnavailableCode, "Unavailable", nil)
	}
	return subprocess.NewRPCError(
		HostAPIUnavailableCode,
		"Unavailable",
		map[string]string{extensionStateError: err.Error()},
	)
}

func notFoundRPCError(resource string, id string, err error) error {
	data := map[string]string{
		hostAPIResourceKey: strings.TrimSpace(resource),
		"id":               strings.TrimSpace(id),
	}
	if err != nil {
		data[extensionStateError] = err.Error()
	}
	return subprocess.NewRPCError(HostAPINotFoundCode, "Not found", data)
}

func methodNotFoundRPCError(method string) error {
	return subprocess.NewRPCError(
		HostAPIMethodNotFoundCode,
		"Method not found",
		map[string]string{hostAPIMethodKey: strings.TrimSpace(method)},
	)
}

func rpcCapabilityDenied(err error) error {
	var denied *ErrCapabilityDenied
	if !errors.As(err, &denied) {
		return err
	}
	if isResourceHostAPIMethod(denied.Data.Method) {
		return hostAPIStatusRPCError(403, "Forbidden", map[string]any{
			extensionStateError: denied.Error(),
			hostAPIMethodKey:    strings.TrimSpace(denied.Data.Method),
			"required":          append([]string(nil), denied.Data.Required...),
			"granted":           append([]string(nil), denied.Data.Granted...),
		})
	}
	return subprocess.NewRPCError(denied.Code(), "Capability denied", denied.Data)
}

func normalizeHostAPIRPCError(method string, err error) error {
	if err == nil {
		return nil
	}
	if !isResourceHostAPIMethod(method) {
		return err
	}

	if rpcErr, ok := errors.AsType[*subprocess.RPCError](err); ok {
		if rpcErr.Code == HostAPIRateLimitedCode {
			return hostAPIStatusRPCError(429, "Rate limited", rpcErr.Data)
		}
		return err
	}

	switch {
	case errors.Is(err, resources.ErrPermissionDenied), errors.Is(err, resources.ErrDirectMutationNotAllowed):
		return hostAPIStatusRPCError(403, "Forbidden", map[string]any{extensionStateError: err.Error()})
	case errors.Is(err, resources.ErrConflict), errors.Is(err, resources.ErrSessionNotActive),
		errors.Is(err, resources.ErrStaleSourceVersion):
		return hostAPIStatusRPCError(409, "Conflict", map[string]any{extensionStateError: err.Error()})
	case errors.Is(err, resources.ErrPayloadTooLarge):
		return hostAPIStatusRPCError(413, "Payload too large", map[string]any{extensionStateError: err.Error()})
	case errors.Is(err, resources.ErrNotFound):
		return notFoundRPCError(hostAPIResourceKey, "", err)
	case errors.Is(err, resources.ErrValidation), errors.Is(err, resources.ErrInvalidScopeBinding):
		return invalidParamsRPCError(err)
	default:
		return err
	}
}

func hostAPIStatusRPCError(code int, message string, data any) error {
	return subprocess.NewRPCError(code, strings.TrimSpace(message), data)
}

func isResourceHostAPIMethod(method string) bool {
	switch strings.TrimSpace(method) {
	case string(extensioncontract.HostAPIMethodResourcesList),
		string(extensioncontract.HostAPIMethodResourcesGet),
		string(extensioncontract.HostAPIMethodResourcesSnapshot):
		return true
	default:
		return false
	}
}

func withHostAPIExtensionName(ctx context.Context, extName string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, hostAPIExtensionNameContextKey, strings.TrimSpace(extName))
}

func hostAPIExtensionNameFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(hostAPIExtensionNameContextKey).(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func drainAgentEvents(events <-chan acp.AgentEvent) {
	for range events {
		continue
	}
}

type hostAPIRateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	limit   int
	burst   int
	entries map[string]hostAPIRateState
}

type hostAPIRateState struct {
	tokens    float64
	updatedAt time.Time
}

func newHostAPIRateLimiter(limit int, burst int, now func() time.Time) *hostAPIRateLimiter {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &hostAPIRateLimiter{
		now:     now,
		limit:   limit,
		burst:   burst,
		entries: make(map[string]hostAPIRateState),
	}
}

func (l *hostAPIRateLimiter) Allow(extName string, method string) error {
	if l == nil || l.limit <= 0 || l.burst <= 0 {
		return nil
	}

	key := strings.TrimSpace(extName)
	if key == "" {
		key = hostAPIUnknownExtensionName
	}
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state := l.entries[key]
	if state.updatedAt.IsZero() {
		state.tokens = float64(l.burst)
		state.updatedAt = now
	}

	elapsed := now.Sub(state.updatedAt).Seconds()
	if elapsed > 0 {
		state.tokens = minFloat(float64(l.burst), state.tokens+(elapsed*float64(l.limit)))
		state.updatedAt = now
	}

	if state.tokens >= 1 {
		state.tokens--
		l.entries[key] = state
		return nil
	}

	needed := 1 - state.tokens
	retryAfter := max(time.Duration((needed/float64(l.limit))*float64(time.Second)), time.Millisecond)
	l.entries[key] = state

	return subprocess.NewRPCError(HostAPIRateLimitedCode, "Rate limited", map[string]any{
		hostAPIScopeKey:  "host_api." + strings.TrimSpace(method),
		"retry_after_ms": retryAfter.Milliseconds(),
		hostAPILimitKey:  l.limit,
		"burst":          l.burst,
	})
}

func hostAPIJobUpdateTouchesImmutableConfigFields(req apicontract.UpdateJobRequest) bool {
	return req.Name != nil ||
		req.AgentName != nil ||
		req.WorkspaceID != nil ||
		req.Prompt != nil ||
		req.Schedule != nil ||
		req.Retry != nil ||
		req.FireLimit != nil
}

func hostAPITriggerUpdateTouchesImmutableConfigFields(req apicontract.UpdateTriggerRequest) bool {
	return req.Name != nil ||
		req.AgentName != nil ||
		req.WorkspaceID != nil ||
		req.Prompt != nil ||
		req.Event != nil ||
		req.Filter != nil ||
		req.Retry != nil ||
		req.FireLimit != nil ||
		req.WebhookID != nil ||
		req.EndpointSlug != nil ||
		req.WebhookSecretValue != nil
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func cloneJSONMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(source))
	maps.Copy(cloned, source)
	return cloned
}

func cloneHostAPIStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(source))
	maps.Copy(cloned, source)
	return cloned
}
