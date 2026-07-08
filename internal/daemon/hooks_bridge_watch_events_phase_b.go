package daemon

import (
	"context"

	hookspkg "github.com/compozy/agh/internal/hooks"
)

func (n *hooksNotifier) DispatchAutomationJobPreFire(
	ctx context.Context,
	payload hookspkg.AutomationJobPreFirePayload,
) (hookspkg.AutomationJobPreFirePayload, error) {
	return dispatchRuntime(ctx, n, hookspkg.HookAutomationJobPreFire, payload, hookRuntime.DispatchAutomationJobPreFire)
}

func (n *hooksNotifier) DispatchAutomationJobPostFire(
	ctx context.Context,
	payload hookspkg.AutomationJobPostFirePayload,
) (hookspkg.AutomationJobPostFirePayload, error) {
	return dispatchRuntime(ctx, n, hookspkg.HookAutomationJobPostFire, payload, hookRuntime.DispatchAutomationJobPostFire)
}

func (n *hooksNotifier) DispatchAutomationTriggerPreFire(
	ctx context.Context,
	payload hookspkg.AutomationTriggerPreFirePayload,
) (hookspkg.AutomationTriggerPreFirePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAutomationTriggerPreFire,
		payload,
		hookRuntime.DispatchAutomationTriggerPreFire,
	)
}

func (n *hooksNotifier) DispatchAutomationTriggerPostFire(
	ctx context.Context,
	payload hookspkg.AutomationTriggerPostFirePayload,
) (hookspkg.AutomationTriggerPostFirePayload, error) {
	return dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAutomationTriggerPostFire,
		payload,
		hookRuntime.DispatchAutomationTriggerPostFire,
	)
}

func (n *hooksNotifier) DispatchAutomationRunCompleted(
	ctx context.Context,
	payload hookspkg.AutomationRunCompletedPayload,
) (hookspkg.AutomationRunCompletedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAutomationRunCompleted,
		payload,
		hookRuntime.DispatchAutomationRunCompleted,
	)
	n.notifyAutomationRunCompletedObservers(ctx, result)
	return result, err
}

func (n *hooksNotifier) DispatchAutomationRunFailed(
	ctx context.Context,
	payload hookspkg.AutomationRunFailedPayload,
) (hookspkg.AutomationRunFailedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		n,
		hookspkg.HookAutomationRunFailed,
		payload,
		hookRuntime.DispatchAutomationRunFailed,
	)
	n.notifyAutomationRunFailedObservers(ctx, result)
	return result, err
}

func dispatchNetworkThreadOpenedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkThreadOpenedPayload,
) (hookspkg.NetworkThreadOpenedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkThreadOpened,
		payload,
		hookRuntime.DispatchNetworkThreadOpened,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkThreadOpened,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkThreadOpened(ctx, payload)
		},
	)
	return result, err
}

func dispatchNetworkDirectRoomOpenedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkDirectRoomOpenedPayload,
) (hookspkg.NetworkDirectRoomOpenedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkDirectRoomOpened,
		payload,
		hookRuntime.DispatchNetworkDirectRoomOpened,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkDirectRoomOpened,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkDirectRoomOpened(ctx, payload)
		},
	)
	return result, err
}

func dispatchNetworkMessagePersistedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkMessagePersistedPayload,
) (hookspkg.NetworkMessagePersistedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkMessagePersisted,
		payload,
		hookRuntime.DispatchNetworkMessagePersisted,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkMessagePersisted,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkMessagePersisted(ctx, payload)
		},
	)
	return result, err
}

func dispatchNetworkWorkOpenedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkWorkOpenedPayload,
) (hookspkg.NetworkWorkOpenedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkWorkOpened,
		payload,
		hookRuntime.DispatchNetworkWorkOpened,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkWorkOpened,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkWorkOpened(ctx, payload)
		},
	)
	return result, err
}

func dispatchNetworkWorkTransitionedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkWorkTransitionedPayload,
) (hookspkg.NetworkWorkTransitionedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkWorkTransitioned,
		payload,
		hookRuntime.DispatchNetworkWorkTransitioned,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkWorkTransitioned,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkWorkTransitioned(ctx, payload)
		},
	)
	return result, err
}

