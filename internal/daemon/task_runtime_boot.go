package daemon

import (
	"context"
	"fmt"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	taskpkg "github.com/compozy/agh/internal/task"
)

func (d *Daemon) bootTasks(ctx context.Context, state *bootState) error {
	if state == nil || state.registry == nil || state.sessions == nil {
		return nil
	}

	store, ok := state.registry.(taskStore)
	if !ok {
		state.logger.Warn(
			"daemon: task runtime skipped because registry does not implement task store",
		)
		return nil
	}

	bridge, err := newTaskSessionBridge(
		state.sessions,
		d.homePaths.HomeDir,
		state.logger,
		withTaskSessionContextOverlay(state.situationContext),
	)
	if err != nil {
		return err
	}
	reentry, err := bootHarnessReentryBridge(ctx, state)
	if err != nil {
		return fmt.Errorf("daemon: create harness reentry bridge: %w", err)
	}
	wakeBridge, err := newTaskWakeBridge(ctx, state.sessions, state.logger)
	if err != nil {
		return fmt.Errorf("daemon: create task wake bridge: %w", err)
	}
	reviewRequests := newRunReviewRequestedForwarder()
	eventObserver, bridgeNotifications, networkTaskStatus := d.composeTaskEventObserver(
		state,
		store,
		reentry,
	)
	coordinatorRunner, err := newBootLoopCoordinatorRunner(store, state, d.homePaths)
	if err != nil {
		return fmt.Errorf("daemon: create loop coordinator runner: %w", err)
	}
	manager, err := newTaskRuntimeManager(
		state,
		store,
		bridge,
		wakeBridge,
		eventObserver,
		reviewRequests,
		coordinatorRunner,
	)
	if err != nil {
		return fmt.Errorf("daemon: create task manager: %w", err)
	}
	if err := installLoopNativeHookObserver(state, manager, store, d.now); err != nil {
		return err
	}
	loopActions, err := installLoopActionRuntime(state, manager, store, coordinatorRunner, d.now)
	if err != nil {
		return err
	}
	detached, err := newHarnessDetachedWorkBridge(manager, store, state.sessions)
	if err != nil {
		return fmt.Errorf("daemon: create detached harness bridge: %w", err)
	}

	installTaskRuntime(
		state,
		manager,
		store,
		detached,
		reentry,
		wakeBridge,
		bridgeNotifications,
		networkTaskStatus,
		loopActions,
		reviewRequests,
	)

	return recoverInstalledTaskRuntime(ctx, state, manager, store, reentry, loopActions)
}

func recoverInstalledTaskRuntime(
	ctx context.Context,
	state *bootState,
	manager *taskpkg.Service,
	store taskStore,
	reentry *harnessReentryBridge,
	loopActions *loopActionRuntime,
) error {
	if err := recoverBootTaskRuns(ctx, state, manager, store); err != nil {
		return err
	}
	if loopActions != nil {
		loopActions.Recover(ctx)
	}
	return recoverDetachedHarnessReentry(ctx, reentry)
}

func installLoopActionRuntime(
	state *bootState,
	manager *taskpkg.Service,
	store taskStore,
	coordinatorRunner *looppkg.CoordinatorRunner,
	now func() time.Time,
) (*loopActionRuntime, error) {
	if coordinatorRunner == nil {
		return nil, nil
	}
	loopActions, err := newLoopActionRuntime(manager, store, coordinatorRunner, state.logger, now)
	if err != nil {
		return nil, fmt.Errorf("daemon: create loop action runtime: %w", err)
	}
	if state.notifier != nil {
		state.notifier.AddTaskRunEnqueuedObserver(loopActions)
	}
	return loopActions, nil
}

func installLoopNativeHookObserver(
	state *bootState,
	manager *taskpkg.Service,
	store taskStore,
	now func() time.Time,
) error {
	if state == nil || state.notifier == nil || manager == nil {
		return nil
	}
	loopStore, ok := store.(loopHookCoordinatorStore)
	if !ok {
		if state.logger != nil {
			state.logger.Warn("daemon: loop native hook observer skipped because task store lacks loop callbacks")
		}
		return nil
	}
	observer, err := newLoopNativeHookObserver(
		loopStore,
		state.notifier,
		schedulerTaskSource{manager: manager, store: store},
		now,
	)
	if err != nil {
		return err
	}
	state.notifier.AddLoopStartedObserver(observer)
	state.notifier.AddTaskRunTerminalObserver(observer)
	state.notifier.AddLoopTerminalObserver(observer)
	return nil
}

