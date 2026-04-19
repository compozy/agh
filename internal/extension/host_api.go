package extensionpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/pedronauck/agh/internal/acp"
	apicontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/frontmatter"
	"github.com/pedronauck/agh/internal/memory"
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
	hostAPIEnvironmentStateSynced       = "synced"
	hostAPIEnvironmentStatePending      = "pending"
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
	memory           *memory.Store
	observer         hostAPIObserver
	skills           hostAPISkillsRegistry
	workspaces       workspacepkg.RuntimeResolver
	bridges          hostAPIBridgeRegistry
	dedupStore       hostAPIBridgeDedupStore
	deliveryBroker   hostAPIDeliveryBroker
	resourceStore    resources.RawStore
	resourceCodecs   *resources.CodecRegistry
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
	ExecEnvironment(ctx context.Context, req session.EnvironmentExecRequest) (session.EnvironmentExecResult, error)
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
	ListTriggers(ctx context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	GetTrigger(ctx context.Context, id string) (automationpkg.Trigger, error)
	CreateTrigger(
		ctx context.Context,
		trigger automationpkg.Trigger,
		webhookSecret string,
	) (automationpkg.Trigger, error)
	UpdateTrigger(
		ctx context.Context,
		trigger automationpkg.Trigger,
		webhookSecret *string,
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

// WithHostAPIBridgeIngressConfig overrides dedup TTL and cleanup cadence for bridge ingest.
func WithHostAPIBridgeIngressConfig(dedupTTL time.Duration, cleanupInterval time.Duration) HostAPIOption {
	return func(handler *HostAPIHandler) {
		handler.bridgeIngestDedupTTL = dedupTTL
		handler.bridgeCleanupInterval = cleanupInterval
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
	return map[string]hostAPIMethodFunc{
		"automation/jobs":                                             handler.handleAutomationJobs,
		"automation/jobs/get":                                         handler.handleAutomationJobsGet,
		"automation/jobs/create":                                      handler.handleAutomationJobsCreate,
		"automation/jobs/update":                                      handler.handleAutomationJobsUpdate,
		"automation/jobs/delete":                                      handler.handleAutomationJobsDelete,
		"automation/jobs/trigger":                                     handler.handleAutomationJobsTrigger,
		"automation/jobs/runs":                                        handler.handleAutomationJobsRuns,
		"automation/triggers":                                         handler.handleAutomationTriggers,
		"automation/triggers/get":                                     handler.handleAutomationTriggersGet,
		"automation/triggers/create":                                  handler.handleAutomationTriggersCreate,
		"automation/triggers/update":                                  handler.handleAutomationTriggersUpdate,
		"automation/triggers/delete":                                  handler.handleAutomationTriggersDelete,
		"automation/triggers/runs":                                    handler.handleAutomationTriggersRuns,
		"automation/triggers/fire":                                    handler.handleAutomationTriggersFire,
		"automation/runs":                                             handler.handleAutomationRuns,
		string(extensioncontract.HostAPIMethodTasks):                  handler.handleTasks,
		string(extensioncontract.HostAPIMethodTasksGet):               handler.handleTasksGet,
		string(extensioncontract.HostAPIMethodTasksTimeline):          handler.handleTasksTimeline,
		string(extensioncontract.HostAPIMethodTasksTree):              handler.handleTasksTree,
		string(extensioncontract.HostAPIMethodTasksDashboard):         handler.handleTasksDashboard,
		string(extensioncontract.HostAPIMethodTasksInbox):             handler.handleTasksInbox,
		string(extensioncontract.HostAPIMethodTasksCreate):            handler.handleTasksCreate,
		string(extensioncontract.HostAPIMethodTasksUpdate):            handler.handleTasksUpdate,
		string(extensioncontract.HostAPIMethodTasksCancel):            handler.handleTasksCancel,
		string(extensioncontract.HostAPIMethodTasksRuns):              handler.handleTasksRuns,
		string(extensioncontract.HostAPIMethodTasksRunsGet):           handler.handleTasksRunsGet,
		string(extensioncontract.HostAPIMethodTasksRunsEnqueue):       handler.handleTasksRunsEnqueue,
		string(extensioncontract.HostAPIMethodTasksRunsClaim):         handler.handleTasksRunsClaim,
		string(extensioncontract.HostAPIMethodTasksRunsStart):         handler.handleTasksRunsStart,
		string(extensioncontract.HostAPIMethodTasksRunsAttachSession): handler.handleTasksRunsAttachSession,
		string(extensioncontract.HostAPIMethodTasksRunsComplete):      handler.handleTasksRunsComplete,
		string(extensioncontract.HostAPIMethodTasksRunsFail):          handler.handleTasksRunsFail,
		string(extensioncontract.HostAPIMethodTasksRunsCancel):        handler.handleTasksRunsCancel,
		"resources/list":                                              handler.handleResourcesList,
		"resources/get":                                               handler.handleResourcesGet,
		"resources/snapshot":                                          handler.handleResourcesSnapshot,
		"bridges/instances/list":                                      handler.handleBridgesInstancesList,
		"bridges/instances/get":                                       handler.handleBridgesInstancesGet,
		"bridges/instances/report_state":                              handler.handleBridgesInstancesReportState,
		"bridges/messages/ingest":                                     handler.handleBridgesMessagesIngest,
		"environment/exec":                                            handler.handleEnvironmentExec,
		"environment/info":                                            handler.handleEnvironmentInfo,
		"environment/list":                                            handler.handleEnvironmentList,
		"memory/forget":                                               handler.handleMemoryForget,
		"memory/recall":                                               handler.handleMemoryRecall,
		"memory/store":                                                handler.handleMemoryStore,
		"observe/events":                                              handler.handleObserveEvents,
		"observe/health":                                              handler.handleObserveHealth,
		"sessions/create":                                             handler.handleSessionsCreate,
		"sessions/events":                                             handler.handleSessionsEvents,
		"sessions/list":                                               handler.handleSessionsList,
		"sessions/prompt":                                             handler.handleSessionsPrompt,
		"sessions/status":                                             handler.handleSessionsStatus,
		"sessions/stop":                                               handler.handleSessionsStop,
		"skills/list":                                                 handler.handleSkillsList,
	}
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

type hostAPIEnvironmentListParams = extensioncontract.EnvironmentListParams

type hostAPIEnvironmentInfoParams = extensioncontract.EnvironmentInfoParams

type hostAPIEnvironmentExecParams = extensioncontract.EnvironmentExecParams

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

type hostAPIEnvironmentListResult = extensioncontract.EnvironmentListResult

type hostAPIEnvironmentSummary = extensioncontract.EnvironmentSummary

type hostAPIEnvironmentInfoResult = extensioncontract.EnvironmentInfoResult

type hostAPIEnvironmentExecResult = extensioncontract.EnvironmentExecResult

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
			filterWorkspaceID = strings.TrimSpace(resolved.ID)
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
		AgentName: strings.TrimSpace(params.Agent),
		Workspace: strings.TrimSpace(params.Workspace),
		Type:      session.SessionTypeSystem,
	})
	if err != nil {
		return nil, err
	}

	if prompt := strings.TrimSpace(params.Prompt); prompt != "" {
		if _, err := h.submitPrompt(ctx, sess.ID, prompt); err != nil {
			return nil, err
		}
	}

	return hostAPISessionCreateResult{SessionID: sess.ID}, nil
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

	info, err := h.sessions.Status(ctx, params.SessionID)
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

func (h *HostAPIHandler) handleEnvironmentList(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}

	var params hostAPIEnvironmentListParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}

	filterWorkspaceID, filterWorkspaceRoot, err := h.resolveEnvironmentWorkspaceFilter(ctx, params.Workspace)
	if err != nil {
		return nil, err
	}

	infos, err := h.sessions.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	result := hostAPIEnvironmentListResult{
		Environments: make([]hostAPIEnvironmentSummary, 0, len(infos)),
	}
	for _, info := range infos {
		if info == nil || info.Environment == nil || info.State == session.StateStopped {
			continue
		}
		if filterWorkspaceID != "" || filterWorkspaceRoot != "" {
			if info.WorkspaceID != filterWorkspaceID && info.Workspace != filterWorkspaceRoot {
				continue
			}
		}
		result.Environments = append(result.Environments, hostAPIEnvironmentSummary{
			SessionID:     info.ID,
			EnvironmentID: strings.TrimSpace(info.Environment.EnvironmentID),
			Backend:       strings.TrimSpace(info.Environment.Backend),
			Profile:       strings.TrimSpace(info.Environment.Profile),
			InstanceID:    strings.TrimSpace(info.Environment.InstanceID),
			State:         strings.TrimSpace(info.Environment.State),
			SyncState:     hostAPIEnvironmentSyncState(info.Environment),
		})
	}
	return result, nil
}

