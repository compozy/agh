package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/testutil"
)

var fixedTestNow = time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

type stubClient struct {
	daemonStatusFn            func(context.Context) (DaemonStatus, error)
	listExtensionsFn          func(context.Context) ([]ExtensionRecord, error)
	installExtensionFn        func(context.Context, InstallExtensionRequest) (ExtensionRecord, error)
	enableExtensionFn         func(context.Context, string) (ExtensionRecord, error)
	disableExtensionFn        func(context.Context, string) (ExtensionRecord, error)
	extensionStatusFn         func(context.Context, string) (ExtensionRecord, error)
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
	listMemoryFn              func(context.Context, memory.Scope, string) ([]MemoryHeaderRecord, error)
	readMemoryFn              func(context.Context, string, memory.Scope, string) (MemoryReadRecord, error)
	writeMemoryFn             func(context.Context, string, MemoryWriteRequest) (MemoryMutationRecord, error)
	deleteMemoryFn            func(context.Context, string, memory.Scope, string) (MemoryMutationRecord, error)
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
}

func (s stubClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	if s.daemonStatusFn != nil {
		return s.daemonStatusFn(ctx)
	}
	return DaemonStatus{}, errors.New("unexpected DaemonStatus call")
}

func (s stubClient) ListExtensions(ctx context.Context) ([]ExtensionRecord, error) {
	if s.listExtensionsFn != nil {
		return s.listExtensionsFn(ctx)
	}
	return nil, errors.New("unexpected ListExtensions call")
}

func (s stubClient) InstallExtension(ctx context.Context, request InstallExtensionRequest) (ExtensionRecord, error) {
	if s.installExtensionFn != nil {
		return s.installExtensionFn(ctx, request)
	}
	return ExtensionRecord{}, errors.New("unexpected InstallExtension call")
}

func (s stubClient) EnableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.enableExtensionFn != nil {
		return s.enableExtensionFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected EnableExtension call")
}

