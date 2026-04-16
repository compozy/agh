// Package core provides the shared transport-facing API layer used by HTTP and UDS bindings.
package core

import (
	"context"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	bundlepkg "github.com/pedronauck/agh/internal/bundles"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

// AgentLoader loads one parsed AGENT.md definition.
type AgentLoader func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error)

// AgentCatalog exposes projected resource-backed agent definitions.
type AgentCatalog interface {
	ListAgents(ctx context.Context) ([]aghconfig.AgentDef, error)
	GetAgent(ctx context.Context, name string) (aghconfig.AgentDef, error)
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
	Transcript(ctx context.Context, id string) ([]transcript.Message, error)
	Stop(ctx context.Context, id string) error
	StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error
	Resume(ctx context.Context, id string) (*session.Session, error)
	Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error)
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
}

// BridgeService is the daemon-owned bridge runtime surface exposed by API transports.
type BridgeService interface {
	bridgepkg.Registry
	bridgepkg.TargetResolver
	ListProviders(ctx context.Context) ([]bridgepkg.BridgeProvider, error)
	ListSecretBindings(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error)
	PutSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error
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
	ListPeers(ctx context.Context, channel string) ([]network.PeerInfo, error)
	ListChannels(ctx context.Context) ([]network.ChannelInfo, error)
	Status(ctx context.Context) (*network.Status, error)
	Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error)
}

// NetworkStore exposes persisted network audit and timeline queries to the API layer.
type NetworkStore interface {
	ListNetworkAudit(ctx context.Context, query store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error)
	ListNetworkMessages(ctx context.Context, query store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error)
}

// DreamTrigger exposes consolidation controls and state to the API layer.
type DreamTrigger interface {
	Trigger(ctx context.Context, workspace string) (bool, string, error)
	LastConsolidatedAt() (time.Time, error)
	Enabled() bool
}

// ResourceService exposes the operator-facing desired-state CRUD surface to API transports.
type ResourceService interface {
	List(ctx context.Context, filter resources.ResourceFilter) ([]resources.RawRecord, error)
	Get(ctx context.Context, kind resources.ResourceKind, id string) (resources.RawRecord, error)
	Put(ctx context.Context, draft resources.RawDraft) (resources.RawRecord, error)
	Delete(ctx context.Context, kind resources.ResourceKind, id string, expectedVersion int64) error
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
		webhookSecret string,
	) (automationpkg.Trigger, error)
	UpdateTrigger(
		ctx context.Context,
		trigger automationpkg.Trigger,
		webhookSecret *string,
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
	CreateTask(ctx context.Context, spec taskpkg.CreateTask, actor taskpkg.ActorContext) (*taskpkg.Task, error)
	CreateChildTask(
		ctx context.Context,
		parentTaskID string,
		spec taskpkg.CreateTask,
		actor taskpkg.ActorContext,
	) (*taskpkg.Task, error)
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

	AddDependency(ctx context.Context, spec taskpkg.AddDependency, actor taskpkg.ActorContext) error
	RemoveDependency(ctx context.Context, taskID string, dependsOnID string, actor taskpkg.ActorContext) error

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

	GetTask(ctx context.Context, id string, actor taskpkg.ActorContext) (*taskpkg.View, error)
	ListTaskRuns(
		ctx context.Context,
		taskID string,
		query taskpkg.RunQuery,
		actor taskpkg.ActorContext,
	) ([]taskpkg.Run, error)
	ListTasks(ctx context.Context, query taskpkg.Query, actor taskpkg.ActorContext) ([]taskpkg.Summary, error)
}

// SkillsRegistry exposes the skill catalog to the API layer.
type SkillsRegistry interface {
	Get(name string) (*skills.Skill, bool)
	List() []*skills.Skill
	ForWorkspace(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	LoadContent(ctx context.Context, skill *skills.Skill) (string, error)
	SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
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
