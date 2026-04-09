package hooks

import (
	"context"
	"fmt"
)

// NativeHookFunc executes an in-process hook without crossing a subprocess
// boundary.
type NativeHookFunc func(ctx context.Context, hook RegisteredHook, payload []byte) ([]byte, error)

// NativeExecutor runs hooks as direct Go callbacks.
type NativeExecutor struct {
	callback NativeHookFunc
}

// NewNativeExecutor constructs a NativeExecutor bound to one callback.
func NewNativeExecutor(callback NativeHookFunc) *NativeExecutor {
	return &NativeExecutor{callback: callback}
}

// Kind returns the executor type.
func (*NativeExecutor) Kind() HookExecutorKind {
	return HookExecutorNative
}

// Execute invokes the configured Go callback directly.
func (e *NativeExecutor) Execute(ctx context.Context, hook RegisteredHook, payload []byte) (result []byte, err error) {
	if e == nil || e.callback == nil {
		return nil, fmt.Errorf("hooks: hook %q: %w", hook.Name, ErrNativeCallbackRequired)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("hooks: hook %q native callback panic: %v", hook.Name, recovered)
			result = nil
		}
	}()

	return e.callback(ctx, hook, payload)
}