func installTaskRuntime(
	state *bootState,
	manager *taskpkg.Service,
	store taskStore,
	detached *harnessDetachedWorkBridge,
	reentry *harnessReentryBridge,
	wakeBridge *taskWakeBridge,
	bridgeNotifications *bridgeTerminalTaskNotificationObserver,
	networkTaskStatus *networkTaskStatusObserver,
	loopActions *loopActionRuntime,
	reviewRequests *runReviewRequestedForwarder,
) {
	state.tasks = &taskRuntime{
		manager:             manager,
		store:               store,
		detached:            detached,
		reentry:             reentry,
		wakeBridge:          wakeBridge,
		bridgeNotifications: bridgeNotifications,
		networkTaskStatus:   networkTaskStatus,
		loopActions:         loopActions,
	}
	state.reviewRequests = reviewRequests
	state.deps.Tasks = manager
}

func newTaskRuntimeManager(
	state *bootState,
	store taskStore,
	bridge taskpkg.SessionExecutor,
	wakeBridge taskpkg.WakeNotifier,
	eventObserver taskpkg.EventObserver,
	reviewRequests taskpkg.RunReviewRequestedObserver,
	coordinatorRunner taskpkg.CoordinatorRunner,
) (*taskpkg.Service, error) {
	return taskpkg.NewManager(
		taskManagerOptions(
			store,
			bridge,
			wakeBridge,
			eventObserver,
			state.notifier,
			reviewRequests,
			coordinatorRunner,
			looppkg.NewStoreFinalizer(),
			state.cfg.Task.Recovery,
			state.cfg.Autonomy.Scheduler,
			state.cfg.Autonomy.BlockRecurrenceLimit,
		)...,
	)
}

func recoverDetachedHarnessReentry(ctx context.Context, reentry *harnessReentryBridge) error {
	if reentry == nil {
		return nil
	}
	if err := reentry.recover(ctx); err != nil {
		return fmt.Errorf("daemon: recover detached harness reentry bridge: %w", err)
	}
	return nil
}

func recoverBootTaskRuns(
	ctx context.Context,
	state *bootState,
	manager *taskpkg.Service,
	store taskStore,
) error {
	actor, err := taskpkg.DeriveDaemonActorContext("boot-recovery", "daemon.boot")
	if err != nil {
		return fmt.Errorf("daemon: derive task boot recovery actor: %w", err)
	}
	stats, err := recoverTaskRunsOnBoot(ctx, manager, store, state.sessions, actor)
	if err != nil {
		return err
	}
	if stats.requeued+stats.markedRunning+stats.failed > 0 {
		state.logger.Info(
			"daemon: task boot recovery complete",
			"requeued_runs", stats.requeued,
			"resumed_running_runs", stats.markedRunning,
			"failed_runs", stats.failed,
		)
	}
	if reconciler, ok := store.(loopCoordinatorBootReconciler); ok {
		runs, err := reconciler.ReconcileLoopCoordinatorsOnBoot(ctx, actor.Origin, time.Now().UTC())
		if err != nil {
			return fmt.Errorf("daemon: reconcile loop coordinators on boot: %w", err)
		}
		if len(runs) > 0 {
			state.logger.Info(
				"daemon: loop coordinator boot reconcile complete",
				"enqueued_runs",
				len(runs),
			)
		}
	}
	started, err := (schedulerTaskSource{manager: manager, store: store}).RunLoopCoordinatorBackstop(
		ctx,
		time.Now().UTC(),
		actor,
	)
	if err != nil {
		return fmt.Errorf("daemon: start loop coordinators on boot: %w", err)
	}
	if started > 0 {
		state.logger.Info("daemon: loop coordinator boot start complete", "started_runs", started)
	}
	return nil
}
