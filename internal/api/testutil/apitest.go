package testutil

import (
	"bufio"
	"bytes"
	"context"
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
	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var ErrStubWorkspaceServiceNotImplemented = errors.New("stub workspace service method not implemented")

type StubSessionManager struct {
	CreateFn        func(context.Context, session.CreateOpts) (*session.Session, error)
	ListFn          func() []*session.SessionInfo
	ListAllFn       func(context.Context) ([]*session.SessionInfo, error)
	StatusFn        func(context.Context, string) (*session.SessionInfo, error)
	EventsFn        func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error)
	HistoryFn       func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error)
	TranscriptFn    func(context.Context, string) ([]transcript.Message, error)
	StopFn          func(context.Context, string) error
	StopWithCauseFn func(context.Context, string, session.StopCause, string) error
	ResumeFn        func(context.Context, string) (*session.Session, error)
	PromptFn        func(context.Context, string, string) (<-chan acp.AgentEvent, error)
	ApproveFn       func(context.Context, string, acp.ApproveRequest) error
}

func (s StubSessionManager) Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error) {
	if s.CreateFn != nil {
		return s.CreateFn(ctx, opts)
	}
	return nil, nil
}

func (s StubSessionManager) List() []*session.SessionInfo {
	if s.ListFn != nil {
		return s.ListFn()
	}
	if s.ListAllFn != nil {
		infos, err := s.ListAllFn(context.Background())
		if err != nil {
			return []*session.SessionInfo{}
		}
		return infos
	}
	return nil
}

func (s StubSessionManager) ListAll(ctx context.Context) ([]*session.SessionInfo, error) {
	if s.ListAllFn != nil {
		return s.ListAllFn(ctx)
	}
	return nil, nil
}

func (s StubSessionManager) Status(ctx context.Context, id string) (*session.SessionInfo, error) {
	if s.StatusFn != nil {
		return s.StatusFn(ctx, id)
	}
	return nil, session.ErrSessionNotFound
}

