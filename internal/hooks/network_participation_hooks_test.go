package hooks

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/network/participation"
)

func TestNetworkParticipationHooks(t *testing.T) {
	t.Parallel()

	t.Run("Should deny pre_resolve when patch Deny is set", func(t *testing.T) {
		t.Parallel()

		hooks := newTestHooks(
			t,
			WithNativeDeclarations([]HookDecl{{
				Name:         "deny-participation",
				Event:        HookNetworkParticipationPreResolve,
				Mode:         HookModeSync,
				Matcher:      HookMatcher{},
				ExecutorKind: HookExecutorNative,
			}}),
			WithExecutorResolver(testExecutorResolver(map[string]Executor{
				"deny-participation": NewTypedNativeExecutor(
					func(
						_ context.Context,
						_ RegisteredHook,
						_ NetworkParticipationPreResolvePayload,
					) (NetworkParticipationPreResolvePatch, error) {
						return NetworkParticipationPreResolvePatch{
							ControlPatch: ControlPatch{Deny: true, DenyReason: "policy"},
						}, nil
					},
				),
			})),
		)
		if err := hooks.Rebuild(t.Context()); err != nil {
			t.Fatalf("Rebuild() error = %v", err)
		}

		_, err := hooks.DispatchNetworkParticipationPreResolve(
			t.Context(),
			NetworkParticipationPreResolvePayload{WorkspaceID: "ws-alpha"},
		)
		if err == nil {
			t.Fatal("expected deny error")
		}
		if !strings.Contains(err.Error(), "denied") {
			t.Fatalf("error = %v, want denied diagnostic", err)
		}
	})

	t.Run("Should deny widen attempts on pre_resolve", func(t *testing.T) {
		t.Parallel()

		hooks := newTestHooks(
			t,
			WithNativeDeclarations([]HookDecl{{
				Name:         "widen-participation",
				Event:        HookNetworkParticipationPreResolve,
				Mode:         HookModeSync,
				Matcher:      HookMatcher{},
				ExecutorKind: HookExecutorNative,
			}}),
			WithExecutorResolver(testExecutorResolver(map[string]Executor{
				"widen-participation": NewTypedNativeExecutor(
					func(
						_ context.Context,
						_ RegisteredHook,
						_ NetworkParticipationPreResolvePayload,
					) (NetworkParticipationPreResolvePatch, error) {
						return NetworkParticipationPreResolvePatch{Widen: true}, nil
					},
				),
			})),
		)
		if err := hooks.Rebuild(t.Context()); err != nil {
			t.Fatalf("Rebuild() error = %v", err)
		}

		_, err := hooks.DispatchNetworkParticipationPreResolve(
			t.Context(),
			NetworkParticipationPreResolvePayload{WorkspaceID: "ws-alpha"},
		)
		if err == nil {
			t.Fatal("expected widen deny error")
		}
		if !strings.Contains(err.Error(), "denied") {
			t.Fatalf("error = %v, want denied diagnostic", err)
		}
	})

	t.Run("Should narrow request on pre_resolve without widening", func(t *testing.T) {
		t.Parallel()

		mode := participation.ModeLocal
		narrowed := &participation.Request{Mode: &mode}
		hooks := newTestHooks(
			t,
			WithNativeDeclarations([]HookDecl{{
				Name:         "narrow-participation",
				Event:        HookNetworkParticipationPreResolve,
				Mode:         HookModeSync,
				Matcher:      HookMatcher{},
				ExecutorKind: HookExecutorNative,
			}}),
			WithExecutorResolver(testExecutorResolver(map[string]Executor{
				"narrow-participation": NewTypedNativeExecutor(
					func(
						_ context.Context,
						_ RegisteredHook,
						_ NetworkParticipationPreResolvePayload,
					) (NetworkParticipationPreResolvePatch, error) {
						return NetworkParticipationPreResolvePatch{Request: narrowed}, nil
					},
				),
			})),
		)
		if err := hooks.Rebuild(t.Context()); err != nil {
			t.Fatalf("Rebuild() error = %v", err)
		}

		live := participation.ModeLive
		got, err := hooks.DispatchNetworkParticipationPreResolve(
			t.Context(),
			NetworkParticipationPreResolvePayload{
				WorkspaceID: "ws-alpha",
				Request:     &participation.Request{Mode: &live},
			},
		)
		if err != nil {
			t.Fatalf("DispatchNetworkParticipationPreResolve() error = %v", err)
		}
		if got.Request == nil || got.Request.Mode == nil || *got.Request.Mode != participation.ModeLocal {
			t.Fatalf("request = %#v, want local mode", got.Request)
		}
	})

	t.Run("Should dispatch resolved payload with Spec object", func(t *testing.T) {
		t.Parallel()

		seen := make(chan participation.Spec, 1)
		hooks := newTestHooks(
			t,
			WithNativeDeclarations([]HookDecl{{
				Name:         "observe-resolved",
				Event:        HookNetworkParticipationResolved,
				Mode:         HookModeAsync,
				Matcher:      HookMatcher{},
				ExecutorKind: HookExecutorNative,
			}}),
			WithExecutorResolver(testExecutorResolver(map[string]Executor{
				"observe-resolved": NewTypedNativeExecutor(
					func(
						_ context.Context,
						_ RegisteredHook,
						payload NetworkParticipationResolvedPayload,
					) (NetworkParticipationResolvedPatch, error) {
						seen <- payload.Spec
						return NetworkParticipationResolvedPatch{}, nil
					},
				),
			})),
			WithAsyncWorkerCount(1),
			WithAsyncQueueCapacity(1),
		)
		if err := hooks.Rebuild(t.Context()); err != nil {
			t.Fatalf("Rebuild() error = %v", err)
		}

		spec := participation.Spec{
			Version: participation.SpecVersion,
			Mode:    participation.ModeLocal,
			Source:  participation.SourceBuiltInLocal,
		}
		if _, err := hooks.DispatchNetworkParticipationResolved(
			t.Context(),
			NetworkParticipationResolvedPayload{
				WorkspaceID: "ws-alpha",
				Spec:        spec,
			},
		); err != nil {
			t.Fatalf("DispatchNetworkParticipationResolved() error = %v", err)
		}
		select {
		case got := <-seen:
			if got.Mode != participation.ModeLocal {
				t.Fatalf("resolved spec = %#v, want local", got)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for resolved observation")
		}
	})
}
