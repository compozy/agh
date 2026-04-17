package hooks

import (
	"context"
	"testing"
)

func TestDispatchInputPreSubmitAsyncHookSeesStablePayloadSnapshot(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	seen := make(chan InputPreSubmitPayload, 1)

	hooks := newBenchmarkHooksRuntime(
		t,
		[]HookDecl{
			{
				Name:  "async-snapshot",
				Event: HookInputPreSubmit,
				Mode:  HookModeAsync,
			},
		},
		map[string]Executor{
			"async-snapshot": NewTypedNativeExecutor(
				func(_ context.Context, _ RegisteredHook, payload InputPreSubmitPayload) (InputPreSubmitPatch, error) {
					<-release
					seen <- payload
					return InputPreSubmitPatch{}, nil
				},
			),
		},
		WithAsyncWorkerCount(1),
		WithAsyncQueueCapacity(1),
	)

	payload := benchmarkInputPayload()
	if _, err := hooks.DispatchInputPreSubmit(t.Context(), payload); err != nil {
		t.Fatalf("DispatchInputPreSubmit() error = %v", err)
	}

	payload.ContextBlocks[0].Text = "mutated"
	payload.ContextBlocks[0].Metadata["scope"] = "mutated"

	close(release)

	select {
	case got := <-seen:
		if got.ContextBlocks[0].Text != "context" {
			t.Fatalf("async hook saw text %q, want %q", got.ContextBlocks[0].Text, "context")
		}
		if got.ContextBlocks[0].Metadata["scope"] != "bench" {
			t.Fatalf("async hook saw metadata scope %q, want %q", got.ContextBlocks[0].Metadata["scope"], "bench")
		}
	case <-t.Context().Done():
		t.Fatal("timed out waiting for async hook")
	}
}