func (s StubSessionManager) Events(ctx context.Context, id string, query store.EventQuery) ([]store.SessionEvent, error) {
	if s.EventsFn != nil {
		return s.EventsFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) History(ctx context.Context, id string, query store.EventQuery) ([]store.TurnHistory, error) {
	if s.HistoryFn != nil {
		return s.HistoryFn(ctx, id, query)
	}
	return nil, nil
}

func (s StubSessionManager) Transcript(ctx context.Context, id string) ([]transcript.Message, error) {
	if s.TranscriptFn != nil {
		return s.TranscriptFn(ctx, id)
	}
	return nil, nil
}

func (s StubSessionManager) Stop(ctx context.Context, id string) error {
	if s.StopFn != nil {
		return s.StopFn(ctx, id)
	}
	return nil
}

func (s StubSessionManager) StopWithCause(ctx context.Context, id string, cause session.StopCause, detail string) error {
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

func (s StubSessionManager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if s.PromptFn != nil {
		return s.PromptFn(ctx, id, msg)
	}
	ch := make(chan acp.AgentEvent)
	close(ch)
	return ch, nil
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
	QueryChannelHealthFn func(context.Context) ([]observe.ChannelInstanceHealth, error)
	HealthFn             func(context.Context) (observe.Health, error)
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

func (s StubAutomationManager) ListJobs(ctx context.Context, query automationpkg.JobListQuery) ([]automationpkg.Job, error) {
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

func (s StubAutomationManager) ListTriggers(ctx context.Context, query automationpkg.TriggerListQuery) ([]automationpkg.Trigger, error) {
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

func (s StubAutomationManager) CreateTrigger(ctx context.Context, trigger automationpkg.Trigger, secret string) (automationpkg.Trigger, error) {
	if s.CreateTriggerFn != nil {
		return s.CreateTriggerFn(ctx, trigger, secret)
	}
	return trigger, nil
}

func (s StubAutomationManager) UpdateTrigger(ctx context.Context, trigger automationpkg.Trigger, secret *string) (automationpkg.Trigger, error) {
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

func (s StubAutomationManager) ListRuns(ctx context.Context, query automationpkg.RunQuery) ([]automationpkg.Run, error) {
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

func (s StubAutomationManager) SetTriggerEnabled(ctx context.Context, id string, enabled bool) (automationpkg.Trigger, error) {
	if s.SetTriggerEnabledFn != nil {
		return s.SetTriggerEnabledFn(ctx, id, enabled)
	}
	return automationpkg.Trigger{}, nil
}

func (s StubAutomationManager) HandleWebhook(ctx context.Context, request automationpkg.WebhookRequest) (automationpkg.TriggerResult, error) {
	if s.HandleWebhookFn != nil {
		return s.HandleWebhookFn(ctx, request)
	}
	return automationpkg.TriggerResult{}, nil
}

func (s StubObserver) QueryEvents(ctx context.Context, query store.EventSummaryQuery) ([]store.EventSummary, error) {
	if s.QueryEventsFn != nil {
		return s.QueryEventsFn(ctx, query)
	}
	return nil, nil
}

func (s StubObserver) Health(ctx context.Context) (observe.Health, error) {
	if s.HealthFn != nil {
		return s.HealthFn(ctx)
	}
	return observe.Health{Status: "ok"}, nil
}

func (s StubObserver) QueryChannelHealth(ctx context.Context) ([]observe.ChannelInstanceHealth, error) {
	if s.QueryChannelHealthFn != nil {
		return s.QueryChannelHealthFn(ctx)
	}
	return nil, nil
}

func (s StubObserver) QueryHookCatalog(ctx context.Context, filter hookspkg.CatalogFilter) ([]hookspkg.CatalogEntry, error) {
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

func (s StubObserver) QueryHookEvents(ctx context.Context, filter hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error) {
	if s.QueryHookEventsFn != nil {
		return s.QueryHookEventsFn(ctx, filter)
	}
	return nil, nil
}

type StubChannelService struct {
	CreateInstanceFn        func(context.Context, channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error)
	GetInstanceFn           func(context.Context, string) (*channelspkg.ChannelInstance, error)
	ListInstancesFn         func(context.Context) ([]channelspkg.ChannelInstance, error)
	UpdateInstanceFn        func(context.Context, channelspkg.UpdateInstanceRequest) (*channelspkg.ChannelInstance, error)
	UpdateInstanceStateFn   func(context.Context, channelspkg.UpdateInstanceStateRequest) (*channelspkg.ChannelInstance, error)
	BuildRoutingKeyFn       func(context.Context, channelspkg.RoutingKey) (channelspkg.RoutingKey, error)
	ResolveRouteFn          func(context.Context, channelspkg.RoutingKey) (*channelspkg.ChannelRoute, error)
	ResolveOrCreateRouteFn  func(context.Context, channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, bool, error)
	UpsertRouteFn           func(context.Context, channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, error)
	ListRoutesFn            func(context.Context, string) ([]channelspkg.ChannelRoute, error)
	ResolveDeliveryTargetFn func(context.Context, channelspkg.ResolveDeliveryTargetRequest) (*channelspkg.DeliveryTarget, error)
	StartInstanceFn         func(context.Context, string) (*channelspkg.ChannelInstance, error)
	StopInstanceFn          func(context.Context, string) (*channelspkg.ChannelInstance, error)
	RestartInstanceFn       func(context.Context, string) (*channelspkg.ChannelInstance, error)
}

var _ core.ChannelService = (*StubChannelService)(nil)

func (s StubChannelService) CreateInstance(ctx context.Context, req channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error) {
	if s.CreateInstanceFn != nil {
		return s.CreateInstanceFn(ctx, req)
	}
	return nil, nil
}

func (s StubChannelService) GetInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if s.GetInstanceFn != nil {
		return s.GetInstanceFn(ctx, id)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) ListInstances(ctx context.Context) ([]channelspkg.ChannelInstance, error) {
	if s.ListInstancesFn != nil {
		return s.ListInstancesFn(ctx)
	}
	return nil, nil
}

func (s StubChannelService) UpdateInstance(ctx context.Context, req channelspkg.UpdateInstanceRequest) (*channelspkg.ChannelInstance, error) {
	if s.UpdateInstanceFn != nil {
		return s.UpdateInstanceFn(ctx, req)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) UpdateInstanceState(ctx context.Context, req channelspkg.UpdateInstanceStateRequest) (*channelspkg.ChannelInstance, error) {
	if s.UpdateInstanceStateFn != nil {
		return s.UpdateInstanceStateFn(ctx, req)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) BuildRoutingKey(ctx context.Context, key channelspkg.RoutingKey) (channelspkg.RoutingKey, error) {
	if s.BuildRoutingKeyFn != nil {
		return s.BuildRoutingKeyFn(ctx, key)
	}
	return channelspkg.RoutingKey{}, nil
}

func (s StubChannelService) ResolveRoute(ctx context.Context, key channelspkg.RoutingKey) (*channelspkg.ChannelRoute, error) {
	if s.ResolveRouteFn != nil {
		return s.ResolveRouteFn(ctx, key)
	}
	return nil, channelspkg.ErrChannelRouteNotFound
}

func (s StubChannelService) ResolveOrCreateRoute(ctx context.Context, route channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, bool, error) {
	if s.ResolveOrCreateRouteFn != nil {
		return s.ResolveOrCreateRouteFn(ctx, route)
	}
	return nil, false, channelspkg.ErrChannelRouteNotFound
}

func (s StubChannelService) UpsertRoute(ctx context.Context, route channelspkg.ChannelRoute) (*channelspkg.ChannelRoute, error) {
	if s.UpsertRouteFn != nil {
		return s.UpsertRouteFn(ctx, route)
	}
	return nil, channelspkg.ErrChannelRouteNotFound
}

func (s StubChannelService) ListRoutes(ctx context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error) {
	if s.ListRoutesFn != nil {
		return s.ListRoutesFn(ctx, channelInstanceID)
	}
	return nil, nil
}

func (s StubChannelService) ResolveDeliveryTarget(ctx context.Context, req channelspkg.ResolveDeliveryTargetRequest) (*channelspkg.DeliveryTarget, error) {
	if s.ResolveDeliveryTargetFn != nil {
		return s.ResolveDeliveryTargetFn(ctx, req)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) StartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if s.StartInstanceFn != nil {
		return s.StartInstanceFn(ctx, id)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) StopInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if s.StopInstanceFn != nil {
		return s.StopInstanceFn(ctx, id)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
}

func (s StubChannelService) RestartInstance(ctx context.Context, id string) (*channelspkg.ChannelInstance, error) {
	if s.RestartInstanceFn != nil {
		return s.RestartInstanceFn(ctx, id)
	}
	return nil, channelspkg.ErrChannelInstanceNotFound
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

func (s StubWorkspaceService) Register(ctx context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
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

func (s StubWorkspaceService) ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
	if s.ResolveOrRegisterFn != nil {
		return s.ResolveOrRegisterFn(ctx, path)
	}
	return workspacepkg.ResolvedWorkspace{}, ErrStubWorkspaceServiceNotImplemented
}

type StubSkillsRegistry struct {
	GetFn          func(name string) (*skills.Skill, bool)
	ListFn         func() []*skills.Skill
	ForWorkspaceFn func(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error)
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

func (s StubSkillsRegistry) ForWorkspace(ctx context.Context, resolved workspacepkg.ResolvedWorkspace) ([]*skills.Skill, error) {
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
`), 0o644); err != nil {
		t.Fatalf("os.WriteFile(AGENT.md) error = %v", err)
	}
}

func NewSessionInfo(id string) *session.SessionInfo {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	return &session.SessionInfo{
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

func PerformRequestWithHeaders(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
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
var _ core.Observer = (*StubObserver)(nil)
var _ core.AutomationManager = (*StubAutomationManager)(nil)
var _ core.WorkspaceService = (*StubWorkspaceService)(nil)
