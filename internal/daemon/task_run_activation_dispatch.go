package daemon

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	hookspkg "github.com/compozy/agh/internal/hooks"
	taskpkg "github.com/compozy/agh/internal/task"
)

type taskRunActivationDispatcher struct {
	store       taskRunActivationStore
	taskRoles   taskRunEnqueuedObserver
	loopActions taskRunEnqueuedObserver
	logger      *slog.Logger
}

type taskRunActivationStore interface {
	GetTaskRun(ctx context.Context, id string) (taskpkg.Run, error)
	ListTaskRunsByStatus(ctx context.Context, statuses []taskpkg.RunStatus) ([]taskpkg.Run, error)
}

func newTaskRunActivationDispatcher(
	store taskRunActivationStore,
	taskRoles taskRunEnqueuedObserver,
	loopActions taskRunEnqueuedObserver,
	logger *slog.Logger,
) (*taskRunActivationDispatcher, error) {
	if store == nil {
		return nil, errors.New("daemon: task run activation store is required")
	}
	if taskRoles == nil && loopActions == nil {
		return nil, errors.New("daemon: task run activation targets are required")
	}
	return &taskRunActivationDispatcher{
		store:       store,
		taskRoles:   taskRoles,
		loopActions: loopActions,
		logger:      logger,
	}, nil
}

func (d *taskRunActivationDispatcher) OnTaskRunEnqueued(
	ctx context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if d == nil || d.store == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	} else {
		ctx = context.WithoutCancel(ctx)
	}
	runID := strings.TrimSpace(payload.RunID)
	if runID == "" {
		return
	}
	run, err := d.store.GetTaskRun(ctx, runID)
	if err != nil {
		d.logError("daemon: load task run for activation dispatch", err, payload)
		return
	}
	d.dispatch(ctx, run, payload)
}

func (d *taskRunActivationDispatcher) Recover(ctx context.Context) {
	if d == nil || d.store == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runs, err := d.store.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		d.logError("daemon: list queued task runs for activation dispatch", err, hookspkg.TaskRunEnqueuedPayload{})
		return
	}
	for _, run := range runs {
		d.dispatch(ctx, run, activationDispatchPayload(run))
	}
}

func (d *taskRunActivationDispatcher) dispatch(
	ctx context.Context,
	run taskpkg.Run,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	target := d.targetForRun(run)
	if target == nil {
		return
	}
	target.OnTaskRunEnqueued(ctx, payload)
}

func (d *taskRunActivationDispatcher) targetForRun(run taskpkg.Run) taskRunEnqueuedObserver {
	switch run.RunKind.Normalize() {
	case taskpkg.RunKindWorker:
		if run.IsLoopWorker() {
			return d.loopActions
		}
		return d.taskRoles
	default:
		return nil
	}
}

func activationDispatchPayload(run taskpkg.Run) hookspkg.TaskRunEnqueuedPayload {
	kind := string(run.RunKind.Normalize())
	return hookspkg.TaskRunEnqueuedPayload{
		TaskRunContext: hookspkg.TaskRunContext{
			TaskID:                strings.TrimSpace(run.TaskID),
			RunID:                 strings.TrimSpace(run.ID),
			RunKind:               &kind,
			LoopRunID:             strings.TrimSpace(run.LoopRunID),
			CoordinationChannelID: strings.TrimSpace(run.CoordinationChannelID),
			RunStatus:             string(run.Status.Normalize()),
		},
	}
}

func (d *taskRunActivationDispatcher) logError(
	msg string,
	err error,
	payload hookspkg.TaskRunEnqueuedPayload,
) {
	if d == nil || d.logger == nil {
		return
	}
	d.logger.Error(
		msg,
		"error", err,
		"task_id", strings.TrimSpace(payload.TaskID),
		daemonLogRunIDKey, strings.TrimSpace(payload.RunID),
	)
}