func dispatchNetworkWorkClosedWithWatchObservers(
	ctx context.Context,
	notifier *hooksNotifier,
	payload hookspkg.NetworkWorkClosedPayload,
) (hookspkg.NetworkWorkClosedPayload, error) {
	result, err := dispatchRuntime(
		ctx,
		notifier,
		hookspkg.HookNetworkWorkClosed,
		payload,
		hookRuntime.DispatchNetworkWorkClosed,
	)
	notifier.notifyNetworkWatchObservers(
		ctx,
		hookspkg.HookNetworkWorkClosed,
		result,
		func(ctx context.Context, observer networkWatchObserver, payload hookspkg.NetworkPayload) error {
			return observer.OnNetworkWorkClosed(ctx, payload)
		},
	)
	return result, err
}

func (n *hooksNotifier) notifyAutomationRunCompletedObservers(
	ctx context.Context,
	payload hookspkg.AutomationRunCompletedPayload,
) {
	for _, observer := range n.automationRunWatchObservers() {
		n.notifyAutomationRunCompletedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyAutomationRunCompletedObserver(
	ctx context.Context,
	observer automationRunWatchObserver,
	payload hookspkg.AutomationRunCompletedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverAutomationRunObserverPanic(payload.RunID, payload.WorkspaceID, hookspkg.HookAutomationRunCompleted)
	if err := observer.OnAutomationRunCompleted(notifyCtx, payload); err != nil {
		n.logAutomationRunObserverError(payload.RunID, payload.WorkspaceID, hookspkg.HookAutomationRunCompleted, err)
	}
}

func (n *hooksNotifier) notifyAutomationRunFailedObservers(
	ctx context.Context,
	payload hookspkg.AutomationRunFailedPayload,
) {
	for _, observer := range n.automationRunWatchObservers() {
		n.notifyAutomationRunFailedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyAutomationRunFailedObserver(
	ctx context.Context,
	observer automationRunWatchObserver,
	payload hookspkg.AutomationRunFailedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer n.recoverAutomationRunObserverPanic(payload.RunID, payload.WorkspaceID, hookspkg.HookAutomationRunFailed)
	if err := observer.OnAutomationRunFailed(notifyCtx, payload); err != nil {
		n.logAutomationRunObserverError(payload.RunID, payload.WorkspaceID, hookspkg.HookAutomationRunFailed, err)
	}
}

func (n *hooksNotifier) recoverAutomationRunObserverPanic(runID string, workspaceID string, event hookspkg.HookEvent) {
	if recovered := recover(); recovered != nil {
		n.logger.Warn(
			"daemon: automation run observer panic",
			"hook_event", event,
			"run_id", runID,
			"workspace_id", workspaceID,
			"panic", recovered,
		)
	}
}

func (n *hooksNotifier) logAutomationRunObserverError(
	runID string,
	workspaceID string,
	event hookspkg.HookEvent,
	err error,
) {
	n.logger.Warn(
		"daemon: automation run observer failed",
		"hook_event", event,
		"run_id", runID,
		"workspace_id", workspaceID,
		"error", err,
	)
}

func (n *hooksNotifier) notifyNetworkWatchObservers(
	ctx context.Context,
	event hookspkg.HookEvent,
	payload hookspkg.NetworkPayload,
	call func(context.Context, networkWatchObserver, hookspkg.NetworkPayload) error,
) {
	for _, observer := range n.networkWatchObservers() {
		n.notifyNetworkWatchObserver(ctx, event, payload, observer, call)
	}
}

func (n *hooksNotifier) notifyNetworkWatchObserver(
	ctx context.Context,
	event hookspkg.HookEvent,
	payload hookspkg.NetworkPayload,
	observer networkWatchObserver,
	call func(context.Context, networkWatchObserver, hookspkg.NetworkPayload) error,
) {
	if observer == nil || call == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			n.logger.Warn(
				"daemon: network observer panic",
				"hook_event", event,
				"message_id", payload.MessageID,
				"work_id", payload.WorkID,
				"workspace_id", payload.WorkspaceID,
				"panic", recovered,
			)
		}
	}()
	if err := call(notifyCtx, observer, payload); err != nil {
		n.logger.Warn(
			"daemon: network observer failed",
			"hook_event", event,
			"message_id", payload.MessageID,
			"work_id", payload.WorkID,
			"workspace_id", payload.WorkspaceID,
			"error", err,
		)
	}
}
