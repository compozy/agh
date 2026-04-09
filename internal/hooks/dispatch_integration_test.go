//go:build integration

package hooks

import (
	"context"
	"testing"
)

func TestDispatchInputPreSubmitOrdersNativeBeforeSubprocess(t *testing.T) {
	hooks := newTestHooks(
		t,
		WithNativeDeclarations([]HookDecl{
			{
				Name:         "native-prefix",
				Event:        HookInputPreSubmit,
				Mode:         HookModeSync,
				ExecutorKind: HookExecutorNative,
			},
		}),
		WithSkillDeclarations([]HookDecl{
			{
				Name:        "skill-shell",
				Event:       HookInputPreSubmit,
				Mode:        HookModeSync,
				Command:     "/bin/sh",
				Args:        []string{"-c", "payload=$(cat); if printf '%s' \"$payload\" | grep -q 'native'; then printf '{\"message\":\"native-shell\"}'; else printf '{\"message\":\"wrong-order\"}'; fi"},
				SkillSource: HookSkillSourceUser,
			},
		}),
		WithExecutorResolver(testExecutorResolver(map[string]Executor{
			"native-prefix": NewTypedNativeExecutor(func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
				msg := payload.Message + "native"
				return InputPreSubmitPatch{Message: &msg}, nil
			}),
		})),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	result, err := hooks.DispatchInputPreSubmit(t.Context(), InputPreSubmitPayload{
		PayloadBase: PayloadBase{Event: HookInputPreSubmit},
		Message:     "",
	})
	if err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v, want nil", err)
	}
	if result.Message != "native-shell" {
		t.Fatalf("result.Message = %q, want %q", result.Message, "native-shell")
	}
}
