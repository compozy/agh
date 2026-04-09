package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	event  HookEvent
	hooks  hookSelector[P]
	apply  func(P, R) P
	encode func(P) ([]byte, error)
	decode func([]byte) (R, error)
	denied denyDetector[R]
	guard  patchGuard[P, R]
}

func (p pipeline[P, R]) execute(ctx context.Context, payload P) (P, error) {
	if err := p.validate(); err != nil {
		return payload, err
	}

	dispatchCtx, _, err := enterDispatch(ctx, p.event)
	if err != nil {
		return payload, err
	}

	current := payload
	for _, hook := range OrderedResolvedHooks(p.hooks(payload)) {
		if hook == nil {
			continue
		}

		next, denied, err := p.executeHook(dispatchCtx, *hook, current)
		if err != nil {
			if hook.Required {
				return current, fmt.Errorf("hooks: required hook %q failed for event %q: %w", hook.Name, p.event, err)
			}
			continue
		}

		current = next
		if denied {
			return current, nil
		}
	}

	return current, nil
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

func (p pipeline[P, R]) executeHook(ctx context.Context, hook ResolvedHook, payload P) (P, bool, error) {
	if hook.Executor == nil {
		return payload, false, fmt.Errorf("hooks: hook %q executor is required", hook.Name)
	}

	hookCtx := ctx
	cancel := func() {}
	if hook.Timeout > 0 {
		hookCtx, cancel = context.WithTimeout(ctx, hook.Timeout)
	}
	defer cancel()

	patch, err := p.runHook(hookCtx, hook.RegisteredHook, payload)
	if err != nil {
		return payload, false, err
	}
	if p.guard != nil {
		if err := p.guard(hookCtx, hook.RegisteredHook, payload, patch); err != nil {
			if errors.Is(err, ErrHookPatchRejected) {
				return payload, false, nil
			}
			return payload, false, err
		}
	}

	next := p.apply(payload, patch)
	return next, p.denied != nil && p.denied(patch), nil
}

func (p pipeline[P, R]) runHook(ctx context.Context, hook RegisteredHook, payload P) (R, error) {
	if hook.Executor.Kind() == HookExecutorNative {
		if executor, ok := hook.Executor.(typedNativeExecutor[P, R]); ok {
			return executor.ExecuteTyped(ctx, hook, payload)
		}
	}

	encoded, err := p.encode(payload)
	if err != nil {
		var zero R
		return zero, fmt.Errorf("hooks: encode payload for hook %q: %w", hook.Name, err)
	}

	rawPatch, err := hook.Executor.Execute(ctx, hook, encoded)
	if err != nil {
		var zero R
		return zero, err
	}

	patch, err := p.decode(rawPatch)
	if err != nil {
		var zero R
		return zero, fmt.Errorf("hooks: decode patch for hook %q: %w", hook.Name, err)
	}

	return patch, nil
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
