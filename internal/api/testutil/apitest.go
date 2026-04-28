package testutil

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
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

var ErrStubWorkspaceServiceNotImplemented = errors.New("stub workspace service method not implemented")

// ConfigWithDisabledNetwork returns a default test config with networking turned off.
func ConfigWithDisabledNetwork(homePaths aghconfig.HomePaths) aghconfig.Config {
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Network.Enabled = false
	return cfg
}

type StubSessionManager struct {
	CreateFn        func(context.Context, session.CreateOpts) (*session.Session, error)
	ListFn          func() []*session.Info
	ListAllFn       func(context.Context) ([]*session.Info, error)
	StatusFn        func(context.Context, string) (*session.Info, error)
	EventsFn        func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	HistoryFn       func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	TranscriptFn    func(context.Context, string) ([]transcript.UIMessage, error)
	RepairFn        func(context.Context, session.RepairOpts) (*session.RepairResult, error)
	DeleteFn        func(context.Context, string) error
	StopFn          func(context.Context, string) error
	StopWithCauseFn func(context.Context, string, session.StopCause, string) error
	ResumeFn        func(context.Context, string) (*session.Session, error)
	ClearFn         func(context.Context, string) (*session.Session, error)
	PromptFn        func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	CancelPromptFn  func(context.Context, string) error
	ApproveFn       func(context.Context, string, acp.ApproveRequest) error
}

func (s StubSessionManager) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.CreateFn != nil {
		return s.CreateFn(ctx, opts)
	}
	return nil, nil
}

func (s StubSessionManager) List() []*session.Info {
	if s.ListFn != nil {
		return s.ListFn()
	}
	if s.ListAllFn != nil {
		infos, err := s.ListAllFn(context.Background())
		if err != nil {
			return []*session.Info{}
		}
		return infos
	}
	return nil
}

func (s StubSessionManager) ListAll(ctx context.Context) ([]*session.Info, error) {
	if s.ListAllFn != nil {
		return s.ListAllFn(ctx)
	}
	return nil, nil
}