func (h *HostAPIHandler) handleEnvironmentInfo(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	var params hostAPIEnvironmentInfoParams
	if err := decodeHostAPIParams(raw, &params); err != nil {
		return nil, err
	}
	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, invalidParamsRPCError(errors.New("session_id is required"))
	}
	info, err := h.sessions.Status(ctx, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, notFoundRPCError("session", sessionID, err)
		}
		return nil, err
	}
	if info == nil || info.Environment == nil {
		return nil, notFoundRPCError("environment", sessionID, errors.New("environment is not configured"))
	}
	return hostAPIEnvironmentInfoResult{
		EnvironmentID: strings.TrimSpace(info.Environment.EnvironmentID),
		Backend:       strings.TrimSpace(info.Environment.Backend),
		Profile:       strings.TrimSpace(info.Environment.Profile),
		InstanceID:    strings.TrimSpace(info.Environment.InstanceID),
		RuntimeRoot:   strings.TrimSpace(info.Environment.RuntimeRootDir),
		SyncState:     hostAPIEnvironmentSyncState(info.Environment),
		CreatedAt:     info.CreatedAt,
		LastSyncError: strings.TrimSpace(info.Environment.LastSyncError),
	}, nil
}

func (h *HostAPIHandler) handleEnvironmentExec(ctx context.Context, raw json.RawMessage) (any, error) {
	if h.sessions == nil {
		return nil, errors.New("extension: session manager is not configured")
	}
	var params hostAPIEnvironmentExecParams
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
	result, err := h.sessions.ExecEnvironment(ctx, session.EnvironmentExecRequest{
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
	return hostAPIEnvironmentExecResult{
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
	}, nil
}

func (h *HostAPIHandler) resolveEnvironmentWorkspaceFilter(
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
	return strings.TrimSpace(resolved.ID), strings.TrimSpace(resolved.RootDir), nil
}

func hostAPIEnvironmentSyncState(meta *store.SessionEnvironmentMeta) string {
	if meta == nil {
		return ""
	}
	if strings.TrimSpace(meta.LastSyncError) != "" {
		return extensionStateError
	}
	if meta.LastSyncAt != nil {
		return hostAPIEnvironmentStateSynced
	}
	return hostAPIEnvironmentStatePending
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
		Key:       filename,
		Scope:     scope,
		Content:   params.Content,
		Tags:      params.Tags,
		AgentName: hostAPIExtensionNameFromContext(ctx),
	})
	if err != nil {
		return nil, err
	}
	if err := storeHandle.Write(scope, filename, []byte(doc)); err != nil {
		return nil, err
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

	sources, err := h.memorySourcesForRecall(ctx, string(params.Scope), params.Workspace)
	if err != nil {
		return nil, err
	}

	results := make([]hostAPIMemoryRecallEntry, 0)
	for _, source := range sources {
		headers, scanErr := source.store.Scan(source.scope)
		if scanErr != nil {
			return nil, scanErr
		}
		for _, header := range headers {
			content, readErr := source.store.Read(source.scope, header.Filename)
			if readErr != nil {
				return nil, readErr
			}
			body, tags := extractMemoryBodyAndTags(content)
			score := scoreMemoryRecall(query, header, body, tags)
			if score <= 0 {
				continue
			}
			results = append(results, hostAPIMemoryRecallEntry{
				Key:     header.Filename,
				Content: body,
				Score:   score,
			})
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Key < results[j].Key
		}
		return results[i].Score > results[j].Score
	})

	limit := params.Limit
	if limit <= 0 {
		limit = defaultHostAPIRecallLimit
	}
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
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
	if err := storeHandle.Delete(scope, normalizeMemoryFilename(params.Key)); err != nil {
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

	events, err := h.observer.QueryEvents(ctx, store.EventSummaryQuery{
		SessionID: strings.TrimSpace(params.SessionID),
		AgentName: strings.TrimSpace(params.AgentName),
		Type:      strings.TrimSpace(params.Type),
		Since:     params.Since,
		Limit:     params.Limit,
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
				"session_id": event.SessionID,
				"agent_name": event.AgentName,
				"summary":    event.Summary,
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
		skills, err = h.skills.ForWorkspace(ctx, &resolved)
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

	return automation.TriggerJob(ctx, params.ID)
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
		return triggers, nil
	}

	filtered := make([]automationpkg.Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		if trigger.Enabled == *params.Enabled {
			filtered = append(filtered, trigger)
		}
	}
	return filtered, nil
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

	return automation.GetTrigger(ctx, params.ID)
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
	if err := trigger.Validate("trigger"); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	return automation.CreateTrigger(ctx, trigger, webhookSecret)
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
		return automation.SetTriggerEnabled(ctx, current.ID, *params.Enabled)
	}

	next, webhookSecret, err := h.applyTriggerUpdateParams(ctx, current, params.UpdateTriggerRequest)
	if err != nil {
		return nil, err
	}
	if err := next.Validate("trigger"); err != nil {
		return nil, invalidParamsRPCError(err)
	}

	return automation.UpdateTrigger(ctx, next, webhookSecret)
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
	go drainAgentEvents(eventsCh)

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

type hostAPIMemorySource struct {
	store *memory.Store
	scope memory.Scope
}

func (h *HostAPIHandler) memorySourcesForRecall(
	ctx context.Context,
	rawScope string,
	rawWorkspace string,
) ([]hostAPIMemorySource, error) {
	if h.memory == nil {
		return nil, errors.New("extension: memory store is not configured")
	}

	scope := memory.Scope(strings.TrimSpace(rawScope)).Normalize()
	switch scope {
	case "":
		sources := []hostAPIMemorySource{{store: h.memory, scope: memory.ScopeGlobal}}
		workspaceRoot, err := h.resolveWorkspaceRoot(ctx, rawWorkspace)
		if err != nil {
			return nil, err
		}
		if workspaceRoot != "" {
			sources = append(sources, hostAPIMemorySource{
				store: h.memory.ForWorkspace(workspaceRoot),
				scope: memory.ScopeWorkspace,
			})
		}
		return sources, nil
	case memory.ScopeGlobal:
		return []hostAPIMemorySource{{store: h.memory, scope: memory.ScopeGlobal}}, nil
	case memory.ScopeWorkspace:
		storeHandle, _, err := h.memoryStoreFor(ctx, rawScope, rawWorkspace)
		if err != nil {
			return nil, err
		}
		return []hostAPIMemorySource{{store: storeHandle, scope: memory.ScopeWorkspace}}, nil
	default:
		return nil, invalidParamsRPCError(fmt.Errorf("memory scope must be one of global or workspace"))
	}
}

func (h *HostAPIHandler) memoryStoreFor(
	ctx context.Context,
	rawScope string,
	rawWorkspace string,
) (*memory.Store, memory.Scope, error) {
	if h.memory == nil {
		return nil, "", errors.New("extension: memory store is not configured")
	}

	scope := memory.Scope(strings.TrimSpace(rawScope)).Normalize()
	workspaceRoot, err := h.resolveWorkspaceRoot(ctx, rawWorkspace)
	if err != nil {
		return nil, "", err
	}
	if scope == "" {
		if workspaceRoot != "" {
			scope = memory.ScopeWorkspace
		} else {
			scope = memory.ScopeGlobal
		}
	}

	switch scope {
	case memory.ScopeGlobal:
		return h.memory, memory.ScopeGlobal, nil
	case memory.ScopeWorkspace:
		if workspaceRoot == "" {
			return nil, "", invalidParamsRPCError(errors.New("workspace is required for workspace memory scope"))
		}
		return h.memory.ForWorkspace(workspaceRoot), memory.ScopeWorkspace, nil
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
	return strings.TrimSpace(resolved.ID), nil
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
) (automationpkg.Trigger, string, error) {
	workspaceID, err := h.resolveAutomationWorkspaceID(ctx, req.WorkspaceID)
	if err != nil {
		return automationpkg.Trigger{}, "", err
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
	return trigger, strings.TrimSpace(req.WebhookSecret), nil
}

func (h *HostAPIHandler) applyTriggerUpdateParams(
	ctx context.Context,
	current automationpkg.Trigger,
	req apicontract.UpdateTriggerRequest,
) (automationpkg.Trigger, *string, error) {
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

	var webhookSecret *string
	if req.WebhookSecret != nil {
		secret := strings.TrimSpace(*req.WebhookSecret)
		webhookSecret = &secret
	}
	return next, webhookSecret, nil
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
	Key       string
	Scope     memory.Scope
	Content   string
	Tags      []string
	AgentName string
}

func renderMemoryDocument(doc hostAPIMemoryDocument) (string, error) {
	header := memory.Header{
		Name:        memoryNameFromFilename(doc.Key),
		Description: memoryDescriptionFromContent(doc.Content),
		Type:        memoryTypeForScope(doc.Scope, doc.Tags),
		AgentName:   strings.TrimSpace(doc.AgentName),
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

func memoryTypeForScope(scope memory.Scope, tags []string) memory.Type {
	for _, tag := range normalizeUniqueStrings(tags) {
		switch memory.Type(tag).Normalize() {
		case memory.MemoryTypeUser, memory.MemoryTypeFeedback, memory.MemoryTypeProject, memory.MemoryTypeReference:
			return memory.Type(tag).Normalize()
		}
	}
	if scope == memory.ScopeWorkspace {
		return memory.MemoryTypeProject
	}
	return memory.MemoryTypeUser
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

func extractMemoryBodyAndTags(content []byte) (string, []string) {
	body := strings.TrimSpace(string(content))
	parts, err := frontmatter.Split(content)
	if err == nil {
		body = strings.TrimSpace(parts.Body)
	}
	if !strings.HasPrefix(body, tagCommentPrefix) {
		return body, nil
	}

	lineEnd := strings.IndexByte(body, '\n')
	if lineEnd < 0 {
		lineEnd = len(body)
	}
	comment := strings.TrimSpace(body[:lineEnd])
	body = strings.TrimSpace(strings.TrimPrefix(body[lineEnd:], "\n"))

	comment = strings.TrimPrefix(comment, tagCommentPrefix)
	comment = strings.TrimSuffix(comment, "-->")
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return body, nil
	}
	return body, normalizeUniqueStrings(strings.Split(comment, ","))
}

func scoreMemoryRecall(query string, header memory.Header, body string, tags []string) float64 {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if normalizedQuery == "" {
		return 0
	}

	haystack := strings.ToLower(strings.Join([]string{
		header.Filename,
		header.Name,
		header.Description,
		header.AgentName,
		strings.Join(tags, " "),
		body,
	}, " "))

	score := 0.0
	if strings.Contains(haystack, normalizedQuery) {
		score += 4
	}

	for token := range strings.FieldsSeq(normalizedQuery) {
		if strings.Contains(haystack, token) {
			score++
		}
	}

	return score
}

func hostAPISessionStatusFromInfo(info *session.Info) hostAPISessionStatus {
	if info == nil {
		return hostAPISessionStatus{}
	}
	return hostAPISessionStatus{
		SessionID:    info.ID,
		Name:         info.Name,
		Agent:        info.AgentName,
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
	return subprocess.NewRPCError(HostAPIInvalidParamsCode, "Invalid params", map[string]string{"error": err.Error()})
}

func unavailableRPCError(err error) error {
	if err == nil {
		return subprocess.NewRPCError(HostAPIUnavailableCode, "Unavailable", nil)
	}
	return subprocess.NewRPCError(HostAPIUnavailableCode, "Unavailable", map[string]string{"error": err.Error()})
}

func notFoundRPCError(resource string, id string, err error) error {
	data := map[string]string{
		"resource": strings.TrimSpace(resource),
		"id":       strings.TrimSpace(id),
	}
	if err != nil {
		data["error"] = err.Error()
	}
	return subprocess.NewRPCError(HostAPINotFoundCode, "Not found", data)
}

func methodNotFoundRPCError(method string) error {
	return subprocess.NewRPCError(
		HostAPIMethodNotFoundCode,
		"Method not found",
		map[string]string{"method": strings.TrimSpace(method)},
	)
}

func rpcCapabilityDenied(err error) error {
	var denied *ErrCapabilityDenied
	if !errors.As(err, &denied) {
		return err
	}
	if isResourceHostAPIMethod(denied.Data.Method) {
		return hostAPIStatusRPCError(403, "Forbidden", map[string]any{
			"error":    denied.Error(),
			"method":   strings.TrimSpace(denied.Data.Method),
			"required": append([]string(nil), denied.Data.Required...),
			"granted":  append([]string(nil), denied.Data.Granted...),
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

	var rpcErr *subprocess.RPCError
	if errors.As(err, &rpcErr) {
		if rpcErr.Code == HostAPIRateLimitedCode {
			return hostAPIStatusRPCError(429, "Rate limited", rpcErr.Data)
		}
		return err
	}

	switch {
	case errors.Is(err, resources.ErrPermissionDenied), errors.Is(err, resources.ErrDirectMutationNotAllowed):
		return hostAPIStatusRPCError(403, "Forbidden", map[string]any{"error": err.Error()})
	case errors.Is(err, resources.ErrConflict), errors.Is(err, resources.ErrSessionNotActive),
		errors.Is(err, resources.ErrStaleSourceVersion):
		return hostAPIStatusRPCError(409, "Conflict", map[string]any{"error": err.Error()})
	case errors.Is(err, resources.ErrPayloadTooLarge):
		return hostAPIStatusRPCError(413, "Payload too large", map[string]any{"error": err.Error()})
	case errors.Is(err, resources.ErrNotFound):
		return notFoundRPCError("resource", "", err)
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
	for event := range events {
		_ = event
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
		"scope":          "host_api." + strings.TrimSpace(method),
		"retry_after_ms": retryAfter.Milliseconds(),
		"limit":          l.limit,
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
		req.WebhookSecret != nil
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
