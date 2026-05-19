package daemon

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/session"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

const (
	taskRoleRuntimeTaskIDKey      = "task_id"
	taskRoleRuntimeWorkspaceIDKey = "workspace_id"
)

const taskRoleActivationReasonRunEnqueued = "task_run_enqueued"
const taskRoleActivationReasonRecovery = "recovery"

type taskRoleStore interface {
	GetTask(ctx context.Context, id string) (taskpkg.Task, error)
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []taskpkg.RunStatus) ([]taskpkg.Run, error)
}

type taskRoleSessionManager interface {
	Create(ctx context.Context, opts session.CreateOpts) (*session.Session, error)
	ListAll(ctx context.Context) ([]*session.Info, error)
}

type taskRoleRuntime struct {
	mu                  sync.Mutex
	store               taskRoleStore
	sessions            taskRoleSessionManager
	globalWorkspacePath string
	logger              *slog.Logger
}

type taskRoleActivation struct {
	TaskID        string
	RunID         string
	Scope         taskpkg.Scope
	WorkspaceID   string
	WorkspacePath string
	AgentName     string
	Channel       string
	Title         string
}

var _ taskRunEnqueuedObserver = (*taskRoleRuntime)(nil)

func newTaskRoleRuntime(
	store taskRoleStore,
	sessions taskRoleSessionManager,
	globalWorkspacePath string,
	logger *slog.Logger,
) (*taskRoleRuntime, error) {
	if store == nil {
		return nil, errors.New("daemon: task role runtime requires task store")
	}
	if sessions == nil {
		return nil, errors.New("daemon: task role runtime requires session manager")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &taskRoleRuntime{
		store:               store,
		sessions:            sessions,
		globalWorkspacePath: strings.TrimSpace(globalWorkspacePath),
		logger:              logger,
	}, nil
}

func (r *taskRoleRuntime) OnTaskRunEnqueued(ctx context.Context, payload hookspkg.TaskRunEnqueuedPayload) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = context.WithoutCancel(ctx)
	}
	runID := strings.TrimSpace(payload.RunID)
	if runID == "" {
		r.logTaskRoleError("daemon: task role enqueue payload missing run id", nil, payload)
		return
	}
	run, err := r.store.GetTaskRun(ctx, runID)
	if err != nil {
		r.logTaskRoleError("daemon: load task run for role activation", err, payload)
		return
	}
	taskRecord, err := r.store.GetTask(ctx, run.TaskID)
	if err != nil {
		r.logTaskRoleError("daemon: load task for role activation", err, payload)
		return
	}
	if err := r.activateRun(ctx, taskRecord, run, taskRoleActivationReasonRunEnqueued); err != nil {
		r.logTaskRoleError("daemon: activate task role session from enqueue", err, payload)
	}
}

func (r *taskRoleRuntime) Recover(ctx context.Context) {
	if r == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runs, err := r.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		r.logTaskRoleError("daemon: list queued task runs for role recovery", err, hookspkg.TaskRunEnqueuedPayload{})
		return
	}
	for _, run := range runs {
		taskRecord, err := r.store.GetTask(ctx, run.TaskID)
		if err != nil {
			r.logTaskRoleError("daemon: load task for role recovery", err, hookspkg.TaskRunEnqueuedPayload{
				TaskRunContext: hookspkg.TaskRunContext{RunID: run.ID, TaskID: run.TaskID},
			})
			continue
		}
		if err := r.activateRun(ctx, taskRecord, run, taskRoleActivationReasonRecovery); err != nil {
			r.logTaskRoleError(
				"daemon: recover task role session for queued run",
				err,
				hookspkg.TaskRunEnqueuedPayload{
					TaskRunContext: hookspkg.TaskRunContext{
						RunID:                 run.ID,
						TaskID:                run.TaskID,
						WorkspaceID:           taskRecord.WorkspaceID,
						CoordinationChannelID: run.CoordinationChannelID,
					},
				},
			)
		}
	}
}

