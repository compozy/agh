package daemon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	taskpkg "github.com/compozy/agh/internal/task"
)

type loopHookCoordinatorStore interface {
	AdvanceLoopRunProgress(ctx context.Context, loopRunID string, at time.Time) error
	EnqueueLoopCoordinatorWake(
		ctx context.Context,
		loopRunID string,
		idempotencyKey string,
		origin taskpkg.Origin,
		now time.Time,
	) (taskpkg.Run, bool, error)
	PromoteOldestQueuedLoopRun(
		ctx context.Context,
		workspaceID string,
		loopName string,
		origin taskpkg.Origin,
		now time.Time,
	) (taskpkg.Run, bool, error)
}

type loopNodeTerminalDispatcher interface {
	DispatchLoopNodeTerminal(
		context.Context,
		hookspkg.LoopNodeTerminalPayload,
	) (hookspkg.LoopNodeTerminalPayload, error)
}

type loopNativeHookObserver struct {
	store      loopHookCoordinatorStore
	dispatcher loopNodeTerminalDispatcher
	actor      taskpkg.ActorContext
	now        func() time.Time
}

var _ taskRunTerminalObserver = (*loopNativeHookObserver)(nil)
var _ loopTerminalObserver = (*loopNativeHookObserver)(nil)

func newLoopNativeHookObserver(
	store loopHookCoordinatorStore,
	dispatcher loopNodeTerminalDispatcher,
	now func() time.Time,
) (*loopNativeHookObserver, error) {
	if store == nil {
		return nil, fmt.Errorf("daemon: loop native hook observer requires coordinator store")
	}
	if dispatcher == nil {
		return nil, fmt.Errorf("daemon: loop native hook observer requires loop dispatcher")
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	actor, err := taskpkg.DeriveDaemonActorContext("loop-hook-observer", "loop.hooks")
	if err != nil {
		return nil, fmt.Errorf("daemon: derive loop hook actor: %w", err)
	}
	return &loopNativeHookObserver{
		store:      store,
		dispatcher: dispatcher,
		actor:      actor,
		now:        now,
	}, nil
}

func (o *loopNativeHookObserver) OnTaskRunTerminal(
	ctx context.Context,
	payload hookspkg.TaskRunLeasePayload,
) error {
	loopRunID := strings.TrimSpace(payload.LoopRunID)
	if loopRunID == "" || optionalRunKind(payload.RunKind) == taskpkg.RunKindCoordinator.String() {
		return nil
	}

	var errs []error
	loopPayload := hookspkg.LoopNodeTerminalPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookLoopNodeTerminal,
			Timestamp: payload.Timestamp,
		},
		LoopContext: hookspkg.LoopContext{
			LoopRunID:             loopRunID,
			WorkspaceID:           strings.TrimSpace(payload.WorkspaceID),
			TaskID:                strings.TrimSpace(payload.TaskID),
			RunID:                 strings.TrimSpace(payload.RunID),
			RunKind:               optionalRunKind(payload.RunKind),
			WorkflowID:            strings.TrimSpace(payload.WorkflowID),
			CoordinationChannelID: strings.TrimSpace(payload.CoordinationChannelID),
			NetworkChannel:        strings.TrimSpace(payload.NetworkChannel),
			AgentName:             strings.TrimSpace(payload.AgentName),
			SessionID:             strings.TrimSpace(payload.SessionID),
			ActorKind:             strings.TrimSpace(payload.ActorKind),
			ActorID:               strings.TrimSpace(payload.ActorID),
			OriginKind:            strings.TrimSpace(payload.OriginKind),
			OriginRef:             strings.TrimSpace(payload.OriginRef),
		},
		TaskStatus: strings.TrimSpace(payload.TaskStatus),
		RunStatus:  strings.TrimSpace(payload.RunStatus),
		Error:      strings.TrimSpace(payload.Error),
	}
	if _, err := o.dispatcher.DispatchLoopNodeTerminal(ctx, loopPayload); err != nil {
		errs = append(errs, fmt.Errorf("dispatch loop.node.terminal: %w", err))
	}
	if err := o.store.AdvanceLoopRunProgress(ctx, loopRunID, payload.Timestamp); err != nil {
		errs = append(errs, err)
	}
	if err := o.enqueueNodeTerminalWake(ctx, payload); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (o *loopNativeHookObserver) enqueueNodeTerminalWake(
	ctx context.Context,
	payload hookspkg.TaskRunLeasePayload,
) error {
	_, _, err := o.store.EnqueueLoopCoordinatorWake(
		ctx,
		strings.TrimSpace(payload.LoopRunID),
		loopNodeTerminalWakeKey(payload),
		o.actor.Origin,
		o.now().UTC(),
	)
	return normalizeLoopWakeError(err)
}

func (o *loopNativeHookObserver) OnLoopTerminal(
	ctx context.Context,
	payload hookspkg.LoopTerminalPayload,
) error {
	var errs []error
	if err := o.enqueueParentAwaitWake(ctx, payload); err != nil {
		errs = append(errs, err)
	}
	if err := o.promoteQueuedLoop(ctx, payload); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func (o *loopNativeHookObserver) enqueueParentAwaitWake(
	ctx context.Context,
	payload hookspkg.LoopTerminalPayload,
) error {
	parentLoopRunID := strings.TrimSpace(payload.ParentLoopRunID)
	if parentLoopRunID == "" {
		return nil
	}
	_, _, err := o.store.EnqueueLoopCoordinatorWake(
		ctx,
		parentLoopRunID,
		loopParentAwaitWakeKey(payload),
		o.actor.Origin,
		o.now().UTC(),
	)
	return normalizeLoopWakeError(err)
}

func (o *loopNativeHookObserver) promoteQueuedLoop(
	ctx context.Context,
	payload hookspkg.LoopTerminalPayload,
) error {
	workspaceID := strings.TrimSpace(payload.WorkspaceID)
	loopName := strings.TrimSpace(payload.LoopName)
	if workspaceID == "" || loopName == "" {
		return nil
	}
	_, _, err := o.store.PromoteOldestQueuedLoopRun(ctx, workspaceID, loopName, o.actor.Origin, o.now().UTC())
	return normalizeLoopWakeError(err)
}

func optionalRunKind(runKind *string) string {
	if runKind == nil {
		return ""
	}
	return strings.TrimSpace(*runKind)
}

func loopNodeTerminalWakeKey(payload hookspkg.TaskRunLeasePayload) string {
	return fmt.Sprintf(
		"loop.coordinator.node_terminal.%s.%s",
		strings.TrimSpace(payload.LoopRunID),
		strings.TrimSpace(payload.RunID),
	)
}

func loopParentAwaitWakeKey(payload hookspkg.LoopTerminalPayload) string {
	return fmt.Sprintf(
		"loop.coordinator.child_terminal.%s.%s",
		strings.TrimSpace(payload.ParentLoopRunID),
		strings.TrimSpace(payload.LoopRunID),
	)
}

func normalizeLoopWakeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, taskpkg.ErrInvalidStatusTransition), errors.Is(err, taskpkg.ErrConflict):
		return nil
	default:
		return err
	}
}
