package contract

import (
	"time"

	apicontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/memory"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
)

// HostAPIMethod identifies one extension -> AGH Host API request.
type HostAPIMethod = extensionprotocol.HostAPIMethod

const (
	HostAPIMethodSessionsList                 = extensionprotocol.HostAPIMethodSessionsList
	HostAPIMethodSessionsCreate               = extensionprotocol.HostAPIMethodSessionsCreate
	HostAPIMethodSessionsPrompt               = extensionprotocol.HostAPIMethodSessionsPrompt
	HostAPIMethodSessionsStop                 = extensionprotocol.HostAPIMethodSessionsStop
	HostAPIMethodSessionsStatus               = extensionprotocol.HostAPIMethodSessionsStatus
	HostAPIMethodSessionsEvents               = extensionprotocol.HostAPIMethodSessionsEvents
	HostAPIMethodMemoryRecall                 = extensionprotocol.HostAPIMethodMemoryRecall
	HostAPIMethodMemoryStore                  = extensionprotocol.HostAPIMethodMemoryStore
	HostAPIMethodMemoryForget                 = extensionprotocol.HostAPIMethodMemoryForget
	HostAPIMethodObserveHealth                = extensionprotocol.HostAPIMethodObserveHealth
	HostAPIMethodObserveEvents                = extensionprotocol.HostAPIMethodObserveEvents
	HostAPIMethodSkillsList                   = extensionprotocol.HostAPIMethodSkillsList
	HostAPIMethodAutomationJobs               = extensionprotocol.HostAPIMethodAutomationJobs
	HostAPIMethodAutomationJobsGet            = extensionprotocol.HostAPIMethodAutomationJobsGet
	HostAPIMethodAutomationJobsCreate         = extensionprotocol.HostAPIMethodAutomationJobsCreate
	HostAPIMethodAutomationJobsUpdate         = extensionprotocol.HostAPIMethodAutomationJobsUpdate
	HostAPIMethodAutomationJobsDelete         = extensionprotocol.HostAPIMethodAutomationJobsDelete
	HostAPIMethodAutomationJobsTrigger        = extensionprotocol.HostAPIMethodAutomationJobsTrigger
	HostAPIMethodAutomationJobsRuns           = extensionprotocol.HostAPIMethodAutomationJobsRuns
	HostAPIMethodAutomationTriggers           = extensionprotocol.HostAPIMethodAutomationTriggers
	HostAPIMethodAutomationTriggersGet        = extensionprotocol.HostAPIMethodAutomationTriggersGet
	HostAPIMethodAutomationTriggersCreate     = extensionprotocol.HostAPIMethodAutomationTriggersCreate
	HostAPIMethodAutomationTriggersUpdate     = extensionprotocol.HostAPIMethodAutomationTriggersUpdate
	HostAPIMethodAutomationTriggersDelete     = extensionprotocol.HostAPIMethodAutomationTriggersDelete
	HostAPIMethodAutomationTriggersRuns       = extensionprotocol.HostAPIMethodAutomationTriggersRuns
	HostAPIMethodAutomationTriggersFire       = extensionprotocol.HostAPIMethodAutomationTriggersFire
	HostAPIMethodAutomationRuns               = extensionprotocol.HostAPIMethodAutomationRuns
	HostAPIMethodChannelsMessagesIngest       = extensionprotocol.HostAPIMethodChannelsMessagesIngest
	HostAPIMethodChannelsInstancesGet         = extensionprotocol.HostAPIMethodChannelsInstancesGet
	HostAPIMethodChannelsInstancesReportState = extensionprotocol.HostAPIMethodChannelsInstancesReportState
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
	Since     time.Time `json:"since,omitempty"`
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
	Since     time.Time `json:"since,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// SkillsListParams filters skills by workspace scope.
type SkillsListParams struct {
	Workspace string `json:"workspace,omitempty"`
}

// AutomationJobsParams filters visible automation jobs.
type AutomationJobsParams struct {
	Scope       automationpkg.AutomationScope `json:"scope,omitempty"`
	WorkspaceID string                        `json:"workspace_id,omitempty"`
	Enabled     *bool                         `json:"enabled,omitempty"`
}

// AutomationTriggersParams filters visible automation triggers.
type AutomationTriggersParams struct {
	Scope       automationpkg.AutomationScope `json:"scope,omitempty"`
	WorkspaceID string                        `json:"workspace_id,omitempty"`
	Event       string                        `json:"event,omitempty"`
	Enabled     *bool                         `json:"enabled,omitempty"`
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
	Event       string                        `json:"event"`
	Scope       automationpkg.AutomationScope `json:"scope"`
	WorkspaceID string                        `json:"workspace_id,omitempty"`
	Payload     map[string]any                `json:"payload,omitempty"`
}

// ChannelsMessagesIngestParams carries one normalized inbound channel message.
type ChannelsMessagesIngestParams = channelspkg.InboundMessageEnvelope

// ChannelsInstancesReportStateParams reports one adapter-observed instance status update.
type ChannelsInstancesReportStateParams struct {
	Status channelspkg.ChannelStatus `json:"status"`
}

// SessionSummary is the lightweight host-visible session listing shape.
type SessionSummary struct {
	ID        string               `json:"id"`
	Name      string               `json:"name,omitempty"`
	Agent     string               `json:"agent"`
	Workspace string               `json:"workspace,omitempty"`
	State     session.SessionState `json:"state"`
	CreatedAt time.Time            `json:"created_at"`
}

// SessionStatus is the detailed host-visible session status shape.
type SessionStatus struct {
	SessionID    string               `json:"session_id"`
	Name         string               `json:"name,omitempty"`
	Agent        string               `json:"agent"`
	WorkspaceID  string               `json:"workspace_id,omitempty"`
	Workspace    string               `json:"workspace,omitempty"`
	State        session.SessionState `json:"state"`
	StopReason   store.StopReason     `json:"stop_reason,omitempty"`
	StopDetail   string               `json:"stop_detail,omitempty"`
	ACPSessionID string               `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
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
}

