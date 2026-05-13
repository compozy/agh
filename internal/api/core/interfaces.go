// Package core provides the shared transport-facing API layer used by HTTP and UDS bindings.
package core

import (
	"context"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/modelcatalog"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/soul"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	toolspkg "github.com/pedronauck/agh/internal/tools"
	"github.com/pedronauck/agh/internal/transcript"
	"github.com/pedronauck/agh/internal/vault"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// AgentLoader loads one parsed AGENT.md definition.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// AgentCatalog exposes projected resource-backed agent definitions.
type AgentCatalog interface {
	ListAgents(ctx context.Context) ([]aghconfig.AgentDef, error)
	GetAgent(ctx context.Context, name string) (aghconfig.AgentDef, error)
}

// ModelCatalogService exposes daemon-owned provider model catalog reads and refreshes.
type ModelCatalogService interface {
	modelcatalog.Service
}

// SessionManager is the runtime session surface exposed by API transports.
// List returns the current in-memory session snapshot without performing I/O.
// ListAll may perform I/O to return the authoritative session set, so it accepts a context.
type SessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	List() []*session.Info
	ListAll(ctx context.Context) ([]*session.Info, error)
	Status(ctx context.Context, id string) (*session.Info, error)
	Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error)
	History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error)
	Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error)
	RepairSession(ctx context.Context, opts session.RepairOpts) (*session.RepairResult, error)
	Delete(ctx context.Context, id string) error
	Stop(ctx context.Context, id string) error
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
	Resume(ctx context.Context, id string) (*session.Session, error)
	ClearConversation(ctx context.Context, id string) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
	CancelPrompt(ctx context.Context, id string) error
	ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error
}

// Observer is the observability surface exposed by API transports.
type Observer interface {
	QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error)
	QueryHookCatalog(ctx context.Context, filter hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error)
	QueryHookRuns(ctx context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error)
	QueryHookEvents(ctx context.Context, filter hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error)
	QueryBridgeHealth(ctx context.Context) ([]observe.BridgeInstanceHealth, error)
	Health(ctx context.Context) (observe.Health, error)
	QueryTaskDashboard(ctx context.Context, query observe.TaskDashboardQuery) (observe.TaskDashboardView, error)
	QueryTaskInbox(
		ctx context.Context,
		query observe.TaskInboxQuery,
		actor taskpkg.ActorIdentity,
	) (observe.TaskInboxView, error)
}

// BridgeService is the daemon-owned bridge runtime surface exposed by API transports.
type BridgeService interface {
	bridgepkg.Registry
	bridgepkg.BridgeTaskSubscriptionStore
	bridgepkg.TargetResolver
	ListProviders(ctx context.Context) ([]bridgepkg.BridgeProvider, error)
	ListSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error)
	PutSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding, secretValue *string) error
	DeleteSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error
	StartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error)
	StopInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error)
	RestartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error)
}

// BundleService exposes extension bundle catalog, activation, and effective
// network-default state to API transports.
type BundleService interface {
	Catalog(ctx context.Context) ([]bundlepkg.CatalogEntry, error)
	PreviewActivation(ctx context.Context, req bundlepkg.ActivateRequest) (bundlepkg.ActivationPreview, error)
	Activate(ctx context.Context, req bundlepkg.ActivateRequest) (bundlepkg.ActivationPreview, error)
	ListActivations(ctx context.Context) ([]bundlepkg.ActivationPreview, error)
	GetActivation(ctx context.Context, id string) (bundlepkg.ActivationPreview, error)
	UpdateActivation(ctx context.Context, req bundlepkg.UpdateActivationRequest) (bundlepkg.ActivationPreview, error)
	Deactivate(ctx context.Context, id string) error
	NetworkSettings(ctx context.Context) (bundlepkg.NetworkSettings, error)
}

// NetworkService is the runtime network surface exposed to daemon transports.
type NetworkService interface {
	Send(ctx context.Context, req network.SendRequest) (string, error)
	ListPeers(ctx context.Context, workspaceID string, channel string) ([]network.PeerInfo, error)
	ListChannels(ctx context.Context, workspaceID string) ([]network.ChannelInfo, error)
	Status(ctx context.Context) (*network.Status, error)
	Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error)
	WaitInbox(ctx context.Context, sessionID string, channel string) ([]network.Envelope, error)
}

// AgentContextService assembles the bounded situation payload for a validated agent session.
type AgentContextService interface {
	ContextForSession(ctx context.Context, info *session.Info) (contract.AgentContextPayload, error)
}

// SoulAuthoringService exposes managed SOUL.md authoring and read validation to API handlers.
type SoulAuthoringService interface {
	soul.AuthoringService
}

// SoulRefresher refreshes a session's resolved Soul snapshot through service-owned CAS.
type SoulRefresher interface {
	RefreshSoulWithExpectedDigest(
		ctx context.Context,
		id string,
		expectedDigest string,
	) (session.SoulRefreshResult, error)
}

// HeartbeatAuthoringService exposes managed HEARTBEAT.md authoring to API handlers.
type HeartbeatAuthoringService interface {
	heartbeat.AuthoringService
}

