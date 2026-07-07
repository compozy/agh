package daemon

import (
	"context"

	hookspkg "github.com/compozy/agh/internal/hooks"
)

type loopStartedObserver interface {
	OnLoopStarted(context.Context, hookspkg.LoopStartedPayload) error
}

func (n *hooksNotifier) AddLoopStartedObserver(observer loopStartedObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.loopStartedHooks = append(n.loopStartedHooks, observer)
}

func (n *hooksNotifier) loopStartedObservers() []loopStartedObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]loopStartedObserver(nil), n.loopStartedHooks...)
}

func (n *hooksNotifier) notifyLoopStartedObservers(
	ctx context.Context,
	payload hookspkg.LoopStartedPayload,
) {
	for _, observer := range n.loopStartedObservers() {
		n.notifyLoopStartedObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyLoopStartedObserver(
	ctx context.Context,
	observer loopStartedObserver,
	payload hookspkg.LoopStartedPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			n.logger.Warn(
				"daemon: loop started observer panic",
				"loop_run_id", payload.LoopRunID,
				"loop_name", payload.LoopName,
				"panic", recovered,
			)
		}
	}()
	if err := observer.OnLoopStarted(notifyCtx, payload); err != nil {
		n.logger.Warn(
			"daemon: loop started observer failed",
			"loop_run_id", payload.LoopRunID,
			"loop_name", payload.LoopName,
			"error", err,
		)
	}
}
