package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type hookSelector[P any] func(P) []*ResolvedHook

type patchGuard[P any, R any] func(context.Context, RegisteredHook, P, R) error

type denyDetector[R any] func(R) bool

type typedNativeExecutor[P any, R any] interface {
	Executor
	ExecuteTyped(context.Context, RegisteredHook, P) (R, error)
}

// pipeline executes one sync hook chain for a concrete payload/patch pair.
type pipeline[P any, R any] struct {
	event        HookEvent
	hooksRuntime *Hooks
	hooks        hookSelector[P]
	apply        func(P, R) P
	encode       func(P) ([]byte, error)
	decode       func([]byte) (R, error)
	denied       denyDetector[R]
	guard        patchGuard[P, R]
	enter        func(context.Context, HookEvent) (context.Context, int, error)
}

func (p pipeline[P, R]) execute(ctx context.Context, payload P) (P, error) {
	result, _, err := p.executeWithDisposition(ctx, payload)
	return result, err
}

func (p pipeline[P, R]) executeWithDisposition(ctx context.Context, payload P) (P, dispatchReport, error) {
	if err := p.validate(); err != nil {
		return payload, dispatchReport{}, err
	}

	enterDispatchFn := p.enter
	if enterDispatchFn == nil {
		enterDispatchFn = enterDispatch
	}

	dispatchCtx, depth, err := enterDispatchFn(ctx, p.event)
	if err != nil {
		return payload, dispatchReport{}, err
	}

	current := payload
	report := dispatchReport{Trace: make([]hookTraceEntry, 0)}
	for _, hook := range orderedResolvedHooksIfNeeded(p.hooks(payload)) {
		if hook == nil {
			continue
		}

		next, denied, trace, err := p.executeHook(dispatchCtx, hook, current, depth)
		if trace.Hook != "" {
			report.Trace = append(report.Trace, trace)
		}
		if err != nil {
			report.FailedHook = hook.Name
			report.FailedRequired = hook.Required
			if hook.Required {
				return current, report, fmt.Errorf(
					"hooks: required hook %q failed for event %q: %w",
					hook.Name,
					p.event,
					err,
				)
			}
			continue
		}

		current = next
		if denied {
			report.Denied = true
			report.DenySource = hook.Name
			return current, report, nil
		}
	}

	return current, report, nil
}

func (p pipeline[P, R]) validate() error {
	if err := p.event.Validate(); err != nil {
		return err
	}
	if p.hooks == nil {
		return fmt.Errorf("hooks: pipeline for event %q requires a hook selector", p.event)
	}
	if p.apply == nil {
		return fmt.Errorf("hooks: pipeline for event %q requires an apply function", p.event)
	}
	if p.encode == nil {
		return fmt.Errorf("hooks: pipeline for event %q requires an encode function", p.event)
	}
	if p.decode == nil {
		return fmt.Errorf("hooks: pipeline for event %q requires a decode function", p.event)
	}
	return nil
}

func (p pipeline[P, R]) executeHook(
	ctx context.Context,
	hook *ResolvedHook,
	payload P,
	depth int,
) (P, bool, hookTraceEntry, error) {
	if hook == nil {
		return payload, false, hookTraceEntry{}, errors.New("hooks: resolved hook is required")
	}
	if hook.Executor == nil {
		return payload, false, hookTraceEntry{}, fmt.Errorf("hooks: hook %q executor is required", hook.Name)
	}

	hookCtx := ctx
	cancel := func() {}
	if hook.Timeout > 0 {
		hookCtx, cancel = context.WithTimeout(ctx, hook.Timeout)
	}
	defer cancel()

	started := time.Now()
	patch, rawPatch, err := p.runHook(hookCtx, hook.RegisteredHook, payload)
	duration := time.Since(started)
	trace := hookTraceEntry{
		Hook:     hook.Name,
		Duration: duration,
		Required: hook.Required,
		Patch:    cloneRawJSON(rawPatch),
	}
	if err != nil {
		trace.Outcome = HookRunOutcomeFailed
		trace.Error = err.Error()
		p.recordHookRun(hookCtx, payload, hook.RegisteredHook, trace.Outcome, duration, rawPatch, err, depth)
		return payload, false, trace, err
	}
	if p.guard != nil {
		if err := p.guard(hookCtx, hook.RegisteredHook, payload, patch); err != nil {
			if errors.Is(err, ErrHookPatchRejected) {
				trace.Outcome = HookRunOutcomeRejected
				trace.Error = err.Error()
				p.recordHookRun(hookCtx, payload, hook.RegisteredHook, trace.Outcome, duration, rawPatch, err, depth)
				return payload, false, trace, nil
			}
			trace.Outcome = HookRunOutcomeFailed
			trace.Error = err.Error()
			p.recordHookRun(hookCtx, payload, hook.RegisteredHook, trace.Outcome, duration, rawPatch, err, depth)
			return payload, false, trace, err
		}
	}

	next := p.apply(payload, patch)
	denied := p.denied != nil && p.denied(patch)
	if denied {
		trace.Outcome = HookRunOutcomeDenied
	} else {
		trace.Outcome = HookRunOutcomeApplied
	}
	p.recordHookRun(hookCtx, payload, hook.RegisteredHook, trace.Outcome, duration, rawPatch, nil, depth)
	return next, denied, trace, nil
}

func (p pipeline[P, R]) runHook(ctx context.Context, hook RegisteredHook, payload P) (R, json.RawMessage, error) {
	if hook.Executor.Kind() == HookExecutorNative {
		if executor, ok := hook.Executor.(typedNativeExecutor[P, R]); ok {
			patch, err := executor.ExecuteTyped(ctx, hook, payload)
			if err != nil {
				var zero R
				return zero, nil, err
			}
			rawPatch, marshalErr := json.Marshal(patch)
			if marshalErr != nil {
				var zero R
				return zero, nil, fmt.Errorf("hooks: encode native patch for hook %q: %w", hook.Name, marshalErr)
			}
			return patch, rawPatch, nil
		}
	}

	encoded, err := p.encode(payload)
	if err != nil {
		var zero R
		return zero, nil, fmt.Errorf("hooks: encode payload for hook %q: %w", hook.Name, err)
	}

	rawPatch, err := hook.Executor.Execute(ctx, hook, encoded)
	if err != nil {
		var zero R
		return zero, nil, err
	}

	patch, err := p.decode(rawPatch)
	if err != nil {
		var zero R
		return zero, rawPatch, fmt.Errorf("hooks: decode patch for hook %q: %w", hook.Name, err)
	}

	return patch, rawPatch, nil
}

func encodeJSON[T any](payload T) ([]byte, error) {
	return json.Marshal(payload)
}

func decodeJSON[T any](payload []byte) (T, error) {
	var decoded T
	if len(bytes.TrimSpace(payload)) == 0 {
		return decoded, nil
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return decoded, err
	}
	return decoded, nil
}

func (p pipeline[P, R]) recordHookRun(
	ctx context.Context,
	payload P,
	hook RegisteredHook,
	outcome HookRunOutcome,
	duration time.Duration,
	rawPatch json.RawMessage,
	err error,
	depth int,
) {
	if p.hooksRuntime == nil {
		return
	}
	p.hooksRuntime.emitHookRun(ctx, payload, hook, outcome, duration, rawPatch, err, depth)
}
