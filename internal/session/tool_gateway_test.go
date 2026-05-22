package session

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/compozy/agh/internal/acp"
	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/testutil"
)

func TestProviderNativeToolGatewayIntercept(t *testing.T) {
	t.Parallel()

	t.Run("Should reject tool execution when tool pre call returns deny patch", func(t *testing.T) {
		t.Parallel()

		hooks := newNativeHookDispatcher(t,
			[]hookspkg.HookDecl{{
				Name:         "deny-tool-call",
				Event:        hookspkg.HookToolPreCall,
				Mode:         hookspkg.HookModeSync,
				ExecutorKind: hookspkg.HookExecutorNative,
				Matcher: hookspkg.HookMatcher{
					ToolID: "agh__write",
				},
			}},
			map[string]hookspkg.Executor{
				"deny-tool-call": hookspkg.NewTypedNativeExecutor(
					func(
						_ context.Context,
						_ hookspkg.RegisteredHook,
						payload hookspkg.ToolPreCallPayload,
					) (hookspkg.ToolCallPatch, error) {
						if payload.ToolID != "agh__write" {
							t.Fatalf("payload.ToolID = %q, want agh__write", payload.ToolID)
						}
						return hookspkg.ToolCallPatch{
							ControlPatch: hookspkg.ControlPatch{
								Deny:       true,
								DenyReason: "blocked by hook",
							},
						}, nil
					},
				),
			},
		)

		h := newHarness(t, WithHookSet(HookSet{Tools: hooks}))
		session := createSession(t, h)
		t.Cleanup(func() {
			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil &&
				!errors.Is(err, ErrSessionNotFound) {
				t.Fatalf("Stop() error = %v", err)
			}
		})

		gateway := newProviderNativeToolGateway(h.manager, session)
		if gateway == nil {
			t.Fatal("newProviderNativeToolGateway() = nil, want gateway")
		}

		_, err := gateway.Intercept(testutil.Context(t), acp.ToolExecutionRequest{
			ToolID: "agh__write",
			Input: json.RawMessage(`{
				"path": "/tmp/blocked.txt",
				"content": "blocked"
			}`),
		})
		if !errors.Is(err, acp.ErrPermissionDenied) {
			t.Fatalf("Intercept() error = %v, want ErrPermissionDenied", err)
		}
	})
}