func (r *taskRoleRuntime) activateRun(
	ctx context.Context,
	taskRecord taskpkg.Task,
	run taskpkg.Run,
	reason string,
) error {
	if ctx == nil {
		return errors.New("daemon: task role activation context is required")
	}
	activation, ok, err := r.activationForRun(taskRecord, run)
	if err != nil || !ok {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	existing, err := r.activeRoleSession(ctx, activation)
	if err != nil {
		return err
	}
	if existing != nil {
		r.logger.Info(
			"daemon: task role session already active",
			taskRoleRuntimeTaskIDKey, activation.TaskID,
			"run_id", activation.RunID,
			"agent_name", activation.AgentName,
			"channel", activation.Channel,
			"reason", reason,
		)
		return nil
	}

	info, err := r.startRoleSession(ctx, activation)
	if err != nil {
		return err
	}
	r.logger.Info(
		"daemon: task role session started",
		"session_id", info.ID,
		taskRoleRuntimeTaskIDKey, activation.TaskID,
		"run_id", activation.RunID,
		"agent_name", activation.AgentName,
		"channel", activation.Channel,
		"reason", reason,
	)
	return nil
}

func (r *taskRoleRuntime) activationForRun(
	taskRecord taskpkg.Task,
	run taskpkg.Run,
) (taskRoleActivation, bool, error) {
	if run.Status.Normalize() != taskpkg.TaskRunStatusQueued {
		return taskRoleActivation{}, false, nil
	}
	switch taskRecord.Status.Normalize() {
	case taskpkg.TaskStatusDraft, taskpkg.TaskStatusBlocked, taskpkg.TaskStatusCanceled:
		return taskRoleActivation{}, false, nil
	default:
	}
	if taskRecord.Owner == nil || taskRecord.Owner.IsZero() {
		return taskRoleActivation{}, false, nil
	}
	owner := *taskRecord.Owner
	if owner.Kind.Normalize() != taskpkg.OwnerKindPool {
		return taskRoleActivation{}, false, nil
	}
	agentName := strings.TrimSpace(owner.Ref)
	if agentName == "" {
		return taskRoleActivation{}, false, nil
	}

	activation := taskRoleActivation{
		TaskID:      strings.TrimSpace(taskRecord.ID),
		RunID:       strings.TrimSpace(run.ID),
		Scope:       taskRecord.Scope.Normalize(),
		WorkspaceID: strings.TrimSpace(taskRecord.WorkspaceID),
		AgentName:   agentName,
		Channel:     taskRunSessionChannel(run),
		Title:       strings.TrimSpace(taskRecord.Title),
	}
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		if activation.WorkspaceID == "" {
			return taskRoleActivation{}, false, fmt.Errorf(
				"%w: workspace-scoped task %q has no workspace id",
				taskpkg.ErrValidation,
				taskRecord.ID,
			)
		}
	case taskpkg.ScopeGlobal:
		if r.globalWorkspacePath == "" {
			return taskRoleActivation{}, false, errors.New("daemon: task role global workspace path is required")
		}
		activation.WorkspacePath = r.globalWorkspacePath
	default:
		return taskRoleActivation{}, false, fmt.Errorf(
			"%w: unsupported task scope %q for role activation",
			taskpkg.ErrValidation,
			taskRecord.Scope,
		)
	}
	return activation, true, nil
}

func (r *taskRoleRuntime) activeRoleSession(
	ctx context.Context,
	activation taskRoleActivation,
) (*session.Info, error) {
	infos, err := r.sessions.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("daemon: list sessions for task role activation: %w", err)
	}
	for _, info := range infos {
		if taskRoleSessionMatches(info, activation) {
			return info, nil
		}
	}
	return nil, nil
}

func (r *taskRoleRuntime) startRoleSession(
	ctx context.Context,
	activation taskRoleActivation,
) (*session.Info, error) {
	opts := session.CreateOpts{
		AgentName:     activation.AgentName,
		Provider:      "",
		Name:          taskRoleSessionName(activation),
		Channel:       activation.Channel,
		PromptOverlay: taskRolePromptOverlay(activation),
		Type:          session.SessionTypeSystem,
	}
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		opts.Workspace = activation.WorkspaceID
	case taskpkg.ScopeGlobal:
		opts.WorkspacePath = activation.WorkspacePath
	default:
		return nil, fmt.Errorf(
			"%w: unsupported task scope %q for role session start",
			taskpkg.ErrValidation,
			activation.Scope,
		)
	}

	created, err := r.sessions.Create(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("daemon: create task role session: %w", err)
	}
	if created == nil {
		return nil, errors.New("daemon: task role session create returned nil")
	}
	info := created.Info()
	if info == nil {
		return nil, errors.New("daemon: task role session create returned nil info")
	}
	return info, nil
}

func taskRoleSessionMatches(info *session.Info, activation taskRoleActivation) bool {
	if info == nil {
		return false
	}
	if !taskRoleSessionStateReusable(info.State) {
		return false
	}
	if strings.TrimSpace(info.AgentName) != activation.AgentName {
		return false
	}
	if strings.TrimSpace(info.Channel) != activation.Channel {
		return false
	}
	switch activation.Scope {
	case taskpkg.ScopeWorkspace:
		return strings.TrimSpace(info.WorkspaceID) == activation.WorkspaceID
	case taskpkg.ScopeGlobal:
		return strings.TrimSpace(info.WorkspaceID) == "" &&
			strings.TrimSpace(info.Workspace) == activation.WorkspacePath
	default:
		return false
	}
}

func taskRoleSessionStateReusable(state session.State) bool {
	switch state {
	case session.StateStarting, session.StateActive:
		return true
	default:
		return false
	}
}

func taskRoleSessionName(activation taskRoleActivation) string {
	return fmt.Sprintf("task-role:%s:%s", activation.AgentName, firstNonEmpty(activation.Channel, "default"))
}

func taskRolePromptOverlay(activation taskRoleActivation) string {
	title := firstNonEmpty(activation.Title, activation.TaskID)
	channel := firstNonEmpty(activation.Channel, "default")
	return fmt.Sprintf(`A queued AGH task run is assigned to this agent.

Task: %s
Run: %s
Coordination channel: %s

Use `+"`agh task next --wait -o json`"+` to claim work for this session before changing files. Complete or fail the claimed run through the AGH task lease commands from this same session. Do not use `+"`agh task run claim`"+` for autonomous work.`,
		title,
		activation.RunID,
		channel,
	)
}

func (r *taskRoleRuntime) logTaskRoleError(
	message string,
	err error,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if r == nil {
		return
	}
	args := []any{
		taskRoleRuntimeTaskIDKey, strings.TrimSpace(payload.TaskID),
		"run_id", strings.TrimSpace(payload.RunID),
		taskRoleRuntimeWorkspaceIDKey, strings.TrimSpace(payload.WorkspaceID),
		"coordination_channel_id", strings.TrimSpace(payload.CoordinationChannelID),
	}
	if err != nil {
		args = append(args, "error", err)
	}
	r.logger.Warn(message, args...)
}
