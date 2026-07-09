package daemon

import (
	"context"

	hookspkg "github.com/compozy/agh/internal/hooks"
)

type taskStatusChangedObserver interface {
	OnTaskStatusChanged(context.Context, hookspkg.TaskStatusChangedPayload) error
}

type taskLifecycleWatchObserver interface {
	OnTaskBlocked(context.Context, hookspkg.TaskBlockedPayload) error
	OnTaskUnblocked(context.Context, hookspkg.TaskUnblockedPayload) error
	OnTaskNeedsAttention(context.Context, hookspkg.TaskNeedsAttentionPayload) error
	OnTaskRecovered(context.Context, hookspkg.TaskRecoveredPayload) error
}

type loopNodeTerminalObserver interface {
	OnLoopNodeTerminal(context.Context, hookspkg.LoopNodeTerminalPayload) error
}

type automationRunWatchObserver interface {
	OnAutomationRunCompleted(context.Context, hookspkg.AutomationRunCompletedPayload) error
	OnAutomationRunFailed(context.Context, hookspkg.AutomationRunFailedPayload) error
}

type networkWatchObserver interface {
	OnNetworkThreadOpened(context.Context, hookspkg.NetworkThreadOpenedPayload) error
	OnNetworkDirectRoomOpened(context.Context, hookspkg.NetworkDirectRoomOpenedPayload) error
	OnNetworkMessagePersisted(context.Context, hookspkg.NetworkMessagePersistedPayload) error
	OnNetworkWorkOpened(context.Context, hookspkg.NetworkWorkOpenedPayload) error
	OnNetworkWorkTransitioned(context.Context, hookspkg.NetworkWorkTransitionedPayload) error
	OnNetworkWorkClosed(context.Context, hookspkg.NetworkWorkClosedPayload) error
}

func (n *hooksNotifier) AddTaskStatusChangedObserver(observer taskStatusChangedObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.taskStatusChangedHooks = append(n.taskStatusChangedHooks, observer)
}

func (n *hooksNotifier) AddTaskLifecycleWatchObserver(observer taskLifecycleWatchObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.taskLifecycleWatchHooks = append(n.taskLifecycleWatchHooks, observer)
}

func (n *hooksNotifier) AddLoopNodeTerminalObserver(observer loopNodeTerminalObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.loopNodeTerminalHooks = append(n.loopNodeTerminalHooks, observer)
}

func (n *hooksNotifier) AddAutomationRunWatchObserver(observer automationRunWatchObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.automationRunWatchHooks = append(n.automationRunWatchHooks, observer)
}

func (n *hooksNotifier) AddNetworkWatchObserver(observer networkWatchObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.networkWatchHooks = append(n.networkWatchHooks, observer)
}

func (n *hooksNotifier) taskStatusChangedObservers() []taskStatusChangedObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]taskStatusChangedObserver(nil), n.taskStatusChangedHooks...)
}

func (n *hooksNotifier) taskLifecycleWatchObservers() []taskLifecycleWatchObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]taskLifecycleWatchObserver(nil), n.taskLifecycleWatchHooks...)
}

func (n *hooksNotifier) loopNodeTerminalObservers() []loopNodeTerminalObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]loopNodeTerminalObserver(nil), n.loopNodeTerminalHooks...)
}

func (n *hooksNotifier) automationRunWatchObservers() []automationRunWatchObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]automationRunWatchObserver(nil), n.automationRunWatchHooks...)
}

func (n *hooksNotifier) networkWatchObservers() []networkWatchObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]networkWatchObserver(nil), n.networkWatchHooks...)
}

func dispatchTaskStatusChangedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.TaskStatusChangedPayload,
) (hookspkg.TaskStatusChangedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookTaskStatusChanged,
		payload,
		hookRuntime.DispatchTaskStatusChanged,
	)
	notifier.notifyTaskStatusChangedObservers(ctx, result)
	return result, err
}

func (n *hooksNotifier) notifyTaskStatusChangedObservers(
	ctx context.Context,
	payload hookspkg.TaskStatusChangedPayload,
) {
	for _, observer := range n.taskStatusChangedObservers() {
		n.notifyTaskStatusChangedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyTaskStatusChangedObserver(
	ctx context.Context,
	observer taskStatusChangedObserver,
	payload hookspkg.TaskStatusChangedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			n.logger.Warn(
				"daemon: task status observer panic",
				"task_id", payload.TaskID,
				"workspace_id", payload.WorkspaceID,
				"panic", recovered,
			)
		}
	}()
	if err := observer.OnTaskStatusChanged(notifyCtx, payload); err != nil {
		n.logger.Warn(
			"daemon: task status observer failed",
			"task_id", payload.TaskID,
			"workspace_id", payload.WorkspaceID,
			"error", err,
		)
	}
}

func dispatchTaskBlockedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.TaskBlockedPayload,
) (hookspkg.TaskBlockedPayload, error) {
	result, err := dispatchRuntime(ctx, notifier, hookspkg.HookTaskBlocked, payload, hookRuntime.DispatchTaskBlocked)
	notifier.notifyTaskBlockedObservers(ctx, result)
	return result, err
}

func dispatchTaskUnblockedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.TaskUnblockedPayload,
) (hookspkg.TaskUnblockedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookTaskUnblocked,
		payload,
		hookRuntime.DispatchTaskUnblocked,
	)
	notifier.notifyTaskUnblockedObservers(ctx, result)
	return result, err
}

func dispatchTaskNeedsAttentionWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.TaskNeedsAttentionPayload,
) (hookspkg.TaskNeedsAttentionPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookTaskNeedsAttention,
		payload,
		hookRuntime.DispatchTaskNeedsAttention,
	)
	notifier.notifyTaskNeedsAttentionObservers(ctx, result)
	return result, err
}

func dispatchTaskRecoveredWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.TaskRecoveredPayload,
) (hookspkg.TaskRecoveredPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookTaskRecovered,
		payload,
		hookRuntime.DispatchTaskRecovered,
	)
	notifier.notifyTaskRecoveredObservers(ctx, result)
	return result, err
}

func (n *hooksNotifier) notifyTaskBlockedObservers(ctx context.Context, payload hookspkg.TaskBlockedPayload) {
	for _, observer := range n.taskLifecycleWatchObservers() {
		n.notifyTaskBlockedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyTaskUnblockedObservers(ctx context.Context, payload hookspkg.TaskUnblockedPayload) {
	for _, observer := range n.taskLifecycleWatchObservers() {
		n.notifyTaskUnblockedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyTaskNeedsAttentionObservers(
	ctx context.Context,
	payload hookspkg.TaskNeedsAttentionPayload,
) {
	for _, observer := range n.taskLifecycleWatchObservers() {
		n.notifyTaskNeedsAttentionObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyTaskRecoveredObservers(ctx context.Context, payload hookspkg.TaskRecoveredPayload) {
	for _, observer := range n.taskLifecycleWatchObservers() {
		n.notifyTaskRecoveredObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyTaskBlockedObserver(
	ctx context.Context,
	observer taskLifecycleWatchObserver,
	payload hookspkg.TaskBlockedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverTaskLifecycleObserverPanic(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskBlocked)
	if err := observer.OnTaskBlocked(notifyCtx, payload); err != nil {
		n.logTaskLifecycleObserverError(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskBlocked, err)
	}
}

func (n *hooksNotifier) notifyTaskUnblockedObserver(
	ctx context.Context,
	observer taskLifecycleWatchObserver,
	payload hookspkg.TaskUnblockedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverTaskLifecycleObserverPanic(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskUnblocked)
	if err := observer.OnTaskUnblocked(notifyCtx, payload); err != nil {
		n.logTaskLifecycleObserverError(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskUnblocked, err)
	}
}

func (n *hooksNotifier) notifyTaskNeedsAttentionObserver(
	ctx context.Context,
	observer taskLifecycleWatchObserver,
	payload hookspkg.TaskNeedsAttentionPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverTaskLifecycleObserverPanic(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskNeedsAttention)
	if err := observer.OnTaskNeedsAttention(notifyCtx, payload); err != nil {
		n.logTaskLifecycleObserverError(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskNeedsAttention, err)
	}
}

func (n *hooksNotifier) notifyTaskRecoveredObserver(
	ctx context.Context,
	observer taskLifecycleWatchObserver,
	payload hookspkg.TaskRecoveredPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverTaskLifecycleObserverPanic(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskRecovered)
	if err := observer.OnTaskRecovered(notifyCtx, payload); err != nil {
		n.logTaskLifecycleObserverError(payload.TaskID, payload.WorkspaceID, hookspkg.HookTaskRecovered, err)
	}
}

func (n *hooksNotifier) recoverTaskLifecycleObserverPanic(
	taskID string,
	workspaceID string,
	event hookspkg.HookEvent,
) {
	if recovered := recover(); recovered != nil {
		n.logger.Warn(
			"daemon: task lifecycle observer panic",
			"hook_event", event,
			"task_id", taskID,
			"workspace_id", workspaceID,
			"panic", recovered,
		)
	}
}

func (n *hooksNotifier) logTaskLifecycleObserverError(
	taskID string,
	workspaceID string,
	event hookspkg.HookEvent,
	err error,
) {
	n.logger.Warn(
		"daemon: task lifecycle observer failed",
		"hook_event", event,
		"task_id", taskID,
		"workspace_id", workspaceID,
		"error", err,
	)
}

func dispatchLoopNodeTerminalWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.LoopNodeTerminalPayload,
) (hookspkg.LoopNodeTerminalPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookLoopNodeTerminal,
		payload,
		hookRuntime.DispatchLoopNodeTerminal,
	)
	notifier.notifyLoopNodeTerminalObservers(ctx, result)
	return result, err
}

func (n *hooksNotifier) notifyLoopNodeTerminalObservers(
	ctx context.Context,
	payload hookspkg.LoopNodeTerminalPayload,
) {
	for _, observer := range n.loopNodeTerminalObservers() {
		n.notifyLoopNodeTerminalObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyLoopNodeTerminalObserver(
	ctx context.Context,
	observer loopNodeTerminalObserver,
	payload hookspkg.LoopNodeTerminalPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			n.logger.Warn(
				"daemon: loop node terminal observer panic",
				"loop_run_id", payload.LoopRunID,
				"node_id", payload.NodeID,
				"panic", recovered,
			)
		}
	}()
	if err := observer.OnLoopNodeTerminal(notifyCtx, payload); err != nil {
		n.logger.Warn(
			"daemon: loop node terminal observer failed",
			"loop_run_id", payload.LoopRunID,
			"node_id", payload.NodeID,
			"error", err,
		)
	}
}