// HeartbeatStatusService composes read-only Heartbeat policy, wake state, and health.
type HeartbeatStatusService interface {
	heartbeat.StatusService
}

// HeartbeatWakeService evaluates one advisory manual Heartbeat wake.
type HeartbeatWakeService interface {
	Wake(ctx context.Context, req heartbeat.WakeRequest) (heartbeat.WakeDecision, error)
}

// SessionHealthReader reads metadata-only session health rows.
type SessionHealthReader interface {
	GetSessionHealth(ctx context.Context, sessionID string) (heartbeat.SessionHealth, error)
}

// HeartbeatWakeEventReader lists retained Heartbeat wake audit rows.
type HeartbeatWakeEventReader interface {
	ListHeartbeatWakeEvents(ctx context.Context, query heartbeat.WakeEventListQuery) ([]heartbeat.WakeEvent, error)
}

// CoordinatorConfigResolver resolves safe coordinator policy for agent-facing reads.
type CoordinatorConfigResolver interface {
	ResolveCoordinatorConfig(ctx context.Context, workspaceID string) (aghconfig.CoordinatorConfig, error)
}

// NetworkStore exposes persisted network audit, channel metadata CRUD, and timeline queries to the API layer.
type NetworkStore interface {
	store.NetworkConversationStore
	ListNetworkAudit(ctx context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error)
	GetNetworkChannel(ctx context.Context, ref store.NetworkChannelRef) (store.NetworkChannelEntry, error)
	ListNetworkChannels(ctx context.Context, query store.NetworkChannelQuery) ([]store.NetworkChannelEntry, error)
	WriteNetworkChannel(ctx context.Context, entry store.NetworkChannelEntry) error
	DeleteNetworkChannel(ctx context.Context, ref store.NetworkChannelRef) error
	ListNetworkMessages(ctx context.Context, query store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error)
}

// DreamTrigger exposes consolidation controls and state to the API layer.
type DreamTrigger interface {
	Trigger(ctx context.Context, workspace string) (bool, string, error)
	LastConsolidatedAt() (time.Time, error)
	Enabled() bool
}

// MemoryExtractorService exposes the daemon-owned Memory v2 extractor runtime.
type MemoryExtractorService interface {
	Status(ctx context.Context) (contract.MemoryExtractorStatusPayload, error)
	ListFailures(ctx context.Context) ([]contract.MemoryExtractorFailurePayload, error)
	Retry(ctx context.Context, req contract.MemoryExtractorRetryRequest) (contract.MemoryExtractorRetryResponse, error)
	Drain(ctx context.Context) (contract.MemoryExtractorDrainResponse, error)
}

// MemoryProviderService exposes the active MemoryProvider registry.
type MemoryProviderService interface {
	List(ctx context.Context, workspaceID string) ([]contract.MemoryProviderPayload, error)
	Get(ctx context.Context, workspaceID string, name string) (contract.MemoryProviderPayload, error)
	Select(ctx context.Context, workspaceID string, name string) (contract.MemoryProviderPayload, error)
	Enable(
		ctx context.Context,
		workspaceID string,
		name string,
		reason string,
	) (contract.MemoryProviderLifecycleResponse, error)
	Disable(
		ctx context.Context,
		workspaceID string,
		name string,
		reason string,
	) (contract.MemoryProviderLifecycleResponse, error)
}

// MemorySessionLedgerService exposes materialized session ledgers and replay.
type MemorySessionLedgerService interface {
	Get(ctx context.Context, sessionID string) (contract.MemorySessionLedgerResponse, error)
	Replay(
		ctx context.Context,
		sessionID string,
		req contract.MemorySessionReplayRequest,
	) (contract.MemorySessionReplayResponse, error)
	Prune(ctx context.Context, req contract.MemorySessionsPruneRequest) (contract.MemorySessionsPruneResponse, error)
	Repair(ctx context.Context) (contract.MemorySessionsRepairResponse, error)
}

// SettingsService exposes the daemon-owned settings read and mutation surface to API transports.
type SettingsService interface {
	GetSection(ctx context.Context, req settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error)
	UpdateSection(ctx context.Context, req settingspkg.SectionUpdateRequest) (settingspkg.MutationResult, error)
	ListCollection(ctx context.Context, req settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error)
	PutCollectionItem(ctx context.Context, req settingspkg.CollectionItemPutRequest) (settingspkg.MutationResult, error)
	DeleteCollectionItem(
		ctx context.Context,
		req settingspkg.CollectionItemDeleteRequest,
	) (
		settingspkg.MutationResult,
		error,
	)
}

// SkillsRegistry exposes the daemon-owned skill catalog.
type SkillsRegistry interface {
	Get(name string) (*skills.Skill, bool)
	List() []*skills.Skill
	ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	ForAgent(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace, agentName string) ([]*skills.Skill, error)
	LoadContent(ctx context.Context, skill *skills.Skill) (string, error)
	LoadResource(ctx context.Context, skill *skills.Skill, relativePath string) (string, error)
	SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
	SetEnabledForAgent(name string, resolved *workspacepkg.ResolvedWorkspace, agentName string, enabled bool) error
}

