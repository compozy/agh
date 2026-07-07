package daemon

import (
	"context"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
)

const loopHookObserverTimeout = 10 * time.Second

type loopTerminalObserver interface {
	OnLoopTerminal(context.Context, hookspkg.LoopTerminalPayload) error
}

func (n *hooksNotifier) AddLoopTerminalObserver(observer loopTerminalObserver) {
	if n == nil || observer == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.loopTerminalHooks = append(n.loopTerminalHooks, observer)
}

func (n *hooksNotifier) loopTerminalObservers() []loopTerminalObserver {
	if n == nil {
		return nil
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]loopTerminalObserver(nil), n.loopTerminalHooks...)
}

func (n *hooksNotifier) notifyLoopTerminalObservers(
	ctx context.Context,
	payload hookspkg.LoopTerminalPayload,
) {
	for _, observer := range n.loopTerminalObservers() {
		n.notifyLoopTerminalObserver(ctx, observer, payload)
	}
}

func (n *hooksNotifier) notifyLoopTerminalObserver(
	ctx context.Context,
	observer loopTerminalObserver,
	payload hookspkg.LoopTerminalPayload,
) {
	if observer == nil {
		return
	}
	notifyCtx, cancel := loopObserverContext(ctx)
	defer cancel()
	defer func() {
		if recovered := recover(); recovered != nil {
			n.logger.Warn(
				"daemon: loop terminal observer panic",
				"loop_run_id", payload.LoopRunID,
				"parent_loop_run_id", payload.ParentLoopRunID,
				"loop_name", payload.LoopName,
				"panic", recovered,
			)
		}
	}()
	if err := observer.OnLoopTerminal(notifyCtx, payload); err != nil {
		n.logger.Warn(
			"daemon: loop terminal observer failed",
			"loop_run_id", payload.LoopRunID,
			"parent_loop_run_id", payload.ParentLoopRunID,
			"loop_name", payload.LoopName,
			"error", err,
		)
	}
}

func loopObserverContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), loopHookObserverTimeout)
}
