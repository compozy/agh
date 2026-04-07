package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pedronauck/agh/internal/testutil"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

var fixedTestNow = time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

type stubClient struct {
	daemonStatusFn        func(context.Context) (DaemonStatus, error)
	listSessionsFn        func(context.Context, SessionListQuery) ([]SessionRecord, error)
	createSessionFn       func(context.Context, CreateSessionRequest) (SessionRecord, error)
	getSessionFn          func(context.Context, string) (SessionRecord, error)
	stopSessionFn         func(context.Context, string) error
	resumeSessionFn       func(context.Context, string) (SessionRecord, error)
	promptSessionFn       func(context.Context, string, string) ([]AgentEventRecord, error)
	sessionEventsFn       func(context.Context, string, SessionEventQuery) ([]SessionEventRecord, error)
	streamSessionFn       func(context.Context, string, SessionEventQuery, string, SSEHandler) error
	sessionHistoryFn      func(context.Context, string, SessionEventQuery) ([]TurnHistoryRecord, error)
	createWorkspaceFn     func(context.Context, WorkspaceCreateRequest) (WorkspaceRecord, error)
	listWorkspacesFn      func(context.Context) ([]WorkspaceRecord, error)
	getWorkspaceFn        func(context.Context, string) (WorkspaceDetailRecord, error)
	updateWorkspaceFn     func(context.Context, string, WorkspaceUpdateRequest) (WorkspaceRecord, error)
	deleteWorkspaceFn     func(context.Context, string) error
	listAgentsFn          func(context.Context) ([]AgentRecord, error)
	getAgentFn            func(context.Context, string) (AgentRecord, error)
	observeEventsFn       func(context.Context, ObserveEventQuery) ([]ObserveEventRecord, error)
	streamObserveEventsFn func(context.Context, ObserveEventQuery, string, SSEHandler) error
	observeHealthFn       func(context.Context) (HealthStatus, error)
	listMemoryFn          func(context.Context, memory.Scope, string) ([]MemoryHeaderRecord, error)
	readMemoryFn          func(context.Context, string, memory.Scope, string) (MemoryReadRecord, error)
	writeMemoryFn         func(context.Context, string, MemoryWriteRequest) (MemoryMutationRecord, error)
	deleteMemoryFn        func(context.Context, string, memory.Scope, string) (MemoryMutationRecord, error)
	consolidateMemoryFn   func(context.Context, string) (MemoryConsolidateRecord, error)
}

func (s stubClient) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	if s.daemonStatusFn != nil {
		return s.daemonStatusFn(ctx)
	}
	return DaemonStatus{}, errors.New("unexpected DaemonStatus call")
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
