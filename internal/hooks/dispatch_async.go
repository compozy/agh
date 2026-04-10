package hooks

import (
	"context"
	"time"
)

func submitAsyncHooks[P any, R any](h *Hooks, parent context.Context, payload P, hooks []*ResolvedHook, pipe pipeline[P, R]) {
	if h == nil || h.pool == nil {
		return
	}

	parentDepth := currentDispatchDepth(parent)
	for _, hook := range hooks {
		if hook == nil {
			continue
		}

		asyncHook := *hook
		asyncPayload := payload
		h.pool.Submit(asyncTask{
			hook: asyncHook.RegisteredHook,
			run: func(poolCtx context.Context) {
				baseCtx := context.WithValue(poolCtx, dispatchDepthContextKey{}, parentDepth)
				baseCtx = context.WithValue(baseCtx, dispatchChainContextKey{}, currentDispatchChain(parent))
				hookCtx, depth, err := h.enterDispatch(baseCtx, asyncHook.Event)
				if err != nil {
					h.emitHookRun(poolCtx, asyncPayload, asyncHook.RegisteredHook, HookRunOutcomeSkipped, 0, nil, err, parentDepth)
					return
				}

				cancel := func() {}
				if asyncHook.Timeout > 0 {
					hookCtx, cancel = context.WithTimeout(hookCtx, asyncHook.Timeout)
				}
				defer cancel()

				started := time.Now()
				_, rawPatch, err := pipe.runHook(hookCtx, asyncHook.RegisteredHook, asyncPayload)
				duration := time.Since(started)
				if err != nil {
					h.emitHookRun(hookCtx, asyncPayload, asyncHook.RegisteredHook, HookRunOutcomeFailed, duration, rawPatch, err, depth)
					h.logger.WarnContext(
						hookCtx,
						"hook.dispatch.async_failed",
						"hook", asyncHook.Name,
						"event", asyncHook.Event.String(),
						"source", asyncHook.Source.String(),
						"error", err,
					)
					return
				}
				h.emitHookRun(hookCtx, asyncPayload, asyncHook.RegisteredHook, HookRunOutcomeApplied, duration, rawPatch, nil, depth)
			},
		})
	}
}