func (s stubClient) DisableExtension(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.disableExtensionFn != nil {
		return s.disableExtensionFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected DisableExtension call")
}

func (s stubClient) ExtensionStatus(ctx context.Context, name string) (ExtensionRecord, error) {
	if s.extensionStatusFn != nil {
		return s.extensionStatusFn(ctx, name)
	}
	return ExtensionRecord{}, errors.New("unexpected ExtensionStatus call")
}

func (s stubClient) ListSessions(ctx context.Context, query SessionListQuery) ([]SessionRecord, error) {
	if s.listSessionsFn != nil {
		return s.listSessionsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListSessions call")
}

func (s stubClient) CreateSession(ctx context.Context, request CreateSessionRequest) (SessionRecord, error) {
	if s.createSessionFn != nil {
		return s.createSessionFn(ctx, request)
	}
	return SessionRecord{}, errors.New("unexpected CreateSession call")
}

func (s stubClient) GetSession(ctx context.Context, id string) (SessionRecord, error) {
	if s.getSessionFn != nil {
		return s.getSessionFn(ctx, id)
	}
	return SessionRecord{}, errors.New("unexpected GetSession call")
}

func (s stubClient) StopSession(ctx context.Context, id string) error {
	if s.stopSessionFn != nil {
		return s.stopSessionFn(ctx, id)
	}
	return errors.New("unexpected StopSession call")
}

func (s stubClient) ResumeSession(ctx context.Context, id string) (SessionRecord, error) {
	if s.resumeSessionFn != nil {
		return s.resumeSessionFn(ctx, id)
	}
	return SessionRecord{}, errors.New("unexpected ResumeSession call")
}

func (s stubClient) PromptSession(ctx context.Context, id string, message string) ([]AgentEventRecord, error) {
	if s.promptSessionFn != nil {
		return s.promptSessionFn(ctx, id, message)
	}
	return nil, errors.New("unexpected PromptSession call")
}

func (s stubClient) SessionEvents(ctx context.Context, id string, query SessionEventQuery) ([]SessionEventRecord, error) {
	if s.sessionEventsFn != nil {
		return s.sessionEventsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected SessionEvents call")
}

func (s stubClient) StreamSessionEvents(ctx context.Context, id string, query SessionEventQuery, lastEventID string, handler SSEHandler) error {
	if s.streamSessionFn != nil {
		return s.streamSessionFn(ctx, id, query, lastEventID, handler)
	}
	return errors.New("unexpected StreamSessionEvents call")
}

func (s stubClient) SessionHistory(ctx context.Context, id string, query SessionEventQuery) ([]TurnHistoryRecord, error) {
	if s.sessionHistoryFn != nil {
		return s.sessionHistoryFn(ctx, id, query)
	}
	return nil, errors.New("unexpected SessionHistory call")
}

func (s stubClient) CreateWorkspace(ctx context.Context, request WorkspaceCreateRequest) (WorkspaceRecord, error) {
	if s.createWorkspaceFn != nil {
		return s.createWorkspaceFn(ctx, request)
	}
	return WorkspaceRecord{}, errors.New("unexpected CreateWorkspace call")
}

func (s stubClient) ListWorkspaces(ctx context.Context) ([]WorkspaceRecord, error) {
	if s.listWorkspacesFn != nil {
		return s.listWorkspacesFn(ctx)
	}
	return nil, errors.New("unexpected ListWorkspaces call")
}

func (s stubClient) GetWorkspace(ctx context.Context, ref string) (WorkspaceDetailRecord, error) {
	if s.getWorkspaceFn != nil {
		return s.getWorkspaceFn(ctx, ref)
	}
	return WorkspaceDetailRecord{}, errors.New("unexpected GetWorkspace call")
}

func (s stubClient) UpdateWorkspace(ctx context.Context, ref string, request WorkspaceUpdateRequest) (WorkspaceRecord, error) {
	if s.updateWorkspaceFn != nil {
		return s.updateWorkspaceFn(ctx, ref, request)
	}
	return WorkspaceRecord{}, errors.New("unexpected UpdateWorkspace call")
}

func (s stubClient) DeleteWorkspace(ctx context.Context, ref string) error {
	if s.deleteWorkspaceFn != nil {
		return s.deleteWorkspaceFn(ctx, ref)
	}
	return errors.New("unexpected DeleteWorkspace call")
}

func (s stubClient) ListAgents(ctx context.Context) ([]AgentRecord, error) {
	if s.listAgentsFn != nil {
		return s.listAgentsFn(ctx)
	}
	return nil, errors.New("unexpected ListAgents call")
}

func (s stubClient) GetAgent(ctx context.Context, name string) (AgentRecord, error) {
	if s.getAgentFn != nil {
		return s.getAgentFn(ctx, name)
	}
	return AgentRecord{}, errors.New("unexpected GetAgent call")
}

func (s stubClient) HookCatalog(ctx context.Context, query HookCatalogQuery) ([]HookCatalogRecord, error) {
	if s.hookCatalogFn != nil {
		return s.hookCatalogFn(ctx, query)
	}
	return nil, errors.New("unexpected HookCatalog call")
}

func (s stubClient) HookRuns(ctx context.Context, query HookRunsQuery) ([]HookRunRecord, error) {
	if s.hookRunsFn != nil {
		return s.hookRunsFn(ctx, query)
	}
	return nil, errors.New("unexpected HookRuns call")
}

func (s stubClient) HookEvents(ctx context.Context, query HookEventsQuery) ([]HookEventRecord, error) {
	if s.hookEventsFn != nil {
		return s.hookEventsFn(ctx, query)
	}
	return nil, errors.New("unexpected HookEvents call")
}

func (s stubClient) ObserveEvents(ctx context.Context, query ObserveEventQuery) ([]ObserveEventRecord, error) {
	if s.observeEventsFn != nil {
		return s.observeEventsFn(ctx, query)
	}
	return nil, errors.New("unexpected ObserveEvents call")
}

func (s stubClient) StreamObserveEvents(ctx context.Context, query ObserveEventQuery, lastEventID string, handler SSEHandler) error {
	if s.streamObserveEventsFn != nil {
		return s.streamObserveEventsFn(ctx, query, lastEventID, handler)
	}
	return errors.New("unexpected StreamObserveEvents call")
}

func (s stubClient) ObserveHealth(ctx context.Context) (HealthStatus, error) {
	if s.observeHealthFn != nil {
		return s.observeHealthFn(ctx)
	}
	return HealthStatus{}, errors.New("unexpected ObserveHealth call")
}

func (s stubClient) ListMemory(ctx context.Context, scope memory.Scope, workspace string) ([]MemoryHeaderRecord, error) {
	if s.listMemoryFn != nil {
		return s.listMemoryFn(ctx, scope, workspace)
	}
	return nil, errors.New("unexpected ListMemory call")
}

func (s stubClient) ReadMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryReadRecord, error) {
	if s.readMemoryFn != nil {
		return s.readMemoryFn(ctx, filename, scope, workspace)
	}
	return MemoryReadRecord{}, errors.New("unexpected ReadMemory call")
}

func (s stubClient) WriteMemory(ctx context.Context, filename string, request MemoryWriteRequest) (MemoryMutationRecord, error) {
	if s.writeMemoryFn != nil {
		return s.writeMemoryFn(ctx, filename, request)
	}
	return MemoryMutationRecord{}, errors.New("unexpected WriteMemory call")
}

func (s stubClient) DeleteMemory(ctx context.Context, filename string, scope memory.Scope, workspace string) (MemoryMutationRecord, error) {
	if s.deleteMemoryFn != nil {
		return s.deleteMemoryFn(ctx, filename, scope, workspace)
	}
	return MemoryMutationRecord{}, errors.New("unexpected DeleteMemory call")
}

func (s stubClient) ConsolidateMemory(ctx context.Context, workspace string) (MemoryConsolidateRecord, error) {
	if s.consolidateMemoryFn != nil {
		return s.consolidateMemoryFn(ctx, workspace)
	}
	return MemoryConsolidateRecord{}, errors.New("unexpected ConsolidateMemory call")
}

func (s stubClient) ListAutomationJobs(ctx context.Context, query AutomationJobQuery) ([]JobRecord, error) {
	if s.listAutomationJobsFn != nil {
		return s.listAutomationJobsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationJobs call")
}

func (s stubClient) CreateAutomationJob(ctx context.Context, request AutomationJobCreateRequest) (JobRecord, error) {
	if s.createAutomationJobFn != nil {
		return s.createAutomationJobFn(ctx, request)
	}
	return JobRecord{}, errors.New("unexpected CreateAutomationJob call")
}

func (s stubClient) GetAutomationJob(ctx context.Context, id string) (JobRecord, error) {
	if s.getAutomationJobFn != nil {
		return s.getAutomationJobFn(ctx, id)
	}
	return JobRecord{}, errors.New("unexpected GetAutomationJob call")
}

func (s stubClient) UpdateAutomationJob(ctx context.Context, id string, request AutomationJobUpdateRequest) (JobRecord, error) {
	if s.updateAutomationJobFn != nil {
		return s.updateAutomationJobFn(ctx, id, request)
	}
	return JobRecord{}, errors.New("unexpected UpdateAutomationJob call")
}

func (s stubClient) DeleteAutomationJob(ctx context.Context, id string) error {
	if s.deleteAutomationJobFn != nil {
		return s.deleteAutomationJobFn(ctx, id)
	}
	return errors.New("unexpected DeleteAutomationJob call")
}

func (s stubClient) TriggerAutomationJob(ctx context.Context, id string) (RunRecord, error) {
	if s.triggerAutomationJobFn != nil {
		return s.triggerAutomationJobFn(ctx, id)
	}
	return RunRecord{}, errors.New("unexpected TriggerAutomationJob call")
}

func (s stubClient) AutomationJobRuns(ctx context.Context, id string, query AutomationRunQuery) ([]RunRecord, error) {
	if s.automationJobRunsFn != nil {
		return s.automationJobRunsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected AutomationJobRuns call")
}

func (s stubClient) ListAutomationTriggers(ctx context.Context, query AutomationTriggerQuery) ([]TriggerRecord, error) {
	if s.listAutomationTriggersFn != nil {
		return s.listAutomationTriggersFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationTriggers call")
}

func (s stubClient) CreateAutomationTrigger(ctx context.Context, request AutomationTriggerCreateRequest) (TriggerRecord, error) {
	if s.createAutomationTriggerFn != nil {
		return s.createAutomationTriggerFn(ctx, request)
	}
	return TriggerRecord{}, errors.New("unexpected CreateAutomationTrigger call")
}

func (s stubClient) GetAutomationTrigger(ctx context.Context, id string) (TriggerRecord, error) {
	if s.getAutomationTriggerFn != nil {
		return s.getAutomationTriggerFn(ctx, id)
	}
	return TriggerRecord{}, errors.New("unexpected GetAutomationTrigger call")
}

func (s stubClient) UpdateAutomationTrigger(ctx context.Context, id string, request AutomationTriggerUpdateRequest) (TriggerRecord, error) {
	if s.updateAutomationTriggerFn != nil {
		return s.updateAutomationTriggerFn(ctx, id, request)
	}
	return TriggerRecord{}, errors.New("unexpected UpdateAutomationTrigger call")
}

func (s stubClient) DeleteAutomationTrigger(ctx context.Context, id string) error {
	if s.deleteAutomationTriggerFn != nil {
		return s.deleteAutomationTriggerFn(ctx, id)
	}
	return errors.New("unexpected DeleteAutomationTrigger call")
}

func (s stubClient) AutomationTriggerRuns(ctx context.Context, id string, query AutomationRunQuery) ([]RunRecord, error) {
	if s.automationTriggerRunsFn != nil {
		return s.automationTriggerRunsFn(ctx, id, query)
	}
	return nil, errors.New("unexpected AutomationTriggerRuns call")
}

func (s stubClient) ListAutomationRuns(ctx context.Context, query AutomationRunQuery) ([]RunRecord, error) {
	if s.listAutomationRunsFn != nil {
		return s.listAutomationRunsFn(ctx, query)
	}
	return nil, errors.New("unexpected ListAutomationRuns call")
}

func (s stubClient) GetAutomationRun(ctx context.Context, id string) (RunRecord, error) {
	if s.getAutomationRunFn != nil {
		return s.getAutomationRunFn(ctx, id)
	}
	return RunRecord{}, errors.New("unexpected GetAutomationRun call")
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

func executeRootCommandWithExit(t *testing.T, deps commandDeps, args ...string) (int, string, string) {
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