// VaultService exposes redacted secret metadata and write-only mutations to API transports.
type VaultService interface {
	GetMetadata(ctx context.Context, ref string) (vault.Metadata, error)
	ListMetadata(ctx context.Context, prefix string) ([]vault.Metadata, error)
	PutSecret(ctx context.Context, ref string, kind string, plaintext string) (vault.Metadata, error)
	DeleteSecret(ctx context.Context, ref string) error
}

// SettingsRestartOperation is the daemon-owned restart record exposed to settings transports.
type SettingsRestartOperation struct {
	OperationID        string
	Status             string
	OldPID             int
	OldStartedAt       time.Time
	OldSocketPath      string
	NewPID             int
	ActiveSessionCount int
	FailureReason      string
	StartedAt          time.Time
	UpdatedAt          time.Time
	CompletedAt        *time.Time
}

// SettingsRestartController exposes the daemon-owned restart action and persisted status surface.
type SettingsRestartController interface {
	RequestRestart(ctx context.Context) (SettingsRestartOperation, error)
	GetRestartOperation(ctx context.Context, operationID string) (SettingsRestartOperation, error)
}

// SettingsUpdateStatus is the daemon-owned software-update snapshot exposed to settings transports.
type SettingsUpdateStatus struct {
	Supported      bool
	Managed        bool
	InstallMethod  string
	CurrentVersion string
	LatestVersion  string
	Available      bool
	Status         string
	Recommendation string
	ReleaseURL     string
	CheckedAt      *time.Time
	LastError      string
}

// SettingsUpdateController exposes the daemon-owned update status surface to settings transports.
type SettingsUpdateController interface {
	GetUpdate(ctx context.Context) (SettingsUpdateStatus, error)
}

// ResourceService exposes the operator-facing desired-state CRUD surface to API transports.
type ResourceService interface {
	List(ctx context.Context, filter resources.ResourceFilter) ([]resources.RawRecord, error)
	Get(ctx context.Context, kind resources.ResourceKind, id string) (resources.RawRecord, error)
	Put(ctx context.Context, draft resources.RawDraft) (resources.RawRecord, error)
	Delete(ctx context.Context, kind resources.ResourceKind, id string, expectedVersion int64) error
}

// ToolRegistry exposes registry projection and dispatch without binding API handlers to backend packages.
type ToolRegistry interface {
	List(ctx context.Context, scope toolspkg.Scope) ([]toolspkg.ToolView, error)
	Search(ctx context.Context, scope toolspkg.Scope, q toolspkg.SearchQuery) ([]toolspkg.ToolView, error)
	Get(ctx context.Context, scope toolspkg.Scope, id toolspkg.ToolID) (toolspkg.ToolView, error)
	Call(ctx context.Context, scope toolspkg.Scope, req toolspkg.CallRequest) (toolspkg.ToolResult, error)
}

// ToolsetRegistry exposes named toolset projections.
type ToolsetRegistry interface {
	ListToolsets(ctx context.Context, scope toolspkg.Scope) ([]toolspkg.ToolsetView, error)
	GetToolset(ctx context.Context, scope toolspkg.Scope, id toolspkg.ToolsetID) (toolspkg.ToolsetView, error)
}

// ToolApprovalIssuer mints local one-shot approval references for operator transports.
type ToolApprovalIssuer interface {
	CreateToolApproval(
		ctx context.Context,
		scope toolspkg.Scope,
		req toolspkg.ApprovalRequest,
	) (toolspkg.ApprovalGrant, error)
}

// AutomationManager exposes automation state and control surfaces to the API layer.
type AutomationManager interface {
	ListJobs(ctx context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error)
	Jobs(ctx context.Context) ([]automationpkg.Job, error)
	GetJob(ctx context.Context, id string) (automationpkg.Job, error)
	CreateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error)
	UpdateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error)
	DeleteJob(ctx context.Context, id string) error
	TriggerJob(ctx context.Context, id string) (automationpkg.Run, error)
	ListTriggers(ctx context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	Triggers(ctx context.Context) ([]automationpkg.Trigger, error)
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
	Runs(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error)
	GetRun(ctx context.Context, id string) (automationpkg.Run, error)
	Status(ctx context.Context) (automationpkg.ManagerStatus, error)
	SetJobEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Job, error)
	SetTriggerEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Trigger, error)
	HandleWebhook(ctx context.Context, request automationpkg.WebhookRequest) (automationpkg.TriggerResult, error)
}

// TaskService exposes task-domain state and lifecycle surfaces to the API layer.
type TaskService interface {
	taskpkg.Manager
}

// WorkspaceService exposes workspace registration and resolution to the API layer.
type WorkspaceService interface {
	Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error)
	Unregister(ctx context.Context, id string) error
	Update(ctx context.Context, id string, opts workspacepkg.UpdateOptions) error
	List(ctx context.Context) ([]workspacepkg.Workspace, error)
	Get(ctx context.Context, idOrNameOrPath string) (workspacepkg.Workspace, error)
	Resolve(ctx context.Context, idOrNameOrPath string) (workspacepkg.ResolvedWorkspace, error)
	ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error)
}