// SessionPromptResult returns the created turn identifier.
type SessionPromptResult struct {
	TurnID string `json:"turn_id"`
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

// ChannelsMessagesIngestResult reports the resolved session association for one inbound message.
type ChannelsMessagesIngestResult struct {
	SessionID    string                 `json:"session_id"`
	RouteCreated bool                   `json:"route_created"`
	RoutingKey   channelspkg.RoutingKey `json:"routing_key"`
}

// HostAPIMethodSpecs returns the canonical Host API method registry in wire order.
func HostAPIMethodSpecs() []HostAPIMethodSpec {
	return []HostAPIMethodSpec{
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
			Result:         NamedType{Name: "Trigger", Value: []automationpkg.Trigger{}},
			OptionalParams: true,
		},
		{
			Method: HostAPIMethodAutomationTriggersGet,
			Params: NamedType{Name: "AutomationTargetParams", Value: AutomationTargetParams{}},
			Result: NamedType{Name: "Trigger", Value: automationpkg.Trigger{}},
		},
		{
			Method: HostAPIMethodAutomationTriggersCreate,
			Params: NamedType{Name: "AutomationTriggerCreateParams", Value: AutomationTriggerCreateParams{}},
			Result: NamedType{Name: "Trigger", Value: automationpkg.Trigger{}},
		},
		{
			Method: HostAPIMethodAutomationTriggersUpdate,
			Params: NamedType{Name: "AutomationTriggerUpdateParams", Value: AutomationTriggerUpdateParams{}},
			Result: NamedType{Name: "Trigger", Value: automationpkg.Trigger{}},
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
			Method: HostAPIMethodChannelsMessagesIngest,
			Params: NamedType{Name: "InboundMessageEnvelope", Value: channelspkg.InboundMessageEnvelope{}},
			Result: NamedType{Name: "ChannelsMessagesIngestResult", Value: ChannelsMessagesIngestResult{}},
		},
		{
			Method:         HostAPIMethodChannelsInstancesGet,
			Params:         NamedType{Name: "EmptyResult", Value: EmptyResult{}},
			Result:         NamedType{Name: "ChannelInstance", Value: channelspkg.ChannelInstance{}},
			OptionalParams: true,
		},
		{
			Method: HostAPIMethodChannelsInstancesReportState,
			Params: NamedType{Name: "ChannelsInstancesReportStateParams", Value: ChannelsInstancesReportStateParams{}},
			Result: NamedType{Name: "ChannelInstance", Value: channelspkg.ChannelInstance{}},
		},
	}
}