func (s StubSessionManager) Status(ctx context.Context, id string) (*session.Info, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) Events(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.SessionEvent, error) {
	if s.EventsFn != nil {
		return s.EventsFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) History(
	ctx context.Context,
	id string,
	query store.EventQuery,
) ([]store.TurnHistory, error) {
	if s.HistoryFn != nil {
		return s.HistoryFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) Transcript(ctx context.Context, id string) ([]transcript.UIMessage, error) {
	if s.TranscriptFn != nil {
		return s.TranscriptFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) RepairSession(
	ctx context.Context,
	opts session.RepairOpts,
) (*session.RepairResult, error) {
	if s.RepairFn != nil {
		return s.RepairFn(ctx, opts)
	}
	return &session.RepairResult{SessionID: opts.SessionID}, nil
}

func (s StubSessionManager) Delete(ctx context.Context, id string) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) Stop(ctx context.Context, id string) error {
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) StopWithCause(
	ctx context.Context,
	id string,
	cause session.StopCause,
	detail string,
) error {
	if s.StopWithCauseFn != nil {
		return s.StopWithCauseFn(ctx, id, cause, detail)
	}
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) Resume(ctx context.Context, id string) (*session.Session, error) {
	if s.ResumeFn != nil {
		return s.ResumeFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) ClearConversation(
	ctx context.Context,
	id string,
) (*session.Session, error) {
	if s.ClearFn != nil {
		return s.ClearFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.PromptFn != nil {
		return s.PromptFn(ctx, id, msg)
	}
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
}

func (s StubSessionManager) CancelPrompt(ctx context.Context, id string) error {
	if s.CancelPromptFn != nil {
		return s.CancelPromptFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if s.ApproveFn != nil {
		return s.ApproveFn(ctx, id, req)
	}
	return nil
}

type StubObserver struct {
	QueryEventsFn        func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error)
	QueryHookCatalogFn   func(context.Context, hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error)
	QueryHookRunsFn      func(context.Context, store.HookRunQuery) ([]hookspkg.HookRunRecord, error)
	QueryHookEventsFn    func(context.Context, hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error)
	QueryBridgeHealthFn  func(context.Context) ([]observe.BridgeInstanceHealth, error)
	HealthFn             func(context.Context) (observe.Health, error)
	QueryTaskDashboardFn func(context.Context, observe.TaskDashboardQuery) (observe.TaskDashboardView, error)
	QueryTaskInboxFn     func(
		context.Context,
		observe.TaskInboxQuery,
		taskpkg.ActorIdentity,
	) (observe.TaskInboxView, error)
}

type StubAutomationManager struct {
	ListJobsFn          func(context.Context, automationpkg.JobListQuery) ([]automationpkg.Job, error)
	JobsFn              func(context.Context) ([]automationpkg.Job, error)
	GetJobFn            func(context.Context, string) (automationpkg.Job, error)
	CreateJobFn         func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	UpdateJobFn         func(context.Context, automationpkg.Job) (automationpkg.Job, error)
	DeleteJobFn         func(context.Context, string) error
	TriggerJobFn        func(context.Context, string) (automationpkg.Run, error)
	ListTriggersFn      func(context.Context, automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error)
	TriggersFn          func(context.Context) ([]automationpkg.Trigger, error)
	GetTriggerFn        func(context.Context, string) (automationpkg.Trigger, error)
	CreateTriggerFn     func(context.Context, automationpkg.Trigger, string) (automationpkg.Trigger, error)
	UpdateTriggerFn     func(context.Context, automationpkg.Trigger, *string) (automationpkg.Trigger, error)
	DeleteTriggerFn     func(context.Context, string) error
	ListRunsFn          func(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error)
	RunsFn              func(context.Context, automationpkg.RunQuery) ([]automationpkg.Run, error)
	GetRunFn            func(context.Context, string) (automationpkg.Run, error)
	StatusFn            func(context.Context) (automationpkg.ManagerStatus, error)
	SetJobEnabledFn     func(context.Context, string, bool) (automationpkg.Job, error)
	SetTriggerEnabledFn func(context.Context, string, bool) (automationpkg.Trigger, error)
	HandleWebhookFn     func(context.Context, automationpkg.WebhookRequest) (automationpkg.TriggerResult, error)
}

type StubTaskManager struct {
	CreateTaskFn      func(context.Context, taskpkg.CreateTask, taskpkg.ActorContext) (*taskpkg.Task, error)
	CreateChildTaskFn func(
		context.Context,
		string,
		taskpkg.CreateTask,
		taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	DeleteTaskFn  func(context.Context, string, taskpkg.ActorContext) error
	UpdateTaskFn  func(context.Context, string, taskpkg.Patch, taskpkg.ActorContext) (*taskpkg.Task, error)
	PublishTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	StartTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	ApproveTaskFn func(
		context.Context,
		string,
		taskpkg.ExecutionRequest,
		taskpkg.ActorContext,
	) (*taskpkg.Execution, error)
	RejectTaskFn func(context.Context, string, taskpkg.ActorContext) (*taskpkg.Task, error)
	CancelTaskFn func(
		context.Context,
		string,
		taskpkg.CancelTask,
		taskpkg.ActorContext,
	) (*taskpkg.Task, error)
	MarkTaskReadFn     func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	ArchiveTaskFn      func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	DismissTaskFn      func(context.Context, string, taskpkg.ActorContext) (taskpkg.TriageState, error)
	AddDependencyFn    func(context.Context, taskpkg.AddDependency, taskpkg.ActorContext) error
	RemoveDependencyFn func(context.Context, string, string, taskpkg.ActorContext) error
	EnqueueRunFn       func(context.Context, taskpkg.EnqueueRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	ClaimNextRunFn     func(
		context.Context,
		taskpkg.ClaimCriteria,
		taskpkg.ActorContext,
	) (*taskpkg.ClaimResult, error)
	ClaimRunFn                func(context.Context, string, taskpkg.ClaimRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	StartRunFn                func(context.Context, string, taskpkg.StartRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	AttachRunSessionFn        func(context.Context, string, string, taskpkg.ActorContext) (*taskpkg.Run, error)
	HeartbeatRunLeaseFn       func(context.Context, taskpkg.LeaseHeartbeat, taskpkg.ActorContext) (*taskpkg.Run, error)
	ReleaseRunLeaseFn         func(context.Context, taskpkg.LeaseRelease, taskpkg.ActorContext) (*taskpkg.Run, error)
	CompleteRunLeaseFn        func(context.Context, taskpkg.LeaseCompletion, taskpkg.ActorContext) (*taskpkg.Run, error)
	FailRunLeaseFn            func(context.Context, taskpkg.LeaseFailure, taskpkg.ActorContext) (*taskpkg.Run, error)
	CompleteRunFn             func(context.Context, string, taskpkg.RunResult, taskpkg.ActorContext) (*taskpkg.Run, error)
	FailRunFn                 func(context.Context, string, taskpkg.RunFailure, taskpkg.ActorContext) (*taskpkg.Run, error)
	CancelRunFn               func(context.Context, string, taskpkg.CancelRun, taskpkg.ActorContext) (*taskpkg.Run, error)
	RecoverExpiredRunLeasesFn func(
		context.Context,
		taskpkg.ExpiredLeaseRecovery,
		taskpkg.ActorContext,
	) ([]taskpkg.ExpiredLeaseRecoveryResult, error)
	GetTaskFn      func(context.Context, string, taskpkg.ActorContext) (*taskpkg.View, error)
	ListTaskRunsFn func(context.Context, string, taskpkg.RunQuery, taskpkg.ActorContext) ([]taskpkg.Run, error)
	ListTasksFn    func(context.Context, taskpkg.Query, taskpkg.ActorContext) ([]taskpkg.Summary, error)
	TimelineFn     func(
		context.Context,
		string,
		taskpkg.TimelineQuery,
		taskpkg.ActorContext,
	) ([]taskpkg.TimelineItem, error)
	StreamFn func(
		context.Context,
		string,
		taskpkg.StreamQuery,
		taskpkg.ActorContext,
	) (<-chan taskpkg.StreamEvent, error)
	TreeFn      func(context.Context, string, taskpkg.ActorContext) (*taskpkg.TreeView, error)
	RunDetailFn func(context.Context, string, taskpkg.ActorContext) (*taskpkg.RunDetailView, error)
}

var _ core.TaskService = (*StubTaskManager)(nil)

type StubResourceService struct {
	ListFn   func(context.Context, resources.ResourceFilter) ([]resources.RawRecord, error)
	GetFn    func(context.Context, resources.ResourceKind, string) (resources.RawRecord, error)
	PutFn    func(context.Context, resources.RawDraft) (resources.RawRecord, error)
	DeleteFn func(context.Context, resources.ResourceKind, string, int64) error
}

var _ core.ResourceService = (*StubResourceService)(nil)

func (s StubResourceService) List(
	ctx context.Context,
	filter resources.ResourceFilter,
) ([]resources.RawRecord, error) {
	if s.ListFn != nil {
		return s.ListFn(ctx, filter)
	}
	return nil, nil
}

func (s StubResourceService) Get(
	ctx context.Context,
	kind resources.ResourceKind,
	id string,
) (resources.RawRecord, error) {
	if s.GetFn != nil {
		return s.GetFn(ctx, kind, id)
	}
	return resources.RawRecord{}, resources.ErrNotFound
}

func (s StubResourceService) Put(
	ctx context.Context,
	draft resources.RawDraft,
) (resources.RawRecord, error) {
	if s.PutFn != nil {
		return s.PutFn(ctx, draft)
	}
	return resources.RawRecord{
		Kind:      draft.Kind,
		ID:        draft.ID,
		Version:   1,
		Scope:     draft.Scope,
		Owner:     resources.ResourceOwner{Kind: "daemon", ID: "daemon-control"},
		Source:    resources.ResourceSource{Kind: "daemon", ID: "system"},
		SpecJSON:  append([]byte(nil), draft.SpecJSON...),
		CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (s StubResourceService) Delete(
	ctx context.Context,
	kind resources.ResourceKind,
	id string,
	expectedVersion int64,
) error {
	if s.DeleteFn != nil {
		return s.DeleteFn(ctx, kind, id, expectedVersion)
	}
	return nil
}

func (s StubAutomationManager) ListJobs(
	ctx context.Context,
	query automationpkg.JobListQuery,
) ([]automationpkg.Job, error) {
	if s.ListJobsFn != nil {
		return s.ListJobsFn(ctx, query)
	}
	if s.JobsFn != nil {
		return s.JobsFn(ctx)
	}
	return nil, nil
}

func (s StubAutomationManager) Jobs(ctx context.Context) ([]automationpkg.Job, error) {
	if s.JobsFn != nil {
		return s.JobsFn(ctx)
	}
	return s.ListJobs(ctx, automationpkg.JobListQuery{})
}

func (s StubAutomationManager) GetJob(ctx context.Context, id string) (automationpkg.Job, error) {
	if s.GetJobFn != nil {
		return s.GetJobFn(ctx, id)
	}
	return automationpkg.Job{}, automationpkg.ErrJobNotFound
}

func (s StubAutomationManager) CreateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.CreateJobFn != nil {
		return s.CreateJobFn(ctx, job)
	}
	return job, nil
}

func (s StubAutomationManager) UpdateJob(ctx context.Context, job automationpkg.Job) (automationpkg.Job, error) {
	if s.UpdateJobFn != nil {
		return s.UpdateJobFn(ctx, job)
	}
	return job, nil
}

func (s StubAutomationManager) DeleteJob(ctx context.Context, id string) error {
	if s.DeleteJobFn != nil {
		return s.DeleteJobFn(ctx, id)
	}
	return nil
}

func (s StubAutomationManager) TriggerJob(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.TriggerJobFn != nil {
		return s.TriggerJobFn(ctx, id)
	}
	return automationpkg.Run{}, nil
}

func (s StubAutomationManager) ListTriggers(
	ctx context.Context,
	query automationpkg.TriggerListQuery,
) ([]automationpkg.Trigger, error) {
	if s.ListTriggersFn != nil {
		return s.ListTriggersFn(ctx, query)
	}
	if s.TriggersFn != nil {
		return s.TriggersFn(ctx)
	}
	return nil, nil
}

func (s StubAutomationManager) Triggers(ctx context.Context) ([]automationpkg.Trigger, error) {
	if s.TriggersFn != nil {
		return s.TriggersFn(ctx)
	}
	return s.ListTriggers(ctx, automationpkg.TriggerListQuery{})
}

func (s StubAutomationManager) GetTrigger(ctx context.Context, id string) (automationpkg.Trigger, error) {
	if s.GetTriggerFn != nil {
		return s.GetTriggerFn(ctx, id)
	}
	return automationpkg.Trigger{}, automationpkg.ErrTriggerNotFound
}

func (s StubAutomationManager) CreateTrigger(
	ctx context.Context,
	trigger automationpkg.Trigger,
	secret string,
) (automationpkg.Trigger, error) {
	if s.CreateTriggerFn != nil {
		return s.CreateTriggerFn(ctx, trigger, secret)
	}
	return trigger, nil
}

func (s StubAutomationManager) UpdateTrigger(
	ctx context.Context,
	trigger automationpkg.Trigger,
	secret *string,
) (automationpkg.Trigger, error) {
	if s.UpdateTriggerFn != nil {
		return s.UpdateTriggerFn(ctx, trigger, secret)
	}
	return trigger, nil
}

func (s StubAutomationManager) DeleteTrigger(ctx context.Context, id string) error {
	if s.DeleteTriggerFn != nil {
		return s.DeleteTriggerFn(ctx, id)
	}
	return nil
}

func (s StubAutomationManager) ListRuns(
	ctx context.Context,
	query automationpkg.RunQuery,
) ([]automationpkg.Run, error) {
	if s.ListRunsFn != nil {
		return s.ListRunsFn(ctx, query)
	}
	if s.RunsFn != nil {
		return s.RunsFn(ctx, query)
	}
	return nil, nil
}

func (s StubAutomationManager) Runs(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
	if s.RunsFn != nil {
		return s.RunsFn(ctx, query)
	}
	return s.ListRuns(ctx, query)
}

func (s StubAutomationManager) GetRun(ctx context.Context, id string) (automationpkg.Run, error) {
	if s.GetRunFn != nil {
		return s.GetRunFn(ctx, id)
	}
	return automationpkg.Run{}, automationpkg.ErrRunNotFound
}

func (s StubAutomationManager) Status(ctx context.Context) (automationpkg.ManagerStatus, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx)
	}
	return automationpkg.ManagerStatus{}, nil
}

func (s StubAutomationManager) SetJobEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Job, error) {
	if s.SetJobEnabledFn != nil {
		return s.SetJobEnabledFn(ctx, id, enabled)
	}
	return automationpkg.Job{}, nil
}

func (s StubAutomationManager) SetTriggerEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (automationpkg.Trigger, error) {
	if s.SetTriggerEnabledFn != nil {
		return s.SetTriggerEnabledFn(ctx, id, enabled)
	}
	return automationpkg.Trigger{}, nil
}

func (s StubAutomationManager) HandleWebhook(
	ctx context.Context,
	request automationpkg.WebhookRequest,
) (automationpkg.TriggerResult, error) {
	if s.HandleWebhookFn != nil {
		return s.HandleWebhookFn(ctx, request)
	}
	return automationpkg.TriggerResult{}, nil
}

func (s StubTaskManager) CreateTask(
	ctx context.Context,
	spec taskpkg.CreateTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CreateTaskFn != nil {
		return s.CreateTaskFn(ctx, spec, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) CreateChildTask(
	ctx context.Context,
	parentTaskID string,
	spec taskpkg.CreateTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CreateChildTaskFn != nil {
		return s.CreateChildTaskFn(ctx, parentTaskID, spec, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) DeleteTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) error {
	if s.DeleteTaskFn != nil {
		return s.DeleteTaskFn(ctx, id, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) UpdateTask(
	ctx context.Context,
	id string,
	patch taskpkg.Patch,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.UpdateTaskFn != nil {
		return s.UpdateTaskFn(ctx, id, patch, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) PublishTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.PublishTaskFn != nil {
		return s.PublishTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) StartTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.StartTaskFn != nil {
		return s.StartTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ApproveTask(
	ctx context.Context,
	id string,
	req taskpkg.ExecutionRequest,
	actor taskpkg.ActorContext,
) (*taskpkg.Execution, error) {
	if s.ApproveTaskFn != nil {
		return s.ApproveTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RejectTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.RejectTaskFn != nil {
		return s.RejectTaskFn(ctx, id, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) CancelTask(
	ctx context.Context,
	id string,
	req taskpkg.CancelTask,
	actor taskpkg.ActorContext,
) (*taskpkg.Task, error) {
	if s.CancelTaskFn != nil {
		return s.CancelTaskFn(ctx, id, req, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) MarkTaskRead(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.MarkTaskReadFn != nil {
		return s.MarkTaskReadFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ArchiveTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.ArchiveTaskFn != nil {
		return s.ArchiveTaskFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) DismissTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (taskpkg.TriageState, error) {
	if s.DismissTaskFn != nil {
		return s.DismissTaskFn(ctx, id, actor)
	}
	return taskpkg.TriageState{}, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) AddDependency(
	ctx context.Context,
	spec taskpkg.AddDependency,
	actor taskpkg.ActorContext,
) error {
	if s.AddDependencyFn != nil {
		return s.AddDependencyFn(ctx, spec, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RemoveDependency(
	ctx context.Context,
	taskID string,
	dependsOnID string,
	actor taskpkg.ActorContext,
) error {
	if s.RemoveDependencyFn != nil {
		return s.RemoveDependencyFn(ctx, taskID, dependsOnID, actor)
	}
	return taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) EnqueueRun(
	ctx context.Context,
	spec taskpkg.EnqueueRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.EnqueueRunFn != nil {
		return s.EnqueueRunFn(ctx, spec, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) ClaimNextRun(
	ctx context.Context,
	criteria taskpkg.ClaimCriteria,
	actor taskpkg.ActorContext,
) (*taskpkg.ClaimResult, error) {
	if s.ClaimNextRunFn != nil {
		return s.ClaimNextRunFn(ctx, criteria, actor)
	}
	return nil, taskpkg.ErrNoClaimableRun
}

func (s StubTaskManager) ClaimRun(
	ctx context.Context,
	runID string,
	claim taskpkg.ClaimRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.ClaimRunFn != nil {
		return s.ClaimRunFn(ctx, runID, claim, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) StartRun(
	ctx context.Context,
	runID string,
	req taskpkg.StartRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.StartRunFn != nil {
		return s.StartRunFn(ctx, runID, req, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) AttachRunSession(
	ctx context.Context,
	runID string,
	sessionID string,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.AttachRunSessionFn != nil {
		return s.AttachRunSessionFn(ctx, runID, sessionID, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) HeartbeatRunLease(
	ctx context.Context,
	heartbeat taskpkg.LeaseHeartbeat,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.HeartbeatRunLeaseFn != nil {
		return s.HeartbeatRunLeaseFn(ctx, heartbeat, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) ReleaseRunLease(
	ctx context.Context,
	release taskpkg.LeaseRelease,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.ReleaseRunLeaseFn != nil {
		return s.ReleaseRunLeaseFn(ctx, release, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CompleteRunLease(
	ctx context.Context,
	completion taskpkg.LeaseCompletion,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CompleteRunLeaseFn != nil {
		return s.CompleteRunLeaseFn(ctx, completion, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) FailRunLease(
	ctx context.Context,
	failure taskpkg.LeaseFailure,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.FailRunLeaseFn != nil {
		return s.FailRunLeaseFn(ctx, failure, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CompleteRun(
	ctx context.Context,
	runID string,
	result taskpkg.RunResult,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CompleteRunFn != nil {
		return s.CompleteRunFn(ctx, runID, result, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) FailRun(
	ctx context.Context,
	runID string,
	failure taskpkg.RunFailure,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.FailRunFn != nil {
		return s.FailRunFn(ctx, runID, failure, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) CancelRun(
	ctx context.Context,
	runID string,
	req taskpkg.CancelRun,
	actor taskpkg.ActorContext,
) (*taskpkg.Run, error) {
	if s.CancelRunFn != nil {
		return s.CancelRunFn(ctx, runID, req, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubTaskManager) RecoverExpiredRunLeases(
	ctx context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
	actor taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	if s.RecoverExpiredRunLeasesFn != nil {
		return s.RecoverExpiredRunLeasesFn(ctx, recovery, actor)
	}
	return nil, nil
}

func (s StubTaskManager) GetTask(
	ctx context.Context,
	id string,
	actor taskpkg.ActorContext,
) (*taskpkg.View, error) {
	if s.GetTaskFn != nil {
		return s.GetTaskFn(ctx, id, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) ListTaskRuns(
	ctx context.Context,
	taskID string,
	query taskpkg.RunQuery,
	actor taskpkg.ActorContext,
) ([]taskpkg.Run, error) {
	if s.ListTaskRunsFn != nil {
		return s.ListTaskRunsFn(ctx, taskID, query, actor)
	}
	return nil, nil
}

func (s StubTaskManager) ListTasks(
	ctx context.Context,
	query taskpkg.Query,
	actor taskpkg.ActorContext,
) ([]taskpkg.Summary, error) {
	if s.ListTasksFn != nil {
		return s.ListTasksFn(ctx, query, actor)
	}
	return nil, nil
}

func (s StubTaskManager) Timeline(
	ctx context.Context,
	taskID string,
	query taskpkg.TimelineQuery,
	actor taskpkg.ActorContext,
) ([]taskpkg.TimelineItem, error) {
	if s.TimelineFn != nil {
		return s.TimelineFn(ctx, taskID, query, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) Stream(
	ctx context.Context,
	taskID string,
	query taskpkg.StreamQuery,
	actor taskpkg.ActorContext,
) (<-chan taskpkg.StreamEvent, error) {
	if s.StreamFn != nil {
		return s.StreamFn(ctx, taskID, query, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) Tree(
	ctx context.Context,
	taskID string,
	actor taskpkg.ActorContext,
) (*taskpkg.TreeView, error) {
	if s.TreeFn != nil {
		return s.TreeFn(ctx, taskID, actor)
	}
	return nil, taskpkg.ErrTaskNotFound
}

func (s StubTaskManager) RunDetail(
	ctx context.Context,
	runID string,
	actor taskpkg.ActorContext,
) (*taskpkg.RunDetailView, error) {
	if s.RunDetailFn != nil {
		return s.RunDetailFn(ctx, runID, actor)
	}
	return nil, taskpkg.ErrTaskRunNotFound
}

func (s StubObserver) QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
	if s.QueryEventsFn != nil {
		return s.QueryEventsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) QueryTaskDashboard(
	ctx context.Context,
	query observe.TaskDashboardQuery,
) (observe.TaskDashboardView, error) {
	if s.QueryTaskDashboardFn != nil {
		return s.QueryTaskDashboardFn(ctx, query)
	}
	return observe.TaskDashboardView{}, nil
}

func (s StubObserver) QueryTaskInbox(
	ctx context.Context,
	query observe.TaskInboxQuery,
	actor taskpkg.ActorIdentity,
) (observe.TaskInboxView, error) {
	if s.QueryTaskInboxFn != nil {
		return s.QueryTaskInboxFn(ctx, query, actor)
	}
	return observe.TaskInboxView{}, nil
}

type StubNetworkService struct {
	SendFn         func(context.Context, network.SendRequest) (string, error)
	ListPeersFn    func(context.Context, string) ([]network.PeerInfo, error)
	ListChannelsFn func(context.Context) ([]network.ChannelInfo, error)
	StatusFn       func(context.Context) (*network.Status, error)
	InboxFn        func(context.Context, string) ([]network.Envelope, error)
	WaitInboxFn    func(context.Context, string, string) ([]network.Envelope, error)
}

type StubNetworkStore struct {
	GetNetworkChannelFn    func(context.Context, string) (store.NetworkChannelEntry, error)
	ListNetworkChannelsFn  func(context.Context, store.NetworkChannelQuery) ([]store.NetworkChannelEntry, error)
	WriteNetworkChannelFn  func(context.Context, store.NetworkChannelEntry) error
	DeleteNetworkChannelFn func(context.Context, string) error
	ListNetworkAuditFn     func(context.Context, store.NetworkAuditQuery) ([]store.NetworkAuditEntry, error)
	ListNetworkMessagesFn  func(context.Context, store.NetworkMessageQuery) ([]store.NetworkMessageEntry, error)
}

func (s StubNetworkService) Send(ctx context.Context, req network.SendRequest) (string, error) {
	if s.SendFn != nil {
		return s.SendFn(ctx, req)
	}
	return "", nil
}

func (s StubNetworkService) ListPeers(ctx context.Context, channel string) ([]network.PeerInfo, error) {
	if s.ListPeersFn != nil {
		return s.ListPeersFn(ctx, channel)
	}
	return nil, nil
}

func (s StubNetworkService) ListChannels(ctx context.Context) ([]network.ChannelInfo, error) {
	if s.ListChannelsFn != nil {
		return s.ListChannelsFn(ctx)
	}
	return nil, nil
}

func (s StubNetworkService) Status(ctx context.Context) (*network.Status, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx)
	}
	return nil, nil
}

func (s StubNetworkService) Inbox(ctx context.Context, sessionID string) ([]network.Envelope, error) {
	if s.InboxFn != nil {
		return s.InboxFn(ctx, sessionID)
	}
	return nil, nil
}

func (s StubNetworkService) WaitInbox(
	ctx context.Context,
	sessionID string,
	channel string,
) ([]network.Envelope, error) {
	if s.WaitInboxFn != nil {
		return s.WaitInboxFn(ctx, sessionID, channel)
	}
	return nil, errors.New("stub network service WaitInbox not implemented")
}

func (s StubNetworkStore) ListNetworkAudit(
	ctx context.Context,
	query store.NetworkAuditQuery,
) ([]store.NetworkAuditEntry, error) {
	if s.ListNetworkAuditFn != nil {
		return s.ListNetworkAuditFn(ctx, query)
	}
	return nil, nil
}

func (s StubNetworkStore) GetNetworkChannel(
	ctx context.Context,
	channel string,
) (store.NetworkChannelEntry, error) {
	if s.GetNetworkChannelFn != nil {
		return s.GetNetworkChannelFn(ctx, channel)
	}
	return store.NetworkChannelEntry{}, sql.ErrNoRows
}

func (s StubNetworkStore) ListNetworkChannels(
	ctx context.Context,
	query store.NetworkChannelQuery,
) ([]store.NetworkChannelEntry, error) {
	if s.ListNetworkChannelsFn != nil {
		return s.ListNetworkChannelsFn(ctx, query)
	}
	return nil, nil
}

func (s StubNetworkStore) WriteNetworkChannel(
	ctx context.Context,
	entry store.NetworkChannelEntry,
) error {
	if s.WriteNetworkChannelFn != nil {
		return s.WriteNetworkChannelFn(ctx, entry)
	}
	return nil
}

func (s StubNetworkStore) DeleteNetworkChannel(ctx context.Context, channel string) error {
	if s.DeleteNetworkChannelFn != nil {
		return s.DeleteNetworkChannelFn(ctx, channel)
	}
	return nil
}

func (s StubNetworkStore) ListNetworkMessages(
	ctx context.Context,
	query store.NetworkMessageQuery,
) ([]store.NetworkMessageEntry, error) {
	if s.ListNetworkMessagesFn != nil {
		return s.ListNetworkMessagesFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) Health(ctx context.Context) (observe.Health, error) {
	if s.HealthFn != nil {
		return s.HealthFn(ctx)
	}
	return observe.Health{Status: "ok"}, nil
}

func (s StubObserver) QueryBridgeHealth(ctx context.Context) ([]observe.BridgeInstanceHealth, error) {
	if s.QueryBridgeHealthFn != nil {
		return s.QueryBridgeHealthFn(ctx)
	}
	return nil, nil
}

func (s StubObserver) QueryHookCatalog(
	ctx context.Context,
	filter hookspkg.CatalogFilter,
) ([]hookspkg.CatalogEntry, error) {
	if s.QueryHookCatalogFn != nil {
		return s.QueryHookCatalogFn(ctx, filter)
	}
	return nil, nil
}

func (s StubObserver) QueryHookRuns(ctx context.Context, query store.HookRunQuery) ([]hookspkg.HookRunRecord, error) {
	if s.QueryHookRunsFn != nil {
		return s.QueryHookRunsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) QueryHookEvents(
	ctx context.Context,
	filter hookspkg.EventFilter,
) ([]hookspkg.EventDescriptor, error) {
	if s.QueryHookEventsFn != nil {
		return s.QueryHookEventsFn(ctx, filter)
	}
	return nil, nil
}

type StubBridgeService struct {
	CreateInstanceFn        func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error)
	GetInstanceFn           func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	ListInstancesFn         func(context.Context) ([]bridgepkg.BridgeInstance, error)
	ListProvidersFn         func(context.Context) ([]bridgepkg.BridgeProvider, error)
	ListSecretBindingsFn    func(context.Context, string) ([]bridgepkg.BridgeSecretBinding, error)
	PutSecretBindingFn      func(context.Context, bridgepkg.BridgeSecretBinding) error
	DeleteSecretBindingFn   func(context.Context, string, string) error
	UpdateInstanceFn        func(context.Context, bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error)
	UpdateInstanceStateFn   func(context.Context, bridgepkg.UpdateInstanceStateRequest) (*bridgepkg.BridgeInstance, error)
	BuildRoutingKeyFn       func(context.Context, bridgepkg.RoutingKey) (bridgepkg.RoutingKey, error)
	ResolveRouteFn          func(context.Context, bridgepkg.RoutingKey) (*bridgepkg.BridgeRoute, error)
	ResolveOrCreateRouteFn  func(context.Context, bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, bool, error)
	UpsertRouteFn           func(context.Context, bridgepkg.BridgeRoute) (*bridgepkg.BridgeRoute, error)
	ListRoutesFn            func(context.Context, string) ([]bridgepkg.BridgeRoute, error)
	ResolveDeliveryTargetFn func(
		context.Context,
		bridgepkg.ResolveDeliveryTargetRequest,
	) (*bridgepkg.DeliveryTarget, error)
	StartInstanceFn   func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	StopInstanceFn    func(context.Context, string) (*bridgepkg.BridgeInstance, error)
	RestartInstanceFn func(context.Context, string) (*bridgepkg.BridgeInstance, error)
}

var _ core.BridgeService = (*StubBridgeService)(nil)

func (s StubBridgeService) CreateInstance(
	ctx context.Context,
	req bridgepkg.CreateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.CreateInstanceFn != nil {
		return s.CreateInstanceFn(ctx, req)
	}
	return nil, nil
}

func (s StubBridgeService) GetInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.GetInstanceFn != nil {
		return s.GetInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) ListInstances(ctx context.Context) ([]bridgepkg.BridgeInstance, error) {
	if s.ListInstancesFn != nil {
		return s.ListInstancesFn(ctx)
	}
	return nil, nil
}

func (s StubBridgeService) ListProviders(ctx context.Context) ([]bridgepkg.BridgeProvider, error) {
	if s.ListProvidersFn != nil {
		return s.ListProvidersFn(ctx)
	}
	return nil, nil
}

func (s StubBridgeService) ListSecretBindings(
	ctx context.Context,
	bridgeInstanceID string,
) ([]bridgepkg.BridgeSecretBinding, error) {
	if s.ListSecretBindingsFn != nil {
		return s.ListSecretBindingsFn(ctx, bridgeInstanceID)
	}
	return nil, nil
}

func (s StubBridgeService) PutSecretBinding(ctx context.Context, binding bridgepkg.BridgeSecretBinding) error {
	if s.PutSecretBindingFn != nil {
		return s.PutSecretBindingFn(ctx, binding)
	}
	return nil
}

func (s StubBridgeService) DeleteSecretBinding(ctx context.Context, bridgeInstanceID string, bindingName string) error {
	if s.DeleteSecretBindingFn != nil {
		return s.DeleteSecretBindingFn(ctx, bridgeInstanceID, bindingName)
	}
	return bridgepkg.ErrBridgeSecretBindingNotFound
}

func (s StubBridgeService) UpdateInstance(
	ctx context.Context,
	req bridgepkg.UpdateInstanceRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.UpdateInstanceFn != nil {
		return s.UpdateInstanceFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) UpdateInstanceState(
	ctx context.Context,
	req bridgepkg.UpdateInstanceStateRequest,
) (*bridgepkg.BridgeInstance, error) {
	if s.UpdateInstanceStateFn != nil {
		return s.UpdateInstanceStateFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) BuildRoutingKey(
	ctx context.Context,
	key bridgepkg.RoutingKey,
) (bridgepkg.RoutingKey, error) {
	if s.BuildRoutingKeyFn != nil {
		return s.BuildRoutingKeyFn(ctx, key)
	}
	return bridgepkg.RoutingKey{}, nil
}

func (s StubBridgeService) ResolveRoute(ctx context.Context, key bridgepkg.RoutingKey) (*bridgepkg.BridgeRoute, error) {
	if s.ResolveRouteFn != nil {
		return s.ResolveRouteFn(ctx, key)
	}
	return nil, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) ResolveOrCreateRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, bool, error) {
	if s.ResolveOrCreateRouteFn != nil {
		return s.ResolveOrCreateRouteFn(ctx, route)
	}
	return nil, false, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) UpsertRoute(
	ctx context.Context,
	route bridgepkg.BridgeRoute,
) (*bridgepkg.BridgeRoute, error) {
	if s.UpsertRouteFn != nil {
		return s.UpsertRouteFn(ctx, route)
	}
	return nil, bridgepkg.ErrBridgeRouteNotFound
}

func (s StubBridgeService) ListRoutes(ctx context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
	if s.ListRoutesFn != nil {
		return s.ListRoutesFn(ctx, bridgeInstanceID)
	}
	return nil, nil
}

func (s StubBridgeService) ResolveDeliveryTarget(
	ctx context.Context,
	req bridgepkg.ResolveDeliveryTargetRequest,
) (*bridgepkg.DeliveryTarget, error) {
	if s.ResolveDeliveryTargetFn != nil {
		return s.ResolveDeliveryTargetFn(ctx, req)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) StartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.StartInstanceFn != nil {
		return s.StartInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) StopInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.StopInstanceFn != nil {
		return s.StopInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

func (s StubBridgeService) RestartInstance(ctx context.Context, id string) (*bridgepkg.BridgeInstance, error) {
	if s.RestartInstanceFn != nil {
		return s.RestartInstanceFn(ctx, id)
	}
	return nil, bridgepkg.ErrBridgeInstanceNotFound
}

type StubWorkspaceService struct {
	RegisterFn          func(context.Context, workspacepkg.RegisterOptions) (workspacepkg.Workspace, error)
	UnregisterFn        func(context.Context, string) error
	UpdateFn            func(context.Context, string, workspacepkg.UpdateOptions) error
	ListFn              func(context.Context) ([]workspacepkg.Workspace, error)
	GetFn               func(context.Context, string) (workspacepkg.Workspace, error)
	ResolveFn           func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
	ResolveOrRegisterFn func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
}

func (s StubWorkspaceService) Register(
	ctx context.Context,
	opts workspacepkg.RegisterOptions,
) (workspacepkg.Workspace, error) {
	if s.RegisterFn != nil {
		return s.RegisterFn(ctx, opts)
	}
	return workspacepkg.Workspace{}, ErrStubWorkspaceServiceNotImplemented
}

func (s StubWorkspaceService) Unregister(ctx context.Context, id string) error {
	if s.UnregisterFn != nil {
		return s.UnregisterFn(ctx, id)
	}
	return workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) Update(ctx context.Context, id string, opts workspacepkg.UpdateOptions) error {
	if s.UpdateFn != nil {
		return s.UpdateFn(ctx, id, opts)
	}
	return workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) List(ctx context.Context) ([]workspacepkg.Workspace, error) {
	if s.ListFn != nil {
		return s.ListFn(ctx)
	}
	return nil, nil
}

func (s StubWorkspaceService) Get(ctx context.Context, ref string) (workspacepkg.Workspace, error) {
	if s.GetFn != nil {
		return s.GetFn(ctx, ref)
	}
	return workspacepkg.Workspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) Resolve(ctx context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
	if s.ResolveFn != nil {
		return s.ResolveFn(ctx, ref)
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (s StubWorkspaceService) ResolveOrRegister(
	ctx context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	if s.ResolveOrRegisterFn != nil {
		return s.ResolveOrRegisterFn(ctx, path)
	}
	return workspacepkg.ResolvedWorkspace{}, ErrStubWorkspaceServiceNotImplemented
}

type StubSkillsRegistry struct {
	GetFn          func(name string) (*skills.Skill, bool)
	ListFn         func() []*skills.Skill
	ForWorkspaceFn func(ctx context.Context, resolved *workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
	LoadContentFn  func(ctx context.Context, skill *skills.Skill) (string, error)
	SetEnabledFn   func(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error
}

func (s StubSkillsRegistry) Get(name string) (*skills.Skill, bool) {
	if s.GetFn != nil {
		return s.GetFn(name)
	}
	return nil, false
}

func (s StubSkillsRegistry) List() []*skills.Skill {
	if s.ListFn != nil {
		return s.ListFn()
	}
	return nil
}

func (s StubSkillsRegistry) ForWorkspace(
	ctx context.Context,
	resolved *workspacepkg.ResolvedWorkspace,
) ([]*skills.Skill, error) {
	if s.ForWorkspaceFn != nil {
		return s.ForWorkspaceFn(ctx, resolved)
	}
	return nil, nil
}

func (s StubSkillsRegistry) LoadContent(ctx context.Context, skill *skills.Skill) (string, error) {
	if s.LoadContentFn != nil {
		return s.LoadContentFn(ctx, skill)
	}
	return "", nil
}

func (s StubSkillsRegistry) SetEnabled(name string, resolved *workspacepkg.ResolvedWorkspace, enabled bool) error {
	if s.SetEnabledFn != nil {
		return s.SetEnabledFn(name, resolved, enabled)
	}
	return nil
}

type SSERecord struct {
	ID    string
	Event string
	Data  []byte
}

func NewTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	return homePaths
}

func WriteAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(agent dir) error = %v", err)
	}
	if err := os.WriteFile(path, []byte(`---
name: `+name+`
provider: fake
permissions: approve-reads
---

You are `+name+`.
`), 0o600); err != nil {
		t.Fatalf("os.WriteFile(AGENT.md) error = %v", err)
	}
}

func NewSessionInfo(id string) *session.Info {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	return &session.Info{
		ID:          id,
		Name:        "demo",
		AgentName:   "coder",
		WorkspaceID: "ws-workspace",
		Workspace:   "/workspace",
		State:       session.StateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func NewSession(id string) *session.Session {
	info := NewSessionInfo(id)
	return &session.Session{
		ID:          info.ID,
		Name:        info.Name,
		AgentName:   info.AgentName,
		WorkspaceID: info.WorkspaceID,
		Workspace:   info.Workspace,
		State:       info.State,
		CreatedAt:   info.CreatedAt,
		UpdatedAt:   info.UpdatedAt,
	}
}

func PerformRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return PerformRequestWithHeaders(t, engine, method, path, body, nil)
}

func PerformRequestWithHeaders(
	t *testing.T,
	engine http.Handler,
	method, path string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(
		context.Background(),
		method,
		path,
		bytes.NewReader(body),
	)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func DecodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), dest); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v; body=%s", err, recorder.Body.String())
	}
}

func DecodeSSEData(t *testing.T, record SSERecord, dest any) {
	t.Helper()

	if err := json.Unmarshal(record.Data, dest); err != nil {
		t.Fatalf("json.Unmarshal(sse data) error = %v; data=%s", err, string(record.Data))
	}
}

func MustJSONBody(t *testing.T, value any) []byte {
	t.Helper()

	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return body
}

func ParseSSE(t *testing.T, body string) []SSERecord {
	t.Helper()

	scanner := bufio.NewScanner(strings.NewReader(body))
	records := make([]SSERecord, 0)
	current := SSERecord{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			records = append(records, current)
			current = SSERecord{}
			continue
		}

		switch {
		case strings.HasPrefix(line, "id: "):
			current.ID = strings.TrimPrefix(line, "id: ")
		case strings.HasPrefix(line, "event: "):
			current.Event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if len(current.Data) > 0 {
				current.Data = append(current.Data, '\n')
			}
			current.Data = append(current.Data, []byte(strings.TrimPrefix(line, "data: "))...)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner.Err() = %v", err)
	}
	if current.Event != "" || current.ID != "" || len(current.Data) > 0 {
		records = append(records, current)
	}

	return records
}

func DiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

var _ core.SessionManager = (*StubSessionManager)(nil)
var _ core.NetworkService = (*StubNetworkService)(nil)
var _ core.NetworkStore = (*StubNetworkStore)(nil)
var _ core.Observer = (*StubObserver)(nil)
var _ core.AutomationManager = (*StubAutomationManager)(nil)
var _ core.WorkspaceService = (*StubWorkspaceService)(nil)
